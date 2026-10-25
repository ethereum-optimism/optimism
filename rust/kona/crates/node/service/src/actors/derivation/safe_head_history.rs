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
/// the depth an L1 reorg can plausibly reach.
///
/// A safe head depends on the L1 data its own attributes were derived from, and on everything its
/// parent depends on. The pipeline's origin bounds only the former: after a reset the pipeline
/// replays from before the safe head's L1 origin, and can produce the next block from an L1 block
/// below the one its parent needed. So the history keeps a floor that is at least the L1 block
/// the current safe head depends on, and records a safe head only if its attributes came from at
/// or above the floor.
#[derive(Debug, Default)]
pub(crate) struct SafeHeadHistory {
    /// L1 block number -> the highest safe head produced with the pipeline at that L1 block.
    by_l1_block: BTreeMap<u64, DerivedSafeHead>,
    /// The lowest L1 block a new record may claim: at least the L1 block the current safe head
    /// depends on.
    floor: u64,
}

impl SafeHeadHistory {
    /// Records `safe_head`, confirmed from attributes the pipeline produced at `derived_from`, if
    /// that is at or above the floor.
    pub(crate) fn record(&mut self, safe_head: L2BlockInfo, derived_from: BlockNumHash) {
        if derived_from.number < self.floor {
            // The safe head depends on no more than the floor, which stays put.
            return;
        }

        self.floor = derived_from.number;
        self.by_l1_block.insert(derived_from.number, DerivedSafeHead { safe_head, derived_from });
        while self.by_l1_block.len() > MAX_ENTRIES {
            self.by_l1_block.pop_first();
        }
    }

    /// The recorded safe heads, newest first.
    pub(crate) fn newest_first(&self) -> Vec<DerivedSafeHead> {
        self.by_l1_block.values().rev().copied().collect()
    }

    /// Derivation restarts from `safe_head`: forgets the safe heads above it.
    ///
    /// If `safe_head` is recorded, the floor is the L1 block it was recorded with. Otherwise
    /// nothing kept is known to lead to it, so the history is cleared and the floor is the L1
    /// block by which derivation must have decided `safe_head`: one past the end of its epoch's
    /// sequencing window.
    pub(crate) fn reset_to(&mut self, safe_head: L2BlockInfo, seq_window_size: u64) {
        self.by_l1_block.retain(|_, derived| {
            derived.safe_head.block_info.number <= safe_head.block_info.number
        });

        match self.by_l1_block.last_key_value() {
            Some((&number, derived)) if derived.safe_head == safe_head => self.floor = number,
            _ => {
                self.by_l1_block.clear();
                self.floor = safe_head.l1_origin.number + seq_window_size + 1;
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

    fn record(history: &mut SafeHeadHistory, number: u64, derived_from: u64) {
        history.record(l2_block(number), l1_block(derived_from));
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
    /// with. That block depends on its parent, so it must not be recorded as derived from the
    /// lower L1 block, and must not overwrite the record already at that L1 block.
    #[test]
    fn replay_after_reset_does_not_record_below_the_kept_safe_head() {
        let mut history = SafeHeadHistory::default();
        record(&mut history, 11, 100);
        record(&mut history, 12, 106);
        record(&mut history, 13, 107);

        history.reset_to(l2_block(12), SEQ_WINDOW_SIZE);
        record(&mut history, 13, 100);
        assert_eq!(recorded(&history), vec![(12, 106), (11, 100)]);

        record(&mut history, 14, 108);
        assert_eq!(recorded(&history), vec![(14, 108), (12, 106), (11, 100)]);
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
        record(&mut history, 12, 105);
        assert!(history.newest_first().is_empty());

        // From the floor on, recording resumes.
        record(&mut history, 13, 106);
        assert_eq!(recorded(&history), vec![(13, 106)]);
    }
}
