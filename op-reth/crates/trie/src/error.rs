//! Errors interfacing with [`OpProofsStore`](crate::OpProofsStore) type.

use alloy_primitives::B256;
use reth_db::DatabaseError;
use reth_trie::Nibbles;
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
