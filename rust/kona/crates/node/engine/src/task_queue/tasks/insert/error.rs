//! Contains the error types for the [`InsertTask`](crate::InsertTask).

use crate::{
    EngineTaskError, ExecutionPayloadEnvelopeVersionError, SynchronizeTaskError,
    task_queue::tasks::task::EngineTaskErrorSeverity,
};
use alloy_json_rpc::ErrorPayload;
use alloy_rpc_types_engine::PayloadStatusEnum;
use alloy_transport::{RpcError, TransportErrorKind};
use kona_protocol::FromBlockError;
use op_alloy_rpc_types_engine::OpPayloadError;

/// Engine API request too large error code.
const TOO_LARGE_REQUEST_ERROR: i64 = -38004;
/// Engine API unsupported fork error code.
const UNSUPPORTED_FORK_ERROR: i64 = -38005;

/// The origin of an execution payload envelope passed to an [`InsertTask`](crate::InsertTask).
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum PayloadEnvelopeOrigin {
    /// The payload was locally built from attributes derived from L1.
    L1Derived,
    /// The payload was locally built by this node's sequencer.
    LocalSequencer,
    /// The payload was received from a remote sequencer through gossip or the admin RPC.
    RemoteSequencer,
}

impl PayloadEnvelopeOrigin {
    /// Returns whether the payload advances the safe chain.
    pub const fn is_safe(self) -> bool {
        matches!(self, Self::L1Derived)
    }

    /// Returns whether the payload came from outside this node and its configured execution layer.
    pub const fn is_remote(self) -> bool {
        matches!(self, Self::RemoteSequencer)
    }
}

/// An error that occurs when running the [`InsertTask`](crate::InsertTask).
#[derive(Debug, thiserror::Error)]
#[error("{kind}")]
pub struct InsertTaskError {
    /// The origin of the payload that failed insertion.
    origin: PayloadEnvelopeOrigin,
    /// The underlying insertion failure.
    #[source]
    kind: InsertTaskErrorKind,
}

impl InsertTaskError {
    /// Creates an insertion error for a payload with the given origin.
    pub const fn new(origin: PayloadEnvelopeOrigin, kind: InsertTaskErrorKind) -> Self {
        Self { origin, kind }
    }

    /// Returns the underlying insertion failure.
    pub const fn kind(&self) -> &InsertTaskErrorKind {
        &self.kind
    }

    /// Classifies an Engine API request failure.
    fn insert_failed_severity(
        &self,
        error: &RpcError<TransportErrorKind>,
    ) -> EngineTaskErrorSeverity {
        match error {
            // These errors describe a malformed remote payload. The same errors for a payload
            // produced by our configured EL indicate a local invariant or compatibility failure.
            RpcError::ErrorResp(response)
                if response.code == ErrorPayload::<()>::invalid_params().code ||
                    response.code == TOO_LARGE_REQUEST_ERROR =>
            {
                if self.origin.is_remote() {
                    EngineTaskErrorSeverity::Drop
                } else {
                    EngineTaskErrorSeverity::Critical
                }
            }
            // These indicate a client or EL capability mismatch, not a bad individual payload.
            RpcError::ErrorResp(response)
                if response.code == ErrorPayload::<()>::invalid_request().code ||
                    response.code == ErrorPayload::<()>::method_not_found().code ||
                    response.code == UNSUPPORTED_FORK_ERROR =>
            {
                EngineTaskErrorSeverity::Critical
            }
            RpcError::UnsupportedFeature(_) |
            RpcError::LocalUsageError(_) |
            RpcError::SerError(_) => EngineTaskErrorSeverity::Critical,
            RpcError::Transport(error) if error.is_non_retryable() => {
                EngineTaskErrorSeverity::Critical
            }
            _ => EngineTaskErrorSeverity::Temporary,
        }
    }
}

impl EngineTaskError for InsertTaskError {
    fn severity(&self) -> EngineTaskErrorSeverity {
        match &self.kind {
            InsertTaskErrorKind::FromBlockError(_) |
            InsertTaskErrorKind::UnexpectedPayloadVersion(_) |
            InsertTaskErrorKind::L2BlockInfoConstruction(_) => {
                if self.origin.is_remote() {
                    EngineTaskErrorSeverity::Drop
                } else {
                    EngineTaskErrorSeverity::Critical
                }
            }
            InsertTaskErrorKind::InsertFailed(error) => self.insert_failed_severity(error),
            InsertTaskErrorKind::UnexpectedPayloadStatus(status)
                if self.origin.is_remote() && status.is_invalid() =>
            {
                EngineTaskErrorSeverity::Drop
            }
            InsertTaskErrorKind::UnexpectedPayloadStatus(_) => EngineTaskErrorSeverity::Temporary,
            InsertTaskErrorKind::ForkchoiceUpdateFailed(inner) => inner.severity(),
        }
    }
}

/// The underlying cause of an [`InsertTaskError`].
#[derive(Debug, thiserror::Error)]
pub enum InsertTaskErrorKind {
    /// Error converting a payload into a block.
    #[error(transparent)]
    FromBlockError(#[from] OpPayloadError),
    /// Failed to insert new payload.
    #[error("Failed to insert new payload: {0}")]
    InsertFailed(RpcError<TransportErrorKind>),
    /// Unexpected payload status.
    #[error("Unexpected payload status: {0}")]
    UnexpectedPayloadStatus(PayloadStatusEnum),
    /// The payload envelope version does not match the fork active at its timestamp.
    #[error(transparent)]
    UnexpectedPayloadVersion(#[from] ExecutionPayloadEnvelopeVersionError),
    /// Error converting the payload + chain genesis into an L2 block info.
    #[error(transparent)]
    L2BlockInfoConstruction(#[from] FromBlockError),
    /// The forkchoice update call to consolidate the block into the engine state failed.
    #[error(transparent)]
    ForkchoiceUpdateFailed(#[from] SynchronizeTaskError),
}
