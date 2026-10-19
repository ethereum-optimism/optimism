//! Super-root witness collection and aggregation proof assembly.
//!
//! Witnesses come from `kona-sp1-super-range-executor`'s `InteropHost`, and aggregation inputs
//! use [`SuperAggregationInputs`]. The aggregation guest validates one shared `l1_head`.
//! Both proof providers share span chunking, native witness and output collection, and
//! aggregation-input validation. The mock provider then returns placeholder bytes; the network
//! provider proves each chunk in compressed mode and aggregates them with PLONK.

use std::{
    collections::{BTreeMap, HashMap},
    future::Future,
    num::NonZeroUsize,
    ops::AsyncFnOnce,
    sync::{
        Arc,
        atomic::{AtomicBool, Ordering},
    },
};

use alloy_primitives::{Address, B256, U256, keccak256};
use alloy_sol_types::SolValue;
use anyhow::{Context, Result, anyhow, bail, ensure};
use futures::{
    FutureExt,
    future::{BoxFuture, Shared},
    stream::{FuturesUnordered, StreamExt},
};
use kona_sp1_client_utils::{
    log_commitment::LogCommitmentInputs,
    super_root::{
        SuperAggregationInputs, SuperAggregationPublicValues, SuperConsolidationOutputs,
        SuperRangeInputs, SuperRangeOutputs, TimestampSpan,
    },
    witness::DefaultWitnessData,
};
use kona_sp1_host_utils::metrics::MetricsGauge;
use kona_sp1_super_range_executor::{
    HostInputs, LogCommitmentProofs, ScannedTimestampCost, SuperRootAtTimestampResponse,
    SynthesizedExecution, build_interop_host, build_super_consolidation_stdin,
    build_super_range_stdin, collect_consolidation_witness, collect_range_witness,
    decode_super_consolidation_public_values, decode_super_range_public_values,
    deployment_chain_configs, estimate_consolidation_pgu, proof_from_super_v1,
    synthesize_execution,
};
use parking_lot::Mutex;
use serde::Serialize;
use sp1_sdk::{HashableKey, SP1Proof, SP1ProofWithPublicValues, SP1Stdin};
use tracing::Instrument;

pub use crate::ports::GameProofInputs;

use crate::{
    config::{LogPath, RangeSplitCount},
    log_proofs::{
        CollectedRequest, LogBatch, LogBatchProver, MockLogProver, NetworkLogProver, RequestBound,
        check_collected, choose_log_path, consolidation_cost, cover, log_batch_poll_interval,
        make_log_batches, receipts_path_costs, request_budget, scan::scan_span,
    },
    metrics::ProposerGauge,
    ports::{L2BlockSource, SuperRootSource},
    prover::{
        MOCK_PROOF_BYTES, ProofId, ProofKeys, ProofProvider, ProofTerminalState, ProofWaitError,
    },
};

/// Marker error: this game can never be proven by this proposer. The
/// defense scheduler maps it into the permanent `undefendable` set so the
/// span is not re-fetched every cycle.
#[derive(Debug, Clone)]
pub struct GameUnprovable(pub String);

impl std::fmt::Display for GameUnprovable {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(f, "game is unprovable: {}", self.0)
    }
}

impl std::error::Error for GameUnprovable {}

/// Marker error: super-root data for a timestamp is not (yet) available
/// from the supernode. Transient: the defense task fails and retries next
/// cycle.
#[derive(Debug, Clone)]
pub struct SuperRootDataUnavailable(pub u64);

impl std::fmt::Display for SuperRootDataUnavailable {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(f, "super-root data unavailable for timestamp {}", self.0)
    }
}

impl std::error::Error for SuperRootDataUnavailable {}

/// Returns true when `err`'s chain contains a [`GameUnprovable`] marker.
pub fn is_unprovable(err: &anyhow::Error) -> bool {
    err.chain().any(|cause| cause.is::<GameUnprovable>())
}

/// Fetches verified `superroot_atTimestamp` responses for
/// `starting_ts..=claim_ts`.
///
/// Missing data is transient. Trusted endpoint mismatches or claims requiring
/// L1 data beyond the game's pinned head make the game permanently
/// unprovable. A response is trusted only when
/// `current_l1 > verified_required_l1`; stale responses during rewinds cannot
/// determine a permanent verdict.
pub(crate) async fn fetch_span_responses(
    source: &dyn SuperRootSource,
    game: &GameProofInputs,
) -> Result<Vec<SuperRootAtTimestampResponse>> {
    ensure!(
        game.starting_ts < game.claim_ts,
        "game claims timestamp {} at or before its starting timestamp {}",
        game.claim_ts,
        game.starting_ts,
    );
    let mut responses = Vec::with_capacity((game.claim_ts - game.starting_ts + 1) as usize);
    let mut roots = Vec::with_capacity(responses.capacity());
    for timestamp in game.starting_ts..=game.claim_ts {
        let super_root_at = source.super_root_at_timestamp(timestamp).await?;
        let Some(root) = super_root_at.root else {
            ProposerGauge::SuperRootUnavailable.increment(1.0);
            return Err(anyhow::Error::new(SuperRootDataUnavailable(timestamp)));
        };
        roots.push(root.super_root);
        responses.push(super_root_at.response);
    }
    check_span_roots(game, &roots, &responses)?;
    check_provable(game, responses.last().expect("span is non-empty"))?;
    Ok(responses)
}

/// Reports whether response data may support a permanent verdict.
///
/// The API reports blocks strictly below `current_l1` as fully processed.
/// Data is trusted only when `current_l1 > verified_required_l1`. Untrusted
/// data may still be proved because the guest re-derives it, but it cannot
/// mark a game permanently unprovable.
pub(crate) fn response_trusted(response: &SuperRootAtTimestampResponse) -> bool {
    response
        .data
        .as_ref()
        .is_some_and(|data| response.current_l1.number > data.verified_required_l1.number)
}

/// Classifies a supernode-vs-game contradiction: permanent when the answer
/// is trusted, transient (retry next cycle) when the supernode itself marks
/// its answer as not yet re-verified.
fn contradiction(
    response: &SuperRootAtTimestampResponse,
    timestamp: u64,
    description: String,
) -> anyhow::Error {
    if response_trusted(response) {
        anyhow::Error::new(GameUnprovable(description))
    } else {
        ProposerGauge::SuperRootUnavailable.increment(1.0);
        tracing::warn!(
            timestamp,
            %description,
            "supernode contradicts the game but its answer is not trusted yet \
             (current_l1 below verified_required_l1, e.g. mid-rewind); retrying"
        );
        anyhow::Error::new(SuperRootDataUnavailable(timestamp))
    }
}

/// Verifies the fetched span's endpoints against the game's on-chain roots.
/// A trusted mismatch means the supernode's canonical view diverged from
/// the claim this proposer admitted at sync time: permanently unprovable.
fn check_span_roots(
    game: &GameProofInputs,
    roots: &[B256],
    responses: &[SuperRootAtTimestampResponse],
) -> Result<()> {
    let first = roots.first().context("span produced no roots")?;
    let last = roots.last().context("span produced no roots")?;
    if *first != game.starting_root {
        return Err(contradiction(
            responses.first().context("span produced no responses")?,
            game.starting_ts,
            format!(
                "supernode super root {first} at timestamp {} does not match the game's starting root {}",
                game.starting_ts, game.starting_root,
            ),
        ));
    }
    if *last != game.root_claim {
        return Err(contradiction(
            responses.last().context("span produced no responses")?,
            game.claim_ts,
            format!(
                "supernode super root {last} at timestamp {} does not match the game's root claim {}",
                game.claim_ts, game.root_claim,
            ),
        ));
    }
    Ok(())
}

/// Verifies the claim is derivable from L1 data at or below the game's
/// pinned L1 head. A trusted violation is permanent: the game pinned its
/// L1 head at creation and can never be honestly proven (#18503/#21773
/// analog for super games). An untrusted one retries.
fn check_provable(game: &GameProofInputs, last: &SuperRootAtTimestampResponse) -> Result<()> {
    let data = last.data.as_ref().context("verified span response lost its data")?;
    let required = data.verified_required_l1.number;
    if required > game.l1_head_number {
        return Err(contradiction(
            last,
            game.claim_ts,
            format!(
                "claim requires L1 block {required} but the game's L1 head is block {}",
                game.l1_head_number,
            ),
        ));
    }
    Ok(())
}

#[derive(Clone, Debug, PartialEq, Eq)]
struct ChunkIdentity {
    index: u8,
    agreed_ts: u64,
    claimed_ts: u64,
    range_input_digest: B256,
    consolidation_input_digest: B256,
}

#[derive(Clone, Debug, PartialEq, Eq)]
enum ProviderIdentity {
    Mock,
    Network { range_vkey: B256, aggregation_vkey: B256, log_vkey: [u32; 8] },
}

#[derive(Clone, Debug, PartialEq, Eq)]
struct AttemptIdentity {
    game: GameProofInputs,
    provider: ProviderIdentity,
    chunks: Vec<ChunkIdentity>,
}

struct ChunkPlan {
    index: usize,
    agreed_timestamp: u64,
    claimed_timestamp: u64,
    /// The chunk's responses are `first..=last` of the game's span; its transitions are
    /// `first..last` of the span's transition timestamps.
    first: usize,
    last: usize,
    synthesized: SynthesizedExecution,
}

#[derive(Clone, Default)]
enum RequestState {
    #[default]
    Missing,
    Submitting,
    Submitted(ProofId),
    Fulfilled(Arc<SP1ProofWithPublicValues>),
    Terminal(ProofTerminalState),
}

/// SPN request kinds of one game proof, as named by [`GameProgress::requests`].
pub(crate) const PROOF_REQUEST_KINDS: [&str; 4] = ["range", "consolidation", "aggregation", "log"];
/// Request states reported by [`InMemoryProofProgress::request_counts`].
pub(crate) const PROOF_REQUEST_STATES: [&str; 4] =
    ["submitting", "submitted", "fulfilled", "terminal"];
/// Request counts keyed by `(kind, state)`; absent keys are zero.
pub(crate) type ProofRequestCounts = BTreeMap<(&'static str, &'static str), usize>;

impl RequestState {
    /// State label for request metrics; `None` for slots with no request yet.
    const fn metric_state(&self) -> Option<&'static str> {
        match self {
            Self::Missing => None,
            Self::Submitting => Some("submitting"),
            Self::Submitted(_) => Some("submitted"),
            Self::Fulfilled(_) => Some("fulfilled"),
            Self::Terminal(_) => Some("terminal"),
        }
    }
}

#[derive(Debug, Default, PartialEq, Eq)]
struct RequestSummary {
    submitted_ids: Vec<ProofId>,
    submitting: usize,
    fulfilled: usize,
}

#[derive(Default)]
struct ChunkProgress {
    completed: Mutex<Option<Arc<ChunkResult>>>,
    range_request: Mutex<RequestState>,
    consolidation_request: Mutex<RequestState>,
}

/// A log-commitment batch of a game and its proof request.
struct LogBatchProgress {
    batch: LogBatch,
    request: Mutex<RequestState>,
}

struct GameProgress {
    identity: AttemptIdentity,
    chunks: Vec<ChunkProgress>,
    aggregation_request: Mutex<RequestState>,
    log_stage: Mutex<Option<Arc<LogStage>>>,
}

impl GameProgress {
    fn new(identity: AttemptIdentity) -> Self {
        let chunks = (0..identity.chunks.len()).map(|_| ChunkProgress::default()).collect();
        Self {
            identity,
            chunks,
            aggregation_request: Mutex::new(RequestState::Missing),
            log_stage: Mutex::new(None),
        }
    }

    fn log_stage(&self) -> Option<Arc<LogStage>> {
        self.log_stage.lock().clone()
    }

    fn store_log_stage(&self, stage: LogStage) -> Arc<LogStage> {
        let stage = Arc::new(stage);
        *self.log_stage.lock() = Some(Arc::clone(&stage));
        stage
    }

    /// Visits every request slot with its chunk index and kind, including the log stage's.
    fn visit_requests(
        &self,
        mut visit: impl FnMut(Option<usize>, &'static str, &Mutex<RequestState>),
    ) {
        for (chunk_index, kind, request) in self.requests() {
            visit(chunk_index, kind, request);
        }
        for progress in self.log_stage.lock().iter().flat_map(|stage| stage.batches.iter()) {
            visit(None, "log", &progress.request);
        }
    }

    fn requests(
        &self,
    ) -> impl Iterator<Item = (Option<usize>, &'static str, &Mutex<RequestState>)> {
        self.chunks
            .iter()
            .enumerate()
            .flat_map(|(index, chunk)| {
                [
                    (Some(index), "range", &chunk.range_request),
                    (Some(index), "consolidation", &chunk.consolidation_request),
                ]
            })
            .chain(std::iter::once((None, "aggregation", &self.aggregation_request)))
    }

    fn check_terminal_requests(&self) -> Result<()> {
        let stage = self.log_stage();
        for progress in stage.iter().flat_map(|stage| stage.batches.iter()) {
            if let RequestState::Terminal(outcome) = &*progress.request.lock() {
                bail!("{} proof request reached terminal state: {outcome:?}", progress.batch);
            }
        }
        for (chunk_index, kind, request) in self.requests() {
            match &*request.lock() {
                RequestState::Terminal(outcome) => {
                    if let Some(index) = chunk_index {
                        bail!(
                            "chunk {index} {kind} proof request reached terminal state: {outcome:?}"
                        );
                    }
                    bail!("{kind} proof request reached terminal state: {outcome:?}");
                }
                RequestState::Missing |
                RequestState::Submitting |
                RequestState::Submitted(_) |
                RequestState::Fulfilled(_) => {}
            }
        }
        Ok(())
    }

    fn request_summary(&self) -> RequestSummary {
        let mut summary = RequestSummary::default();
        let mut record = |state: &Mutex<RequestState>| match &*state.lock() {
            RequestState::Submitting => summary.submitting += 1,
            RequestState::Submitted(proof_id) => summary.submitted_ids.push(*proof_id),
            RequestState::Fulfilled(_) => summary.fulfilled += 1,
            RequestState::Missing | RequestState::Terminal(_) => {}
        };
        self.visit_requests(|_, _, request| record(request));
        summary
    }
}

/// Process-local proof work retained across scheduler retries.
#[derive(Default)]
pub struct InMemoryProofProgress {
    games: Mutex<HashMap<Address, Arc<GameProgress>>>,
}

impl std::fmt::Debug for InMemoryProofProgress {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        f.debug_struct("InMemoryProofProgress").field("games", &self.games.lock().len()).finish()
    }
}

impl InMemoryProofProgress {
    fn load_or_create(
        &self,
        game_address: Address,
        identity: AttemptIdentity,
    ) -> Arc<GameProgress> {
        let mut games = self.games.lock();
        if let Some(progress) = games.get(&game_address) {
            if progress.identity == identity {
                return Arc::clone(progress);
            }
            let summary = progress.request_summary();
            tracing::warn!(
                %game_address,
                submitted_request_ids = ?summary.submitted_ids,
                submitting_request_count = summary.submitting,
                fulfilled_request_count = summary.fulfilled,
                game_changed = progress.identity.game != identity.game,
                provider_changed = progress.identity.provider != identity.provider,
                chunks_changed = progress.identity.chunks != identity.chunks,
                "Proving inputs changed; replacing cached proof progress"
            );
        }

        let progress = Arc::new(GameProgress::new(identity));
        games.insert(game_address, Arc::clone(&progress));
        progress
    }

    pub(crate) fn clear(&self, game_address: Address) {
        self.games.lock().remove(&game_address);
    }

    /// Resets only terminal request slots.
    pub(crate) fn retry_terminal_requests(&self, game_address: Address) -> usize {
        let games = self.games.lock();
        let Some(progress) = games.get(&game_address) else {
            return 0;
        };
        let mut reset = 0;
        progress.visit_requests(|_, _, request| {
            let mut state = request.lock();
            match &*state {
                RequestState::Terminal(_) => {
                    *state = RequestState::Missing;
                    reset += 1;
                }
                RequestState::Missing |
                RequestState::Submitting |
                RequestState::Submitted(_) |
                RequestState::Fulfilled(_) => {}
            }
        });
        reset
    }

    /// Counts request slots of every game with retained progress, by kind and state. Progress is
    /// cleared once a game's proof is submitted, so these are the games still being proven.
    pub(crate) fn request_counts(&self) -> ProofRequestCounts {
        let mut counts = ProofRequestCounts::new();
        for progress in self.games.lock().values() {
            progress.visit_requests(|_, kind, request| {
                if let Some(state) = request.lock().metric_state() {
                    *counts.entry((kind, state)).or_default() += 1;
                }
            });
        }
        counts
    }
}

fn fingerprint<T: Serialize>(value: &T) -> Result<B256> {
    let bytes = bincode::serde::encode_to_vec(value, bincode::config::standard())
        .context("encode proof input fingerprint")?;
    Ok(keccak256(bytes))
}

fn provider_identity(
    provider: &ProofProvider,
    keys: Option<&ProofKeys>,
) -> Result<ProviderIdentity> {
    if provider.is_mock() {
        return Ok(ProviderIdentity::Mock);
    }
    let keys = keys.context("network proving requires prestate proving keys")?;
    Ok(ProviderIdentity::Network {
        range_vkey: B256::from(keys.range_vk.bytes32_raw()),
        aggregation_vkey: B256::from(keys.agg_vk.bytes32_raw()),
        log_vkey: keys.log_vk.hash_u32(),
    })
}

fn expected_aggregation_public_values(game: &GameProofInputs) -> Vec<u8> {
    SuperAggregationPublicValues {
        l1Head: game.l1_head,
        startingRootHash: game.starting_root,
        rootClaim: game.root_claim,
        l2SequenceNumber: U256::from(game.claim_ts),
        prover: game.prover,
    }
    .abi_encode()
}
const fn terminal_request_can_retry(outcome: ProofTerminalState) -> bool {
    match outcome {
        ProofTerminalState::Cancelled |
        ProofTerminalState::Expired |
        ProofTerminalState::Reverted |
        ProofTerminalState::Unfulfillable => true,
        ProofTerminalState::Unexecutable | ProofTerminalState::ValidationFailed => false,
    }
}

/// A request slot after [`ensure_submitted`].
enum Claimed {
    Submitted(ProofId),
    Fulfilled(Arc<SP1ProofWithPublicValues>),
}

/// Submits the request of a missing slot. A submitted or fulfilled slot is left as it is.
async fn ensure_submitted(
    state: &Mutex<RequestState>,
    submit: impl AsyncFnOnce() -> Result<ProofId>,
) -> Result<Claimed> {
    {
        let mut state = state.lock();
        match &*state {
            RequestState::Missing => *state = RequestState::Submitting,
            RequestState::Submitting => bail!("proof request submission is already in progress"),
            RequestState::Submitted(proof_id) => return Ok(Claimed::Submitted(*proof_id)),
            RequestState::Fulfilled(proof) => return Ok(Claimed::Fulfilled(Arc::clone(proof))),
            RequestState::Terminal(outcome) => {
                bail!("proof request reached terminal state: {outcome:?}")
            }
        }
    }
    match submit().await {
        Ok(proof_id) => {
            *state.lock() = RequestState::Submitted(proof_id);
            Ok(Claimed::Submitted(proof_id))
        }
        Err(err) => {
            *state.lock() = RequestState::Missing;
            Err(err)
        }
    }
}

async fn complete_request(
    state: &Mutex<RequestState>,
    submit: impl AsyncFnOnce() -> Result<ProofId>,
    poll: impl AsyncFnOnce(ProofId) -> Result<SP1ProofWithPublicValues, ProofWaitError>,
    verify: impl FnOnce(&SP1ProofWithPublicValues) -> Result<()>,
) -> Result<Arc<SP1ProofWithPublicValues>> {
    let proof_id = match ensure_submitted(state, submit).await? {
        Claimed::Submitted(proof_id) => proof_id,
        Claimed::Fulfilled(proof) => return Ok(proof),
    };

    let proof = match poll(proof_id).await {
        Ok(proof) => proof,
        Err(err @ ProofWaitError::Terminal { state: outcome, .. }) => {
            *state.lock() = if terminal_request_can_retry(outcome) {
                RequestState::Missing
            } else {
                RequestState::Terminal(outcome)
            };
            return Err(err.into());
        }
        Err(err) => return Err(err.into()),
    };
    if let Err(err) = verify(&proof) {
        *state.lock() = RequestState::Terminal(ProofTerminalState::ValidationFailed);
        return Err(err);
    }

    let proof = Arc::new(proof);
    *state.lock() = RequestState::Fulfilled(Arc::clone(&proof));
    Ok(proof)
}

/// Submits the batch request of `prover` for `batch`, and counts it.
async fn submit_log_batch(prover: &dyn LogBatchProver, batch: &LogBatch) -> Result<ProofId> {
    let proof_id = prover.submit(batch).await?;
    ProposerGauge::LogBatchesSubmitted.increment(1.0);
    tracing::info!(%proof_id, batch = %batch, "Submitted a log-commitment batch proof request");
    Ok(proof_id)
}

/// Submits every batch request that is not submitted yet, before any wait.
async fn submit_log_batches(
    prover: &dyn LogBatchProver,
    batches: &[LogBatchProgress],
) -> Result<()> {
    for progress in batches {
        let batch = &progress.batch;
        ensure_submitted(&progress.request, async || submit_log_batch(prover, batch).await)
            .await
            .with_context(|| format!("failed to submit {batch}"))?;
    }
    Ok(())
}

/// Waits for every batch proof, in batch order. A batch that fails ends the wait.
async fn wait_for_log_batches(
    prover: &dyn LogBatchProver,
    batches: &[LogBatchProgress],
) -> Result<BatchProofs> {
    let proofs = futures::future::try_join_all(batches.iter().map(|progress| async move {
        let batch = &progress.batch;
        complete_request(
            &progress.request,
            async || submit_log_batch(prover, batch).await,
            async |proof_id| prover.poll(batch, proof_id).await,
            |proof| prover.verify(batch, proof),
        )
        .await
        .inspect_err(|_| ProposerGauge::LogBatchError.increment(1.0))
        .with_context(|| format!("{batch} failed"))
    }))
    .await?;
    Ok(proofs.into())
}

/// Terminal requests are rejected by the preflight in `plan_game_attempt`.
fn reuse_fulfilled_aggregation(
    state: &Mutex<RequestState>,
    game: &GameProofInputs,
    verify: impl FnOnce(&SP1ProofWithPublicValues) -> Result<()>,
) -> Result<Option<Vec<u8>>> {
    let proof = match &*state.lock() {
        RequestState::Fulfilled(proof) => Arc::clone(proof),
        _ => return Ok(None),
    };
    verify(&proof)?;
    ensure!(
        proof.public_values.as_slice() == expected_aggregation_public_values(game),
        "aggregation proof public values do not match the game's expected tuple"
    );
    Ok(Some(proof.bytes()))
}

async fn run_chunk_with_progress<F>(progress: &ChunkProgress, task: F) -> Result<Arc<ChunkResult>>
where
    F: Future<Output = Result<ChunkResult>>,
{
    if let Some(result) = progress.completed.lock().as_ref() {
        return Ok(Arc::clone(result));
    }
    let result = Arc::new(task.await?);
    *progress.completed.lock() = Some(Arc::clone(&result));
    Ok(result)
}

#[derive(Clone)]
struct ChildProofs {
    range: Arc<SP1ProofWithPublicValues>,
    consolidation: Arc<SP1ProofWithPublicValues>,
}

/// One proven (or natively executed) span chunk, tagged with its position.
#[derive(Clone)]
struct ChunkResult {
    index: usize,
    range_outputs: SuperRangeOutputs,
    consolidation_outputs: SuperConsolidationOutputs,
    proofs: Option<ChildProofs>,
}

/// Runs a range-witness chunk with proof-replay context attached for its full async lifetime.
async fn with_proof_replay_chunk_span<F>(
    index: usize,
    chunk_count: usize,
    agreed_timestamp: u64,
    claimed_timestamp: u64,
    future: F,
) -> F::Output
where
    F: Future,
{
    let guest_span_start_timestamp = agreed_timestamp + 1;
    future
        .instrument(tracing::warn_span!(
            "proof_replay_chunk",
            derivation_mode = "proof_replay",
            chunk_index = index,
            chunk_count,
            agreed_timestamp,
            claimed_timestamp,
            guest_span_start_timestamp,
            guest_span_end_timestamp = claimed_timestamp,
        ))
        .await
}

/// Runs tasks with bounded concurrency.
///
/// After the first error, stops admitting queued tasks, drains admitted tasks,
/// and returns the first error. Returns every result when all tasks succeed.
async fn run_chunk_tasks_until_error<I, F, T>(tasks: I, max_concurrent: usize) -> Result<Vec<T>>
where
    I: IntoIterator<Item = F>,
    F: Future<Output = Result<T>>,
{
    let mut tasks = tasks.into_iter();
    let mut active: FuturesUnordered<_> = tasks.by_ref().take(max_concurrent).collect();
    let mut results = Vec::with_capacity(tasks.size_hint().0 + active.len());
    let mut first_error = None;

    while let Some(result) = active.next().await {
        match result {
            Ok(result) if first_error.is_none() => results.push(result),
            Ok(_) => {}
            Err(err) if first_error.is_some() => {
                tracing::warn!(error = %err, "Admitted chunk failed after another chunk");
            }
            Err(err) => first_error = Some(err),
        }

        if first_error.is_none() &&
            let Some(task) = tasks.next()
        {
            active.push(task);
        }
    }

    first_error.map_or_else(|| Ok(results), Err)
}

async fn run_indexed_chunk<F>(
    progress: Arc<GameProgress>,
    index: usize,
    task: F,
) -> Result<Arc<ChunkResult>>
where
    F: Future<Output = Result<ChunkResult>>,
{
    run_chunk_with_progress(&progress.chunks[index], task).await
}

async fn run_chunk_attempts_with_progress<F>(
    attempts: Vec<(usize, F)>,
    progress: Arc<GameProgress>,
    max_concurrent: usize,
) -> Result<Vec<Arc<ChunkResult>>>
where
    F: Future<Output = Result<ChunkResult>>,
{
    let tasks = attempts
        .into_iter()
        .map(|(index, task)| run_indexed_chunk(Arc::clone(&progress), index, task));
    run_chunk_tasks_until_error(tasks, max_concurrent).await
}

async fn prove_range_request(
    provider: &ProofProvider,
    keys: &ProofKeys,
    inputs: &SuperRangeInputs,
    witness: DefaultWitnessData,
    expected: &SuperRangeOutputs,
    state: &Mutex<RequestState>,
) -> Result<Arc<SP1ProofWithPublicValues>> {
    let stdin = build_super_range_stdin(inputs, witness)?;
    complete_request(
        state,
        async move || provider.request_range_proof(keys, stdin).await,
        async |proof_id| provider.wait_for_proof(proof_id).await,
        |proof| {
            provider.verify_range_proof(keys, proof)?;
            let mut public_values = proof.public_values.clone();
            let actual = decode_super_range_public_values(&mut public_values)?;
            ensure!(actual == *expected, "range proof output changed");
            Ok(())
        },
    )
    .await
}

async fn prove_consolidation_request(
    provider: &ProofProvider,
    keys: &ProofKeys,
    stdin: SP1Stdin,
    expected: &SuperConsolidationOutputs,
    state: &Mutex<RequestState>,
) -> Result<Arc<SP1ProofWithPublicValues>> {
    complete_request(
        state,
        async move || provider.request_range_proof(keys, stdin).await,
        async |proof_id| provider.wait_for_proof(proof_id).await,
        |proof| {
            provider.verify_range_proof(keys, proof)?;
            let mut public_values = proof.public_values.clone();
            let actual = decode_super_consolidation_public_values(&mut public_values)?;
            ensure!(actual == *expected, "consolidation proof output changed");
            Ok(())
        },
    )
    .await
}

/// Executes one chunk in three steps:
/// 1. collect the range and consolidation witnesses and native outputs;
/// 2. in network mode, submit or resume the two child proof requests in order;
/// 3. return the outputs and proofs for game-level aggregation.
///
/// Per-stage request state lives in `progress`. The caller caches this function's result only
/// after both child proofs complete.
async fn prove_chunk(
    attempt: &GameAttempt<'_, '_>,
    plan: ChunkPlan,
    chunk_count: usize,
    progress: Arc<GameProgress>,
) -> Result<ChunkResult> {
    let index = plan.index;
    let agreed_timestamp = plan.agreed_timestamp;
    let claimed_timestamp = plan.claimed_timestamp;
    with_proof_replay_chunk_span(
        index,
        chunk_count,
        agreed_timestamp,
        claimed_timestamp,
        async move {
            let result = prove_chunk_inner(attempt, plan, progress).await;
            match &result {
                Ok(chunk) => tracing::info!(
                    range_transition_count = chunk.range_outputs.transitions.len(),
                    consolidation_transition_count = chunk.consolidation_outputs.transitions.len(),
                    proofs_generated = chunk.proofs.is_some(),
                    "Proof replay chunk completed after validating range output-root/block bindings and consolidated super-root claims"
                ),
                Err(error) => tracing::error!(
                    error = ?error,
                    "Proof replay chunk failed"
                ),
            }
            result
        },
    )
    .await
}

async fn prove_chunk_inner(
    attempt: &GameAttempt<'_, '_>,
    plan: ChunkPlan,
    progress: Arc<GameProgress>,
) -> Result<ChunkResult> {
    let GameAttempt { provider, keys, host_inputs, game, logs, settings } = *attempt;
    check_log_batches(&logs.proofs)?;
    let index = plan.index;
    let chunk_progress = &progress.chunks[index];
    let synthesized = &plan.synthesized;
    let span = synthesized.range_inputs.span;
    let host = || {
        build_interop_host(
            host_inputs,
            game.l1_head,
            &synthesized.previous_super_root_proof_bytes,
            synthesized.current_super_root,
            span.end,
        )
    };
    let range_host = host()?;
    let configs = deployment_chain_configs(&range_host, &synthesized.range_inputs.chain_ids)?;
    let (range_witness, range_outputs) = collect_range_witness(
        range_host,
        &synthesized.range_inputs,
        &synthesized.preloaded_preimages,
        Some(&configs),
    )
    .await
    .with_context(|| format!("range witness collection failed for span {span:?}"))?;

    let range_proof = match (provider, keys) {
        (ProofProvider::Mock(_), _) => None,
        (provider, Some(keys)) => Some(
            prove_range_request(
                provider,
                keys,
                &synthesized.range_inputs,
                range_witness,
                &range_outputs,
                &chunk_progress.range_request,
            )
            .await?,
        ),
        (_, None) => bail!("network proving requires prestate proving keys"),
    };

    let batch_proofs = logs.proofs.clone().await.map_err(|err| anyhow!("{err:#}"))?;
    let batch_public_values: Vec<Vec<u8>> =
        batch_proofs.iter().map(|proof| proof.public_values.to_vec()).collect();
    let collected = collect_consolidation_witness(
        host()?,
        &synthesized.consolidation_inputs,
        &synthesized.preloaded_preimages,
        Some(&configs),
        &batch_public_values,
    )
    .await
    .with_context(|| format!("consolidation witness collection failed for span {span:?}"))?;

    let stage = &logs.stage;
    let batches: Vec<&LogBatch> = stage.batches.iter().map(|progress| &progress.batch).collect();
    let covering = cover(&batches, &collected.served)?;
    let carried: Vec<&LogBatch> = covering.iter().map(|&index| batches[index]).collect();
    let estimate =
        estimate_consolidation_pgu(&consolidation_cost(&stage.costs[index], &collected, &carried));
    if stage.table && !collected.missing.is_empty() {
        // A block read without an entry means the stage's scan is out of date or wrong.
        stage.mark_stale();
    }
    let log_commitments = LogCommitmentInputs {
        batches: covering.iter().map(|&index| batch_public_values[index].clone()).collect(),
        openings: collected.openings,
    };
    let request = CollectedRequest {
        span,
        missing: &collected.missing,
        estimate,
        witness_bytes: collected.witness_bytes,
        batch_proofs: covering.len(),
        log_commitment_bytes: log_commitments.encode().len() as u64,
    };
    check_collected(stage.table, &request, settings.range_gas_limit)?;
    tracing::info!(
        estimate,
        budget = request_budget(settings.range_gas_limit),
        batch_proofs = covering.len(),
        receipts_path_blocks = collected.missing.len(),
        witness_bytes = collected.witness_bytes,
        read_bytes = collected.read_bytes,
        "Checked the consolidation request"
    );

    let proofs = match (provider, keys, range_proof) {
        (ProofProvider::Mock(_), _, _) => None,
        (provider, Some(keys), Some(range)) => {
            let carried_proofs = LogCommitmentProofs {
                vk: (*keys.log_vk).clone(),
                proofs: covering.iter().map(|&index| (*batch_proofs[index]).clone()).collect(),
            };
            let stdin = build_super_consolidation_stdin(
                &synthesized.consolidation_inputs,
                collected.witness,
                &log_commitments,
                Some(&carried_proofs),
            )?;
            let consolidation = prove_consolidation_request(
                provider,
                keys,
                stdin,
                &collected.outputs,
                &chunk_progress.consolidation_request,
            )
            .await?;
            Some(ChildProofs { range, consolidation })
        }
        _ => bail!("network proving requires prestate proving keys"),
    };

    Ok(ChunkResult { index, range_outputs, consolidation_outputs: collected.outputs, proofs })
}

/// The batch proofs of a game, in batch order.
type BatchProofs = Arc<[Arc<SP1ProofWithPublicValues>]>;

/// The proofs of a game's batches, in batch order, shared by its chunk tasks.
type LogBatchProofs<'a> = Shared<BoxFuture<'a, Result<BatchProofs, Arc<anyhow::Error>>>>;

/// The log-commitment stage of a game: its path and, on the table path, its batches. It serves
/// later attempts until a collection reads a block without an entry.
#[derive(Default)]
struct LogStage {
    table: bool,
    /// The receipts-path costs of each chunk's transitions, by chunk index.
    costs: Vec<Vec<ScannedTimestampCost>>,
    /// The table path's batches; empty on the receipts path.
    batches: Vec<LogBatchProgress>,
    /// Set when the next attempt must scan the span again.
    stale: AtomicBool,
}

impl LogStage {
    fn mark_stale(&self) {
        self.stale.store(true, Ordering::Relaxed);
    }

    fn is_stale(&self) -> bool {
        self.stale.load(Ordering::Relaxed)
    }
}

/// The log-commitment stage of one game attempt.
struct GameLogs<'a> {
    stage: Arc<LogStage>,
    /// Chunk tasks await these before their consolidation collection.
    proofs: LogBatchProofs<'a>,
}

impl GameLogs<'_> {
    fn without_batches(stage: Arc<LogStage>) -> Self {
        Self { stage, proofs: futures::future::ready(Ok(BatchProofs::from([]))).boxed().shared() }
    }
}

/// What the chunk tasks of one game attempt share.
#[derive(Clone, Copy)]
struct GameAttempt<'a, 'b> {
    provider: &'a ProofProvider,
    keys: Option<&'a ProofKeys>,
    host_inputs: &'a HostInputs,
    game: &'a GameProofInputs,
    logs: &'b GameLogs<'a>,
    settings: &'a LogProofSettings,
}

/// The game's log stage, from `progress` or from a new scan, with every batch request submitted
/// before any wait.
async fn start_game_logs<'a>(
    provider: &'a ProofProvider,
    keys: Option<&'a ProofKeys>,
    host_inputs: &HostInputs,
    request: &ProveGameRequest<'a>,
    plans: &[ChunkPlan],
    progress: &GameProgress,
) -> Result<GameLogs<'a>> {
    let stage = match progress.log_stage() {
        Some(stage) if !stage.is_stale() => stage,
        previous => progress.store_log_stage(
            scan_log_stage(host_inputs, request, plans, previous.as_deref()).await?,
        ),
    };
    if !stage.table {
        return Ok(GameLogs::without_batches(stage));
    }
    let prover: Arc<dyn LogBatchProver + 'a> = match (provider, keys) {
        (ProofProvider::Mock(_), _) => Arc::new(MockLogProver),
        (provider, Some(keys)) => Arc::new(NetworkLogProver {
            provider,
            keys,
            l2: request.l2,
            poll_interval: log_batch_poll_interval(stage.batches.len()),
        }),
        (_, None) => bail!("network proving requires prestate proving keys"),
    };
    let proofs = start_log_batches(prover, Arc::clone(&stage)).await?;
    Ok(GameLogs { stage, proofs })
}

/// Scans the span, chooses the game's path, and on the table path makes the batches, with the
/// requests of `previous` (see [`batch_progress`]).
async fn scan_log_stage(
    host_inputs: &HostInputs,
    request: &ProveGameRequest<'_>,
    plans: &[ChunkPlan],
    previous: Option<&LogStage>,
) -> Result<LogStage> {
    let first = plans.first().context("game proof has no chunks")?;
    let host = build_interop_host(
        host_inputs,
        request.game.l1_head,
        &first.synthesized.previous_super_root_proof_bytes,
        first.synthesized.current_super_root,
        first.claimed_timestamp,
    )?;
    let configs = deployment_chain_configs(&host, &first.synthesized.range_inputs.chain_ids)?;
    let scan = scan_span(request.l2, request.responses, &configs)
        .await
        .context("failed to scan the span's executing messages")?;
    let costs: Vec<_> =
        plans.iter().map(|plan| receipts_path_costs(&scan, plan.first, plan.last)).collect();
    let bounds: Vec<RequestBound> = costs.iter().map(|costs| RequestBound::of(costs)).collect();
    let settings = request.log;
    let choice = choose_log_path(settings.path, &bounds, settings.range_gas_limit);
    tracing::info!(
        table = choice.table,
        setting = ?settings.path,
        largest_receipts_path_bound = choice.largest_bound,
        budget = request_budget(settings.range_gas_limit),
        referenced_blocks = scan.blocks.len(),
        "Chose the consolidation path"
    );
    let batches = if choice.table {
        // TODO(proofs-team#31): reuse the batches of earlier games from a persistent cache keyed
        // by receipts root and log vkey.
        let batches = make_log_batches(request.l2, &scan, settings.log_gas_limit).await?;
        batch_progress(batches, previous)
    } else {
        Vec::new()
    };
    Ok(LogStage { table: choice.table, costs, batches, stale: AtomicBool::new(false) })
}

/// The progress of `batches`. A batch of `previous` with the same ID keeps its request.
fn batch_progress(batches: Vec<LogBatch>, previous: Option<&LogStage>) -> Vec<LogBatchProgress> {
    let mut requests: HashMap<B256, RequestState> = previous
        .iter()
        .flat_map(|previous| &previous.batches)
        .map(|progress| (progress.batch.id, progress.request.lock().clone()))
        .collect();
    batches
        .into_iter()
        .map(|batch| {
            let request = requests.remove(&batch.id).unwrap_or_default();
            LogBatchProgress { batch, request: Mutex::new(request) }
        })
        .collect()
}

/// Submits every batch request of `stage` that is not submitted yet, and returns the wait for
/// their proofs.
async fn start_log_batches<'a>(
    prover: Arc<dyn LogBatchProver + 'a>,
    stage: Arc<LogStage>,
) -> Result<LogBatchProofs<'a>> {
    submit_log_batches(prover.as_ref(), &stage.batches).await?;
    Ok(async move { wait_for_log_batches(prover.as_ref(), &stage.batches).await.map_err(Arc::new) }
        .boxed()
        .shared())
}

/// Fails once the batch proofs failed, so a chunk admitted after a batch failure does no work.
fn check_log_batches(proofs: &LogBatchProofs<'_>) -> Result<()> {
    if let Some(Err(err)) = proofs.peek() {
        bail!("{err:#}");
    }
    Ok(())
}

/// Runs `chunks` while it polls the batch proofs, so batch requests progress during the first
/// range proofs. After a batch failure, the admitted chunks run to their end and the attempt
/// returns the batch error.
async fn run_with_log_batches<T>(
    chunks: impl Future<Output = Result<T>>,
    mut proofs: LogBatchProofs<'_>,
) -> Result<T> {
    tokio::pin!(chunks);
    let mut batch_error = None;
    let mut waiting = true;
    let result = loop {
        tokio::select! {
            result = &mut chunks => break result,
            batches = &mut proofs, if waiting => {
                waiting = false;
                batch_error = batches.err();
            }
        }
    };
    batch_error.map_or(result, |err| Err(anyhow!("{err:#}")))
}

/// Selects cached progress and rejects terminal attempts before returning executable chunks.
fn plan_game_attempt(
    provider: &ProofProvider,
    keys: Option<&ProofKeys>,
    request: &ProveGameRequest<'_>,
) -> Result<(Vec<ChunkPlan>, Arc<GameProgress>)> {
    let ProveGameRequest { game_address, game, responses, split, proof_progress, .. } = *request;
    let chunks = split.split(game.starting_ts, game.claim_ts)?;
    let mut plans = Vec::with_capacity(chunks.len());
    let mut chunk_identities = Vec::with_capacity(chunks.len());
    for (index, (agreed_ts, claimed_ts)) in chunks.into_iter().enumerate() {
        let first = (agreed_ts - game.starting_ts) as usize;
        let last = (claimed_ts - game.starting_ts) as usize;
        let span = TimestampSpan::new(agreed_ts + 1, claimed_ts)
            .map_err(|err| anyhow!("invalid chunk span: {err:?}"))?;
        let synthesized =
            synthesize_execution(span, game.l1_head, game.l1_head_number, &responses[first..=last])
                .with_context(|| {
                    format!("failed to synthesize span {}..={}", span.start, span.end)
                })?;
        chunk_identities.push(ChunkIdentity {
            index: u8::try_from(index).context("chunk index exceeds u8")?,
            agreed_ts,
            claimed_ts,
            range_input_digest: fingerprint(&synthesized.range_inputs)?,
            consolidation_input_digest: fingerprint(&synthesized.consolidation_inputs)?,
        });
        plans.push(ChunkPlan {
            index,
            agreed_timestamp: agreed_ts,
            claimed_timestamp: claimed_ts,
            first,
            last,
            synthesized,
        });
    }
    let identity = AttemptIdentity {
        game: game.clone(),
        provider: provider_identity(provider, keys)?,
        chunks: chunk_identities,
    };
    let progress = proof_progress.load_or_create(game_address, identity);
    progress.check_terminal_requests()?;
    Ok((plans, progress))
}

/// Log-commitment settings of game proofs.
#[derive(Debug, Clone)]
pub(crate) struct LogProofSettings {
    pub(crate) path: LogPath,
    /// The range and consolidation request gas limit.
    pub(crate) range_gas_limit: u64,
    pub(crate) log_gas_limit: u64,
}

/// Inputs for one in-process game-proving attempt.
pub(crate) struct ProveGameRequest<'a> {
    /// Dispute game whose work owns the cache entry.
    pub(crate) game_address: Address,
    /// Immutable inputs bound into the proof.
    pub(crate) game: &'a GameProofInputs,
    /// Validated super-root responses covering the game span.
    pub(crate) responses: &'a [SuperRootAtTimestampResponse],
    /// Number of chunks used for the span.
    pub(crate) split: RangeSplitCount,
    /// Maximum number of concurrently active chunk tasks.
    pub(crate) max_concurrent: NonZeroUsize,
    /// Process-local progress retained between scheduler attempts.
    pub(crate) proof_progress: &'a InMemoryProofProgress,
    /// L2 reads for the message scan and the log-commitment batches.
    pub(crate) l2: &'a dyn L2BlockSource,
    pub(crate) log: &'a LogProofSettings,
}

/// Proves the game's span end to end and returns the on-chain proof bytes.
///
/// `keys` must be `Some` in network mode (resolved per-prestate through the
/// prestate cache) and may be `None` in mock mode.
pub(crate) async fn prove_game_inner(
    provider: &ProofProvider,
    keys: Option<&ProofKeys>,
    host_inputs: &HostInputs,
    request: ProveGameRequest<'_>,
) -> Result<Vec<u8>> {
    if !provider.is_mock() {
        ensure!(keys.is_some(), "network proving requires prestate proving keys");
    }
    let (plans, progress) = plan_game_attempt(provider, keys, &request)?;
    let ProveGameRequest { game, responses, max_concurrent, log, .. } = request;
    let chunk_count = plans.len();
    tracing::info!(
        starting_ts = game.starting_ts,
        claim_ts = game.claim_ts,
        chunks = chunk_count,
        "Proving game span"
    );

    if !provider.is_mock() &&
        let Some(proof) =
            reuse_fulfilled_aggregation(&progress.aggregation_request, game, |proof| {
                let keys = keys.context("network proving requires prestate proving keys")?;
                provider.verify_aggregation_proof(keys, proof)
            })?
    {
        return Ok(proof);
    }

    let logs = if progress.chunks.iter().all(|chunk| chunk.completed.lock().is_some()) {
        GameLogs::without_batches(Arc::default())
    } else {
        start_game_logs(provider, keys, host_inputs, &request, &plans, &progress).await?
    };
    let attempt = GameAttempt { provider, keys, host_inputs, game, logs: &logs, settings: log };
    let attempts = plans
        .into_iter()
        .map(|plan| {
            let index = plan.index;
            (index, prove_chunk(&attempt, plan, chunk_count, Arc::clone(&progress)))
        })
        .collect();

    let max_concurrent = max_concurrent.get().min(chunk_count);
    let results = run_with_log_batches(
        run_chunk_attempts_with_progress(attempts, Arc::clone(&progress), max_concurrent),
        logs.proofs.clone(),
    )
    .await?;

    let mut range_outputs = vec![None; chunk_count];
    let mut consolidation_outputs = vec![None; chunk_count];
    let mut proofs = vec![None; chunk_count];
    for result in results {
        range_outputs[result.index] = Some(result.range_outputs.clone());
        consolidation_outputs[result.index] = Some(result.consolidation_outputs.clone());
        proofs[result.index] = result.proofs.clone();
    }
    let range_outputs = collect_indexed(range_outputs, "range outputs")?;
    let consolidation_outputs = collect_indexed(consolidation_outputs, "consolidation outputs")?;

    let starting_data = responses
        .first()
        .and_then(|response| response.data.as_ref())
        .context("verified span response lost its data")?;
    let inputs = SuperAggregationInputs {
        l1_head: game.l1_head,
        starting_root_hash: game.starting_root,
        starting_super_root_proof: proof_from_super_v1(&starting_data.super_v1)?,
        root_claim: game.root_claim,
        l2_sequence_number: game.claim_ts,
        prover: game.prover,
        range_outputs,
        consolidation_outputs,
    };
    inputs.validate().with_context(|| {
        format!(
            "assembled aggregation inputs failed validation for span {}..={}",
            game.starting_ts, game.claim_ts
        )
    })?;

    match provider {
        ProofProvider::Mock(_) => {
            tracing::debug!(
                public_values = %alloy_primitives::hex::encode(inputs.zk_dispute_game_public_values()),
                "Mock provider: submitting placeholder proof bytes"
            );
            Ok(MOCK_PROOF_BYTES.to_vec())
        }
        ProofProvider::Network(_) => {
            let keys = keys.context("network proving requires prestate proving keys")?;
            let mut stdin = SP1Stdin::new();
            let (range_proofs, consolidation_proofs): (Vec<_>, Vec<_>) = proofs
                .into_iter()
                .enumerate()
                .map(|(index, proofs)| {
                    let proofs =
                        proofs.ok_or_else(|| anyhow!("missing proofs for chunk {index}"))?;
                    Ok((proofs.range, proofs.consolidation))
                })
                .collect::<Result<Vec<_>>>()?
                .into_iter()
                .unzip();
            for proof in range_proofs.into_iter().chain(consolidation_proofs) {
                let SP1Proof::Compressed(compressed) = &proof.proof else {
                    bail!("child proof is not in compressed form");
                };
                stdin.write_proof((**compressed).clone(), keys.range_vk.vk.clone());
            }
            stdin.write(&inputs);

            let expected = inputs.zk_dispute_game_public_values();
            let aggregation = complete_request(
                &progress.aggregation_request,
                async move || provider.request_aggregation_proof(keys, stdin).await,
                async |proof_id| provider.wait_for_proof(proof_id).await,
                |proof| {
                    provider.verify_aggregation_proof(keys, proof)?;
                    ensure!(
                        proof.public_values.as_slice() == expected.as_slice(),
                        "aggregation proof public values do not match the game's expected tuple"
                    );
                    Ok(())
                },
            )
            .await?;
            Ok(aggregation.bytes())
        }
    }
}

/// Unwraps a fully populated index-placed vector, erroring on gaps.
fn collect_indexed<T>(items: Vec<Option<T>>, what: &str) -> Result<Vec<T>> {
    items
        .into_iter()
        .enumerate()
        .map(|(index, item)| item.ok_or_else(|| anyhow!("missing {what} for chunk {index}")))
        .collect()
}

#[cfg(test)]
mod tests {
    use std::{
        collections::{BTreeMap, HashMap},
        fmt::Write,
        path::Path,
        sync::{
            Arc, Mutex as StdMutex,
            atomic::{AtomicBool, AtomicUsize, Ordering},
        },
    };

    use alloy_primitives::{Address, U256, keccak256};
    use futures::StreamExt;
    use kona_sp1_client_utils::test_utils::valid_aggregation_inputs;
    use kona_sp1_super_range_executor::{
        BlockId, ChainId, ChainIdAndOutput, OutputV0, OutputWithRequiredL1, SuperRootResponseData,
        SuperV1,
    };
    use sp1_sdk::SP1PublicValues;
    use tracing::{
        Event, Instrument, Level, Metadata, Subscriber,
        field::{Field, Visit},
        span::{Attributes, Id, Record},
    };

    use super::*;

    #[derive(Debug, Default)]
    struct CapturedSpan {
        name: String,
        parent: Option<u64>,
        fields: String,
        level: Option<Level>,
    }

    #[derive(Debug, Default)]
    struct TraceCapture {
        next_id: u64,
        spans: HashMap<u64, CapturedSpan>,
        entered: Vec<u64>,
        event_spans: Vec<u64>,
    }

    #[derive(Debug)]
    struct CapturingSubscriber(Arc<StdMutex<TraceCapture>>);

    struct FieldVisitor<'a>(&'a mut String);

    impl Visit for FieldVisitor<'_> {
        fn record_debug(&mut self, field: &Field, value: &dyn std::fmt::Debug) {
            write!(self.0, "{}={value:?} ", field.name()).unwrap();
        }
    }

    impl Subscriber for CapturingSubscriber {
        fn enabled(&self, _metadata: &Metadata<'_>) -> bool {
            true
        }

        fn new_span(&self, attributes: &Attributes<'_>) -> Id {
            let mut capture = self.0.lock().unwrap();
            capture.next_id += 1;
            let id = capture.next_id;
            let parent = attributes
                .parent()
                .map(|parent| parent.clone().into_u64())
                .or_else(|| capture.entered.last().copied());
            let mut span = CapturedSpan {
                name: attributes.metadata().name().to_string(),
                parent,
                level: Some(*attributes.metadata().level()),
                ..Default::default()
            };
            attributes.record(&mut FieldVisitor(&mut span.fields));
            capture.spans.insert(id, span);
            Id::from_u64(id)
        }

        fn record(&self, _span: &Id, _values: &Record<'_>) {}

        fn record_follows_from(&self, _span: &Id, _follows: &Id) {}

        fn event(&self, _event: &Event<'_>) {
            let mut capture = self.0.lock().unwrap();
            let current = capture.entered.last().copied().unwrap();
            capture.event_spans.push(current);
        }

        fn enter(&self, span: &Id) {
            self.0.lock().unwrap().entered.push(span.clone().into_u64());
        }

        fn exit(&self, span: &Id) {
            assert_eq!(self.0.lock().unwrap().entered.pop(), Some(span.clone().into_u64()));
        }
    }

    /// An L2 source for tests that make no L2 reads.
    struct NoL2;

    #[async_trait::async_trait]
    impl L2BlockSource for NoL2 {
        async fn header_by_hash(
            &self,
            _chain_id: u64,
            _hash: B256,
        ) -> Result<alloy_consensus::Header> {
            bail!("no L2 reads in this test")
        }

        async fn header_by_number(
            &self,
            _chain_id: u64,
            _number: u64,
        ) -> Result<alloy_consensus::Sealed<alloy_consensus::Header>> {
            bail!("no L2 reads in this test")
        }

        async fn receipts(
            &self,
            _chain_id: u64,
            _hash: B256,
            _receipts_root: B256,
        ) -> Result<Vec<op_alloy_consensus::OpReceiptEnvelope>> {
            bail!("no L2 reads in this test")
        }
    }

    fn log_settings() -> LogProofSettings {
        LogProofSettings { path: LogPath::Auto, range_gas_limit: 1, log_gas_limit: 1 }
    }

    /// Builds a self-consistent supernode response trusted by default.
    fn response_at(
        timestamp: u64,
        output_byte: u8,
        required_l1: u64,
    ) -> SuperRootAtTimestampResponse {
        use kona_sp1_client_utils::super_root::hash_super_root_proof;

        let chain_id = ChainId(U256::from(10));
        let output = OutputV0 {
            state_root: B256::repeat_byte(0x11),
            message_passer_storage_root: B256::repeat_byte(0x22),
            block_hash: B256::repeat_byte(output_byte),
        };
        let mut preimage = vec![0u8; 128];
        preimage[32..64].copy_from_slice(output.state_root.as_slice());
        preimage[64..96].copy_from_slice(output.message_passer_storage_root.as_slice());
        preimage[96..128].copy_from_slice(output.block_hash.as_slice());
        let output_root = keccak256(preimage);
        let super_v1 =
            SuperV1 { timestamp, chains: vec![ChainIdAndOutput { chain_id, output: output_root }] };
        let proof = proof_from_super_v1(&super_v1).unwrap();
        let super_root = B256::from(*hash_super_root_proof(&proof).unwrap());
        SuperRootAtTimestampResponse {
            current_l1: BlockId { number: required_l1 + 1, ..Default::default() },
            current_safe_timestamp: timestamp,
            current_local_safe_timestamp: timestamp,
            current_finalized_timestamp: timestamp,
            optimistic_at_timestamp: BTreeMap::from([(
                chain_id,
                OutputWithRequiredL1 {
                    output: Some(output),
                    output_root,
                    required_l1: BlockId { number: required_l1, ..Default::default() },
                },
            )]),
            chain_ids: vec![chain_id],
            data: Some(SuperRootResponseData {
                verified_required_l1: BlockId { number: required_l1, ..Default::default() },
                super_v1,
                super_root,
            }),
        }
    }

    /// Marks a response untrusted: the supernode has not re-verified its
    /// answer (`current_l1` below `verified_required_l1`, e.g. mid-rewind).
    fn untrusted(mut response: SuperRootAtTimestampResponse) -> SuperRootAtTimestampResponse {
        response.current_l1.number =
            response.data.as_ref().unwrap().verified_required_l1.number.saturating_sub(1);
        response
    }

    /// Sets `current_l1` equal to `verified_required_l1`. The API may still
    /// report incomplete data at that block, so the response is untrusted.
    fn at_required_l1(mut response: SuperRootAtTimestampResponse) -> SuperRootAtTimestampResponse {
        response.current_l1.number = response.data.as_ref().unwrap().verified_required_l1.number;
        response
    }

    fn root_of(response: &SuperRootAtTimestampResponse) -> B256 {
        response.data.as_ref().unwrap().super_root
    }

    fn game_for(
        start: &SuperRootAtTimestampResponse,
        end: &SuperRootAtTimestampResponse,
        l1_head_number: u64,
    ) -> GameProofInputs {
        GameProofInputs {
            l1_head: B256::repeat_byte(0x11),
            l1_head_number,
            starting_root: root_of(start),
            starting_ts: start.data.as_ref().unwrap().super_v1.timestamp,
            root_claim: root_of(end),
            claim_ts: end.data.as_ref().unwrap().super_v1.timestamp,
            prestate: B256::repeat_byte(0x22),
            prover: Address::repeat_byte(0x33),
        }
    }

    fn attempt_identity(game: GameProofInputs) -> AttemptIdentity {
        AttemptIdentity {
            game,
            provider: ProviderIdentity::Mock,
            chunks: (0..2)
                .map(|index| ChunkIdentity {
                    index,
                    agreed_ts: u64::from(index) + 100,
                    claimed_ts: u64::from(index) + 101,
                    range_input_digest: B256::repeat_byte(index + 0x44),
                    consolidation_input_digest: B256::repeat_byte(index + 0x46),
                })
                .collect(),
        }
    }

    fn chunk_result(index: usize) -> ChunkResult {
        let inputs = valid_aggregation_inputs();
        ChunkResult {
            index,
            range_outputs: inputs.range_outputs[index].clone(),
            consolidation_outputs: inputs.consolidation_outputs[index].clone(),
            proofs: None,
        }
    }

    fn fulfilled_proof() -> SP1ProofWithPublicValues {
        SP1ProofWithPublicValues {
            proof: SP1Proof::Core(Vec::new()),
            public_values: SP1PublicValues::new(),
            sp1_version: String::new(),
            tee_proof: None,
        }
    }

    #[test]
    fn fulfilled_aggregation_is_reverified_before_reuse() {
        let start = response_at(100, 0x01, 5);
        let end = response_at(101, 0x02, 5);
        let game = game_for(&start, &end, 10);
        let expected = expected_aggregation_public_values(&game);
        let proof = SP1ProofWithPublicValues {
            proof: SP1Proof::Plonk(Default::default()),
            public_values: SP1PublicValues::from(&expected),
            sp1_version: String::new(),
            tee_proof: None,
        };
        let proof_bytes = proof.bytes();
        let state = Mutex::new(RequestState::Fulfilled(Arc::new(proof)));
        let verified = AtomicBool::new(false);

        let reused = reuse_fulfilled_aggregation(&state, &game, |_| {
            verified.store(true, Ordering::SeqCst);
            Ok(())
        })
        .unwrap()
        .unwrap();

        assert!(verified.load(Ordering::SeqCst));
        assert_eq!(reused, proof_bytes);

        let mut changed_game = game;
        changed_game.root_claim = B256::repeat_byte(0x03);
        let err = reuse_fulfilled_aggregation(&state, &changed_game, |_| Ok(())).unwrap_err();
        assert!(err.to_string().contains("public values do not match"));
    }

    #[test]
    fn terminal_attempt_retries_only_after_identity_change_or_reset() {
        let start = response_at(100, 0x01, 5);
        let middle = response_at(101, 0x02, 5);
        let end = response_at(102, 0x03, 5);
        let game = game_for(&start, &end, 10);
        let responses = vec![start.clone(), middle, end.clone()];
        let provider = ProofProvider::Mock(crate::prover::MockProofProvider);
        let cache = InMemoryProofProgress::default();
        let request = ProveGameRequest {
            game_address: Address::repeat_byte(0x45),
            game: &game,
            responses: &responses,
            split: RangeSplitCount::one(),
            max_concurrent: NonZeroUsize::MIN,
            proof_progress: &cache,
            l2: &NoL2,
            log: &log_settings(),
        };
        let (_, progress) = plan_game_attempt(&provider, None, &request).unwrap();
        *progress.chunks[0].range_request.lock() =
            RequestState::Terminal(ProofTerminalState::Unexecutable);
        assert!(plan_game_attempt(&provider, None, &request).is_err());

        let changed_responses = vec![start, response_at(101, 0x04, 5), end];
        let request = ProveGameRequest { responses: &changed_responses, ..request };
        let (_, progress) = plan_game_attempt(&provider, None, &request).unwrap();
        *progress.chunks[0].range_request.lock() =
            RequestState::Terminal(ProofTerminalState::ValidationFailed);
        assert!(plan_game_attempt(&provider, None, &request).is_err());

        assert_eq!(cache.retry_terminal_requests(request.game_address), 1);
        let (_, progress) = plan_game_attempt(&provider, None, &request).unwrap();
        *progress.chunks[0].range_request.lock() =
            RequestState::Terminal(ProofTerminalState::Unexecutable);
        assert!(plan_game_attempt(&provider, None, &request).is_err());
    }

    #[test]
    fn terminal_preflight_checks_later_children_and_aggregation() {
        let responses =
            vec![response_at(100, 1, 5), response_at(101, 2, 5), response_at(102, 3, 5)];
        let game = game_for(&responses[0], &responses[2], 10);
        let provider = ProofProvider::Mock(crate::prover::MockProofProvider);
        for (slot, state) in [
            (2, ProofTerminalState::Unexecutable),
            (3, ProofTerminalState::ValidationFailed),
            (4, ProofTerminalState::Unexecutable),
        ] {
            let cache = InMemoryProofProgress::default();
            let request = ProveGameRequest {
                game_address: Address::repeat_byte(0x45),
                game: &game,
                responses: &responses,
                split: RangeSplitCount::new(2).unwrap(),
                max_concurrent: NonZeroUsize::MIN,
                proof_progress: &cache,
                l2: &NoL2,
                log: &log_settings(),
            };
            let (_, progress) = plan_game_attempt(&provider, None, &request).unwrap();
            let (_, _, request_state) = progress.requests().nth(slot).unwrap();
            *request_state.lock() = RequestState::Terminal(state);
            let error = plan_game_attempt(&provider, None, &request)
                .err()
                .expect("terminal attempts must not return executable chunk plans");
            assert!(!is_unprovable(&error), "{error}");
        }
    }

    #[tokio::test]
    async fn terminal_reset_preserves_reusable_work() {
        let start = response_at(100, 1, 5);
        let end = response_at(102, 3, 5);
        let game = game_for(&start, &end, 10);
        let address = Address::repeat_byte(0x45);
        let cache = InMemoryProofProgress::default();
        let identity = attempt_identity(game);
        let progress = cache.load_or_create(address, identity.clone());
        run_chunk_with_progress(&progress.chunks[0], async { Ok(chunk_result(0)) }).await.unwrap();
        let mut proof = fulfilled_proof();
        proof.public_values = SP1PublicValues::from(&[0x42]);
        let proof = Arc::new(proof);
        *progress.chunks[0].range_request.lock() = RequestState::Fulfilled(proof.clone());
        *progress.chunks[0].consolidation_request.lock() = RequestState::Fulfilled(proof);
        let pending_id = B256::repeat_byte(0x46);
        *progress.chunks[1].range_request.lock() = RequestState::Submitted(pending_id);
        *progress.chunks[1].consolidation_request.lock() =
            RequestState::Terminal(ProofTerminalState::Unexecutable);
        *progress.aggregation_request.lock() =
            RequestState::Terminal(ProofTerminalState::ValidationFailed);

        assert_eq!(cache.retry_terminal_requests(address), 2);
        let progress = cache.load_or_create(address, identity);
        run_chunk_with_progress(&progress.chunks[0], async { panic!("completed chunk was lost") })
            .await
            .unwrap();
        let reused = complete_request(
            &progress.chunks[0].range_request,
            async || panic!("fulfilled request was resubmitted"),
            async |_| panic!("fulfilled request was polled"),
            |_| Ok(()),
        )
        .await
        .unwrap();
        assert_eq!(reused.public_values.as_slice(), &[0x42]);
        complete_request(
            &progress.chunks[1].range_request,
            async || panic!("pending request was resubmitted"),
            async |id| {
                assert_eq!(id, pending_id);
                Ok(fulfilled_proof())
            },
            |_| Ok(()),
        )
        .await
        .unwrap();
        let submissions = AtomicUsize::new(0);
        for slot in [&progress.chunks[1].consolidation_request, &progress.aggregation_request] {
            complete_request(
                slot,
                async || {
                    submissions.fetch_add(1, Ordering::Relaxed);
                    Ok(B256::repeat_byte(0x47))
                },
                async |_| Ok(fulfilled_proof()),
                |_| Ok(()),
            )
            .await
            .unwrap();
        }
        assert_eq!(submissions.load(Ordering::Relaxed), 2);
    }

    #[test]
    fn terminal_reset_preserves_fulfilled_aggregation() {
        let start = response_at(100, 1, 5);
        let end = response_at(102, 3, 5);
        let game = game_for(&start, &end, 10);
        let address = Address::repeat_byte(0x48);
        let cache = InMemoryProofProgress::default();
        let identity = attempt_identity(game.clone());
        let progress = cache.load_or_create(address, identity.clone());
        let expected = expected_aggregation_public_values(&game);
        let proof = SP1ProofWithPublicValues {
            proof: SP1Proof::Plonk(Default::default()),
            public_values: SP1PublicValues::from(&expected),
            ..fulfilled_proof()
        };
        let bytes = proof.bytes();
        *progress.aggregation_request.lock() = RequestState::Fulfilled(Arc::new(proof));
        assert_eq!(cache.retry_terminal_requests(address), 0);
        let progress = cache.load_or_create(address, identity);
        assert_eq!(
            reuse_fulfilled_aggregation(&progress.aggregation_request, &game, |_| Ok(())).unwrap(),
            Some(bytes),
        );
    }

    #[test]
    fn network_vkey_change_replaces_progress() {
        let start = response_at(100, 0x01, 5);
        let end = response_at(101, 0x02, 5);
        let game = game_for(&start, &end, 10);
        let game_address = Address::repeat_byte(0x46);
        let cache = InMemoryProofProgress::default();
        let mut identity = attempt_identity(game);
        identity.provider = ProviderIdentity::Network {
            range_vkey: B256::repeat_byte(0x10),
            aggregation_vkey: B256::repeat_byte(0x11),
            log_vkey: [0x12; 8],
        };
        let first = cache.load_or_create(game_address, identity.clone());

        identity.provider = ProviderIdentity::Network {
            range_vkey: B256::repeat_byte(0x12),
            aggregation_vkey: B256::repeat_byte(0x11),
            log_vkey: [0x12; 8],
        };
        let changed = cache.load_or_create(game_address, identity);

        assert!(!Arc::ptr_eq(&first, &changed));
    }

    #[test]
    fn request_summary_reports_abandoned_work() {
        let start = response_at(100, 0x01, 5);
        let end = response_at(101, 0x02, 5);
        let progress = GameProgress::new(attempt_identity(game_for(&start, &end, 10)));
        let first_id = B256::repeat_byte(0x13);
        let second_id = B256::repeat_byte(0x14);
        *progress.chunks[0].range_request.lock() = RequestState::Submitted(first_id);
        *progress.chunks[0].consolidation_request.lock() = RequestState::Submitting;
        *progress.chunks[1].range_request.lock() =
            RequestState::Fulfilled(Arc::new(fulfilled_proof()));
        *progress.chunks[1].consolidation_request.lock() = RequestState::Submitted(second_id);
        *progress.aggregation_request.lock() = RequestState::Fulfilled(Arc::new(fulfilled_proof()));

        assert_eq!(
            progress.request_summary(),
            RequestSummary {
                submitted_ids: vec![first_id, second_id],
                submitting: 1,
                fulfilled: 2,
            }
        );
    }

    #[tokio::test]
    async fn retry_reuses_completed_chunks_and_changed_identity_invalidates_them() {
        let start = response_at(100, 0x01, 5);
        let end = response_at(101, 0x02, 5);
        let game = game_for(&start, &end, 10);
        let address = Address::repeat_byte(0x46);
        let cache = InMemoryProofProgress::default();
        let generations = Arc::new([AtomicUsize::new(0), AtomicUsize::new(0)]);
        let first_progress = cache.load_or_create(address, attempt_identity(game.clone()));
        let first_done = Arc::new(tokio::sync::Notify::new());
        let first_attempt = (0..2)
            .map(|index| {
                let generations = Arc::clone(&generations);
                let first_done = Arc::clone(&first_done);
                (index, async move {
                    if index == 1 {
                        first_done.notified().await;
                        generations[index].fetch_add(1, Ordering::SeqCst);
                        Err(anyhow!("chunk B failed"))
                    } else {
                        generations[index].fetch_add(1, Ordering::SeqCst);
                        let result = chunk_result(index);
                        first_done.notify_one();
                        Ok(result)
                    }
                })
            })
            .collect();

        let first =
            run_chunk_attempts_with_progress(first_attempt, Arc::clone(&first_progress), 2).await;
        assert_eq!(first.err().unwrap().to_string(), "chunk B failed");
        assert_eq!(generations[0].load(Ordering::SeqCst), 1);
        assert_eq!(generations[1].load(Ordering::SeqCst), 1);

        let retry = (0..2)
            .map(|index| {
                let generations = Arc::clone(&generations);
                (index, async move {
                    generations[index].fetch_add(1, Ordering::SeqCst);
                    Ok(chunk_result(index))
                })
            })
            .collect();
        run_chunk_attempts_with_progress(retry, Arc::clone(&first_progress), 2).await.unwrap();
        assert_eq!(generations[0].load(Ordering::SeqCst), 1);
        assert_eq!(generations[1].load(Ordering::SeqCst), 2);

        let mut changed = game;
        changed.root_claim = B256::repeat_byte(0x47);
        let changed_progress = cache.load_or_create(address, attempt_identity(changed));
        let changed_attempt = (0..2)
            .map(|index| {
                let generations = Arc::clone(&generations);
                (index, async move {
                    generations[index].fetch_add(1, Ordering::SeqCst);
                    Ok(chunk_result(index))
                })
            })
            .collect();
        run_chunk_attempts_with_progress(changed_attempt, changed_progress, 2).await.unwrap();
        assert_eq!(generations[0].load(Ordering::SeqCst), 2);
        assert_eq!(generations[1].load(Ordering::SeqCst), 3);
    }

    #[test]
    fn changed_identity_replaces_submitted_request_for_liveness() {
        let start = response_at(100, 0x01, 5);
        let end = response_at(101, 0x02, 5);
        let game = game_for(&start, &end, 10);
        let address = Address::repeat_byte(0x48);
        let cache = InMemoryProofProgress::default();
        let progress = cache.load_or_create(address, attempt_identity(game.clone()));
        *progress.chunks[0].range_request.lock() = RequestState::Submitted(B256::repeat_byte(0x49));

        let mut changed = game;
        changed.root_claim = B256::repeat_byte(0x4a);
        let replacement = cache.load_or_create(address, attempt_identity(changed));

        assert!(!Arc::ptr_eq(&progress, &replacement));
        assert!(matches!(*replacement.chunks[0].range_request.lock(), RequestState::Missing));
    }

    #[tokio::test]
    async fn submitted_request_id_is_polled_without_resubmission() {
        let state = Mutex::new(RequestState::Missing);
        let submissions = AtomicUsize::new(0);
        let polls = AtomicUsize::new(0);
        let proof_id = B256::repeat_byte(0x4b);

        let first = complete_request(
            &state,
            async || {
                submissions.fetch_add(1, Ordering::SeqCst);
                Ok(proof_id)
            },
            async |observed| {
                assert_eq!(observed, proof_id);
                polls.fetch_add(1, Ordering::SeqCst);
                Err(ProofWaitError::Uncertain(anyhow!("temporary polling failure")))
            },
            |_| Ok(()),
        )
        .await;
        assert!(first.is_err());

        let proof = complete_request(
            &state,
            async || {
                submissions.fetch_add(1, Ordering::SeqCst);
                Ok(B256::repeat_byte(0xff))
            },
            async |observed| {
                assert_eq!(observed, proof_id);
                polls.fetch_add(1, Ordering::SeqCst);
                Ok(fulfilled_proof())
            },
            |_| Ok(()),
        )
        .await
        .unwrap();
        let cached = complete_request(
            &state,
            async || {
                submissions.fetch_add(1, Ordering::SeqCst);
                Ok(B256::repeat_byte(0xfe))
            },
            async |_| {
                polls.fetch_add(1, Ordering::SeqCst);
                Ok(fulfilled_proof())
            },
            |_| Ok(()),
        )
        .await
        .unwrap();

        assert!(Arc::ptr_eq(&proof, &cached));

        assert!(proof.public_values.as_slice().is_empty());
        assert_eq!(submissions.load(Ordering::SeqCst), 1);
        assert_eq!(polls.load(Ordering::SeqCst), 2);
    }

    #[tokio::test]
    async fn settled_terminal_request_can_be_replaced() {
        let proof_id = B256::repeat_byte(0x4c);
        let replacement_id = B256::repeat_byte(0x4d);
        let state = Mutex::new(RequestState::Submitted(proof_id));
        let submissions = AtomicUsize::new(0);

        let first = complete_request(
            &state,
            async || panic!("submitted request must not be replaced yet"),
            async |observed| {
                Err(ProofWaitError::Terminal {
                    proof_id: observed,
                    state: ProofTerminalState::Cancelled,
                    fulfillment_status: 0,
                    execution_status: 0,
                })
            },
            |_| Ok(()),
        )
        .await;
        assert!(first.is_err());
        assert!(matches!(*state.lock(), RequestState::Missing));

        complete_request(
            &state,
            async || {
                submissions.fetch_add(1, Ordering::SeqCst);
                Ok(replacement_id)
            },
            async |observed| {
                assert_eq!(observed, replacement_id);
                Ok(fulfilled_proof())
            },
            |_| Ok(()),
        )
        .await
        .unwrap();

        assert_eq!(submissions.load(Ordering::SeqCst), 1);
    }

    #[tokio::test]
    async fn validation_failure_stays_terminal_without_resubmission() {
        let proof_id = B256::repeat_byte(0x4e);
        let state = Mutex::new(RequestState::Missing);

        let first = complete_request(
            &state,
            async || Ok(proof_id),
            async |_| Ok(fulfilled_proof()),
            |_| bail!("invalid proof"),
        )
        .await;
        assert!(first.is_err());
        assert!(matches!(
            *state.lock(),
            RequestState::Terminal(ProofTerminalState::ValidationFailed)
        ));

        let second = complete_request(
            &state,
            async || panic!("terminal request must not be resubmitted"),
            async |_| panic!("terminal request must not be polled"),
            |_| panic!("terminal proof must not be reverified"),
        )
        .await;
        assert!(second.unwrap_err().to_string().contains("terminal state"));
    }

    #[test]
    fn span_roots_reject_endpoint_mismatch() {
        let start = response_at(100, 0x01, 5);
        let end = response_at(103, 0x02, 5);
        let game = game_for(&start, &end, 10);
        let responses = [start.clone(), end.clone()];

        assert!(check_span_roots(&game, &[root_of(&start), root_of(&end)], &responses).is_ok());

        // A diverged starting root from a trusted response is permanently unprovable.
        let err = check_span_roots(&game, &[B256::repeat_byte(0xee), root_of(&end)], &responses)
            .unwrap_err();
        assert!(is_unprovable(&err), "expected GameUnprovable, got: {err}");

        // A diverged claim root from a trusted response is permanently unprovable.
        let err = check_span_roots(&game, &[root_of(&start), B256::repeat_byte(0xee)], &responses)
            .unwrap_err();
        assert!(is_unprovable(&err), "expected GameUnprovable, got: {err}");

        // The same mismatch from an untrusted response is transient.
        let end_root = root_of(&end);
        let untrusted_responses = [untrusted(start), untrusted(end)];
        let err =
            check_span_roots(&game, &[B256::repeat_byte(0xee), end_root], &untrusted_responses)
                .unwrap_err();
        assert!(!is_unprovable(&err), "untrusted mismatch must be transient, got: {err}");
    }

    #[test]
    fn unprovable_guard_triggers_on_required_l1_beyond_head() {
        let start = response_at(100, 0x01, 5);
        let end = response_at(103, 0x02, 42);
        let game = game_for(&start, &end, 41);

        // Trusted violation: permanent.
        let err = check_provable(&game, &end).unwrap_err();
        assert!(is_unprovable(&err), "expected GameUnprovable, got: {err}");

        // Untrusted violation (mid-rewind answer): transient.
        let err = check_provable(&game, &untrusted(end.clone())).unwrap_err();
        assert!(!is_unprovable(&err), "untrusted violation must be transient, got: {err}");

        // Boundary (current_l1 == required L1): that block may still be
        // mid-processing, so the verdict must stay transient.
        let err = check_provable(&game, &at_required_l1(end.clone())).unwrap_err();
        assert!(!is_unprovable(&err), "boundary violation must be transient, got: {err}");

        // At or below the game's L1 head is provable.
        let game = game_for(&start, &end, 42);
        assert!(check_provable(&game, &end).is_ok());
    }

    #[test]
    fn unavailable_data_is_transient_not_unprovable() {
        let err = anyhow::Error::new(SuperRootDataUnavailable(101));
        assert!(!is_unprovable(&err));
    }

    #[test]
    fn aggregation_assembly_validates_and_binds_prover() {
        // The canonical two-chain fixture validates and its public values
        // ABI-encode exactly the tuple ZKDisputeGame.prove() reconstructs:
        // abi.encode(l1Head, startingRootHash, rootClaim, l2SequenceNumber,
        // prover).
        let inputs = valid_aggregation_inputs();
        inputs.validate().unwrap();

        let encoded = inputs.zk_dispute_game_public_values();
        assert_eq!(encoded.len(), 160, "five ABI words");
        assert_eq!(&encoded[0..32], inputs.l1_head.as_slice());
        assert_eq!(&encoded[32..64], inputs.starting_root_hash.as_slice());
        assert_eq!(&encoded[64..96], inputs.root_claim.as_slice());
        assert_eq!(U256::from_be_slice(&encoded[96..128]), U256::from(inputs.l2_sequence_number));
        assert_eq!(&encoded[140..160], inputs.prover.as_slice());
    }

    #[test]
    fn chunk_ordering_preserved_under_unordered_completion() {
        // collect_indexed restores chunk order regardless of completion
        // order, and errors on gaps instead of silently reordering.
        let mut slots = vec![None, None, None];
        for index in [2usize, 0, 1] {
            slots[index] = Some(index * 10);
        }
        assert_eq!(collect_indexed(slots, "chunks").unwrap(), vec![0, 10, 20]);

        let err = collect_indexed(vec![Some(1), None::<u64>], "chunks").unwrap_err();
        assert!(err.to_string().contains("missing chunks for chunk 1"));
    }

    #[tokio::test(flavor = "current_thread")]
    async fn proof_replay_chunk_span_survives_unordered_execution() {
        let capture = Arc::new(StdMutex::new(TraceCapture::default()));
        let subscriber = CapturingSubscriber(capture.clone());
        let _guard = tracing::subscriber::set_default(subscriber);

        let work = async {
            let tasks = (0..2usize).map(|index| {
                let agreed_timestamp = 100 + index as u64 * 10;
                let claimed_timestamp = agreed_timestamp + 10;
                with_proof_replay_chunk_span(
                    index,
                    2,
                    agreed_timestamp,
                    claimed_timestamp,
                    async move {
                        tokio::task::yield_now().await;
                        tracing::warn!(index, "nested derivation warning");
                    },
                )
            });
            futures::stream::iter(tasks).buffer_unordered(2).collect::<Vec<_>>().await;
        };
        work.instrument(tracing::info_span!("game", game_address = "0x1234")).await;

        let capture = capture.lock().unwrap();
        assert_eq!(capture.event_spans.len(), 2);
        let outer = capture.spans.values().find(|span| span.name == "game").unwrap();
        assert!(outer.fields.contains("game_address=\"0x1234\""));
        for index in 0..2 {
            let chunk = capture
                .event_spans
                .iter()
                .map(|span_id| &capture.spans[span_id])
                .find(|span| span.fields.contains(&format!("chunk_index={index} ")))
                .unwrap();
            assert_eq!(chunk.name, "proof_replay_chunk");
            assert_eq!(chunk.level, Some(Level::WARN));
            assert_eq!(capture.spans[&chunk.parent.unwrap()].name, "game");
            assert!(chunk.fields.contains("derivation_mode=\"proof_replay\""));
            assert!(chunk.fields.contains("chunk_count=2 "));
            assert!(chunk.fields.contains(&format!("agreed_timestamp={} ", 100 + index * 10)));
            assert!(chunk.fields.contains(&format!("claimed_timestamp={} ", 110 + index * 10)));
            assert!(
                chunk.fields.contains(&format!("guest_span_start_timestamp={} ", 101 + index * 10))
            );
            assert!(
                chunk.fields.contains(&format!("guest_span_end_timestamp={} ", 110 + index * 10))
            );
        }
    }

    #[tokio::test]
    async fn chunk_error_drains_admitted_siblings() {
        let admitted_started = Arc::new(tokio::sync::Notify::new());
        let error_reported = Arc::new(tokio::sync::Notify::new());
        let release_admitted = Arc::new(tokio::sync::Notify::new());
        let admitted_completed = Arc::new(AtomicBool::new(false));
        let queued_started = Arc::new(AtomicBool::new(false));
        let start = response_at(100, 1, 5);
        let end = response_at(102, 3, 5);
        let progress = Arc::new(GameProgress::new(attempt_identity(game_for(&start, &end, 10))));

        let tasks = (0..3).map(|index| {
            let admitted_started = Arc::clone(&admitted_started);
            let error_reported = Arc::clone(&error_reported);
            let release_admitted = Arc::clone(&release_admitted);
            let admitted_completed = Arc::clone(&admitted_completed);
            let queued_started = Arc::clone(&queued_started);
            let progress = Arc::clone(&progress);
            async move {
                match index {
                    0 => {
                        admitted_started.notify_one();
                        release_admitted.notified().await;
                        let result = complete_request(
                            &progress.chunks[0].range_request,
                            async || Ok(B256::repeat_byte(0x49)),
                            async |proof_id| {
                                Err(ProofWaitError::Terminal {
                                    proof_id,
                                    state: ProofTerminalState::Unexecutable,
                                    fulfillment_status: 0,
                                    execution_status: 0,
                                })
                            },
                            |_| Ok(()),
                        )
                        .await;
                        admitted_completed.store(true, Ordering::SeqCst);
                        result.map(|_| index)
                    }
                    1 => {
                        admitted_started.notified().await;
                        error_reported.notify_one();
                        Err(std::io::Error::from(std::io::ErrorKind::TimedOut).into())
                    }
                    2 => {
                        queued_started.store(true, Ordering::SeqCst);
                        Ok(index)
                    }
                    _ => unreachable!(),
                }
            }
        });

        let controller_error_reported = Arc::clone(&error_reported);
        let controller_release_admitted = Arc::clone(&release_admitted);
        let controller = tokio::spawn(async move {
            controller_error_reported.notified().await;
            tokio::task::yield_now().await;
            controller_release_admitted.notify_one();
        });

        let result = run_chunk_tasks_until_error(tasks, 2).await;
        controller.await.unwrap();

        assert!(result.unwrap_err().downcast_ref::<std::io::Error>().is_some());
        assert!(progress.check_terminal_requests().is_err());
        assert!(
            admitted_completed.load(Ordering::SeqCst),
            "admitted sibling was dropped before completion"
        );
        assert!(!queued_started.load(Ordering::SeqCst), "queued chunk started after sibling error");
    }

    /// A batch prover that records each submit and poll, fails a batch's poll once with a
    /// retryable terminal state when the batch is in `cancel_once`, and fails the verification of
    /// a batch in `fail_verify`.
    #[derive(Default)]
    struct ScriptedLogProver {
        events: StdMutex<Vec<String>>,
        cancel_once: StdMutex<std::collections::HashSet<B256>>,
        fail_verify: StdMutex<std::collections::HashSet<B256>>,
    }

    #[async_trait::async_trait]
    impl LogBatchProver for ScriptedLogProver {
        async fn submit(&self, batch: &LogBatch) -> Result<ProofId> {
            self.events.lock().unwrap().push(format!("submit {}", batch.blocks[0].number));
            Ok(batch.id)
        }

        async fn poll(
            &self,
            batch: &LogBatch,
            proof_id: ProofId,
        ) -> Result<SP1ProofWithPublicValues, ProofWaitError> {
            self.events.lock().unwrap().push(format!("poll {}", batch.blocks[0].number));
            if self.cancel_once.lock().unwrap().remove(&batch.id) {
                return Err(ProofWaitError::Terminal {
                    proof_id,
                    state: ProofTerminalState::Cancelled,
                    fulfillment_status: 0,
                    execution_status: 0,
                });
            }
            Ok(SP1ProofWithPublicValues {
                public_values: SP1PublicValues::from(&batch.expected),
                ..fulfilled_proof()
            })
        }

        fn verify(&self, batch: &LogBatch, proof: &SP1ProofWithPublicValues) -> Result<()> {
            ensure!(!self.fail_verify.lock().unwrap().contains(&batch.id), "scripted failure");
            ensure!(
                proof.public_values.as_slice() == batch.expected,
                "{batch} commits other entries"
            );
            Ok(())
        }
    }

    fn log_batch(number: u64) -> LogBatch {
        let block = kona_sp1_super_range_executor::LogBatchBlock {
            chain_id: 1,
            number,
            hash: B256::repeat_byte(number as u8),
            receipts_root: B256::repeat_byte(number as u8),
            pgu: 1,
            witness_bytes: 1,
        };
        LogBatch {
            id: B256::repeat_byte(number as u8),
            blocks: vec![block],
            expected: vec![number as u8; 72],
        }
    }

    fn game_identity() -> AttemptIdentity {
        attempt_identity(game_for(&response_at(100, 1, 5), &response_at(102, 3, 5), 10))
    }

    fn table_stage(batches: impl IntoIterator<Item = LogBatch>) -> LogStage {
        LogStage {
            table: true,
            batches: batch_progress(batches.into_iter().collect(), None),
            ..Default::default()
        }
    }

    /// Pins every batch request submitted before the first poll.
    #[tokio::test]
    async fn all_batches_are_submitted_before_any_wait() {
        let stage = Arc::new(table_stage((1..=3).map(log_batch)));
        let prover = Arc::new(ScriptedLogProver::default());
        let proofs = start_log_batches(prover.clone(), Arc::clone(&stage)).await.unwrap();
        assert_eq!(*prover.events.lock().unwrap(), ["submit 1", "submit 2", "submit 3"]);
        let proofs = proofs.await.unwrap();
        assert_eq!(prover.events.lock().unwrap().len(), 6);
        let committed: Vec<_> = proofs.iter().map(|proof| proof.public_values.to_vec()).collect();
        let expected: Vec<_> = stage.batches.iter().map(|b| b.batch.expected.clone()).collect();
        assert_eq!(committed, expected);
    }

    /// Pins a batch error after the admitted chunk ends, and no work in a chunk admitted later.
    #[tokio::test]
    async fn batch_failure_drains_admitted_chunks() {
        let stage = Arc::new(table_stage([log_batch(1)]));
        let failed = stage.batches[0].batch.to_string();
        let prover = Arc::new(ScriptedLogProver::default());
        prover.cancel_once.lock().unwrap().insert(stage.batches[0].batch.id);
        let proofs = start_log_batches(prover, stage).await.unwrap();
        let (release, released) = tokio::sync::oneshot::channel();
        let work = StdMutex::new(Vec::new());
        let admitted = async {
            released.await.unwrap();
            work.lock().unwrap().push(0);
            Ok(0)
        };
        let later = async {
            check_log_batches(&proofs)?;
            work.lock().unwrap().push(1);
            Ok(1)
        };
        let chunks = run_chunk_tasks_until_error([admitted.boxed_local(), later.boxed_local()], 1);
        let (result, ()) = tokio::join!(run_with_log_batches(chunks, proofs.clone()), async {
            proofs.clone().await.unwrap_err();
            release.send(()).unwrap();
        });
        let err = result.unwrap_err();
        assert!(err.to_string().contains(&failed), "{err}");
        assert_eq!(*work.lock().unwrap(), [0]);
    }

    /// Pins a retry that reuses a fulfilled batch and requests a cancelled one again.
    #[tokio::test]
    async fn retry_reuses_fulfilled_batches() {
        let progress = GameProgress::new(game_identity());
        let stage = progress.store_log_stage(table_stage((1..=2).map(log_batch)));
        let prover = Arc::new(ScriptedLogProver::default());
        prover.cancel_once.lock().unwrap().insert(stage.batches[1].batch.id);
        start_log_batches(prover.clone(), stage).await.unwrap().await.unwrap_err();

        prover.events.lock().unwrap().clear();
        let stage = progress.log_stage().expect("a batch failure keeps the stage");
        start_log_batches(prover.clone(), stage).await.unwrap().await.unwrap();
        assert_eq!(*prover.events.lock().unwrap(), ["submit 2", "poll 2"]);

        progress.log_stage().unwrap().mark_stale();
        assert!(progress.log_stage().unwrap().is_stale());
    }

    /// Pins a rescan that keeps the request of each batch with an unchanged ID.
    #[test]
    fn rescan_keeps_requests_of_unchanged_batches() {
        let previous = table_stage((1..=2).map(log_batch));
        *previous.batches[0].request.lock() = RequestState::Submitted(B256::repeat_byte(9));
        let batches = batch_progress(vec![log_batch(1), log_batch(3)], Some(&previous));
        assert!(
            matches!(*batches[0].request.lock(), RequestState::Submitted(id) if id == B256::repeat_byte(9))
        );
        assert!(matches!(*batches[1].request.lock(), RequestState::Missing));
    }

    /// Pins a batch that fails verification as a terminal log request until a retry signal.
    #[tokio::test]
    async fn failed_batch_verification_is_terminal_until_retry() {
        let cache = InMemoryProofProgress::default();
        let game_address = Address::repeat_byte(0x47);
        let progress = cache.load_or_create(game_address, game_identity());
        let stage = progress.store_log_stage(table_stage([log_batch(1)]));
        let failed = stage.batches[0].batch.to_string();
        let prover = Arc::new(ScriptedLogProver::default());
        prover.fail_verify.lock().unwrap().insert(stage.batches[0].batch.id);
        start_log_batches(prover, stage).await.unwrap().await.unwrap_err();

        let err = progress.check_terminal_requests().unwrap_err();
        assert!(err.to_string().contains(&failed), "{err}");
        assert_eq!(cache.request_counts().get(&("log", "terminal")), Some(&1));
        assert_eq!(cache.retry_terminal_requests(game_address), 1);
        progress.check_terminal_requests().unwrap();
    }

    /// An L2 source whose every block has no receipts.
    struct EmptyReceiptsL2;

    #[async_trait::async_trait]
    impl L2BlockSource for EmptyReceiptsL2 {
        async fn header_by_hash(
            &self,
            _chain_id: u64,
            _hash: B256,
        ) -> Result<alloy_consensus::Header> {
            bail!("no header reads in this test")
        }

        async fn header_by_number(
            &self,
            _chain_id: u64,
            _number: u64,
        ) -> Result<alloy_consensus::Sealed<alloy_consensus::Header>> {
            bail!("no header reads in this test")
        }

        async fn receipts(
            &self,
            _chain_id: u64,
            _hash: B256,
            _receipts_root: B256,
        ) -> Result<Vec<op_alloy_consensus::OpReceiptEnvelope>> {
            Ok(Vec::new())
        }
    }

    /// Pins each prestate's log key on its batch request and arrival check.
    #[tokio::test]
    #[serial_test::serial]
    async fn real_elf_two_prestates_route_log_requests_by_vkey() {
        use crate::prover::{
            NetworkProofProvider,
            scripted::{ScriptedNetworkApi, fulfilled},
        };
        use sp1_sdk::network::NetworkMode;

        let Ok(dir) = std::env::var("KONA_SP1_ELF_DIR") else { return };
        let read = |name: &str| std::fs::read(Path::new(&dir).join(name)).unwrap();
        let programs = crate::config::PrestatePrograms {
            aggregation_elf: read("super-aggregation-elf"),
            range_elf: read("super-range-elf"),
            log_commitment_elf: read("log-commitment-elf"),
        };
        let first = crate::prover::setup_proof_keys(&programs).await.unwrap();
        let second = crate::prover::setup_proof_keys(&crate::config::PrestatePrograms {
            log_commitment_elf: programs.aggregation_elf.clone(),
            ..programs
        })
        .await
        .unwrap();
        assert_ne!(first.log_vk.hash_u32(), second.log_vk.hash_u32());

        let mut batch = log_batch(1);
        batch.blocks[0].receipts_root = alloy_consensus::EMPTY_ROOT_HASH;
        let api = Arc::new(ScriptedNetworkApi::new(Vec::new(), None));
        let config = crate::config::ProofProviderConfig::from_env().unwrap();
        let provider = ProofProvider::Network(NetworkProofProvider::scripted(
            api.clone(),
            config,
            NetworkMode::Reserved,
        ));
        for keys in [&first, &second] {
            let stage = Arc::new(table_stage([batch.clone()]));
            api.log_request_ids.lock().push_back(batch.id);
            api.statuses.lock().push_back(fulfilled(SP1ProofWithPublicValues {
                public_values: SP1PublicValues::from(&batch.expected),
                ..fulfilled_proof()
            }));
            let prover = Arc::new(NetworkLogProver {
                provider: &provider,
                keys,
                l2: &EmptyReceiptsL2,
                poll_interval: std::time::Duration::ZERO,
            });
            start_log_batches(prover, stage).await.unwrap().await.unwrap();
        }
        let requested: Vec<_> =
            api.log_requests.lock().iter().map(|request| request.vkey).collect();
        let expected = [first.log_vk.hash_u32(), second.log_vk.hash_u32()];
        assert_eq!(requested, expected);
        assert_eq!(*api.verified.lock(), expected);
    }

    #[tokio::test]
    async fn chunk_success_preserves_bounded_concurrency() {
        let first_pair_started = Arc::new(tokio::sync::Barrier::new(2));
        let started = Arc::new(AtomicUsize::new(0));
        let active = Arc::new(AtomicUsize::new(0));
        let max_active = Arc::new(AtomicUsize::new(0));

        let tasks = (0..4).map(|index| {
            let first_pair_started = Arc::clone(&first_pair_started);
            let started = Arc::clone(&started);
            let active = Arc::clone(&active);
            let max_active = Arc::clone(&max_active);
            async move {
                started.fetch_add(1, Ordering::SeqCst);
                let current = active.fetch_add(1, Ordering::SeqCst) + 1;
                max_active.fetch_max(current, Ordering::SeqCst);

                if index < 2 {
                    first_pair_started.wait().await;
                }
                tokio::task::yield_now().await;

                active.fetch_sub(1, Ordering::SeqCst);
                Ok::<_, anyhow::Error>((index, index * 10))
            }
        });

        let results = run_chunk_tasks_until_error(tasks, 2).await.unwrap();

        assert_eq!(started.load(Ordering::SeqCst), 4);
        assert_eq!(active.load(Ordering::SeqCst), 0);
        assert_eq!(max_active.load(Ordering::SeqCst), 2);

        let mut slots = vec![None; 4];
        for (index, value) in results {
            slots[index] = Some(value);
        }
        assert_eq!(collect_indexed(slots, "chunks").unwrap(), vec![0, 10, 20, 30]);
    }
}
