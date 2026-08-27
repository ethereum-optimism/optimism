//! Contains error types for the [`crate::SynchronizeTask`].

use crate::{
    EngineTaskError, InsertTaskError, SealedPayload,
    task_queue::tasks::task::EngineTaskErrorSeverity,
};
use alloy_rpc_types_engine::PayloadId;
use alloy_transport::{RpcError, TransportErrorKind};
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
    /// The execution layer's payload builder no longer knows the build job, so there is nothing
    /// to seal and the block must be rebuilt.
    #[error("Payload build job {0} is unknown to the execution layer")]
    PayloadJobUnknown(PayloadId),
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
    MpscSend(#[from] Box<mpsc::error::SendError<Result<SealedPayload, Self>>>),
    /// The clock went backwards.
    #[error("The clock went backwards")]
    ClockWentBackwards,
    /// Unsafe head changed between build and seal. This likely means that there was some race
    /// condition between the previous seal updating the unsafe head and the build attributes
    /// being created. This build has been invalidated.
    ///
    /// If not propagated to the original caller for handling (i.e. there was no original caller),
    /// this should not happen and is a critical error.
    #[error("Unsafe head changed between build and seal")]
    UnsafeHeadChangedSinceBuild,
}

impl EngineTaskError for SealTaskError {
    fn severity(&self) -> EngineTaskErrorSeverity {
        match self {
            Self::PayloadInsertionFailed(inner) => inner.severity(),
            Self::GetPayloadFailed(_) | Self::PayloadJobUnknown(_) => {
                EngineTaskErrorSeverity::Temporary
            }
            Self::HoloceneInvalidFlush => EngineTaskErrorSeverity::Flush,
            Self::DepositOnlyPayloadReattemptFailed |
            Self::DepositOnlyPayloadFailed |
            Self::MpscSend(_) |
            Self::ClockWentBackwards |
            Self::UnsafeHeadChangedSinceBuild => EngineTaskErrorSeverity::Critical,
        }
    }
}
