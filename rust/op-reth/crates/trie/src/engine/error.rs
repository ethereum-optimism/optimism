//! Error type for the live trie engine.

use super::persistence::error::PersistenceError;
use crate::OpProofsStorageError;
use alloy_primitives::B256;
use reth_execution_errors::BlockExecutionError;
use reth_provider::ProviderError;
use thiserror::Error;

/// Errors produced by the live trie engine.
#[derive(Debug, Error)]
pub enum EngineError {
    /// Block was not found in the provider during sync catch-up.
    #[error("Block {0} not found in provider")]
    BlockNotFound(u64),
    /// Reth has pruned the block body, so we cannot recompute its state delta.
    ///
    /// Detected when `recovered_block` returns a block whose header advertises a non-empty
    /// transaction trie but whose body comes back empty: reth still has the header/body-indices
    /// (so `recovered_block` returns `Some`) but the transaction data behind them has been pruned.
    /// Executing such a block would yield zero state changes and a misleading
    /// [`Self::StateRootMismatch`] against the header's real post-state root.
    #[error(
        "Block #{0} body has been pruned by reth; cannot recompute state root. \
         Re-sync reth without body pruning, or reduce the backfill window."
    )]
    BlockBodyPruned(u64),
    /// The background persistence service channel closed unexpectedly.
    #[error("Persistence service disconnected")]
    PersistenceDisconnected,
    /// A persistence save or unwind operation timed out.
    #[error("Persistence operation timed out")]
    PersistenceTimeout,
    /// The collector engine thread terminated unexpectedly.
    #[error("Collector engine terminated unexpectedly")]
    EngineDied,
    /// Block is at the correct number but its parent hash does not match the current tip.
    #[error(
        "Parent hash mismatch at block {block_number}: expected {expected_parent_hash}, got {actual_parent_hash}"
    )]
    ParentHashMismatch {
        /// The block number where the mismatch occurred.
        block_number: u64,
        /// The expected parent hash (current tip hash).
        expected_parent_hash: B256,
        /// The actual parent hash from the block header.
        actual_parent_hash: B256,
    },
    /// The computed state root after EVM execution does not match the block header.
    #[error(
        "State root mismatch for block {block_number} (have: {current_state_hash}, expected: {expected_state_hash})"
    )]
    StateRootMismatch {
        /// The block number where the mismatch occurred.
        block_number: u64,
        /// The actual state root computed from execution.
        current_state_hash: B256,
        /// The expected state root from the block header.
        expected_state_hash: B256,
    },
    /// An error from the persistence layer.
    #[error(transparent)]
    Persistence(#[from] PersistenceError),
    /// A block execution error during EVM execution.
    #[error(transparent)]
    Execution(#[from] BlockExecutionError),
    /// A provider error propagated from the block provider.
    #[error(transparent)]
    Provider(#[from] ProviderError),
    /// A storage-layer error propagated from the underlying store.
    #[error(transparent)]
    Storage(#[from] OpProofsStorageError),
}
