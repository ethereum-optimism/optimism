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

impl EngineError {
    /// Returns whether executing the block again may succeed.
    ///
    /// Execution reads bytecode and block hashes from reth. If reth has left the block's fork,
    /// those reads can come from another chain and execution fails; the sync loop then retries
    /// with reth's canonical block at that height. Storage, persistence, and provider errors are
    /// not retryable, including a provider error raised while the EVM reads state.
    pub fn is_retryable(&self) -> bool {
        match self {
            Self::StateRootMismatch { .. } => true,
            Self::Execution(err) => !is_provider_failure(err),
            _ => false,
        }
    }
}

/// Returns whether `err` wraps a [`ProviderError`] raised while executing the block.
///
/// The EVM reports database failures as internal errors whose payload is not exposed as a
/// `source`, so the walk starts from the payload.
fn is_provider_failure(err: &BlockExecutionError) -> bool {
    let Some(internal) = err.as_internal() else { return false };
    let Some(payload) = internal.as_evm().map(|(_, err)| err).or_else(|| internal.as_other())
    else {
        return false;
    };
    let payload: &(dyn core::error::Error + 'static) = payload;
    core::iter::successors(Some(payload), |err| err.source()).any(|err| err.is::<ProviderError>())
}

#[cfg(test)]
mod tests {
    use super::*;
    use alloy_primitives::B256;
    use reth_revm::revm::context::result::{EVMError, InvalidTransaction};

    #[test]
    fn provider_failures_during_execution_are_not_retryable() {
        let database = EVMError::<ProviderError, InvalidTransaction>::Database(
            ProviderError::StateForHashNotFound(B256::ZERO),
        );
        let err = EngineError::Execution(BlockExecutionError::evm(database, B256::ZERO));
        assert!(!err.is_retryable(), "{err}");

        let other = EngineError::Execution(BlockExecutionError::other(
            ProviderError::StateForHashNotFound(B256::ZERO),
        ));
        assert!(!other.is_retryable(), "{other}");
    }

    #[test]
    fn execution_and_state_root_failures_are_retryable() {
        let invalid_tx = EVMError::<ProviderError, InvalidTransaction>::Transaction(
            InvalidTransaction::NonceTooHigh { tx: 5, state: 1 },
        );
        assert!(
            EngineError::Execution(BlockExecutionError::evm(invalid_tx, B256::ZERO)).is_retryable()
        );
        assert!(
            EngineError::StateRootMismatch {
                block_number: 1,
                current_state_hash: B256::ZERO,
                expected_state_hash: B256::ZERO,
            }
            .is_retryable()
        );
        assert!(
            !EngineError::Provider(ProviderError::StateForHashNotFound(B256::ZERO)).is_retryable()
        );
    }
}
