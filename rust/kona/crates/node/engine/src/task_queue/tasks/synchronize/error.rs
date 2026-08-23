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
    /// The execution layer answered a forkchoice update with `SYNCING` after its initial sync had
    /// already finished, so the heads the update carries were not adopted.
    ///
    /// The mirror of op-node's steady-state forkchoice check, where only `VALID` is acceptable
    /// once the engine is past EL sync (`op-node/rollup/engine/engine_controller.go:586-595`):
    /// adopting a head the execution layer cannot canonicalize detaches the node's forkchoice
    /// from the chain the EL actually has — derivation then consolidates against unsafe blocks
    /// the EL cannot serve, and stalls. During the initial EL sync the same answer is expected
    /// and accepted, the mirror of op-node's EL-sync regime (`engine_controller.go:591-594`).
    ///
    /// Answered with a reset, as op-node answers a `SYNCING` forkchoice status outside the
    /// unsafe-insert path (`engine_controller.go:700-706`): re-discover the execution layer's
    /// actual chain state rather than retry an update it already refused. The unsafe-insert path
    /// intercepts this error and drops the payload instead (see
    /// [`crate::EngineTask`]'s insert arm), the counterpart of op-node not adopting the head and
    /// not resetting there either (`engine_controller.go:873-879`).
    #[error("Forkchoice update returned SYNCING after EL sync finished")]
    ForkchoiceUpdatedSyncing,
}

impl EngineTaskError for SynchronizeTaskError {
    fn severity(&self) -> EngineTaskErrorSeverity {
        match self {
            Self::FinalizedAheadOfUnsafe(_, _) => EngineTaskErrorSeverity::Critical,
            Self::ForkchoiceUpdateFailed(_) | Self::UnexpectedPayloadStatus(_) => {
                EngineTaskErrorSeverity::Temporary
            }
            Self::InvalidForkchoiceState | Self::ForkchoiceUpdatedSyncing => {
                EngineTaskErrorSeverity::Reset
            }
        }
    }
}
