use alloy_eips::BlockNumHash;
use kona_supervisor_types::BlockSeal;

/// Commands for managing a node in the supervisor.
/// These commands are sent to the managed node actor to perform various operations.
#[derive(Debug, PartialEq, Eq)]
pub enum ManagedNodeCommand {
    /// Updates the finalized block in the managed node.
    UpdateFinalized {
        /// [`BlockNumHash`] of the finalized block.
        block_id: BlockNumHash,
    },

    /// Updates the cross-unsafe block in the managed node.
    UpdateCrossUnsafe {
        /// [`BlockNumHash`] of the cross-unsafe block.
        block_id: BlockNumHash,
    },

    /// Updates the cross-safe block in the managed node.
    UpdateCrossSafe {
        /// [`BlockNumHash`] of the source block.
        source_block_id: BlockNumHash,
        /// [`BlockNumHash`] of the derived block.
        derived_block_id: BlockNumHash,
    },

    /// Resets the managed node.
    Reset {},

    /// Asks managed node to invalidate the block.
    InvalidateBlock {
        /// [`BlockSeal`] of the block to invalidate.
        seal: BlockSeal,
    },
}
