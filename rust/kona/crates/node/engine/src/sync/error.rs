//! Contains the error types used for finding the starting forkchoice state.

use alloy_eips::{BlockId, BlockNumHash};
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
    /// Invalid L1 genesis hash.
    #[error("Invalid L1 genesis hash. Expected {0}, Got {1}")]
    InvalidL1GenesisHash(B256, B256),
    /// Invalid L2 genesis hash.
    #[error("Invalid L2 genesis hash. Expected {0}, Got {1}")]
    InvalidL2GenesisHash(B256, B256),
    /// The L1 provider holds a different block at the height of the L1 origin of the execution
    /// layer's finalized block, so it follows another L1 chain.
    #[error("L1 provider has {l1_block:?} at the height of finalized L1 origin {l1_origin:?}")]
    FinalizedL1OriginNotCanonical {
        /// The L1 provider's block at that height.
        l1_block: BlockNumHash,
        /// The finalized block's L1 origin.
        l1_origin: BlockNumHash,
    },
    /// The L1 provider serves the L1 origin of the execution layer's finalized block by number
    /// but not by hash, as a broken or lagging block-hash index does.
    #[error("L1 provider serves finalized L1 origin {l1_origin:?} by number but not by hash")]
    FinalizedL1OriginNotServedByHash {
        /// The L1 origin it serves only by number.
        l1_origin: BlockNumHash,
    },
    /// The L1 provider does not have the L1 origin of the execution layer's finalized block.
    #[error("L1 origin {l1_origin:?} of finalized L2 block {l2_block:?} is not available")]
    FinalizedL1OriginUnavailable {
        /// The missing L1 origin.
        l1_origin: BlockNumHash,
        /// The finalized L2 block.
        l2_block: BlockNumHash,
    },
    /// The execution layer returned no block at its finalized height, although its canonical chain
    /// extends above it, as a briefly inconsistent execution layer does.
    #[error("Execution layer has no block at finalized height {0}")]
    FinalizedHeightUnavailable(u64),
    /// The chain walked from the execution layer's unsafe head does not contain its finalized
    /// block.
    #[error(
        "Walked chain reached finalized height on {walked:?}, not finalized block {finalized:?}"
    )]
    MismatchedFinalizedBlock {
        /// The execution layer's finalized block.
        finalized: BlockNumHash,
        /// The walked block at or below the finalized height.
        walked: BlockNumHash,
    },
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
