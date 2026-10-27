//! Contains the error types used for finding the starting forkchoice state.

use alloy_eips::BlockId;
use alloy_primitives::B256;
use alloy_rpc_types_engine::PayloadStatus;
use alloy_transport::{RpcError, TransportErrorKind};
use kona_protocol::FromBlockError;
use thiserror::Error;

/// An error that can occur during the sync start process.
#[derive(Error, Debug)]
pub enum SyncStartError {
    /// The EL is still syncing; retry without trusting the retained checkpoint's labels.
    #[error("Execution layer has not validated the sync-start checkpoint")]
    CheckpointNotValidated,
    /// The EL rejected the checkpoint forkchoice or returned an unexpected status; do not retry.
    #[error("Execution layer rejected the sync-start checkpoint: {0:?}")]
    CheckpointRejected(PayloadStatus),
    /// The canonical L1 view changed during recovery; retry with a fresh view.
    #[error("Canonical chain changed during sync-start recovery")]
    ChainViewChanged,
    /// An rpc error occurred
    #[error("An RPC error occurred: {0}")]
    RpcError(#[from] RpcError<TransportErrorKind>),
    /// An error occurred while converting a block to [`L2BlockInfo`].
    ///
    /// [`L2BlockInfo`]: kona_protocol::L2BlockInfo
    #[error(transparent)]
    FromBlock(#[from] FromBlockError),
    /// A block could not be found.
    #[error("Block not found: {0}")]
    BlockNotFound(BlockId),
    /// Invalid L1 genesis hash.
    #[error("Invalid L1 genesis hash. Expected {0}, Got {1}")]
    InvalidL1GenesisHash(B256, B256),
    /// Invalid L2 genesis hash.
    #[error("Invalid L2 genesis hash. Expected {0}, Got {1}")]
    InvalidL2GenesisHash(B256, B256),
    /// Finalized block mismatch
    #[error("Finalized block mismatch. Expected {0}, Got {1}")]
    MismatchedFinalizedBlock(B256, B256),
    /// Recovery would rewind the unsafe origin by more than op-node's five lookback windows.
    #[error("L1 reorg exceeds five sync lookback windows")]
    ReorgTooDeep,
    /// L1 origin mismatch.
    #[error("L1 origin mismatch")]
    L1OriginMismatch,
    /// Non-zero sequence number.
    #[error("Non-zero sequence number for block with different L1 origin")]
    NonZeroSequenceNumber,
    /// Inconsistent sequence number.
    #[error("Inconsistent sequence number; Must monotonically increase.")]
    InconsistentSequenceNumber,
}
