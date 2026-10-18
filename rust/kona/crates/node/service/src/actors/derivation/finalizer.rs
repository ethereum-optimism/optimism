//! Finality records are installed only after the engine accepts the actual safe block.

use kona_protocol::{BlockInfo, L2BlockInfo};
use std::collections::BTreeMap;

/// The accepted block and the L1 block containing its derivation inputs.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub(crate) struct FinalizationRecord {
    pub(crate) block: L2BlockInfo,
    pub(crate) source: BlockInfo,
}

/// Retains candidates until acknowledged, including across transient engine failures.
#[derive(Debug, Default)]
pub(crate) struct L2Finalizer {
    awaiting_finalization: BTreeMap<u64, FinalizationRecord>,
}

impl L2Finalizer {
    pub(crate) fn record(&mut self, block: L2BlockInfo, source: BlockInfo) {
        let record = FinalizationRecord { block, source };
        self.awaiting_finalization
            .entry(source.number)
            .and_modify(|current| {
                if block.block_info.number >= current.block.block_info.number {
                    *current = record;
                }
            })
            .or_insert(record);
    }

    pub(crate) fn clear(&mut self) {
        self.awaiting_finalization.clear();
    }

    pub(crate) fn candidate(&self, finalized_l1: u64) -> Option<FinalizationRecord> {
        self.awaiting_finalization
            .range(..=finalized_l1)
            .map(|(_, record)| *record)
            .max_by_key(|record| record.block.block_info.number)
    }

    pub(crate) fn acknowledge(&mut self, record: FinalizationRecord) {
        self.awaiting_finalization
            .retain(|_, current| current.block.block_info.number > record.block.block_info.number);
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use alloy_primitives::B256;

    #[test]
    fn actual_hash_and_source_survive_retries_until_acknowledged() {
        let mut finalizer = L2Finalizer::default();
        let block = L2BlockInfo {
            block_info: BlockInfo { number: 9, hash: B256::repeat_byte(9), ..Default::default() },
            ..Default::default()
        };
        let source = BlockInfo { number: 4, hash: B256::repeat_byte(4), ..Default::default() };
        finalizer.record(block, source);
        assert_eq!(finalizer.candidate(3), None);
        let record = finalizer.candidate(4).unwrap();
        assert_eq!(record, FinalizationRecord { block, source });
        assert_eq!(finalizer.candidate(5), Some(record));
        finalizer.acknowledge(record);
        assert_eq!(finalizer.candidate(5), None);
    }

    #[test]
    fn reset_discards_records_even_when_the_safe_head_is_unchanged() {
        let mut finalizer = L2Finalizer::default();
        finalizer.record(L2BlockInfo::default(), BlockInfo::default());
        finalizer.clear();
        assert_eq!(finalizer.candidate(u64::MAX), None);
    }
}
