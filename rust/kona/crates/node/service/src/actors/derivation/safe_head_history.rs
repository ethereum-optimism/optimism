//! The [`SafeHeadHistory`] for the derivation actor.

use alloy_eips::BlockNumHash;
use kona_engine::DerivedSafeHead;
use kona_protocol::L2BlockInfo;
use std::collections::BTreeMap;

/// How many safe heads are kept, one per L1 block that produced one. A reorg that reaches below
/// the oldest kept one falls back to a full engine reset.
const MAX_ENTRIES: usize = 64;

/// The safe heads derivation produced from the most recent L1 blocks, each with an L1 block that
/// all of the L1 data it depends on is at or below. After an L1 reorg, the engine keeps the newest
/// one whose L1 block the reorg left canonical, instead of rewinding a full sequencing window;
/// see [`kona_engine::find_reorg_forkchoice`].
///
/// This is kept in memory whether or not the safe head database is enabled, and is bounded to
/// the depth an L1 reorg can plausibly reach. The database records each safe head at the L1 block
/// [`SafeHeadHistory::record`] returns, so the two never disagree.
///
/// A safe head depends on the L1 data its own attributes were derived from, and on everything its
/// parent depends on. The pipeline's origin bounds only the former: after a reset the pipeline
/// replays from before the safe head's L1 origin, and can produce the next block from an L1 block
/// below the one its parent was recorded at, as when its batch landed first and was buffered.
/// So the history keeps a [`Floor`] that is at least the L1 block the current safe head depends
/// on, and records no safe head below it.
#[derive(Debug, Default)]
pub(crate) struct SafeHeadHistory {
    /// L1 block number -> the highest safe head produced with the pipeline at that L1 block.
    by_l1_block: BTreeMap<u64, DerivedSafeHead>,
    /// The lowest L1 block a new record may claim.
    floor: Floor,
}

/// The lowest L1 block a new record may claim: at least the L1 block the current safe head
/// depends on.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
enum Floor {
    /// The L1 block the current safe head was recorded at. A safe head derived below it cannot
    /// have become safe before its parent, so it is recorded there.
    Recorded(BlockNumHash),
    /// Only a bound on the L1 blocks the current safe head depends on, with no block to record
    /// at. A safe head derived below it is not recorded.
    Bound(u64),
}

impl Default for Floor {
    fn default() -> Self {
        Self::Bound(0)
    }
}

impl SafeHeadHistory {
    /// Records `safe_head`, confirmed from attributes the pipeline produced at `derived_from`, at
    /// `derived_from` raised to the floor, and returns that L1 block. `None` when it is below a
    /// floor known only as a bound, and so is not recorded.
    ///
    /// The returned block bounds the L1 data `safe_head` depends on, and is the earliest L1 block
    /// it can have become safe at. Before Holocene that is where it became safe. After Holocene a
    /// node that reset can derive it from a batch its first pass dropped as a future batch, and
    /// the first pass took it from a later one.
    pub(crate) fn record(
        &mut self,
        safe_head: L2BlockInfo,
        derived_from: BlockNumHash,
    ) -> Option<BlockNumHash> {
        let safe_at = match self.floor {
            Floor::Recorded(floor) if derived_from.number < floor.number => floor,
            Floor::Bound(bound) if derived_from.number < bound => return None,
            _ => derived_from,
        };

        self.floor = Floor::Recorded(safe_at);
        self.by_l1_block
            .insert(safe_at.number, DerivedSafeHead { safe_head, derived_from: safe_at });
        while self.by_l1_block.len() > MAX_ENTRIES {
            self.by_l1_block.pop_first();
        }
        Some(safe_at)
    }

    /// The recorded safe heads, newest first.
    pub(crate) fn newest_first(&self) -> Vec<DerivedSafeHead> {
        self.by_l1_block.values().rev().copied().collect()
    }

    /// Derivation restarts from `safe_head`: forgets the safe heads above it.
    ///
    /// If `safe_head` is recorded, the floor is the L1 block it was recorded at. Otherwise
    /// nothing kept is known to lead to it, so the history is cleared and the floor is bounded by
    /// the L1 block by which derivation must have decided `safe_head`: one past the end of its
    /// epoch's sequencing window.
    pub(crate) fn reset_to(&mut self, safe_head: L2BlockInfo, seq_window_size: u64) {
        self.by_l1_block.retain(|_, derived| {
            derived.safe_head.block_info.number <= safe_head.block_info.number
        });

        match self.by_l1_block.last_key_value() {
            Some((_, derived)) if derived.safe_head == safe_head => {
                self.floor = Floor::Recorded(derived.derived_from);
            }
            _ => {
                self.by_l1_block.clear();
                self.floor = Floor::Bound(safe_head.l1_origin.number + seq_window_size + 1);
            }
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use alloy_primitives::B256;
    use kona_protocol::BlockInfo;

    const SEQ_WINDOW_SIZE: u64 = 10;

    fn l2_block(number: u64) -> L2BlockInfo {
        L2BlockInfo {
            block_info: BlockInfo {
                number,
                hash: B256::with_last_byte(number as u8),
                parent_hash: B256::with_last_byte(number.wrapping_sub(1) as u8),
                timestamp: number,
            },
            ..Default::default()
        }
    }

    fn l1_block(number: u64) -> BlockNumHash {
        BlockNumHash { number, hash: B256::repeat_byte(number as u8) }
    }

    fn record(history: &mut SafeHeadHistory, number: u64, derived_from: u64) -> Option<u64> {
        history.record(l2_block(number), l1_block(derived_from)).map(|safe_at| safe_at.number)
    }

    /// `(safe head number, L1 block number)` of each record, newest first.
    fn recorded(history: &SafeHeadHistory) -> Vec<(u64, u64)> {
        history
            .newest_first()
            .iter()
            .map(|d| (d.safe_head.block_info.number, d.derived_from.number))
            .collect()
    }

    #[test]
    fn keeps_the_highest_safe_head_per_l1_block_newest_first() {
        let mut history = SafeHeadHistory::default();
        record(&mut history, 11, 100);
        record(&mut history, 12, 100);
        record(&mut history, 13, 102);

        assert_eq!(
            history.newest_first(),
            vec![
                DerivedSafeHead { safe_head: l2_block(13), derived_from: l1_block(102) },
                DerivedSafeHead { safe_head: l2_block(12), derived_from: l1_block(100) },
            ]
        );
    }

    #[test]
    fn bounds_the_entries_kept() {
        let mut history = SafeHeadHistory::default();
        for i in 0..(MAX_ENTRIES as u64 + 10) {
            record(&mut history, 1 + i, 100 + i);
        }

        let recorded = history.newest_first();
        assert_eq!(recorded.len(), MAX_ENTRIES);
        assert_eq!(recorded.last().unwrap().derived_from.number, 110);
    }

    /// After a reset to a recorded safe head, the pipeline replays from before its L1 origin and
    /// can produce the next block from an L1 block below the one that safe head was recorded
    /// with. That block depends on its parent, so it became safe at its parent's L1 block, where
    /// it is recorded instead.
    #[test]
    fn replay_after_reset_records_at_the_kept_safe_heads_l1_block() {
        let mut history = SafeHeadHistory::default();
        record(&mut history, 11, 100);
        record(&mut history, 12, 106);
        record(&mut history, 13, 107);

        history.reset_to(l2_block(12), SEQ_WINDOW_SIZE);
        assert_eq!(record(&mut history, 13, 100), Some(106));
        assert_eq!(recorded(&history), vec![(13, 106), (11, 100)]);

        assert_eq!(record(&mut history, 14, 108), Some(108));
        assert_eq!(recorded(&history), vec![(14, 108), (13, 106), (11, 100)]);
    }

    /// A reset to a safe head the history did not record, such as one from a full reset, leaves
    /// that safe head's L1 dependency unknown beyond the end of its sequencing window.
    #[test]
    fn reset_to_an_unrecorded_safe_head_floors_at_its_sequencing_window() {
        let mut history = SafeHeadHistory::default();
        record(&mut history, 11, 100);
        record(&mut history, 12, 101);

        let mut safe_head = l2_block(11);
        safe_head.l1_origin.number = 95;
        safe_head.block_info.hash = B256::repeat_byte(0xaa);
        history.reset_to(safe_head, SEQ_WINDOW_SIZE);
        assert!(history.newest_first().is_empty());

        // Derived below the end of the sequencing window: not recorded.
        assert_eq!(record(&mut history, 12, 105), None);
        assert!(history.newest_first().is_empty());

        // From the floor on, recording resumes.
        assert_eq!(record(&mut history, 13, 106), Some(106));
        assert_eq!(recorded(&history), vec![(13, 106)]);
    }
}
