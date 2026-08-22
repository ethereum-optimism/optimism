//! The atomic read behind the interop query: the local-safe head at an L2 timestamp, the L1 block
//! it was derived from, and the sync status all of it was read with.

use crate::{EngineState, EngineSyncState, state::LocalSafeHead};
use kona_genesis::RollupConfig;

/// Where a requested L2 timestamp falls relative to the local-safe head.
///
/// The live state holds the L1 origin of exactly one L2 block — the local-safe head — so only one
/// of these cases can carry a pairing. The rest say *why* there is none, which a caller has to be
/// able to tell apart: [`Self::NotLocalSafeYet`] is a "retry later", [`Self::BehindHead`] is a
/// "look it up in history", and [`Self::BeforeGenesis`] is a "there is no such block, and history
/// will not have one either". Collapsing them into one absent answer is how a consumer ends up
/// halting on a history gap that was really an out-of-range request.
// The paired variant is the whole point of the type, and boxing it would cost `Copy` on a value
// whose reason for existing is that it can be read out of the state in one move.
#[allow(clippy::large_enum_variant)]
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum LocalSafeAtTimestamp {
    /// The timestamp is before L2 genesis, so no L2 block carries it.
    ///
    /// Distinct from [`Self::BehindHead`] on purpose: both are "not in the live state", but this
    /// one is permanent and not a history lookup.
    BeforeGenesis,
    /// The timestamp is ahead of the local-safe head: the block it names has not been derived from
    /// L1 yet, and may not exist yet at all.
    ///
    /// The corresponding op-supernode answer is `LocalSafeBlockAtTimestamp`'s `ethereum.NotFound`,
    /// which callers back off and retry on.
    NotLocalSafeYet,
    /// The timestamp is the local-safe head's own, so the answer is that head together with the L1
    /// origin recorded for it.
    ///
    /// The origin may be [`crate::LocalSafeOrigin::Unpaired`]: whoever wrote the head held no L1
    /// key for it (a reset walkback, or the derivation-delegation path). That is an answer, not a
    /// missing one, and it stays visible here rather than being flattened into a defaulted
    /// [`kona_protocol::BlockInfo`] that would read as "derived from block 0".
    Head(LocalSafeHead),
    /// The timestamp is behind the local-safe head. The block it names is local-safe — safety is
    /// monotone along the canonical chain — but the live state records the L1 origin only of the
    /// head, so answering which L1 block *this* one became safe at is a history lookup.
    ///
    /// A stale origin is never substituted: the head's L1 key describes the head, and handing it
    /// out for an ancestor is exactly the silently-wrong answer this query exists to avoid.
    BehindHead,
}

impl LocalSafeAtTimestamp {
    /// Returns the local-safe head and its origin when the timestamp named the head, and [`None`]
    /// in every other case.
    ///
    /// Deliberately not an `Option<BlockInfo>` of the L1 origin: that would make an unpaired head
    /// indistinguishable from a timestamp the live state cannot answer for at all.
    pub const fn head(&self) -> Option<LocalSafeHead> {
        match self {
            Self::Head(local_safe) => Some(*local_safe),
            Self::BeforeGenesis | Self::NotLocalSafeYet | Self::BehindHead => None,
        }
    }
}

/// One consistent answer to "which L2 block was local-safe at this timestamp, which L1 block was
/// it derived from, and where was the chain when you looked?".
///
/// This is the in-process equivalent of op-supernode's `ChainContainer.OptimisticAt`, which reaches
/// the same answer with two reads — `LocalSafeBlockAtTimestamp`, then `safeDBAtL2`, each taking its
/// own sync-status sample — and carries a TOCTOU gap between them: the local-safe head can advance
/// or be reset in between, so the L2 block can come from one view of the chain and the L1 key from
/// another. Here every field is computed from a single [`EngineState`] value, so the pairing, the
/// timestamp verdict and the sync status cannot disagree with each other.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct LocalSafeSnapshot {
    /// The L2 timestamp this snapshot answers for.
    pub timestamp: u64,
    /// Where [`Self::timestamp`] fell relative to the local-safe head, and the pairing when it
    /// named that head.
    pub local_safe_at: LocalSafeAtTimestamp,
    /// The sync state the verdict was read from: the unsafe, local-safe, cross-safe and finalized
    /// heads as they stood at that one read.
    ///
    /// Carried in the same answer because a caller deciding cross-safety needs the verdict and the
    /// safety levels to describe the same instant. Only the L2 half is here — the L1 sync fields
    /// live in the L1 watcher, a different actor, and cannot be sampled atomically with this.
    pub sync_state: EngineSyncState,
    /// Whether the execution layer had finished syncing at that read.
    ///
    /// Until it has, the heads describe a node that is still catching up, and a consumer that
    /// treats them as authoritative is reading a chain that is merely the part this node has seen.
    pub el_sync_finished: bool,
}

impl EngineState {
    /// Answers the local-safe-at-timestamp query from this one state value.
    ///
    /// Every field of the returned [`LocalSafeSnapshot`] is derived from `self`, so a caller that
    /// takes one copy of the state — as [`crate::EngineQueries::handle`] does, with a single
    /// `borrow` of the state watch — gets an answer with no window inside it. The head and its L1
    /// origin come out of [`EngineSyncState::local_safe`], which is itself a paired read.
    ///
    /// The comparison is on timestamps rather than on a block number computed from the requested
    /// timestamp. op-node's `TargetBlockNumber` floors an unaligned timestamp onto the preceding
    /// block, which for a timestamp just past the head would return the head as the answer; here
    /// only a timestamp that *is* the head's is answered with the head's pairing, and anything
    /// else is reported as ahead or behind. Interop's timestamps are block-aligned across the
    /// dependency set, so the two agree wherever the answer matters, and where they differ this
    /// one declines to attach the head's L1 key to a block that is not the head.
    pub const fn local_safe_snapshot_at(
        &self,
        rollup: &RollupConfig,
        timestamp: u64,
    ) -> LocalSafeSnapshot {
        let local_safe = self.sync_state.local_safe();
        let head_timestamp = local_safe.head.block_info.timestamp;

        let local_safe_at = if timestamp < rollup.genesis.l2_time {
            LocalSafeAtTimestamp::BeforeGenesis
        } else if timestamp > head_timestamp {
            LocalSafeAtTimestamp::NotLocalSafeYet
        } else if timestamp < head_timestamp {
            LocalSafeAtTimestamp::BehindHead
        } else {
            LocalSafeAtTimestamp::Head(local_safe)
        };

        LocalSafeSnapshot {
            timestamp,
            local_safe_at,
            sync_state: self.sync_state,
            el_sync_finished: self.el_sync_finished,
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::{LocalSafeOrigin, test_utils::TestEngineStateBuilder};
    use kona_genesis::{ChainGenesis, RollupConfig};
    use kona_protocol::{BlockInfo, L2BlockInfo};

    /// L2 genesis at timestamp 90, so a request below it is out of range rather than history.
    fn rollup() -> RollupConfig {
        RollupConfig {
            block_time: 2,
            genesis: ChainGenesis { l2_time: 90, ..Default::default() },
            ..Default::default()
        }
    }

    fn l1(number: u64) -> BlockInfo {
        BlockInfo { number, ..Default::default() }
    }

    fn l2(number: u64, timestamp: u64) -> L2BlockInfo {
        L2BlockInfo {
            block_info: BlockInfo { number, timestamp, ..Default::default() },
            ..Default::default()
        }
    }

    /// A local-safe head at timestamp 100 derived from L1 block 5, with the other heads spread out
    /// so the sync status carried alongside is distinguishable.
    fn state() -> EngineState {
        TestEngineStateBuilder::new()
            .with_unsafe_head(l2(12, 104))
            .with_local_safe_head(l2(10, 100))
            .with_local_safe_origin(LocalSafeOrigin::DerivedFrom(l1(5)))
            .with_cross_safe_head(l2(9, 98))
            .with_finalized_head(l2(8, 96))
            .build()
    }

    #[test]
    fn the_head_timestamp_is_answered_with_the_pairing() {
        let snapshot = state().local_safe_snapshot_at(&rollup(), 100);

        assert_eq!(snapshot.timestamp, 100);
        assert_eq!(
            snapshot.local_safe_at,
            LocalSafeAtTimestamp::Head(LocalSafeHead::derived_from(l2(10, 100), l1(5)))
        );
        assert_eq!(snapshot.local_safe_at.head().unwrap().derived_from_l1(), Some(l1(5)));
    }

    /// The whole point of the pairing: an unpaired head is an answer, and it is not the same answer
    /// as "derived from block 0".
    #[test]
    fn an_unpaired_head_is_answered_as_unpaired() {
        let state = TestEngineStateBuilder::new()
            .with_unsafe_head(l2(10, 100))
            .with_local_safe_head(l2(10, 100))
            .with_local_safe_origin(LocalSafeOrigin::Unpaired)
            .with_finalized_head(l2(8, 96))
            .build();

        let snapshot = state.local_safe_snapshot_at(&rollup(), 100);
        let head = snapshot.local_safe_at.head().expect("the timestamp named the head");

        assert_eq!(head.origin, LocalSafeOrigin::Unpaired);
        assert_eq!(head.derived_from_l1(), None);
        assert_ne!(
            snapshot.local_safe_at,
            LocalSafeAtTimestamp::Head(LocalSafeHead::derived_from(
                l2(10, 100),
                BlockInfo::default()
            ))
        );
    }

    /// Ahead of the head is a retry, and it must not borrow the head's L1 key.
    #[test]
    fn a_timestamp_ahead_of_the_head_is_not_local_safe_yet() {
        let snapshot = state().local_safe_snapshot_at(&rollup(), 102);

        assert_eq!(snapshot.local_safe_at, LocalSafeAtTimestamp::NotLocalSafeYet);
        assert_eq!(snapshot.local_safe_at.head(), None);
    }

    /// Behind the head the block is local-safe, but its origin is history rather than the head's
    /// origin.
    #[test]
    fn a_timestamp_behind_the_head_is_a_history_lookup() {
        let snapshot = state().local_safe_snapshot_at(&rollup(), 98);

        assert_eq!(snapshot.local_safe_at, LocalSafeAtTimestamp::BehindHead);
        assert_eq!(snapshot.local_safe_at.head(), None);
    }

    /// Below L2 genesis there is no block at all, which is not the same as one whose origin has to
    /// be looked up: a consumer that conflated the two would report a history gap for a request
    /// that was simply out of range.
    #[test]
    fn a_timestamp_below_genesis_is_not_a_history_lookup() {
        let snapshot = state().local_safe_snapshot_at(&rollup(), 80);

        assert_eq!(snapshot.local_safe_at, LocalSafeAtTimestamp::BeforeGenesis);
        assert_ne!(snapshot.local_safe_at, LocalSafeAtTimestamp::BehindHead);
    }

    /// The verdict and the safety levels come out of the same state value.
    #[test]
    fn the_snapshot_carries_the_sync_status_it_was_read_with() {
        let state = state();
        let snapshot = state.local_safe_snapshot_at(&rollup(), 100);

        assert_eq!(snapshot.sync_state, state.sync_state);
        assert_eq!(snapshot.sync_state.unsafe_head(), l2(12, 104));
        assert_eq!(snapshot.sync_state.local_safe_head(), l2(10, 100));
        assert_eq!(snapshot.sync_state.cross_safe_head(), l2(9, 98));
        assert_eq!(snapshot.sync_state.finalized_head(), l2(8, 96));
        assert_eq!(snapshot.sync_state.local_safe(), snapshot.local_safe_at.head().unwrap());
        assert!(snapshot.el_sync_finished);
    }

    /// A node that has not finished EL sync says so in the same answer, rather than leaving a
    /// consumer to ask separately and pair the two itself.
    #[test]
    fn the_snapshot_reports_an_unsynced_execution_layer() {
        let state = TestEngineStateBuilder::new()
            .with_unsafe_head(l2(10, 100))
            .with_local_safe_head(l2(10, 100))
            .with_finalized_head(l2(8, 96))
            .with_el_sync_finished(false)
            .build();

        assert!(!state.local_safe_snapshot_at(&rollup(), 100).el_sync_finished);
    }
}
