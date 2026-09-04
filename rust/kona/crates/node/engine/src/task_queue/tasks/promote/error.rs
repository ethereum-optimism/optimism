//! Contains error types for the [`crate::PromoteCrossSafeTask`].

use crate::{
    EngineTaskError, SynchronizeTaskError, task_queue::tasks::task::EngineTaskErrorSeverity,
};
use thiserror::Error;

/// An error that occurs when running the [`crate::PromoteCrossSafeTask`].
#[derive(Debug, Error)]
pub enum PromoteCrossSafeTaskError {
    /// The forkchoice update carrying the promoted cross-safe head failed.
    #[error(transparent)]
    ForkchoiceUpdateFailed(#[from] SynchronizeTaskError),
}

impl EngineTaskError for PromoteCrossSafeTaskError {
    fn severity(&self) -> EngineTaskErrorSeverity {
        match self {
            Self::ForkchoiceUpdateFailed(inner) => inner.severity(),
        }
    }
}
