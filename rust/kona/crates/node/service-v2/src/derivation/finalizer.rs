//! Mapping from finalized L1 inputs to already-safe L2 blocks.

use kona_protocol::{BlockInfo, L2BlockInfo};
use std::collections::BTreeMap;

/// Tracks the highest accepted safe L2 block and its exact L1 derivation source at each height.
#[derive(Debug, Default)]
pub(super) struct L2Finalizer {
    pending: BTreeMap<u64, (BlockInfo, L2BlockInfo)>,
}

impl L2Finalizer {
    pub(super) fn record(&mut self, derived_from: BlockInfo, safe: L2BlockInfo) {
        self.pending
            .entry(derived_from.number)
            .and_modify(|known| {
                if known.1.block_info.number < safe.block_info.number {
                    *known = (derived_from, safe);
                }
            })
            .or_insert((derived_from, safe));
    }

    pub(super) fn finalized_by(
        &mut self,
        finalized_l1: BlockInfo,
    ) -> Option<(BlockInfo, L2BlockInfo)> {
        let safe = self
            .pending
            .range(..=finalized_l1.number)
            .next_back()
            .map(|(_, source_and_safe)| *source_and_safe);
        if safe.is_some() {
            self.pending.retain(|number, _| *number > finalized_l1.number);
        }
        safe
    }

    pub(super) fn clear(&mut self) {
        self.pending.clear();
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    fn l1(number: u64) -> BlockInfo {
        BlockInfo { number, ..Default::default() }
    }

    fn l2(number: u64) -> L2BlockInfo {
        L2BlockInfo { block_info: BlockInfo { number, ..Default::default() }, ..Default::default() }
    }

    #[test]
    fn finalizes_the_highest_safe_block_covered_by_l1_finality() {
        let mut finalizer = L2Finalizer::default();
        finalizer.record(l1(2), l2(4));
        finalizer.record(l1(3), l2(6));
        finalizer.record(l1(4), l2(7));

        assert_eq!(finalizer.finalized_by(l1(3)), Some((l1(3), l2(6))));
        assert_eq!(finalizer.finalized_by(l1(3)), None);
        assert_eq!(finalizer.finalized_by(l1(4)), Some((l1(4), l2(7))));
    }

    #[test]
    fn reset_discards_pending_finality_without_touching_engine_state() {
        let mut finalizer = L2Finalizer::default();
        finalizer.record(l1(2), l2(4));
        finalizer.clear();
        assert_eq!(finalizer.finalized_by(l1(2)), None);
    }
}
