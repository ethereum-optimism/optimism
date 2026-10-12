//! Contains error types for the [`crate::SynchronizeTask`].

use crate::{EngineTaskError, task_queue::tasks::task::EngineTaskErrorSeverity};
use alloy_rpc_types_engine::PayloadStatusEnum;
use alloy_transport::{RpcError, TransportErrorKind};
use thiserror::Error;

/// An error that occurs when running the [`crate::SynchronizeTask`].
#[derive(Debug, Error)]
pub enum SynchronizeTaskError {
    /// The forkchoice update call to the engine api failed.
    #[error("Forkchoice update engine api call failed due to an RPC error: {0}")]
    ForkchoiceUpdateFailed(RpcError<TransportErrorKind>),
    /// The finalized head is behind the unsafe head.
    #[error("Invalid forkchoice state: unsafe head {0} is ahead of finalized head {1}")]
    FinalizedAheadOfUnsafe(u64, u64),
    /// The forkchoice state is invalid.
    #[error("Invalid forkchoice state")]
    InvalidForkchoiceState,
    /// The payload status is unexpected.
    #[error("Unexpected payload status: {0}")]
    UnexpectedPayloadStatus(PayloadStatusEnum),
}

impl EngineTaskError for SynchronizeTaskError {
    fn severity(&self) -> EngineTaskErrorSeverity {
        match self {
            Self::FinalizedAheadOfUnsafe(_, _) => EngineTaskErrorSeverity::Critical,
            Self::ForkchoiceUpdateFailed(_) => EngineTaskErrorSeverity::Temporary,
            // The execution layer rejected the forkchoice (INVALID or ACCEPTED). Retrying returns
            // the same verdict, so rediscover the forkchoice from the execution layer instead.
            Self::InvalidForkchoiceState | Self::UnexpectedPayloadStatus(_) => {
                EngineTaskErrorSeverity::Reset
            }
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[rstest::rstest]
    #[case::invalid(PayloadStatusEnum::Invalid { validation_error: String::new() })]
    #[case::accepted(PayloadStatusEnum::Accepted)]
    fn rejected_forkchoice_requires_a_reset(#[case] status: PayloadStatusEnum) {
        assert_eq!(
            SynchronizeTaskError::UnexpectedPayloadStatus(status).severity(),
            EngineTaskErrorSeverity::Reset
        );
    }
}
