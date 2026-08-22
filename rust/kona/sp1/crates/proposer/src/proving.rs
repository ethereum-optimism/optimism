//! Super-root witness collection and aggregation proof assembly.
//!
//! Witnesses come from `kona-sp1-super-range-executor`'s `InteropHost`, and aggregation inputs
//! use [`SuperAggregationInputs`]. The aggregation guest validates one shared `l1_head`.
//! Both proof providers share span chunking, native witness and output collection, and
//! aggregation-input validation. The mock provider then returns placeholder bytes; the network
//! provider proves each chunk in compressed mode and aggregates them with PLONK.

use std::{future::Future, num::NonZeroUsize};

use alloy_primitives::B256;
use anyhow::{Context, Result, anyhow, bail, ensure};
use futures::stream::{FuturesUnordered, StreamExt};
use kona_sp1_client_utils::super_root::{SuperAggregationInputs, TimestampSpan};
use kona_sp1_host_utils::metrics::MetricsGauge;
use kona_sp1_super_range_executor::{
    HostInputs, SuperRootAtTimestampResponse, build_interop_host, build_super_consolidation_stdin,
    build_super_range_stdin, collect_consolidation_witness, collect_range_witness,
    proof_from_super_v1, synthesize_execution,
};
use sp1_sdk::{SP1Proof, SP1Stdin};

pub use crate::ports::GameProofInputs;

use crate::{
    config::RangeSplitCount,
    metrics::ProposerGauge,
    ports::SuperRootSource,
    prover::{MOCK_PROOF_BYTES, ProofKeys, ProofProvider},
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

/// One proven (or natively executed) span chunk, tagged with its position.
struct ChunkResult {
    index: usize,
    range_outputs: kona_sp1_client_utils::super_root::SuperRangeOutputs,
    consolidation_outputs: kona_sp1_client_utils::super_root::SuperConsolidationOutputs,
    /// `Some((range_proof, consolidation_proof))` in network mode.
    proofs: Option<(SP1Proof, SP1Proof)>,
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

/// Proves the game's span end to end and returns the on-chain proof bytes.
///
/// `keys` must be `Some` in network mode (resolved per-prestate through the
/// prestate cache) and may be `None` in mock mode.
pub async fn prove_game_inner(
    provider: &ProofProvider,
    keys: Option<&ProofKeys>,
    host_inputs: &HostInputs,
    game: &GameProofInputs,
    responses: &[SuperRootAtTimestampResponse],
    split: RangeSplitCount,
    max_concurrent: NonZeroUsize,
) -> Result<Vec<u8>> {
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

    let tasks = chunks.into_iter().enumerate().map(|(index, (agreed_ts, claimed_ts))| {
        // Responses cover `starting_ts..=claim_ts`; this chunk needs
        // `agreed_ts..=claimed_ts`.
        let first = (agreed_ts - game.starting_ts) as usize;
        let last = (claimed_ts - game.starting_ts) as usize;
        let chunk_responses = &responses[first..=last];
        async move {
            // `split` yields `(agreed, claimed]` chunks; the guest span
            // names the claimed timestamps.
            let span = TimestampSpan::new(agreed_ts + 1, claimed_ts)
                .map_err(|err| anyhow!("invalid chunk span: {err:?}"))?;
            let synthesized =
                synthesize_execution(span, game.l1_head, game.l1_head_number, chunk_responses)
                    .with_context(|| {
                        format!("failed to synthesize span {}..={}", span.start, span.end)
                    })?;

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
            .with_context(|| {
                format!("consolidation witness collection failed for span {span:?}")
            })?;

            let proofs = match (provider, keys) {
                (ProofProvider::Mock(_), _) => None,
                (provider, Some(keys)) => {
                    let range_stdin =
                        build_super_range_stdin(&synthesized.range_inputs, range_witness)?;
                    let range_proof = provider.generate_range_proof(keys, range_stdin).await?;
                    let consolidation_stdin = build_super_consolidation_stdin(
                        &synthesized.consolidation_inputs,
                        consolidation_witness,
                    )?;
                    // Both guest modes run the super-range ELF, so both
                    // proofs use the range proving key.
                    let consolidation_proof =
                        provider.generate_range_proof(keys, consolidation_stdin).await?;
                    Some((range_proof.proof, consolidation_proof.proof))
                }
                (_, None) => bail!("network proving requires prestate proving keys"),
            };

            Ok::<_, anyhow::Error>(ChunkResult {
                index,
                range_outputs,
                consolidation_outputs,
                proofs,
            })
        }
    });

    let max_concurrent = max_concurrent.get().min(chunk_count);
    let results: Vec<ChunkResult> = run_chunk_tasks_until_error(tasks, max_concurrent).await?;

    // Chunk tasks complete out of order; restore chunk order by index.
    let mut range_outputs = vec![None; chunk_count];
    let mut consolidation_outputs = vec![None; chunk_count];
    let mut proofs = vec![None; chunk_count];
    for result in results {
        range_outputs[result.index] = Some(result.range_outputs);
        consolidation_outputs[result.index] = Some(result.consolidation_outputs);
        proofs[result.index] = result.proofs;
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
    // Invalid aggregation inputs must not reach proof generation.
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
            // Child proofs ride the SP1 proof stream: all range-mode proofs
            // in chunk order, then all consolidation-mode proofs in chunk
            // order (the aggregation guest consumes them in exactly this
            // order). Both verify against the super-range vkey.
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
                let SP1Proof::Compressed(compressed) = proof else {
                    bail!("child proof is not in compressed form");
                };
                stdin.write_proof(*compressed, keys.range_vk.vk.clone());
            }
            stdin.write(&inputs);

            let aggregation = provider.generate_agg_proof(keys, stdin).await?;
            // Never submit a proof whose public values differ from the
            // tuple `prove()` reconstructs on-chain.
            let expected = inputs.zk_dispute_game_public_values();
            ensure!(
                aggregation.public_values.as_slice() == expected.as_slice(),
                "aggregation proof public values do not match the game's expected tuple",
            );
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
    use std::sync::{
        Arc,
        atomic::{AtomicBool, AtomicUsize, Ordering},
    };

    use alloy_primitives::{Address, U256};
    use kona_sp1_client_utils::test_utils::valid_aggregation_inputs;
    use kona_sp1_super_range_executor::{
        BlockId, ChainId, ChainIdAndOutput, SuperRootResponseData, SuperV1,
    };

    use super::*;

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
