//! Errors interfacing with [`OpProofsStore`](crate::OpProofsStore) type.

use alloy_primitives::B256;
use reth_db::DatabaseError;
use reth_execution_errors::BlockExecutionError;
use reth_provider::ProviderError;
use reth_trie_common::Nibbles;
use std::sync::Arc;
use thiserror::Error;

/// Error type for storage operations
#[derive(Debug, Clone, Error)]
pub enum OpProofsStorageError {
    /// No blocks found
    #[error("No blocks found")]
    NoBlocksFound,
    /// Parent block number is less than earliest stored block number
    #[error("Parent block number is less than earliest stored block number")]
    UnknownParent,
    /// Block is out of order
    #[error("Block {block_number} is out of order (parent: {parent_block_hash}, latest stored block hash: {latest_block_hash})")]
    OutOfOrder {
        /// The block number being inserted
        block_number: u64,
        /// The parent hash of the block being inserted
        parent_block_hash: B256,
        /// block hash of the latest stored block
        latest_block_hash: B256,
    },
    /// Block update failed since parent state
    #[error("Cannot execute block updates for block {block_number} without parent state {parent_block_number} (latest stored block number: {latest_block_number})")]
    MissingParentBlock {
        /// The block number being executed
        block_number: u64,
        /// The parent state of the block being executed
        parent_block_number: u64,
        /// Latest stored block number
        latest_block_number: u64,
    },
    /// State root mismatch
    #[error("State root mismatch for block {block_number} (have: {current_state_hash}, expected: {expected_state_hash})")]
    StateRootMismatch {
        /// Block number
        block_number: u64,
        /// Have state root
        current_state_hash: B256,
        /// Expected state root
        expected_state_hash: B256,
    },
    /// No change set for block
    #[error("No change set found for block {0}")]
    NoChangeSetForBlock(u64),
    /// Missing account trie history for a specific path at a specific block number
    #[error("Missing account trie history for path {0:?} at block {1}")]
    MissingAccountTrieHistory(Nibbles, u64),
    /// Missing storage trie history for a specific address and path at a specific block number
    #[error("Missing storage trie history for address {0:?}, path {1:?} at block {2}")]
    MissingStorageTrieHistory(B256, Nibbles, u64),
    /// Missing hashed account history for a specific key at a specific block number
    #[error("Missing hashed account history for key {0:?} at block {1}")]
    MissingHashedAccountHistory(B256, u64),
    /// Missing hashed storage history for a specific address and key at a specific block number
    #[error("Missing hashed storage history for address {hashed_address:?}, key {hashed_storage_key:?} at block {block_number}")]
    MissingHashedStorageHistory {
        /// The hashed address
        hashed_address: B256,
        /// The hashed storage key
        hashed_storage_key: B256,
        /// The block number
        block_number: u64,
    },
    /// Attempted to unwind to a block beyond the earliest stored block
    #[error("Attempted to unwind to block {unwind_block_number} beyond earliest stored block {earliest_block_number}")]
    UnwindBeyondEarliest {
        /// The block number being unwound to
        unwind_block_number: u64,
        /// The earliest stored block number
        earliest_block_number: u64,
    },
    /// Error occurred while interacting with the database.
    #[error(transparent)]
    DatabaseError(DatabaseError),
    /// Error occurred while trying to acquire a lock.
    #[error("failed lock attempt")]
    TryLockError,
    /// Error occurred during block execution.
    #[error(transparent)]
    ExecutionError(Arc<BlockExecutionError>),
    /// Error occurred while interacting with the provider.
    #[error(transparent)]
    ProviderError(Arc<ProviderError>),
    /// Initialization detected inconsistent state between proofs storage and source DB.
    #[error(
        "Initialization Proofs storage detected inconsistent state. Storage does not match source DB. \
         Please clear proofs data and retry initialization."
    )]
    InitializeStorageInconsistentState,
}

impl From<BlockExecutionError> for OpProofsStorageError {
    fn from(error: BlockExecutionError) -> Self {
        Self::ExecutionError(Arc::new(error))
    }
}

impl From<ProviderError> for OpProofsStorageError {
    fn from(error: ProviderError) -> Self {
        Self::ProviderError(Arc::new(error))
    }
}

impl From<OpProofsStorageError> for DatabaseError {
    fn from(error: OpProofsStorageError) -> Self {
        match error {
            OpProofsStorageError::DatabaseError(err) => err,
            _ => Self::Custom(Arc::new(error)),
        }
    }
}

impl From<DatabaseError> for OpProofsStorageError {
    fn from(error: DatabaseError) -> Self {
        if let DatabaseError::Custom(ref err) = error &&
            let Some(err) = err.downcast_ref::<Self>()
        {
            return err.clone()
        }
        Self::DatabaseError(error)
    }
}

/// Result type for storage operations
pub type OpProofsStorageResult<T> = Result<T, OpProofsStorageError>;

#[cfg(test)]
mod test {
    use super::*;
    use reth_execution_errors::BlockValidationError;

    #[test]
    fn test_op_proofs_store_error_to_db_error() {
        let original_error = OpProofsStorageError::NoBlocksFound;

        let db_error = DatabaseError::from(original_error);
        assert!(matches!(db_error, DatabaseError::Custom(_)));

        let converted_error = OpProofsStorageError::from(db_error);
        assert!(matches!(converted_error, OpProofsStorageError::NoBlocksFound))
    }

    #[test]
    fn test_db_error_to_op_proofs_store_error() {
        let original_error = DatabaseError::Decode;

        let op_proofs_store_error = OpProofsStorageError::from(original_error);
        assert!(matches!(
            op_proofs_store_error,
            OpProofsStorageError::DatabaseError(DatabaseError::Decode)
        ));

        let converted_error = DatabaseError::from(op_proofs_store_error);
        assert!(matches!(converted_error, DatabaseError::Decode))
    }

    #[test]
    fn test_conversion_from_block_execution_error() {
        let block_execution_error =
            BlockExecutionError::Validation(BlockValidationError::IncrementBalanceFailed);

        let op_proofs_store_error = OpProofsStorageError::from(block_execution_error);
        assert!(
            matches!(op_proofs_store_error, OpProofsStorageError::ExecutionError(err) if matches!(*err, BlockExecutionError::Validation(BlockValidationError::IncrementBalanceFailed)))
        )
    }

    #[test]
    fn test_conversion_from_provider_error() {
        let provider_error = ProviderError::SenderRecoveryError;
        let op_proofs_store_error = OpProofsStorageError::from(provider_error);
        assert!(
            matches!(op_proofs_store_error, OpProofsStorageError::ProviderError(err) if matches!(*err, ProviderError::SenderRecoveryError))
        )
    }
}
