//! Contains the `ControlEvent`.

use alloy_primitives::B256;
use kona_protocol::BlockInfo;

/// Control Event
///
/// The `ControlEvent` is an action performed by the supervisor
/// on the L2 consensus node, in this case the `kona-node`.
#[derive(Debug, Clone, PartialEq, Eq)]
#[allow(clippy::large_enum_variant)]
pub enum ControlEvent {
    /// Invalidates a specified block.
    ///
    /// Based on some dependency or L1 changes, the supervisor
    /// can instruct the L2 to invalidate a specific block.
    InvalidateBlock(B256),

    /// The supervisor sends the next L1 block to the node.
    /// Ideally sent after the node emits exhausted-l1.
    ProviderL1(BlockInfo),

    /// Forces a reset to a specific local-unsafe/local-safe/finalized
    /// starting point only if the blocks did exist. Resets may override
    /// local-unsafe, to reset the very end of the chain. Resets may
    /// override local-safe, since post-interop we need the local-safe
    /// block derivation to continue.
    Reset {
        /// The local-unsafe block to reset to.
        local_unsafe: Option<BlockInfo>,
        /// The cross-unsafe block to reset to.
        cross_unsafe: Option<BlockInfo>,
        /// The local-safe block to reset to.
        local_safe: Option<BlockInfo>,
        /// The cross-safe block to reset to.
        cross_safe: Option<BlockInfo>,
        /// The finalized block to reset to.
        finalized: Option<BlockInfo>,
    },

    /// Signal that a block can be promoted to cross-safe.
    UpdateCrossSafe(BlockInfo),

    /// Signal that a block can be promoted to cross-unsafe.
    UpdateCrossUnsafe(BlockInfo),

    /// Signal that a block can be marked as finalized.
    UpdateFinalized(BlockInfo),
}
