//! Error types for sealing a block.

use super::{EngineTaskError, EngineTaskErrorSeverity, InsertTaskError};
use alloy_transport::{RpcError, TransportErrorKind};
use thiserror::Error;

/// An error sealing a block, from [`Handle::seal`] or from sealing a block during consolidation.
///
/// [`Handle::seal`]: crate::engine::Handle::seal
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
            Self::GetPayloadFailed(_) => EngineTaskErrorSeverity::Temporary,
            Self::HoloceneInvalidFlush => EngineTaskErrorSeverity::Flush,
            Self::DepositOnlyPayloadReattemptFailed |
            Self::DepositOnlyPayloadFailed |
            Self::ClockWentBackwards |
            Self::UnsafeHeadChangedSinceBuild => EngineTaskErrorSeverity::Critical,
        }
    }
}
