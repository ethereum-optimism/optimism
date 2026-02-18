//! Common sync types

use crate::{BlockInfo, L2BlockInfo};

/// The [`SyncStatus`][ss] of an Optimism Rollup Node.
///
/// The sync status is a snapshot of the current state of the node's sync process.
/// Values may not be derived yet and are zeroed out if they are not yet derived.
///
/// [ss]: https://github.com/ethereum-optimism/optimism/blob/develop/op-service/eth/sync_status.go#L5
#[derive(Clone, Debug, PartialEq, Eq)]
#[cfg_attr(feature = "serde", derive(serde::Serialize, serde::Deserialize))]
#[cfg_attr(feature = "serde", serde(rename_all = "snake_case"))]
pub struct SyncStatus {
    /// The current L1 block.
    ///
    /// This is the L1 block that the derivation process is last idled at.
    /// This may not be fully derived into L2 data yet.
    /// The safe L2 blocks were produced/included fully from the L1 chain up to _but excluding_
    /// this L1 block. If the node is synced, this matches the `head_l1`, minus the verifier
    /// confirmation distance.
    pub current_l1: BlockInfo,
    /// The current L1 finalized block.
    ///
    /// This is a legacy sync-status attribute. This is deprecated.
    /// A previous version of the L1 finalization-signal was updated only after the block was
    /// retrieved by number. This attribute just matches `finalized_l1` now.
    pub current_l1_finalized: BlockInfo,
    /// The L1 head block ref.
    ///
    /// The head is not guaranteed to build on the other L1 sync status fields,
    /// as the node may be in progress of resetting to adapt to a L1 reorg.
    pub head_l1: BlockInfo,
    /// The L1 safe head block ref.
    pub safe_l1: BlockInfo,
    /// The finalized L1 block ref.
    pub finalized_l1: BlockInfo,
    /// The unsafe L2 block ref.
    ///
    /// This is the absolute tip of the L2 chain, pointing to block data that has not been
    /// submitted to L1 yet. The sequencer is building this, and verifiers may also be ahead of
    /// the safe L2 block if they sync blocks via p2p or other offchain sources.
    /// This is considered to only be local-unsafe post-interop, see `cross_unsafe_l2` for cross-L2
    /// guarantees.
    pub unsafe_l2: L2BlockInfo,
    /// The safe L2 block ref.
    ///
    /// This points to the L2 block that was derived from the L1 chain.
    /// This point may still reorg if the L1 chain reorgs.
    /// This is considered to be cross-safe post-interop, see `local_safe_l2` to ignore cross-L2
    /// guarantees.
    pub safe_l2: L2BlockInfo,
    /// The finalized L2 block ref.
    ///
    /// This points to the L2 block that was derived fully from finalized L1 information, thus
    /// irreversible.
    pub finalized_l2: L2BlockInfo,
    /// Cross unsafe L2 block ref.
    ///
    /// This is an unsafe L2 block, that has been verified to match cross-L2 dependencies.
    /// Pre-interop every unsafe L2 block is also cross-unsafe.
    pub cross_unsafe_l2: L2BlockInfo,
    /// Local safe L2 block ref.
    ///
    /// This is an L2 block derived from L1, not yet verified to have valid cross-L2 dependencies.
    pub local_safe_l2: L2BlockInfo,
}
