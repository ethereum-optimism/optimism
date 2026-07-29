//! Flashblock sequence validation and reorganization detection.
//!
//! Provides stateless validation logic for flashblock sequencing and chain reorg detection.
//!
//! This module contains three main components:
//!
//! 1. [`FlashblockSequenceValidator`] - Validates that incoming flashblocks follow the expected
//!    sequence ordering (consecutive indices within a block, proper block transitions).
//!
//! 2. [`ReorgDetector`] - Detects chain reorganizations by comparing full block fingerprints (block
//!    hash, parent hash, and transaction hashes) between tracked (pending) state and canonical
//!    chain state.
//!
//! 3. [`CanonicalBlockReconciler`] - Determines the appropriate strategy for reconciling pending
//!    flashblock state when new canonical blocks arrive.

use alloy_primitives::B256;

/// Result of validating a flashblock's position in the sequence.
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum SequenceValidationResult {
    /// Next consecutive flashblock within the current block (same block, index + 1).
    NextInSequence,
    /// First flashblock (index 0) of the next block (block + 1).
    FirstOfNextBlock,
    /// Duplicate flashblock (same block and index) - should be ignored.
    Duplicate,
    /// Non-sequential index within the same block - indicates missed flashblocks.
    NonSequentialGap {
        /// Expected flashblock index.
        expected: u64,
        /// Actual incoming flashblock index.
        actual: u64,
    },
    /// New block received with non-zero index - missed the base flashblock.
    InvalidNewBlockIndex {
        /// Block number of the incoming flashblock.
        block_number: u64,
        /// The invalid (non-zero) index received.
        index: u64,
    },
}

/// Stateless validator for flashblock sequence ordering.
///
/// Flashblocks must arrive in strict sequential order:
/// - Within a block: indices must be consecutive (0, 1, 2, ...)
/// - Across blocks: new block must start with index 0 and be exactly `block_number + 1`
///
/// # Example
///
/// ```
/// use reth_optimism_flashblocks::validation::{
///     FlashblockSequenceValidator, SequenceValidationResult,
/// };
///
/// // Valid: next flashblock in sequence
/// let result = FlashblockSequenceValidator::validate(100, 2, 100, 3);
/// assert_eq!(result, SequenceValidationResult::NextInSequence);
///
/// // Valid: first flashblock of next block
/// let result = FlashblockSequenceValidator::validate(100, 5, 101, 0);
/// assert_eq!(result, SequenceValidationResult::FirstOfNextBlock);
///
/// // Invalid: gap in sequence
/// let result = FlashblockSequenceValidator::validate(100, 2, 100, 5);
/// assert!(matches!(result, SequenceValidationResult::NonSequentialGap { .. }));
/// ```
#[derive(Debug, Clone, Copy, Default)]
pub struct FlashblockSequenceValidator;

impl FlashblockSequenceValidator {
    /// Validates whether an incoming flashblock follows the expected sequence.
    ///
    /// Returns the appropriate [`SequenceValidationResult`] based on:
    /// - Same block, index + 1 → `NextInSequence`
    /// - Next block, index 0 → `FirstOfNextBlock`
    /// - Same block and index → `Duplicate`
    /// - Same block, wrong index → `NonSequentialGap`
    /// - Different block, non-zero index or block gap → `InvalidNewBlockIndex`
    pub const fn validate(
        latest_block_number: u64,
        latest_flashblock_index: u64,
        incoming_block_number: u64,
        incoming_index: u64,
    ) -> SequenceValidationResult {
        // Next flashblock within the current block
        if incoming_block_number == latest_block_number &&
            incoming_index == latest_flashblock_index + 1
        {
            SequenceValidationResult::NextInSequence
        // First flashblock of the next block
        } else if incoming_block_number == latest_block_number + 1 && incoming_index == 0 {
            SequenceValidationResult::FirstOfNextBlock
        // New block with non-zero index or block gap
        } else if incoming_block_number != latest_block_number {
            SequenceValidationResult::InvalidNewBlockIndex {
                block_number: incoming_block_number,
                index: incoming_index,
            }
        } else if incoming_index == latest_flashblock_index {
            // Duplicate flashblock
            SequenceValidationResult::Duplicate
        } else {
            // Non-sequential index within the same block
            SequenceValidationResult::NonSequentialGap {
                expected: latest_flashblock_index + 1,
                actual: incoming_index,
            }
        }
    }
}

/// Fingerprint for a tracked block (pending/cached sequence).
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct TrackedBlockFingerprint {
    /// Block number.
    pub block_number: u64,
    /// Block hash.
    pub block_hash: B256,
    /// Parent hash.
    pub parent_hash: B256,
    /// Ordered transaction hashes in the block.
    pub tx_hashes: Vec<B256>,
}

/// Fingerprint for a canonical block notification.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct CanonicalBlockFingerprint {
    /// Block number.
    pub block_number: u64,
    /// Block hash.
    pub block_hash: B256,
    /// Parent hash.
    pub parent_hash: B256,
    /// Ordered transaction hashes in the block.
    pub tx_hashes: Vec<B256>,
}

/// Result of a reorganization detection check.
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum ReorgDetectionResult {
    /// Tracked and canonical fingerprints match exactly.
    NoReorg,
    /// Tracked and canonical fingerprints differ.
    ReorgDetected,
}

impl ReorgDetectionResult {
    /// Returns `true` if a reorganization was detected.
    #[inline]
    pub const fn is_reorg(&self) -> bool {
        matches!(self, Self::ReorgDetected)
    }

    /// Returns `true` if no reorganization was detected.
    #[inline]
    pub const fn is_no_reorg(&self) -> bool {
        matches!(self, Self::NoReorg)
    }
}

/// Detects chain reorganizations by comparing full block fingerprints.
///
/// A reorg is detected when any fingerprint component differs:
/// - Block hash
/// - Parent hash
/// - Transaction hash list (including ordering)
///
/// # Example
///
/// ```
/// use alloy_primitives::B256;
/// use reth_optimism_flashblocks::validation::{
///     CanonicalBlockFingerprint, ReorgDetectionResult, ReorgDetector, TrackedBlockFingerprint,
/// };
///
/// let tracked = TrackedBlockFingerprint {
///     block_number: 100,
///     block_hash: B256::repeat_byte(0xAA),
///     parent_hash: B256::repeat_byte(0x11),
///     tx_hashes: vec![B256::repeat_byte(1), B256::repeat_byte(2)],
/// };
/// let canonical = CanonicalBlockFingerprint {
///     block_number: 100,
///     block_hash: B256::repeat_byte(0xAA),
///     parent_hash: B256::repeat_byte(0x11),
///     tx_hashes: vec![B256::repeat_byte(1), B256::repeat_byte(2)],
/// };
///
/// let result = ReorgDetector::detect(&tracked, &canonical);
/// assert_eq!(result, ReorgDetectionResult::NoReorg);
/// ```
#[derive(Debug, Clone, Copy, Default)]
pub struct ReorgDetector;

impl ReorgDetector {
    /// Compares tracked vs canonical block fingerprints to detect reorgs.
    pub fn detect(
        tracked: &TrackedBlockFingerprint,
        canonical: &CanonicalBlockFingerprint,
    ) -> ReorgDetectionResult {
        if tracked.block_hash == canonical.block_hash &&
            tracked.parent_hash == canonical.parent_hash &&
            tracked.tx_hashes == canonical.tx_hashes
        {
            ReorgDetectionResult::NoReorg
        } else {
            ReorgDetectionResult::ReorgDetected
        }
    }
}

/// Strategy for reconciling pending state with canonical state on new canonical blocks.
///
/// When a new canonical block arrives, the system must decide how to update
/// the pending flashblock state. This enum represents the possible strategies.
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum ReconciliationStrategy {
    /// Canonical caught up or passed pending (canonical >= latest pending). Clear pending state.
    CatchUp,
    /// Reorg detected (tx mismatch). Rebuild pending from canonical.
    HandleReorg,
    /// Pending too far ahead of canonical.
    DepthLimitExceeded {
        /// Current depth of pending blocks.
        depth: u64,
        /// Configured maximum depth.
        max_depth: u64,
    },
    /// No issues - continue building on pending state.
    Continue,
    /// No pending state exists (startup or after clear).
    NoPendingState,
}

/// Determines reconciliation strategy for canonical block updates.
///
/// This reconciler helps maintain consistency between pending flashblock state
/// and the canonical chain. It's used when new canonical blocks arrive to
/// determine whether to:
/// - Clear pending state (canonical caught up)
/// - Rebuild pending state (reorg detected)
/// - Continue as-is (pending still ahead and valid)
///
/// # Priority Order
///
/// The reconciler checks conditions in this order:
/// 1. `NoPendingState` - No pending state to reconcile
/// 2. `CatchUp` - Canonical has caught up to or passed pending
/// 3. `HandleReorg` - Reorg detected (takes precedence over depth limit)
/// 4. `DepthLimitExceeded` - Pending is too far ahead
/// 5. `Continue` - Everything is fine, keep building
///
/// # Example
///
/// ```
/// use reth_optimism_flashblocks::validation::{CanonicalBlockReconciler, ReconciliationStrategy};
///
/// // Canonical caught up to pending
/// let strategy = CanonicalBlockReconciler::reconcile(
///     Some(100), // earliest pending
///     Some(105), // latest pending
///     105,       // canonical block number
///     10,        // max depth
///     false,     // no reorg detected
/// );
/// assert_eq!(strategy, ReconciliationStrategy::CatchUp);
/// ```
#[derive(Debug, Clone, Copy, Default)]
pub struct CanonicalBlockReconciler;

impl CanonicalBlockReconciler {
    /// Returns the appropriate [`ReconciliationStrategy`] based on pending vs canonical state.
    ///
    /// Priority: `NoPendingState` → `CatchUp` → `HandleReorg` → `DepthLimitExceeded` → `Continue`
    pub const fn reconcile(
        pending_earliest_block: Option<u64>,
        pending_latest_block: Option<u64>,
        canonical_block_number: u64,
        max_depth: u64,
        reorg_detected: bool,
    ) -> ReconciliationStrategy {
        // Check if pending state exists
        let latest = match (pending_earliest_block, pending_latest_block) {
            (Some(_e), Some(l)) => l,
            _ => return ReconciliationStrategy::NoPendingState,
        };

        // Check if canonical has caught up or passed pending
        if latest <= canonical_block_number {
            return ReconciliationStrategy::CatchUp;
        }

        // Check for reorg
        if reorg_detected {
            return ReconciliationStrategy::HandleReorg;
        }

        // Check depth limit: how many pending blocks are ahead of canonical tip.
        let depth = latest.saturating_sub(canonical_block_number);
        if depth > max_depth {
            return ReconciliationStrategy::DepthLimitExceeded { depth, max_depth };
        }

        // No issues, continue building
        ReconciliationStrategy::Continue
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    // ==================== FlashblockSequenceValidator Tests ====================

    mod sequence_validator {
        use super::*;

        #[test]
        fn test_next_in_sequence() {
            // Consecutive indices within the same block
            assert_eq!(
                FlashblockSequenceValidator::validate(100, 2, 100, 3),
                SequenceValidationResult::NextInSequence
            );
            assert_eq!(
                FlashblockSequenceValidator::validate(100, 0, 100, 1),
                SequenceValidationResult::NextInSequence
            );
            assert_eq!(
                FlashblockSequenceValidator::validate(100, 999, 100, 1000),
                SequenceValidationResult::NextInSequence
            );
            assert_eq!(
                FlashblockSequenceValidator::validate(0, 0, 0, 1),
                SequenceValidationResult::NextInSequence
            );
        }

        #[test]
        fn test_first_of_next_block() {
            // Index 0 of the next block
            assert_eq!(
                FlashblockSequenceValidator::validate(0, 0, 1, 0),
                SequenceValidationResult::FirstOfNextBlock
            );
            assert_eq!(
                FlashblockSequenceValidator::validate(100, 5, 101, 0),
                SequenceValidationResult::FirstOfNextBlock
            );
            assert_eq!(
                FlashblockSequenceValidator::validate(100, 0, 101, 0),
                SequenceValidationResult::FirstOfNextBlock
            );
            assert_eq!(
                FlashblockSequenceValidator::validate(999999, 10, 1000000, 0),
                SequenceValidationResult::FirstOfNextBlock
            );
        }

        #[test]
        fn test_duplicate() {
            // Same block and index
            assert_eq!(
                FlashblockSequenceValidator::validate(100, 5, 100, 5),
                SequenceValidationResult::Duplicate
            );
            assert_eq!(
                FlashblockSequenceValidator::validate(100, 0, 100, 0),
                SequenceValidationResult::Duplicate
            );
        }

        #[test]
        fn test_non_sequential_gap() {
            // Non-consecutive indices within the same block
            assert_eq!(
                FlashblockSequenceValidator::validate(100, 2, 100, 4),
                SequenceValidationResult::NonSequentialGap { expected: 3, actual: 4 }
            );
            assert_eq!(
                FlashblockSequenceValidator::validate(100, 0, 100, 10),
                SequenceValidationResult::NonSequentialGap { expected: 1, actual: 10 }
            );
            // Going backwards
            assert_eq!(
                FlashblockSequenceValidator::validate(100, 5, 100, 3),
                SequenceValidationResult::NonSequentialGap { expected: 6, actual: 3 }
            );
        }

        #[test]
        fn test_invalid_new_block_index() {
            // New block with non-zero index
            assert_eq!(
                FlashblockSequenceValidator::validate(100, 5, 101, 1),
                SequenceValidationResult::InvalidNewBlockIndex { block_number: 101, index: 1 }
            );
            // Block gap
            assert_eq!(
                FlashblockSequenceValidator::validate(100, 5, 105, 3),
                SequenceValidationResult::InvalidNewBlockIndex { block_number: 105, index: 3 }
            );
            assert_eq!(
                FlashblockSequenceValidator::validate(100, 5, 102, 0),
                SequenceValidationResult::InvalidNewBlockIndex { block_number: 102, index: 0 }
            );
            // Going backwards in block number
            assert_eq!(
                FlashblockSequenceValidator::validate(100, 5, 99, 0),
                SequenceValidationResult::InvalidNewBlockIndex { block_number: 99, index: 0 }
            );
        }
    }

    // ==================== ReorgDetector Tests ====================

    mod reorg_detector {
        use super::*;

        fn tracked(
            block_hash: B256,
            parent_hash: B256,
            tx_hashes: Vec<B256>,
        ) -> TrackedBlockFingerprint {
            TrackedBlockFingerprint { block_number: 100, block_hash, parent_hash, tx_hashes }
        }

        fn canonical(
            block_hash: B256,
            parent_hash: B256,
            tx_hashes: Vec<B256>,
        ) -> CanonicalBlockFingerprint {
            CanonicalBlockFingerprint { block_number: 100, block_hash, parent_hash, tx_hashes }
        }

        #[test]
        fn test_no_reorg_identical_fingerprint() {
            let hashes = vec![B256::repeat_byte(0x01), B256::repeat_byte(0x02)];
            let tracked = tracked(B256::repeat_byte(0xAA), B256::repeat_byte(0x11), hashes.clone());
            let canonical = canonical(B256::repeat_byte(0xAA), B256::repeat_byte(0x11), hashes);
            assert_eq!(ReorgDetector::detect(&tracked, &canonical), ReorgDetectionResult::NoReorg);
        }

        #[test]
        fn test_reorg_on_parent_hash_mismatch_with_identical_txs() {
            let hashes = vec![B256::repeat_byte(0x01), B256::repeat_byte(0x02)];
            let tracked = tracked(B256::repeat_byte(0xAA), B256::repeat_byte(0x11), hashes.clone());
            let canonical = canonical(B256::repeat_byte(0xAA), B256::repeat_byte(0x22), hashes);

            assert_eq!(
                ReorgDetector::detect(&tracked, &canonical),
                ReorgDetectionResult::ReorgDetected
            );
        }

        #[test]
        fn test_reorg_on_block_hash_mismatch_with_identical_txs() {
            let hashes = vec![B256::repeat_byte(0x01), B256::repeat_byte(0x02)];
            let tracked = tracked(B256::repeat_byte(0xAA), B256::repeat_byte(0x11), hashes.clone());
            let canonical = canonical(B256::repeat_byte(0xBB), B256::repeat_byte(0x11), hashes);

            assert_eq!(
                ReorgDetector::detect(&tracked, &canonical),
                ReorgDetectionResult::ReorgDetected
            );
        }

        #[test]
        fn test_reorg_on_tx_hash_mismatch() {
            let tracked = tracked(
                B256::repeat_byte(0xAA),
                B256::repeat_byte(0x11),
                vec![B256::repeat_byte(0x01), B256::repeat_byte(0x02)],
            );
            let canonical = canonical(
                B256::repeat_byte(0xAA),
                B256::repeat_byte(0x11),
                vec![B256::repeat_byte(0x01), B256::repeat_byte(0x03)],
            );

            assert_eq!(
                ReorgDetector::detect(&tracked, &canonical),
                ReorgDetectionResult::ReorgDetected
            );
        }

        #[test]
        fn test_result_helpers() {
            let no_reorg = ReorgDetectionResult::NoReorg;
            assert!(no_reorg.is_no_reorg());
            assert!(!no_reorg.is_reorg());

            let reorg = ReorgDetectionResult::ReorgDetected;
            assert!(reorg.is_reorg());
            assert!(!reorg.is_no_reorg());
        }
    }

    // ==================== CanonicalBlockReconciler Tests ====================

    mod reconciler {
        use super::*;

        #[test]
        fn test_no_pending_state() {
            assert_eq!(
                CanonicalBlockReconciler::reconcile(None, None, 100, 10, false),
                ReconciliationStrategy::NoPendingState
            );
            assert_eq!(
                CanonicalBlockReconciler::reconcile(Some(100), None, 100, 10, false),
                ReconciliationStrategy::NoPendingState
            );
            assert_eq!(
                CanonicalBlockReconciler::reconcile(None, Some(100), 100, 10, false),
                ReconciliationStrategy::NoPendingState
            );
        }

        #[test]
        fn test_catchup() {
            // Canonical equals latest pending
            assert_eq!(
                CanonicalBlockReconciler::reconcile(Some(100), Some(105), 105, 10, false),
                ReconciliationStrategy::CatchUp
            );
            // Canonical passed latest pending
            assert_eq!(
                CanonicalBlockReconciler::reconcile(Some(100), Some(105), 110, 10, false),
                ReconciliationStrategy::CatchUp
            );
            // Single pending block
            assert_eq!(
                CanonicalBlockReconciler::reconcile(Some(100), Some(100), 100, 10, false),
                ReconciliationStrategy::CatchUp
            );
            // CatchUp takes priority over reorg
            assert_eq!(
                CanonicalBlockReconciler::reconcile(Some(100), Some(105), 105, 10, true),
                ReconciliationStrategy::CatchUp
            );
        }

        #[test]
        fn test_handle_reorg() {
            assert_eq!(
                CanonicalBlockReconciler::reconcile(Some(100), Some(110), 102, 10, true),
                ReconciliationStrategy::HandleReorg
            );
            // Reorg takes priority over depth limit
            assert_eq!(
                CanonicalBlockReconciler::reconcile(Some(100), Some(130), 120, 10, true),
                ReconciliationStrategy::HandleReorg
            );
        }

        #[test]
        fn test_depth_limit_exceeded() {
            assert_eq!(
                CanonicalBlockReconciler::reconcile(Some(100), Some(120), 115, 10, false),
                ReconciliationStrategy::Continue
            );
            assert_eq!(
                CanonicalBlockReconciler::reconcile(Some(100), Some(105), 101, 0, false),
                ReconciliationStrategy::DepthLimitExceeded { depth: 4, max_depth: 0 }
            );
            assert_eq!(
                CanonicalBlockReconciler::reconcile(Some(100), Some(200), 130, 64, false),
                ReconciliationStrategy::DepthLimitExceeded { depth: 70, max_depth: 64 }
            );
        }

        #[test]
        fn test_continue() {
            // Normal case: pending ahead of canonical
            assert_eq!(
                CanonicalBlockReconciler::reconcile(Some(100), Some(110), 105, 10, false),
                ReconciliationStrategy::Continue
            );
            // Exactly at depth limit
            assert_eq!(
                CanonicalBlockReconciler::reconcile(Some(100), Some(120), 110, 10, false),
                ReconciliationStrategy::Continue
            );
            // Canonical at earliest pending
            assert_eq!(
                CanonicalBlockReconciler::reconcile(Some(100), Some(105), 100, 10, false),
                ReconciliationStrategy::Continue
            );
            // Zero depth is OK with max_depth=0
            assert_eq!(
                CanonicalBlockReconciler::reconcile(Some(100), Some(105), 100, 0, false),
                ReconciliationStrategy::DepthLimitExceeded { depth: 5, max_depth: 0 }
            );
        }
    }
}
