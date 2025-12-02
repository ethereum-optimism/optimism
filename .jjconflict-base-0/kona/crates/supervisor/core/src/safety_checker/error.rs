use alloy_primitives::{B256, ChainId};
use kona_interop::InteropValidationError;
use kona_protocol::BlockInfo;
use kona_supervisor_storage::StorageError;
use op_alloy_consensus::interop::SafetyLevel;
use thiserror::Error;

/// Errors returned when validating cross-chain message dependencies.
#[derive(Debug, Error, Eq, PartialEq)]
pub enum CrossSafetyError {
    /// Indicates a failure while accessing storage during dependency checking.
    #[error("storage error: {0}")]
    Storage(#[from] StorageError),

    /// The block that a message depends on does not meet the required safety level.
    #[error(
        "dependency on block {block_number} (chain {chain_id}) does not meet required safety level"
    )]
    DependencyNotSafe {
        /// The ID of the chain containing the unsafe dependency.
        chain_id: ChainId,
        /// The block number of the dependency that failed the safety check
        block_number: u64,
    },

    /// No candidate block is currently available for promotion.
    #[error("no candidate block found to promote")]
    NoBlockToPromote,

    /// The requested safety level is not supported for promotion.
    #[error("promotion to level {0} is not supported")]
    UnsupportedTargetLevel(SafetyLevel),

    /// Indicates that error occurred while validating block
    #[error(transparent)]
    ValidationError(#[from] ValidationError),
}

/// Errors returned when block validation fails due to a fatal violation.
/// These errors indicate that the block must be invalidated.
#[derive(Debug, Error, PartialEq, Eq)]
pub enum ValidationError {
    /// Indicates that error occurred while validating interop config for the block messages
    #[error(transparent)]
    InteropValidationError(#[from] InteropValidationError),

    /// Indicates a mismatch between the executing message hash and the expected original log hash.
    #[error(
        "executing message hash {message_hash} does not match original log hash {original_hash}"
    )]
    InvalidMessageHash {
        /// The hash provided in the executing message.
        message_hash: B256,
        /// The expected hash from the original initiating log.
        original_hash: B256,
    },

    /// Indicates that the timestamp in the executing message does not match the timestamp
    /// of the initiating block, violating the timestamp invariant required for validation.
    #[error(
        "timestamp invariant violated while validating executing message: expected {expected_timestamp}, but found {actual_timestamp}"
    )]
    TimestampInvariantViolation {
        /// The timestamp of the initiating block.
        expected_timestamp: u64,
        /// The timestamp found in the executing message.
        actual_timestamp: u64,
    },

    /// The initiating message corresponding to the executing message could not be found in log
    /// storage.
    #[error("initiating message not found for the executing message")]
    InitiatingMessageNotFound,

    /// Cyclic dependency detected involving the candidate block
    #[error("cyclic dependency detected while promoting block {block}")]
    CyclicDependency {
        /// The candidate block which is creating cyclic dependency
        block: BlockInfo,
    },
}
