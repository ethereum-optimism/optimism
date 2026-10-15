//! Agreement between a block's published flashblocks and the block that sealed.
//!
//! A flashblock is a preconfirmation: the transactions it publishes for block `N` promise to
//! seal in canonical block `N`, in that order. [`AgreementTracker`] holds both sides, the
//! transaction hashes published per `(block number, payload id)` and the transaction hashes of
//! each sealed block, and reports per block whether the promise held.
//!
//! A sealed block is evaluated only once the flashblock stream has moved past it, because the
//! sealed block can be imported through the engine before its last flashblock arrives over the
//! websocket. The sealed side never advances that frontier, so a lagging stream delays the
//! verdict instead of producing a false mismatch.

use crate::FlashBlock;
use alloy_primitives::{B256, keccak256};
use alloy_rpc_types_engine::PayloadId;
use std::collections::BTreeMap;
use tracing::debug;

/// Blocks retained on each side while the other side has not arrived. Past this, the oldest
/// published block is dropped without a verdict and the oldest sealed block is judged as is.
const DEFAULT_RETAINED_BLOCKS: usize = 512;

/// Transactions published in one flashblock.
#[derive(Debug, Clone, Default, PartialEq, Eq)]
struct PublishedDelta {
    tx_hashes: Vec<B256>,
    /// The cumulative post-exec (`0x7D`) transaction carried by this flashblock, if any.
    post_exec_tx_hash: Option<B256>,
}

/// Transactions published for one block by one payload job.
#[derive(Debug, Clone, PartialEq, Eq)]
struct PublishedPayload {
    payload_id: PayloadId,
    /// Keyed by flashblock index so a duplicate or out-of-order delivery overwrites instead of
    /// appending.
    deltas_by_index: BTreeMap<u64, PublishedDelta>,
}

impl PublishedPayload {
    const fn new(payload_id: PayloadId) -> Self {
        Self { payload_id, deltas_by_index: BTreeMap::new() }
    }

    fn observe(&mut self, flashblock: &FlashBlock) {
        let delta = PublishedDelta {
            tx_hashes: flashblock.diff.transactions.iter().map(keccak256).collect(),
            post_exec_tx_hash: flashblock.diff.post_exec_tx.as_ref().map(keccak256),
        };
        self.deltas_by_index.insert(flashblock.index, delta);
    }

    /// The transaction list this payload promised, in sealing order.
    ///
    /// The producer republishes the cumulative post-exec transaction on every flashblock and the
    /// sealed block carries only the final one, appended after the other transactions, so only
    /// the highest index's post-exec transaction counts.
    fn promised_tx_hashes(&self) -> Vec<B256> {
        let mut hashes: Vec<B256> = self
            .deltas_by_index
            .values()
            .flat_map(|delta| delta.tx_hashes.iter().copied())
            .collect();
        let final_post_exec =
            self.deltas_by_index.values().next_back().and_then(|delta| delta.post_exec_tx_hash);
        hashes.extend(final_post_exec);
        hashes
    }
}

/// How the published transaction list differs from the sealed one.
#[derive(Debug, Clone, PartialEq, Eq)]
pub(crate) struct Divergence {
    /// The payload whose promise is compared: the one whose first flashblock arrived last.
    pub(crate) payload_id: PayloadId,
    /// First position at which the two lists differ.
    pub(crate) first_divergent_index: usize,
    pub(crate) published_tx_count: usize,
    pub(crate) sealed_tx_count: usize,
    /// Published transactions absent from the sealed block.
    pub(crate) unsealed_tx_count: usize,
    /// Sealed transactions never published.
    pub(crate) unpublished_tx_count: usize,
}

impl Divergence {
    fn between(payload_id: PayloadId, published: &[B256], sealed: &[B256]) -> Self {
        let first_divergent_index = published
            .iter()
            .zip(sealed)
            .position(|(published, sealed)| published != sealed)
            .unwrap_or_else(|| published.len().min(sealed.len()));
        Self {
            payload_id,
            first_divergent_index,
            published_tx_count: published.len(),
            sealed_tx_count: sealed.len(),
            unsealed_tx_count: published.iter().filter(|hash| !sealed.contains(hash)).count(),
            unpublished_tx_count: sealed.iter().filter(|hash| !published.contains(hash)).count(),
        }
    }

    const fn is_reordering(&self) -> bool {
        self.unsealed_tx_count == 0 && self.unpublished_tx_count == 0
    }
}

/// Outcome of comparing a sealed block with the flashblocks published for it.
#[derive(Debug, Clone, PartialEq, Eq)]
pub(crate) enum AgreementOutcome {
    /// One published payload promised exactly the sealed transaction list.
    Match { payload_id: PayloadId },
    /// A published transaction did not seal, or a sealed transaction was never published.
    ContentMismatch(Divergence),
    /// The same transactions sealed in a different order.
    OrderingMismatch(Divergence),
    /// No complete flashblock sequence was received for the block.
    Unobserved,
}

/// The verdict for one sealed block.
#[derive(Debug, Clone, PartialEq, Eq)]
pub(crate) struct AgreementReport {
    pub(crate) block_number: u64,
    pub(crate) outcome: AgreementOutcome,
    /// Payloads other than the one the outcome is about that published for the block: a deposed
    /// leader's stream, or a payload job the consensus client abandoned.
    pub(crate) orphaned_payloads: usize,
}

/// Holds published and sealed transaction lists per block and reports their agreement.
#[derive(Debug)]
pub(crate) struct AgreementTracker {
    /// Published payloads per block number, in the order their first flashblock arrived.
    published: BTreeMap<u64, Vec<PublishedPayload>>,
    /// Sealed transaction lists waiting for the stream to move past them.
    sealed: BTreeMap<u64, Vec<B256>>,
    /// Lowest block number a verdict can still be given for. Sealed blocks below it are
    /// ignored: they predate the first complete sequence this tracker saw, were already judged,
    /// or had their published side dropped.
    verifiable_from: Option<u64>,
    retained_blocks: usize,
}

impl Default for AgreementTracker {
    fn default() -> Self {
        Self::new(DEFAULT_RETAINED_BLOCKS)
    }
}

impl AgreementTracker {
    pub(crate) const fn new(retained_blocks: usize) -> Self {
        Self {
            published: BTreeMap::new(),
            sealed: BTreeMap::new(),
            verifiable_from: None,
            retained_blocks,
        }
    }

    /// Records a received flashblock and returns the verdicts it makes ready.
    pub(crate) fn observe_flashblock(&mut self, flashblock: &FlashBlock) -> Vec<AgreementReport> {
        let block_number = flashblock.block_number();
        if self.verifiable_from.is_some_and(|from| block_number < from) {
            return Vec::new();
        }

        let payloads = self.published.entry(block_number).or_default();
        match payloads.iter_mut().find(|payload| payload.payload_id == flashblock.payload_id) {
            Some(payload) => payload.observe(flashblock),
            // A sequence joined after its first flashblock is missing transactions, so it cannot
            // be judged. Only index 0 starts a payload.
            None if flashblock.index == 0 => {
                let mut payload = PublishedPayload::new(flashblock.payload_id);
                payload.observe(flashblock);
                payloads.push(payload);
                self.verifiable_from.get_or_insert(block_number);
            }
            None => {
                if payloads.is_empty() {
                    self.published.remove(&block_number);
                }
            }
        }

        while self.published.len() > self.retained_blocks {
            if let Some((dropped, _)) = self.published.pop_first() {
                debug!(
                    target: "flashblocks",
                    block_number = dropped,
                    "Dropping published flashblocks whose sealed block never arrived"
                );
                self.verifiable_from = Some(dropped + 1);
            }
        }
        self.evaluate_ready()
    }

    /// Records a sealed block's transaction list and returns the verdicts it makes ready.
    pub(crate) fn observe_sealed_block(
        &mut self,
        block_number: u64,
        tx_hashes: Vec<B256>,
    ) -> Vec<AgreementReport> {
        if self.verifiable_from.is_none_or(|from| block_number < from) {
            return Vec::new();
        }
        self.sealed.insert(block_number, tx_hashes);

        let mut reports = Vec::new();
        while self.sealed.len() > self.retained_blocks {
            if let Some((oldest, tx_hashes)) = self.sealed.pop_first() {
                reports.push(self.evaluate(oldest, tx_hashes));
            }
        }
        reports.extend(self.evaluate_ready());
        reports
    }

    /// Evaluates every sealed block the published stream has moved past.
    fn evaluate_ready(&mut self) -> Vec<AgreementReport> {
        let Some(frontier) = self.published.keys().next_back().copied() else {
            return Vec::new();
        };

        let mut reports = Vec::new();
        while let Some(block_number) =
            self.sealed.keys().next().copied().filter(|number| *number < frontier)
        {
            if let Some(tx_hashes) = self.sealed.remove(&block_number) {
                reports.push(self.evaluate(block_number, tx_hashes));
            }
        }
        reports
    }

    fn evaluate(&mut self, block_number: u64, sealed: Vec<B256>) -> AgreementReport {
        let published = self.published.remove(&block_number).unwrap_or_default();
        self.published.retain(|number, _| *number > block_number);
        self.verifiable_from = Some(block_number + 1);

        let outcome = classify(&published, &sealed);
        // Every payload but the one the verdict is about published for a block it did not seal.
        let orphaned_payloads = published.len().saturating_sub(1);
        AgreementReport { block_number, outcome, orphaned_payloads }
    }
}

fn classify(published: &[PublishedPayload], sealed: &[B256]) -> AgreementOutcome {
    if let Some(sealing_payload) =
        published.iter().find(|payload| payload.promised_tx_hashes() == sealed)
    {
        return AgreementOutcome::Match { payload_id: sealing_payload.payload_id };
    }
    let Some(newest) = published.last() else { return AgreementOutcome::Unobserved };

    let divergence = Divergence::between(newest.payload_id, &newest.promised_tx_hashes(), sealed);
    if divergence.is_reordering() {
        AgreementOutcome::OrderingMismatch(divergence)
    } else {
        AgreementOutcome::ContentMismatch(divergence)
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::test_utils::TestFlashBlockFactory;
    use alloy_primitives::Bytes;
    use rstest::rstest;

    fn tx(byte: u8) -> Bytes {
        Bytes::from(vec![byte; 4])
    }

    fn hash(byte: u8) -> B256 {
        keccak256(tx(byte))
    }

    fn hashes(bytes: &[u8]) -> Vec<B256> {
        bytes.iter().map(|byte| hash(*byte)).collect()
    }

    fn payload_id(byte: u8) -> PayloadId {
        PayloadId::new([byte; 8])
    }

    /// Flashblocks for one block under one payload id, one flashblock per transaction.
    fn publish(
        tracker: &mut AgreementTracker,
        block_number: u64,
        id: u8,
        tx_bytes: &[u8],
    ) -> Vec<AgreementReport> {
        let factory = TestFlashBlockFactory::new();
        let mut reports = Vec::new();
        for (index, byte) in tx_bytes.iter().enumerate() {
            let flashblock = factory
                .builder()
                .block_number(block_number)
                .index(index as u64)
                .payload_id(payload_id(id))
                .transactions(vec![tx(*byte)])
                .build();
            reports.extend(tracker.observe_flashblock(&flashblock));
        }
        reports
    }

    fn divergence(
        id: u8,
        first_divergent_index: usize,
        published_tx_count: usize,
        sealed_tx_count: usize,
        unsealed_tx_count: usize,
        unpublished_tx_count: usize,
    ) -> Divergence {
        Divergence {
            payload_id: payload_id(id),
            first_divergent_index,
            published_tx_count,
            sealed_tx_count,
            unsealed_tx_count,
            unpublished_tx_count,
        }
    }

    fn matched(id: u8) -> AgreementOutcome {
        AgreementOutcome::Match { payload_id: payload_id(id) }
    }

    fn verdicts(reports: &[AgreementReport]) -> Vec<(u64, AgreementOutcome)> {
        reports.iter().map(|report| (report.block_number, report.outcome.clone())).collect()
    }

    #[rstest]
    #[case::exact_match(&[1, 2, 3], &[1, 2, 3], matched(1))]
    #[case::sealed_dropped_tail(&[1, 2, 3], &[1, 2], AgreementOutcome::ContentMismatch(divergence(1, 2, 3, 2, 1, 0)))]
    #[case::sealed_added_tx(&[1, 2], &[1, 2, 3], AgreementOutcome::ContentMismatch(divergence(1, 2, 2, 3, 0, 1)))]
    #[case::sealed_replaced_tx(&[1, 2, 3], &[1, 9, 3], AgreementOutcome::ContentMismatch(divergence(1, 1, 3, 3, 1, 1)))]
    #[case::sealed_reordered(&[1, 2, 3], &[1, 3, 2], AgreementOutcome::OrderingMismatch(divergence(1, 1, 3, 3, 0, 0)))]
    fn single_payload_verdicts(
        #[case] published: &[u8],
        #[case] sealed: &[u8],
        #[case] expected: AgreementOutcome,
    ) {
        let mut tracker = AgreementTracker::default();
        assert!(publish(&mut tracker, 100, 1, published).is_empty());

        assert!(tracker.observe_sealed_block(100, hashes(sealed)).is_empty());
        let reports = publish(&mut tracker, 101, 2, &[1]);

        assert_eq!(
            reports,
            vec![AgreementReport { block_number: 100, outcome: expected, orphaned_payloads: 0 }]
        );
    }

    #[test]
    fn sealed_block_arriving_after_the_stream_moved_on_is_judged_at_once() {
        let mut tracker = AgreementTracker::default();
        publish(&mut tracker, 100, 1, &[1, 2]);
        publish(&mut tracker, 101, 2, &[1]);

        let reports = tracker.observe_sealed_block(100, hashes(&[1, 2]));

        assert_eq!(verdicts(&reports), vec![(100, matched(1))]);
    }

    #[test]
    fn sealed_blocks_ahead_of_the_stream_wait_for_it() {
        let mut tracker = AgreementTracker::default();
        publish(&mut tracker, 100, 1, &[1]);
        assert!(tracker.observe_sealed_block(100, hashes(&[1, 2])).is_empty());
        assert!(tracker.observe_sealed_block(101, hashes(&[3])).is_empty());

        let factory = TestFlashBlockFactory::new();
        let late =
            factory.flashblock_at(1).payload_id(payload_id(1)).transactions(vec![tx(2)]).build();
        assert!(tracker.observe_flashblock(&late).is_empty());
        let reports = publish(&mut tracker, 101, 2, &[3]);

        assert_eq!(verdicts(&reports), vec![(100, matched(1))]);
    }

    #[test]
    fn leadership_transfer_matches_the_payload_that_sealed_and_orphans_the_other() {
        let mut tracker = AgreementTracker::default();
        publish(&mut tracker, 100, 1, &[1, 2]);
        publish(&mut tracker, 100, 2, &[1, 3]);
        tracker.observe_sealed_block(100, hashes(&[1, 2]));

        let reports = publish(&mut tracker, 101, 3, &[1]);

        assert_eq!(
            reports,
            vec![AgreementReport { block_number: 100, outcome: matched(1), orphaned_payloads: 1 }]
        );
    }

    #[test]
    fn mismatch_is_reported_against_the_payload_that_started_publishing_last() {
        let mut tracker = AgreementTracker::default();
        publish(&mut tracker, 100, 1, &[1, 2]);
        publish(&mut tracker, 100, 2, &[1, 3]);
        tracker.observe_sealed_block(100, hashes(&[1, 4]));

        let reports = publish(&mut tracker, 101, 3, &[1]);

        assert_eq!(
            reports,
            vec![AgreementReport {
                block_number: 100,
                outcome: AgreementOutcome::ContentMismatch(divergence(2, 1, 2, 2, 1, 1)),
                orphaned_payloads: 1,
            }]
        );
    }

    #[test]
    fn sealed_block_without_flashblocks_is_unobserved() {
        let mut tracker = AgreementTracker::default();
        publish(&mut tracker, 100, 1, &[1]);
        tracker.observe_sealed_block(100, hashes(&[1]));
        tracker.observe_sealed_block(101, hashes(&[2]));

        let reports = publish(&mut tracker, 102, 2, &[3]);

        assert_eq!(
            verdicts(&reports),
            vec![(100, matched(1)), (101, AgreementOutcome::Unobserved)]
        );
    }

    #[test]
    fn stream_joined_mid_block_gives_no_verdict_for_that_block() {
        let mut tracker = AgreementTracker::default();
        let factory = TestFlashBlockFactory::new();
        let partial =
            factory.flashblock_at(1).payload_id(payload_id(1)).transactions(vec![tx(2)]).build();
        assert!(tracker.observe_flashblock(&partial).is_empty());

        assert!(tracker.observe_sealed_block(100, hashes(&[1, 2])).is_empty());
        publish(&mut tracker, 101, 2, &[3]);
        tracker.observe_sealed_block(101, hashes(&[3]));
        let reports = publish(&mut tracker, 102, 3, &[4]);

        assert_eq!(verdicts(&reports), vec![(101, matched(2))]);
    }

    #[test]
    fn sealed_blocks_before_the_first_complete_sequence_are_ignored() {
        let mut tracker = AgreementTracker::default();
        assert!(tracker.observe_sealed_block(50, hashes(&[1])).is_empty());
        publish(&mut tracker, 100, 1, &[1]);

        assert!(tracker.observe_sealed_block(99, hashes(&[1])).is_empty());
        tracker.observe_sealed_block(100, hashes(&[1]));
        let reports = publish(&mut tracker, 101, 2, &[2]);

        assert_eq!(verdicts(&reports), vec![(100, matched(1))]);
    }

    #[test]
    fn reorg_recommitting_a_judged_block_is_ignored() {
        let mut tracker = AgreementTracker::default();
        publish(&mut tracker, 100, 1, &[1]);
        tracker.observe_sealed_block(100, hashes(&[1]));
        publish(&mut tracker, 101, 2, &[2]);

        assert!(tracker.observe_sealed_block(100, hashes(&[9])).is_empty());
        assert!(publish(&mut tracker, 100, 9, &[9]).is_empty());
        assert!(!tracker.published.contains_key(&100));
    }

    #[rstest]
    #[case::final_post_exec_appended(Some(0x51), Some(0x52), &[1, 2, 0x52], matched(1))]
    #[case::post_exec_dropped_by_last_flashblock(Some(0x51), None, &[1, 2], matched(1))]
    #[case::stale_post_exec_not_counted(Some(0x51), None, &[1, 2, 0x51], AgreementOutcome::ContentMismatch(divergence(1, 2, 2, 3, 0, 1)))]
    fn only_the_highest_index_post_exec_tx_counts(
        #[case] first_post_exec: Option<u8>,
        #[case] last_post_exec: Option<u8>,
        #[case] sealed: &[u8],
        #[case] expected: AgreementOutcome,
    ) {
        let mut tracker = AgreementTracker::default();
        let factory = TestFlashBlockFactory::new();
        let mut fb0 = factory.flashblock_at(0).payload_id(payload_id(1)).transactions(vec![tx(1)]);
        if let Some(byte) = first_post_exec {
            fb0 = fb0.post_exec_tx(tx(byte));
        }
        let fb0 = fb0.build();
        let mut fb1 = factory.flashblock_after(&fb0).transactions(vec![tx(2)]);
        if let Some(byte) = last_post_exec {
            fb1 = fb1.post_exec_tx(tx(byte));
        }
        tracker.observe_flashblock(&fb0);
        tracker.observe_flashblock(&fb1.build());
        tracker.observe_sealed_block(100, hashes(sealed));

        let reports = publish(&mut tracker, 101, 2, &[3]);

        assert_eq!(verdicts(&reports), vec![(100, expected)]);
    }

    #[test]
    fn redelivered_flashblock_does_not_duplicate_its_transactions() {
        let mut tracker = AgreementTracker::default();
        let factory = TestFlashBlockFactory::new();
        let fb0 =
            factory.flashblock_at(0).payload_id(payload_id(1)).transactions(vec![tx(1)]).build();
        tracker.observe_flashblock(&fb0);
        tracker.observe_flashblock(&fb0);
        tracker.observe_sealed_block(100, hashes(&[1]));

        let reports = publish(&mut tracker, 101, 2, &[2]);

        assert_eq!(verdicts(&reports), vec![(100, matched(1))]);
    }

    #[test]
    fn published_blocks_past_retention_are_dropped_without_a_verdict() {
        let mut tracker = AgreementTracker::new(2);
        for block_number in 100..=102 {
            publish(&mut tracker, block_number, 1, &[1]);
        }

        assert!(tracker.observe_sealed_block(100, hashes(&[1])).is_empty());
        let reports = tracker.observe_sealed_block(101, hashes(&[1]));

        assert_eq!(verdicts(&reports), vec![(101, matched(1))]);
    }

    #[test]
    fn sealed_blocks_past_retention_are_judged_without_waiting() {
        let mut tracker = AgreementTracker::new(2);
        publish(&mut tracker, 100, 1, &[1]);
        tracker.observe_sealed_block(100, hashes(&[1]));
        tracker.observe_sealed_block(101, hashes(&[2]));

        let reports = tracker.observe_sealed_block(102, hashes(&[3]));

        assert_eq!(verdicts(&reports), vec![(100, matched(1))]);
    }
}
