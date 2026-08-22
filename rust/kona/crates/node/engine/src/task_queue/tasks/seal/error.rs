//! Contains error types for the [`crate::SynchronizeTask`].

use crate::{EngineTaskError, InsertTaskError, task_queue::tasks::task::EngineTaskErrorSeverity};
use alloy_transport::{RpcError, TransportErrorKind};
use op_alloy_rpc_types_engine::OpExecutionPayloadEnvelope;
use thiserror::Error;
use tokio::sync::mpsc;

/// An error that occurs when running the [`crate::SealTask`].
#[derive(Debug, Error)]
pub enum SealTaskError {
    /// Impossible to insert the payload into the engine.
    #[error(transparent)]
    PayloadInsertionFailed(#[from] Box<InsertTaskError>),
    /// The get payload call to the engine api failed.
    #[error(transparent)]
    GetPayloadFailed(RpcError<TransportErrorKind>),
    /// A deposit-only payload failed to import.
    #[error("Deposit-only payload failed to import")]
    DepositOnlyPayloadFailed,
    /// Failed to re-attempt payload import with deposit-only payload.
    #[error("Failed to re-attempt payload import with deposit-only payload")]
    DepositOnlyPayloadReattemptFailed,
    /// The payload is invalid, and the derivation pipeline must
    /// be flushed post-holocene.
    #[error("Invalid payload, must flush post-holocene")]
    HoloceneInvalidFlush,
    /// Error sending the built payload envelope.
    #[error(transparent)]
    MpscSend(#[from] Box<mpsc::error::SendError<Result<OpExecutionPayloadEnvelope, Self>>>),
    /// The clock went backwards.
    #[error("The clock went backwards")]
    ClockWentBackwards,
    /// Unsafe head changed between build and seal. This likely means that there was some race
    /// condition between the previous seal updating the unsafe head and the build attributes
    /// being created. This build has been invalidated.
    ///
    /// When a caller is waiting on the task's channel, it hears this and rebuilds. Without one —
    /// the consolidation path — the state that moved is real and expected: a deposits-only
    /// replacement (the Holocene fallback, or an invalidation's replacement block) lands as the
    /// new unsafe head underneath a consolidate task still queued with the pre-replacement
    /// attributes. That stale work must be dropped and re-derived, which is a reset — the same
    /// answer op-node gives when its queued attributes conflict with a moved pending-safe head
    /// (`op-node/rollup/attributes/attributes.go:175-183`, a `ResetEvent`). Escalating it to
    /// Critical instead kills the node over a state it recovers from by re-deriving.
    #[error("Unsafe head changed between build and seal")]
    UnsafeHeadChangedSinceBuild,
}

impl EngineTaskError for SealTaskError {
    fn severity(&self) -> EngineTaskErrorSeverity {
        match self {
            Self::PayloadInsertionFailed(inner) => inner.severity(),
            Self::GetPayloadFailed(_) => EngineTaskErrorSeverity::Temporary,
            Self::HoloceneInvalidFlush => EngineTaskErrorSeverity::Flush,
            Self::UnsafeHeadChangedSinceBuild => EngineTaskErrorSeverity::Reset,
            Self::DepositOnlyPayloadReattemptFailed |
            Self::DepositOnlyPayloadFailed |
            Self::MpscSend(_) |
            Self::ClockWentBackwards => EngineTaskErrorSeverity::Critical,
        }
    }
}
