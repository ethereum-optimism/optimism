//! Errors from resetting the engine.

use super::{EngineTaskError, EngineTaskErrorSeverity, SynchronizeTaskError};
use crate::engine::SyncStartError;
use thiserror::Error;

/// An error from resetting the engine.
#[derive(Debug, Error)]
pub enum EngineResetError {
    /// An error that occurred while updating the forkchoice state.
    #[error(transparent)]
    Forkchoice(#[from] SynchronizeTaskError),
    /// An error occurred while traversing the L1 for the sync starting point.
    #[error(transparent)]
    SyncStart(#[from] SyncStartError),
}

impl EngineResetError {
    /// Whether retrying the reset can recover without changing the node configuration.
    pub fn is_temporary(&self) -> bool {
        match self {
            Self::SyncStart(SyncStartError::RpcError(_)) => true,
            Self::SyncStart(_) => false,
            Self::Forkchoice(err) => err.severity() != EngineTaskErrorSeverity::Critical,
        }
    }
}
