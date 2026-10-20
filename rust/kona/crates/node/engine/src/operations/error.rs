//! Errors retain the Engine API stage so the owner can choose retry, reset or shutdown.

use crate::SyncStartError;
use alloy_eips::{BlockId, BlockNumHash};
use alloy_rpc_types_engine::{INVALID_FORK_CHOICE_STATE_ERROR, PayloadStatusEnum};
use alloy_transport::TransportError;
use kona_protocol::FromBlockError;
use op_alloy_rpc_types_engine::OpPayloadError;

/// A forkchoice update failed or could not be confirmed.
#[derive(Debug, thiserror::Error)]
pub enum ForkchoiceError {
    /// The RPC response does not establish whether forkchoice was applied.
    #[error(transparent)]
    Rpc(TransportError),
    /// The execution layer rejected the forkchoice pre-state (`-38002`).
    #[error("Invalid forkchoice state")]
    InvalidState,
    /// The requested unsafe head is below finalized.
    #[error("Unsafe head {0} is below finalized head {1}")]
    FinalizedAheadOfUnsafe(u64, u64),
    /// The execution layer did not accept the forkchoice.
    #[error("Unexpected forkchoice status: {0}")]
    UnexpectedStatus(PayloadStatusEnum),
}

impl From<TransportError> for ForkchoiceError {
    fn from(error: TransportError) -> Self {
        if error.as_error_resp().is_some_and(|e| e.code == INVALID_FORK_CHOICE_STATE_ERROR as i64) {
            return Self::InvalidState;
        }
        Self::Rpc(error)
    }
}

/// Starting a build failed.
#[derive(Debug, thiserror::Error)]
pub enum BuildError {
    /// Forkchoice was rejected or its application is uncertain.
    #[error(transparent)]
    Forkchoice(#[from] ForkchoiceError),
    /// The execution layer rejected the derived payload.
    #[error("Build rejected: {0}")]
    Rejected(String),
    /// Payload attributes were rejected (`-38003`); forkchoice may already have changed.
    #[error("Invalid payload attributes: {0}")]
    InvalidAttributes(TransportError),
    /// The execution layer is still syncing.
    #[error("Execution layer is syncing")]
    Syncing,
    /// Forkchoice was accepted but no build identifier was returned.
    #[error("Accepted build has no payload ID")]
    MissingPayloadId,
    /// The response is not a build-start status.
    #[error("Unexpected build status: {0}")]
    UnexpectedStatus(PayloadStatusEnum),
}

/// Importing or canonicalizing an execution payload failed.
#[derive(Debug, thiserror::Error)]
pub enum InsertError {
    /// The idempotent newPayload request failed.
    #[error(transparent)]
    Rpc(TransportError),
    /// The payload was not accepted for this import mode.
    #[error("Payload was not accepted: {0}")]
    Rejected(PayloadStatusEnum),
    /// The execution payload cannot be decoded.
    #[error(transparent)]
    Payload(#[from] OpPayloadError),
    /// The decoded block does not contain valid rollup metadata.
    #[error(transparent)]
    BlockInfo(#[from] FromBlockError),
    /// Canonicalizing the inserted block failed.
    #[error(transparent)]
    Forkchoice(#[from] ForkchoiceError),
}

/// Applying derived attributes or an upstream safe block failed.
#[derive(Debug, thiserror::Error)]
pub enum ConsolidateError {
    /// Reading the existing unsafe block failed.
    #[error(transparent)]
    Read(TransportError),
    /// Starting a derived build failed.
    #[error(transparent)]
    Build(#[from] BuildError),
    /// Retrieving the derived payload failed.
    #[error(transparent)]
    GetPayload(TransportError),
    /// Importing the derived payload failed.
    #[error(transparent)]
    Insert(#[from] InsertError),
    /// Updating safety labels failed.
    #[error(transparent)]
    Forkchoice(#[from] ForkchoiceError),
    /// An essential deposits-only block was deterministically rejected.
    #[error("Deposits-only block was rejected: {0}")]
    DepositsOnlyRejected(String),
}

/// Finalizing a hash-pinned block failed.
#[derive(Debug, thiserror::Error)]
pub enum FinalizeError {
    /// Finalization cannot advance past the applied safe head.
    #[error("Block is not safe")]
    NotSafe,
    /// A required L1 or L2 block is unavailable.
    #[error("Block not found: {0}")]
    BlockNotFound(BlockId),
    /// A dependency read failed.
    #[error(transparent)]
    Rpc(#[from] TransportError),
    /// The L2 block's rollup metadata is malformed.
    #[error(transparent)]
    BlockInfo(#[from] FromBlockError),
    /// The recorded L2 block is not canonical at its claimed height.
    #[error("L2 finalization candidate is no longer canonical: {0:?}")]
    L2Mismatch(BlockNumHash),
    /// The recorded inclusion source is no longer canonical.
    #[error("L1 finalization source is no longer canonical: {0:?}")]
    L1Mismatch(BlockNumHash),
    /// Finalization forkchoice failed.
    #[error(transparent)]
    Forkchoice(#[from] ForkchoiceError),
}

/// Finding or applying reset heads failed.
#[derive(Debug, thiserror::Error)]
pub enum ResetError {
    /// The canonical reset walk failed.
    #[error(transparent)]
    SyncStart(#[from] SyncStartError),
    /// Applying reset heads failed.
    #[error(transparent)]
    Forkchoice(#[from] ForkchoiceError),
}

impl ResetError {
    /// Whether a later reset attempt can recover without changing the node configuration.
    pub const fn is_temporary(&self) -> bool {
        match self {
            Self::SyncStart(
                SyncStartError::RpcError(_) |
                SyncStartError::BlockNotFound(_) |
                SyncStartError::L1ChainChanged,
            ) => true,
            Self::SyncStart(_) | Self::Forkchoice(ForkchoiceError::FinalizedAheadOfUnsafe(..)) => {
                false
            }
            Self::Forkchoice(_) => true,
        }
    }
}
