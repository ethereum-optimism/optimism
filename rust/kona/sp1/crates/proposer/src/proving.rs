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
    sync::Arc,
};

use alloy_primitives::{Address, B256, U256, keccak256};
use alloy_sol_types::SolValue;
use anyhow::{Context, Result, anyhow, bail, ensure};
use futures::{
    future::{BoxFuture, FutureExt},
    stream::{FuturesUnordered, StreamExt},
};
use kona_sp1_client_utils::{
    super_root::{
        SuperAggregationInputs, SuperAggregationPublicValues, SuperConsolidationInputs,
        SuperConsolidationOutputs, SuperRangeInputs, SuperRangeOutputs, TimestampSpan,
    },
    witness::DefaultWitnessData,
};
use kona_sp1_host_utils::metrics::MetricsGauge;
use kona_sp1_super_range_executor::{
    HostInputs, MAX_ARCHIVED_WITNESS_BYTES, SuperRootAtTimestampResponse, SynthesizedExecution,
    build_interop_host, build_super_consolidation_stdin, build_super_range_stdin,
    collect_consolidation_witness, collect_range_witness, decode_super_consolidation_public_values,
    decode_super_range_public_values, deployment_chain_configs, is_witness_too_large,
    proof_from_super_v1, synthesize_execution,
};
use parking_lot::Mutex;
use serde::Serialize;
use sp1_sdk::{HashableKey, SP1Proof, SP1ProofWithPublicValues, SP1Stdin};
use tracing::Instrument;

pub use crate::ports::GameProofInputs;

use crate::{
    config::{RangeSplitCount, StdinByteLimit, WitnessByteLimit},
    metrics::ProposerGauge,
    ports::SuperRootSource,
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
    Network { range_vkey: B256, aggregation_vkey: B256 },
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

/// SPN request kinds of one game proof, as named by [`GameProgress::for_each_request`].
pub(crate) const PROOF_REQUEST_KINDS: [&str; 3] = ["range", "consolidation", "aggregation"];
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
    /// Halves proved instead of this span once it exceeded a witness or execution limit.
    split: Mutex<Option<Arc<[Self; 2]>>>,
}

impl ChunkProgress {
    /// Visits the live request slots of this span, in timestamp order. Once the span has been
    /// split, its own slots are superseded and only the halves' slots are visited.
    fn for_each_request(&self, visit: &mut dyn FnMut(&'static str, &Mutex<RequestState>)) {
        let halves = self.split.lock().clone();
        if let Some(halves) = halves {
            for half in halves.iter() {
                half.for_each_request(visit);
            }
            return;
        }
        visit("range", &self.range_request);
        visit("consolidation", &self.consolidation_request);
    }
}

struct GameProgress {
    identity: AttemptIdentity,
    chunks: Vec<ChunkProgress>,
    aggregation_request: Mutex<RequestState>,
}

impl GameProgress {
    fn new(identity: AttemptIdentity) -> Self {
        let chunks = (0..identity.chunks.len()).map(|_| ChunkProgress::default()).collect();
        Self { identity, chunks, aggregation_request: Mutex::new(RequestState::Missing) }
    }

    /// Visits every live request slot: each top-level chunk's (with its index), then the
    /// aggregation slot.
    fn for_each_request(
        &self,
        mut visit: impl FnMut(Option<usize>, &'static str, &Mutex<RequestState>),
    ) {
        for (index, chunk) in self.chunks.iter().enumerate() {
            chunk.for_each_request(&mut |kind, request| visit(Some(index), kind, request));
        }
        visit(None, "aggregation", &self.aggregation_request);
    }

    fn check_terminal_requests(&self) -> Result<()> {
        let mut terminal = None;
        self.for_each_request(|chunk_index, kind, request| {
            if terminal.is_none() &&
                let RequestState::Terminal(outcome) = &*request.lock()
            {
                terminal = Some((chunk_index, kind, *outcome));
            }
        });
        match terminal {
            Some((Some(index), kind, outcome)) => {
                bail!("chunk {index} {kind} proof request reached terminal state: {outcome:?}")
            }
            Some((None, kind, outcome)) => {
                bail!("{kind} proof request reached terminal state: {outcome:?}")
            }
            None => Ok(()),
        }
    }

    fn request_summary(&self) -> RequestSummary {
        let mut summary = RequestSummary::default();
        self.for_each_request(|_, _, request| match &*request.lock() {
            RequestState::Submitting => summary.submitting += 1,
            RequestState::Submitted(proof_id) => summary.submitted_ids.push(*proof_id),
            RequestState::Fulfilled(_) => summary.fulfilled += 1,
            RequestState::Missing | RequestState::Terminal(_) => {}
        });
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
        progress.for_each_request(|_, _, request| {
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
            progress.for_each_request(|_, kind, request| {
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

async fn complete_request(
    state: &Mutex<RequestState>,
    submit: impl AsyncFnOnce() -> Result<ProofId>,
    poll: impl AsyncFnOnce(ProofId) -> Result<SP1ProofWithPublicValues, ProofWaitError>,
    verify: impl FnOnce(&SP1ProofWithPublicValues) -> Result<()>,
) -> Result<Arc<SP1ProofWithPublicValues>> {
    let existing = {
        let mut state = state.lock();
        match &*state {
            RequestState::Missing => {
                *state = RequestState::Submitting;
                None
            }
            RequestState::Submitting => bail!("proof request submission is already in progress"),
            RequestState::Submitted(proof_id) => Some(*proof_id),
            RequestState::Fulfilled(proof) => return Ok(Arc::clone(proof)),
            RequestState::Terminal(outcome) => {
                bail!("proof request reached terminal state: {outcome:?}")
            }
        }
    };

    let proof_id = match existing {
        Some(proof_id) => proof_id,
        None => match submit().await {
            Ok(proof_id) => {
                *state.lock() = RequestState::Submitted(proof_id);
                proof_id
            }
            Err(err) => {
                *state.lock() = RequestState::Missing;
                return Err(err);
            }
        },
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

/// Outputs and proofs of one range + consolidation pair covering a contiguous span.
#[derive(Clone)]
struct SpanResult {
    range_outputs: SuperRangeOutputs,
    consolidation_outputs: SuperConsolidationOutputs,
    proofs: Option<ChildProofs>,
}

/// One proven (or natively executed) top-level chunk, tagged with its position. A chunk that
/// was bisected holds several spans, in timestamp order.
#[derive(Clone)]
struct ChunkResult {
    index: usize,
    spans: Vec<SpanResult>,
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

/// A proof request's stdin exceeds `OP_ZK_PROPOSER_MAX_STDIN_BYTES`.
#[derive(Debug, Clone, PartialEq, Eq, thiserror::Error)]
#[error("{kind} stdin is {bytes} encoded bytes, above the {limit}-byte stdin limit")]
pub struct StdinTooLarge {
    /// Request kind: `range`, `consolidation` or `aggregation`.
    pub kind: &'static str,
    /// Encoded stdin bytes.
    pub bytes: u64,
    /// The configured limit.
    pub limit: u64,
}

/// Bytes of `stdin` as the SP1 SDK uploads it, before compression: bincode 1 of the input
/// buffers, the read pointer, and every embedded proof with its verifying key.
///
/// The buffers are counted arithmetically instead of encoded, since the witness buffer can be
/// gigabytes: a `u64` length prefix for the buffer list and for each buffer, then its bytes.
fn stdin_encoded_len(stdin: &SP1Stdin) -> Result<u64> {
    let buffers: u64 = stdin.buffer.iter().map(|buffer| 8 + buffer.len() as u64).sum();
    let proofs =
        bincode1::serialized_size(&stdin.proofs).context("failed to measure embedded proofs")?;
    Ok(8 + buffers + 8 + proofs)
}

/// Fails with [`StdinTooLarge`] when `stdin` encodes to more than `limit` bytes.
fn ensure_stdin_fits(kind: &'static str, stdin: &SP1Stdin, limit: StdinByteLimit) -> Result<()> {
    let bytes = stdin_encoded_len(stdin)?;
    if bytes > limit.get() {
        return Err(anyhow::Error::new(StdinTooLarge { kind, bytes, limit: limit.get() }));
    }
    Ok(())
}

async fn prove_range_request(
    provider: &ProofProvider,
    keys: &ProofKeys,
    inputs: &SuperRangeInputs,
    witness: DefaultWitnessData,
    expected: &SuperRangeOutputs,
    state: &Mutex<RequestState>,
    stdin_limit: StdinByteLimit,
) -> Result<Arc<SP1ProofWithPublicValues>> {
    let stdin = build_super_range_stdin(inputs, witness)?;
    complete_request(
        state,
        async move || {
            ensure_stdin_fits("range", &stdin, stdin_limit)?;
            provider.request_range_proof(keys, stdin).await
        },
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
    inputs: &SuperConsolidationInputs,
    witness: DefaultWitnessData,
    expected: &SuperConsolidationOutputs,
    state: &Mutex<RequestState>,
    stdin_limit: StdinByteLimit,
) -> Result<Arc<SP1ProofWithPublicValues>> {
    let stdin = build_super_consolidation_stdin(inputs, witness)?;
    complete_request(
        state,
        async move || {
            ensure_stdin_fits("consolidation", &stdin, stdin_limit)?;
            provider.request_range_proof(keys, stdin).await
        },
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

/// A span of timestamps proved by one range + consolidation pair: `agreed + 1..=claimed`.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
struct ChunkSpan {
    agreed: u64,
    claimed: u64,
}

/// Proves one span as a single range + consolidation pair.
trait SpanProver: Sync {
    fn prove_leaf<'s>(
        &'s self,
        span: ChunkSpan,
        synthesized: Option<SynthesizedExecution>,
        node: &'s ChunkProgress,
    ) -> BoxFuture<'s, Result<SpanResult>>;
}

/// Per-game inputs shared by every chunk of one proving attempt.
struct ChunkContext<'a> {
    provider: &'a ProofProvider,
    keys: Option<&'a ProofKeys>,
    host_inputs: &'a HostInputs,
    game: &'a GameProofInputs,
    responses: &'a [SuperRootAtTimestampResponse],
    witness_limit: WitnessByteLimit,
    stdin_limit: StdinByteLimit,
}

impl SpanProver for ChunkContext<'_> {
    fn prove_leaf<'s>(
        &'s self,
        span: ChunkSpan,
        synthesized: Option<SynthesizedExecution>,
        node: &'s ChunkProgress,
    ) -> BoxFuture<'s, Result<SpanResult>> {
        async move {
            let synthesized = match synthesized {
                Some(synthesized) => synthesized,
                None => synthesize_chunk(self.game, self.responses, span)?,
            };
            prove_span_pair(self, &synthesized, node, span).await
        }
        .boxed()
    }
}

/// Collects the range and consolidation witnesses of one span and, in network mode, submits or
/// resumes its two child proof requests in order.
async fn prove_span_pair(
    ctx: &ChunkContext<'_>,
    synthesized: &SynthesizedExecution,
    node: &ChunkProgress,
    chunk_span: ChunkSpan,
) -> Result<SpanResult> {
    let ChunkContext { provider, keys, host_inputs, game, witness_limit, stdin_limit, .. } = *ctx;
    let span = synthesized.range_inputs.span;
    // A single timestamp cannot be split further, so it may use the whole archive.
    let collection_limit = if chunk_span.claimed - chunk_span.agreed > 1 {
        witness_limit.get()
    } else {
        MAX_ARCHIVED_WITNESS_BYTES
    };
    let range_host = build_interop_host(
        host_inputs,
        game.l1_head,
        &synthesized.previous_super_root_proof_bytes,
        synthesized.current_super_root,
        span.end,
    )?;
    let configs = deployment_chain_configs(&range_host, &synthesized.range_inputs.chain_ids)?;
    let (range_witness, range_outputs) = collect_range_witness(
        range_host,
        &synthesized.range_inputs,
        &synthesized.preloaded_preimages,
        Some(&configs),
        Some(collection_limit),
    )
    .await
    .with_context(|| format!("range witness collection failed for span {span:?}"))?;

    let consolidation_host = build_interop_host(
        host_inputs,
        game.l1_head,
        &synthesized.previous_super_root_proof_bytes,
        synthesized.current_super_root,
        span.end,
    )?;
    let (consolidation_witness, consolidation_outputs) = collect_consolidation_witness(
        consolidation_host,
        &synthesized.consolidation_inputs,
        &synthesized.preloaded_preimages,
        Some(&configs),
        Some(collection_limit),
    )
    .await
    .with_context(|| format!("consolidation witness collection failed for span {span:?}"))?;

    let proofs = match (provider, keys) {
        (ProofProvider::Mock(_), _) => None,
        (provider, Some(keys)) => {
            let range_proof = prove_range_request(
                provider,
                keys,
                &synthesized.range_inputs,
                range_witness,
                &range_outputs,
                &node.range_request,
                stdin_limit,
            )
            .await?;
            let consolidation_proof = prove_consolidation_request(
                provider,
                keys,
                &synthesized.consolidation_inputs,
                consolidation_witness,
                &consolidation_outputs,
                &node.consolidation_request,
                stdin_limit,
            )
            .await?;
            Some(ChildProofs { range: range_proof, consolidation: consolidation_proof })
        }
        (_, None) => bail!("network proving requires prestate proving keys"),
    };

    Ok(SpanResult { range_outputs, consolidation_outputs, proofs })
}

/// Synthesizes the guest inputs of `span` from the game's verified responses.
fn synthesize_chunk(
    game: &GameProofInputs,
    responses: &[SuperRootAtTimestampResponse],
    span: ChunkSpan,
) -> Result<SynthesizedExecution> {
    let first = (span.agreed - game.starting_ts) as usize;
    let last = (span.claimed - game.starting_ts) as usize;
    let timestamps = TimestampSpan::new(span.agreed + 1, span.claimed)
        .map_err(|err| anyhow!("invalid chunk span: {err:?}"))?;
    synthesize_execution(timestamps, game.l1_head, game.l1_head_number, &responses[first..=last])
        .with_context(|| {
            format!("failed to synthesize span {}..={}", timestamps.start, timestamps.end)
        })
}

/// True when `err` shows the span's witness or stdin is too large for one child proof request.
fn exceeds_size_limits(err: &anyhow::Error) -> bool {
    is_witness_too_large(err) || err.chain().any(|cause| cause.is::<StdinTooLarge>())
}

/// True when `err` shows the span is too large for one child proof.
///
/// Any `Unexecutable` counts, whatever SPN's execute-failure cause: SP1 execution is
/// deterministic, so the same span cannot succeed on resubmission. A genuine guest failure is
/// bisected down to a single timestamp, costing at most log2(chunk length) extra requests.
fn span_exceeds_limits(err: &anyhow::Error) -> bool {
    exceeds_size_limits(err) ||
        err.chain().any(|cause| {
            matches!(
                cause.downcast_ref::<ProofWaitError>(),
                Some(ProofWaitError::Terminal { state: ProofTerminalState::Unexecutable, .. })
            )
        })
}

/// Proves `span` as one range + consolidation pair, or, once it exceeds a witness or execution
/// limit, as two halves recorded on `node` and proved in timestamp order.
///
/// Halves stay on `node` across scheduler retries, so an oversized span is not collected again,
/// and each half caches its own result. The caller caches `node`'s result.
fn prove_span<'s>(
    prover: &'s dyn SpanProver,
    index: usize,
    span: ChunkSpan,
    synthesized: Option<SynthesizedExecution>,
    node: &'s ChunkProgress,
) -> BoxFuture<'s, Result<ChunkResult>> {
    async move {
        let mid = span.agreed + (span.claimed - span.agreed) / 2;
        let recorded = node.split.lock().clone();
        let halves = match recorded {
            Some(halves) => halves,
            None => {
                let err = match prover.prove_leaf(span, synthesized, node).await {
                    Ok(result) => return Ok(ChunkResult { index, spans: vec![result] }),
                    Err(err) => err,
                };
                if !span_exceeds_limits(&err) {
                    return Err(err);
                }
                if span.claimed - span.agreed == 1 {
                    if exceeds_size_limits(&err) {
                        return Err(anyhow::Error::new(GameUnprovable(format!(
                            "timestamp {} does not fit one child proof request: {err:#}",
                            span.claimed
                        ))));
                    }
                    // Unexecutable: the request stays terminal and SIGUSR1 can retry it.
                    return Err(err);
                }
                tracing::warn!(
                    chunk_index = index,
                    agreed_timestamp = span.agreed,
                    claimed_timestamp = span.claimed,
                    split_timestamp = mid,
                    error = %format!("{err:#}"),
                    "Span exceeds proving limits; bisecting"
                );
                Arc::clone(node.split.lock().get_or_insert_with(|| {
                    Arc::new([ChunkProgress::default(), ChunkProgress::default()])
                }))
            }
        };

        let first = run_chunk_with_progress(
            &halves[0],
            prove_span(
                prover,
                index,
                ChunkSpan { agreed: span.agreed, claimed: mid },
                None,
                &halves[0],
            ),
        )
        .await?;
        let second = run_chunk_with_progress(
            &halves[1],
            prove_span(
                prover,
                index,
                ChunkSpan { agreed: mid, claimed: span.claimed },
                None,
                &halves[1],
            ),
        )
        .await?;
        let spans = first.spans.iter().chain(&second.spans).cloned().collect();
        Ok(ChunkResult { index, spans })
    }
    .boxed()
}

/// Proves one top-level chunk, bisecting it as needed, and returns its spans for game-level
/// aggregation.
///
/// Per-span request state lives in `progress`. The caller caches this function's result only
/// after every child proof completes.
async fn prove_chunk(
    ctx: &ChunkContext<'_>,
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
            let span = ChunkSpan { agreed: agreed_timestamp, claimed: claimed_timestamp };
            let result =
                prove_span(ctx, index, span, Some(plan.synthesized), &progress.chunks[index])
                    .await;
            match &result {
                Ok(chunk) => tracing::info!(
                    span_count = chunk.spans.len(),
                    range_transition_count = chunk
                        .spans
                        .iter()
                        .map(|span| span.range_outputs.transitions.len())
                        .sum::<usize>(),
                    consolidation_transition_count = chunk
                        .spans
                        .iter()
                        .map(|span| span.consolidation_outputs.transitions.len())
                        .sum::<usize>(),
                    proofs_generated = chunk.spans.iter().all(|span| span.proofs.is_some()),
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
        let synthesized = synthesize_chunk(
            game,
            responses,
            ChunkSpan { agreed: agreed_ts, claimed: claimed_ts },
        )?;
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

/// Inputs for one in-process game-proving attempt.
#[derive(Debug)]
pub struct ProveGameRequest<'a> {
    /// Dispute game whose work owns the cache entry.
    pub game_address: Address,
    /// Immutable inputs bound into the proof.
    pub game: &'a GameProofInputs,
    /// Validated super-root responses covering the game span.
    pub responses: &'a [SuperRootAtTimestampResponse],
    /// Number of chunks used for the span.
    pub split: RangeSplitCount,
    /// Witness bytes a multi-timestamp chunk may collect.
    pub witness_limit: WitnessByteLimit,
    /// Encoded stdin bytes any one proof request may upload.
    pub stdin_limit: StdinByteLimit,
    /// Maximum number of concurrently active chunk tasks.
    pub max_concurrent: NonZeroUsize,
    /// Process-local progress retained between scheduler attempts.
    pub proof_progress: &'a InMemoryProofProgress,
}

/// Proves the game's span end to end and returns the on-chain proof bytes.
///
/// `keys` must be `Some` in network mode (resolved per-prestate through the
/// prestate cache) and may be `None` in mock mode.
pub async fn prove_game_inner(
    provider: &ProofProvider,
    keys: Option<&ProofKeys>,
    host_inputs: &HostInputs,
    request: ProveGameRequest<'_>,
) -> Result<Vec<u8>> {
    if !provider.is_mock() {
        ensure!(keys.is_some(), "network proving requires prestate proving keys");
    }
    let (plans, progress) = plan_game_attempt(provider, keys, &request)?;
    let ProveGameRequest { game, responses, witness_limit, stdin_limit, max_concurrent, .. } =
        request;
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

    let ctx =
        ChunkContext { provider, keys, host_inputs, game, responses, witness_limit, stdin_limit };
    let attempts = plans
        .into_iter()
        .map(|plan| {
            let index = plan.index;
            let attempt = prove_chunk(&ctx, plan, chunk_count, Arc::clone(&progress));
            (index, attempt)
        })
        .collect();

    let max_concurrent = max_concurrent.get().min(chunk_count);
    let results =
        run_chunk_attempts_with_progress(attempts, Arc::clone(&progress), max_concurrent).await?;

    let mut chunks = vec![None; chunk_count];
    for result in results {
        let index = result.index;
        chunks[index] = Some(result);
    }
    let chunks = collect_indexed(chunks, "chunk results")?;
    let spans: Vec<&SpanResult> = chunks.iter().flat_map(|chunk| chunk.spans.iter()).collect();
    let range_outputs = spans.iter().map(|span| span.range_outputs.clone()).collect();
    let consolidation_outputs =
        spans.iter().map(|span| span.consolidation_outputs.clone()).collect();

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
            let (range_proofs, consolidation_proofs): (Vec<_>, Vec<_>) = spans
                .iter()
                .enumerate()
                .map(|(position, span)| {
                    let proofs = span
                        .proofs
                        .as_ref()
                        .ok_or_else(|| anyhow!("missing proofs for span {position}"))?;
                    Ok((Arc::clone(&proofs.range), Arc::clone(&proofs.consolidation)))
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
            let child_proofs = stdin.proofs.len();
            let aggregation = complete_request(
                &progress.aggregation_request,
                async move || {
                    let bytes = stdin_encoded_len(&stdin)?;
                    if bytes > stdin_limit.get() {
                        // Bisection only adds child proofs, so this needs an operator.
                        ProposerGauge::AggregationStdinTooLarge.increment(1.0);
                        let err =
                            StdinTooLarge { kind: "aggregation", bytes, limit: stdin_limit.get() };
                        return Err(anyhow::Error::new(GameUnprovable(format!(
                            "aggregation over {child_proofs} child proofs does not fit one proof \
                             request: {err}"
                        ))));
                    }
                    provider.request_aggregation_proof(keys, stdin).await
                },
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
        SuperV1, WitnessTooLarge,
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
            spans: vec![SpanResult {
                range_outputs: inputs.range_outputs[index].clone(),
                consolidation_outputs: inputs.consolidation_outputs[index].clone(),
                proofs: None,
            }],
        }
    }

    /// Records every span it is asked to prove and returns a scripted outcome.
    struct ScriptedSpanProver<F> {
        calls: Mutex<Vec<(u64, u64)>>,
        outcome: F,
    }

    impl<F> ScriptedSpanProver<F>
    where
        F: Fn(ChunkSpan, &ChunkProgress) -> Result<SpanResult> + Sync,
    {
        fn new(outcome: F) -> Self {
            Self { calls: Mutex::default(), outcome }
        }

        fn calls(&self) -> Vec<(u64, u64)> {
            self.calls.lock().clone()
        }
    }

    impl<F> SpanProver for ScriptedSpanProver<F>
    where
        F: Fn(ChunkSpan, &ChunkProgress) -> Result<SpanResult> + Sync,
    {
        fn prove_leaf<'s>(
            &'s self,
            span: ChunkSpan,
            _synthesized: Option<SynthesizedExecution>,
            node: &'s ChunkProgress,
        ) -> BoxFuture<'s, Result<SpanResult>> {
            self.calls.lock().push((span.agreed, span.claimed));
            let outcome = (self.outcome)(span, node);
            async move { outcome }.boxed()
        }
    }

    fn span_result(span: ChunkSpan) -> SpanResult {
        let inputs = valid_aggregation_inputs();
        let timestamps = TimestampSpan::new(span.agreed + 1, span.claimed).unwrap();
        let mut range_outputs = inputs.range_outputs[0].clone();
        range_outputs.span = timestamps;
        let mut consolidation_outputs = inputs.consolidation_outputs[0].clone();
        consolidation_outputs.span = timestamps;
        SpanResult { range_outputs, consolidation_outputs, proofs: None }
    }

    /// The `(agreed, claimed)` pairs of a result's spans, in order.
    fn result_spans(result: &ChunkResult) -> Vec<(u64, u64)> {
        result
            .spans
            .iter()
            .map(|span| (span.range_outputs.span.start - 1, span.range_outputs.span.end))
            .collect()
    }

    fn witness_too_large() -> anyhow::Error {
        anyhow::Error::new(WitnessTooLarge::Collected { bytes: 2, limit: 1 })
    }

    fn unexecutable() -> anyhow::Error {
        ProofWaitError::Terminal {
            proof_id: B256::ZERO,
            state: ProofTerminalState::Unexecutable,
            fulfillment_status: 0,
            execution_status: 0,
        }
        .into()
    }

    #[tokio::test]
    async fn oversized_span_bisects_until_witness_fits() {
        let prover = ScriptedSpanProver::new(|span, _| {
            if span.claimed - span.agreed > 2 {
                Err(witness_too_large())
            } else {
                Ok(span_result(span))
            }
        });
        let node = ChunkProgress::default();

        let result = prove_span(&prover, 0, ChunkSpan { agreed: 100, claimed: 108 }, None, &node)
            .await
            .unwrap();

        assert_eq!(result_spans(&result), vec![(100, 102), (102, 104), (104, 106), (106, 108)]);
        assert_eq!(
            prover.calls(),
            vec![
                (100, 108),
                (100, 104),
                (100, 102),
                (102, 104),
                (104, 108),
                (104, 106),
                (106, 108)
            ]
        );
        assert!(node.split.lock().is_some());
    }

    #[tokio::test]
    async fn bisected_retry_reuses_halves_and_completed_leaves() {
        let node = ChunkProgress::default();
        let span = ChunkSpan { agreed: 100, claimed: 108 };
        let failing = ScriptedSpanProver::new(|span, _| match (span.agreed, span.claimed) {
            (106, 108) => Err(anyhow!("transient")),
            _ if span.claimed - span.agreed > 2 => Err(witness_too_large()),
            _ => Ok(span_result(span)),
        });
        let err = prove_span(&failing, 0, span, None, &node).await.err().unwrap();
        assert!(err.to_string().contains("transient"), "{err:#}");

        let retry = ScriptedSpanProver::new(|span, _| Ok(span_result(span)));
        let result = prove_span(&retry, 0, span, None, &node).await.unwrap();

        assert_eq!(retry.calls(), vec![(106, 108)]);
        assert_eq!(result_spans(&result), vec![(100, 102), (102, 104), (104, 106), (106, 108)]);
    }

    #[tokio::test]
    async fn single_timestamp_witness_overflow_is_unprovable() {
        let prover = ScriptedSpanProver::new(|_, _| Err(witness_too_large()));
        let node = ChunkProgress::default();

        let err = prove_span(&prover, 0, ChunkSpan { agreed: 100, claimed: 101 }, None, &node)
            .await
            .err()
            .unwrap();

        assert!(is_unprovable(&err), "{err:#}");
        assert!(node.split.lock().is_none());
    }

    #[tokio::test]
    async fn unexecutable_span_bisects_and_preflight_ignores_superseded_slot() {
        let start = response_at(100, 0x01, 5);
        let end = response_at(104, 0x02, 5);
        let cache = InMemoryProofProgress::default();
        let progress = cache.load_or_create(
            Address::repeat_byte(0x49),
            attempt_identity(game_for(&start, &end, 10)),
        );
        let prover = ScriptedSpanProver::new(|span, node| {
            if span == (ChunkSpan { agreed: 100, claimed: 104 }) {
                *node.range_request.lock() =
                    RequestState::Terminal(ProofTerminalState::Unexecutable);
                return Err(unexecutable());
            }
            Ok(span_result(span))
        });

        let result = prove_span(
            &prover,
            0,
            ChunkSpan { agreed: 100, claimed: 104 },
            None,
            &progress.chunks[0],
        )
        .await
        .unwrap();

        assert_eq!(result_spans(&result), vec![(100, 102), (102, 104)]);
        progress.check_terminal_requests().unwrap();
        assert!(!cache.request_counts().contains_key(&("range", "terminal")));
    }

    #[tokio::test]
    async fn single_timestamp_unexecutable_stays_sticky() {
        let prover = ScriptedSpanProver::new(|_, _| Err(unexecutable()));
        let node = ChunkProgress::default();

        let err = prove_span(&prover, 0, ChunkSpan { agreed: 100, claimed: 101 }, None, &node)
            .await
            .err()
            .unwrap();

        assert!(err.downcast_ref::<ProofWaitError>().is_some(), "{err:#}");
        assert!(!is_unprovable(&err));
        assert!(node.split.lock().is_none());
    }

    #[tokio::test]
    async fn oversized_stdin_bisects_then_single_timestamp_is_unprovable() {
        let prover = ScriptedSpanProver::new(|_, _| {
            Err(anyhow::Error::new(StdinTooLarge { kind: "range", bytes: 2, limit: 1 }))
        });
        let node = ChunkProgress::default();

        let err = prove_span(&prover, 0, ChunkSpan { agreed: 100, claimed: 102 }, None, &node)
            .await
            .err()
            .unwrap();

        assert!(is_unprovable(&err), "{err:#}");
        assert_eq!(prover.calls(), vec![(100, 102), (100, 101)]);
        assert!(node.split.lock().is_some());
    }

    #[test]
    fn stdin_encoded_len_matches_full_encoding() {
        let mut stdin = SP1Stdin::new();
        stdin.write(&0x0102_0304_u64);
        stdin.write_slice(&[7u8; 1000]);
        stdin.write(&vec![1u32, 2, 3]);
        stdin.write_slice(&[]);

        let full = bincode1::serialize(&stdin).unwrap();

        assert_eq!(stdin_encoded_len(&stdin).unwrap(), full.len() as u64);
        assert_eq!(stdin_encoded_len(&SP1Stdin::new()).unwrap(), 24);
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
            witness_limit: WitnessByteLimit::default(),
            stdin_limit: StdinByteLimit::default(),
            max_concurrent: NonZeroUsize::MIN,
            proof_progress: &cache,
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
                witness_limit: WitnessByteLimit::default(),
                stdin_limit: StdinByteLimit::default(),
                max_concurrent: NonZeroUsize::MIN,
                proof_progress: &cache,
            };
            let (_, progress) = plan_game_attempt(&provider, None, &request).unwrap();
            let mut visited = 0;
            progress.for_each_request(|_, _, request| {
                if visited == slot {
                    *request.lock() = RequestState::Terminal(state);
                }
                visited += 1;
            });
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
        };
        let first = cache.load_or_create(game_address, identity.clone());

        identity.provider = ProviderIdentity::Network {
            range_vkey: B256::repeat_byte(0x12),
            aggregation_vkey: B256::repeat_byte(0x11),
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
