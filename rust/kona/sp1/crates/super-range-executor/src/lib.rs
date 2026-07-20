//! Super-range SP1 executor support.
//!
//! The executor queries `superroot_atTimestamp` for the typed super-range inputs, collects a
//! witness by running the shared native range logic against the interop preimage server, and then
//! executes the real `super-range` SP1 ELF over that witness.

use std::{
    collections::BTreeMap,
    fmt::{self, Debug},
    path::PathBuf,
    str::FromStr,
    sync::{Arc, Mutex},
};

use alloy_primitives::{B256, Bytes, U256, keccak256};
use anyhow::{Context, Result, anyhow, bail, ensure};
use async_trait::async_trait;
use jsonrpsee_core::{client::ClientT, rpc_params};
use jsonrpsee_http_client::{HttpClient, HttpClientBuilder};
use kona_host::{DataFormat, interop::InteropHost};
use kona_preimage::{
    BidirectionalChannel, HintWriter, HintWriterClient, NativeChannel, OracleReader, PreimageKey,
    PreimageOracleClient, errors::PreimageOracleResult,
};
use kona_proof::{CachingOracle, FlushableCache, l1::OracleBlobProvider};
use kona_sp1_client_utils::{
    super_root::{
        SuperConsolidationInputs, SuperConsolidationOutputs, SuperConsolidationTransitionInput,
        SuperInteropInputs, SuperInteropOutputs, SuperOptimisticBlock, SuperOutputRoot,
        SuperRangeInputs, SuperRangeOutputs, SuperRangeTransition, SuperRootProof, TimestampSpan,
        encode_super_root_proof, hash_super_root_proof,
    },
    witness::{BlobData, DefaultWitnessData, WitnessData, preimage_store::PreimageStore},
};
use kona_sp1_ethereum_client_utils::{
    super_consolidation::build_consolidation_outputs, super_range::build_range_outputs,
};
use kona_sp1_host_utils::witness_generation::{OnlineBlobStore, PreimageWitnessCollector};
use serde::Deserialize;
use sp1_sdk::{Elf, Prover, ProverClient, SP1PublicValues, SP1Stdin};

/// Exit code signalling a valid super-range claim.
pub const EXIT_VALID: u8 = 0;
/// Exit code signalling an invalid super-range claim.
pub const EXIT_INVALID: u8 = 1;
/// Exit code signalling an infrastructure failure before a verdict was reached.
pub const EXIT_INFRA: u8 = 2;

/// CLI/runtime configuration for one super-range execution.
#[derive(Clone, Debug)]
pub struct RunConfig {
    /// Replay collected witnesses through the shared native cores instead of executing the guest.
    pub native_core: bool,
    /// Supernode JSON-RPC endpoint.
    pub supernode_address: String,
    /// L1 execution JSON-RPC endpoint.
    pub l1_node_address: String,
    /// L1 beacon API endpoint.
    pub l1_beacon_address: String,
    /// L2 execution JSON-RPC endpoints.
    pub l2_node_addresses: Vec<String>,
    /// Trusted L1 head hash to derive against.
    pub l1_head: B256,
    /// Superchain timestamp to prove.
    pub end_timestamp: u64,
    /// Optional rollup config JSON files passed through to `InteropHost`.
    pub rollup_config_paths: Option<Vec<PathBuf>>,
    /// Optional L1 config JSON file passed through to `InteropHost`.
    pub l1_config_path: Option<PathBuf>,
    /// Optional dependency-set JSON file passed through to `InteropHost`.
    pub dependency_set_path: Option<PathBuf>,
}

/// Verdict from executing the SP1 guest.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum Verdict {
    /// The guest accepted the synthesized super-range inputs and witness.
    Valid,
    /// The guest rejected the synthesized super-range inputs or committed unexpected output.
    Invalid,
}

/// A full synthesized super-range execution bundle.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct SynthesizedExecution {
    /// The timestamp immediately before `inputs.span.start`.
    pub previous_timestamp: u64,
    /// Typed inputs for the SP1 guest's range mode.
    pub range_inputs: SuperRangeInputs,
    /// Typed inputs for the SP1 guest's consolidation mode.
    pub consolidation_inputs: SuperConsolidationInputs,
    /// Preimages that must be available before host hinting can discover the rest of the witness.
    pub preloaded_preimages: Vec<(PreimageKey, Vec<u8>)>,
    /// The current response's super-root hash, used as the host post-state claim.
    pub current_super_root: B256,
    /// The previous super-root proof, encoded as the host agreed pre-state.
    pub previous_super_root_proof_bytes: Vec<u8>,
}

/// Runs the full two-pass super-range executor.
pub async fn run(config: RunConfig) -> Result<Verdict> {
    let super_range_elf = if config.native_core {
        None
    } else {
        let bytes =
            kona_sp1_elfs::load_elf("super-range-elf").context("super-range ELF unavailable")?;
        Some(Arc::<[u8]>::from(bytes))
    };

    let client = HttpClientBuilder::default()
        .build(&config.supernode_address)
        .with_context(|| format!("failed to connect to supernode {}", config.supernode_address))?;
    let previous_timestamp = config
        .end_timestamp
        .checked_sub(1)
        .ok_or_else(|| anyhow!("end timestamp 0 has no previous super-root timestamp"))?;
    let previous = fetch_superroot_at_timestamp(&client, previous_timestamp).await?;
    let current = fetch_superroot_at_timestamp(&client, config.end_timestamp).await?;
    let l1_head_number = fetch_l1_head_number(&config.l1_node_address, config.l1_head).await?;

    let synthesized = synthesize_execution(
        config.end_timestamp,
        config.l1_head,
        l1_head_number,
        &previous,
        &current,
    )
    .context("failed to synthesize super-range inputs")?;
    let range_host = build_interop_host(&config, &synthesized)?;
    let (range_witness, native_range_outputs) = collect_range_witness(
        range_host,
        &synthesized.range_inputs,
        &synthesized.preloaded_preimages,
    )
    .await?;

    let replayed_range_outputs = match replay_range(
        config.native_core,
        &synthesized.range_inputs,
        range_witness,
        super_range_elf.clone(),
    )
    .await
    {
        Ok(outputs) => outputs,
        Err(err) => {
            tracing::info!("super-range range replay failed (claim rejected): {err}");
            return Ok(Verdict::Invalid);
        }
    };

    if replayed_range_outputs != native_range_outputs {
        tracing::error!(
            expected = ?native_range_outputs,
            actual = ?replayed_range_outputs,
            "super-range replay output differs from native witness-discovery pass",
        );
        return Ok(Verdict::Invalid);
    }

    ensure_range_outputs_match_inputs(&replayed_range_outputs, &synthesized.range_inputs)?;

    let consolidation_host = build_interop_host(&config, &synthesized)?;
    let (consolidation_witness, native_consolidation_outputs) = collect_consolidation_witness(
        consolidation_host,
        &synthesized.consolidation_inputs,
        &synthesized.preloaded_preimages,
    )
    .await?;

    let replayed_consolidation_outputs = match replay_consolidation(
        config.native_core,
        &synthesized.consolidation_inputs,
        consolidation_witness,
        super_range_elf,
    )
    .await
    {
        Ok(outputs) => outputs,
        Err(err) => {
            tracing::info!("super-range consolidation replay failed (claim rejected): {err}");
            return Ok(Verdict::Invalid);
        }
    };

    if replayed_consolidation_outputs != native_consolidation_outputs {
        tracing::error!(
            expected = ?native_consolidation_outputs,
            actual = ?replayed_consolidation_outputs,
            "super-range consolidation replay output differs from native witness-discovery pass",
        );
        return Ok(Verdict::Invalid);
    }

    ensure_consolidation_outputs_match_inputs(
        &replayed_consolidation_outputs,
        &synthesized.consolidation_inputs,
    )?;
    tracing::info!(
        timestamp = config.end_timestamp,
        chain_count = synthesized.range_inputs.chain_ids.len(),
        native_core = config.native_core,
        "super-range program validated the optimistic transitions and consolidation",
    );
    Ok(Verdict::Valid)
}

/// Fetches `superroot_atTimestamp` from the supplied supernode client.
pub async fn fetch_superroot_at_timestamp(
    client: &HttpClient,
    timestamp: u64,
) -> Result<SuperRootAtTimestampResponse> {
    client
        .request("superroot_atTimestamp", rpc_params![format!("0x{timestamp:x}")])
        .await
        .with_context(|| format!("superroot_atTimestamp({timestamp}) failed"))
}

/// Builds one-timestamp super-range inputs from adjacent supernode responses.
pub fn synthesize_execution(
    end_timestamp: u64,
    l1_head: B256,
    l1_head_number: u64,
    previous: &SuperRootAtTimestampResponse,
    current: &SuperRootAtTimestampResponse,
) -> Result<SynthesizedExecution> {
    let previous_timestamp = end_timestamp
        .checked_sub(1)
        .ok_or_else(|| anyhow!("end timestamp 0 has no previous super-root timestamp"))?;
    let previous_data = previous
        .data
        .as_ref()
        .ok_or_else(|| anyhow!("previous superroot_atTimestamp response has no data"))?;
    let current_data = current
        .data
        .as_ref()
        .ok_or_else(|| anyhow!("current superroot_atTimestamp response has no data"))?;

    ensure!(
        previous_data.super_v1.timestamp == previous_timestamp,
        "previous super-root timestamp {} does not match expected {previous_timestamp}",
        previous_data.super_v1.timestamp,
    );
    ensure!(
        current_data.super_v1.timestamp == end_timestamp,
        "current super-root timestamp {} does not match expected {end_timestamp}",
        current_data.super_v1.timestamp,
    );

    let chain_ids = response_chain_ids(current)?;
    ensure_chain_ids_match_super(&chain_ids, &previous_data.super_v1, "previous")?;
    ensure_chain_ids_match_super(&chain_ids, &current_data.super_v1, "current")?;

    let previous_proof = proof_from_super_v1(&previous_data.super_v1)?;
    let previous_super_root = hash_super_root_proof(&previous_proof)?;
    ensure!(
        previous_super_root == previous_data.super_root,
        "previous super-root proof hashes to {previous_super_root}, response reports {}",
        previous_data.super_root,
    );

    let current_proof = proof_from_super_v1(&current_data.super_v1)?;
    let current_super_root = hash_super_root_proof(&current_proof)?;
    ensure!(
        current_super_root == current_data.super_root,
        "current super-root proof hashes to {current_super_root}, response reports {}",
        current_data.super_root,
    );

    let mut preloaded_preimages = Vec::new();
    preload_super_root_proof(&mut preloaded_preimages, &previous_proof)?;
    preload_super_root_proof(&mut preloaded_preimages, &current_proof)?;

    let mut claimed_transitions = Vec::with_capacity(chain_ids.len());
    for &chain_id in &chain_ids {
        let current_output = output_entry(current, chain_id, "current")?;
        ensure_required_l1_within_head(current_output.required_l1, l1_head_number, chain_id)?;
        let (current_root, current_preimage) =
            checked_output_preimage(current_output, chain_id, "current")?;
        preloaded_preimages.push((PreimageKey::new_keccak256(*current_root), current_preimage));

        claimed_transitions.push(SuperRangeTransition {
            timestamp: end_timestamp,
            optimistic_block: SuperOptimisticBlock {
                chain_id: U256::from(chain_id),
                block_hash: current_output
                    .output
                    .as_ref()
                    .expect("checked output exists")
                    .block_hash,
                output_root: current_root,
            },
        });
    }

    for &chain_id in &chain_ids {
        let previous_output = output_entry(previous, chain_id, "previous")?;
        ensure_required_l1_within_head(previous_output.required_l1, l1_head_number, chain_id)?;
        let (previous_root, previous_preimage) =
            checked_output_preimage(previous_output, chain_id, "previous")?;
        let proof_root = previous_proof
            .super_root
            .output_roots
            .iter()
            .find(|root| root.chain_id == chain_id)
            .map(|root| root.output_root)
            .ok_or_else(|| anyhow!("previous proof missing chain {chain_id}"))?;
        ensure!(
            previous_root == proof_root,
            "previous optimistic output root {previous_root} for chain {chain_id} does not match previous super-root proof output {proof_root}",
        );
        preloaded_preimages.push((PreimageKey::new_keccak256(*previous_root), previous_preimage));
    }

    let range_inputs = SuperRangeInputs {
        span: TimestampSpan::new(end_timestamp, end_timestamp)?,
        l1_head,
        chain_ids: chain_ids.iter().copied().map(U256::from).collect(),
        previous_super_root_proofs: vec![previous_proof.clone()],
        claimed_transitions,
    };
    range_inputs.validate()?;

    let consolidation_inputs = SuperConsolidationInputs {
        span: TimestampSpan::new(end_timestamp, end_timestamp)?,
        previous_super_root,
        transitions: vec![SuperConsolidationTransitionInput {
            optimistic_blocks: range_inputs
                .claimed_transitions
                .iter()
                .map(|transition| transition.optimistic_block)
                .collect(),
            claimed_super_root_proof: current_proof,
        }],
    };
    consolidation_inputs.validate()?;

    Ok(SynthesizedExecution {
        previous_timestamp,
        range_inputs,
        consolidation_inputs,
        preloaded_preimages,
        current_super_root,
        previous_super_root_proof_bytes: encode_super_root_proof(&previous_proof)?,
    })
}

/// Encodes super-range SP1 stdin as typed inputs followed by rkyv witness bytes.
pub fn build_super_range_stdin(
    inputs: &SuperRangeInputs,
    witness: DefaultWitnessData,
) -> Result<SP1Stdin> {
    build_super_interop_stdin(SuperInteropInputs::Range(inputs.clone()), witness)
}

/// Encodes consolidation-mode SP1 stdin as typed inputs followed by rkyv witness bytes.
pub fn build_super_consolidation_stdin(
    inputs: &SuperConsolidationInputs,
    witness: DefaultWitnessData,
) -> Result<SP1Stdin> {
    build_super_interop_stdin(SuperInteropInputs::Consolidation(inputs.clone()), witness)
}

fn build_super_interop_stdin(
    inputs: SuperInteropInputs,
    witness: DefaultWitnessData,
) -> Result<SP1Stdin> {
    let mut stdin = SP1Stdin::new();
    stdin.write(&inputs);
    let witness_bytes = rkyv::to_bytes::<rkyv::rancor::Error>(&witness)?;
    stdin.write_slice(&witness_bytes);
    Ok(stdin)
}

/// Decodes the public output committed by the super-range guest.
pub fn decode_super_range_public_values(
    public_values: &mut SP1PublicValues,
) -> Result<SuperRangeOutputs> {
    ensure!(!public_values.as_slice().is_empty(), "super-range guest committed no output");
    match public_values.read::<SuperInteropOutputs>() {
        SuperInteropOutputs::Range(outputs) => Ok(outputs),
        SuperInteropOutputs::Consolidation(_) => {
            bail!("super-range guest committed consolidation output in range mode")
        }
    }
}

/// Decodes the consolidation output committed by the super-range guest.
pub fn decode_super_consolidation_public_values(
    public_values: &mut SP1PublicValues,
) -> Result<SuperConsolidationOutputs> {
    ensure!(!public_values.as_slice().is_empty(), "super-range guest committed no output");
    match public_values.read::<SuperInteropOutputs>() {
        SuperInteropOutputs::Consolidation(outputs) => Ok(outputs),
        SuperInteropOutputs::Range(_) => {
            bail!("super-range guest committed range output in consolidation mode")
        }
    }
}

async fn execute_sp1_range(
    inputs: &SuperRangeInputs,
    witness: DefaultWitnessData,
    super_range_elf: Arc<[u8]>,
) -> Result<SuperRangeOutputs> {
    let stdin = build_super_range_stdin(inputs, witness)?;
    let client = ProverClient::builder().cpu().build().await;
    let elf = Elf::Dynamic(super_range_elf);
    let (mut public_values, _report) = client.execute(elf, stdin).await?;
    decode_super_range_public_values(&mut public_values)
}

async fn execute_sp1_consolidation(
    inputs: &SuperConsolidationInputs,
    witness: DefaultWitnessData,
    super_range_elf: Arc<[u8]>,
) -> Result<SuperConsolidationOutputs> {
    let stdin = build_super_consolidation_stdin(inputs, witness)?;
    let client = ProverClient::builder().cpu().build().await;
    let elf = Elf::Dynamic(super_range_elf);
    let (mut public_values, _report) = client.execute(elf, stdin).await?;
    decode_super_consolidation_public_values(&mut public_values)
}

async fn replay_range(
    native_core: bool,
    inputs: &SuperRangeInputs,
    witness: DefaultWitnessData,
    super_range_elf: Option<Arc<[u8]>>,
) -> Result<SuperRangeOutputs> {
    if !native_core {
        return execute_sp1_range(
            inputs,
            witness,
            super_range_elf.expect("super-range ELF is loaded for SP1 execution"),
        )
        .await;
    }
    let (oracle, beacon) = witness.get_oracle_and_blob_provider().await?;
    build_range_outputs(inputs.clone(), oracle, beacon).await
}

async fn replay_consolidation(
    native_core: bool,
    inputs: &SuperConsolidationInputs,
    witness: DefaultWitnessData,
    super_range_elf: Option<Arc<[u8]>>,
) -> Result<SuperConsolidationOutputs> {
    if !native_core {
        return execute_sp1_consolidation(
            inputs,
            witness,
            super_range_elf.expect("super-range ELF is loaded for SP1 execution"),
        )
        .await;
    }
    let (oracle, _) = witness.get_oracle_and_blob_provider().await?;
    build_consolidation_outputs(inputs.clone(), oracle).await
}

fn ensure_range_outputs_match_inputs(
    outputs: &SuperRangeOutputs,
    inputs: &SuperRangeInputs,
) -> Result<()> {
    let expected_previous_super_roots = inputs
        .previous_super_root_proofs
        .iter()
        .map(hash_super_root_proof)
        .collect::<std::result::Result<Vec<_>, _>>()?;
    ensure!(outputs.span == inputs.span, "guest output span does not match inputs");
    ensure!(outputs.l1_head == inputs.l1_head, "guest output L1 head does not match inputs");
    ensure!(
        outputs.previous_super_roots == expected_previous_super_roots,
        "guest output previous super roots do not match inputs",
    );
    ensure!(
        outputs.transitions == inputs.claimed_transitions,
        "guest output transitions do not match inputs",
    );
    Ok(())
}

fn ensure_consolidation_outputs_match_inputs(
    outputs: &SuperConsolidationOutputs,
    inputs: &SuperConsolidationInputs,
) -> Result<()> {
    ensure!(outputs.span == inputs.span, "guest consolidation span does not match inputs");
    ensure!(
        outputs.previous_super_root == inputs.previous_super_root,
        "guest consolidation previous super root does not match inputs",
    );
    ensure!(
        outputs.transitions.len() == inputs.transitions.len(),
        "guest consolidation transition count does not match inputs",
    );
    for (output, input) in outputs.transitions.iter().zip(&inputs.transitions) {
        let expected_super_root = hash_super_root_proof(&input.claimed_super_root_proof)?;
        ensure!(
            output.timestamp == input.claimed_super_root_proof.super_root.timestamp,
            "guest consolidation transition timestamp does not match input claim",
        );
        ensure!(
            output.optimistic_blocks == input.optimistic_blocks,
            "guest consolidation optimistic blocks do not match inputs",
        );
        ensure!(
            output.super_root == expected_super_root,
            "guest consolidation super root {} does not match claimed {expected_super_root}",
            output.super_root,
        );
    }
    Ok(())
}

async fn collect_range_witness(
    host: InteropHost,
    inputs: &SuperRangeInputs,
    preloaded_preimages: &[(PreimageKey, Vec<u8>)],
) -> Result<(DefaultWitnessData, SuperRangeOutputs)> {
    collect_witness(preloaded_preimages, host, |oracle, beacon| {
        let inputs = inputs.clone();
        async move { build_range_outputs(inputs, oracle, beacon).await }
    })
    .await
}

async fn collect_consolidation_witness(
    host: InteropHost,
    inputs: &SuperConsolidationInputs,
    preloaded_preimages: &[(PreimageKey, Vec<u8>)],
) -> Result<(DefaultWitnessData, SuperConsolidationOutputs)> {
    collect_witness(preloaded_preimages, host, |oracle, _beacon| {
        let inputs = inputs.clone();
        async move { build_consolidation_outputs(inputs, oracle).await }
    })
    .await
}

async fn collect_witness<T, F, Fut>(
    preloaded_preimages: &[(PreimageKey, Vec<u8>)],
    host: InteropHost,
    run_native: F,
) -> Result<(DefaultWitnessData, T)>
where
    F: FnOnce(
        Arc<PreimageWitnessCollector<SeededOracleBase>>,
        OnlineBlobStore<OracleBlobProvider<SeededOracleBase>>,
    ) -> Fut,
    Fut: std::future::Future<Output = Result<T>>,
{
    let preimage = BidirectionalChannel::new()?;
    let hint = BidirectionalChannel::new()?;

    let mut seeded_store = PreimageStore::default();
    for (key, value) in preloaded_preimages {
        seeded_store.save_preimage(*key, value.clone())?;
    }

    let preimage_witness_store = Arc::new(Mutex::new(seeded_store.clone()));
    let blob_data = Arc::new(Mutex::new(BlobData::default()));
    let host_oracle =
        CachingOracle::new(2048, OracleReader::new(preimage.client), HintWriter::new(hint.client));
    let seeded_oracle = SeededOracle::new(host_oracle, Arc::new(seeded_store));
    let preimage_oracle = Arc::new(seeded_oracle);
    let oracle = Arc::new(PreimageWitnessCollector {
        preimage_oracle: preimage_oracle.clone(),
        preimage_witness_store: preimage_witness_store.clone(),
    });
    let beacon = OnlineBlobStore {
        provider: OracleBlobProvider::new(preimage_oracle),
        store: blob_data.clone(),
    };

    let server_task = host.start_server(hint.host, preimage.host).await?;
    let native_outputs = run_native(oracle, beacon).await;
    server_task.abort();
    let native_outputs = native_outputs?;

    let witness = DefaultWitnessData::from_parts(
        preimage_witness_store
            .lock()
            .map_err(|_| anyhow!("failed to acquire preimage witness store lock"))?
            .clone(),
        blob_data.lock().map_err(|_| anyhow!("failed to acquire blob data lock"))?.clone(),
    );
    Ok((witness, native_outputs))
}

fn build_interop_host(
    config: &RunConfig,
    synthesized: &SynthesizedExecution,
) -> Result<InteropHost> {
    ensure!(!config.l2_node_addresses.is_empty(), "at least one L2 node address is required",);
    Ok(InteropHost {
        l1_head: config.l1_head,
        agreed_l2_pre_state: Bytes::from(synthesized.previous_super_root_proof_bytes.clone()),
        claimed_l2_post_state: synthesized.current_super_root,
        claimed_l2_timestamp: config.end_timestamp,
        l2_node_addresses: Some(config.l2_node_addresses.clone()),
        l1_node_address: Some(config.l1_node_address.clone()),
        l1_beacon_address: Some(config.l1_beacon_address.clone()),
        data_dir: None,
        data_format: DataFormat::Directory,
        native: false,
        server: true,
        rollup_config_paths: config.rollup_config_paths.clone(),
        l1_config_path: config.l1_config_path.clone(),
        dependency_set_path: config.dependency_set_path.clone(),
    })
}

async fn fetch_l1_head_number(l1_node_address: &str, l1_head: B256) -> Result<u64> {
    let client = HttpClientBuilder::default()
        .build(l1_node_address)
        .with_context(|| format!("failed to connect to L1 node {l1_node_address}"))?;
    let block: Option<RpcBlock> = client
        .request("eth_getBlockByHash", rpc_params![format!("{l1_head}"), false])
        .await
        .with_context(|| format!("failed to fetch L1 head {l1_head}"))?;
    let block = block.ok_or_else(|| anyhow!("L1 head {l1_head} not found"))?;
    ensure!(
        block.hash == l1_head,
        "L1 head lookup returned hash {}, expected {l1_head}",
        block.hash,
    );
    Ok(block.number)
}

fn response_chain_ids(response: &SuperRootAtTimestampResponse) -> Result<Vec<u64>> {
    ensure!(!response.chain_ids.is_empty(), "supernode response has no chain IDs");
    let mut out = Vec::with_capacity(response.chain_ids.len());
    for chain_id in &response.chain_ids {
        out.push(chain_id.as_u64()?);
    }
    ensure_strictly_increasing_u64s(&out, "supernode response chain IDs")?;
    Ok(out)
}

fn ensure_chain_ids_match_super(chain_ids: &[u64], super_v1: &SuperV1, label: &str) -> Result<()> {
    let super_chain_ids =
        super_v1.chains.iter().map(|chain| chain.chain_id.as_u64()).collect::<Result<Vec<_>>>()?;
    ensure!(
        chain_ids == super_chain_ids,
        "{label} super-root chain IDs {super_chain_ids:?} do not match response chain IDs {chain_ids:?}",
    );
    Ok(())
}

fn proof_from_super_v1(super_v1: &SuperV1) -> Result<SuperRootProof> {
    let output_roots = super_v1
        .chains
        .iter()
        .map(|chain| Ok(SuperOutputRoot::new(chain.chain_id.as_u64()?, chain.output)))
        .collect::<Result<Vec<_>>>()?;
    Ok(SuperRootProof::new(super_v1.timestamp, output_roots))
}

fn preload_super_root_proof(
    preloaded_preimages: &mut Vec<(PreimageKey, Vec<u8>)>,
    proof: &SuperRootProof,
) -> Result<()> {
    let encoded = encode_super_root_proof(proof)?;
    let hash = hash_super_root_proof(proof)?;
    preloaded_preimages.push((PreimageKey::new_keccak256(*hash), encoded));
    Ok(())
}

fn output_entry<'a>(
    response: &'a SuperRootAtTimestampResponse,
    chain_id: u64,
    label: &str,
) -> Result<&'a OutputWithRequiredL1> {
    response.optimistic_at_timestamp.get(&ChainId(U256::from(chain_id))).ok_or_else(|| {
        anyhow!("missing optimistic output for chain {chain_id} in {label} response")
    })
}

fn ensure_required_l1_within_head(
    required_l1: BlockId,
    l1_head_number: u64,
    chain_id: u64,
) -> Result<()> {
    ensure!(
        required_l1.number <= l1_head_number,
        "required L1 block {} for chain {chain_id} is after supplied L1 head {l1_head_number}",
        required_l1.number,
    );
    Ok(())
}

fn checked_output_preimage(
    output: &OutputWithRequiredL1,
    chain_id: u64,
    label: &str,
) -> Result<(B256, Vec<u8>)> {
    let output_v0 = output.output.as_ref().ok_or_else(|| {
        anyhow!("missing optimistic OutputV0 for chain {chain_id} in {label} response")
    })?;
    let preimage = output_v0.marshal();
    let computed = B256::from(keccak256(&preimage));
    ensure!(
        computed == output.output_root,
        "{label} output root for chain {chain_id} hashes to {computed}, response reports {}",
        output.output_root,
    );
    Ok((computed, preimage))
}

fn ensure_strictly_increasing_u64s(values: &[u64], label: &str) -> Result<()> {
    for pair in values.windows(2) {
        ensure!(
            pair[0] < pair[1],
            "{label} must be strictly increasing, got {} then {}",
            pair[0],
            pair[1],
        );
    }
    Ok(())
}

#[derive(Clone)]
struct SeededOracle<P> {
    inner: P,
    seeds: Arc<PreimageStore>,
}

type SeededOracleBase =
    SeededOracle<CachingOracle<OracleReader<NativeChannel>, HintWriter<NativeChannel>>>;

impl<P> SeededOracle<P> {
    const fn new(inner: P, seeds: Arc<PreimageStore>) -> Self {
        Self { inner, seeds }
    }
}

impl<P> Debug for SeededOracle<P>
where
    P: Debug,
{
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        f.debug_struct("SeededOracle").field("inner", &self.inner).finish_non_exhaustive()
    }
}

#[async_trait]
impl<P> PreimageOracleClient for SeededOracle<P>
where
    P: PreimageOracleClient + Send + Sync,
{
    async fn get(&self, key: PreimageKey) -> PreimageOracleResult<Vec<u8>> {
        match self.seeds.get(key).await {
            Ok(value) => Ok(value),
            Err(_) => self.inner.get(key).await,
        }
    }

    async fn get_exact(&self, key: PreimageKey, buf: &mut [u8]) -> PreimageOracleResult<()> {
        match self.seeds.get(key).await {
            Ok(value) if value.len() == buf.len() => {
                buf.copy_from_slice(&value);
                Ok(())
            }
            _ => self.inner.get_exact(key, buf).await,
        }
    }
}

#[async_trait]
impl<P> HintWriterClient for SeededOracle<P>
where
    P: HintWriterClient + Send + Sync,
{
    async fn write(&self, hint: &str) -> PreimageOracleResult<()> {
        self.inner.write(hint).await
    }
}

impl<P> FlushableCache for SeededOracle<P>
where
    P: FlushableCache,
{
    fn flush(&self) {
        self.inner.flush();
    }
}

/// Decoded response from `superroot_atTimestamp`.
#[derive(Clone, Debug, PartialEq, Eq, Deserialize)]
pub struct SuperRootAtTimestampResponse {
    /// Latest L1 block currently being processed by the supernode.
    pub current_l1: BlockId,
    /// Optimistic outputs keyed by decimal or hex chain ID.
    #[serde(default)]
    pub optimistic_at_timestamp: BTreeMap<ChainId, OutputWithRequiredL1>,
    /// Dependency-set chain IDs sorted ascending by the supernode.
    #[serde(default)]
    pub chain_ids: Vec<ChainId>,
    /// Validated super-root data for the requested timestamp.
    pub data: Option<SuperRootResponseData>,
}

/// Decoded super-root data section.
#[derive(Clone, Debug, PartialEq, Eq, Deserialize)]
pub struct SuperRootResponseData {
    /// Required L1 block for fully verified data.
    pub verified_required_l1: BlockId,
    /// Super-root proof preimage.
    #[serde(rename = "super")]
    pub super_v1: SuperV1,
    /// Super-root hash.
    pub super_root: B256,
}

/// Decoded `eth.SuperV1`.
#[derive(Clone, Debug, PartialEq, Eq, Deserialize)]
pub struct SuperV1 {
    /// Super-root timestamp.
    #[serde(deserialize_with = "deserialize_u64_or_hex")]
    pub timestamp: u64,
    /// Per-chain output roots.
    pub chains: Vec<ChainIdAndOutput>,
}

/// Decoded `eth.ChainIDAndOutput`.
#[derive(Clone, Debug, PartialEq, Eq, Deserialize)]
pub struct ChainIdAndOutput {
    /// Chain ID.
    #[serde(rename = "ChainID")]
    pub chain_id: ChainId,
    /// Output root.
    #[serde(rename = "Output")]
    pub output: B256,
}

/// Decoded `eth.OutputWithRequiredL1`.
#[derive(Clone, Debug, PartialEq, Eq, Deserialize)]
pub struct OutputWithRequiredL1 {
    /// Output-root preimage, if the supernode supplied one.
    pub output: Option<OutputV0>,
    /// Output-root hash.
    pub output_root: B256,
    /// Minimum required L1 block.
    pub required_l1: BlockId,
}

/// Decoded `eth.OutputV0`.
#[derive(Clone, Debug, PartialEq, Eq, Deserialize)]
pub struct OutputV0 {
    /// L2 state root.
    #[serde(rename = "stateRoot")]
    pub state_root: B256,
    /// `L2ToL1MessagePasser` storage root.
    #[serde(rename = "messagePasserStorageRoot")]
    pub message_passer_storage_root: B256,
    /// L2 block hash.
    #[serde(rename = "blockHash")]
    pub block_hash: B256,
}

impl OutputV0 {
    fn marshal(&self) -> Vec<u8> {
        let mut out = vec![0u8; 128];
        out[32..64].copy_from_slice(self.state_root.as_slice());
        out[64..96].copy_from_slice(self.message_passer_storage_root.as_slice());
        out[96..128].copy_from_slice(self.block_hash.as_slice());
        out
    }
}

/// Decoded `eth.BlockID`.
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq, Deserialize)]
pub struct BlockId {
    /// Block hash.
    pub hash: B256,
    /// Block number.
    #[serde(deserialize_with = "deserialize_u64_or_hex")]
    pub number: u64,
}

/// JSON map key for `eth.ChainID`.
#[derive(Clone, Copy, Debug, PartialEq, Eq, PartialOrd, Ord, Hash)]
pub struct ChainId(pub U256);

impl ChainId {
    fn as_u64(self) -> Result<u64> {
        if self.0 > U256::from(u64::MAX) {
            bail!("chain ID {} does not fit in u64", self.0);
        }
        Ok(self.0.saturating_to::<u64>())
    }
}

impl<'de> Deserialize<'de> for ChainId {
    fn deserialize<D>(deserializer: D) -> std::result::Result<Self, D::Error>
    where
        D: serde::Deserializer<'de>,
    {
        deserializer.deserialize_any(ChainIdVisitor)
    }
}

struct ChainIdVisitor;

impl serde::de::Visitor<'_> for ChainIdVisitor {
    type Value = ChainId;

    fn expecting(&self, formatter: &mut fmt::Formatter<'_>) -> fmt::Result {
        formatter.write_str("a decimal or 0x-prefixed chain ID")
    }

    fn visit_u64<E>(self, value: u64) -> std::result::Result<Self::Value, E>
    where
        E: serde::de::Error,
    {
        Ok(ChainId(U256::from(value)))
    }

    fn visit_str<E>(self, value: &str) -> std::result::Result<Self::Value, E>
    where
        E: serde::de::Error,
    {
        parse_u256(value).map(ChainId).map_err(E::custom)
    }
}

fn deserialize_u64_or_hex<'de, D>(deserializer: D) -> std::result::Result<u64, D::Error>
where
    D: serde::Deserializer<'de>,
{
    deserializer.deserialize_any(U64Visitor)
}

struct U64Visitor;

impl serde::de::Visitor<'_> for U64Visitor {
    type Value = u64;

    fn expecting(&self, formatter: &mut fmt::Formatter<'_>) -> fmt::Result {
        formatter.write_str("a JSON number or 0x-prefixed quantity")
    }

    fn visit_u64<E>(self, value: u64) -> std::result::Result<Self::Value, E>
    where
        E: serde::de::Error,
    {
        Ok(value)
    }

    fn visit_str<E>(self, value: &str) -> std::result::Result<Self::Value, E>
    where
        E: serde::de::Error,
    {
        parse_u64(value).map_err(E::custom)
    }
}

fn parse_u64(value: &str) -> Result<u64> {
    value.strip_prefix("0x").map_or_else(
        || value.parse::<u64>().with_context(|| format!("invalid decimal quantity {value}")),
        |hex| u64::from_str_radix(hex, 16).with_context(|| format!("invalid hex quantity {value}")),
    )
}

fn parse_u256(value: &str) -> Result<U256> {
    value.strip_prefix("0x").map_or_else(
        || U256::from_str(value).with_context(|| format!("invalid decimal U256 {value}")),
        |hex| U256::from_str_radix(hex, 16).with_context(|| format!("invalid hex U256 {value}")),
    )
}

#[derive(Clone, Debug, PartialEq, Eq, Deserialize)]
struct RpcBlock {
    hash: B256,
    #[serde(deserialize_with = "deserialize_u64_or_hex")]
    number: u64,
}

#[cfg(test)]
mod tests {
    use super::*;
    use kona_preimage::errors::PreimageOracleError;

    fn b256(fill: u8) -> B256 {
        B256::from([fill; 32])
    }

    fn block(number: u64) -> BlockId {
        BlockId { hash: b256(number as u8), number }
    }

    fn output(block_hash: B256) -> OutputV0 {
        OutputV0 { state_root: b256(0x11), message_passer_storage_root: b256(0x22), block_hash }
    }

    fn output_entry(block_hash: B256, required_l1: u64) -> OutputWithRequiredL1 {
        let output = output(block_hash);
        let output_root = B256::from(keccak256(output.marshal()));
        OutputWithRequiredL1 { output: Some(output), output_root, required_l1: block(required_l1) }
    }

    fn response(
        timestamp: u64,
        chain_ids: &[u64],
        required_l1: u64,
    ) -> SuperRootAtTimestampResponse {
        let mut optimistic_at_timestamp = BTreeMap::new();
        let mut chains = Vec::new();
        for &chain_id in chain_ids {
            let entry = output_entry(b256(chain_id as u8), required_l1);
            optimistic_at_timestamp.insert(ChainId(U256::from(chain_id)), entry.clone());
            chains.push(ChainIdAndOutput {
                chain_id: ChainId(U256::from(chain_id)),
                output: entry.output_root,
            });
        }
        let super_v1 = SuperV1 { timestamp, chains };
        let proof = proof_from_super_v1(&super_v1).expect("proof builds");
        let super_root = hash_super_root_proof(&proof).expect("proof hashes");
        SuperRootAtTimestampResponse {
            current_l1: block(required_l1 + 1),
            optimistic_at_timestamp,
            chain_ids: chain_ids.iter().copied().map(|id| ChainId(U256::from(id))).collect(),
            data: Some(SuperRootResponseData {
                verified_required_l1: block(required_l1),
                super_v1,
                super_root,
            }),
        }
    }

    #[test]
    fn synthesize_execution_orders_chains_and_preloads_outputs() {
        let previous = response(100, &[10, 20], 7);
        let current = response(101, &[10, 20], 8);

        let synthesized =
            synthesize_execution(101, b256(0xaa), 8, &previous, &current).expect("synthesizes");

        assert_eq!(synthesized.previous_timestamp, 100);
        assert_eq!(synthesized.range_inputs.span, TimestampSpan { start: 101, end: 101 });
        assert_eq!(synthesized.range_inputs.chain_ids, vec![U256::from(10), U256::from(20)]);
        assert_eq!(
            synthesized.range_inputs.previous_super_root_proofs[0].super_root.timestamp,
            100,
        );
        assert_eq!(synthesized.range_inputs.claimed_transitions.len(), 2);
        assert_eq!(
            synthesized
                .range_inputs
                .claimed_transitions
                .iter()
                .map(|transition| transition.optimistic_block.chain_id)
                .collect::<Vec<_>>(),
            vec![U256::from(10), U256::from(20)],
        );
        assert_eq!(synthesized.preloaded_preimages.len(), 6);
    }

    #[test]
    fn synthesize_execution_uses_optimistic_range_inputs_and_verified_consolidation_claim() {
        let previous = response(100, &[10, 20], 7);
        let mut current = response(101, &[10, 20], 8);
        let replacement_shaped = output_entry(b256(0xfe), 8);
        current.optimistic_at_timestamp.insert(ChainId(U256::from(20)), replacement_shaped.clone());

        let synthesized =
            synthesize_execution(101, b256(0xaa), 8, &previous, &current).expect("synthesizes");

        assert_eq!(
            synthesized.range_inputs.claimed_transitions[1].optimistic_block.output_root,
            replacement_shaped.output_root,
        );
        assert_eq!(
            synthesized.consolidation_inputs.transitions[0].optimistic_blocks[1].output_root,
            replacement_shaped.output_root,
        );
        assert_eq!(
            synthesized.consolidation_inputs.transitions[0]
                .claimed_super_root_proof
                .super_root
                .output_roots[1]
                .output_root,
            current.data.as_ref().unwrap().super_v1.chains[1].output,
        );
        assert_ne!(
            replacement_shaped.output_root,
            current.data.as_ref().unwrap().super_v1.chains[1].output,
        );
    }

    #[test]
    fn synthesize_execution_rejects_unsorted_chain_ids() {
        let previous = response(100, &[10, 20], 7);
        let current = response(101, &[20, 10], 8);

        let err = synthesize_execution(101, b256(0xaa), 8, &previous, &current).unwrap_err();

        assert!(err.to_string().contains("strictly increasing"));
    }

    #[test]
    fn synthesize_execution_rejects_missing_optimistic_output() {
        let previous = response(100, &[10, 20], 7);
        let mut current = response(101, &[10, 20], 8);
        current.optimistic_at_timestamp.remove(&ChainId(U256::from(20)));

        let err = synthesize_execution(101, b256(0xaa), 8, &previous, &current).unwrap_err();

        assert!(err.to_string().contains("missing optimistic output for chain 20"));
    }

    #[test]
    fn synthesize_execution_rejects_required_l1_after_l1_head() {
        let previous = response(100, &[10, 20], 7);
        let current = response(101, &[10, 20], 9);

        let err = synthesize_execution(101, b256(0xaa), 8, &previous, &current).unwrap_err();

        assert!(err.to_string().contains("after supplied L1 head"));
    }

    #[test]
    fn decodes_supernode_json_shapes() {
        let json = r#"{
            "current_l1":{"hash":"0x0101010101010101010101010101010101010101010101010101010101010101","number":9},
            "optimistic_at_timestamp":{
                "10":{
                    "output":{
                        "stateRoot":"0x1111111111111111111111111111111111111111111111111111111111111111",
                        "messagePasserStorageRoot":"0x2222222222222222222222222222222222222222222222222222222222222222",
                        "blockHash":"0x3333333333333333333333333333333333333333333333333333333333333333"
                    },
                    "output_root":"0x4444444444444444444444444444444444444444444444444444444444444444",
                    "required_l1":{"hash":"0x0505050505050505050505050505050505050505050505050505050505050505","number":8}
                }
            },
            "chain_ids":["10"],
            "data":{
                "verified_required_l1":{"hash":"0x0505050505050505050505050505050505050505050505050505050505050505","number":8},
                "super":{
                    "timestamp":"0x65",
                    "chains":[{"ChainID":"10","Output":"0x4444444444444444444444444444444444444444444444444444444444444444"}]
                },
                "super_root":"0x6666666666666666666666666666666666666666666666666666666666666666"
            }
        }"#;

        let decoded: SuperRootAtTimestampResponse = serde_json::from_str(json).unwrap();

        assert_eq!(decoded.current_l1.number, 9);
        assert_eq!(decoded.chain_ids, vec![ChainId(U256::from(10))]);
        assert_eq!(decoded.data.unwrap().super_v1.timestamp, 101);
    }

    #[test]
    fn stdin_and_public_values_round_trip_range_outputs() {
        let previous = response(100, &[10], 7);
        let current = response(101, &[10], 8);
        let synthesized =
            synthesize_execution(101, b256(0xaa), 8, &previous, &current).expect("synthesizes");
        let witness = DefaultWitnessData::default();

        let _stdin =
            build_super_range_stdin(&synthesized.range_inputs, witness).expect("stdin encodes");

        let outputs = SuperRangeOutputs {
            span: synthesized.range_inputs.span,
            l1_head: synthesized.range_inputs.l1_head,
            previous_super_roots: synthesized
                .range_inputs
                .previous_super_root_proofs
                .iter()
                .map(hash_super_root_proof)
                .collect::<std::result::Result<Vec<_>, _>>()
                .unwrap(),
            transitions: synthesized.range_inputs.claimed_transitions.clone(),
        };
        let mut public_values = SP1PublicValues::new();
        public_values.write(&SuperInteropOutputs::Range(outputs.clone()));

        assert_eq!(decode_super_range_public_values(&mut public_values).unwrap(), outputs);
    }

    #[test]
    fn seeded_oracle_prefers_preloaded_preimages() {
        let mut seeds = PreimageStore::default();
        let value = b"seeded".to_vec();
        let key = PreimageKey::new_keccak256(*B256::from(keccak256(&value)));
        seeds.save_preimage(key, value.clone()).unwrap();
        let oracle = SeededOracle::new(FailingOracle, Arc::new(seeds));

        let got = kona_proof::block_on(oracle.get(key)).unwrap();

        assert_eq!(got, value);
    }

    #[derive(Clone, Debug)]
    struct FailingOracle;

    #[async_trait]
    impl PreimageOracleClient for FailingOracle {
        async fn get(&self, _key: PreimageKey) -> PreimageOracleResult<Vec<u8>> {
            Err(PreimageOracleError::InvalidPreimageKey)
        }

        async fn get_exact(&self, _key: PreimageKey, _buf: &mut [u8]) -> PreimageOracleResult<()> {
            Err(PreimageOracleError::InvalidPreimageKey)
        }
    }

    #[async_trait]
    impl HintWriterClient for FailingOracle {
        async fn write(&self, _hint: &str) -> PreimageOracleResult<()> {
            Ok(())
        }
    }

    impl FlushableCache for FailingOracle {
        fn flush(&self) {}
    }
}
