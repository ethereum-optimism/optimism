//! Conservative-minima aggregation, and composing a super root from what the chains report.
//!
//! This is the arithmetic half of the two supernode RPCs, kept apart from the reads so it can be
//! tested against op-supernode's rules directly. Those rules are in
//! `op-supernode/supernode/activity/internal/syncstatus/syncstatus.go` (the aggregation) and
//! `op-supernode/supernode/activity/superroot/superroot.go` (the composition), and each function
//! below names the one it mirrors.
//!
//! Everything here is a *minimum*. A supernode's answer is a statement about the whole chain set,
//! so the safety level it publishes is the safety level of its least advanced chain — an aggregate
//! computed any other way would tell a proposer that data is safe because most of the set has it.
//! The one exception is `verified_required_l1` on the pre-interop handoff branch, which is a
//! *maximum*: it is the L1 block from which every chain's block can be derived, so it is the
//! latest of them, not the earliest.

use crate::query::{
    chain::OptimisticOutput,
    error::QueryError,
    wire::{
        WireBlockId, WireChainId, WireChainIdAndOutput, WireOutputV0, WireOutputWithRequiredL1,
        WireSuperRootData, WireSuperV1,
    },
};
use alloy_eips::BlockNumHash;
use alloy_primitives::ChainId;
use kona_interop::{OutputRootWithChain, SuperRoot};
use kona_protocol::SyncStatus;
use std::collections::BTreeMap;

/// The chain set's sync status, reduced to what a supernode publishes about the set as a whole.
///
/// op-supernode: `syncstatus.Aggregate`.
#[derive(Debug, Clone, PartialEq, Eq)]
pub(crate) struct Aggregate {
    /// Each chain's own sync status, unmodified.
    pub(crate) chains: BTreeMap<ChainId, SyncStatus>,
    /// The L1 block the least advanced L1 processor in the supernode is on.
    ///
    /// Every L1 block strictly *below* this one has been fully processed by every chain; this one
    /// itself may not be. A consumer gating on "L1 up to X is processed" must require
    /// `current_l1.number > X`, which is the contract the Go type documents.
    pub(crate) current_l1: BlockNumHash,
    /// The highest L2 timestamp that is cross-safe across the whole set.
    pub(crate) safe_timestamp: u64,
    /// The highest L2 timestamp that is local-safe across the whole set.
    pub(crate) local_safe_timestamp: u64,
    /// The highest L2 timestamp that is finalized across the whole set.
    pub(crate) finalized_timestamp: u64,
}

impl Aggregate {
    /// Reduces the per-chain statuses to the set's.
    ///
    /// A chain that has not initialized a head reports timestamp zero, which floors the aggregate
    /// to zero. That is deliberate, and op-supernode says so: the aggregate describes the whole
    /// set, and a set containing a chain that has derived nothing is a set at zero.
    ///
    /// The minimum of `current_l1` is taken over the chains' values as they are. op-supernode's
    /// loop additionally re-seeds the minimum whenever it currently holds the zero block, which
    /// makes a set where one chain reports zero and another does not depend on Go's map iteration
    /// order — the zero can be overwritten by a later chain or not. This takes the plain minimum,
    /// which is that comparison's intent and is the conservative side of it: a chain that has
    /// processed no L1 holds the aggregate at zero rather than being silently dropped from it.
    pub(crate) fn of(chains: BTreeMap<ChainId, SyncStatus>) -> Self {
        let mut current_l1: Option<BlockNumHash> = None;
        let (mut safe, mut local_safe, mut finalized) = (None, None, None);

        for status in chains.values() {
            let chain_l1 = status.current_l1.id();
            current_l1 =
                Some(current_l1.map_or(chain_l1, |current| Self::lower(current, chain_l1)));
            safe = Some(Self::min_or(safe, status.safe_l2.block_info.timestamp));
            local_safe = Some(Self::min_or(local_safe, status.local_safe_l2.block_info.timestamp));
            finalized = Some(Self::min_or(finalized, status.finalized_l2.block_info.timestamp));
        }

        Self {
            chains,
            current_l1: current_l1.unwrap_or_default(),
            safe_timestamp: safe.unwrap_or_default(),
            local_safe_timestamp: local_safe.unwrap_or_default(),
            finalized_timestamp: finalized.unwrap_or_default(),
        }
    }

    /// Folds a verifier's L1 progress into `current_l1`.
    ///
    /// op-supernode does this inside the per-chain loop, through `ChainContainer.VerifierCurrentL1`
    /// — once per chain, with the same process-wide verifier, so folding it once here is the same
    /// minimum. It is only folded in when interop is configured, which is exactly when a verifier
    /// is registered on the chains over there.
    #[must_use]
    pub(crate) const fn with_verifier_l1(mut self, verifier_l1: BlockNumHash) -> Self {
        self.current_l1 = Self::lower(self.current_l1, verifier_l1);
        self
    }

    /// The chain set, ascending — the ordering op-supernode sorts `chain_ids` into.
    pub(crate) fn chain_ids(&self) -> Vec<WireChainId> {
        self.chains.keys().copied().map(WireChainId).collect()
    }

    /// How many chains this supernode hosts.
    ///
    /// The aggregate's own chain set is the hosted set: it has one entry per chain the query API
    /// was composed over. Reading it from here rather than assembling a second list means the
    /// `chain_ids` a consumer sees and the count the super root is checked against cannot differ.
    pub(crate) fn hosted(&self) -> usize {
        self.chains.len()
    }

    /// Returns the lower of `current` and `candidate`, comparing block numbers.
    ///
    /// Numbers only, as op-supernode compares: two chains reporting the same L1 height with
    /// different hashes are in the middle of an L1 reorg, and picking either is as good as the
    /// other. `current` wins a tie, which with a chain-id-ordered iteration makes the answer
    /// deterministic rather than dependent on map order.
    ///
    /// Also used to fold the verifier's snapshot L1 into a superroot response, which op-supernode
    /// does with the same strict comparison.
    pub(crate) const fn lower(current: BlockNumHash, candidate: BlockNumHash) -> BlockNumHash {
        if current.number <= candidate.number { current } else { candidate }
    }

    /// Returns the lower of `current` and `candidate`, seeding from `candidate` when unset.
    const fn min_or(current: Option<u64>, candidate: u64) -> u64 {
        match current {
            Some(current) if current < candidate => current,
            _ => candidate,
        }
    }

    /// Checks that a verified frontier covers exactly the chains this supernode hosts.
    ///
    /// op-supernode: the two guards at the top of `composeVerifiedData`. Either direction of
    /// mismatch means the frontier and the boot configuration disagree about the dependency set,
    /// and a super root computed over either one would disagree with peers running the full set.
    pub(crate) fn require_same_chain_set(
        &self,
        timestamp: u64,
        verified: &BTreeMap<ChainId, BlockNumHash>,
    ) -> Result<(), QueryError> {
        if verified.len() != self.hosted() {
            return Err(QueryError::ChainSetMismatch {
                timestamp,
                verified: verified.len(),
                hosted: self.hosted(),
            });
        }
        for &chain_id in self.chains.keys() {
            if !verified.contains_key(&chain_id) {
                return Err(QueryError::ChainNotVerified { timestamp, chain_id });
            }
        }
        Ok(())
    }
}

impl OptimisticOutput {
    /// Renders a whole optimistic branch for the wire.
    pub(crate) fn branch(
        optimistic: &BTreeMap<ChainId, Self>,
    ) -> BTreeMap<WireChainId, WireOutputWithRequiredL1> {
        optimistic
            .iter()
            .map(|(&chain_id, entry)| {
                (
                    WireChainId(chain_id),
                    WireOutputWithRequiredL1 {
                        output: WireOutputV0::from(entry.output),
                        output_root: entry.output.hash(),
                        required_l1: entry.required_l1.into(),
                    },
                )
            })
            .collect()
    }
}

impl WireSuperRootData {
    /// Composes the super root at `timestamp` from the optimistic outputs.
    ///
    /// op-supernode: `composeHandoffDataFromOptimistic`. Used where the optimistic outputs *are*
    /// the canonical ones — before interop activates, and below the first timestamp the verifier
    /// covers, where the safe-head handoff guarantees it. Returns [`None`] when any chain is
    /// missing from the branch, because a super root over a subset of the set is not this set's
    /// super root.
    ///
    /// `verified_required_l1` is the *highest* required L1 of the chains, not the lowest: it
    /// answers "from which L1 block can all of this be derived", and that is the last one any
    /// chain needs.
    pub(crate) fn from_handoff(
        timestamp: u64,
        aggregate: &Aggregate,
        optimistic: &BTreeMap<ChainId, OptimisticOutput>,
    ) -> Option<Self> {
        if optimistic.len() != aggregate.hosted() {
            return None;
        }
        let mut required_l1 = BlockNumHash::default();
        let roots = optimistic
            .iter()
            .map(|(&chain_id, entry)| {
                if entry.required_l1.number > required_l1.number {
                    required_l1 = entry.required_l1;
                }
                OutputRootWithChain::new(chain_id, entry.output.hash())
            })
            .collect();
        Some(Self::new(timestamp, required_l1, roots))
    }

    /// Composes the super root at `timestamp` from per-chain output roots and the L1 block the
    /// verified frontier was derived from.
    ///
    /// op-supernode: the tail of `composeVerifiedData`.
    pub(crate) fn verified(
        timestamp: u64,
        l1_inclusion: BlockNumHash,
        roots: Vec<OutputRootWithChain>,
    ) -> Self {
        Self::new(timestamp, l1_inclusion, roots)
    }

    /// Builds the response's `data` section from the roots that make up the super root.
    ///
    /// [`SuperRoot::new`] sorts by chain id and [`SuperRoot::hash`] is the same
    /// encoding-then-keccak that `eth.SuperV1.Marshal` feeds `eth.SuperRoot`, so the commitment
    /// published here is bit-identical to op-supernode's over the same inputs. Nothing in this
    /// file computes a hash of its own.
    fn new(timestamp: u64, required_l1: BlockNumHash, roots: Vec<OutputRootWithChain>) -> Self {
        let super_root = SuperRoot::new(timestamp, roots);
        Self {
            verified_required_l1: WireBlockId::from(required_l1),
            super_v1: WireSuperV1 {
                timestamp: super_root.timestamp,
                chains: super_root
                    .output_roots
                    .iter()
                    .map(|root| WireChainIdAndOutput {
                        chain_id: WireChainId(root.chain_id),
                        output: root.output_root,
                    })
                    .collect(),
            },
            super_root: super_root.hash(),
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use alloy_primitives::{B256, b256};
    use kona_protocol::{BlockInfo, L2BlockInfo};

    fn l1(number: u64) -> BlockInfo {
        BlockInfo { hash: B256::with_last_byte(number as u8), number, ..Default::default() }
    }

    fn l2(timestamp: u64) -> L2BlockInfo {
        L2BlockInfo {
            block_info: BlockInfo { timestamp, ..Default::default() },
            ..Default::default()
        }
    }

    fn status(current_l1: u64, safe: u64, local_safe: u64, finalized: u64) -> SyncStatus {
        SyncStatus {
            current_l1: l1(current_l1),
            current_l1_finalized: l1(current_l1),
            head_l1: l1(current_l1),
            safe_l1: l1(current_l1),
            finalized_l1: l1(current_l1),
            unsafe_l2: l2(local_safe),
            safe_l2: l2(safe),
            finalized_l2: l2(finalized),
            local_safe_l2: l2(local_safe),
        }
    }

    fn statuses(entries: [(ChainId, SyncStatus); 2]) -> BTreeMap<ChainId, SyncStatus> {
        entries.into_iter().collect()
    }

    /// The whole point of the aggregate: it describes the least advanced chain, not the average
    /// or the best.
    #[test]
    fn every_field_is_the_minimum_across_the_set() {
        let aggregate = Aggregate::of(statuses([
            (901, status(9, 300, 302, 200)),
            (902, status(7, 280, 290, 210)),
        ]));

        assert_eq!(aggregate.current_l1, l1(7).id());
        assert_eq!(aggregate.safe_timestamp, 280);
        assert_eq!(aggregate.local_safe_timestamp, 290);
        assert_eq!(aggregate.finalized_timestamp, 200);
    }

    /// A chain that has derived nothing holds the whole set at zero. Reporting the other chain's
    /// progress instead would tell a proposer the set is safe somewhere it is not.
    #[test]
    fn an_uninitialized_chain_floors_the_set() {
        let aggregate =
            Aggregate::of(statuses([(901, status(9, 300, 302, 200)), (902, status(0, 0, 0, 0))]));

        assert_eq!(aggregate.current_l1, BlockNumHash::default());
        assert_eq!(aggregate.safe_timestamp, 0);
        assert_eq!(aggregate.local_safe_timestamp, 0);
        assert_eq!(aggregate.finalized_timestamp, 0);
    }

    /// The verifier is one more L1 processor in the set, and a verifier that has not advanced is
    /// at block zero — which is what keeps a consumer gating on L1 progress waiting for it rather
    /// than acting on the chains' progress alone.
    #[test]
    fn a_verifier_that_has_not_advanced_holds_the_l1_at_zero() {
        let aggregate = Aggregate::of(statuses([
            (901, status(9, 300, 302, 200)),
            (902, status(8, 300, 302, 200)),
        ]))
        .with_verifier_l1(BlockNumHash::default());

        assert_eq!(aggregate.current_l1, BlockNumHash::default());
    }

    /// A verifier ahead of the chains does not raise the aggregate: the minimum is still the
    /// chains'.
    #[test]
    fn a_verifier_ahead_of_the_chains_does_not_raise_the_aggregate() {
        let aggregate = Aggregate::of(statuses([
            (901, status(9, 300, 302, 200)),
            (902, status(8, 300, 302, 200)),
        ]))
        .with_verifier_l1(l1(20).id());

        assert_eq!(aggregate.current_l1, l1(8).id());
    }

    /// `chain_ids` is ascending, which is what op-supernode sorts it into and what a consumer
    /// diffing two responses relies on.
    #[test]
    fn the_chain_ids_are_ascending() {
        let aggregate =
            Aggregate::of(statuses([(902, status(1, 1, 1, 1)), (901, status(1, 1, 1, 1))]));

        assert_eq!(aggregate.chain_ids(), vec![WireChainId(901), WireChainId(902)]);
    }

    /// A branch missing a chain is not this set's super root, and saying so as an absent `data` is
    /// the difference between a consumer waiting and a consumer committing to a wrong claim.
    #[test]
    fn a_partial_optimistic_branch_yields_no_super_root() {
        let mut optimistic = BTreeMap::new();
        optimistic.insert(901, output(1, 5));

        assert!(WireSuperRootData::from_handoff(1_000, &hosting(2), &optimistic).is_none());
        assert!(WireSuperRootData::from_handoff(1_000, &hosting(1), &optimistic).is_some());
    }

    /// The handoff branch's required L1 is the *latest* of the chains': it answers "from which L1
    /// block is all of this derivable", so a lower one would be a claim the data cannot support.
    #[test]
    fn the_handoff_required_l1_is_the_highest_of_the_chains() {
        let mut optimistic = BTreeMap::new();
        optimistic.insert(901, output(1, 5));
        optimistic.insert(902, output(2, 9));

        let data = WireSuperRootData::from_handoff(1_000, &hosting(2), &optimistic)
            .expect("both chains present");
        assert_eq!(data.verified_required_l1.number, 9);
    }

    /// The super root is [`SuperRoot`]'s hash of the sorted roots, which is the same image
    /// `eth.SuperV1.Marshal` produces. Pinned against a literal so a change to either side of the
    /// wire shows up here.
    #[test]
    fn the_super_root_is_the_keccak_of_the_encoded_preimage() {
        let mut optimistic = BTreeMap::new();
        optimistic.insert(902, output(2, 9));
        optimistic.insert(901, output(1, 5));

        let data = WireSuperRootData::from_handoff(1_000, &hosting(2), &optimistic)
            .expect("both chains present");
        assert_eq!(data.super_v1.timestamp, 1_000);
        assert_eq!(
            data.super_v1.chains.iter().map(|c| c.chain_id.0).collect::<Vec<_>>(),
            vec![901, 902],
            "the preimage is sorted by chain id"
        );

        let expected = SuperRoot::new(
            1_000,
            vec![
                OutputRootWithChain::new(901, output(1, 5).output.hash()),
                OutputRootWithChain::new(902, output(2, 9).output.hash()),
            ],
        );
        assert_eq!(data.super_root, expected.hash());
    }

    /// A frontier that names a different set than the supernode hosts is refused rather than
    /// served over the chains they share.
    #[test]
    fn a_frontier_over_a_different_chain_set_is_refused() {
        let aggregate =
            Aggregate::of(statuses([(901, status(1, 1, 1, 1)), (902, status(1, 1, 1, 1))]));
        let mut verified = BTreeMap::new();
        verified.insert(901, BlockNumHash::default());
        assert!(matches!(
            aggregate.require_same_chain_set(1_000, &verified),
            Err(QueryError::ChainSetMismatch { verified: 1, hosted: 2, .. })
        ));

        verified.insert(903, BlockNumHash::default());
        assert!(matches!(
            aggregate.require_same_chain_set(1_000, &verified),
            Err(QueryError::ChainNotVerified { chain_id: 902, .. })
        ));

        verified.remove(&903);
        verified.insert(902, BlockNumHash::default());
        assert!(aggregate.require_same_chain_set(1_000, &verified).is_ok());
    }

    /// An aggregate over `count` chains, for the handoff branch's "every chain present" check.
    fn hosting(count: u64) -> Aggregate {
        Aggregate::of((901..901 + count).map(|id| (id, status(1, 1, 1, 1))).collect())
    }

    /// A hand-built optimistic entry, distinguishable by its two seeds.
    fn output(seed: u8, required_l1: u64) -> OptimisticOutput {
        OptimisticOutput {
            output: kona_protocol::OutputRoot::from_parts(
                B256::with_last_byte(seed),
                b256!("00000000000000000000000000000000000000000000000000000000000000ff"),
                B256::with_last_byte(seed),
            ),
            required_l1: BlockNumHash {
                number: required_l1,
                hash: B256::with_last_byte(required_l1 as u8),
            },
        }
    }
}
