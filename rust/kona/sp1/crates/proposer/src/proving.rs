//! Super-root witness collection and aggregation proof assembly.
//!
//! Witnesses come from `kona-sp1-super-range-executor`'s `InteropHost`, and aggregation inputs
//! use [`SuperAggregationInputs`]. The aggregation guest validates one shared `l1_head`.
//! Both proof providers share span chunking, native witness and output collection, and
//! aggregation-input validation. The mock provider then returns placeholder bytes; the network
//! provider proves each chunk in compressed mode and aggregates them with PLONK.

use std::{collections::HashMap, future::Future, num::NonZeroUsize, ops::AsyncFnOnce, sync::Arc};

use alloy_primitives::{Address, B256, U256, keccak256};
use alloy_sol_types::SolValue;
use anyhow::{Context, Result, anyhow, bail, ensure};
use futures::stream::{FuturesUnordered, StreamExt};
use kona_sp1_client_utils::{
    super_root::{
        SuperAggregationInputs, SuperAggregationPublicValues, SuperConsolidationInputs,
        SuperConsolidationOutputs, SuperRangeInputs, SuperRangeOutputs, TimestampSpan,
    },
    witness::DefaultWitnessData,
};
use kona_sp1_host_utils::metrics::MetricsGauge;
use kona_sp1_super_range_executor::{
    HostInputs, SuperRootAtTimestampResponse, SynthesizedExecution, build_interop_host,
    build_super_consolidation_stdin, build_super_range_stdin, collect_consolidation_witness,
    collect_range_witness, decode_super_consolidation_public_values,
    decode_super_range_public_values, proof_from_super_v1, synthesize_execution,
};
use parking_lot::Mutex;
use serde::Serialize;
use sp1_sdk::{HashableKey, SP1Proof, SP1ProofWithPublicValues, SP1Stdin};
use tracing::Instrument;

pub use crate::ports::GameProofInputs;

use crate::{
    config::RangeSplitCount,
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

#[derive(Default)]
struct ChunkProgress {
    completed: Mutex<Option<Arc<ChunkResult>>>,
    range_request: Mutex<RequestState>,
    consolidation_request: Mutex<RequestState>,
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

    fn has_owned_requests(&self) -> bool {
        let owned = |state: &Mutex<RequestState>| {
            matches!(*state.lock(), RequestState::Submitting | RequestState::Submitted(_))
        };
        self.chunks
            .iter()
            .any(|chunk| owned(&chunk.range_request) || owned(&chunk.consolidation_request)) ||
            owned(&self.aggregation_request)
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
    ) -> Result<Arc<GameProgress>> {
        let mut games = self.games.lock();
        if let Some(progress) = games.get(&game_address) {
            if progress.identity == identity {
                return Ok(Arc::clone(progress));
            }
            ensure!(
                !progress.has_owned_requests(),
                "proving inputs changed while SPN requests remain owned"
            );
        }

        let progress = Arc::new(GameProgress::new(identity));
        games.insert(game_address, Arc::clone(&progress));
        Ok(progress)
    }

    pub(crate) fn clear(&self, game_address: Address) {
        self.games.lock().remove(&game_address);
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
    matches!(
        outcome,
        ProofTerminalState::Cancelled |
            ProofTerminalState::Expired |
            ProofTerminalState::Reverted |
            ProofTerminalState::Unfulfillable
    )
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

type ChildProofs = (Arc<SP1ProofWithPublicValues>, Arc<SP1ProofWithPublicValues>);

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
    inputs: &SuperConsolidationInputs,
    witness: DefaultWitnessData,
    expected: &SuperConsolidationOutputs,
    state: &Mutex<RequestState>,
) -> Result<Arc<SP1ProofWithPublicValues>> {
    let stdin = build_super_consolidation_stdin(inputs, witness)?;
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
    provider: &ProofProvider,
    keys: Option<&ProofKeys>,
    host_inputs: &HostInputs,
    game: &GameProofInputs,
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
            let result =
                prove_chunk_inner(provider, keys, host_inputs, game, plan, progress).await;
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
    provider: &ProofProvider,
    keys: Option<&ProofKeys>,
    host_inputs: &HostInputs,
    game: &GameProofInputs,
    plan: ChunkPlan,
    progress: Arc<GameProgress>,
) -> Result<ChunkResult> {
    let index = plan.index;
    let chunk_progress = &progress.chunks[index];
    let synthesized = &plan.synthesized;
    let span = synthesized.range_inputs.span;

    let range_host = build_interop_host(
        host_inputs,
        game.l1_head,
        &synthesized.previous_super_root_proof_bytes,
        synthesized.current_super_root,
        span.end,
    )?;
    let (range_witness, range_outputs) = collect_range_witness(
        range_host,
        &synthesized.range_inputs,
        &synthesized.preloaded_preimages,
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
                &chunk_progress.range_request,
            )
            .await?;
            let consolidation_proof = prove_consolidation_request(
                provider,
                keys,
                &synthesized.consolidation_inputs,
                consolidation_witness,
                &consolidation_outputs,
                &chunk_progress.consolidation_request,
            )
            .await?;
            Some((range_proof, consolidation_proof))
        }
        (_, None) => bail!("network proving requires prestate proving keys"),
    };

    Ok(ChunkResult { index, range_outputs, consolidation_outputs, proofs })
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
    let ProveGameRequest { game_address, game, responses, split, max_concurrent, proof_progress } =
        request;
    if !provider.is_mock() {
        ensure!(keys.is_some(), "network proving requires prestate proving keys");
    }
    let chunks = split.split(game.starting_ts, game.claim_ts)?;
    let chunk_count = chunks.len();
    tracing::info!(
        starting_ts = game.starting_ts,
        claim_ts = game.claim_ts,
        chunks = chunk_count,
        "Proving game span"
    );

    let mut plans = Vec::with_capacity(chunk_count);
    let mut chunk_identities = Vec::with_capacity(chunk_count);
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
            synthesized,
        });
    }

    let progress = proof_progress.load_or_create(
        game_address,
        AttemptIdentity {
            game: game.clone(),
            provider: provider_identity(provider, keys)?,
            chunks: chunk_identities,
        },
    )?;

    if !provider.is_mock() {
        let fulfilled = match &*progress.aggregation_request.lock() {
            RequestState::Fulfilled(proof) => Some(Arc::clone(proof)),
            RequestState::Terminal(outcome) => {
                bail!("aggregation proof request reached terminal state: {outcome:?}")
            }
            _ => None,
        };
        if let Some(proof) = fulfilled {
            let keys = keys.context("network proving requires prestate proving keys")?;
            provider.verify_aggregation_proof(keys, &proof)?;
            ensure!(
                proof.public_values.as_slice() == expected_aggregation_public_values(game),
                "aggregation proof public values do not match the game's expected tuple"
            );
            return Ok(proof.bytes());
        }
    }

    let attempts = plans
        .into_iter()
        .map(|plan| {
            let index = plan.index;
            let attempt = prove_chunk(
                provider,
                keys,
                host_inputs,
                game,
                plan,
                chunk_count,
                Arc::clone(&progress),
            );
            (index, attempt)
        })
        .collect();

    let max_concurrent = max_concurrent.get().min(chunk_count);
    let results =
        run_chunk_attempts_with_progress(attempts, Arc::clone(&progress), max_concurrent).await?;

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
                .map(|(index, pair)| {
                    pair.ok_or_else(|| anyhow!("missing proofs for chunk {index}"))
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
        collections::HashMap,
        fmt::Write,
        sync::{
            Arc, Mutex as StdMutex,
            atomic::{AtomicBool, AtomicUsize, Ordering},
        },
    };

    use alloy_primitives::{Address, U256};
    use futures::StreamExt;
    use kona_sp1_client_utils::test_utils::valid_aggregation_inputs;
    use kona_sp1_super_range_executor::{
        BlockId, ChainId, ChainIdAndOutput, SuperRootResponseData, SuperV1,
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
        let super_v1 = SuperV1 {
            timestamp,
            chains: vec![ChainIdAndOutput {
                chain_id: ChainId(U256::from(10)),
                output: B256::repeat_byte(output_byte),
            }],
        };
        let proof = proof_from_super_v1(&super_v1).unwrap();
        let super_root = B256::from(*hash_super_root_proof(&proof).unwrap());
        SuperRootAtTimestampResponse {
            current_l1: BlockId { number: required_l1 + 1, ..Default::default() },
            current_safe_timestamp: timestamp,
            current_local_safe_timestamp: timestamp,
            current_finalized_timestamp: timestamp,
            optimistic_at_timestamp: Default::default(),
            chain_ids: vec![ChainId(U256::from(10))],
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

    #[tokio::test]
    async fn retry_reuses_completed_chunks_and_changed_identity_invalidates_them() {
        let start = response_at(100, 0x01, 5);
        let end = response_at(101, 0x02, 5);
        let game = game_for(&start, &end, 10);
        let address = Address::repeat_byte(0x46);
        let cache = InMemoryProofProgress::default();
        let generations = Arc::new([AtomicUsize::new(0), AtomicUsize::new(0)]);
        let first_progress = cache.load_or_create(address, attempt_identity(game.clone())).unwrap();
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
        let changed_progress = cache.load_or_create(address, attempt_identity(changed)).unwrap();
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
    fn changed_identity_does_not_discard_submitted_request() {
        let start = response_at(100, 0x01, 5);
        let end = response_at(101, 0x02, 5);
        let game = game_for(&start, &end, 10);
        let address = Address::repeat_byte(0x48);
        let cache = InMemoryProofProgress::default();
        let progress = cache.load_or_create(address, attempt_identity(game.clone())).unwrap();
        *progress.chunks[0].range_request.lock() = RequestState::Submitted(B256::repeat_byte(0x49));

        let mut changed = game;
        changed.root_claim = B256::repeat_byte(0x4a);
        let err = cache.load_or_create(address, attempt_identity(changed)).err().unwrap();

        assert!(err.to_string().contains("requests remain owned"));
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

        let tasks = (0..3).map(|index| {
            let admitted_started = Arc::clone(&admitted_started);
            let error_reported = Arc::clone(&error_reported);
            let release_admitted = Arc::clone(&release_admitted);
            let admitted_completed = Arc::clone(&admitted_completed);
            let queued_started = Arc::clone(&queued_started);
            async move {
                match index {
                    0 => {
                        admitted_started.notify_one();
                        release_admitted.notified().await;
                        admitted_completed.store(true, Ordering::SeqCst);
                        Ok(index)
                    }
                    1 => {
                        admitted_started.notified().await;
                        error_reported.notify_one();
                        Err(anyhow::anyhow!("chunk B failed"))
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

        assert_eq!(result.unwrap_err().to_string(), "chunk B failed");
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
