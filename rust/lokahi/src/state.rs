//! What the verifier has established, and the heads the authority answers from it.
//!
//! The [`Store`] is the record: a result is written to it before the authority can answer it, so
//! a restart answers at least the heads answered before it. Only the latest verified and the
//! latest finalized results are held in memory; older ones are read from the store.

use crate::store::{Store, StoreError};
use alloy_eips::BlockNumHash;
use alloy_primitives::B256;
use kona_engine::VerifierHead;
use std::collections::BTreeMap;

/// The verified result of one timestamp: every chain's local-safe block at it, all of whose
/// executing messages are valid.
#[derive(Debug, Clone, PartialEq, Eq)]
pub(crate) struct VerifiedResult {
    /// The latest L1 block any of the chains needed to derive its block at this or an earlier
    /// timestamp: the result holds on an L1 chain that includes it.
    pub l1_inclusion: BlockNumHash,
    /// Each chain's block at the timestamp, by chain ID.
    pub heads: BTreeMap<u64, BlockNumHash>,
}

/// A block the verifier denied: its chain must replace it.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub(crate) struct DeniedRecord {
    /// The chain of the block.
    pub chain_id: u64,
    /// The denied block.
    pub block: BlockNumHash,
    /// The output root of the denied block.
    pub output_root: B256,
    /// The timestamp whose verification denied the block.
    pub timestamp: u64,
    /// The latest L1 block the decision depended on: the denial holds on an L1 chain that
    /// includes it, like a result at the timestamp would.
    pub l1: BlockNumHash,
}

/// A verified result and its timestamp.
pub(crate) type AtTimestamp = (u64, VerifiedResult);

/// The verifier's state, shared with the authority RPC.
#[derive(Debug)]
pub(crate) struct Verified {
    /// The first timestamp the verifier covers.
    activation_timestamp: u64,
    /// Each chain's L2 genesis time, by chain ID.
    genesis_times: BTreeMap<u64, u64>,
    /// The verified results, contiguous from activation, and the denied blocks.
    store: Store,
    /// The latest verified result.
    latest: Option<AtTimestamp>,
    /// The latest verified result whose L1 inclusion is on the finalized L1 chain. It only moves
    /// forward: no rewind goes below it.
    finalized: Option<AtTimestamp>,
    /// Whether verification has reached the sources' local-safe heads since the verifier
    /// started. Until it has, the store may be far behind what the nodes were told before, so
    /// the authority answers no heads.
    caught_up: bool,
}

impl Verified {
    /// The state held in `store`, for chains with the given genesis times. A store written for
    /// other chains or another activation is refused.
    pub(crate) fn open(
        store: Store,
        activation_timestamp: u64,
        genesis_times: BTreeMap<u64, u64>,
    ) -> Result<Self, StoreError> {
        let latest = store.last_result()?;
        if let Some((_, result)) = &latest {
            let first = store.first_result()?.map_or(activation_timestamp, |(ts, _)| ts);
            if first != activation_timestamp || !result.heads.keys().eq(genesis_times.keys()) {
                return Err(StoreError::OtherConfig {
                    first,
                    chains: result.heads.keys().copied().collect(),
                    activation: activation_timestamp,
                    expected: genesis_times.keys().copied().collect(),
                });
            }
        }
        Ok(Self {
            activation_timestamp,
            genesis_times,
            store,
            latest,
            finalized: None,
            caught_up: false,
        })
    }

    /// The first timestamp the verifier covers.
    pub(crate) const fn activation_timestamp(&self) -> u64 {
        self.activation_timestamp
    }

    /// The next timestamp to verify.
    pub(crate) fn next_timestamp(&self) -> u64 {
        self.latest.as_ref().map_or(self.activation_timestamp, |(ts, _)| ts + 1)
    }

    /// The latest verified result.
    pub(crate) const fn latest(&self) -> Option<&AtTimestamp> {
        self.latest.as_ref()
    }

    /// The latest finalized result.
    pub(crate) const fn finalized(&self) -> Option<&AtTimestamp> {
        self.finalized.as_ref()
    }

    /// The first timestamp a rewind may delete: the one after the finalized result.
    pub(crate) fn first_unfinalized_timestamp(&self) -> u64 {
        self.finalized.as_ref().map_or(self.activation_timestamp, |(ts, _)| ts + 1)
    }

    /// Commits the result of the next timestamp, durably, before the authority answers it.
    ///
    /// A result holds only on an L1 chain that includes every earlier result too, since its
    /// messages can depend on them, so its L1 inclusion is raised to the previous one's if lower.
    /// L1 inclusions then never decrease with the timestamp.
    pub(crate) fn commit(
        &mut self,
        timestamp: u64,
        mut result: VerifiedResult,
    ) -> Result<(), StoreError> {
        debug_assert_eq!(timestamp, self.next_timestamp(), "timestamps are verified in order");
        if let Some((_, latest)) = &self.latest &&
            latest.l1_inclusion.number > result.l1_inclusion.number
        {
            result.l1_inclusion = latest.l1_inclusion;
        }
        self.store.put_result(timestamp, &result)?;
        self.latest = Some((timestamp, result));
        Ok(())
    }

    /// Deletes the results from `timestamp` on, with the denials decided at or after it, which
    /// were decided on them, durably before the authority answers the rewound heads.
    ///
    /// `timestamp` must be above the finalized result: finality never moves back.
    pub(crate) fn rewind(&mut self, timestamp: u64) -> Result<(usize, usize), StoreError> {
        assert!(
            timestamp >= self.first_unfinalized_timestamp(),
            "rewind to {timestamp} below the finalized result {:?}",
            self.finalized.as_ref().map(|(ts, _)| ts)
        );
        let deleted = self.store.rewind(timestamp)?;
        self.latest = self.store.last_result()?;
        Ok(deleted)
    }

    /// The latest result whose L1 inclusion is at or below `l1_number`, by binary search over
    /// the stored results, whose L1 inclusions never decrease. It is final once its L1 inclusion
    /// is on the finalized L1 chain, which only its hash can tell.
    pub(crate) fn latest_included_by(
        &self,
        l1_number: u64,
    ) -> Result<Option<AtTimestamp>, StoreError> {
        let Some((latest_ts, latest)) = &self.latest else { return Ok(None) };
        if latest.l1_inclusion.number <= l1_number {
            return Ok(self.latest.clone());
        }
        // Every result at `hi` or later is above `l1_number`.
        let (mut lo, mut hi) = (self.activation_timestamp, *latest_ts);
        let mut found = None;
        while lo < hi {
            let mid = lo + (hi - lo) / 2;
            let result = self.store.result(mid)?.ok_or(StoreError::Missing(mid))?;
            if result.l1_inclusion.number <= l1_number {
                found = Some((mid, result));
                lo = mid + 1;
            } else {
                hi = mid;
            }
        }
        Ok(found)
    }

    /// Moves the finalized head to `finalized`, whose L1 inclusion the caller found on the
    /// finalized L1 chain. An older result leaves it.
    pub(crate) fn set_finalized(&mut self, finalized: AtTimestamp) {
        if self.finalized.as_ref().is_none_or(|(ts, _)| *ts < finalized.0) {
            self.finalized = Some(finalized);
        }
    }

    /// The result of `timestamp`, if verified.
    pub(crate) fn result_at(&self, timestamp: u64) -> Result<Option<VerifiedResult>, StoreError> {
        match &self.latest {
            Some((latest_ts, latest)) if *latest_ts == timestamp => Ok(Some(latest.clone())),
            Some((latest_ts, _)) if *latest_ts > timestamp => self.store.result(timestamp),
            _ => Ok(None),
        }
    }

    /// Records a denied block, durably, before the authority answers it.
    pub(crate) fn deny(&self, record: &DeniedRecord) -> Result<(), StoreError> {
        self.store.put_denied(record)
    }

    /// The denial of `chain_id`'s `block`, if it is denied.
    pub(crate) fn denial(
        &self,
        chain_id: u64,
        block: BlockNumHash,
    ) -> Result<Option<DeniedRecord>, StoreError> {
        let denied = self.store.denied(chain_id, block.number)?;
        Ok(denied
            .into_iter()
            .take_while(|record| record.block.number == block.number)
            .find(|record| record.block == block))
    }

    /// Deletes a denial, durably, before the authority stops answering it.
    pub(crate) fn undeny(&self, record: &DeniedRecord) -> Result<(), StoreError> {
        self.store.delete_denied(record)
    }

    /// The L1 inclusion a result at the next timestamp would have with the frontier inclusion
    /// `frontier`: raised to the latest result's, as [`Self::commit`] does.
    pub(crate) const fn next_l1_inclusion(&self, frontier: BlockNumHash) -> BlockNumHash {
        match &self.latest {
            Some((_, latest)) if latest.l1_inclusion.number > frontier.number => {
                latest.l1_inclusion
            }
            _ => frontier,
        }
    }

    /// The denied blocks of `chain_id` at or above `from_number`, by number.
    pub(crate) fn denied_blocks(
        &self,
        chain_id: u64,
        from_number: u64,
    ) -> Result<Vec<DeniedRecord>, StoreError> {
        self.store.denied(chain_id, from_number)
    }

    /// Records that verification reached the sources' local-safe heads.
    pub(crate) const fn set_caught_up(&mut self) {
        self.caught_up = true;
    }

    /// Whether verification has reached the sources' local-safe heads since the start.
    pub(crate) const fn is_caught_up(&self) -> bool {
        self.caught_up
    }

    /// The verified safe head of `chain_id`, or `None` for a chain the verifier does not cover.
    pub(crate) fn verified_head(&self, chain_id: u64) -> Option<VerifierHead> {
        self.head_in(chain_id, self.latest.as_ref())
    }

    /// The latest verified head of `chain_id` whose L1 inclusion is finalized, or `None` for a
    /// chain the verifier does not cover.
    pub(crate) fn finalized_head(&self, chain_id: u64) -> Option<VerifierHead> {
        self.head_in(chain_id, self.finalized.as_ref())
    }

    /// The head of `chain_id` in `verified`, like op-supernode's `LatestVerifiedL2Block` and
    /// `VerifiedBlockAtL1`.
    ///
    /// Without a verified result it is the anchor, the last timestamp before activation. That
    /// also covers a node whose local-safe head is before activation: its safe label is then its
    /// local-safe head, which the node decides from its own head, not from lokahi's sources.
    fn head_in(&self, chain_id: u64, verified: Option<&AtTimestamp>) -> Option<VerifierHead> {
        let genesis_time = *self.genesis_times.get(&chain_id)?;
        Some(match verified {
            Some((timestamp, result)) => {
                VerifierHead::Verified { block: result.heads[&chain_id], timestamp: *timestamp }
            }
            // Clamped to genesis so the node can resolve it.
            None => VerifierHead::Anchor {
                timestamp: self.activation_timestamp.saturating_sub(1).max(genesis_time),
            },
        })
    }
}

#[cfg(test)]
pub(crate) mod tests {
    use super::*;
    use std::path::Path;

    const CHAIN: u64 = 10;
    const ACTIVATION: u64 = 100;

    fn block(number: u64) -> BlockNumHash {
        BlockNumHash { number, hash: B256::with_last_byte(number as u8) }
    }

    fn result(l2: u64, l1: u64) -> VerifiedResult {
        VerifiedResult { l1_inclusion: block(l1), heads: BTreeMap::from([(CHAIN, block(l2))]) }
    }

    /// The state in the store at `dir`.
    pub(crate) fn open_state(
        dir: &Path,
        activation: u64,
        genesis_times: BTreeMap<u64, u64>,
    ) -> Verified {
        Verified::open(Store::open(dir).unwrap(), activation, genesis_times).unwrap()
    }

    fn state(dir: &Path) -> Verified {
        open_state(dir, ACTIVATION, BTreeMap::from([(CHAIN, ACTIVATION)]))
    }

    fn verified(l2: u64, timestamp: u64) -> Option<VerifierHead> {
        Some(VerifierHead::Verified { block: block(l2), timestamp })
    }

    #[test]
    fn nothing_verified_answers_the_anchor_clamped_to_genesis() {
        let dir = tempfile::tempdir().unwrap();
        let state = state(dir.path());

        assert_eq!(
            state.verified_head(CHAIN),
            Some(VerifierHead::Anchor { timestamp: ACTIVATION })
        );
        assert_eq!(
            state.finalized_head(CHAIN),
            Some(VerifierHead::Anchor { timestamp: ACTIVATION })
        );
        assert_eq!(state.verified_head(CHAIN + 1), None);
    }

    /// The latest finalized result as the verifier would find it for the finalized L1 block
    /// `l1`, had the L1 inclusion been on the finalized chain.
    fn finalize_at(state: &mut Verified, l1: u64) {
        if let Some(finalized) = state.latest_included_by(l1).unwrap() {
            state.set_finalized(finalized);
        }
    }

    #[test]
    fn verified_head_is_the_latest_result_and_finalized_the_latest_finalized_inclusion() {
        let dir = tempfile::tempdir().unwrap();
        let mut state = state(dir.path());
        state.commit(ACTIVATION, result(1, 5)).unwrap();
        state.commit(ACTIVATION + 1, result(2, 6)).unwrap();
        state.commit(ACTIVATION + 2, result(3, 8)).unwrap();
        finalize_at(&mut state, 7);

        assert_eq!(state.verified_head(CHAIN), verified(3, ACTIVATION + 2));
        assert_eq!(state.finalized_head(CHAIN), verified(2, ACTIVATION + 1));
        assert_eq!(state.next_timestamp(), ACTIVATION + 3);

        // Finality never moves back, also for an older finalized L1 block.
        finalize_at(&mut state, 5);
        assert_eq!(state.finalized_head(CHAIN), verified(2, ACTIVATION + 1));
        finalize_at(&mut state, 9);
        assert_eq!(state.finalized_head(CHAIN), verified(3, ACTIVATION + 2));
    }

    /// An L1 inclusion below an earlier one is raised to it: the later result needs the earlier
    /// one, so it is not final before it.
    #[test]
    fn l1_inclusion_never_decreases() {
        let dir = tempfile::tempdir().unwrap();
        let mut state = state(dir.path());
        state.commit(ACTIVATION, result(1, 8)).unwrap();
        state.commit(ACTIVATION + 1, result(2, 6)).unwrap();

        assert_eq!(state.result_at(ACTIVATION + 1).unwrap().unwrap().l1_inclusion, block(8));
        assert_eq!(state.latest_included_by(7).unwrap(), None);
    }

    /// A rewind deletes the results from its timestamp on and the denials decided on them, so
    /// the verified head moves back and verification resumes at that timestamp.
    #[test]
    fn rewind_moves_the_verified_head_back() {
        let dir = tempfile::tempdir().unwrap();
        let mut state = state(dir.path());
        for (i, l1) in [5, 6, 7, 7].into_iter().enumerate() {
            state.commit(ACTIVATION + i as u64, result(i as u64 + 1, l1)).unwrap();
        }
        finalize_at(&mut state, 5);
        let denial = |number: u64, timestamp: u64| DeniedRecord {
            chain_id: CHAIN,
            block: block(number),
            output_root: B256::repeat_byte(1),
            timestamp,
            l1: block(7),
        };
        state.deny(&denial(2, ACTIVATION + 1)).unwrap();
        state.deny(&denial(5, ACTIVATION + 4)).unwrap();

        assert_eq!(state.rewind(ACTIVATION + 2).unwrap(), (2, 1));

        assert_eq!(state.verified_head(CHAIN), verified(2, ACTIVATION + 1));
        assert_eq!(state.next_timestamp(), ACTIVATION + 2);
        assert_eq!(state.finalized_head(CHAIN), verified(1, ACTIVATION));
        assert_eq!(state.denied_blocks(CHAIN, 0).unwrap(), vec![denial(2, ACTIVATION + 1)]);

        // Everything after the finalized result can go.
        state.rewind(ACTIVATION + 1).unwrap();
        assert_eq!(state.verified_head(CHAIN), verified(1, ACTIVATION));
    }

    #[test]
    #[should_panic(expected = "below the finalized result")]
    fn rewind_never_goes_below_the_finalized_result() {
        let dir = tempfile::tempdir().unwrap();
        let mut state = state(dir.path());
        state.commit(ACTIVATION, result(1, 5)).unwrap();
        state.commit(ACTIVATION + 1, result(2, 6)).unwrap();
        finalize_at(&mut state, 5);

        let _ = state.rewind(ACTIVATION);
    }

    /// A reopened state answers the heads it answered before, finds the finalized head in the
    /// store, and resumes at the next timestamp.
    #[test]
    fn reopened_state_answers_the_same_heads_and_resumes() {
        let dir = tempfile::tempdir().unwrap();
        {
            let mut state = state(dir.path());
            for (i, l1) in [5, 5, 6, 8, 8, 9, 12].into_iter().enumerate() {
                state.commit(ACTIVATION + i as u64, result(i as u64 + 1, l1)).unwrap();
            }
            state
                .deny(&DeniedRecord {
                    chain_id: CHAIN,
                    block: block(9),
                    output_root: B256::repeat_byte(1),
                    timestamp: ACTIVATION + 7,
                    l1: block(12),
                })
                .unwrap();
        }

        let mut state = state(dir.path());

        assert_eq!(state.verified_head(CHAIN), verified(7, ACTIVATION + 6));
        assert_eq!(state.next_timestamp(), ACTIVATION + 7);
        assert_eq!(
            state.finalized_head(CHAIN),
            Some(VerifierHead::Anchor { timestamp: ACTIVATION })
        );
        assert_eq!(state.latest_included_by(4).unwrap(), None);
        assert_eq!(state.latest_included_by(5).unwrap(), Some((ACTIVATION + 1, result(2, 5))));
        finalize_at(&mut state, 8);
        assert_eq!(state.finalized_head(CHAIN), verified(5, ACTIVATION + 4));
        assert_eq!(state.result_at(ACTIVATION + 2).unwrap(), Some(result(3, 6)));
        assert_eq!(state.result_at(ACTIVATION + 7).unwrap(), None);
        assert_eq!(
            state.denied_blocks(CHAIN, 0).unwrap().iter().map(|r| r.block).collect::<Vec<_>>(),
            vec![block(9)]
        );
        state.commit(ACTIVATION + 7, result(8, 12)).unwrap();
        assert_eq!(state.verified_head(CHAIN), verified(8, ACTIVATION + 7));
    }

    #[rstest::rstest]
    #[case::other_activation(ACTIVATION + 1, BTreeMap::from([(CHAIN, ACTIVATION)]))]
    #[case::other_chains(ACTIVATION, BTreeMap::from([(CHAIN, ACTIVATION), (CHAIN + 1, ACTIVATION)]))]
    fn a_store_of_another_config_is_refused(
        #[case] activation: u64,
        #[case] genesis_times: BTreeMap<u64, u64>,
    ) {
        let dir = tempfile::tempdir().unwrap();
        state(dir.path()).commit(ACTIVATION, result(1, 5)).unwrap();

        let opened = Verified::open(Store::open(dir.path()).unwrap(), activation, genesis_times);

        assert!(matches!(opened, Err(StoreError::OtherConfig { .. })), "got {opened:?}");
    }

    /// A denied block is told apart from another block at its height and on another chain.
    #[test]
    fn a_denial_matches_the_chain_and_the_block_and_can_go() {
        let dir = tempfile::tempdir().unwrap();
        let state = state(dir.path());
        let denied = DeniedRecord {
            chain_id: CHAIN,
            block: block(3),
            output_root: B256::repeat_byte(3),
            timestamp: ACTIVATION + 2,
            l1: block(6),
        };
        state.deny(&denied).unwrap();
        state.deny(&DeniedRecord { block: block(5), ..denied }).unwrap();

        assert_eq!(state.denial(CHAIN, block(3)).unwrap(), Some(denied));
        assert_eq!(
            state.denial(CHAIN, BlockNumHash { number: 3, hash: B256::repeat_byte(9) }).unwrap(),
            None
        );
        assert_eq!(state.denial(CHAIN, block(4)).unwrap(), None);
        assert_eq!(state.denial(CHAIN + 1, block(3)).unwrap(), None);

        state.undeny(&denied).unwrap();
        assert_eq!(state.denial(CHAIN, block(3)).unwrap(), None);
        assert!(state.denial(CHAIN, block(5)).unwrap().is_some());
    }
}
