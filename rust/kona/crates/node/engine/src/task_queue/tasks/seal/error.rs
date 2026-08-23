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
    /// Unsafe head changed between build and seal on a *sequencer* build: the job's parent is no
    /// longer the unsafe head, so the job is stale work. This mirrors op-node's `ErrStaleBuild`
    /// (`op-node/rollup/engine/build_start.go:62-68`), which is likewise scoped to sequencer
    /// builds (`!attrs.IsDerived()`).
    ///
    /// The sequencer waits on the task's channel, hears this, and rebuilds on the new head.
    /// Derived builds never produce it: consolidation forces them on the local-safe parent
    /// exactly when the unsafe chain ahead has to be reorged out, so their parent legitimately
    /// differs from the unsafe head and the seal proceeds — failing them instead resets
    /// derivation, which re-derives the same attributes and livelocks. Should a sequencer build
    /// ever run without a channel, the severity is a reset — dropping the stale work — never
    /// Critical, which killed the node over a state it recovers from by re-deriving.
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
