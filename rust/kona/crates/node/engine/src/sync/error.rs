//! Contains the error types used for finding the starting forkchoice state.

use alloy_eips::BlockId;
use alloy_primitives::B256;
use alloy_transport::{RpcError, TransportErrorKind};
use kona_protocol::FromBlockError;
use thiserror::Error;

/// An error that can occur during the sync start process.
#[derive(Error, Debug)]
pub enum SyncStartError {
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
    /// The L1 anchor changed or disappeared while traversing the chain.
    #[error("L1 chain changed during the reset walk")]
    L1ChainChanged,
    /// The execution layer returned heads in an invalid order.
    #[error("L2 heads must satisfy genesis <= finalized <= safe <= unsafe")]
    InvalidHeadOrder,
    /// A response does not match the requested L1 hash or number.
    #[error("L1 response does not match requested block {0}")]
    InvalidL1Response(BlockId),
    /// A response does not match the requested L2 hash or number.
    #[error("L2 response does not match requested block {0}")]
    InvalidL2Response(BlockId),
    /// L1 ancestry did not decrease by exactly one block.
    #[error("L1 parent has an inconsistent block number")]
    InvalidL1Parent,
    /// L2 ancestry did not decrease by exactly one block.
    #[error("L2 parent has an inconsistent block number")]
    InvalidL2Parent,
    /// Replacing unsafe would exceed five sync lookback windows.
    #[error("L2 unsafe reorg exceeds five sync lookback windows")]
    ReorgTooDeep,
    /// The finalized L2 block has a non-canonical L1 origin.
    #[error("Finalized L2 block has a non-canonical L1 origin")]
    FinalizedL1OriginMismatch,
    /// Invalid L1 genesis hash.
    #[error("Invalid L1 genesis hash. Expected {0}, Got {1}")]
    InvalidL1GenesisHash(B256, B256),
    /// Invalid L2 genesis hash.
    #[error("Invalid L2 genesis hash. Expected {0}, Got {1}")]
    InvalidL2GenesisHash(B256, B256),
    /// Finalized block mismatch
    #[error("Finalized block mismatch. Expected {0}, Got {1}")]
    MismatchedFinalizedBlock(B256, B256),
    /// L1 origins do not follow consecutive blocks on the same ancestry.
    #[error("L1 origin mismatch")]
    L1OriginMismatch,
    /// Non-zero sequence number.
    #[error("Non-zero sequence number for block with different L1 origin")]
    NonZeroSequenceNumber,
    /// Inconsistent sequence number.
    #[error("Inconsistent sequence number; Must monotonically increase.")]
    InconsistentSequenceNumber,
}
