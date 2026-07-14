//! Shared execution logic for the SP1 super-range guest's range mode.

use std::{collections::BTreeMap, fmt::Debug, sync::Arc};

use alloy_consensus::{Header, Sealed};
use alloy_primitives::{B256, U256};
use alloy_rlp::Decodable;
use anyhow::{anyhow, bail, ensure};
use kona_derive::BlobProvider;
use kona_genesis::{L1ChainConfig, RollupConfig};
use kona_interop::DependencySet;
use kona_preimage::{
    CommsClient, DEPENDENCY_SET_KEY, L1_CONFIG_KEY, L2_ROLLUP_CONFIG_KEY, PreimageKey,
    PreimageOracleClient,
};
use kona_proof::{
    BootInfo, FlushableCache, l1::OracleL1ChainProvider, l2::OracleL2ChainProvider,
    sync::new_oracle_pipeline_cursor,
};
use kona_proof_interop::HintType;
use kona_registry::{DEPENDENCY_SETS, L1_CONFIGS, ROLLUP_CONFIGS};
use kona_sp1_client_utils::{
    boot::BootInfoStruct,
    super_root::{
        SuperOptimisticBlock, SuperRangeInputs, SuperRangeOutputs, SuperRangeTransition,
        hash_super_root_proof,
    },
    witness::executor::WitnessExecutor,
};

use crate::executor::ETHDAWitnessExecutor;

/// Builds range-mode public outputs from typed inputs and an oracle-backed witness source.
pub async fn build_range_outputs<O, B>(
    inputs: SuperRangeInputs,
    oracle: Arc<O>,
    beacon: B,
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

    let dependency_set = Arc::new(load_dependency_set(&inputs.chain_ids, oracle.as_ref()).await?);
    let rollup_configs = load_rollup_configs(&inputs.chain_ids, oracle.as_ref()).await?;
    let l1_config = load_l1_config(&rollup_configs, oracle.as_ref()).await?;

    let mut transitions = Vec::with_capacity(inputs.claimed_transitions.len());
    for transition in &inputs.claimed_transitions {
        let boot_info = run_super_range_transition(
            &inputs,
            transition,
            oracle.clone(),
            beacon.clone(),
            dependency_set.clone(),
            &rollup_configs,
            &l1_config,
        )
        .await?;
        validate_range_transition_output(transition, oracle.as_ref(), &boot_info).await?;
        transitions.push(*transition);
    }

    Ok(SuperRangeOutputs {
        span: inputs.span,
        l1_head: inputs.l1_head,
        previous_super_roots,
        transitions,
    })
}

/// Loads the dependency set for the supplied range chain IDs.
pub async fn load_dependency_set<O>(
    input_chain_ids: &[U256],
    oracle: &O,
) -> anyhow::Result<DependencySet>
where
    O: PreimageOracleClient,
{
    let chain_ids = input_chain_ids_as_u64(input_chain_ids)?;
    let dependency_set = if let Some(dependency_set) =
        chain_ids.first().and_then(|chain_id| DEPENDENCY_SETS.get(chain_id))
    {
        dependency_set.clone()
    } else {
        eprintln!(
            "No embedded dependency set found for proof chain ids {:?}, falling back to preimage oracle. This is insecure in production without additional validation!",
            chain_ids
        );
        let serialized = oracle
            .get(PreimageKey::new_local(DEPENDENCY_SET_KEY.to()))
            .await
            .map_err(|err| anyhow!("failed to fetch dependency set fallback: {err}"))?;
        serde_json::from_slice(&serialized)
            .map_err(|err| anyhow!("failed to decode dependency set fallback: {err}"))?
    };

    ensure_dependency_set_matches_inputs(input_chain_ids, &dependency_set)?;
    Ok(dependency_set)
}

/// Loads rollup configs for the supplied range chain IDs.
pub async fn load_rollup_configs<O>(
    input_chain_ids: &[U256],
    oracle: &O,
) -> anyhow::Result<BTreeMap<u64, RollupConfig>>
where
    O: PreimageOracleClient,
{
    let chain_ids = input_chain_ids_as_u64(input_chain_ids)?;
    let rollup_configs = if chain_ids.iter().all(|chain_id| ROLLUP_CONFIGS.contains_key(chain_id)) {
        chain_ids
            .iter()
            .map(|chain_id| {
                (*chain_id, ROLLUP_CONFIGS.get(chain_id).expect("checked above").clone())
            })
            .collect()
    } else {
        let serialized = oracle
            .get(PreimageKey::new_local(L2_ROLLUP_CONFIG_KEY.to()))
            .await
            .map_err(|err| anyhow!("failed to fetch rollup config fallback: {err}"))?;
        decode_rollup_config_fallback(&serialized, &chain_ids)?
    };

    ensure_rollup_configs_match_inputs(input_chain_ids, &rollup_configs)?;
    Ok(rollup_configs)
}

fn decode_rollup_config_fallback(
    serialized: &[u8],
    chain_ids: &[u64],
) -> anyhow::Result<BTreeMap<u64, RollupConfig>> {
    match serde_json::from_slice::<BTreeMap<u64, RollupConfig>>(serialized) {
        Ok(configs) => Ok(configs),
        Err(map_err) => {
            ensure!(
                chain_ids.len() == 1,
                "failed to decode rollup config fallback as chain-id map: {map_err}",
            );
            let config = serde_json::from_slice::<RollupConfig>(serialized).map_err(|err| {
                anyhow!(
                    "failed to decode rollup config fallback as chain-id map ({map_err}) or single config ({err})",
                )
            })?;
            let mut configs = BTreeMap::new();
            configs.insert(chain_ids[0], config);
            Ok(configs)
        }
    }
}

fn ensure_rollup_configs_match_inputs(
    input_chain_ids: &[U256],
    rollup_configs: &BTreeMap<u64, RollupConfig>,
) -> anyhow::Result<()> {
    let config_chain_ids = rollup_configs.keys().copied().map(U256::from).collect::<Vec<_>>();
    ensure!(
        input_chain_ids == config_chain_ids,
        "super-range chain IDs {input_chain_ids:?} must exactly match rollup config chain IDs {config_chain_ids:?}",
    );

    for (chain_id, config) in rollup_configs {
        ensure!(
            config.l2_chain_id.id() == *chain_id,
            "rollup config key {chain_id} does not match config L2 chain ID {actual}",
            actual = config.l2_chain_id.id(),
        );
    }

    Ok(())
}

/// Loads the L1 chain config shared by the supplied rollup configs.
pub async fn load_l1_config<O>(
    rollup_configs: &BTreeMap<u64, RollupConfig>,
    oracle: &O,
) -> anyhow::Result<L1ChainConfig>
where
    O: PreimageOracleClient,
{
    let first_l1_chain_id = rollup_configs
        .values()
        .next()
        .map(|config| config.l1_chain_id)
        .ok_or_else(|| anyhow!("super-range rollup config set is empty"))?;

    for config in rollup_configs.values() {
        ensure!(
            config.l1_chain_id == first_l1_chain_id,
            "super-range rollup configs must share one L1 chain ID, got {first} and {actual}",
            first = first_l1_chain_id,
            actual = config.l1_chain_id,
        );
    }

    if let Some(config) = L1_CONFIGS.get(&first_l1_chain_id) {
        return Ok(config.clone());
    }

    let serialized = oracle
        .get(PreimageKey::new_local(L1_CONFIG_KEY.to()))
        .await
        .map_err(|err| anyhow!("failed to fetch L1 config fallback: {err}"))?;
    serde_json::from_slice(&serialized)
        .map_err(|err| anyhow!("failed to decode L1 config fallback: {err}"))
}

fn input_chain_ids_as_u64(input_chain_ids: &[U256]) -> anyhow::Result<Vec<u64>> {
    input_chain_ids
        .iter()
        .map(|chain_id| {
            if *chain_id > U256::from(u64::MAX) {
                bail!("super-range chain ID {chain_id} does not fit in dependency set keys");
            }
            Ok(chain_id.saturating_to::<u64>())
        })
        .collect()
}

fn ensure_dependency_set_matches_inputs(
    input_chain_ids: &[U256],
    dependency_set: &DependencySet,
) -> anyhow::Result<()> {
    let dependency_set_chain_ids =
        dependency_set.dependencies.keys().copied().map(U256::from).collect::<Vec<_>>();
    ensure!(
        input_chain_ids == dependency_set_chain_ids,
        "super-range chain IDs {input_chain_ids:?} must exactly match dependency set chain IDs {dependency_set_chain_ids:?}",
    );

    Ok(())
}

#[derive(Debug)]
#[allow(clippy::large_enum_variant)]
enum RangeTransitionBoot {
    NoOp { boot: BootInfo },
    Progress { boot: BootInfo, safe_head_hash: B256, safe_head: Header },
}

async fn run_super_range_transition<O, B>(
    inputs: &SuperRangeInputs,
    transition: &SuperRangeTransition,
    oracle: Arc<O>,
    beacon: B,
    dependency_set: Arc<DependencySet>,
    rollup_configs: &BTreeMap<u64, RollupConfig>,
    l1_config: &L1ChainConfig,
) -> anyhow::Result<BootInfoStruct>
where
    O: CommsClient + FlushableCache + Send + Sync + Debug + 'static,
    B: BlobProvider + Send + Sync + Debug + Clone + 'static,
{
    let transition_boot = build_super_range_transition_boot(
        inputs,
        transition,
        oracle.as_ref(),
        rollup_configs,
        l1_config,
    )
    .await?;

    let (boot, safe_head_hash, safe_head) = match transition_boot {
        RangeTransitionBoot::NoOp { boot } => return Ok(BootInfoStruct::from(boot)),
        RangeTransitionBoot::Progress { boot, safe_head_hash, safe_head } => {
            (boot, safe_head_hash, safe_head)
        }
    };

    let rollup_config = Arc::new(boot.rollup_config.clone());
    let l1_config = Arc::new(boot.l1_config.clone());
    let mut l1_provider = OracleL1ChainProvider::new(boot.l1_head, oracle.clone());
    let mut l2_provider =
        OracleL2ChainProvider::new(safe_head_hash, rollup_config.clone(), oracle.clone());
    l2_provider.set_chain_id(Some(boot.chain_id));

    let cursor = new_oracle_pipeline_cursor(
        rollup_config.as_ref(),
        Sealed::new_unchecked(safe_head, safe_head_hash),
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
    let boot = executor.run(boot, pipeline, cursor, l2_provider).await?;

    Ok(BootInfoStruct::from(boot))
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
    let safe_head = fetch_l2_header(oracle, chain_id, safe_head_hash).await?;
    let output_claimed_block_hash = fetch_output_block_hash(oracle, claimed_output_root).await?;
    ensure!(
        output_claimed_block_hash == claimed_block_hash,
        "claimed optimistic output root commits to block hash {actual}, expected {expected}",
        actual = output_claimed_block_hash,
        expected = claimed_block_hash,
    );
    let claimed_header = fetch_l2_header(oracle, chain_id, claimed_block_hash).await?;

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

    let chain_id = transition_chain_id_as_u64(transition)?;
    let block_hash = fetch_output_block_hash(oracle, committed_boot_info.l2PostRoot).await?;
    ensure!(
        block_hash == transition.optimistic_block.block_hash,
        "output root commits to block hash {actual}, expected {expected}",
        actual = block_hash,
        expected = transition.optimistic_block.block_hash,
    );

    let header = fetch_l2_header(oracle, chain_id, block_hash).await?;
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

/// Fetches an output root preimage and returns the L2 block hash it commits to.
pub async fn fetch_output_block_hash<O>(oracle: &O, output_root: B256) -> anyhow::Result<B256>
where
    O: PreimageOracleClient,
{
    let output_preimage = oracle
        .get(PreimageKey::new_keccak256(*output_root))
        .await
        .map_err(|err| anyhow!("failed to fetch output-root preimage {output_root}: {err}"))?;
    ensure!(
        output_preimage.len() == 128,
        "output-root preimage {output_root} has length {}, expected 128",
        output_preimage.len(),
    );
    if output_preimage[..32].iter().any(|byte| *byte != 0) {
        bail!(
            "output-root preimage {output_root} has unsupported version {}",
            B256::from_slice(&output_preimage[..32]),
        );
    }

    Ok(B256::from_slice(&output_preimage[96..128]))
}

/// Fetches and decodes an L2 header through the interop hint path.
pub async fn fetch_l2_header<O>(
    oracle: &O,
    chain_id: u64,
    block_hash: B256,
) -> anyhow::Result<Header>
where
    O: CommsClient,
{
    HintType::L2BlockHeader
        .with_data(&[block_hash.as_slice(), chain_id.to_be_bytes().as_ref()])
        .send(oracle)
        .await
        .map_err(|err| anyhow!("failed to hint L2 header {block_hash}: {err}"))?;

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
    use alloy_primitives::{B256, U256, keccak256};
    use alloy_rlp::Encodable;
    use kona_genesis::RollupConfig;
    use kona_interop::{ChainDependency, DependencySet};
    use kona_preimage::PreimageKey;
    use kona_proof::block_on;
    use kona_sp1_client_utils::{
        super_root::{
            SuperOptimisticBlock, SuperOutputRoot, SuperRangeInputs, SuperRangeTransition,
            SuperRootProof, TimestampSpan,
        },
        witness::preimage_store::PreimageStore,
    };

    use super::*;

    fn b256(fill: u8) -> B256 {
        B256::from([fill; 32])
    }

    fn save_header(oracle: &mut PreimageStore, header: &Header) -> B256 {
        let hash = header.hash_slow();
        let mut rlp = Vec::new();
        header.encode(&mut rlp);
        oracle.save_preimage(PreimageKey::new_keccak256(*hash), rlp).unwrap();
        hash
    }

    fn save_output_root(oracle: &mut PreimageStore, block_hash: B256) -> B256 {
        let mut output_preimage = [0u8; 128];
        output_preimage[96..128].copy_from_slice(block_hash.as_slice());
        let output_root = B256::from(keccak256(output_preimage));
        oracle
            .save_preimage(PreimageKey::new_keccak256(*output_root), output_preimage.to_vec())
            .unwrap();
        output_root
    }

    #[allow(clippy::zero_sized_map_values)]
    fn dependency_set(
        chain_ids: &[u64],
        override_message_expiry_window: Option<u64>,
    ) -> DependencySet {
        let dependencies =
            chain_ids.iter().map(|chain_id| (*chain_id, ChainDependency {})).collect();
        DependencySet { dependencies, override_message_expiry_window }
    }

    fn rollup_config(chain_id: u64, l1_chain_id: u64) -> RollupConfig {
        RollupConfig {
            block_time: 1,
            l1_chain_id,
            l2_chain_id: alloy_chains::Chain::from(chain_id),
            ..Default::default()
        }
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
            claimed_transitions: vec![],
        }
    }

    #[test]
    fn dependency_set_validation_requires_exact_range_chain_coverage() {
        let dependency_set = dependency_set(&[10, 20], Some(123));

        ensure_dependency_set_matches_inputs(&[U256::from(10), U256::from(20)], &dependency_set)
            .expect("matching depset chains are valid");

        let err = ensure_dependency_set_matches_inputs(&[U256::from(10)], &dependency_set)
            .expect_err("partial depset coverage must fail");
        assert!(err.to_string().contains("must exactly match"), "unexpected error: {err}");

        let err = ensure_dependency_set_matches_inputs(
            &[U256::from(10), U256::from(30)],
            &dependency_set,
        )
        .expect_err("wrong depset chain must fail");
        assert!(err.to_string().contains("must exactly match"), "unexpected error: {err}");
    }

    #[test]
    fn dependency_set_fallback_preserves_override_expiry_window() {
        let chain_id = u64::MAX;
        let dependency_set = dependency_set(&[chain_id], Some(123));
        let mut oracle = PreimageStore::default();
        oracle
            .save_preimage(
                PreimageKey::new_local(DEPENDENCY_SET_KEY.to()),
                serde_json::to_vec(&dependency_set).unwrap(),
            )
            .unwrap();

        let inputs = range_inputs_for_chain_ids(&[chain_id]);
        let loaded = block_on(load_dependency_set(&inputs.chain_ids, &oracle)).unwrap();

        assert_eq!(loaded, dependency_set);
        assert_eq!(loaded.override_message_expiry_window, Some(123));
    }

    #[test]
    fn rollup_config_fallback_requires_exact_range_chain_coverage() {
        let mut oracle = PreimageStore::default();
        let mut fallback = BTreeMap::new();
        fallback.insert(u64::MAX - 1, rollup_config(u64::MAX - 1, 1));
        oracle
            .save_preimage(
                PreimageKey::new_local(L2_ROLLUP_CONFIG_KEY.to()),
                serde_json::to_vec(&fallback).unwrap(),
            )
            .unwrap();
        let inputs = range_inputs_for_chain_ids(&[u64::MAX - 1, u64::MAX]);

        let err = block_on(load_rollup_configs(&inputs.chain_ids, &oracle)).unwrap_err();

        assert!(err.to_string().contains("must exactly match"), "unexpected error: {err}");
    }

    #[test]
    fn l1_config_requires_one_l1_chain_across_range() {
        let configs = [
            (u64::MAX - 1, rollup_config(u64::MAX - 1, 1)),
            (u64::MAX, rollup_config(u64::MAX, 2)),
        ]
        .into_iter()
        .collect();
        let oracle = PreimageStore::default();

        let err = block_on(load_l1_config(&configs, &oracle)).unwrap_err();

        assert!(err.to_string().contains("must share one L1 chain ID"), "unexpected error: {err}");
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
}
