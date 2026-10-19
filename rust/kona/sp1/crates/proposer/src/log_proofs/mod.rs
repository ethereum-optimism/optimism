//! Log-commitment batch proofs at proof time.
//!
//! A game proves its consolidations on one path. On the receipts path, the consolidation reads
//! the receipts of every block below a local-safe head that a message references. On the table
//! path, the proposer first proves every such block in log-commitment batches, and each
//! consolidation checks its messages against the batches' entries.

pub(crate) mod scan;

use std::{
    collections::{BTreeMap, BTreeSet},
    time::Duration,
};

use alloy_primitives::B256;
use anyhow::{Context, Result, anyhow, bail, ensure};
use async_trait::async_trait;
use kona_sp1_client_utils::{log_commitment::BlockReceipts, super_root::TimestampSpan};
use kona_sp1_super_range_executor::{
    CollectedConsolidation, ConsolidationCost, LogBatchBlock, MissingLogCommitment,
    ScannedTimestampCost, batch_log_blocks, bound_receipts_path_pgu,
    bound_receipts_path_witness_bytes, build_log_commitment_stdin_from_receipts,
    estimate_log_block_pgu, expected_batch_public_values, receipts_trie_bytes,
};
use sha2::{Digest, Sha256};
use sp1_sdk::{SP1Proof, SP1ProofWithPublicValues, SP1PublicValues, SP1Stdin};

use crate::{
    config::{LogPath, StdinByteLimit},
    ports::L2BlockSource,
    prover::{PROOF_STATUS_POLL_INTERVAL, ProofId, ProofKeys, ProofProvider, ProofWaitError},
    proving::ensure_stdin_fits,
};
use scan::ScannedSpan;

/// Requests fill to this share of their gas limit, by estimate.
const REQUEST_FILL_PERCENT: u64 = 80;

/// The share of `gas_limit` that a request may fill by estimate.
pub(crate) const fn request_budget(gas_limit: u64) -> u64 {
    gas_limit / 100 * REQUEST_FILL_PERCENT
}

/// The receipts-path bounds of one consolidation request, before its collection.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub(crate) struct RequestBound {
    pub(crate) pgu: u64,
    pub(crate) witness_bytes: u64,
}

impl RequestBound {
    pub(crate) fn of(costs: &[ScannedTimestampCost]) -> Self {
        Self {
            pgu: bound_receipts_path_pgu(costs),
            witness_bytes: bound_receipts_path_witness_bytes(costs),
        }
    }
}

/// The path of a game, with the largest receipts-path bound of its consolidation requests.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub(crate) struct PathChoice {
    pub(crate) table: bool,
    pub(crate) largest_bound: u64,
}

/// Chooses the table path when `setting` forces it, or when a request's receipts-path bound is
/// not under the budget of `gas_limit` or its witness bound is over `stdin_limit`.
pub(crate) fn choose_log_path(
    setting: LogPath,
    bounds: &[RequestBound],
    gas_limit: u64,
    stdin_limit: StdinByteLimit,
) -> PathChoice {
    let largest_bound = bounds.iter().map(|bound| bound.pgu).max().unwrap_or_default();
    let over = bounds.iter().any(|bound| {
        bound.pgu >= request_budget(gas_limit) || bound.witness_bytes > stdin_limit.get()
    });
    let table = match setting {
        LogPath::Receipts => false,
        LogPath::Table => true,
        LogPath::Auto => over,
    };
    PathChoice { table, largest_bound }
}

/// One log-commitment batch of a game.
#[derive(Debug, Clone, PartialEq, Eq)]
pub(crate) struct LogBatch {
    /// `sha256(expected)`.
    pub(crate) id: B256,
    pub(crate) blocks: Vec<LogBatchBlock>,
    /// The public values that the batch's proof must commit.
    pub(crate) expected: Vec<u8>,
}

impl std::fmt::Display for LogBatch {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        let (first, last) = (self.blocks.first(), self.blocks.last());
        write!(f, "log batch {} of {} blocks", self.id, self.blocks.len())?;
        if let (Some(first), Some(last)) = (first, last) {
            write!(
                f,
                " (chain {} block {} to chain {} block {})",
                first.chain_id, first.number, last.chain_id, last.number
            )?;
        }
        Ok(())
    }
}

/// Makes the table path's batches for the blocks of `scan`. Each block's entry comes from the
/// program's own function over its receipts.
pub(crate) async fn make_log_batches(
    l2: &dyn L2BlockSource,
    scan: &ScannedSpan,
    log_gas_limit: u64,
    stdin_limit: StdinByteLimit,
) -> Result<Vec<LogBatch>> {
    let mut entries = BTreeMap::new();
    let mut blocks = Vec::with_capacity(scan.blocks.len());
    for block in scan.blocks.values() {
        if entries.contains_key(&block.receipts_root) {
            continue;
        }
        let receipts = l2.receipts(block.chain_id, block.hash, block.receipts_root).await?;
        let pgu = estimate_log_block_pgu(&receipts);
        let witness_bytes = receipts_trie_bytes(&receipts);
        let stdin = build_log_commitment_stdin_from_receipts(&[BlockReceipts {
            receipts_root: block.receipts_root,
            receipts,
        }])?;
        entries.insert(block.receipts_root, expected_batch_public_values(&stdin));
        blocks.push(LogBatchBlock {
            chain_id: block.chain_id,
            number: block.number,
            hash: block.hash,
            receipts_root: block.receipts_root,
            pgu,
            witness_bytes,
        });
    }
    Ok(batch_log_blocks(blocks, request_budget(log_gas_limit), stdin_limit.get())
        .into_iter()
        .map(|blocks| {
            let expected: Vec<u8> =
                blocks.iter().flat_map(|block| entries[&block.receipts_root].clone()).collect();
            LogBatch {
                id: B256::from(<[u8; 32]>::from(Sha256::digest(&expected))),
                blocks,
                expected,
            }
        })
        .collect())
}

/// The stdin of `batch`, from its blocks' receipts, checked against `limit`.
pub(crate) async fn batch_stdin(
    l2: &dyn L2BlockSource,
    batch: &LogBatch,
    limit: StdinByteLimit,
) -> Result<SP1Stdin> {
    let mut blocks = Vec::with_capacity(batch.blocks.len());
    for block in &batch.blocks {
        let receipts = l2.receipts(block.chain_id, block.hash, block.receipts_root).await?;
        blocks.push(BlockReceipts { receipts_root: block.receipts_root, receipts });
    }
    let stdin = build_log_commitment_stdin_from_receipts(&blocks)?;
    ensure_stdin_fits("log", &stdin, limit)?;
    Ok(stdin)
}

/// The indices of the batches that hold a root of `served`, in batch order. Batches hold disjoint
/// roots, so this is the only cover.
pub(crate) fn cover(batches: &[&LogBatch], served: &[B256]) -> Result<Vec<usize>> {
    let batch_of: BTreeMap<_, _> = batches
        .iter()
        .enumerate()
        .flat_map(|(index, batch)| {
            batch.blocks.iter().map(move |block| (block.receipts_root, index))
        })
        .collect();
    let indices = served
        .iter()
        .map(|root| {
            batch_of
                .get(root)
                .copied()
                .with_context(|| format!("served receipts root {root} is in no batch"))
        })
        .collect::<Result<BTreeSet<_>>>()?;
    Ok(indices.into_iter().collect())
}

/// Submits, polls, and checks the proof request of a batch.
#[async_trait]
pub(crate) trait LogBatchProver: Send + Sync {
    async fn submit(&self, batch: &LogBatch) -> Result<ProofId>;
    async fn poll(
        &self,
        batch: &LogBatch,
        proof_id: ProofId,
    ) -> Result<SP1ProofWithPublicValues, ProofWaitError>;
    /// Fails unless `proof` verifies with the log vkey and commits exactly `batch.expected`.
    fn verify(&self, batch: &LogBatch, proof: &SP1ProofWithPublicValues) -> Result<()>;
}

/// Status polls per second of one game's batch waits, on average.
const LOG_BATCH_POLLS_PER_SECOND: u32 = 8;

/// The status poll interval of each of `batches` concurrent batch waits.
pub(crate) fn log_batch_poll_interval(batches: usize) -> Duration {
    let spread = Duration::from_secs(1) * u32::try_from(batches).unwrap_or(u32::MAX);
    PROOF_STATUS_POLL_INTERVAL.max(spread / LOG_BATCH_POLLS_PER_SECOND)
}

/// Batch requests on the SP1 prover network.
pub(crate) struct NetworkLogProver<'a> {
    pub(crate) provider: &'a ProofProvider,
    pub(crate) keys: &'a ProofKeys,
    pub(crate) l2: &'a dyn L2BlockSource,
    pub(crate) max_stdin_bytes: StdinByteLimit,
    /// See [`log_batch_poll_interval`].
    pub(crate) poll_interval: Duration,
}

#[async_trait]
impl LogBatchProver for NetworkLogProver<'_> {
    async fn submit(&self, batch: &LogBatch) -> Result<ProofId> {
        let stdin = batch_stdin(self.l2, batch, self.max_stdin_bytes).await?;
        self.provider.request_log_proof(self.keys, stdin).await
    }

    async fn poll(
        &self,
        _batch: &LogBatch,
        proof_id: ProofId,
    ) -> Result<SP1ProofWithPublicValues, ProofWaitError> {
        self.provider.wait_for_proof_every(proof_id, self.poll_interval).await
    }

    fn verify(&self, batch: &LogBatch, proof: &SP1ProofWithPublicValues) -> Result<()> {
        self.provider.verify_log_proof(self.keys, proof)?;
        ensure!(
            proof.public_values.as_slice() == batch.expected,
            "{batch} proof commits other entries"
        );
        Ok(())
    }
}

/// The mock provider's batches: no SP1 request; a batch's proof is empty and commits its expected
/// public values. Submission still builds and checks the batch's stdin.
pub(crate) struct MockLogProver<'a> {
    pub(crate) l2: &'a dyn L2BlockSource,
    pub(crate) max_stdin_bytes: StdinByteLimit,
}

#[async_trait]
impl LogBatchProver for MockLogProver<'_> {
    async fn submit(&self, batch: &LogBatch) -> Result<ProofId> {
        batch_stdin(self.l2, batch, self.max_stdin_bytes).await?;
        Ok(batch.id)
    }

    async fn poll(
        &self,
        batch: &LogBatch,
        proof_id: ProofId,
    ) -> Result<SP1ProofWithPublicValues, ProofWaitError> {
        if proof_id != batch.id {
            return Err(ProofWaitError::Uncertain(anyhow!("mock {batch} has request {proof_id}")));
        }
        Ok(SP1ProofWithPublicValues {
            proof: SP1Proof::Core(Vec::new()),
            public_values: SP1PublicValues::from(&batch.expected),
            sp1_version: String::new(),
            tee_proof: None,
        })
    }

    fn verify(&self, batch: &LogBatch, proof: &SP1ProofWithPublicValues) -> Result<()> {
        ensure!(
            proof.public_values.as_slice() == batch.expected,
            "{batch} proof commits other entries"
        );
        Ok(())
    }
}

/// A collected consolidation request, before its checks.
pub(crate) struct CollectedRequest<'a> {
    pub(crate) span: TimestampSpan,
    pub(crate) missing: &'a [MissingLogCommitment],
    pub(crate) estimate: u64,
    pub(crate) batch_proofs: usize,
}

/// A consolidation request's estimate is not under its budget.
#[derive(Debug, Clone, PartialEq, Eq, thiserror::Error)]
#[error(
    "the consolidation of timestamps {}..={} is estimated at {estimate} PGU with {batch_proofs} \
     batch proofs and {receipts_path_blocks} blocks read through their receipts, not under \
     {budget} ({REQUEST_FILL_PERCENT}% of the range gas limit)",
    span.start,
    span.end
)]
pub(crate) struct EstimateOverBudget {
    pub(crate) span: TimestampSpan,
    pub(crate) estimate: u64,
    pub(crate) budget: u64,
    pub(crate) batch_proofs: usize,
    pub(crate) receipts_path_blocks: usize,
}

/// Checks a collected consolidation before its request: on the table path, every block below a
/// head must have had an entry; when the request's span can be split, the estimate must be under
/// the budget of `gas_limit` ([`EstimateOverBudget`]). A span that cannot be split is requested
/// whatever its estimate.
pub(crate) fn check_collected(
    table: bool,
    request: &CollectedRequest<'_>,
    gas_limit: u64,
    splittable: bool,
) -> Result<()> {
    let CollectedRequest { span, missing, estimate, batch_proofs } = *request;
    if table && !missing.is_empty() {
        let blocks: Vec<_> = missing
            .iter()
            .map(|block| {
                format!(
                    "chain {} block {} (receipts root {})",
                    block.chain_id, block.block_hash, block.receipts_root
                )
            })
            .collect();
        bail!(
            "the consolidation of timestamps {}..={} read blocks below a head without an entry, \
             which the message scan missed: {}",
            span.start,
            span.end,
            blocks.join(", "),
        );
    }
    let budget = request_budget(gas_limit);
    if splittable && estimate >= budget {
        return Err(EstimateOverBudget {
            span,
            estimate,
            budget,
            batch_proofs,
            receipts_path_blocks: missing.len(),
        }
        .into());
    }
    Ok(())
}

/// The receipts-path costs of the transition timestamps `first..last` (indices into
/// `scan.timestamps`).
pub(crate) fn receipts_path_costs(
    scan: &ScannedSpan,
    first: usize,
    last: usize,
) -> Vec<ScannedTimestampCost> {
    scan.timestamps[first..last]
        .iter()
        .map(|timestamp| {
            let mut referenced_gas_used = BTreeMap::<u64, Vec<u64>>::new();
            for reference @ (chain_id, _) in &timestamp.references {
                referenced_gas_used
                    .entry(*chain_id)
                    .or_default()
                    .push(scan.blocks[reference].gas_used);
            }
            ScannedTimestampCost {
                chains: timestamp.chains,
                replaced_chains: timestamp.replaced_chains,
                messages: timestamp.messages,
                head_receipts_bytes: timestamp.head_receipts_bytes,
                replaced_deposit_gas: timestamp.replaced_deposit_gas,
                replaced_head_gas_used: timestamp.replaced_head_gas_used,
                referenced_gas_used,
            }
        })
        .collect()
}

/// The cost of a consolidation request over `costs` with its collection and the batches it
/// carries.
pub(crate) fn consolidation_cost(
    costs: &[ScannedTimestampCost],
    collected: &CollectedConsolidation,
    batches: &[&LogBatch],
) -> ConsolidationCost {
    ConsolidationCost {
        chain_timestamps: costs.iter().map(|cost| cost.chains).sum(),
        message_checks: costs.iter().map(|cost| (cost.replaced_chains + 1) * cost.messages).sum(),
        read_bytes: collected.read_bytes,
        log_check_siblings: collected.log_check_siblings,
        batch_proofs: batches.len() as u64,
        batch_entries: batches.iter().map(|batch| batch.blocks.len() as u64).sum(),
        replaced_gas: costs.iter().map(|cost| cost.replaced_deposit_gas).sum(),
    }
}

#[cfg(test)]
mod tests {
    use super::{
        scan::{ReferencedBlock, ScannedTimestamp},
        *,
    };

    const LIMIT: u64 = 1_000;

    fn block(chain_id: u64, number: u64) -> LogBatchBlock {
        LogBatchBlock {
            chain_id,
            number,
            hash: B256::repeat_byte(number as u8),
            receipts_root: B256::left_padding_from(&[chain_id as u8, number as u8]),
            pgu: 1,
            witness_bytes: 1,
        }
    }

    fn bound(pgu: u64) -> RequestBound {
        RequestBound { pgu, witness_bytes: 0 }
    }

    fn batch(blocks: Vec<LogBatchBlock>) -> LogBatch {
        LogBatch { id: B256::repeat_byte(blocks[0].number as u8), blocks, expected: Vec::new() }
    }

    fn missing(number: u64) -> MissingLogCommitment {
        MissingLogCommitment {
            chain_id: 7,
            block_hash: B256::repeat_byte(number as u8),
            receipts_root: B256::left_padding_from(&[7, number as u8]),
        }
    }

    fn check(
        table: bool,
        missing: &[MissingLogCommitment],
        estimate: u64,
        batch_proofs: usize,
    ) -> Result<()> {
        let request = CollectedRequest { span: span(), missing, estimate, batch_proofs };
        check_collected(table, &request, LIMIT, true)
    }

    fn stdin_limit(bytes: u64) -> StdinByteLimit {
        StdinByteLimit::new(bytes).unwrap()
    }

    fn span() -> TimestampSpan {
        TimestampSpan::new(10, 12).unwrap()
    }

    /// Pins the receipts path when every request's bound is under 80% of the limit.
    #[test]
    fn path_is_receipts_when_every_request_fits() {
        let choice =
            choose_log_path(LogPath::Auto, &[bound(100), bound(799)], LIMIT, stdin_limit(1));
        assert_eq!(choice, PathChoice { table: false, largest_bound: 799 });
    }

    /// Pins the table path when a request's bound reaches 80% of the limit or the stdin limit.
    #[test]
    fn path_is_table_when_any_request_is_over() {
        let choice =
            choose_log_path(LogPath::Auto, &[bound(100), bound(800)], LIMIT, stdin_limit(1));
        assert_eq!(choice, PathChoice { table: true, largest_bound: 800 });
        let wide = RequestBound { pgu: 1, witness_bytes: 11 };
        assert!(!choose_log_path(LogPath::Auto, &[wide], LIMIT, stdin_limit(11)).table);
        assert!(choose_log_path(LogPath::Auto, &[wide], LIMIT, stdin_limit(10)).table);
    }

    /// Pins the forced path on either side of the limit.
    #[test]
    fn log_path_setting_forces_the_path() {
        for bounds in [[bound(10)], [bound(10_000)]] {
            assert!(!choose_log_path(LogPath::Receipts, &bounds, LIMIT, stdin_limit(1)).table);
            assert!(choose_log_path(LogPath::Table, &bounds, LIMIT, stdin_limit(1)).table);
        }
    }

    /// Pins a referenced block counted once at each timestamp that references it.
    #[test]
    fn receipts_estimate_counts_a_shared_block_once_per_timestamp() {
        let referenced = |number, gas_used| {
            (
                (7, number),
                ReferencedBlock {
                    chain_id: 7,
                    number,
                    hash: B256::ZERO,
                    receipts_root: B256::ZERO,
                    gas_used,
                },
            )
        };
        let scan = ScannedSpan {
            timestamps: vec![
                ScannedTimestamp { references: [(7, 1)].into(), ..Default::default() },
                ScannedTimestamp { references: [(7, 1), (7, 2)].into(), ..Default::default() },
            ],
            blocks: [referenced(1, 100), referenced(2, 200)].into(),
        };
        let costs = receipts_path_costs(&scan, 0, 2);
        let gas: Vec<_> = costs.iter().map(|cost| cost.referenced_gas_used.clone()).collect();
        assert_eq!(gas, [BTreeMap::from([(7, vec![100])]), BTreeMap::from([(7, vec![100, 200])])]);
        assert_eq!(receipts_path_costs(&scan, 1, 2).len(), 1);
    }

    /// Pins an error that names each block the table path read without an entry.
    #[test]
    fn table_path_missing_block_fails_with_named_block() {
        let err = check(true, &[missing(3), missing(4)], 1, 2).unwrap_err();
        let err = err.to_string();
        for number in [3u8, 4] {
            assert!(err.contains(&B256::repeat_byte(number).to_string()), "{err}");
        }
        check(false, &[missing(3)], 1, 0).unwrap();
    }

    /// Pins a typed refusal on the receipts path at 80% of the limit.
    #[test]
    fn receipts_path_over_limit_fails() {
        check(false, &[missing(3)], 799, 0).unwrap();
        let err = check(false, &[missing(3)], 800, 0).unwrap_err();
        assert_eq!(
            err.downcast_ref::<EstimateOverBudget>(),
            Some(&EstimateOverBudget {
                span: span(),
                estimate: 800,
                budget: 800,
                batch_proofs: 0,
                receipts_path_blocks: 1,
            })
        );
    }

    /// Pins a span that cannot be split requested over its estimate, but not without an entry.
    #[test]
    fn check_collected_skips_the_budget_when_not_splittable() {
        let request =
            CollectedRequest { span: span(), missing: &[], estimate: 5_000, batch_proofs: 0 };
        check_collected(true, &request, LIMIT, false).unwrap();
        let missing = [missing(3)];
        let request = CollectedRequest { missing: &missing, ..request };
        check_collected(true, &request, LIMIT, false).unwrap_err();
        check_collected(false, &request, LIMIT, false).unwrap();
    }

    /// Pins the 2 s poll interval for few batches and at most 8 polls per second for many.
    #[test]
    fn batch_poll_interval_bounds_the_poll_rate() {
        assert_eq!(log_batch_poll_interval(1), PROOF_STATUS_POLL_INTERVAL);
        assert_eq!(log_batch_poll_interval(16), PROOF_STATUS_POLL_INTERVAL);
        assert_eq!(log_batch_poll_interval(1_140), Duration::from_millis(142_500));
    }

    /// Pins a refused request on the table path when its batch proofs push it over the limit.
    #[test]
    fn over_limit_part_with_empty_missing_fails() {
        let err = check(true, &[], 900, 300).unwrap_err();
        assert!(err.to_string().contains("300 batch proofs"), "{err}");
    }

    /// Pins a cover of the batches that hold a served root, and an error for a root in none.
    #[test]
    fn cover_includes_every_served_root_and_no_missing_root() {
        let batches = [
            batch(vec![block(7, 1), block(7, 2)]),
            batch(vec![block(7, 3)]),
            batch(vec![block(8, 1)]),
        ];
        let batches: Vec<&LogBatch> = batches.iter().collect();
        let served = [block(8, 1).receipts_root, block(7, 2).receipts_root];
        assert_eq!(cover(&batches, &served).unwrap(), [0, 2]);
        assert!(cover(&batches, &[]).unwrap().is_empty());
        let err = cover(&batches, &[missing(9).receipts_root]).unwrap_err();
        assert!(err.to_string().contains("in no batch"), "{err}");
    }
}
