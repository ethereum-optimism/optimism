//! Super-range SP1 executor support.
//!
//! The executor queries `superroot_atTimestamp` for typed super-range inputs and collects real
//! range and consolidation witnesses using the shared native cores. It then replays those
//! witnesses either through the shared cores with `--native-core` or through the real
//! `super-range` SP1 ELF.

use std::{
    borrow::Cow,
    collections::BTreeMap,
    fmt::{self, Debug},
    path::{Path, PathBuf},
    str::FromStr,
    sync::{Arc, Mutex},
};

use alloy_consensus::Header;
use alloy_primitives::{B256, Bytes, U256, keccak256};
use anyhow::{Context, Result, anyhow, bail, ensure};
use async_trait::async_trait;
use jsonrpsee_core::{client::ClientT, rpc_params};
use jsonrpsee_http_client::{HttpClient, HttpClientBuilder};
use kona_host::DataFormat;
pub use kona_host::interop::InteropHost;
use kona_preimage::{
    BidirectionalChannel, DEPENDENCY_SET_KEY, HintWriter, HintWriterClient, L1_CONFIG_KEY,
    L2_ROLLUP_CONFIG_KEY, NativeChannel, OracleReader, PreimageKey, PreimageOracleClient,
    errors::PreimageOracleResult,
};
use kona_proof::{CachingOracle, FlushableCache, l1::OracleBlobProvider};
use kona_proof_interop::HintType;
use kona_sp1_client_utils::{
    log_commitment::{
        BlockReceipts, LogCommitmentInputs, LogOpening, VerifiedLogCommitments,
        log_commitment_public_values, log_commitment_witness,
    },
    super_root::{
        SuperConsolidationInputs, SuperConsolidationOutputs, SuperConsolidationTransitionInput,
        SuperInteropInputs, SuperInteropOutputs, SuperOptimisticBlock, SuperOutputRoot,
        SuperRangeInputs, SuperRangeOutputs, SuperRangeTransition, SuperRootProof, TimestampSpan,
        encode_super_root_proof, hash_super_root_proof,
    },
    witness::{BlobData, DefaultWitnessData, WitnessData, preimage_store::PreimageStore},
};
use kona_sp1_ethereum_client_utils::{
    chain_config::ChainConfigs,
    super_consolidation::build_consolidation_outputs,
    super_range::{build_range_outputs, fetch_l2_header, fetch_output_block_hash},
};
use kona_sp1_host_utils::witness_generation::{OnlineBlobStore, PreimageWitnessCollector};
use op_alloy_consensus::OpReceiptEnvelope;
use serde::Deserialize;
use sp1_sdk::{
    Elf, Prover, ProverClient, SP1Proof, SP1ProofWithPublicValues, SP1PublicValues, SP1Stdin,
    SP1VerifyingKey,
};

mod cost;
pub use cost::{
    ConsolidationCost, DEFAULT_MAX_STDIN_BYTES, LogBatchBlock, MAX_ENTRIES_PER_BATCH,
    ScannedTimestampCost, batch_log_blocks, estimate_consolidation_pgu, estimate_log_block_pgu,
    estimate_log_commitment_pgu, receipts_trie_bytes, split_log_batches,
};
mod l2;
pub use l2::L2BlockClient;
mod log_commitments;
pub use log_commitments::MissingLogCommitment;
use log_commitments::{NativeConsolidation, hinted_receipts, run_native_consolidation};

#[cfg(test)]
mod consolidation_tests;

/// Exit code signalling a valid super-range claim.
pub const EXIT_VALID: u8 = 0;
/// Exit code signalling an invalid super-range claim.
pub const EXIT_INVALID: u8 = 1;
/// Exit code signalling an infrastructure failure before a verdict was reached.
pub const EXIT_INFRA: u8 = 2;

/// Largest witness archive rkyv can write. rkyv 0.8 without `pointer_width_64` stores each
/// relative pointer as an `i32` and panics, rather than erroring, when one overflows. No offset
/// within an archive this size can overflow.
pub const MAX_ARCHIVED_WITNESS_BYTES: u64 = i32::MAX as u64;

/// Archive bytes per preimage beyond the key and value that
/// [`PreimageStore::preimage_bytes`] counts. rkyv's hash map stores each entry as the archived
/// key and an 8-byte vector header in a table of at most `len * 8 / 7 + 1` slots, with one
/// control byte per slot: under 16 bytes per entry, so 64 leaves a 4× margin.
const WITNESS_ENTRY_OVERHEAD_BYTES: u64 = 64;

/// Archive bytes independent of the witness size: the root, vector headers, the hash table's
/// slot rounding and control-byte tail, and alignment padding.
const WITNESS_ARCHIVE_FIXED_BYTES: u64 = 4_096;

/// A witness whose rkyv archive could exceed [`MAX_ARCHIVED_WITNESS_BYTES`].
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct WitnessTooLarge {
    /// Upper bound on the witness's archive size.
    pub bound: u64,
    /// The archive limit.
    pub limit: u64,
}

impl fmt::Display for WitnessTooLarge {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(
            f,
            "witness archive may reach {} bytes, above the {}-byte rkyv limit",
            self.bound, self.limit
        )
    }
}

impl std::error::Error for WitnessTooLarge {}

/// Returns true when `err`'s chain contains a [`WitnessTooLarge`].
pub fn is_witness_too_large(err: &anyhow::Error) -> bool {
    err.chain().any(|cause| cause.is::<WitnessTooLarge>())
}

/// An upper bound on the size of `witness`'s rkyv archive, computed without archiving it.
fn witness_archive_bound(witness: &DefaultWitnessData) -> u64 {
    let preimages = witness.preimage_store();
    let blobs = witness.blob_data();
    preimages.preimage_bytes() as u64 +
        preimages.preimage_count() as u64 * WITNESS_ENTRY_OVERHEAD_BYTES +
        size_of_val(blobs.blobs.as_slice()) as u64 +
        size_of_val(blobs.commitments.as_slice()) as u64 +
        size_of_val(blobs.proofs.as_slice()) as u64 +
        WITNESS_ARCHIVE_FIXED_BYTES
}

/// Archives `witness` with rkyv, or returns [`WitnessTooLarge`] without calling rkyv when its
/// archive could exceed [`MAX_ARCHIVED_WITNESS_BYTES`].
fn archive_witness(witness: &DefaultWitnessData) -> Result<rkyv::util::AlignedVec> {
    archive_witness_within(witness, MAX_ARCHIVED_WITNESS_BYTES)
}

fn archive_witness_within(
    witness: &DefaultWitnessData,
    limit: u64,
) -> Result<rkyv::util::AlignedVec> {
    let bound = witness_archive_bound(witness);
    if bound > limit {
        return Err(WitnessTooLarge { bound, limit }.into());
    }
    rkyv::to_bytes::<rkyv::rancor::Error>(witness).context("rkyv failed to archive the witness")
}

/// CLI/runtime configuration for one super-range execution.
#[derive(Clone, Debug)]
pub struct RunConfig {
    /// Replay collected witnesses through the shared native cores instead of executing the guest.
    /// Like the guest, the replay panics on a preimage missing from the witness.
    pub native_core: bool,
    /// Corrupt the claimed optimistic output root the guest sees, after collecting witnesses on
    /// the honest one, so the guest rejects the claim.
    pub corrupt_claimed_root: bool,
    /// Print the PGU, instruction count, stdin size, and cycle-tracker totals of each SP1 guest
    /// execution.
    pub report: bool,
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

impl RunConfig {
    /// Returns the endpoints and config paths used by `InteropHost`.
    pub fn host_inputs(&self) -> HostInputs {
        HostInputs {
            l1_node_address: self.l1_node_address.clone(),
            l1_beacon_address: self.l1_beacon_address.clone(),
            l2_node_addresses: self.l2_node_addresses.clone(),
            rollup_config_paths: self.rollup_config_paths.clone(),
            l1_config_path: self.l1_config_path.clone(),
            dependency_set_path: self.dependency_set_path.clone(),
        }
    }
}

/// Deployment-scoped endpoints and config files needed to build an [`InteropHost`].
///
/// These fields are fixed per deployment; the per-span facts (L1 head, agreed pre-state,
/// claimed post-state, claimed timestamp) are passed to [`build_interop_host`] separately.
#[derive(Clone, Debug)]
pub struct HostInputs {
    /// L1 execution JSON-RPC endpoint.
    pub l1_node_address: String,
    /// L1 beacon API endpoint.
    pub l1_beacon_address: String,
    /// L2 execution JSON-RPC endpoints.
    pub l2_node_addresses: Vec<String>,
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
    /// Typed inputs for the SP1 guest's range mode.
    pub range_inputs: SuperRangeInputs,
    /// Typed inputs for the SP1 guest's consolidation mode.
    pub consolidation_inputs: SuperConsolidationInputs,
    /// Preimages that must be available before host hinting can discover the rest of the witness.
    /// The first collection caches resolved replacement preimages here and clears
    /// `replaced_agreed_outputs`.
    preloaded_preimages: Vec<(PreimageKey, Vec<u8>)>,
    /// Replaced agreed outputs awaiting first collection, grouped by timestamp.
    replaced_agreed_outputs: Vec<ReplacedAgreedOutputs>,
    /// The span-end super-root hash, used as the host post-state claim.
    pub current_super_root: B256,
    /// The pre-span super-root proof, encoded as the host agreed pre-state.
    pub previous_super_root_proof_bytes: Vec<u8>,
}

/// The agreed outputs that one pre-transition super root of a span replaced. The supernode
/// reports the denied block as optimistic at that timestamp, so the response holds no preimage of
/// these, and kona-host reads them only at the timestamp of its agreed pre-state.
#[derive(Clone, Debug, PartialEq, Eq)]
struct ReplacedAgreedOutputs {
    /// The encoded super-root proof at their timestamp.
    agreed_pre_state: Vec<u8>,
    /// The outputs that the proof commits in place of the optimistic ones.
    outputs: Vec<SuperOutputRoot>,
}

impl SynthesizedExecution {
    /// Collects a range-mode witness and the natively computed range outputs through `host`,
    /// seeded with the preloaded preimages and the agreed outputs that the span's super roots
    /// replaced, which it reads through `host`'s L2 nodes.
    /// Before the first collection, updates `range_inputs` and `consolidation_inputs` to retain
    /// agreed blocks on no-op transitions; build stdin after collecting.
    pub async fn collect_range_witness(
        &mut self,
        host: InteropHost,
        configs: Option<&ChainConfigs>,
    ) -> Result<(DefaultWitnessData, SuperRangeOutputs)> {
        self.prepare_inputs(&host).await?;
        collect_witness(&self.preloaded_preimages, host, |oracle, beacon| {
            let inputs = self.range_inputs.clone();
            async move { build_range_outputs(inputs, oracle, beacon, configs).await }
        })
        .await
    }

    /// Collects a consolidation-mode witness and the natively computed consolidation outputs
    /// through `host`, seeded with the preloaded preimages and the agreed outputs that the span's
    /// super roots replaced, which it reads through `host`'s L2 nodes.
    /// Before the first collection, updates `range_inputs` and `consolidation_inputs` to retain
    /// agreed blocks on no-op transitions; build stdin after collecting.
    ///
    /// `batches` are the public values of the log-commitment batch proofs that the consolidation
    /// stdin will carry; the run takes the guest's path for them (see [`VerifiedLogCommitments`]).
    /// With `log_tree_dir`, the run keeps the log trees it builds there, and builds each tree once.
    pub async fn collect_consolidation_witness(
        &mut self,
        host: InteropHost,
        configs: Option<&ChainConfigs>,
        batches: &[Vec<u8>],
        log_tree_dir: Option<&Path>,
    ) -> Result<CollectedConsolidation> {
        self.prepare_inputs(&host).await?;
        let (witness, native) =
            collect_witness(&self.preloaded_preimages, host, |oracle, _beacon| {
                run_native_consolidation(
                    oracle,
                    self.consolidation_inputs.clone(),
                    configs,
                    batches,
                    log_tree_dir,
                )
            })
            .await?;
        let NativeConsolidation {
            outputs,
            openings,
            missing,
            served,
            read_bytes,
            log_check_siblings,
        } = native;
        Ok(CollectedConsolidation {
            witness,
            outputs,
            openings,
            missing,
            served,
            read_bytes,
            log_check_siblings,
        })
    }

    /// Resolves replaced agreed outputs and retains them on non-advancing transitions.
    async fn prepare_inputs(&mut self, host: &InteropHost) -> Result<()> {
        if self.replaced_agreed_outputs.is_empty() {
            return Ok(());
        }
        let preimages = self
            .seeded_preimages(host)
            .await
            .context("failed to read the replaced agreed outputs")?
            .into_owned();
        let mut store = PreimageStore::default();
        for (key, value) in &preimages {
            store.save_preimage(*key, value.clone())?;
        }
        let chain_count = self.range_inputs.chain_ids.len();
        for ((range, consolidation), previous) in self
            .range_inputs
            .claimed_transitions
            .chunks_exact_mut(chain_count)
            .zip(&mut self.consolidation_inputs.transitions)
            .zip(&self.range_inputs.previous_super_root_proofs)
        {
            for (index, (agreed, claimed)) in previous
                .super_root
                .output_roots
                .iter()
                .zip(&consolidation.claimed_super_root_proof.super_root.output_roots)
                .enumerate()
            {
                if agreed.output_root != claimed.output_root ||
                    range[index].optimistic_block.output_root == agreed.output_root
                {
                    continue;
                }
                // An unchanged verified output retains the agreed block, even after replacement.
                let retained = SuperOptimisticBlock {
                    chain_id: range[index].optimistic_block.chain_id,
                    block_hash: fetch_output_block_hash(&store, agreed.output_root).await?,
                    output_root: agreed.output_root,
                };
                range[index].optimistic_block = retained;
                consolidation.optimistic_blocks[index] = retained;
            }
        }
        self.preloaded_preimages = preimages;
        self.replaced_agreed_outputs.clear();
        Ok(())
    }

    /// The preloaded preimages with those of `replaced_agreed_outputs`. Each timestamp's outputs
    /// are read through a clone of `host` agreeing on that timestamp's super root, since
    /// kona-host's `L2OutputRoot` hint computes the output at the timestamp of its agreed
    /// pre-state (see `replaced_agreed_output_preimages`). The clones keep no data directory, so
    /// they share no store with `host`.
    async fn seeded_preimages(
        &self,
        host: &InteropHost,
    ) -> Result<Cow<'_, [(PreimageKey, Vec<u8>)]>> {
        if self.replaced_agreed_outputs.is_empty() {
            return Ok(Cow::Borrowed(&self.preloaded_preimages));
        }
        let mut preimages = self.preloaded_preimages.clone();
        for replaced in &self.replaced_agreed_outputs {
            let host = InteropHost {
                agreed_l2_pre_state: Bytes::from(replaced.agreed_pre_state.clone()),
                data_dir: None,
                ..host.clone()
            };
            preimages.extend(replaced_agreed_output_preimages(host, &replaced.outputs).await?);
        }
        Ok(Cow::Owned(preimages))
    }
}

/// The verdict of a `mode` replay that failed with `err`. A witness over the archive limit is the
/// executor's limit, not a rejected claim, so it is returned as an error (exit 2).
fn replay_failure(mode: &str, err: anyhow::Error) -> Result<Verdict> {
    if is_witness_too_large(&err) {
        return Err(err);
    }
    tracing::info!("super-range {mode} replay failed (claim rejected): {err}");
    Ok(Verdict::Invalid)
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

    let span = TimestampSpan::new(config.end_timestamp, config.end_timestamp)
        .context("failed to build single-timestamp span")?;
    let mut synthesized =
        synthesize_execution(span, config.l1_head, l1_head_number, &[previous, current])
            .context("failed to synthesize super-range inputs")?;
    let host_inputs = config.host_inputs();
    let range_host = build_interop_host(
        &host_inputs,
        config.l1_head,
        &synthesized.previous_super_root_proof_bytes,
        synthesized.current_super_root,
        config.end_timestamp,
        None,
    )?;
    let configs =
        deployment_chain_configs(&range_host, &synthesized.range_inputs.super_root_chain_ids())?;
    if !config.native_core {
        // Test guests consume Local config keys; production guests ignore these preimages.
        for (key, bytes) in [
            (DEPENDENCY_SET_KEY, serde_json::to_vec(&configs.dependency_set)?),
            (L2_ROLLUP_CONFIG_KEY, serde_json::to_vec(&configs.rollup_configs)?),
            (L1_CONFIG_KEY, serde_json::to_vec(&configs.l1_config)?),
        ] {
            synthesized.preloaded_preimages.push((PreimageKey::new_local(key.to()), bytes));
        }
    }
    let (range_witness, native_range_outputs) =
        synthesized.collect_range_witness(range_host, Some(&configs)).await?;

    // Witnesses were collected on the honest claim; see [`corrupt_range_claim`].
    let replay_range_inputs = if config.corrupt_claimed_root {
        corrupt_range_claim(&synthesized.range_inputs)?
    } else {
        synthesized.range_inputs.clone()
    };

    let replayed_range_outputs = match replay_range(
        config.native_core,
        config.report,
        &replay_range_inputs,
        range_witness,
        super_range_elf.clone(),
        Some(&configs),
    )
    .await
    {
        Ok(outputs) => outputs,
        Err(err) => return replay_failure("range", err),
    };

    if replayed_range_outputs != native_range_outputs {
        tracing::error!(
            expected = ?native_range_outputs,
            actual = ?replayed_range_outputs,
            "super-range replay output differs from native witness-discovery pass",
        );
        return Ok(Verdict::Invalid);
    }

    // Reaching here under --corrupt-claimed-root means the guest ran the tampered claim to
    // completion and still agreed with the honest outputs. That is a soundness failure, not a
    // verdict, so surface it as an error rather than reporting the claim valid.
    ensure!(
        !config.corrupt_claimed_root,
        "super-range guest accepted a corrupted claimed optimistic output root",
    );

    ensure_range_outputs_match_inputs(&replayed_range_outputs, &synthesized.range_inputs)?;

    let mut collect = async |batches: &[Vec<u8>]| {
        let host = build_interop_host(
            &host_inputs,
            config.l1_head,
            &synthesized.previous_super_root_proof_bytes,
            synthesized.current_super_root,
            config.end_timestamp,
            None,
        )?;
        synthesized.collect_consolidation_witness(host, Some(&configs), batches, None).await
    };
    // The table path: commit the blocks below the heads that the first run read, then recollect.
    let mut consolidation = collect(&[]).await?;
    let mut batches = Vec::new();
    if !consolidation.missing.is_empty() {
        batches.push(
            commit_missing_blocks(&config, &consolidation.missing)
                .await
                .context("failed to commit the blocks below the local safe heads")?,
        );
        consolidation = collect(&batches).await?;
        ensure!(
            consolidation.missing.is_empty(),
            "blocks below the local safe heads have no entry after the log stage: {:?}",
            consolidation.missing,
        );
    }
    let native_consolidation_outputs = consolidation.outputs;
    let log_commitments = LogCommitmentInputs { batches, openings: consolidation.openings };

    let replayed_consolidation_outputs = match replay_consolidation(
        config.native_core,
        config.report,
        &synthesized.consolidation_inputs,
        consolidation.witness,
        &log_commitments,
        super_range_elf,
        Some(&configs),
    )
    .await
    {
        Ok(outputs) => outputs,
        Err(err) => return replay_failure("consolidation", err),
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

/// The batch of entries of `missing`, from their receipts on the L2 nodes, computed by the
/// `log-commitment` ELF, or natively with `--native-core`.
async fn commit_missing_blocks(
    config: &RunConfig,
    missing: &[MissingLogCommitment],
) -> Result<Vec<u8>> {
    let mut clients = BTreeMap::new();
    for address in &config.l2_node_addresses {
        let client = L2BlockClient::connect(address).await?;
        clients.insert(client.chain_id().await?, client);
    }
    let mut blocks = Vec::with_capacity(missing.len());
    for block in missing {
        let client = clients
            .get(&block.chain_id)
            .with_context(|| format!("no L2 node for chain {}", block.chain_id))?;
        let receipts = client.receipts(block.block_hash, block.receipts_root).await?;
        blocks.push(BlockReceipts { receipts_root: block.receipts_root, receipts });
    }
    let stdin = build_log_commitment_stdin_from_receipts(&blocks)?;
    if config.native_core {
        return Ok(expected_batch_public_values(&stdin));
    }
    let elf =
        kona_sp1_elfs::load_elf("log-commitment-elf").context("log-commitment ELF unavailable")?;
    let public_values =
        execute_sp1("log-commitment", stdin, elf.into(), config.report, true).await?;
    Ok(public_values.to_vec())
}

/// Resolves deployment files as registry overrides, using embedded values for omitted files.
pub fn deployment_chain_configs(host: &InteropHost, chain_ids: &[U256]) -> Result<ChainConfigs> {
    ensure!(
        chain_ids.iter().all(|id| *id <= U256::from(u64::MAX)),
        "deployment chain ID does not fit in u64"
    );
    let rollup_configs = match host.read_rollup_configs() {
        Some(configs) => configs.context("failed to read deployment rollup configs")?,
        None => chain_ids
            .iter()
            .map(|id| {
                let chain_id = id.saturating_to::<u64>();
                let config =
                    kona_registry::ROLLUP_CONFIGS.get(&chain_id).cloned().ok_or_else(|| {
                        anyhow!("no embedded rollup config for deployment chain ID {chain_id}")
                    })?;
                Ok((chain_id, config))
            })
            .collect::<Result<BTreeMap<_, _>>>()?,
    };
    let dependency_set = match host.read_dependency_set() {
        Some(config) => config.context("failed to read deployment dependency set")?,
        None => chain_ids
            .first()
            .and_then(|chain_id| {
                kona_registry::DEPENDENCY_SETS.get(&chain_id.saturating_to::<u64>())
            })
            .cloned()
            .ok_or_else(|| anyhow!("custom chains require a dependency-set config file"))?,
    };
    let l1_config = if host.l1_config_path.is_some() {
        host.read_l1_config().context("failed to read deployment L1 config")?
    } else {
        let l1_chain_id = rollup_configs
            .values()
            .next()
            .ok_or_else(|| anyhow!("deployment rollup config set is empty"))?
            .l1_chain_id;
        kona_registry::L1_CONFIGS.get(&l1_chain_id).cloned().ok_or_else(|| {
            anyhow!("deployment L1 chain {l1_chain_id} requires an L1 config file")
        })?
    };
    let configs = ChainConfigs { dependency_set, rollup_configs, l1_config };
    configs.validate(chain_ids)?;
    Ok(configs)
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

/// Builds span-shaped super-range inputs from consecutive supernode responses.
///
/// `responses[i]` must be the verified `superroot_atTimestamp` response for timestamp
/// `span.start - 1 + i`, so the slice covers `span.start - 1..=span.end` and holds
/// `span.end - span.start + 2` entries. The pre-span response supplies the agreed
/// super-root proof; every span response contributes one claimed transition per chain.
pub fn synthesize_execution(
    span: TimestampSpan,
    l1_head: B256,
    l1_head_number: u64,
    responses: &[SuperRootAtTimestampResponse],
) -> Result<SynthesizedExecution> {
    span.validate()?;
    let first_timestamp = span.start.checked_sub(1).ok_or_else(|| {
        anyhow!("span starting at timestamp 0 has no previous super-root timestamp")
    })?;
    let timestamp_count = usize::try_from(span.end - span.start)
        .ok()
        .and_then(|count| count.checked_add(1))
        .ok_or_else(|| anyhow!("span {}..={} has too many timestamps", span.start, span.end))?;
    ensure!(
        responses.len() == timestamp_count + 1,
        "span {}..={} requires {} responses, got {}",
        span.start,
        span.end,
        timestamp_count + 1,
        responses.len(),
    );

    let chain_ids = response_chain_ids(&responses[timestamp_count])?;

    let mut proofs = Vec::with_capacity(responses.len());
    let mut super_roots = Vec::with_capacity(responses.len());
    let mut preloaded_preimages = Vec::new();
    for (offset, response) in responses.iter().enumerate() {
        let expected_timestamp = first_timestamp + offset as u64;
        let data = response.data.as_ref().ok_or_else(|| {
            anyhow!("superroot_atTimestamp response for timestamp {expected_timestamp} has no data")
        })?;
        ensure!(
            data.super_v1.timestamp == expected_timestamp,
            "super-root timestamp {} does not match expected {expected_timestamp}",
            data.super_v1.timestamp,
        );
        ensure_chain_ids_match_super(
            &chain_ids,
            &data.super_v1,
            &format!("timestamp {expected_timestamp}"),
        )?;

        let proof = proof_from_super_v1(&data.super_v1)?;
        let super_root = hash_super_root_proof(&proof)?;
        ensure!(
            super_root == data.super_root,
            "super-root proof for timestamp {expected_timestamp} hashes to {super_root}, response reports {}",
            data.super_root,
        );
        preload_super_root_proof(&mut preloaded_preimages, &proof)?;
        proofs.push(proof);
        super_roots.push(super_root);
    }

    let mut claimed_transitions =
        Vec::with_capacity(timestamp_count.saturating_mul(chain_ids.len()));
    for (offset, response) in responses[1..].iter().enumerate() {
        let timestamp = span.start + offset as u64;
        let label = format!("timestamp {timestamp}");
        for &chain_id in &chain_ids {
            let claimed_output = output_entry(response, chain_id, &label)?;
            ensure_required_l1_within_head(claimed_output.required_l1, l1_head_number, chain_id)?;
            let (claimed_root, claimed_preimage) =
                checked_output_preimage(claimed_output, chain_id, &label)?;
            preloaded_preimages.push((PreimageKey::new_keccak256(*claimed_root), claimed_preimage));

            claimed_transitions.push(SuperRangeTransition {
                timestamp,
                optimistic_block: SuperOptimisticBlock {
                    chain_id: U256::from(chain_id),
                    block_hash: claimed_output
                        .output
                        .as_ref()
                        .expect("checked output exists")
                        .block_hash,
                    output_root: claimed_root,
                },
            });
        }
    }

    // The pre-span response holds no claimed output, so its outputs are checked here, and those
    // that the pre-span super root kept are preloaded.
    let previous_label = format!("timestamp {first_timestamp}");
    for agreed in &proofs[0].super_root.output_roots {
        let previous = output_entry(&responses[0], agreed.chain_id, &previous_label)?;
        ensure_required_l1_within_head(previous.required_l1, l1_head_number, agreed.chain_id)?;
        let (previous_root, previous_preimage) =
            checked_output_preimage(previous, agreed.chain_id, &previous_label)?;
        if previous_root == agreed.output_root {
            preloaded_preimages
                .push((PreimageKey::new_keccak256(*previous_root), previous_preimage));
        }
    }

    // Every pre-transition super root of the span is an agreed pre-state, whose outputs the range
    // and the consolidation read. Where it replaced a chain's optimistic block, the response at
    // that timestamp reports the denied block, so the agreed output is left for the host to read.
    let mut replaced_agreed_outputs = Vec::new();
    for (offset, (response, proof)) in responses.iter().zip(&proofs[..timestamp_count]).enumerate()
    {
        let label = format!("timestamp {}", first_timestamp + offset as u64);
        let mut outputs = Vec::new();
        for agreed in &proof.super_root.output_roots {
            if output_entry(response, agreed.chain_id, &label)?.output_root != agreed.output_root {
                outputs.push(agreed.clone());
            }
        }
        if !outputs.is_empty() {
            replaced_agreed_outputs.push(ReplacedAgreedOutputs {
                agreed_pre_state: encode_super_root_proof(proof)?,
                outputs,
            });
        }
    }

    let range_inputs = SuperRangeInputs {
        span,
        l1_head,
        chain_ids: chain_ids.iter().copied().map(U256::from).collect(),
        previous_super_root_proofs: proofs[..timestamp_count].to_vec(),
        claimed_transitions,
    };
    range_inputs.validate()?;

    let transitions = range_inputs
        .claimed_transitions
        .chunks_exact(chain_ids.len())
        .zip(proofs[1..].iter().cloned())
        .map(|(chunk, claimed_super_root_proof)| SuperConsolidationTransitionInput {
            optimistic_blocks: chunk.iter().map(|transition| transition.optimistic_block).collect(),
            claimed_super_root_proof,
        })
        .collect();
    let consolidation_inputs =
        SuperConsolidationInputs { span, previous_super_root: super_roots[0], transitions };
    consolidation_inputs.validate()?;

    Ok(SynthesizedExecution {
        range_inputs,
        consolidation_inputs,
        preloaded_preimages,
        replaced_agreed_outputs,
        current_super_root: super_roots[timestamp_count],
        previous_super_root_proof_bytes: encode_super_root_proof(&proofs[0])?,
    })
}

/// Encodes super-range SP1 stdin as typed inputs followed by rkyv witness bytes.
pub fn build_super_range_stdin(
    inputs: &SuperRangeInputs,
    witness: DefaultWitnessData,
) -> Result<SP1Stdin> {
    build_super_interop_stdin(SuperInteropInputs::Range(inputs.clone()), witness)
}

/// Compressed batch proofs of the `log-commitment` program, with its verifying key.
#[derive(Clone)]
pub struct LogCommitmentProofs {
    /// The `log-commitment` program's verifying key.
    pub vk: SP1VerifyingKey,
    /// The batch proofs, in the order of the batches they commit.
    pub proofs: Vec<SP1ProofWithPublicValues>,
}

impl Debug for LogCommitmentProofs {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        f.debug_struct("LogCommitmentProofs").field("proofs", &self.proofs.len()).finish()
    }
}

impl LogCommitmentProofs {
    /// The public values of each proof: the batches of [`LogCommitmentInputs`].
    pub fn batches(&self) -> Vec<Vec<u8>> {
        self.proofs.iter().map(|proof| proof.public_values.to_vec()).collect()
    }
}

/// Encodes consolidation-mode SP1 stdin: typed inputs, rkyv witness bytes, and the encoded
/// `log_commitments`. `proofs` go to the proof stream (see [`write_log_commitment_proofs`]).
/// Without `proofs`, only execution with the deferred-proof check off can use batches.
pub fn build_super_consolidation_stdin(
    inputs: &SuperConsolidationInputs,
    witness: DefaultWitnessData,
    log_commitments: &LogCommitmentInputs,
    proofs: Option<&LogCommitmentProofs>,
) -> Result<SP1Stdin> {
    log_commitments.validate()?;
    let mut stdin =
        build_super_interop_stdin(SuperInteropInputs::Consolidation(inputs.clone()), witness)?;
    stdin.write_vec(log_commitments.encode());
    if let Some(proofs) = proofs {
        write_log_commitment_proofs(
            &mut stdin,
            &proofs.vk,
            &log_commitments.batches,
            &proofs.proofs,
        )?;
    }
    Ok(stdin)
}

/// Adds `proofs` to the proof stream of a consolidation `stdin` that carries `batches`, with the
/// `log-commitment` program's `vk`. Each proof must be compressed and commit the batch at its
/// position.
pub fn write_log_commitment_proofs(
    stdin: &mut SP1Stdin,
    vk: &SP1VerifyingKey,
    batches: &[impl AsRef<[u8]>],
    proofs: &[impl std::borrow::Borrow<SP1ProofWithPublicValues>],
) -> Result<()> {
    ensure!(
        proofs.len() == batches.len(),
        "{} log-commitment proofs for {} batches",
        proofs.len(),
        batches.len(),
    );
    for (index, (proof, batch)) in proofs.iter().zip(batches).enumerate() {
        let proof = proof.borrow();
        ensure!(
            proof.public_values.as_slice() == batch.as_ref(),
            "log-commitment proof {index} does not commit its batch",
        );
        let SP1Proof::Compressed(recursion_proof) = &proof.proof else {
            bail!("log-commitment proof {index} is not compressed");
        };
        stdin.write_proof((**recursion_proof).clone(), vk.vk.clone());
    }
    Ok(())
}

/// Encodes log-commitment SP1 stdin for `blocks`. Fails if receipts do not hash to their root.
pub fn build_log_commitment_stdin_from_receipts(blocks: &[BlockReceipts]) -> Result<SP1Stdin> {
    let receipts_roots: Vec<B256> = blocks.iter().map(|block| block.receipts_root).collect();
    build_log_commitment_stdin(&receipts_roots, log_commitment_witness(blocks)?)
}

/// Encodes log-commitment SP1 stdin as the receipts roots followed by rkyv witness bytes.
pub fn build_log_commitment_stdin(
    receipts_roots: &[B256],
    witness: DefaultWitnessData,
) -> Result<SP1Stdin> {
    let mut stdin = SP1Stdin::new();
    stdin.write(&receipts_roots);
    stdin.write_slice(&archive_witness(&witness)?);
    Ok(stdin)
}

/// The public values that the `log-commitment` program commits for `stdin` from
/// [`build_log_commitment_stdin`], by the program's own function over the same stdin items.
pub fn expected_batch_public_values(stdin: &SP1Stdin) -> Vec<u8> {
    let receipts_roots: Vec<B256> = SP1Stdin::from(&stdin.buffer[0]).read();
    log_commitment_public_values(&receipts_roots, &stdin.buffer[1])
}

fn build_super_interop_stdin(
    inputs: SuperInteropInputs,
    witness: DefaultWitnessData,
) -> Result<SP1Stdin> {
    let mut stdin = SP1Stdin::new();
    stdin.write(&inputs);
    stdin.write_slice(&archive_witness(&witness)?);
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

/// Executes `elf` on `stdin`. Without `deferred_proof_verification`, the guest's
/// `verify_sp1_proof` calls pass with no proof in the stdin.
async fn execute_sp1(
    mode: &str,
    stdin: SP1Stdin,
    elf: Arc<[u8]>,
    report: bool,
    deferred_proof_verification: bool,
) -> Result<SP1PublicValues> {
    let stdin_bytes: usize = stdin.buffer.iter().map(Vec::len).sum();
    let client = ProverClient::builder().cpu().build().await;
    let (public_values, execution_report) = client
        .execute(Elf::Dynamic(elf), stdin)
        .deferred_proof_verification(deferred_proof_verification)
        .calculate_gas(true)
        .await?;
    ensure!(
        execution_report.exit_code == 0,
        "{mode} guest exited with code {}",
        execution_report.exit_code
    );
    if report {
        let pgu = execution_report.gas().expect("calculate_gas(true) reports PGU");
        println!(
            "{mode}: pgu={pgu} instructions={} stdin_bytes={stdin_bytes}",
            execution_report.total_instruction_count(),
        );
        // SP1 fills the cycle trackers only when built with `sp1-sdk/profiling`.
        let trackers: BTreeMap<_, _> = execution_report.cycle_tracker.iter().collect();
        for (label, cycles) in trackers {
            let invocations =
                execution_report.invocation_tracker.get(label).copied().unwrap_or_default();
            println!("{mode} {label}: cycles={cycles} invocations={invocations}");
        }
    }
    Ok(public_values)
}

async fn execute_sp1_range(
    inputs: &SuperRangeInputs,
    witness: DefaultWitnessData,
    super_range_elf: Arc<[u8]>,
    report: bool,
) -> Result<SuperRangeOutputs> {
    let stdin = build_super_range_stdin(inputs, witness)?;
    let mut public_values = execute_sp1("range", stdin, super_range_elf, report, true).await?;
    decode_super_range_public_values(&mut public_values)
}

async fn execute_sp1_consolidation(
    inputs: &SuperConsolidationInputs,
    witness: DefaultWitnessData,
    log_commitments: &LogCommitmentInputs,
    super_range_elf: Arc<[u8]>,
    report: bool,
) -> Result<SuperConsolidationOutputs> {
    let stdin = build_super_consolidation_stdin(inputs, witness, log_commitments, None)?;
    // The stdin carries the batches' public values without their proofs.
    let mut public_values =
        execute_sp1("consolidation", stdin, super_range_elf, report, false).await?;
    decode_super_consolidation_public_values(&mut public_values)
}

async fn replay_range(
    native_core: bool,
    report: bool,
    inputs: &SuperRangeInputs,
    witness: DefaultWitnessData,
    super_range_elf: Option<Arc<[u8]>>,
    configs: Option<&ChainConfigs>,
) -> Result<SuperRangeOutputs> {
    if !native_core {
        return execute_sp1_range(
            inputs,
            witness,
            super_range_elf.expect("super-range ELF is loaded for SP1 execution"),
            report,
        )
        .await;
    }
    let (oracle, beacon) = witness.get_oracle_and_blob_provider().await?;
    build_range_outputs(inputs.clone(), oracle, beacon, configs).await
}

async fn replay_consolidation(
    native_core: bool,
    report: bool,
    inputs: &SuperConsolidationInputs,
    witness: DefaultWitnessData,
    log_commitments: &LogCommitmentInputs,
    super_range_elf: Option<Arc<[u8]>>,
    configs: Option<&ChainConfigs>,
) -> Result<SuperConsolidationOutputs> {
    if !native_core {
        return execute_sp1_consolidation(
            inputs,
            witness,
            log_commitments,
            super_range_elf.expect("super-range ELF is loaded for SP1 execution"),
            report,
        )
        .await;
    }
    let (oracle, _) = witness.get_oracle_and_blob_provider().await?;
    // The guest's source; the native replay has no proof to verify.
    let log_commitments = VerifiedLogCommitments::new(log_commitments.clone(), &[0; 8], |_, _| {});
    build_consolidation_outputs(inputs.clone(), oracle, configs, Arc::new(log_commitments)).await
}

/// Flips a bit in the first claimed transition's optimistic output root.
///
/// Only the replayed inputs are corrupted, so the collected witness has no preimage for the
/// tampered root. The SP1 guest aborts when it reads that preimage and the executor reports the
/// claim invalid. Native-core replay panics on the same read instead of returning an error, so
/// combining the corruption with native-core replay is unsupported: keep invalid-claim tests on the
/// SP1 execute path.
fn corrupt_range_claim(inputs: &SuperRangeInputs) -> Result<SuperRangeInputs> {
    let mut corrupted = inputs.clone();
    let transition = corrupted
        .claimed_transitions
        .first_mut()
        .ok_or_else(|| anyhow!("no claimed transition to corrupt"))?;
    transition.optimistic_block.output_root.0[0] ^= 0x01;
    Ok(corrupted)
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

/// A consolidation witness with what the native run that collected it computed.
#[derive(Debug)]
pub struct CollectedConsolidation {
    /// The witness.
    pub witness: DefaultWitnessData,
    /// The natively computed consolidation outputs.
    pub outputs: SuperConsolidationOutputs,
    /// The openings of the logs that the entry table served, for [`LogCommitmentInputs`].
    pub openings: Vec<LogOpening>,
    /// The blocks below the local safe heads that had no entry, so the witness holds their
    /// receipts.
    pub missing: Vec<MissingLogCommitment>,
    /// The receipts roots whose entries the run read, so the stdin must carry their batches.
    pub served: Vec<B256>,
    /// The bytes that the run read from the oracle, with repeats.
    pub read_bytes: u64,
    /// The siblings of the openings that the run's log checks verify.
    pub log_check_siblings: u64,
}

/// The header and receipts of an optimistic block.
#[derive(Debug, Clone)]
pub struct OptimisticBlockReceipts {
    /// The block's header, checked against the requested hash.
    pub header: Header,
    /// The block's receipts, read from the header's receipts root.
    pub receipts: Vec<OpReceiptEnvelope>,
}

/// The header and receipts of `block_hash`, the optimistic block of `chain_id` built on the
/// canonical block `parent_hash`, read through `host` the way the consolidation reads its heads.
/// When the block is not canonical, kona-host re-derives it from L1 and re-executes it, so no L2
/// node needs to hold it.
pub async fn optimistic_block_receipts(
    host: InteropHost,
    chain_id: u64,
    parent_hash: B256,
    block_hash: B256,
) -> Result<OptimisticBlockReceipts> {
    let (_, block) = collect_witness(&[], host, |oracle, _beacon| async move {
        HintType::L2BlockData
            .with_data(&[parent_hash.as_slice(), block_hash.as_slice(), &chain_id.to_be_bytes()])
            .send(oracle.as_ref())
            .await?;
        let header = fetch_l2_header(oracle.as_ref(), block_hash, chain_id).await?;
        let receipts =
            hinted_receipts(oracle.as_ref(), chain_id, block_hash, header.receipts_root).await?;
        Ok(OptimisticBlockReceipts { header, receipts })
    })
    .await?;
    Ok(block)
}

/// The output-root preimages of `outputs`, agreed outputs that a pre-transition super root
/// replaced (see [`SynthesizedExecution::replaced_agreed_outputs`]), read through `host` with
/// kona-host's `L2OutputRoot` hint. The hint computes the output of the L2 node's canonical block
/// at the timestamp of the host's agreed pre-state. kona-host retries the hint until the node
/// serves the block.
async fn replaced_agreed_output_preimages(
    host: InteropHost,
    outputs: &[SuperOutputRoot],
) -> Result<Vec<(PreimageKey, Vec<u8>)>> {
    let (_, preimages) = collect_witness(&[], host, |oracle, _beacon| async move {
        let mut preimages = Vec::with_capacity(outputs.len());
        for output in outputs {
            HintType::L2OutputRoot
                .with_data(&[output.output_root.as_slice(), output.chain_id.to_be_bytes().as_ref()])
                .send(oracle.as_ref())
                .await?;
            let key = PreimageKey::new_keccak256(*output.output_root);
            let preimage = oracle.get(key).await.map_err(|err| {
                anyhow!("agreed output {} of chain {}: {err}", output.output_root, output.chain_id)
            })?;
            preimages.push((key, preimage));
        }
        Ok(preimages)
    })
    .await?;
    Ok(preimages)
}

struct AbortOnDrop(tokio::task::AbortHandle);

impl Drop for AbortOnDrop {
    fn drop(&mut self) {
        self.0.abort();
    }
}

/// Collects a replayable witness by running `run_native` against a hint-serving host.
///
/// Preloaded preimages seed the oracle before host hinting; the returned witness bundles
/// every preimage and blob the native pass touched, alongside the native outputs.
pub async fn collect_witness<T, F, Fut>(
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
    let _server_guard = AbortOnDrop(server_task.abort_handle());
    let native_outputs = run_native(oracle, beacon).await;
    server_task.abort();
    let server_result = server_task.await;
    let native_outputs = native_outputs?;
    match server_result {
        Ok(Ok(())) => {}
        Ok(Err(error)) => return Err(error.into()),
        Err(error) if error.is_cancelled() => {}
        Err(error) => return Err(error.into()),
    }

    let witness = DefaultWitnessData::from_parts(
        preimage_witness_store
            .lock()
            .map_err(|_| anyhow!("failed to acquire preimage witness store lock"))?
            .clone(),
        blob_data.lock().map_err(|_| anyhow!("failed to acquire blob data lock"))?.clone(),
    );
    Ok((witness, native_outputs))
}

/// Builds the `InteropHost` for one span execution.
///
/// Deployment-scoped endpoints come from `inputs`; the remaining arguments are the per-span
/// facts: the trusted L1 head, the encoded agreed pre-state super-root proof, the claimed
/// post-state super root, and the claimed (span-end) timestamp. With `data_dir`, the host keeps
/// its preimages in a `RocksDB` store there instead of in memory; the caller removes the
/// directory after the collection.
pub fn build_interop_host(
    inputs: &HostInputs,
    l1_head: B256,
    agreed_pre_state: &[u8],
    claimed_post_state: B256,
    claimed_timestamp: u64,
    data_dir: Option<&Path>,
) -> Result<InteropHost> {
    ensure!(!inputs.l2_node_addresses.is_empty(), "at least one L2 node address is required",);
    Ok(InteropHost {
        l1_head,
        agreed_l2_pre_state: Bytes::from(agreed_pre_state.to_vec()),
        claimed_l2_post_state: claimed_post_state,
        claimed_l2_timestamp: claimed_timestamp,
        l2_node_addresses: Some(inputs.l2_node_addresses.clone()),
        l1_node_address: Some(inputs.l1_node_address.clone()),
        l1_beacon_address: Some(inputs.l1_beacon_address.clone()),
        data_dir: data_dir.map(Path::to_path_buf),
        data_format: DataFormat::Rocksdb,
        native: false,
        server: true,
        rollup_config_paths: inputs.rollup_config_paths.clone(),
        l1_config_path: inputs.l1_config_path.clone(),
        dependency_set_path: inputs.dependency_set_path.clone(),
    })
}

/// Fetches the block number for `l1_head` from an L1 execution endpoint.
pub async fn fetch_l1_head_number(l1_node_address: &str, l1_head: B256) -> Result<u64> {
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

/// Converts a decoded `SuperV1` into the guest-side `SuperRootProof` type.
pub fn proof_from_super_v1(super_v1: &SuperV1) -> Result<SuperRootProof> {
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

/// Preimage oracle that serves seeded preimages before falling back to the wrapped host oracle.
#[derive(Clone)]
pub struct SeededOracle<P> {
    inner: P,
    seeds: Arc<PreimageStore>,
}

/// The concrete seeded oracle used when collecting witnesses over a native host channel.
pub type SeededOracleBase =
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
    /// Highest cross-safe L2 timestamp across the dependency set (min over
    /// chains). Required; a serving node that predates the field fails
    /// deserialization loudly rather than defaulting to zero.
    #[serde(rename = "safe_timestamp", deserialize_with = "deserialize_u64_or_hex")]
    pub current_safe_timestamp: u64,
    /// Highest local-safe L2 timestamp across the dependency set.
    #[serde(rename = "local_safe_timestamp", deserialize_with = "deserialize_u64_or_hex")]
    pub current_local_safe_timestamp: u64,
    /// Highest finalized L2 timestamp across the dependency set.
    #[serde(rename = "finalized_timestamp", deserialize_with = "deserialize_u64_or_hex")]
    pub current_finalized_timestamp: u64,
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
    use alloy_consensus::{Receipt, ReceiptWithBloom, proofs::ordered_trie_root_with_encoder};
    use alloy_eips::eip2718::Encodable2718;
    use alloy_primitives::{Address, Log};
    use httpmock::{
        HttpMockRequest, HttpMockResponse, MockServer,
        prelude::{GET, POST},
    };
    use kona_preimage::errors::PreimageOracleError;
    use serde_json::{Value, json};

    #[test]
    fn replay_witness_over_the_limit_is_not_a_rejected_claim() {
        let too_large =
            anyhow::Error::new(WitnessTooLarge { bound: 2, limit: 1 }).context("replay");
        let err = replay_failure("range", too_large).expect_err("an oversized witness is an error");
        assert!(is_witness_too_large(&err));

        let verdict = replay_failure("range", anyhow!("guest panicked")).expect("a verdict");
        assert_eq!(verdict, Verdict::Invalid);
    }

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
        response_with_block_hash(timestamp, chain_ids, required_l1, |chain_id| b256(chain_id as u8))
    }

    fn response_with_block_hash(
        timestamp: u64,
        chain_ids: &[u64],
        required_l1: u64,
        block_hash: impl Fn(u64) -> B256,
    ) -> SuperRootAtTimestampResponse {
        let mut optimistic_at_timestamp = BTreeMap::new();
        let mut chains = Vec::new();
        for &chain_id in chain_ids {
            let entry = output_entry(block_hash(chain_id), required_l1);
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
            current_safe_timestamp: timestamp,
            current_local_safe_timestamp: timestamp,
            current_finalized_timestamp: timestamp,
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
    fn deployment_configs_use_registry_defaults_and_file_overrides() {
        for case in ["registry", "rollup_override", "missing_custom_depset"] {
            let dir = tempfile::tempdir().unwrap();
            let mut expected = ChainConfigs::from_registry(&[U256::from(10)]).unwrap();
            let rollup_config_paths = match case {
                "registry" => None,
                "rollup_override" => {
                    expected.rollup_configs.get_mut(&10).unwrap().l1_chain_id = 11155111;
                    expected.l1_config = kona_registry::L1_CONFIGS[&11155111].clone();
                    let path = dir.path().join("rollup.json");
                    std::fs::write(
                        &path,
                        serde_json::to_vec(&expected.rollup_configs[&10]).unwrap(),
                    )
                    .unwrap();
                    Some(vec![path])
                }
                "missing_custom_depset" => {
                    let mut rollup = expected.rollup_configs.remove(&10).unwrap();
                    rollup.l2_chain_id = u64::MAX.into();
                    let path = dir.path().join("rollup.json");
                    std::fs::write(&path, serde_json::to_vec(&rollup).unwrap()).unwrap();
                    expected.rollup_configs.insert(u64::MAX, rollup);
                    Some(vec![path])
                }
                _ => unreachable!(),
            };
            let chain_ids =
                expected.rollup_configs.keys().copied().map(U256::from).collect::<Vec<_>>();
            let host = build_interop_host(
                &HostInputs {
                    l1_node_address: "http://127.0.0.1:1".into(),
                    l1_beacon_address: "http://127.0.0.1:1".into(),
                    l2_node_addresses: vec!["http://127.0.0.1:1".into()],
                    rollup_config_paths,
                    l1_config_path: None,
                    dependency_set_path: None,
                },
                b256(0xaa),
                &[],
                b256(0xbb),
                101,
                None,
            )
            .unwrap();
            let result = deployment_chain_configs(&host, &chain_ids);
            if case == "missing_custom_depset" {
                assert_eq!(
                    result.unwrap_err().to_string(),
                    "custom chains require a dependency-set config file",
                );
            } else {
                assert_eq!(result.unwrap(), expected, "case: {case}");
            }
        }
    }

    #[tokio::test]
    async fn native_replay_uses_resolved_deployment_configs_without_elf_or_local_config_keys() {
        let dir = tempfile::tempdir().unwrap();
        let mut configs = ChainConfigs::from_registry(&[U256::from(10)]).unwrap();
        configs.l1_config = Default::default();
        let mut rollup = configs.rollup_configs.remove(&10).unwrap();
        rollup.l2_chain_id = u64::MAX.into();
        configs.rollup_configs.insert(u64::MAX, rollup.clone());
        let dependency = configs.dependency_set.dependencies.remove(&10).unwrap();
        configs.dependency_set.dependencies.insert(u64::MAX, dependency);
        configs.dependency_set.override_message_expiry_window = Some(123.try_into().unwrap());
        let rollup_path = dir.path().join("rollup.json");
        let dependency_path = dir.path().join("depset.json");
        let l1_path = dir.path().join("l1.json");
        for (path, bytes) in [
            (&rollup_path, serde_json::to_vec(&rollup).unwrap()),
            (&dependency_path, serde_json::to_vec(&configs.dependency_set).unwrap()),
            (&l1_path, serde_json::to_vec(&configs.l1_config).unwrap()),
        ] {
            std::fs::write(path, bytes).unwrap();
        }
        let host = build_interop_host(
            &HostInputs {
                l1_node_address: "http://127.0.0.1:1".into(),
                l1_beacon_address: "http://127.0.0.1:1".into(),
                l2_node_addresses: vec!["http://127.0.0.1:1".into()],
                rollup_config_paths: Some(vec![rollup_path.clone()]),
                l1_config_path: Some(l1_path),
                dependency_set_path: Some(dependency_path),
            },
            b256(0xaa),
            &[],
            b256(0xbb),
            101,
            None,
        )
        .unwrap();
        let loaded = deployment_chain_configs(&host, &[U256::from(u64::MAX)]).unwrap();
        assert_eq!(loaded, configs);
        std::fs::write(rollup_path, "invalid JSON after config resolution").unwrap();

        let header = alloy_consensus::Header { number: 3, timestamp: 100, ..Default::default() };
        let block_hash = header.hash_slow();
        let responses = [100, 101]
            .map(|timestamp| response_with_block_hash(timestamp, &[u64::MAX], 7, |_| block_hash));
        let synthesized =
            synthesize_execution(TimestampSpan::new(101, 101).unwrap(), b256(0xaa), 8, &responses)
                .unwrap();
        let mut preimages = PreimageStore::default();
        for (key, bytes) in &synthesized.preloaded_preimages {
            preimages.save_preimage(*key, bytes.clone()).unwrap();
        }
        preimages
            .save_preimage(PreimageKey::new_keccak256(*block_hash), alloy_rlp::encode(&header))
            .unwrap();
        let witness = DefaultWitnessData::from_parts(preimages, BlobData::default());
        let outputs = replay_range(
            true,
            false,
            &synthesized.range_inputs,
            witness.clone(),
            None,
            Some(&loaded),
        )
        .await
        .unwrap();
        assert_eq!(outputs.transitions, synthesized.range_inputs.claimed_transitions);
        assert_eq!(outputs.span, synthesized.range_inputs.span);
        assert_eq!(outputs.l1_head, synthesized.range_inputs.l1_head);
        let err = replay_range(true, false, &synthesized.range_inputs, witness, None, None)
            .await
            .unwrap_err();
        assert!(err.to_string().contains("no embedded dependency set"), "unexpected error: {err}");
    }

    #[tokio::test(flavor = "multi_thread")]
    async fn replaced_two_second_head_is_retained_during_noop_replay() {
        let configs = ChainConfigs::from_registry(&[U256::from(10)]).unwrap();
        let rollup = &configs.rollup_configs[&10];
        assert_eq!(rollup.block_time, 2);
        let timestamp = rollup.genesis.l2_time + 20;
        let server = mock_nodes().await;
        let (replacement_hash, replacement_preimage, reads) =
            mock_canonical_output(&server, timestamp).await;
        let replacement = Header {
            number: rollup.block_number_from_timestamp(timestamp),
            timestamp,
            state_root: b256(0x11),
            receipts_root: alloy_consensus::EMPTY_ROOT_HASH,
            ..Default::default()
        };
        assert_eq!(replacement.hash_slow(), replacement_hash);
        let original = Header { extra_data: Bytes::from_static(b"invalid"), ..replacement.clone() };
        let original_hash = original.hash_slow();
        let mut responses = [timestamp, timestamp + 1]
            .map(|at| response_with_block_hash(at, &[10], 8, |_| replacement_hash));
        for response in &mut responses {
            response
                .optimistic_at_timestamp
                .insert(ChainId(U256::from(10)), output_entry(original_hash, 8));
        }
        let mut synthesized = synthesize_execution(
            TimestampSpan::new(timestamp + 1, timestamp + 1).unwrap(),
            b256(0xaa),
            8,
            &responses,
        )
        .unwrap();
        let host = build_interop_host(
            &mock_host_inputs(&server),
            synthesized.range_inputs.l1_head,
            &synthesized.previous_super_root_proof_bytes,
            synthesized.current_super_root,
            timestamp + 1,
            None,
        )
        .unwrap();
        for header in [&original, &replacement] {
            synthesized
                .preloaded_preimages
                .push((PreimageKey::new_keccak256(*header.hash_slow()), alloy_rlp::encode(header)));
        }
        synthesized
            .preloaded_preimages
            .push((PreimageKey::new_keccak256(*alloy_consensus::EMPTY_ROOT_HASH), vec![0x80]));
        let (witness, discovered) = synthesized
            .collect_range_witness(host.clone(), Some(&configs))
            .await
            .expect("native witness collection must retain the replacement");
        let range = replay_range(
            true,
            false,
            &synthesized.range_inputs,
            witness.clone(),
            None,
            Some(&configs),
        )
        .await
        .expect("the no-op after replacement must replay");
        let retained = SuperOptimisticBlock {
            chain_id: U256::from(10),
            block_hash: replacement_hash,
            output_root: keccak256(&replacement_preimage),
        };
        assert_eq!(range, discovered);
        assert_eq!(range.transitions[0].optimistic_block, retained);
        assert_eq!(
            synthesized.consolidation_inputs.transitions[0].optimistic_blocks,
            vec![retained]
        );
        let mut in_span_responses = vec![response(timestamp - 1, &[10], 8)];
        in_span_responses.extend(responses);
        let mut in_span = synthesize_execution(
            TimestampSpan::new(timestamp, timestamp + 1).unwrap(),
            b256(0xaa),
            8,
            &in_span_responses,
        )
        .unwrap();
        in_span.prepare_inputs(&host).await.unwrap();
        let expected = vec![
            SuperOptimisticBlock {
                chain_id: U256::from(10),
                block_hash: original_hash,
                output_root: output_entry(original_hash, 8).output_root,
            },
            retained,
        ];
        assert_eq!(
            in_span
                .range_inputs
                .claimed_transitions
                .iter()
                .map(|transition| transition.optimistic_block)
                .collect::<Vec<_>>(),
            expected,
        );
        assert_eq!(
            in_span
                .consolidation_inputs
                .transitions
                .iter()
                .flat_map(|transition| transition.optimistic_blocks.iter().copied())
                .collect::<Vec<_>>(),
            expected,
        );
        let collected = synthesized
            .collect_consolidation_witness(host, Some(&configs), &[], None)
            .await
            .expect("native consolidation collection must retain the replacement");
        let consolidation = replay_consolidation(
            true,
            false,
            &synthesized.consolidation_inputs,
            collected.witness,
            &Default::default(),
            None,
            Some(&configs),
        )
        .await
        .expect("consolidation must retain the replacement too");
        assert_eq!(consolidation, collected.outputs);
        assert_eq!(consolidation.transitions[0].optimistic_blocks, vec![retained]);
        assert_eq!(consolidation.transitions[0].super_root, synthesized.current_super_root);
        for read in reads {
            read.assert_calls_async(2).await;
        }
    }

    #[test]
    fn synthesize_execution_orders_chains_and_preloads_outputs() {
        let previous = response(100, &[10, 20], 7);
        let current = response(101, &[10, 20], 8);

        let synthesized = synthesize_execution(
            TimestampSpan::new(101, 101).unwrap(),
            b256(0xaa),
            8,
            &[previous, current],
        )
        .expect("synthesizes");

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

        let synthesized = synthesize_execution(
            TimestampSpan::new(101, 101).unwrap(),
            b256(0xaa),
            8,
            &[previous, current.clone()],
        )
        .expect("synthesizes");

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

    /// Pins that every agreed output that a pre-transition super root of the span replaced is
    /// reported with that super root, for a host agreeing on it to read, and not preloaded: the
    /// response's optimistic output there is the denied block's, and on a chain with a block at
    /// every timestamp no claimed output is the replacement either.
    #[test]
    fn synthesize_execution_reports_replaced_agreed_outputs_by_timestamp() {
        let mut responses: Vec<_> = (100u64..=102)
            .map(|timestamp| {
                response_with_block_hash(timestamp, &[10, 20], 8, |chain_id| {
                    b256(chain_id as u8 ^ timestamp as u8)
                })
            })
            .collect();
        // Chain 20's block at the pre-span timestamp and chain 10's inside the span were denied.
        let denied_pre_span = output_entry(b256(0xf0), 8);
        responses[0]
            .optimistic_at_timestamp
            .insert(ChainId(U256::from(20)), denied_pre_span.clone());
        responses[1]
            .optimistic_at_timestamp
            .insert(ChainId(U256::from(10)), output_entry(b256(0xf1), 8));

        let synthesized =
            synthesize_execution(TimestampSpan::new(101, 102).unwrap(), b256(0xaa), 8, &responses)
                .expect("synthesizes");

        let replaced = |offset: usize, chain_id: u64, index: usize| {
            let super_v1 = &responses[offset].data.as_ref().unwrap().super_v1;
            let proof = proof_from_super_v1(super_v1).unwrap();
            ReplacedAgreedOutputs {
                agreed_pre_state: encode_super_root_proof(&proof).unwrap(),
                outputs: vec![SuperOutputRoot {
                    chain_id,
                    output_root: super_v1.chains[index].output,
                }],
            }
        };
        let expected = vec![replaced(0, 20, 1), replaced(1, 10, 0)];
        assert_eq!(synthesized.replaced_agreed_outputs, expected);
        let not_preloaded = expected
            .iter()
            .flat_map(|replaced| &replaced.outputs)
            .map(|output| output.output_root)
            .chain([denied_pre_span.output_root])
            .map(|root| PreimageKey::new_keccak256(*root));
        for key in not_preloaded {
            assert!(synthesized.preloaded_preimages.iter().all(|(preloaded, _)| *preloaded != key));
        }
    }

    /// Pins that each replaced agreed output of a span is read through kona-host from the L2
    /// node's canonical block at its own timestamp, by number, and seeded as the root's preimage.
    #[tokio::test]
    async fn seeded_preimages_read_each_replaced_agreed_output_at_its_timestamp() {
        let rollup = &kona_registry::ROLLUP_CONFIGS[&10];
        let first = rollup.genesis.l2_time + 10 * rollup.block_time;
        let second = first + rollup.block_time;
        let server = mock_nodes().await;
        let (first_hash, first_preimage, first_reads) = mock_canonical_output(&server, first).await;
        let (second_hash, second_preimage, second_reads) =
            mock_canonical_output(&server, second).await;

        // Both replacements are inside the span, so a host agreeing on the pre-span super root,
        // or one host for both, would ask for a block the node does not serve.
        let mut responses: Vec<_> = (first - 1..=second + 1)
            .map(|at| {
                response_with_block_hash(at, &[10], 8, |_| match at {
                    _ if at == first => first_hash,
                    _ if at == second => second_hash,
                    _ => b256(at as u8),
                })
            })
            .collect();
        for at in [first, second] {
            responses[(at - (first - 1)) as usize]
                .optimistic_at_timestamp
                .insert(ChainId(U256::from(10)), output_entry(b256(0xfe), 8));
        }
        let synthesized = synthesize_execution(
            TimestampSpan::new(first, second + 1).unwrap(),
            b256(0xaa),
            8,
            &responses,
        )
        .unwrap();
        let host = build_interop_host(
            &mock_host_inputs(&server),
            synthesized.range_inputs.l1_head,
            &synthesized.previous_super_root_proof_bytes,
            synthesized.current_super_root,
            second + 1,
            None,
        )
        .unwrap();

        // kona-host retries a hint it cannot serve, so a wrong read would hang, not fail.
        let preimages = tokio::time::timeout(
            std::time::Duration::from_secs(30),
            synthesized.seeded_preimages(&host),
        )
        .await
        .expect("the host must serve the agreed outputs")
        .unwrap();

        for preimage in [first_preimage, second_preimage] {
            let key = PreimageKey::new_keccak256(*keccak256(&preimage));
            assert!(preimages.contains(&(key, preimage)));
        }
        for read in first_reads.iter().chain(&second_reads) {
            read.assert_async().await;
        }
    }

    /// Serves the canonical block of chain 10 at `timestamp` only by number, the way kona-host's
    /// `L2OutputRoot` hint reads it. Returns the block's hash and output preimage, and the mocks
    /// of the two reads.
    async fn mock_canonical_output(
        server: &MockServer,
        timestamp: u64,
    ) -> (B256, Vec<u8>, [httpmock::Mock<'_>; 2]) {
        let number = kona_registry::ROLLUP_CONFIGS[&10].block_number_from_timestamp(timestamp);
        let header = Header {
            number,
            timestamp,
            state_root: b256(0x11),
            receipts_root: alloy_consensus::EMPTY_ROOT_HASH,
            ..Default::default()
        };
        let storage_root = b256(0x22);
        let header_read = server
            .mock_async(|when, then| {
                when.method(POST).json_body_includes(
                    json!({ "method": "debug_getRawHeader", "params": [format!("0x{number:x}")] })
                        .to_string(),
                );
                then.respond_with(rpc_result(json!(alloy_primitives::hex::encode_prefixed(
                    alloy_rlp::encode(&header)
                ))));
            })
            .await;
        let message_passer = "0x4200000000000000000000000000000000000016";
        let proof_read = server
            .mock_async(|when, then| {
                when.method(POST).json_body_includes(
                    json!({
                        "method": "eth_getProof",
                        "params": [message_passer, [], format!("0x{number:x}")],
                    })
                    .to_string(),
                );
                then.respond_with(rpc_result(json!({
                    "address": message_passer,
                    "balance": "0x0",
                    "codeHash": B256::ZERO,
                    "nonce": "0x0",
                    "storageHash": storage_root,
                    "accountProof": [],
                    "storageProof": [],
                })));
            })
            .await;
        let preimage = OutputV0 {
            state_root: header.state_root,
            message_passer_storage_root: storage_root,
            block_hash: header.hash_slow(),
        }
        .marshal();
        (header.hash_slow(), preimage, [header_read, proof_read])
    }

    /// Pins that kona-host serves the header and receipts of a canonical optimistic block by hash,
    /// once it finds the block canonical above `parent`.
    // The receipts trie walk blocks in place, which needs the multi-threaded runtime.
    #[tokio::test(flavor = "multi_thread")]
    async fn optimistic_block_receipts_read_a_canonical_block_through_the_host() {
        let log = Log::new_unchecked(Address::ZERO, vec![b256(0x33)], Bytes::from_static(b"log"));
        let receipts = vec![OpReceiptEnvelope::Eip1559(ReceiptWithBloom {
            receipt: Receipt { status: true.into(), cumulative_gas_used: 21_000, logs: vec![log] },
            logs_bloom: Default::default(),
        })];
        let raw_receipts: Vec<Bytes> =
            receipts.iter().map(|receipt| receipt.encoded_2718().into()).collect();
        let parent = Header { number: 9, ..Default::default() };
        let header = Header {
            number: 10,
            parent_hash: parent.hash_slow(),
            receipts_root: ordered_trie_root_with_encoder(&raw_receipts, |receipt, out| {
                out.extend_from_slice(receipt)
            }),
            ..Default::default()
        };
        let hash = header.hash_slow();
        let rpc_block = |header: &Header| {
            let mut block = serde_json::to_value(header).unwrap();
            block["hash"] = json!(header.hash_slow());
            block["transactions"] = json!([]);
            block["uncles"] = json!([]);
            block
        };

        let server = mock_nodes().await;
        let mut reads = Vec::new();
        for (method, param, result) in [
            ("eth_getBlockByHash", json!(parent.hash_slow()), rpc_block(&parent)),
            ("eth_getBlockByNumber", json!("0xa"), rpc_block(&header)),
            (
                "debug_getRawHeader",
                json!(hash),
                json!(alloy_primitives::hex::encode_prefixed(alloy_rlp::encode(&header))),
            ),
            ("debug_getRawReceipts", json!(hash), json!(raw_receipts)),
        ] {
            reads.push(
                server
                    .mock_async(|when, then| {
                        when.method(POST).json_body_includes(
                            json!({ "method": method, "params": [param] }).to_string(),
                        );
                        then.respond_with(rpc_result(result));
                    })
                    .await,
            );
        }
        let agreed = SuperRootProof::new(
            100,
            vec![SuperOutputRoot { chain_id: 10, output_root: b256(0x44) }],
        );
        let host = build_interop_host(
            &mock_host_inputs(&server),
            b256(0xaa),
            &encode_super_root_proof(&agreed).unwrap(),
            b256(0xbb),
            101,
            None,
        )
        .unwrap();
        // kona-host retries a hint it cannot serve, so a wrong read would hang, not fail.
        let block = tokio::time::timeout(
            std::time::Duration::from_secs(30),
            optimistic_block_receipts(host, 10, parent.hash_slow(), hash),
        )
        .await
        .expect("the host must serve the block")
        .unwrap();

        assert_eq!(block.header, header);
        assert_eq!(block.receipts, receipts);
        for read in reads {
            read.assert_async().await;
        }
    }

    /// A mock chain-10 L2 node, L1 node, and beacon node that answers what kona-host reads at
    /// startup.
    async fn mock_nodes() -> MockServer {
        let server = MockServer::start_async().await;
        server
            .mock_async(|when, then| {
                when.method(GET).path("/eth/v1/beacon/genesis");
                then.status(200).json_body(json!({ "data": { "genesis_time": "10" } }));
            })
            .await;
        server
            .mock_async(|when, then| {
                when.method(GET).path("/eth/v1/config/spec");
                then.status(200).json_body(json!({ "data": { "SECONDS_PER_SLOT": "12" } }));
            })
            .await;
        server
            .mock_async(|when, then| {
                when.method(POST)
                    .json_body_includes(json!({ "method": "eth_chainId" }).to_string());
                then.respond_with(rpc_result(json!("0xa")));
            })
            .await;
        server
    }

    /// Host inputs that read every node from `server`.
    fn mock_host_inputs(server: &MockServer) -> HostInputs {
        HostInputs {
            l1_node_address: server.base_url(),
            l1_beacon_address: server.base_url(),
            l2_node_addresses: vec![server.base_url()],
            rollup_config_paths: None,
            l1_config_path: None,
            dependency_set_path: None,
        }
    }

    /// Answers with `result` under the request's JSON-RPC id, which the client checks.
    fn rpc_result(
        result: Value,
    ) -> impl Fn(&HttpMockRequest) -> HttpMockResponse + Send + Sync + 'static {
        move |request| {
            let id = serde_json::from_slice::<Value>(&request.body().0).unwrap()["id"].clone();
            HttpMockResponse::builder()
                .status(200)
                .header("content-type", "application/json")
                .body(json!({ "jsonrpc": "2.0", "id": id, "result": result }).to_string())
                .build()
        }
    }

    #[test]
    fn synthesize_execution_rejects_unsorted_chain_ids() {
        let previous = response(100, &[10, 20], 7);
        let current = response(101, &[20, 10], 8);

        let err = synthesize_execution(
            TimestampSpan::new(101, 101).unwrap(),
            b256(0xaa),
            8,
            &[previous, current],
        )
        .unwrap_err();

        assert!(err.to_string().contains("strictly increasing"));
    }

    #[test]
    fn synthesize_execution_rejects_missing_optimistic_output() {
        let previous = response(100, &[10, 20], 7);
        let mut current = response(101, &[10, 20], 8);
        current.optimistic_at_timestamp.remove(&ChainId(U256::from(20)));

        let err = synthesize_execution(
            TimestampSpan::new(101, 101).unwrap(),
            b256(0xaa),
            8,
            &[previous, current],
        )
        .unwrap_err();

        assert!(err.to_string().contains("missing optimistic output for chain 20"));
    }

    #[test]
    fn synthesize_execution_rejects_required_l1_after_l1_head() {
        let previous = response(100, &[10, 20], 7);
        let current = response(101, &[10, 20], 9);

        let err = synthesize_execution(
            TimestampSpan::new(101, 101).unwrap(),
            b256(0xaa),
            8,
            &[previous, current],
        )
        .unwrap_err();

        assert!(err.to_string().contains("after supplied L1 head"));
    }

    #[test]
    fn synthesize_execution_span_multi_ts() {
        let chain_ids = [10u64, 20];
        let responses: Vec<_> = (100u64..=103)
            .map(|timestamp| {
                response_with_block_hash(timestamp, &chain_ids, 8, |chain_id| {
                    b256(chain_id as u8 ^ timestamp as u8)
                })
            })
            .collect();
        let span = TimestampSpan::new(101, 103).unwrap();

        let synthesized =
            synthesize_execution(span, b256(0xaa), 8, &responses).expect("synthesizes");

        // `previous_super_root_proofs[i]` is the proof for `span.start + i - 1`.
        assert_eq!(synthesized.range_inputs.span, span);
        assert_eq!(synthesized.range_inputs.previous_super_root_proofs.len(), 3);
        for (offset, proof) in
            synthesized.range_inputs.previous_super_root_proofs.iter().enumerate()
        {
            assert_eq!(proof.super_root.timestamp, 100 + offset as u64);
            assert_eq!(
                *proof,
                proof_from_super_v1(&responses[offset].data.as_ref().unwrap().super_v1).unwrap(),
            );
        }

        // One claimed transition per (timestamp, chain), timestamp-major and chain-minor.
        let transitions = &synthesized.range_inputs.claimed_transitions;
        assert_eq!(transitions.len(), 6);
        for (index, transition) in transitions.iter().enumerate() {
            let timestamp = 101 + (index / 2) as u64;
            let chain_id = chain_ids[index % 2];
            assert_eq!(transition.timestamp, timestamp);
            assert_eq!(transition.optimistic_block.chain_id, U256::from(chain_id));
            let expected = &responses[(timestamp - 100) as usize].optimistic_at_timestamp
                [&ChainId(U256::from(chain_id))];
            assert_eq!(transition.optimistic_block.output_root, expected.output_root);
            assert_eq!(
                transition.optimistic_block.block_hash,
                expected.output.as_ref().unwrap().block_hash,
            );
        }

        // Consolidation covers every timestamp with the verified super-root claims.
        assert_eq!(synthesized.consolidation_inputs.span, span);
        assert_eq!(
            synthesized.consolidation_inputs.previous_super_root,
            responses[0].data.as_ref().unwrap().super_root,
        );
        assert_eq!(synthesized.consolidation_inputs.transitions.len(), 3);
        for (offset, transition) in synthesized.consolidation_inputs.transitions.iter().enumerate()
        {
            assert_eq!(
                transition.claimed_super_root_proof.super_root.timestamp,
                101 + offset as u64,
            );
            assert_eq!(
                transition.optimistic_blocks,
                transitions[offset * 2..(offset + 1) * 2]
                    .iter()
                    .map(|transition| transition.optimistic_block)
                    .collect::<Vec<_>>(),
            );
        }

        // Preloading covers every super-root proof and every optimistic output root in the span.
        let preloaded: std::collections::HashSet<_> =
            synthesized.preloaded_preimages.iter().map(|(key, _)| *key).collect();
        for response in &responses {
            let data = response.data.as_ref().unwrap();
            assert!(preloaded.contains(&PreimageKey::new_keccak256(*data.super_root)));
            for entry in response.optimistic_at_timestamp.values() {
                assert!(preloaded.contains(&PreimageKey::new_keccak256(*entry.output_root)));
            }
        }
        assert_eq!(synthesized.preloaded_preimages.len(), 12);
        assert_eq!(synthesized.current_super_root, responses[3].data.as_ref().unwrap().super_root,);
    }

    #[test]
    fn synthesize_execution_single_ts_unchanged() {
        let previous = response(100, &[10, 20], 7);
        let current = response(101, &[10, 20], 8);
        let previous_proof =
            proof_from_super_v1(&previous.data.as_ref().unwrap().super_v1).unwrap();
        let current_proof = proof_from_super_v1(&current.data.as_ref().unwrap().super_v1).unwrap();

        let synthesized = synthesize_execution(
            TimestampSpan::new(101, 101).unwrap(),
            b256(0xaa),
            8,
            &[previous.clone(), current.clone()],
        )
        .expect("synthesizes");

        // Pin the exact bundle the pre-span single-timestamp function produced on this fixture,
        // including preimage preload order: both proofs, then current outputs, then previous.
        let expected_transitions: Vec<_> = [10u64, 20]
            .iter()
            .map(|&chain_id| {
                let entry = &current.optimistic_at_timestamp[&ChainId(U256::from(chain_id))];
                SuperRangeTransition {
                    timestamp: 101,
                    optimistic_block: SuperOptimisticBlock {
                        chain_id: U256::from(chain_id),
                        block_hash: entry.output.as_ref().unwrap().block_hash,
                        output_root: entry.output_root,
                    },
                }
            })
            .collect();
        let mut expected_preimages = Vec::new();
        for proof in [&previous_proof, &current_proof] {
            expected_preimages.push((
                PreimageKey::new_keccak256(*hash_super_root_proof(proof).unwrap()),
                encode_super_root_proof(proof).unwrap(),
            ));
        }
        for response in [&current, &previous] {
            for &chain_id in &[10u64, 20] {
                let entry = &response.optimistic_at_timestamp[&ChainId(U256::from(chain_id))];
                expected_preimages.push((
                    PreimageKey::new_keccak256(*entry.output_root),
                    entry.output.as_ref().unwrap().marshal(),
                ));
            }
        }
        let expected = SynthesizedExecution {
            range_inputs: SuperRangeInputs {
                span: TimestampSpan::new(101, 101).unwrap(),
                l1_head: b256(0xaa),
                chain_ids: vec![U256::from(10), U256::from(20)],
                previous_super_root_proofs: vec![previous_proof.clone()],
                claimed_transitions: expected_transitions.clone(),
            },
            consolidation_inputs: SuperConsolidationInputs {
                span: TimestampSpan::new(101, 101).unwrap(),
                previous_super_root: previous.data.as_ref().unwrap().super_root,
                transitions: vec![SuperConsolidationTransitionInput {
                    optimistic_blocks: expected_transitions
                        .iter()
                        .map(|transition| transition.optimistic_block)
                        .collect(),
                    claimed_super_root_proof: current_proof,
                }],
            },
            preloaded_preimages: expected_preimages,
            replaced_agreed_outputs: Vec::new(),
            current_super_root: current.data.as_ref().unwrap().super_root,
            previous_super_root_proof_bytes: encode_super_root_proof(&previous_proof).unwrap(),
        };

        assert_eq!(synthesized, expected);
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
            "safe_timestamp":"0x65",
            "local_safe_timestamp":102,
            "finalized_timestamp":100,
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
        assert_eq!(decoded.current_safe_timestamp, 101);
        assert_eq!(decoded.current_local_safe_timestamp, 102);
        assert_eq!(decoded.current_finalized_timestamp, 100);
        assert_eq!(decoded.data.unwrap().super_v1.timestamp, 101);
    }

    #[test]
    fn rejects_supernode_json_missing_timestamp_fields() {
        let json = r#"{
            "current_l1":{"hash":"0x0101010101010101010101010101010101010101010101010101010101010101","number":9},
            "chain_ids":["10"],
            "data":null
        }"#;

        let err = serde_json::from_str::<SuperRootAtTimestampResponse>(json).unwrap_err();

        assert!(err.to_string().contains("missing field `safe_timestamp`"));
    }

    #[test]
    fn stdin_and_public_values_round_trip_range_outputs() {
        let previous = response(100, &[10], 7);
        let current = response(101, &[10], 8);
        let synthesized = synthesize_execution(
            TimestampSpan::new(101, 101).unwrap(),
            b256(0xaa),
            8,
            &[previous, current],
        )
        .expect("synthesizes");
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

    #[tokio::test]
    async fn dropping_witness_collection_aborts_server_task() {
        let (started_tx, started_rx) = tokio::sync::oneshot::channel();
        let server_task = tokio::spawn(async move {
            let _ = started_tx.send(());
            std::future::pending::<()>().await;
        });
        let server_guard = AbortOnDrop(server_task.abort_handle());
        started_rx.await.unwrap();

        drop(server_guard);

        assert!(server_task.await.unwrap_err().is_cancelled());
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

    fn sized_witness(entries: u64, value_len: usize, blobs: usize) -> DefaultWitnessData {
        let mut store = PreimageStore::default();
        for index in 0..entries {
            let mut key = [0; 32];
            key[..8].copy_from_slice(&index.to_le_bytes());
            let key = PreimageKey::new(key, kona_preimage::PreimageKeyType::GlobalGeneric);
            store.save_preimage(key, vec![0xab; value_len]).unwrap();
        }
        let blob_data = BlobData {
            blobs: vec![kzg_rs::Blob([0; kzg_rs::BYTES_PER_BLOB]); blobs],
            commitments: vec![kzg_rs::Bytes48([0; 48]); blobs],
            proofs: vec![kzg_rs::Bytes48([0; 48]); blobs],
        };
        DefaultWitnessData::from_parts(store, blob_data)
    }

    #[test]
    fn witness_archive_bound_covers_the_real_archive() {
        for entries in [0, 1, 16, 17, 1_024, 1_025, 65_536, 65_537] {
            for value_len in [0, 1, 33, 4_096] {
                if entries > 1_025 && value_len > 33 {
                    continue;
                }
                for blobs in [0, 2] {
                    let witness = sized_witness(entries, value_len, blobs);
                    let archived = rkyv::to_bytes::<rkyv::rancor::Error>(&witness).unwrap();
                    assert!(
                        archived.len() as u64 <= witness_archive_bound(&witness),
                        "{entries} entries of {value_len} bytes and {blobs} blobs archive to {} \
                         bytes, above the bound {}",
                        archived.len(),
                        witness_archive_bound(&witness),
                    );
                }
            }
        }
    }

    #[test]
    fn archive_witness_refuses_a_bound_over_the_limit() {
        let witness = sized_witness(17, 33, 2);
        let bound = witness_archive_bound(&witness);

        let err = archive_witness_within(&witness, bound - 1).unwrap_err();
        assert!(is_witness_too_large(&err));
        assert_eq!(
            err.downcast_ref::<WitnessTooLarge>(),
            Some(&WitnessTooLarge { bound, limit: bound - 1 }),
        );

        let archived = archive_witness_within(&witness, bound).unwrap();
        let expected = rkyv::to_bytes::<rkyv::rancor::Error>(&witness).unwrap();
        assert_eq!(archived.as_slice(), expected.as_slice());
    }
}
