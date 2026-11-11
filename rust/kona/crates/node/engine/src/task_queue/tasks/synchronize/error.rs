//! Contains the error type for the forkchoice update that synchronizes the execution layer.

use crate::{EngineTaskError, task_queue::tasks::task::EngineTaskErrorSeverity};
use alloy_rpc_types_engine::PayloadStatusEnum;
use alloy_transport::{RpcError, TransportErrorKind};
use thiserror::Error;

/// A safe label of the engine's sync state.
#[derive(Debug, Clone, Copy, PartialEq, Eq, derive_more::Display)]
pub enum SafeLabel {
    /// The local-safe head.
    #[display("local-safe")]
    LocalSafe,
    /// The safe head.
    #[display("safe")]
    Safe,
}

/// An error that occurs when synchronizing the execution layer's forkchoice with the engine state.
#[derive(Debug, Error)]
pub enum SynchronizeTaskError {
    /// The forkchoice update call to the engine api failed.
    #[error("Forkchoice update engine api call failed due to an RPC error: {0}")]
    ForkchoiceUpdateFailed(RpcError<TransportErrorKind>),
    /// A safe label would sit below the finalized head.
    #[error(
        "Invalid forkchoice state: {label} head {label_number} is behind finalized head {finalized_number}"
    )]
    FinalizedAheadOfSafe {
        /// The number of the finalized head.
        finalized_number: u64,
        /// The safe label behind the finalized head.
        label: SafeLabel,
        /// The number of that safe label.
        label_number: u64,
    },
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
            // A reset rebuilds the labels, so a label update that would break finality is
            // recoverable without stopping the node.
            Self::FinalizedAheadOfSafe { .. } |
            Self::InvalidForkchoiceState |
            Self::UnexpectedPayloadStatus(PayloadStatusEnum::Invalid { .. }) => {
                EngineTaskErrorSeverity::Reset
            }
            Self::ForkchoiceUpdateFailed(_) | Self::UnexpectedPayloadStatus(_) => {
                EngineTaskErrorSeverity::Temporary
            }
        }
    }
}
