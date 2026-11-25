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
}

/// A verified result and its timestamp.
type AtTimestamp = (u64, VerifiedResult);

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
    /// The latest verified result whose L1 inclusion is finalized.
    finalized: Option<AtTimestamp>,
    /// The latest finalized L1 block seen.
    finalized_l1: Option<BlockNumHash>,
    /// Each chain's latest local-safe timestamp, as its source reported.
    local_safe_timestamps: BTreeMap<u64, u64>,
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
            finalized_l1: None,
            local_safe_timestamps: BTreeMap::new(),
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
        if self.finalized_l1.is_some_and(|l1| result.l1_inclusion.number <= l1.number) {
            self.finalized = Some((timestamp, result.clone()));
        }
        self.latest = Some((timestamp, result));
        Ok(())
    }

    /// Records the latest finalized L1 block.
    pub(crate) fn set_finalized_l1(
        &mut self,
        finalized_l1: BlockNumHash,
    ) -> Result<(), StoreError> {
        if self.finalized_l1 == Some(finalized_l1) {
            return Ok(());
        }
        self.finalized = self.latest_included_by(finalized_l1.number)?;
        self.finalized_l1 = Some(finalized_l1);
        Ok(())
    }

    /// The latest result whose L1 inclusion is at or below `l1_number`, by binary search over
    /// the stored results, whose L1 inclusions never decrease.
    fn latest_included_by(&self, l1_number: u64) -> Result<Option<AtTimestamp>, StoreError> {
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

    /// The result of `timestamp`, if verified.
    pub(crate) fn result_at(&self, timestamp: u64) -> Result<Option<VerifiedResult>, StoreError> {
        match &self.latest {
            Some((latest_ts, latest)) if *latest_ts == timestamp => Ok(Some(latest.clone())),
            Some((latest_ts, _)) if *latest_ts > timestamp => self.store.result(timestamp),
            _ => Ok(None),
        }
    }

    /// Records a denied block, durably, before the authority answers it.
    #[cfg_attr(not(test), expect(dead_code, reason = "nothing denies blocks yet"))]
    pub(crate) fn deny(&self, record: &DeniedRecord) -> Result<(), StoreError> {
        self.store.put_denied(record)
    }

    /// The denied blocks of `chain_id` at or above `from_number`, by number.
    #[cfg_attr(not(test), expect(dead_code, reason = "nothing denies blocks yet"))]
    pub(crate) fn denied_blocks(
        &self,
        chain_id: u64,
        from_number: u64,
    ) -> Result<Vec<DeniedRecord>, StoreError> {
        self.store.denied(chain_id, from_number)
    }

    /// Records a chain's latest local-safe timestamp.
    pub(crate) fn set_local_safe_timestamp(&mut self, chain_id: u64, timestamp: u64) {
        self.local_safe_timestamps.insert(chain_id, timestamp);
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
    fn head_in(&self, chain_id: u64, verified: Option<&AtTimestamp>) -> Option<VerifierHead> {
        let genesis_time = *self.genesis_times.get(&chain_id)?;
        let is_pre_activation = self
            .local_safe_timestamps
            .get(&chain_id)
            .is_some_and(|&local_safe| local_safe < self.activation_timestamp);
        if is_pre_activation {
            return Some(VerifierHead::PreActivation);
        }
        Some(match verified {
            Some((timestamp, result)) => {
                VerifierHead::Verified { block: result.heads[&chain_id], timestamp: *timestamp }
            }
            // The last timestamp before activation, clamped to genesis so the node can resolve it.
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

    #[test]
    fn local_safe_before_activation_answers_pre_activation() {
        let dir = tempfile::tempdir().unwrap();
        let mut state = state(dir.path());
        state.set_local_safe_timestamp(CHAIN, ACTIVATION - 1);

        assert_eq!(state.verified_head(CHAIN), Some(VerifierHead::PreActivation));
    }

    #[test]
    fn verified_head_is_the_latest_result_and_finalized_the_latest_finalized_inclusion() {
        let dir = tempfile::tempdir().unwrap();
        let mut state = state(dir.path());
        state.commit(ACTIVATION, result(1, 5)).unwrap();
        state.commit(ACTIVATION + 1, result(2, 6)).unwrap();
        state.commit(ACTIVATION + 2, result(3, 8)).unwrap();
        state.set_finalized_l1(block(7)).unwrap();

        assert_eq!(state.verified_head(CHAIN), verified(3, ACTIVATION + 2));
        assert_eq!(state.finalized_head(CHAIN), verified(2, ACTIVATION + 1));
        assert_eq!(state.next_timestamp(), ACTIVATION + 3);

        // A commit within finality moves the finalized head along.
        state.commit(ACTIVATION + 3, result(4, 7)).unwrap();
        assert_eq!(state.finalized_head(CHAIN), verified(2, ACTIVATION + 1));
        state.set_finalized_l1(block(9)).unwrap();
        state.commit(ACTIVATION + 4, result(5, 9)).unwrap();
        assert_eq!(state.finalized_head(CHAIN), verified(5, ACTIVATION + 4));
    }

    /// An L1 inclusion below an earlier one is raised to it: the later result needs the earlier
    /// one, so it is not final before it.
    #[test]
    fn l1_inclusion_never_decreases() {
        let dir = tempfile::tempdir().unwrap();
        let mut state = state(dir.path());
        state.commit(ACTIVATION, result(1, 8)).unwrap();
        state.commit(ACTIVATION + 1, result(2, 6)).unwrap();
        state.set_finalized_l1(block(7)).unwrap();

        assert_eq!(state.result_at(ACTIVATION + 1).unwrap().unwrap().l1_inclusion, block(8));
        assert_eq!(
            state.finalized_head(CHAIN),
            Some(VerifierHead::Anchor { timestamp: ACTIVATION })
        );
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
                })
                .unwrap();
        }

        let mut state = state(dir.path());

        assert_eq!(state.verified_head(CHAIN), verified(7, ACTIVATION + 6));
        assert_eq!(state.next_timestamp(), ACTIVATION + 7);
        state.set_finalized_l1(block(8)).unwrap();
        assert_eq!(state.finalized_head(CHAIN), verified(5, ACTIVATION + 4));
        state.set_finalized_l1(block(5)).unwrap();
        assert_eq!(state.finalized_head(CHAIN), verified(2, ACTIVATION + 1));
        state.set_finalized_l1(block(4)).unwrap();
        assert_eq!(
            state.finalized_head(CHAIN),
            Some(VerifierHead::Anchor { timestamp: ACTIVATION })
        );
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
}
