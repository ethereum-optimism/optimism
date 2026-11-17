//! Contains the error types used for finding the starting forkchoice state.

use alloy_eips::BlockId;
use alloy_primitives::B256;
use alloy_transport::{RpcError, TransportErrorKind};
use kona_protocol::FromBlockError;
use kona_providers_alloy::AlloyChainProviderError;
use thiserror::Error;

/// An error that can occur during the sync start process.
#[derive(Error, Debug)]
pub enum SyncStartError {
    /// An rpc error occurred
    #[error("An RPC error occurred: {0}")]
    RpcError(#[from] RpcError<TransportErrorKind>),
    /// The L1 provider failed.
    #[error("L1 provider error: {0}")]
    L1(#[from] AlloyChainProviderError),
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
    /// The unsafe chain does not lead back to a block it must descend from.
    #[error("Unsafe chain does not lead back to {expected}; found {found}")]
    UnsafeChainMismatch {
        /// The block the unsafe chain must descend from.
        expected: B256,
        /// The block the unsafe chain has at that height.
        found: B256,
    },
    /// The unsafe head is below a block it must descend from.
    #[error("Unsafe head {unsafe_head} is below block {floor}, which it must descend from")]
    UnsafeHeadBelowFloor {
        /// The unsafe head's number.
        unsafe_head: u64,
        /// The number of the block it must descend from.
        floor: u64,
    },
    /// The L1 RPC reported a head at or above a height, then served no block there.
    #[error("L1 block {number} is missing below the L1 head {l1_head}")]
    L1BlockMissing {
        /// The missing L1 block's number.
        number: u64,
        /// The L1 head the RPC reported.
        l1_head: u64,
    },
    /// The unsafe head would rewind across more L1 origins than a plausible L1 reorg.
    #[error("Reorg too deep: unsafe head {previous_unsafe_head} would rewind to {unsafe_head}")]
    TooDeepReorg {
        /// The L2 block the unsafe head would rewind to.
        unsafe_head: u64,
        /// The unsafe head before the reset.
        previous_unsafe_head: u64,
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
