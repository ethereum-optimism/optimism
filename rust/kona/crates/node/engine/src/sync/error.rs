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
    #[error("L2 execution layer served an invalid L2 block: {0}")]
    FromBlock(#[from] FromBlockError),
    /// The L2 execution layer has no block for a lookup during the walk.
    ///
    /// An execution layer that restarted can have lost the newest blocks it had not yet written
    /// to disk, so a block it served a moment ago can be missing; a later walk starts from its
    /// new head.
    #[error("L2 execution layer has no block {0}")]
    BlockNotFound(BlockId),
    /// The L1 RPC's canonical block at the L1 genesis height is not the configured L1 genesis.
    #[error("L1 RPC serves block {1} at the L1 genesis height, expected L1 genesis {0}")]
    InvalidL1GenesisHash(B256, B256),
    /// Invalid L2 genesis hash.
    #[error("Invalid L2 genesis hash. Expected {0}, Got {1}")]
    InvalidL2GenesisHash(B256, B256),
    /// The L2 execution layer's chain has another block at the finalized head's height.
    #[error(
        "L2 execution layer has block {1} at the finalized height, expected finalized block {0}"
    )]
    MismatchedFinalizedBlock(B256, B256),
    /// The L1 RPC reported a head at or above a height, then served no block there.
    #[error("L1 block {number} is missing below the L1 head {l1_head}")]
    L1BlockMissing {
        /// The missing L1 block's number.
        number: u64,
        /// The L1 head the RPC reported.
        l1_head: u64,
    },
    /// The unsafe head would rewind across more L1 origins than a plausible L1 reorg.
    #[error(
        "Reorg deeper than a plausible L1 reorg: the L1 RPC's canonical chain would rewind the \
         unsafe head from block {previous_unsafe_head} to {unsafe_head}"
    )]
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
