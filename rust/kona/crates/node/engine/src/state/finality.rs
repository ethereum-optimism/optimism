//! Checks that keep a label from moving below finality.

use crate::EngineSyncState;
use alloy_eips::BlockNumHash;
use thiserror::Error;

/// A block that a label cannot move to, because it would move the label below finality.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Error)]
pub enum FinalityViolation {
    /// The block is below the finalized head.
    #[error("Block {target} is behind the finalized head {finalized:?}")]
    BehindFinalizedHead {
        /// The finalized head.
        finalized: BlockNumHash,
        /// The number of the block.
        target: u64,
    },
    /// The block is a different block at the finalized head's height.
    #[error("Block {target:?} conflicts with the finalized head {finalized:?}")]
    ConflictsWithFinalizedHead {
        /// The finalized head.
        finalized: BlockNumHash,
        /// The block.
        target: BlockNumHash,
    },
}

impl EngineSyncState {
    /// Rejects a block number below the established finalized head.
    pub fn ensure_not_behind_finality(&self, target: u64) -> Result<(), FinalityViolation> {
        match self.established_finalized_head() {
            Some(finalized) if target < finalized.number => {
                Err(FinalityViolation::BehindFinalizedHead { finalized, target })
            }
            _ => Ok(()),
        }
    }

    /// Rejects a different block at the established finalized head's height. Before finality is
    /// established no block pins it, so any block, including the configured genesis, passes.
    pub fn ensure_no_conflict_with_finality(
        &self,
        target: BlockNumHash,
    ) -> Result<(), FinalityViolation> {
        match self.established_finalized_head() {
            Some(finalized)
                if target.number == finalized.number && target.hash != finalized.hash =>
            {
                Err(FinalityViolation::ConflictsWithFinalizedHead { finalized, target })
            }
            _ => Ok(()),
        }
    }

    /// Rejects a block below the established finalized head or conflicting with it.
    pub fn ensure_respects_finality(&self, target: BlockNumHash) -> Result<(), FinalityViolation> {
        self.ensure_not_behind_finality(target.number)?;
        self.ensure_no_conflict_with_finality(target)
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::EngineSyncStateUpdate;
    use alloy_primitives::B256;
    use kona_protocol::{BlockInfo, L2BlockInfo};

    const FINALIZED_NUMBER: u64 = 5;

    fn block(number: u64, hash: u8) -> L2BlockInfo {
        L2BlockInfo {
            block_info: BlockInfo { hash: B256::repeat_byte(hash), number, ..Default::default() },
            ..Default::default()
        }
    }

    fn sync_state_finalized_at(finalized_head: L2BlockInfo) -> EngineSyncState {
        EngineSyncState::default().apply_update(EngineSyncStateUpdate {
            finalized_head: Some(finalized_head),
            ..Default::default()
        })
    }

    /// The finalized head at 5 once finality is established, and the default head before.
    fn finalized_head(is_finality_established: bool) -> L2BlockInfo {
        if is_finality_established { block(FINALIZED_NUMBER, 1) } else { L2BlockInfo::default() }
    }

    /// The violation expected for `target` checked against the finalized head at 5.
    fn behind(target: L2BlockInfo) -> Option<FinalityViolation> {
        Some(FinalityViolation::BehindFinalizedHead {
            finalized: block(FINALIZED_NUMBER, 1).block_info.id(),
            target: target.block_info.number,
        })
    }

    /// The violation expected for `target` conflicting with the finalized head at 5.
    fn conflicting(target: L2BlockInfo) -> Option<FinalityViolation> {
        Some(FinalityViolation::ConflictsWithFinalizedHead {
            finalized: block(FINALIZED_NUMBER, 1).block_info.id(),
            target: target.block_info.id(),
        })
    }

    /// A block below the established finalized head, or a different block at its height, breaks
    /// finality. Before finality is established no block pins it, so a different block at the
    /// default head's height, genesis, passes.
    #[rstest::rstest]
    #[case::above(true, block(FINALIZED_NUMBER + 1, 2), None)]
    #[case::at(true, block(FINALIZED_NUMBER, 1), None)]
    #[case::below(true, block(FINALIZED_NUMBER - 1, 2), behind(block(FINALIZED_NUMBER - 1, 2)))]
    #[case::conflicting(true, block(FINALIZED_NUMBER, 2), conflicting(block(FINALIZED_NUMBER, 2)))]
    #[case::not_established(false, block(0, 2), None)]
    fn block_is_checked_against_established_finality(
        #[case] is_finality_established: bool,
        #[case] target: L2BlockInfo,
        #[case] expected_violation: Option<FinalityViolation>,
    ) {
        let sync_state = sync_state_finalized_at(finalized_head(is_finality_established));

        let result = sync_state.ensure_respects_finality(target.block_info.id());

        assert_eq!(result.err(), expected_violation);
    }
}
