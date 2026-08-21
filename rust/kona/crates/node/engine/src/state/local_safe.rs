//! The local-safe head and the L1 block it was derived from.

use kona_protocol::{BlockInfo, L2BlockInfo};
use serde::{Deserialize, Serialize};

/// The L1 block a local-safe head was derived from, or the explicit absence of one.
///
/// Interop asks the node "which L1 block was the chain safe at, at this L2 timestamp?" and uses the
/// answer to decide cross-safety. Answering with an L1 block the head was *not* derived from is
/// worse than not answering, so the two states are kept distinct in the type: a writer that holds
/// no L1 key records [`Self::Unpaired`] rather than leaving the previous key in place, and a reader
/// cannot mistake that for a real origin the way an [`Option<BlockInfo>`] flattened with
/// `unwrap_or_default` would (a defaulted [`BlockInfo`] is block 0 of an unnamed chain, which looks
/// like an answer).
///
/// Three writers are legitimately unpaired:
///
/// - `ConsolidateTask::reconcile_to_local_safe_head`, on the derivation-delegation path, where the
///   input is a bare [`L2BlockInfo`] injected by the delegating derivation actor with no attributes
///   and therefore no L1 origin attached.
/// - [`Engine::reset`] and [`Engine::reset_to`], which install a walkback point found by traversing
///   the L2 chain rather than one produced by derivation. A reset also *invalidates* whatever
///   pairing was recorded before it, which is why it has to overwrite rather than leave the field
///   alone.
/// - Any derived attributes whose own `derived_from` is [`None`], since
///   [`OpAttributesWithParent::derived_from`] is itself optional.
///
/// [`Engine::reset`]: crate::Engine::reset
/// [`Engine::reset_to`]: crate::Engine::reset_to
/// [`OpAttributesWithParent::derived_from`]: kona_protocol::OpAttributesWithParent::derived_from
#[derive(Debug, Default, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
pub enum LocalSafeOrigin {
    /// The writer of this local-safe head held no L1 origin for it.
    ///
    /// This is a real answer — "this head has no L1 key" — not a missing one, and consumers are
    /// expected to represent it rather than substituting a stale or defaulted block.
    #[default]
    Unpaired,
    /// The local-safe head was derived from this L1 block.
    DerivedFrom(BlockInfo),
}

impl LocalSafeOrigin {
    /// Returns the L1 block this head was derived from, or [`None`] when unpaired.
    pub const fn derived_from(&self) -> Option<BlockInfo> {
        match self {
            Self::Unpaired => None,
            Self::DerivedFrom(l1) => Some(*l1),
        }
    }

    /// Returns whether an L1 origin is recorded.
    pub const fn is_paired(&self) -> bool {
        matches!(self, Self::DerivedFrom(_))
    }
}

impl From<Option<BlockInfo>> for LocalSafeOrigin {
    fn from(derived_from: Option<BlockInfo>) -> Self {
        derived_from.map_or(Self::Unpaired, Self::DerivedFrom)
    }
}

/// A local-safe head paired with the L1 origin it was derived from.
///
/// This is both the write and the read shape of the pairing. As the `local_safe_head` field of an
/// [`EngineSyncStateUpdate`] it makes the origin unforgettable: a writer cannot move the local-safe
/// head without saying where it came from, so the pairing cannot silently go stale behind a head
/// that moved. As the return of [`EngineSyncState::local_safe`] it is the atomic read — head and
/// origin come out of one snapshot together, with no window in which the two halves disagree.
///
/// [`EngineSyncStateUpdate`]: crate::EngineSyncStateUpdate
/// [`EngineSyncState::local_safe`]: crate::EngineSyncState::local_safe
#[derive(Debug, Default, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
pub struct LocalSafeHead {
    /// The local-safe L2 block.
    pub head: L2BlockInfo,
    /// The L1 block it was derived from, if its writer knew one.
    pub origin: LocalSafeOrigin,
}

impl LocalSafeHead {
    /// Creates a pairing of `head` with `origin`.
    pub const fn new(head: L2BlockInfo, origin: LocalSafeOrigin) -> Self {
        Self { head, origin }
    }

    /// Creates a pairing of `head` with the L1 block `l1` it was derived from.
    pub const fn derived_from(head: L2BlockInfo, l1: BlockInfo) -> Self {
        Self::new(head, LocalSafeOrigin::DerivedFrom(l1))
    }

    /// Creates `head` with no L1 origin, for a writer that holds none.
    ///
    /// Spelled out rather than defaulted so that an unpaired write is a deliberate statement at the
    /// call site instead of an omission.
    pub const fn unpaired(head: L2BlockInfo) -> Self {
        Self::new(head, LocalSafeOrigin::Unpaired)
    }

    /// Returns the L1 block the head was derived from, or [`None`] when unpaired.
    pub const fn derived_from_l1(&self) -> Option<BlockInfo> {
        self.origin.derived_from()
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use kona_protocol::BlockInfo;

    fn l1(number: u64) -> BlockInfo {
        BlockInfo { number, ..Default::default() }
    }

    fn l2(number: u64) -> L2BlockInfo {
        L2BlockInfo { block_info: BlockInfo { number, ..Default::default() }, ..Default::default() }
    }

    #[test]
    fn unpaired_is_the_default_origin() {
        assert_eq!(LocalSafeOrigin::default(), LocalSafeOrigin::Unpaired);
        assert_eq!(LocalSafeHead::default().origin, LocalSafeOrigin::Unpaired);
    }

    #[test]
    fn unpaired_does_not_read_as_an_origin() {
        let unpaired = LocalSafeHead::unpaired(l2(7));

        assert!(!unpaired.origin.is_paired());
        assert_eq!(unpaired.derived_from_l1(), None);
        // The absent case is distinguishable from a real origin that happens to be block 0, which
        // is what a defaulted `BlockInfo` would look like.
        assert_ne!(unpaired.origin, LocalSafeOrigin::DerivedFrom(BlockInfo::default()));
    }

    #[test]
    fn derived_from_carries_the_l1_block() {
        let paired = LocalSafeHead::derived_from(l2(7), l1(3));

        assert!(paired.origin.is_paired());
        assert_eq!(paired.derived_from_l1(), Some(l1(3)));
        assert_eq!(paired.head, l2(7));
    }

    #[test]
    fn origin_converts_from_optional_attributes_field() {
        assert_eq!(LocalSafeOrigin::from(Some(l1(3))), LocalSafeOrigin::DerivedFrom(l1(3)));
        assert_eq!(LocalSafeOrigin::from(None), LocalSafeOrigin::Unpaired);
    }
}
