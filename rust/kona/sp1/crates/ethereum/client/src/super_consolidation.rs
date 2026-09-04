//! Shared execution logic for the SP1 super-range guest's consolidation mode.

use std::{fmt::Debug, sync::Arc};

use alloy_consensus::{Header, Sealed};
use alloy_op_evm::post_exec::PostExecEvmFactoryAdapter;
use alloy_primitives::{B256, U256};
use anyhow::{anyhow, ensure};
use kona_genesis::{L1ChainConfig, RollupConfig};
use kona_interop::DependencySet;
use kona_preimage::{CommsClient, PreimageKey};
use kona_proof::l2::OracleL2ChainProvider;
use kona_proof_interop::{
    BootInfo as InteropBootInfo, HintType, OptimisticBlock as InteropOptimisticBlock,
    OracleInteropProvider, PreState, SuperchainConsolidator, TRANSITION_STATE_MAX_STEPS,
    TransitionState,
};
use kona_registry::{HashMap as RegistryHashMap, ROLLUP_CONFIGS};
use kona_sp1_client_utils::{
    precompiles::{CustomCrypto, ZkvmOpEvmFactory},
    super_root::{
        SuperConsolidationInputs, SuperConsolidationOutputs, SuperConsolidationTransition,
        SuperOptimisticBlock, SuperRoot, SuperRootProof, hash_super_root_proof,
    },
};

use crate::super_range::{
    fetch_l2_header, fetch_output_block_hash, load_dependency_set, load_l1_config,
    load_rollup_configs, optimistic_chain_id_as_u64,
};

/// Builds consolidation-mode public outputs from typed inputs and an oracle-backed witness source.
pub async fn build_consolidation_outputs<C>(
    inputs: SuperConsolidationInputs,
    oracle: Arc<C>,
) -> anyhow::Result<SuperConsolidationOutputs>
where
    C: CommsClient + Send + Sync + Debug + 'static,
{
    inputs.validate()?;

    // Deposit-only replacement re-executes blocks through revm, which may hit the KZG
    // point-evaluation precompile. Install the SP1-compatible crypto provider first.
    revm::precompile::install_crypto(CustomCrypto::default());

    let chain_ids = consolidation_chain_ids(&inputs)?;
    let dependency_set = load_dependency_set(&chain_ids, oracle.as_ref()).await?;
    let rollup_configs = load_rollup_configs(&chain_ids, oracle.as_ref()).await?;
    let l1_config = load_l1_config(&rollup_configs, oracle.as_ref()).await?;
    let rollup_configs: RegistryHashMap<_, _> = rollup_configs.into_iter().collect();

    let mut previous_super_root =
        fetch_super_root(oracle.as_ref(), inputs.previous_super_root).await?;
    let mut transitions = Vec::with_capacity(inputs.transitions.len());
    for input in inputs.transitions {
        ensure!(
            previous_super_root.timestamp.checked_add(1) ==
                Some(input.claimed_super_root_proof.super_root.timestamp),
            "consolidation previous super-root timestamp {} does not precede claimed timestamp {}",
            previous_super_root.timestamp,
            input.claimed_super_root_proof.super_root.timestamp,
        );

        let transition = run_transition(
            oracle.clone(),
            previous_super_root,
            input.optimistic_blocks,
            &input.claimed_super_root_proof,
            dependency_set.clone(),
            &rollup_configs,
            &l1_config,
        )
        .await?;
        previous_super_root = input.claimed_super_root_proof.super_root;
        transitions.push(transition);
    }

    Ok(SuperConsolidationOutputs {
        span: inputs.span,
        previous_super_root: inputs.previous_super_root,
        transitions,
    })
}

/// Returns `true` when interop is active on any consolidation chain at `timestamp`.
///
/// Returns `true` if there is a missing embedded config.
fn interop_active_for_consolidation(
    rollup_configs: &RegistryHashMap<u64, RollupConfig>,
    previous_super_root: &SuperRoot,
    timestamp: u64,
) -> bool {
    previous_super_root.output_roots.iter().any(|output_root| {
        rollup_configs
            .get(&output_root.chain_id)
            .is_none_or(|config| config.is_interop_active(timestamp))
    })
}

fn consolidation_chain_ids(inputs: &SuperConsolidationInputs) -> anyhow::Result<Vec<U256>> {
    inputs
        .transitions
        .first()
        .map(|input| {
            input
                .optimistic_blocks
                .iter()
                .map(|optimistic_block| optimistic_block.chain_id)
                .collect()
        })
        .ok_or_else(|| anyhow!("consolidation input span contains no transitions"))
}

async fn fetch_super_root<C>(oracle: &C, super_root_hash: B256) -> anyhow::Result<SuperRoot>
where
    C: CommsClient,
{
    let encoded = oracle
        .get(PreimageKey::new_keccak256(*super_root_hash))
        .await
        .map_err(|err| anyhow!("failed to fetch super-root preimage {super_root_hash}: {err}"))?;
    let super_root = SuperRoot::decode(&mut encoded.as_slice())
        .map_err(|err| anyhow!("failed to decode super-root preimage {super_root_hash}: {err}"))?;
    ensure!(
        super_root.hash() == super_root_hash,
        "decoded super-root hashes to {}, expected {super_root_hash}",
        super_root.hash(),
    );
    Ok(super_root)
}

async fn run_transition<C>(
    oracle: Arc<C>,
    previous_super_root: SuperRoot,
    optimistic_blocks: Vec<SuperOptimisticBlock>,
    claimed_super_root_proof: &SuperRootProof,
    dependency_set: DependencySet,
    rollup_configs: &RegistryHashMap<u64, RollupConfig>,
    l1_config: &L1ChainConfig,
) -> anyhow::Result<SuperConsolidationTransition>
where
    C: CommsClient + Send + Sync + Debug + 'static,
{
    ensure_previous_super_root_matches_optimistic_blocks(&previous_super_root, &optimistic_blocks)?;
    let claimed_super_root = hash_super_root_proof(claimed_super_root_proof)?;
    let transition_state = TransitionState::new(
        previous_super_root.clone(),
        optimistic_blocks
            .iter()
            .map(|block| InteropOptimisticBlock::new(block.block_hash, block.output_root))
            .collect(),
        TRANSITION_STATE_MAX_STEPS,
    );
    let agreed_pre_state = PreState::TransitionState(transition_state);
    let mut boot = InteropBootInfo {
        l1_head: B256::ZERO,
        agreed_pre_state_commitment: agreed_pre_state.hash(),
        agreed_pre_state,
        claimed_post_state: claimed_super_root,
        claimed_l2_timestamp: claimed_super_root_proof.super_root.timestamp,
        rollup_configs: rollup_configs.clone(),
        dependency_set,
        l1_config: l1_config.clone(),
    };

    // CrossL2Inbox has no code before interop, so no executing messages can exist. Skip the
    // provider and message-graph work while retaining the common transition finalization below.
    if interop_active_for_consolidation(
        &ROLLUP_CONFIGS,
        &previous_super_root,
        claimed_super_root_proof.super_root.timestamp,
    ) {
        let ConsolidationProviders { headers, l2_providers } = build_providers(
            oracle.clone(),
            &previous_super_root,
            &optimistic_blocks,
            claimed_super_root_proof.super_root.timestamp,
            rollup_configs,
        )
        .await?;

        let interop_provider = OracleInteropProvider::new(oracle, boot.clone(), headers);
        let evm_factory = PostExecEvmFactoryAdapter::new(ZkvmOpEvmFactory);
        SuperchainConsolidator::new(&mut boot, interop_provider, l2_providers, evm_factory)
            .consolidate()
            .await?;
    }

    let post_state = boot
        .agreed_pre_state
        .transition(None)
        .ok_or_else(|| anyhow!("failed to finalize consolidated transition state"))?;
    let post_commitment = post_state.hash();
    ensure!(
        post_commitment == claimed_super_root,
        "consolidated super-root {post_commitment} does not match claim {claimed_super_root}",
    );

    Ok(SuperConsolidationTransition {
        timestamp: claimed_super_root_proof.super_root.timestamp,
        optimistic_blocks,
        super_root: post_commitment,
    })
}

#[derive(Debug)]
struct ConsolidationProviders<C: CommsClient> {
    headers: RegistryHashMap<u64, Sealed<Header>>,
    l2_providers: RegistryHashMap<u64, OracleL2ChainProvider<C>>,
}

async fn build_providers<C>(
    oracle: Arc<C>,
    previous_super_root: &SuperRoot,
    optimistic_blocks: &[SuperOptimisticBlock],
    timestamp: u64,
    rollup_configs: &RegistryHashMap<u64, RollupConfig>,
) -> anyhow::Result<ConsolidationProviders<C>>
where
    C: CommsClient + Send + Sync + Debug + 'static,
{
    let mut headers = RegistryHashMap::default();
    let mut l2_providers = RegistryHashMap::default();
    for (previous_output, optimistic_block) in
        previous_super_root.output_roots.iter().zip(optimistic_blocks)
    {
        let chain_id = optimistic_chain_id_as_u64(optimistic_block)?;
        let rollup_config = rollup_configs.get(&chain_id).ok_or_else(|| {
            anyhow!("missing rollup config for consolidation chain ID {chain_id}")
        })?;
        let cross_safe_head_hash =
            fetch_output_block_hash(oracle.as_ref(), previous_output.output_root).await?;
        let optimistic_output_block_hash =
            fetch_output_block_hash(oracle.as_ref(), optimistic_block.output_root).await?;
        ensure!(
            optimistic_output_block_hash == optimistic_block.block_hash,
            "consolidation optimistic output root commits to block hash {actual}, expected {expected}",
            actual = optimistic_output_block_hash,
            expected = optimistic_block.block_hash,
        );
        HintType::L2BlockData
            .with_data(&[
                cross_safe_head_hash.as_slice(),
                optimistic_block.block_hash.as_slice(),
                chain_id.to_be_bytes().as_ref(),
            ])
            .send(oracle.as_ref())
            .await?;
        let local_safe_header =
            fetch_l2_header(oracle.as_ref(), optimistic_block.block_hash, chain_id).await?;
        ensure!(
            local_safe_header.timestamp <= timestamp,
            "consolidation optimistic block timestamp {actual} is after super-root timestamp {expected}",
            actual = local_safe_header.timestamp,
            expected = timestamp,
        );

        let mut l2_provider = OracleL2ChainProvider::new(
            cross_safe_head_hash,
            Arc::new(rollup_config.clone()),
            oracle.clone(),
        );
        l2_provider.set_chain_id(Some(chain_id));
        headers.insert(
            chain_id,
            Sealed::new_unchecked(local_safe_header, optimistic_block.block_hash),
        );
        l2_providers.insert(chain_id, l2_provider);
    }

    Ok(ConsolidationProviders { headers, l2_providers })
}

fn ensure_previous_super_root_matches_optimistic_blocks(
    previous_super_root: &SuperRoot,
    optimistic_blocks: &[SuperOptimisticBlock],
) -> anyhow::Result<()> {
    ensure!(
        previous_super_root.output_roots.len() == optimistic_blocks.len(),
        "previous super-root has {} chains but consolidation has {} optimistic blocks",
        previous_super_root.output_roots.len(),
        optimistic_blocks.len(),
    );

    for (previous_output, optimistic_block) in
        previous_super_root.output_roots.iter().zip(optimistic_blocks)
    {
        ensure!(
            U256::from(previous_output.chain_id) == optimistic_block.chain_id,
            "previous super-root chain {} does not match optimistic chain {}",
            previous_output.chain_id,
            optimistic_block.chain_id,
        );
    }

    Ok(())
}

#[cfg(test)]
mod tests {
    use std::sync::Mutex;

    use alloy_consensus::{EMPTY_ROOT_HASH, Header};
    use alloy_primitives::{B256, U256};
    use alloy_rlp::EMPTY_STRING_CODE;
    use async_trait::async_trait;
    use kona_genesis::RollupConfig;
    use kona_preimage::{
        DEPENDENCY_SET_KEY, HintWriterClient, L2_ROLLUP_CONFIG_KEY, PreimageKey,
        PreimageOracleClient, errors::PreimageOracleResult,
    };
    use kona_proof::block_on;
    use kona_sp1_client_utils::{
        super_root::{
            SuperConsolidationTransitionInput, SuperOptimisticBlock, SuperOutputRoot,
            SuperRootProof, TimestampSpan,
        },
        witness::preimage_store::PreimageStore,
    };

    use super::*;
    use crate::test_utils::{b256, dependency_set, rollup_config, save_header, save_output_root};

    fn save_super_root(oracle: &mut PreimageStore, super_root: &SuperRoot) -> B256 {
        let hash = super_root.hash();
        let mut encoded = Vec::new();
        super_root.encode(&mut encoded);
        oracle.save_preimage(PreimageKey::new_keccak256(*hash), encoded).unwrap();
        hash
    }

    fn save_empty_trie(oracle: &mut PreimageStore) {
        oracle
            .save_preimage(PreimageKey::new_keccak256(*EMPTY_ROOT_HASH), vec![EMPTY_STRING_CODE])
            .unwrap();
    }

    fn rollup_configs(chain_ids: &[u64]) -> RegistryHashMap<u64, RollupConfig> {
        chain_ids.iter().map(|chain_id| (*chain_id, rollup_config(*chain_id, 1))).collect()
    }

    fn save_fallback_chain_config(oracle: &mut PreimageStore, chain_id: u64) {
        oracle
            .save_preimage(
                PreimageKey::new_local(DEPENDENCY_SET_KEY.to()),
                serde_json::to_vec(&dependency_set(&[chain_id], None)).unwrap(),
            )
            .unwrap();
        oracle
            .save_preimage(
                PreimageKey::new_local(L2_ROLLUP_CONFIG_KEY.to()),
                serde_json::to_vec(&rollup_configs(&[chain_id])).unwrap(),
            )
            .unwrap();
    }

    #[derive(Clone, Debug)]
    struct RecordingOracle {
        inner: PreimageStore,
        requests: Arc<Mutex<Vec<PreimageKey>>>,
    }

    impl RecordingOracle {
        fn new(inner: PreimageStore) -> Self {
            Self { inner, requests: Arc::new(Mutex::new(Vec::new())) }
        }

        fn request_count(&self, key: PreimageKey) -> usize {
            self.requests.lock().unwrap().iter().filter(|request| **request == key).count()
        }
    }

    #[async_trait]
    impl PreimageOracleClient for RecordingOracle {
        async fn get(&self, key: PreimageKey) -> PreimageOracleResult<Vec<u8>> {
            self.requests.lock().unwrap().push(key);
            self.inner.get(key).await
        }

        async fn get_exact(&self, key: PreimageKey, buf: &mut [u8]) -> PreimageOracleResult<()> {
            self.requests.lock().unwrap().push(key);
            self.inner.get_exact(key, buf).await
        }
    }

    #[async_trait]
    impl HintWriterClient for RecordingOracle {
        async fn write(&self, _hint: &str) -> PreimageOracleResult<()> {
            Ok(())
        }
    }

    #[test]
    fn unresolved_consolidation_config_takes_full_path() {
        let previous_super_root = SuperRoot::new(
            99,
            vec![SuperOutputRoot { chain_id: u64::MAX, output_root: B256::ZERO }],
        );

        assert!(interop_active_for_consolidation(&ROLLUP_CONFIGS, &previous_super_root, 100));
    }

    fn run_inactive_transition(
        claimed_output_root: B256,
    ) -> anyhow::Result<SuperConsolidationTransition> {
        let chain_id = 10;
        let previous_super_root =
            SuperRoot::new(100, vec![SuperOutputRoot { chain_id, output_root: b256(0x44) }]);
        let optimistic_output_root = b256(0x55);
        let optimistic_blocks = vec![SuperOptimisticBlock {
            chain_id: U256::from(chain_id),
            block_hash: b256(0x22),
            output_root: optimistic_output_root,
        }];
        let claimed_super_root_proof = SuperRootProof::new(
            101,
            vec![SuperOutputRoot { chain_id, output_root: claimed_output_root }],
        );

        block_on(run_transition(
            Arc::new(PreimageStore::default()),
            previous_super_root,
            optimistic_blocks,
            &claimed_super_root_proof,
            dependency_set(&[chain_id], None),
            &rollup_configs(&[chain_id]),
            &Default::default(),
        ))
    }

    #[test]
    fn inactive_consolidation_finalizes_without_provider_preimages() {
        let transition = run_inactive_transition(b256(0x55)).unwrap();
        let expected = hash_super_root_proof(&SuperRootProof::new(
            101,
            vec![SuperOutputRoot { chain_id: 10, output_root: b256(0x55) }],
        ))
        .unwrap();

        assert_eq!(transition.super_root, expected);
    }

    #[test]
    fn inactive_consolidation_rejects_mismatched_claim() {
        let err = run_inactive_transition(b256(0x66)).unwrap_err();

        assert!(err.to_string().contains("does not match claim"), "unexpected error: {err}");
    }

    #[test]
    fn fetch_super_root_decodes_witness_preimage_bound_to_public_hash() {
        let super_root =
            SuperRoot::new(100, vec![SuperOutputRoot { chain_id: 10, output_root: b256(0x44) }]);
        let mut oracle = PreimageStore::default();
        let super_root_hash = save_super_root(&mut oracle, &super_root);

        let decoded = block_on(fetch_super_root(&oracle, super_root_hash)).unwrap();

        assert_eq!(decoded, super_root);
    }

    #[test]
    fn consolidation_previous_root_must_match_optimistic_chain_coverage() {
        let previous_super_root =
            SuperRoot::new(100, vec![SuperOutputRoot { chain_id: 10, output_root: b256(0x44) }]);
        let optimistic_blocks = vec![SuperOptimisticBlock {
            chain_id: U256::from(20),
            block_hash: b256(0x22),
            output_root: b256(0x33),
        }];

        let err = ensure_previous_super_root_matches_optimistic_blocks(
            &previous_super_root,
            &optimistic_blocks,
        )
        .unwrap_err();

        assert!(
            err.to_string().contains("does not match optimistic chain"),
            "unexpected error: {err}"
        );
    }

    #[test]
    fn consolidation_providers_bind_optimistic_output_root_to_block_hash() {
        let safe_head = Header { number: 3, timestamp: 100, ..Default::default() };
        let claimed_head = Header { number: 4, timestamp: 101, ..Default::default() };
        let mut oracle = PreimageStore::default();
        let safe_head_hash = save_header(&mut oracle, &safe_head);
        let claimed_head_hash = save_header(&mut oracle, &claimed_head);
        let previous_output_root = save_output_root(&mut oracle, safe_head_hash);
        let optimistic_output_root = save_output_root(&mut oracle, claimed_head_hash);
        let previous_super_root = SuperRoot::new(
            100,
            vec![SuperOutputRoot { chain_id: 10, output_root: previous_output_root }],
        );
        let optimistic_blocks = vec![SuperOptimisticBlock {
            chain_id: U256::from(10),
            block_hash: b256(0xaa),
            output_root: optimistic_output_root,
        }];

        let err = block_on(build_providers(
            Arc::new(oracle),
            &previous_super_root,
            &optimistic_blocks,
            101,
            &rollup_configs(&[10]),
        ))
        .unwrap_err();

        assert!(err.to_string().contains("commits to block hash"), "unexpected error: {err}");
    }

    #[test]
    fn consolidation_providers_reject_future_optimistic_header_timestamp() {
        let safe_head = Header { number: 3, timestamp: 100, ..Default::default() };
        let future_head = Header { number: 4, timestamp: 102, ..Default::default() };
        let mut oracle = PreimageStore::default();
        let safe_head_hash = save_header(&mut oracle, &safe_head);
        let future_head_hash = save_header(&mut oracle, &future_head);
        let previous_output_root = save_output_root(&mut oracle, safe_head_hash);
        let optimistic_output_root = save_output_root(&mut oracle, future_head_hash);
        let previous_super_root = SuperRoot::new(
            100,
            vec![SuperOutputRoot { chain_id: 10, output_root: previous_output_root }],
        );
        let optimistic_blocks = vec![SuperOptimisticBlock {
            chain_id: U256::from(10),
            block_hash: future_head_hash,
            output_root: optimistic_output_root,
        }];

        let err = block_on(build_providers(
            Arc::new(oracle),
            &previous_super_root,
            &optimistic_blocks,
            101,
            &rollup_configs(&[10]),
        ))
        .unwrap_err();

        assert!(
            err.to_string().contains("is after super-root timestamp"),
            "unexpected error: {err}"
        );
    }

    #[test]
    fn consolidation_outputs_chain_claimed_root_across_timestamps() {
        let chain_id = u64::MAX;
        let previous_head = Header {
            number: 3,
            timestamp: 99,
            receipts_root: EMPTY_ROOT_HASH,
            transactions_root: EMPTY_ROOT_HASH,
            ..Default::default()
        };
        let mut oracle = PreimageStore::default();
        save_empty_trie(&mut oracle);
        save_fallback_chain_config(&mut oracle, chain_id);

        let previous_head_hash = save_header(&mut oracle, &previous_head);
        let first_head = Header {
            number: 4,
            timestamp: 100,
            parent_hash: previous_head_hash,
            receipts_root: EMPTY_ROOT_HASH,
            transactions_root: EMPTY_ROOT_HASH,
            ..Default::default()
        };
        let first_head_hash = save_header(&mut oracle, &first_head);
        let second_head = Header {
            number: 5,
            timestamp: 101,
            parent_hash: first_head_hash,
            receipts_root: EMPTY_ROOT_HASH,
            transactions_root: EMPTY_ROOT_HASH,
            ..Default::default()
        };
        let second_head_hash = save_header(&mut oracle, &second_head);

        let previous_output_root = save_output_root(&mut oracle, previous_head_hash);
        let first_output_root = save_output_root(&mut oracle, first_head_hash);
        let second_output_root = save_output_root(&mut oracle, second_head_hash);
        let previous_super_root = SuperRoot::new(
            99,
            vec![SuperOutputRoot { chain_id, output_root: previous_output_root }],
        );
        let previous_super_root_hash = save_super_root(&mut oracle, &previous_super_root);
        let first_claim = SuperRootProof::new(
            100,
            vec![SuperOutputRoot { chain_id, output_root: first_output_root }],
        );
        let second_claim = SuperRootProof::new(
            101,
            vec![SuperOutputRoot { chain_id, output_root: second_output_root }],
        );
        let inputs = SuperConsolidationInputs {
            span: TimestampSpan::new(100, 101).unwrap(),
            previous_super_root: previous_super_root_hash,
            transitions: vec![
                SuperConsolidationTransitionInput {
                    optimistic_blocks: vec![SuperOptimisticBlock {
                        chain_id: U256::from(chain_id),
                        block_hash: first_head_hash,
                        output_root: first_output_root,
                    }],
                    claimed_super_root_proof: first_claim.clone(),
                },
                SuperConsolidationTransitionInput {
                    optimistic_blocks: vec![SuperOptimisticBlock {
                        chain_id: U256::from(chain_id),
                        block_hash: second_head_hash,
                        output_root: second_output_root,
                    }],
                    claimed_super_root_proof: second_claim.clone(),
                },
            ],
        };
        let oracle = RecordingOracle::new(oracle);

        let outputs = block_on(build_consolidation_outputs(inputs, Arc::new(oracle.clone())))
            .expect("two-timestamp consolidation succeeds");

        assert_eq!(
            outputs.transitions.iter().map(|transition| transition.super_root).collect::<Vec<_>>(),
            vec![
                hash_super_root_proof(&first_claim).unwrap(),
                hash_super_root_proof(&second_claim).unwrap(),
            ]
        );
        assert_eq!(
            oracle.request_count(PreimageKey::new_keccak256(*first_output_root)),
            2,
            "the intermediate output is read once as the first optimistic output and again as the second transition's pre-state"
        );
    }

    #[test]
    fn consolidation_outputs_reject_starting_root_before_span_predecessor() {
        let chain_id = u64::MAX;
        let mut oracle = PreimageStore::default();
        save_fallback_chain_config(&mut oracle, chain_id);
        let stale_super_root =
            SuperRoot::new(98, vec![SuperOutputRoot { chain_id, output_root: b256(0x44) }]);
        let stale_super_root_hash = save_super_root(&mut oracle, &stale_super_root);
        let inputs = SuperConsolidationInputs {
            span: TimestampSpan::new(100, 100).unwrap(),
            previous_super_root: stale_super_root_hash,
            transitions: vec![SuperConsolidationTransitionInput {
                optimistic_blocks: vec![SuperOptimisticBlock {
                    chain_id: U256::from(chain_id),
                    block_hash: b256(0x22),
                    output_root: b256(0x33),
                }],
                claimed_super_root_proof: SuperRootProof::new(
                    100,
                    vec![SuperOutputRoot { chain_id, output_root: b256(0x55) }],
                ),
            }],
        };
        inputs.validate().expect("typed consolidation timestamps are valid");

        let err = block_on(build_consolidation_outputs(inputs, Arc::new(oracle))).unwrap_err();

        assert!(
            err.to_string().contains(
                "previous super-root timestamp 98 does not precede claimed timestamp 100"
            ),
            "unexpected error: {err}"
        );
    }
}
