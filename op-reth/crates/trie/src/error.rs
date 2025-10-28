//! Errors interfacing with [`OpProofsStore`](crate::OpProofsStore) type.

use alloy_primitives::B256;
use reth_db::DatabaseError;
use thiserror::Error;

/// Error type for storage operations
#[derive(Debug, Error)]
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
    #[error("Cannot execute block updates for block {0} without parent state {1} (latest stored block number: {2})")]
    BlockUpdateFailed(u64, u64, u64),
    /// State root mismatch
    #[error("State root mismatch for block {0} (have: {1}, expected: {2})")]
    StateRootMismatch(u64, B256, B256),
    /// Error occurred while interacting with the database.
    #[error(transparent)]
    DatabaseError(#[from] DatabaseError),
    /// Other error
    #[error("Other error: {0}")]
    Other(eyre::Error),
}

impl From<OpProofsStorageError> for DatabaseError {
    fn from(error: OpProofsStorageError) -> Self {
        Self::Other(error.to_string())
    }
}

/// Result type for storage operations
pub type OpProofsStorageResult<T> = Result<T, OpProofsStorageError>;
