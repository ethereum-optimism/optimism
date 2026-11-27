//! The safe heads a derivation delegate reports.

use alloy_eips::BlockNumHash;
use kona_protocol::L2BlockInfo;
use thiserror::Error;

/// The safe heads a derivation delegate reports, imported as separate labels.
///
/// Before interop both are the same block. Under interop the cross-safe head can lag the
/// local-safe head while the delegate verifies cross-chain dependencies, and only the cross-safe
/// head may reach the execution layer as its safe block.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct DelegatedSafeHeads {
    local_safe_head: L2BlockInfo,
    safe_head: L2BlockInfo,
}

/// Safe heads that no consistent delegate reports.
#[derive(Debug, Clone, Copy, Error, PartialEq, Eq)]
pub enum InconsistentSafeHeadsError {
    /// The cross-safe head is ahead of the local-safe head.
    #[error("Safe head {safe} is ahead of local safe head {local_safe}")]
    SafeAheadOfLocalSafe {
        /// The number of the delegate's local-safe head.
        local_safe: u64,
        /// The number of the delegate's cross-safe head.
        safe: u64,
    },
    /// The cross-safe head is a different block at the local-safe head's height.
    #[error("Safe head {safe:?} conflicts with local safe head {local_safe:?}")]
    SafeConflictsWithLocalSafe {
        /// The delegate's local-safe head.
        local_safe: BlockNumHash,
        /// The delegate's cross-safe head.
        safe: BlockNumHash,
    },
}

impl DelegatedSafeHeads {
    /// Pairs the delegate's local-safe head, derived from L1 with cross-chain dependencies not
    /// verified yet, with its cross-safe head, which must be at or below it on the same chain: not
    /// ahead of it, and not a different block at its height.
    pub fn try_new(
        local_safe_head: L2BlockInfo,
        safe_head: L2BlockInfo,
    ) -> Result<Self, InconsistentSafeHeadsError> {
        let local_safe = local_safe_head.block_info.id();
        let safe = safe_head.block_info.id();
        if safe.number > local_safe.number {
            return Err(InconsistentSafeHeadsError::SafeAheadOfLocalSafe {
                local_safe: local_safe.number,
                safe: safe.number,
            });
        }
        if safe.number == local_safe.number && safe.hash != local_safe.hash {
            return Err(InconsistentSafeHeadsError::SafeConflictsWithLocalSafe {
                local_safe,
                safe,
            });
        }
        Ok(Self { local_safe_head, safe_head })
    }

    /// The delegate's local-safe head.
    pub const fn local_safe_head(&self) -> L2BlockInfo {
        self.local_safe_head
    }

    /// The delegate's cross-safe head.
    pub const fn safe_head(&self) -> L2BlockInfo {
        self.safe_head
    }

    /// The cross-safe head when it is above `current_safe_head`, and `None` otherwise: a
    /// cross-safe head at or below the safe label leaves it where it is.
    pub(crate) const fn advancing_safe_head(
        &self,
        current_safe_head: &L2BlockInfo,
    ) -> Option<L2BlockInfo> {
        if self.safe_head.block_info.number > current_safe_head.block_info.number {
            Some(self.safe_head)
        } else {
            None
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::test_utils::test_block_info;
    use alloy_primitives::B256;
    use kona_protocol::BlockInfo;
    use rstest::rstest;

    const LOCAL_SAFE_HEAD: L2BlockInfo = L2BlockInfo {
        block_info: BlockInfo {
            hash: B256::repeat_byte(0x10),
            number: 10,
            parent_hash: B256::ZERO,
            timestamp: 0,
        },
        l1_origin: BlockNumHash { hash: B256::ZERO, number: 0 },
        seq_num: 0,
    };

    /// A different block at the height of `block`.
    fn conflicting_block(block: L2BlockInfo) -> L2BlockInfo {
        L2BlockInfo {
            block_info: BlockInfo { hash: !block.block_info.hash, ..block.block_info },
            ..block
        }
    }

    #[rstest]
    #[case::cross_safe_lags(test_block_info(8), true)]
    #[case::same_block(LOCAL_SAFE_HEAD, true)]
    #[case::conflicting_at_same_height(conflicting_block(LOCAL_SAFE_HEAD), false)]
    #[case::cross_safe_ahead(test_block_info(11), false)]
    fn try_new_rejects_cross_safe_off_the_local_safe_chain(
        #[case] safe_head: L2BlockInfo,
        #[case] accepted: bool,
    ) {
        let heads = DelegatedSafeHeads::try_new(LOCAL_SAFE_HEAD, safe_head);

        assert_eq!(heads.is_ok(), accepted, "got {heads:?}");
    }

    /// Only a cross-safe head strictly above the safe label advances it.
    #[rstest]
    #[case::above(8, true)]
    #[case::same_height(7, false)]
    #[case::below(6, false)]
    fn advancing_safe_head_requires_a_higher_block(#[case] safe: u64, #[case] advances: bool) {
        let heads =
            DelegatedSafeHeads::try_new(test_block_info(10), test_block_info(safe)).unwrap();

        assert_eq!(heads.advancing_safe_head(&test_block_info(7)).is_some(), advances);
    }
}
