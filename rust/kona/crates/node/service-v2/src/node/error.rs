//! V2 node supervision errors.

use crate::{engine::EngineServiceError, unsafe_chain::UnsafeChainError};
use thiserror::Error;

/// A terminal error from a supervised V2 node task.
#[derive(Debug, Error)]
pub enum NodeError {
    /// The engine service stopped unexpectedly.
    #[error(transparent)]
    Engine(#[from] EngineServiceError),
    /// The unsafe-chain service stopped unexpectedly.
    #[error(transparent)]
    UnsafeChain(#[from] UnsafeChainError),
    /// A supervised task panicked or was aborted.
    #[error("supervised node task failed: {0}")]
    Task(String),
}

impl From<tokio::task::JoinError> for NodeError {
    fn from(error: tokio::task::JoinError) -> Self {
        Self::Task(error.to_string())
    }
}
