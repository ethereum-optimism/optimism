//! Shared execution logic for the SP1 super-range guest's range mode.

use std::{collections::BTreeMap, fmt::Debug, ops::Range, sync::Arc};

use alloy_consensus::{Header, Sealed};
use alloy_primitives::{B256, U256};
use alloy_rlp::Decodable;
use anyhow::{anyhow, bail, ensure};
use kona_derive::BlobProvider;
use kona_genesis::{L1ChainConfig, RollupConfig};
use kona_interop::DependencySet;
use kona_preimage::{CommsClient, PreimageKey, PreimageOracleClient};
use kona_proof::{
    BootInfo, FlushableCache, l1::OracleL1ChainProvider, l2::OracleL2ChainProvider,
    sync::new_oracle_pipeline_cursor,
};
use kona_proof_interop::HintType;
use kona_sp1_client_utils::{
    boot::BootInfoStruct,
    super_root::{
        SuperOptimisticBlock, SuperRangeInputs, SuperRangeOutputs, SuperRangeTransition,
        hash_super_root_proof,
    },
    witness::executor::{BlockClaim, SegmentClaims, WitnessExecutor},
};

use crate::{chain_config::ChainConfigs, executor::ETHDAWitnessExecutor};

const OUTPUT_ROOT_WORD_BYTES: usize = 32;
const OUTPUT_ROOT_V0_BYTES: usize = 4 * OUTPUT_ROOT_WORD_BYTES;
const OUTPUT_ROOT_V0_VERSION_RANGE: Range<usize> = 0..OUTPUT_ROOT_WORD_BYTES;
const OUTPUT_ROOT_V0_BLOCK_HASH_RANGE: Range<usize> =
    3 * OUTPUT_ROOT_WORD_BYTES..OUTPUT_ROOT_V0_BYTES;

/// Builds range outputs; `None` uses the embedded registry (see [`ChainConfigs`]).
pub async fn build_range_outputs<O, B>(
    inputs: SuperRangeInputs,
    oracle: Arc<O>,
    beacon: B,
    configs: Option<&ChainConfigs>,
) -> anyhow::Result<SuperRangeOutputs>
where
    O: CommsClient + FlushableCache + Send + Sync + Debug + 'static,
    B: BlobProvider + Send + Sync + Debug + Clone + 'static,
{
    inputs.validate()?;
    let previous_super_roots = inputs
        .previous_super_root_proofs
        .iter()
        .map(hash_super_root_proof)
        .collect::<Result<Vec<_>, _>>()?;

    let embedded_configs;
    let configs = match configs {
        Some(configs) => {
            configs.validate(&inputs.chain_ids)?;
            configs
        }
        None => {
            embedded_configs = ChainConfigs::from_registry(&inputs.chain_ids)?;
            &embedded_configs
        }
    };
    let dependency_set = Arc::new(configs.dependency_set.clone());
    let rollup_configs = &configs.rollup_configs;
    let l1_config = &configs.l1_config;

    for segment in range_segments(&inputs)? {
        let boot_infos = run_super_range_segment(
            &inputs,
            &segment,
            oracle.clone(),
            beacon.clone(),
            dependency_set.clone(),
            rollup_configs,
            l1_config,
        )
        .await?;
        for (transition, boot_info) in segment.into_iter().zip(boot_infos) {
            validate_range_transition_output(transition, oracle.as_ref(), &boot_info).await?;
        }
    }

    Ok(SuperRangeOutputs {
        span: inputs.span,
        l1_head: inputs.l1_head,
        previous_super_roots,
        transitions: inputs.claimed_transitions,
    })
}

fn range_segments(inputs: &SuperRangeInputs) -> anyhow::Result<Vec<Vec<&SuperRangeTransition>>> {
    let mut segments: Vec<Vec<&SuperRangeTransition>> = Vec::new();
    let mut last_segment_by_chain = BTreeMap::<U256, usize>::new();
    for transition in &inputs.claimed_transitions {
        let chain_id = transition.optimistic_block.chain_id;
        let agreed_root = previous_output_root(inputs, transition)?;
        if let Some(&index) = last_segment_by_chain.get(&chain_id) &&
            segments[index]
                .last()
                .is_some_and(|previous| previous.optimistic_block.output_root == agreed_root)
        {
            segments[index].push(transition);
            continue;
        }
        last_segment_by_chain.insert(chain_id, segments.len());
        segments.push(vec![transition]);
    }
    Ok(segments)
}

#[derive(Debug)]
#[allow(clippy::large_enum_variant)]
enum RangeTransitionBoot {
    NoOp { boot: BootInfo },
    Progress { boot: BootInfo, safe_head_hash: B256, safe_head: Header },
}

async fn run_super_range_segment<O, B>(
    inputs: &SuperRangeInputs,
    transitions: &[&SuperRangeTransition],
    oracle: Arc<O>,
    beacon: B,
    dependency_set: Arc<DependencySet>,
    rollup_configs: &BTreeMap<u64, RollupConfig>,
    l1_config: &L1ChainConfig,
) -> anyhow::Result<Vec<BootInfoStruct>>
where
    O: CommsClient + FlushableCache + Send + Sync + Debug + 'static,
    B: BlobProvider + Send + Sync + Debug + Clone + 'static,
{
    let mut segment: Option<(SegmentClaims, Sealed<Header>)> = None;
    let mut boot_infos = Vec::with_capacity(transitions.len());
    for transition in transitions {
        let transition_boot = build_super_range_transition_boot(
            inputs,
            transition,
            oracle.as_ref(),
            rollup_configs,
            l1_config,
        )
        .await?;
        match transition_boot {
            RangeTransitionBoot::NoOp { boot } => boot_infos.push(BootInfoStruct::from(boot)),
            RangeTransitionBoot::Progress { boot, safe_head_hash, safe_head } => {
                if let Some((claims, _)) = &mut segment {
                    claims.following.push(BlockClaim::from(&boot));
                    boot_infos.push(BootInfoStruct::from(boot));
                } else {
                    boot_infos.push(BootInfoStruct::from(boot.clone()));
                    segment = Some((
                        SegmentClaims { first: boot, following: Vec::new() },
                        Sealed::new_unchecked(safe_head, safe_head_hash),
                    ));
                }
            }
        }
    }

    let Some((claims, safe_head)) = segment else {
        return Ok(boot_infos);
    };
    let boot = &claims.first;

    let rollup_config = Arc::new(boot.rollup_config.clone());
    let l1_config = Arc::new(boot.l1_config.clone());
    let mut l1_provider = OracleL1ChainProvider::new(boot.l1_head, oracle.clone());
    let mut l2_provider =
        OracleL2ChainProvider::new(safe_head.hash(), rollup_config.clone(), oracle.clone());
    l2_provider.set_chain_id(Some(boot.chain_id));

    let cursor = new_oracle_pipeline_cursor(
        rollup_config.as_ref(),
        safe_head,
        boot.agreed_l2_output_root,
        &mut l1_provider,
        &mut l2_provider,
    )
    .await?;
    l2_provider.set_cursor(cursor.clone());

    let executor = ETHDAWitnessExecutor::new_with_dependency_set(dependency_set);
    let pipeline = executor
        .create_pipeline(
            rollup_config,
            l1_config,
            cursor.clone(),
            oracle,
            beacon,
            l1_provider,
            l2_provider.clone(),
        )
        .await?;
    executor.run(&claims, pipeline, cursor, l2_provider).await?;

    Ok(boot_infos)
}

async fn build_super_range_transition_boot<O>(
    inputs: &SuperRangeInputs,
    transition: &SuperRangeTransition,
    oracle: &O,
    rollup_configs: &BTreeMap<u64, RollupConfig>,
    l1_config: &L1ChainConfig,
) -> anyhow::Result<RangeTransitionBoot>
where
    O: CommsClient,
{
    let chain_id = transition_chain_id_as_u64(transition)?;
    let rollup_config = rollup_configs
        .get(&chain_id)
        .ok_or_else(|| anyhow!("missing rollup config for super-range chain ID {chain_id}"))?
        .clone();
    let expected_pre_root = previous_output_root(inputs, transition)?;
    let claimed_output_root = transition.optimistic_block.output_root;
    let claimed_block_hash = transition.optimistic_block.block_hash;

    let safe_head_hash = fetch_output_block_hash(oracle, expected_pre_root).await?;
    let safe_head = fetch_l2_header(oracle, safe_head_hash, chain_id).await?;
    let output_claimed_block_hash = fetch_output_block_hash(oracle, claimed_output_root).await?;
    ensure!(
        output_claimed_block_hash == claimed_block_hash,
        "claimed optimistic output root commits to block hash {actual}, expected {expected}",
        actual = output_claimed_block_hash,
        expected = claimed_block_hash,
    );
    HintType::L2BlockData
        .with_data(&[
            safe_head_hash.as_slice(),
            claimed_block_hash.as_slice(),
            chain_id.to_be_bytes().as_ref(),
        ])
        .send(oracle)
        .await?;
    let claimed_header = fetch_l2_header(oracle, claimed_block_hash, chain_id).await?;

    let next_l2_timestamp = safe_head
        .timestamp
        .checked_add(rollup_config.block_time)
        .ok_or_else(|| anyhow!("next L2 timestamp overflows for chain {chain_id}"))?;

    if next_l2_timestamp > transition.timestamp {
        ensure!(
            claimed_output_root == expected_pre_root,
            "no-op super-range transition changed output root from {pre} to {post}",
            pre = expected_pre_root,
            post = claimed_output_root,
        );
        ensure!(
            claimed_block_hash == safe_head_hash,
            "no-op super-range transition block hash {actual} does not match safe head {expected}",
            actual = claimed_block_hash,
            expected = safe_head_hash,
        );
        ensure!(
            claimed_header.number == safe_head.number,
            "no-op super-range transition must target safe head block #{expected}, got #{actual}",
            expected = safe_head.number,
            actual = claimed_header.number,
        );
        let boot = BootInfo {
            l1_head: inputs.l1_head,
            agreed_l2_output_root: expected_pre_root,
            claimed_l2_output_root: claimed_output_root,
            claimed_l2_block_number: safe_head.number,
            chain_id,
            rollup_config,
            l1_config: l1_config.clone(),
        };
        return Ok(RangeTransitionBoot::NoOp { boot });
    }

    ensure!(
        next_l2_timestamp == transition.timestamp,
        "progressing super-range transition must target timestamp {expected}, got {actual}",
        expected = next_l2_timestamp,
        actual = transition.timestamp,
    );

    let expected_block_number = safe_head
        .number
        .checked_add(1)
        .ok_or_else(|| anyhow!("next L2 block number overflows for chain {chain_id}"))?;
    ensure!(
        claimed_header.number == expected_block_number,
        "progressing super-range transition must target next L2 block #{expected}, got #{actual}",
        expected = expected_block_number,
        actual = claimed_header.number,
    );
    ensure!(
        claimed_header.timestamp == transition.timestamp,
        "progressing super-range transition block timestamp {actual} does not match target {expected}",
        actual = claimed_header.timestamp,
        expected = transition.timestamp,
    );

    let boot = BootInfo {
        l1_head: inputs.l1_head,
        agreed_l2_output_root: expected_pre_root,
        claimed_l2_output_root: claimed_output_root,
        claimed_l2_block_number: claimed_header.number,
        chain_id,
        rollup_config,
        l1_config: l1_config.clone(),
    };

    Ok(RangeTransitionBoot::Progress { boot, safe_head_hash, safe_head })
}

async fn validate_range_transition_output<O>(
    transition: &SuperRangeTransition,
    oracle: &O,
    committed_boot_info: &BootInfoStruct,
) -> anyhow::Result<()>
where
    O: CommsClient,
{
    ensure!(
        committed_boot_info.l2PostRoot == transition.optimistic_block.output_root,
        "range program committed output root {actual}, expected {expected}",
        actual = committed_boot_info.l2PostRoot,
        expected = transition.optimistic_block.output_root,
    );

    let block_hash = fetch_output_block_hash(oracle, committed_boot_info.l2PostRoot).await?;
    ensure!(
        block_hash == transition.optimistic_block.block_hash,
        "output root commits to block hash {actual}, expected {expected}",
        actual = block_hash,
        expected = transition.optimistic_block.block_hash,
    );

    let header = fetch_l2_header(
        oracle,
        block_hash,
        optimistic_chain_id_as_u64(&transition.optimistic_block)?,
    )
    .await?;
    ensure!(
        header.number == committed_boot_info.l2BlockNumber,
        "range witness committed block #{actual}, but output root header is block #{expected}",
        actual = committed_boot_info.l2BlockNumber,
        expected = header.number,
    );
    ensure!(
        header.timestamp <= transition.timestamp,
        "range witness block timestamp {actual} is after super-range timestamp {expected}",
        actual = header.timestamp,
        expected = transition.timestamp,
    );

    Ok(())
}

fn transition_chain_id_as_u64(transition: &SuperRangeTransition) -> anyhow::Result<u64> {
    optimistic_chain_id_as_u64(&transition.optimistic_block)
}

/// Converts a super optimistic block chain ID into a host `u64`.
pub fn optimistic_chain_id_as_u64(optimistic_block: &SuperOptimisticBlock) -> anyhow::Result<u64> {
    let chain_id = optimistic_block.chain_id;
    if chain_id > U256::from(u64::MAX) {
        bail!("super-range optimistic block chain ID {chain_id} does not fit in u64");
    }
    Ok(chain_id.saturating_to::<u64>())
}

fn previous_output_root(
    inputs: &SuperRangeInputs,
    transition: &SuperRangeTransition,
) -> anyhow::Result<B256> {
    let offset = transition
        .timestamp
        .checked_sub(inputs.span.start)
        .and_then(|offset| usize::try_from(offset).ok())
        .ok_or_else(|| {
            anyhow!(
                "transition timestamp {} is outside range span {}..={}",
                transition.timestamp,
                inputs.span.start,
                inputs.span.end
            )
        })?;
    let previous_proof = inputs
        .previous_super_root_proofs
        .get(offset)
        .ok_or_else(|| anyhow!("missing previous super-root proof at offset {offset}"))?;
    previous_proof
        .super_root
        .output_roots
        .iter()
        .find(|output_root| {
            U256::from(output_root.chain_id) == transition.optimistic_block.chain_id
        })
        .map(|output_root| output_root.output_root)
        .ok_or_else(|| {
            anyhow!(
                "previous super-root proof at timestamp {} does not cover chain {}",
                previous_proof.super_root.timestamp,
                transition.optimistic_block.chain_id
            )
        })
}

/// Fetches a V0 output-root preimage and returns the L2 block hash it commits to.
///
/// V0 consists of four consecutive 32-byte words:
/// `version || state_root || message_passer_storage_root || block_hash`.
pub async fn fetch_output_block_hash<O>(oracle: &O, output_root: B256) -> anyhow::Result<B256>
where
    O: PreimageOracleClient,
{
    let output_preimage = oracle
        .get(PreimageKey::new_keccak256(*output_root))
        .await
        .map_err(|err| anyhow!("failed to fetch output-root preimage {output_root}: {err}"))?;
    ensure!(
        output_preimage.len() == OUTPUT_ROOT_V0_BYTES,
        "output-root preimage {output_root} has length {}, expected {OUTPUT_ROOT_V0_BYTES}",
        output_preimage.len(),
    );
    if output_preimage[OUTPUT_ROOT_V0_VERSION_RANGE].iter().any(|byte| *byte != 0) {
        bail!(
            "output-root preimage {output_root} has unsupported version {}",
            B256::from_slice(&output_preimage[OUTPUT_ROOT_V0_VERSION_RANGE]),
        );
    }

    Ok(B256::from_slice(&output_preimage[OUTPUT_ROOT_V0_BLOCK_HASH_RANGE]))
}

/// Fetches and decodes an L2 header through the interop hint path.
pub async fn fetch_l2_header<O>(
    oracle: &O,
    block_hash: B256,
    chain_id: u64,
) -> anyhow::Result<Header>
where
    O: CommsClient,
{
    HintType::L2BlockHeader
        .with_data(&[block_hash.as_slice(), chain_id.to_be_bytes().as_ref()])
        .send(oracle)
        .await?;
    let header_rlp = oracle
        .get(PreimageKey::new_keccak256(*block_hash))
        .await
        .map_err(|err| anyhow!("failed to fetch L2 header {block_hash}: {err}"))?;
    let header = Header::decode(&mut header_rlp.as_slice())
        .map_err(|err| anyhow!("failed to decode L2 header {block_hash}: {err}"))?;
    ensure!(
        header.hash_slow() == block_hash,
        "decoded L2 header hash {} does not match expected {block_hash}",
        header.hash_slow(),
    );

    Ok(header)
}

#[cfg(test)]
mod tests {
    use alloy_consensus::Header;
    use alloy_eips::BlockNumHash;
    use alloy_primitives::{B256, FixedBytes, U256};
    use alloy_trie::EMPTY_ROOT_HASH;
    use kona_genesis::RollupConfig;
    use kona_preimage::PreimageKey;
    use kona_proof::block_on;
    use kona_protocol::{BatchValidity, BlockInfo, L2BlockInfo, SpanBatch, SpanBatchElement};
    use kona_sp1_client_utils::{
        super_root::{
            SuperOptimisticBlock, SuperOutputRoot, SuperRangeInputs, SuperRangeTransition,
            SuperRootProof, TimestampSpan,
        },
        witness::{BlobData, DefaultWitnessData, WitnessData, preimage_store::PreimageStore},
    };

    use super::*;
    use crate::test_utils::{
        b256, chain_configs, dependency_set, rollup_config, save_header, save_output_root,
    };
    use kona_preimage::{DEPENDENCY_SET_KEY, L1_CONFIG_KEY, L2_ROLLUP_CONFIG_KEY};
    use kona_registry::L1_CONFIGS;

    #[test]
    fn range_outputs_reject_untrusted_preimage_configs() {
        for (chain_ids, expected_error) in [
            (vec![u64::MAX], "no embedded dependency set"),
            (vec![10, u64::MAX], "no embedded rollup config"),
        ] {
            let configs = chain_configs(&chain_ids);
            let mut oracle = PreimageStore::default();
            for (key, serialized) in [
                (DEPENDENCY_SET_KEY, serde_json::to_vec(&configs.dependency_set).unwrap()),
                (L2_ROLLUP_CONFIG_KEY, serde_json::to_vec(&configs.rollup_configs).unwrap()),
                (L1_CONFIG_KEY, serde_json::to_vec(&configs.l1_config).unwrap()),
            ] {
                oracle.save_preimage(PreimageKey::new_local(key.to()), serialized).unwrap();
            }
            let err = block_on(build_range_outputs(
                range_inputs_for_chain_ids(&chain_ids),
                Arc::new(oracle),
                kona_sp1_client_utils::BlobStore::default(),
                None,
            ))
            .expect_err("witness configs must not authorize an unknown chain");
            assert!(err.to_string().contains(expected_error), "unexpected error: {err}");
        }
    }

    fn save_range_state(oracle: &mut PreimageStore, depositor_nonce: u64) -> B256 {
        use alloy_primitives::{address, keccak256};
        use alloy_trie::{Nibbles, TrieAccount};
        use kona_mpt::{NoopTrieProvider, TrieNode};
        use kona_protocol::Predeploys;

        fn save_node(oracle: &mut PreimageStore, node: &TrieNode) {
            oracle
                .save_preimage(PreimageKey::new_keccak256(*node.blind()), alloy_rlp::encode(node))
                .unwrap();
            match node {
                TrieNode::Extension { node, .. } => save_node(oracle, node),
                TrieNode::Branch { stack } => {
                    for child in stack {
                        save_node(oracle, child);
                    }
                }
                _ => {}
            }
        }

        let mut root = TrieNode::Empty;
        for (address, nonce) in [
            (Predeploys::L2_TO_L1_MESSAGE_PASSER, 1),
            (address!("deaddeaddeaddeaddeaddeaddeaddeaddead0001"), depositor_nonce),
        ] {
            if nonce == 0 {
                continue;
            }
            root.insert(
                &Nibbles::unpack(keccak256(address)),
                alloy_rlp::encode(TrieAccount { nonce, ..Default::default() }).into(),
                &NoopTrieProvider,
            )
            .unwrap();
        }
        save_node(oracle, &root);
        root.blind()
    }

    fn save_range_transaction(oracle: &mut PreimageStore, tx: &[u8]) {
        let mut trie = kona_mpt::ordered_trie_with_encoder(&[tx], |tx, out| out.put_slice(tx));
        trie.root();
        for node in trie.take_proof_nodes().into_inner().into_values() {
            oracle
                .save_preimage(
                    PreimageKey::new_keccak256(*alloy_primitives::keccak256(&node)),
                    node.to_vec(),
                )
                .unwrap();
        }
    }

    /// A Bedrock chain with two deposit-only blocks and a no-op timestamp between them.
    /// Expiring the sequencing window supplies empty batches without external DA fixtures.
    fn progressing_range_fixture() -> (SuperRangeInputs, PreimageStore, ChainConfigs) {
        use alloy_eips::Encodable2718;
        use alloy_op_evm::{OpEvmFactory, block::OpAlloyReceiptBuilder};
        use alloy_primitives::keccak256;
        use kona_executor::StatelessL2Builder;
        use kona_genesis::SystemConfig;
        use kona_protocol::{L1BlockInfoTx, Predeploys};
        use op_alloy_rpc_types_engine::OpPayloadAttributes;

        let chain_id = u64::MAX;
        let mut oracle = PreimageStore::default();
        oracle.save_preimage(PreimageKey::new_keccak256(*EMPTY_ROOT_HASH), vec![0x80]).unwrap();
        let l1_genesis = Header {
            number: 1,
            timestamp: 100,
            transactions_root: EMPTY_ROOT_HASH,
            receipts_root: EMPTY_ROOT_HASH,
            base_fee_per_gas: Some(1),
            ..Default::default()
        };
        let l1_genesis_hash = save_header(&mut oracle, &l1_genesis);
        let l1_head = save_header(
            &mut oracle,
            &Header {
                parent_hash: l1_genesis_hash,
                number: 2,
                timestamp: 112,
                transactions_root: EMPTY_ROOT_HASH,
                receipts_root: EMPTY_ROOT_HASH,
                ..Default::default()
            },
        );
        let genesis = Header {
            number: 0,
            timestamp: 100,
            state_root: save_range_state(&mut oracle, 0),
            transactions_root: EMPTY_ROOT_HASH,
            receipts_root: EMPTY_ROOT_HASH,
            gas_limit: 30_000_000,
            base_fee_per_gas: Some(1),
            ..Default::default()
        };
        let genesis_hash = save_header(&mut oracle, &genesis);
        let mut config = rollup_config(chain_id, 1);
        config.block_time = 2;
        config.seq_window_size = 1;
        config.genesis.l1 = alloy_eips::BlockNumHash { number: 1, hash: l1_genesis_hash };
        config.genesis.l2 = alloy_eips::BlockNumHash { number: 0, hash: genesis_hash };
        config.genesis.l2_time = 100;
        config.genesis.system_config =
            Some(SystemConfig { gas_limit: genesis.gas_limit, ..Default::default() });
        let mut headers = vec![genesis];
        for sequence in 1..=2 {
            let (_, tx) = L1BlockInfoTx::try_new_with_deposit_tx(
                &config,
                L1_CONFIGS.get(&1).unwrap(),
                &config.genesis.system_config.unwrap(),
                sequence,
                &l1_genesis,
                100 + 2 * sequence,
            )
            .unwrap();
            let tx = tx.encoded_2718();
            let mut attrs = OpPayloadAttributes::default();
            attrs.payload_attributes.timestamp = 100 + 2 * sequence;
            attrs.payload_attributes.prev_randao = l1_genesis.mix_hash;
            attrs.payload_attributes.suggested_fee_recipient = Predeploys::SEQUENCER_FEE_VAULT;
            attrs.transactions = Some(vec![tx.clone().into()]);
            attrs.no_tx_pool = Some(true);
            attrs.gas_limit = Some(headers[0].gas_limit);
            let provider = OracleL2ChainProvider::new(
                headers.last().unwrap().hash_slow(),
                Arc::new(config.clone()),
                Arc::new(oracle.clone()),
            );
            let mut builder = StatelessL2Builder::new(
                &config,
                OpEvmFactory::<alloy_op_evm::OpTx>::default(),
                OpAlloyReceiptBuilder::default(),
                provider.clone(),
                provider,
                Sealed::new(headers.last().unwrap().clone()),
            );
            let header = builder.build_block(attrs).unwrap().header;
            assert_eq!(header.state_root, save_range_state(&mut oracle, sequence));
            save_range_transaction(&mut oracle, &tx);
            save_header(&mut oracle, &header);
            headers.push(header.into_inner());
        }
        let blocks = headers
            .iter()
            .map(|header| {
                let mut preimage = vec![0; OUTPUT_ROOT_V0_BYTES];
                preimage[32..64].copy_from_slice(header.state_root.as_slice());
                preimage[64..96].copy_from_slice(EMPTY_ROOT_HASH.as_slice());
                preimage[96..128].copy_from_slice(header.hash_slow().as_slice());
                let output_root = keccak256(&preimage);
                oracle.save_preimage(PreimageKey::new_keccak256(*output_root), preimage).unwrap();
                SuperOptimisticBlock {
                    chain_id: U256::from(chain_id),
                    block_hash: header.hash_slow(),
                    output_root,
                }
            })
            .collect::<Vec<_>>();
        let inputs = SuperRangeInputs {
            span: TimestampSpan::new(102, 104).unwrap(),
            l1_head,
            chain_ids: vec![U256::from(chain_id)],
            previous_super_root_proofs: [0, 1, 1]
                .into_iter()
                .enumerate()
                .map(|(offset, block)| {
                    SuperRootProof::new(
                        101 + offset as u64,
                        vec![SuperOutputRoot { chain_id, output_root: blocks[block].output_root }],
                    )
                })
                .collect(),
            claimed_transitions: [1, 1, 2]
                .into_iter()
                .enumerate()
                .map(|(offset, block)| SuperRangeTransition {
                    timestamp: 102 + offset as u64,
                    optimistic_block: blocks[block],
                })
                .collect(),
        };
        let configs = ChainConfigs {
            dependency_set: dependency_set(&[chain_id], None),
            rollup_configs: BTreeMap::from([(chain_id, config)]),
            l1_config: L1_CONFIGS[&1].clone(),
        };
        (inputs, oracle, configs)
    }

    #[test]
    fn range_outputs_derive_multiple_blocks_across_noop() {
        let (inputs, oracle, configs) = progressing_range_fixture();
        let expected = inputs.claimed_transitions.clone();
        let actual = block_on(build_range_outputs(
            inputs,
            Arc::new(oracle),
            kona_sp1_client_utils::BlobStore::default(),
            Some(&configs),
        ))
        .unwrap();
        assert_eq!(actual.transitions, expected);
    }

    #[test]
    fn range_outputs_reject_invalid_later_claim_after_noop() {
        let (mut inputs, mut oracle, configs) = progressing_range_fixture();
        let last_claim = inputs.claimed_transitions.last_mut().unwrap();
        let mut preimage = block_on(
            oracle.get(PreimageKey::new_keccak256(*last_claim.optimistic_block.output_root)),
        )
        .unwrap();
        // Preserve the block hash and V0 structure: only execution can reject this state root.
        preimage[32] ^= 1;
        let invalid_root = alloy_primitives::keccak256(&preimage);
        oracle.save_preimage(PreimageKey::new_keccak256(*invalid_root), preimage).unwrap();
        last_claim.optimistic_block.output_root = invalid_root;
        let err = block_on(build_range_outputs(
            inputs,
            Arc::new(oracle),
            kona_sp1_client_utils::BlobStore::default(),
            Some(&configs),
        ))
        .unwrap_err();
        assert!(
            err.to_string().contains("Failed to validate L2 block #2"),
            "unexpected error: {err}"
        );
    }

    fn rollup_configs(chain_ids: &[u64]) -> BTreeMap<u64, RollupConfig> {
        chain_ids.iter().map(|chain_id| (*chain_id, rollup_config(*chain_id, 1))).collect()
    }

    fn range_inputs(
        previous_output_root: B256,
        transition: SuperRangeTransition,
    ) -> SuperRangeInputs {
        SuperRangeInputs {
            span: TimestampSpan::new(101, 101).unwrap(),
            l1_head: b256(0x11),
            chain_ids: vec![U256::from(10)],
            previous_super_root_proofs: vec![SuperRootProof::new(
                100,
                vec![SuperOutputRoot { chain_id: 10, output_root: previous_output_root }],
            )],
            claimed_transitions: vec![transition],
        }
    }

    fn range_inputs_for_chain_ids(chain_ids: &[u64]) -> SuperRangeInputs {
        SuperRangeInputs {
            span: TimestampSpan::new(101, 101).unwrap(),
            l1_head: b256(0x11),
            chain_ids: chain_ids.iter().copied().map(U256::from).collect(),
            previous_super_root_proofs: vec![SuperRootProof::new(
                100,
                chain_ids
                    .iter()
                    .copied()
                    .map(|chain_id| SuperOutputRoot { chain_id, output_root: b256(0x44) })
                    .collect(),
            )],
            claimed_transitions: chain_ids
                .iter()
                .map(|chain_id| SuperRangeTransition {
                    timestamp: 101,
                    optimistic_block: SuperOptimisticBlock {
                        chain_id: U256::from(*chain_id),
                        block_hash: b256(0x22),
                        output_root: b256(0x44),
                    },
                })
                .collect(),
        }
    }

    fn contiguous_range_inputs(chain_roots: &[&[u8]]) -> SuperRangeInputs {
        let steps = chain_roots[0].len() - 1;
        let chain_ids = (1..=chain_roots.len()).map(|i| U256::from(i * 10)).collect();
        let mut previous_super_root_proofs = Vec::new();
        let mut claimed_transitions = Vec::new();
        for step in 0..steps {
            let mut output_roots = Vec::new();
            for (chain, roots) in chain_roots.iter().enumerate() {
                let chain_id = (chain as u64 + 1) * 10;
                output_roots.push(SuperOutputRoot { chain_id, output_root: b256(roots[step]) });
                claimed_transitions.push(SuperRangeTransition {
                    timestamp: 101 + step as u64,
                    optimistic_block: SuperOptimisticBlock {
                        chain_id: U256::from(chain_id),
                        block_hash: b256(roots[step + 1]),
                        output_root: b256(roots[step + 1]),
                    },
                });
            }
            previous_super_root_proofs.push(SuperRootProof::new(100 + step as u64, output_roots));
        }
        SuperRangeInputs {
            span: TimestampSpan::new(101, 100 + steps as u64).unwrap(),
            l1_head: b256(0x11),
            chain_ids,
            previous_super_root_proofs,
            claimed_transitions,
        }
    }

    #[test]
    fn contiguous_transitions_share_one_pipeline_segment() {
        let inputs = contiguous_range_inputs(&[&[1, 2, 3, 4, 5]]);
        inputs.validate().unwrap();

        let segments = range_segments(&inputs).unwrap();

        assert_eq!(segments.len(), 1, "contiguous claims must construct only one pipeline");
        assert_eq!(segments[0], inputs.claimed_transitions.iter().collect::<Vec<_>>());
    }

    #[test]
    fn changed_agreed_root_starts_a_new_pipeline_segment() {
        let mut inputs = contiguous_range_inputs(&[&[1, 2, 3, 4, 5]]);
        inputs.previous_super_root_proofs[2].super_root.output_roots[0].output_root = b256(9);

        let segments = range_segments(&inputs).unwrap();

        assert_eq!(segments.len(), 2);
        assert_eq!(segments[0], inputs.claimed_transitions[..2].iter().collect::<Vec<_>>());
        assert_eq!(segments[1], inputs.claimed_transitions[2..].iter().collect::<Vec<_>>());
    }

    #[test]
    fn noop_transitions_preserve_pipeline_segments() {
        let inputs = contiguous_range_inputs(&[&[1, 1, 2, 2, 3, 3]]);

        let segments = range_segments(&inputs).unwrap();

        assert_eq!(segments.len(), 1);
        assert_eq!(segments[0].len(), 5);
    }

    #[test]
    fn discontinuity_at_noop_starts_a_new_pipeline_segment() {
        let mut inputs = contiguous_range_inputs(&[&[1, 2, 9, 10]]);
        inputs.previous_super_root_proofs[1].super_root.output_roots[0].output_root = b256(9);

        let segments = range_segments(&inputs).unwrap();

        assert_eq!(segments.len(), 2);
        assert_eq!(segments[0], vec![&inputs.claimed_transitions[0]]);
        assert_eq!(segments[1], inputs.claimed_transitions[1..].iter().collect::<Vec<_>>());
    }

    #[test]
    fn pipeline_segments_are_chain_local() {
        let inputs = contiguous_range_inputs(&[&[1, 2, 3], &[1, 2, 3]]);
        inputs.validate().unwrap();

        let segments = range_segments(&inputs).unwrap();

        assert_eq!(segments.len(), 2);
        assert_eq!(
            segments[0],
            vec![&inputs.claimed_transitions[0], &inputs.claimed_transitions[2]]
        );
        assert_eq!(
            segments[1],
            vec![&inputs.claimed_transitions[1], &inputs.claimed_transitions[3]]
        );
    }

    #[test]
    fn range_outputs_keep_timestamp_major_order_without_advancing_noops() {
        let chain_ids = [u64::MAX - 1, u64::MAX];
        let mut oracle = PreimageStore::default();
        let mut configs = rollup_configs(&chain_ids);
        for config in configs.values_mut() {
            config.block_time = 10;
        }
        let configs = ChainConfigs {
            dependency_set: dependency_set(&chain_ids, None),
            rollup_configs: configs,
            l1_config: L1_CONFIGS[&1].clone(),
        };
        let mut output_roots = Vec::new();
        let mut optimistic_blocks = Vec::new();
        for (i, chain_id) in chain_ids.into_iter().enumerate() {
            let header = Header { number: 3 + i as u64, timestamp: 100, ..Default::default() };
            let block_hash = save_header(&mut oracle, &header);
            let output_root = save_output_root(&mut oracle, block_hash);
            output_roots.push(SuperOutputRoot { chain_id, output_root });
            optimistic_blocks.push(SuperOptimisticBlock {
                chain_id: U256::from(chain_id),
                block_hash,
                output_root,
            });
        }
        let inputs = SuperRangeInputs {
            span: TimestampSpan::new(101, 103).unwrap(),
            l1_head: b256(0x11),
            chain_ids: chain_ids.into_iter().map(U256::from).collect(),
            previous_super_root_proofs: (100..103)
                .map(|timestamp| SuperRootProof::new(timestamp, output_roots.clone()))
                .collect(),
            claimed_transitions: (101..=103)
                .flat_map(|timestamp| {
                    optimistic_blocks.iter().copied().map(move |optimistic_block| {
                        SuperRangeTransition { timestamp, optimistic_block }
                    })
                })
                .collect(),
        };
        let expected = SuperRangeOutputs {
            span: inputs.span,
            l1_head: inputs.l1_head,
            previous_super_roots: inputs
                .previous_super_root_proofs
                .iter()
                .map(hash_super_root_proof)
                .collect::<Result<Vec<_>, _>>()
                .unwrap(),
            transitions: inputs.claimed_transitions.clone(),
        };

        // No L1 witness is supplied, so creating a pipeline for a no-op would fail.
        let actual = block_on(build_range_outputs(
            inputs,
            Arc::new(oracle),
            kona_sp1_client_utils::BlobStore::default(),
            Some(&configs),
        ))
        .unwrap();

        assert_eq!(actual, expected);
    }

    #[test]
    fn range_outputs_require_exact_config_chain_coverage() {
        let chain_ids = [u64::MAX - 1, u64::MAX];
        let configs = chain_configs(&[chain_ids[0]]);
        let err = block_on(build_range_outputs(
            range_inputs_for_chain_ids(&chain_ids),
            Arc::new(PreimageStore::default()),
            kona_sp1_client_utils::BlobStore::default(),
            Some(&configs),
        ))
        .unwrap_err();
        assert!(err.to_string().contains("must exactly match"), "unexpected error: {err}");
    }

    #[test]
    fn range_boot_is_derived_from_public_inputs_and_shared_witness() {
        let safe_head = Header { number: 3, timestamp: 100, ..Default::default() };
        let claimed_head = Header { number: 4, timestamp: 101, ..Default::default() };
        let mut oracle = PreimageStore::default();
        let safe_head_hash = save_header(&mut oracle, &safe_head);
        let claimed_head_hash = save_header(&mut oracle, &claimed_head);
        let agreed_root = save_output_root(&mut oracle, safe_head_hash);
        let claimed_root = save_output_root(&mut oracle, claimed_head_hash);

        let transition = SuperRangeTransition {
            timestamp: 101,
            optimistic_block: SuperOptimisticBlock {
                chain_id: U256::from(10),
                block_hash: claimed_head_hash,
                output_root: claimed_root,
            },
        };
        let inputs = range_inputs(agreed_root, transition);

        let transition_boot = block_on(build_super_range_transition_boot(
            &inputs,
            &transition,
            &oracle,
            &rollup_configs(&[10]),
            &Default::default(),
        ))
        .unwrap();

        let RangeTransitionBoot::Progress {
            boot,
            safe_head_hash: actual_safe_head_hash,
            safe_head: actual_safe_head,
        } = transition_boot
        else {
            panic!("expected progressing transition boot");
        };
        assert_eq!(boot.l1_head, inputs.l1_head);
        assert_eq!(boot.chain_id, 10);
        assert_eq!(boot.agreed_l2_output_root, agreed_root);
        assert_eq!(boot.claimed_l2_output_root, claimed_root);
        assert_eq!(boot.claimed_l2_block_number, 4);
        assert_eq!(actual_safe_head_hash, safe_head_hash);
        assert_eq!(actual_safe_head, safe_head);
    }

    #[test]
    fn range_boot_accepts_noop_before_next_l2_timestamp() {
        let safe_head = Header { number: 3, timestamp: 100, ..Default::default() };
        let mut oracle = PreimageStore::default();
        let safe_head_hash = save_header(&mut oracle, &safe_head);
        let agreed_root = save_output_root(&mut oracle, safe_head_hash);

        let transition = SuperRangeTransition {
            timestamp: 101,
            optimistic_block: SuperOptimisticBlock {
                chain_id: U256::from(10),
                block_hash: safe_head_hash,
                output_root: agreed_root,
            },
        };
        let inputs = range_inputs(agreed_root, transition);
        let mut configs = rollup_configs(&[10]);
        configs.get_mut(&10).unwrap().block_time = 2;

        let transition_boot = block_on(build_super_range_transition_boot(
            &inputs,
            &transition,
            &oracle,
            &configs,
            &Default::default(),
        ))
        .unwrap();

        let RangeTransitionBoot::NoOp { boot } = transition_boot else {
            panic!("expected no-op transition boot");
        };
        assert_eq!(boot.claimed_l2_block_number, safe_head.number);
        assert_eq!(boot.claimed_l2_output_root, agreed_root);
    }

    #[test]
    fn range_boot_rejects_arbitrary_progress_target_block() {
        let safe_head = Header { number: 3, timestamp: 100, ..Default::default() };
        let skipped_head = Header { number: 9, timestamp: 101, ..Default::default() };
        let mut oracle = PreimageStore::default();
        let safe_head_hash = save_header(&mut oracle, &safe_head);
        let skipped_head_hash = save_header(&mut oracle, &skipped_head);
        let agreed_root = save_output_root(&mut oracle, safe_head_hash);
        let skipped_root = save_output_root(&mut oracle, skipped_head_hash);

        let transition = SuperRangeTransition {
            timestamp: 101,
            optimistic_block: SuperOptimisticBlock {
                chain_id: U256::from(10),
                block_hash: skipped_head_hash,
                output_root: skipped_root,
            },
        };
        let inputs = range_inputs(agreed_root, transition);

        let err = block_on(build_super_range_transition_boot(
            &inputs,
            &transition,
            &oracle,
            &rollup_configs(&[10]),
            &Default::default(),
        ))
        .unwrap_err();

        assert!(
            err.to_string().contains("must target next L2 block #4"),
            "unexpected error: {err}"
        );
    }

    struct OverlappingSpanBatch {
        witness: PreimageStore,
        config: RollupConfig,
        l1_origin: BlockInfo,
        safe_head: L2BlockInfo,
        batch: SpanBatch,
        parent_hash: B256,
    }

    /// Safe head #1 (ts 101) on top of genesis #0 (ts 100). The span batch covers ts 101-102, so
    /// it overlaps the safe head and its parent is block #0.
    fn overlapping_span_batch(withhold_parent_header: bool) -> OverlappingSpanBatch {
        let l1_origin = BlockInfo::new(b256(0x11), 1, B256::ZERO, 100);
        let mut witness = PreimageStore::default();
        witness.save_preimage(PreimageKey::new_keccak256(*EMPTY_ROOT_HASH), vec![0x80]).unwrap();
        let parent = Header {
            number: 0,
            timestamp: 100,
            transactions_root: EMPTY_ROOT_HASH,
            ..Default::default()
        };
        let parent_hash = if withhold_parent_header {
            parent.hash_slow()
        } else {
            save_header(&mut witness, &parent)
        };
        let safe_head_hash = save_header(
            &mut witness,
            &Header {
                number: 1,
                timestamp: 101,
                parent_hash,
                transactions_root: EMPTY_ROOT_HASH,
                ..Default::default()
            },
        );

        let mut config = rollup_config(u64::MAX, 1);
        config.hardforks.delta_time = Some(0);
        config.hardforks.holocene_time = Some(0);
        config.genesis.l1 = BlockNumHash { number: 1, hash: l1_origin.hash };
        config.genesis.l2 = BlockNumHash { number: 0, hash: parent_hash };
        config.genesis.l2_time = 100;

        OverlappingSpanBatch {
            witness,
            config,
            l1_origin,
            safe_head: L2BlockInfo {
                block_info: BlockInfo::new(safe_head_hash, 1, parent_hash, 101),
                l1_origin: BlockNumHash { number: 1, hash: l1_origin.hash },
                seq_num: 1,
            },
            batch: SpanBatch {
                parent_check: FixedBytes::from_slice(&parent_hash[..20]),
                l1_origin_check: FixedBytes::from_slice(&l1_origin.hash[..20]),
                batches: vec![
                    SpanBatchElement { epoch_num: 1, timestamp: 101, transactions: vec![] },
                    SpanBatchElement { epoch_num: 1, timestamp: 102, transactions: vec![] },
                ],
                ..Default::default()
            },
            parent_hash,
        }
    }

    fn check_prefix_through_guest_oracle(
        fixture: OverlappingSpanBatch,
    ) -> (BatchValidity, Option<L2BlockInfo>) {
        block_on(async {
            let (oracle, _) = DefaultWitnessData::from_parts(fixture.witness, BlobData::default())
                .get_oracle_and_blob_provider()
                .await
                .unwrap();
            let mut provider = OracleL2ChainProvider::new(
                fixture.safe_head.block_info.hash,
                Arc::new(fixture.config.clone()),
                oracle,
            );
            fixture
                .batch
                .check_batch_prefix(
                    &fixture.config,
                    &[fixture.l1_origin],
                    fixture.safe_head,
                    &fixture.l1_origin,
                    &mut provider,
                )
                .await
        })
    }

    #[test]
    fn span_batch_prefix_accepts_overlapping_batch_with_complete_witness() {
        let fixture = overlapping_span_batch(false);
        let parent_hash = fixture.parent_hash;

        let (validity, parent) = check_prefix_through_guest_oracle(fixture);

        assert_eq!(validity, BatchValidity::Accept);
        assert_eq!(parent.map(|parent| parent.block_info.hash), Some(parent_hash));
    }

    /// Regression test for #23206: a prover that withholds a span batch's parent header must abort
    /// the guest. Otherwise `check_batch_prefix` returns `Undecided` and the Holocene
    /// `BatchStream` skips the valid batch.
    #[test]
    #[should_panic(expected = "requested preimage key not present in witness")]
    fn span_batch_prefix_aborts_when_parent_header_missing_from_witness() {
        let _ = check_prefix_through_guest_oracle(overlapping_span_batch(true));
    }
}
