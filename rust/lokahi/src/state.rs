//! What the verifier has established, and the heads the authority answers from it.
//!
//! The [`Store`] is the record: a result is written to it before the authority can answer it, so
//! a restart answers at least the heads answered before it. Only the latest verified and the
//! latest finalized results are held in memory; older ones are read from the store.

use crate::store::{Store, StoreError};
use alloy_eips::BlockNumHash;
use kona_engine::VerifierHead;
use kona_protocol::OutputRoot;
use std::collections::BTreeMap;

/// The verified result of one timestamp: every chain's local-safe block at it, all of whose
/// executing messages are valid.
#[derive(Debug, Clone, PartialEq, Eq)]
pub(crate) struct VerifiedResult {
    /// The latest L1 block any of the chains needed to derive its block at this or an earlier
    /// timestamp: the result holds on an L1 chain that includes it.
    pub l1_inclusion: BlockNumHash,
    /// Each chain's block at the timestamp, by chain ID.
    pub chains: BTreeMap<u64, ChainResult>,
}

/// One chain's block in a [`VerifiedResult`], with what `superroot_atTimestamp` answers for it.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub(crate) struct ChainResult {
    /// The block.
    pub block: BlockNumHash,
    /// The block's output root preimage.
    pub output: OutputRoot,
    /// The L1 block the chain's source reported the block safe at.
    pub required_l1: BlockNumHash,
    /// The output root preimage of the denied block this one replaced, where the chain produced
    /// the block at exactly the timestamp as the replacement of a block verification denied.
    pub replaced: Option<OutputRoot>,
}

impl ChainResult {
    /// The output the chain would have at the timestamp had verification denied nothing: the
    /// denied block where this one replaced it.
    pub(crate) fn optimistic(&self) -> &OutputRoot {
        self.replaced.as_ref().unwrap_or(&self.output)
    }
}

/// A block the verifier denied: its chain must replace it.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub(crate) struct DeniedRecord {
    /// The chain of the block.
    pub chain_id: u64,
    /// The denied block.
    pub block: BlockNumHash,
    /// The output root preimage of the denied block.
    pub output: OutputRoot,
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
    /// The first timestamp the verifier covers, or `None` when no chain schedules interop: every
    /// timestamp is then before activation, and nothing is verified.
    activation_timestamp: Option<u64>,
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
    /// The next timestamp when the state was opened: where this run started verifying.
    start_timestamp: u64,
    /// The L1 block verification is at: every L1 block strictly below it has been fully
    /// considered, so a timestamp that is not verified needs L1 data at or above it. Zero until
    /// the first round since the start decides it.
    current_l1: BlockNumHash,
    /// The timestamp from which on verification pauses, for tests.
    pause_at: Option<u64>,
}

impl Verified {
    /// The state held in `store`, for chains with the given genesis times. A store written for
    /// other chains or another activation is refused.
    pub(crate) fn open(
        store: Store,
        activation_timestamp: Option<u64>,
        genesis_times: BTreeMap<u64, u64>,
    ) -> Result<Self, StoreError> {
        let activation = activation_or_never(activation_timestamp);
        let latest = store.last_result()?;
        if let Some((_, result)) = &latest {
            let first = store.first_result()?.map_or(activation, |(ts, _)| ts);
            if first != activation || !result.chains.keys().eq(genesis_times.keys()) {
                return Err(StoreError::OtherConfig {
                    first,
                    chains: result.chains.keys().copied().collect(),
                    activation,
                    expected: genesis_times.keys().copied().collect(),
                });
            }
        }
        let start_timestamp = latest.as_ref().map_or(activation, |(ts, _)| ts + 1);
        Ok(Self {
            activation_timestamp,
            genesis_times,
            store,
            latest,
            finalized: None,
            // Without interop there is nothing to verify, so nothing to catch up with.
            caught_up: activation_timestamp.is_none(),
            start_timestamp,
            current_l1: BlockNumHash::default(),
            pause_at: None,
        })
    }

    /// Where this run started verifying: the activation timestamp on an empty store, the one
    /// after the last stored result on a resume.
    pub(crate) const fn start_timestamp(&self) -> u64 {
        self.start_timestamp
    }

    /// The L1 block verification is at: a timestamp that is not verified needs L1 data at or
    /// above it.
    pub(crate) const fn current_l1(&self) -> BlockNumHash {
        self.current_l1
    }

    /// Records the L1 block verification is at.
    pub(crate) const fn set_current_l1(&mut self, current_l1: BlockNumHash) {
        self.current_l1 = current_l1;
    }

    /// The timestamp from which on verification pauses, if set.
    pub(crate) const fn pause_at(&self) -> Option<u64> {
        self.pause_at
    }

    /// Pauses verification before `timestamp` and every later one, or resumes it with `None`.
    pub(crate) const fn set_pause_at(&mut self, timestamp: Option<u64>) {
        self.pause_at = timestamp;
    }

    /// Whether `timestamp` is verified: before activation every timestamp is.
    pub(crate) fn is_verified_at(&self, timestamp: u64) -> bool {
        !self.is_active_at(timestamp) || timestamp < self.next_timestamp()
    }

    /// The first timestamp the verifier covers, or `None` when no chain schedules interop.
    pub(crate) const fn activation_timestamp(&self) -> Option<u64> {
        self.activation_timestamp
    }

    /// Whether interop is active at `timestamp`.
    pub(crate) fn is_active_at(&self, timestamp: u64) -> bool {
        self.activation_timestamp.is_some_and(|activation| timestamp >= activation)
    }

    /// The next timestamp to verify.
    pub(crate) fn next_timestamp(&self) -> u64 {
        self.latest
            .as_ref()
            .map_or_else(|| activation_or_never(self.activation_timestamp), |(ts, _)| ts + 1)
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
        self.finalized
            .as_ref()
            .map_or_else(|| activation_or_never(self.activation_timestamp), |(ts, _)| ts + 1)
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
    /// were decided on them, durably before the authority answers the rewound heads. Verification
    /// is then at no known L1 block until a round decides it again.
    ///
    /// `timestamp` must be above the finalized result: finality never moves back.
    pub(crate) fn rewind(&mut self, timestamp: u64) -> Result<(usize, usize), StoreError> {
        assert!(
            timestamp >= self.first_unfinalized_timestamp(),
            "rewind to {timestamp} below the finalized result {:?}",
            self.finalized.as_ref().map(|(ts, _)| ts)
        );
        self.current_l1 = BlockNumHash::default();
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
        let (mut lo, mut hi) = (activation_or_never(self.activation_timestamp), *latest_ts);
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
        Ok(self
            .denials_at(chain_id, block.number)?
            .into_iter()
            .find(|record| record.block == block))
    }

    /// The denied blocks of `chain_id` at height `number`.
    pub(crate) fn denials_at(
        &self,
        chain_id: u64,
        number: u64,
    ) -> Result<Vec<DeniedRecord>, StoreError> {
        let mut denied = self.store.denied(chain_id, number)?;
        denied.retain(|record| record.block.number == number);
        Ok(denied)
    }

    /// The denied blocks of every chain decided at or after `timestamp`.
    pub(crate) fn denials_decided_from(
        &self,
        timestamp: u64,
    ) -> Result<Vec<DeniedRecord>, StoreError> {
        self.store.denials_decided_from(timestamp)
    }

    /// The epoch, which changes with every rewind that deletes a denial.
    pub(crate) const fn epoch(&self) -> alloy_primitives::B256 {
        self.store.epoch()
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
    /// Without interop it is [`VerifierHead::PreActivation`], as op-supernode answers without a
    /// verifier: the node's labels follow its own local view.
    fn head_in(&self, chain_id: u64, verified: Option<&AtTimestamp>) -> Option<VerifierHead> {
        let genesis_time = *self.genesis_times.get(&chain_id)?;
        Some(match (verified, self.activation_timestamp) {
            (Some((timestamp, result)), _) => VerifierHead::Verified {
                block: result.chains[&chain_id].block,
                timestamp: *timestamp,
                l1: result.l1_inclusion,
            },
            (None, None) => VerifierHead::PreActivation,
            // Clamped to genesis so the node can resolve it.
            (None, Some(activation)) => {
                VerifierHead::Anchor { timestamp: activation.saturating_sub(1).max(genesis_time) }
            }
        })
    }
}

/// The timestamp verification starts at: `activation_timestamp`, or one no timestamp reaches when
/// no chain schedules interop.
fn activation_or_never(activation_timestamp: Option<u64>) -> u64 {
    activation_timestamp.unwrap_or(u64::MAX)
}

#[cfg(test)]
pub(crate) mod tests {
    use super::*;
    use alloy_primitives::B256;
    use std::path::Path;

    const CHAIN: u64 = 10;
    const ACTIVATION: u64 = 100;

    fn block(number: u64) -> BlockNumHash {
        BlockNumHash { number, hash: B256::with_last_byte(number as u8) }
    }

    fn result(l2: u64, l1: u64) -> VerifiedResult {
        VerifiedResult {
            l1_inclusion: block(l1),
            chains: BTreeMap::from([(CHAIN, chain_result(block(l2)))]),
        }
    }

    /// The output of `block` in tests: its state roots are fixed.
    pub(crate) fn output_of(block: BlockNumHash) -> OutputRoot {
        OutputRoot::from_parts(B256::repeat_byte(0x51), B256::repeat_byte(0x52), block.hash)
    }

    /// `block` as a chain's block in a verified result, safe at L1 block 1, replacing nothing.
    pub(crate) fn chain_result(block: BlockNumHash) -> ChainResult {
        ChainResult {
            block,
            output: output_of(block),
            required_l1: BlockNumHash { number: 1, hash: B256::repeat_byte(1) },
            replaced: None,
        }
    }

    /// The state in the store at `dir`.
    pub(crate) fn open_state(
        dir: &Path,
        activation: impl Into<Option<u64>>,
        genesis_times: BTreeMap<u64, u64>,
    ) -> Verified {
        Verified::open(Store::open(dir).unwrap(), activation.into(), genesis_times).unwrap()
    }

    fn state(dir: &Path) -> Verified {
        open_state(dir, ACTIVATION, BTreeMap::from([(CHAIN, ACTIVATION)]))
    }

    fn verified(l2: u64, timestamp: u64) -> Option<(BlockNumHash, u64)> {
        Some((block(l2), timestamp))
    }

    /// The verified block and timestamp of `head`, without its L1 block.
    fn verified_at(head: Option<VerifierHead>) -> Option<(BlockNumHash, u64)> {
        match head? {
            VerifierHead::Verified { block, timestamp, .. } => Some((block, timestamp)),
            _ => None,
        }
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

        assert_eq!(verified_at(state.verified_head(CHAIN)), verified(3, ACTIVATION + 2));
        assert_eq!(verified_at(state.finalized_head(CHAIN)), verified(2, ACTIVATION + 1));
        assert_eq!(state.next_timestamp(), ACTIVATION + 3);

        // Finality never moves back, also for an older finalized L1 block.
        finalize_at(&mut state, 5);
        assert_eq!(verified_at(state.finalized_head(CHAIN)), verified(2, ACTIVATION + 1));
        finalize_at(&mut state, 9);
        assert_eq!(verified_at(state.finalized_head(CHAIN)), verified(3, ACTIVATION + 2));
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
            output: output_of(block(number)),
            timestamp,
            l1: block(7),
        };
        state.deny(&denial(2, ACTIVATION + 1)).unwrap();
        state.deny(&denial(5, ACTIVATION + 4)).unwrap();

        assert_eq!(state.rewind(ACTIVATION + 2).unwrap(), (2, 1));

        assert_eq!(verified_at(state.verified_head(CHAIN)), verified(2, ACTIVATION + 1));
        assert_eq!(state.next_timestamp(), ACTIVATION + 2);
        assert_eq!(verified_at(state.finalized_head(CHAIN)), verified(1, ACTIVATION));
        assert_eq!(state.denied_blocks(CHAIN, 0).unwrap(), vec![denial(2, ACTIVATION + 1)]);

        // Everything after the finalized result can go.
        state.rewind(ACTIVATION + 1).unwrap();
        assert_eq!(verified_at(state.verified_head(CHAIN)), verified(1, ACTIVATION));
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
                    output: output_of(block(9)),
                    timestamp: ACTIVATION + 7,
                    l1: block(12),
                })
                .unwrap();
        }

        let mut state = state(dir.path());

        assert_eq!(verified_at(state.verified_head(CHAIN)), verified(7, ACTIVATION + 6));
        assert_eq!(state.next_timestamp(), ACTIVATION + 7);
        assert_eq!(
            state.finalized_head(CHAIN),
            Some(VerifierHead::Anchor { timestamp: ACTIVATION })
        );
        assert_eq!(state.latest_included_by(4).unwrap(), None);
        assert_eq!(state.latest_included_by(5).unwrap(), Some((ACTIVATION + 1, result(2, 5))));
        finalize_at(&mut state, 8);
        assert_eq!(verified_at(state.finalized_head(CHAIN)), verified(5, ACTIVATION + 4));
        assert_eq!(state.result_at(ACTIVATION + 2).unwrap(), Some(result(3, 6)));
        assert_eq!(state.result_at(ACTIVATION + 7).unwrap(), None);
        assert_eq!(
            state.denied_blocks(CHAIN, 0).unwrap().iter().map(|r| r.block).collect::<Vec<_>>(),
            vec![block(9)]
        );
        state.commit(ACTIVATION + 7, result(8, 12)).unwrap();
        assert_eq!(verified_at(state.verified_head(CHAIN)), verified(8, ACTIVATION + 7));
    }

    #[rstest::rstest]
    #[case::other_activation(Some(ACTIVATION + 1), BTreeMap::from([(CHAIN, ACTIVATION)]))]
    #[case::without_interop(None, BTreeMap::from([(CHAIN, ACTIVATION)]))]
    #[case::other_chains(
        Some(ACTIVATION),
        BTreeMap::from([(CHAIN, ACTIVATION), (CHAIN + 1, ACTIVATION)])
    )]
    fn a_store_of_another_config_is_refused(
        #[case] activation: Option<u64>,
        #[case] genesis_times: BTreeMap<u64, u64>,
    ) {
        let dir = tempfile::tempdir().unwrap();
        state(dir.path()).commit(ACTIVATION, result(1, 5)).unwrap();

        let opened = Verified::open(Store::open(dir.path()).unwrap(), activation, genesis_times);

        assert!(matches!(opened, Err(StoreError::OtherConfig { .. })), "got {opened:?}");
    }

    /// Without interop every timestamp is verified, the state is caught up from the start, and
    /// both heads let the node follow its own local view.
    #[test]
    fn without_interop_the_heads_follow_the_nodes_own_view() {
        let dir = tempfile::tempdir().unwrap();
        let state = open_state(dir.path(), None, BTreeMap::from([(CHAIN, ACTIVATION)]));

        assert!(state.is_caught_up());
        assert!(state.is_verified_at(u64::MAX - 1));
        assert!(!state.is_active_at(u64::MAX - 1));
        assert_eq!(state.verified_head(CHAIN), Some(VerifierHead::PreActivation));
        assert_eq!(state.finalized_head(CHAIN), Some(VerifierHead::PreActivation));
        assert_eq!(state.verified_head(CHAIN + 1), None, "a chain the verifier does not cover");
    }

    /// A denied block is told apart from another block at its height and on another chain, and a
    /// rewind that deletes it changes the epoch.
    #[test]
    fn a_denial_matches_the_chain_and_the_block_and_a_rewind_deleting_it_changes_the_epoch() {
        let dir = tempfile::tempdir().unwrap();
        let mut state = state(dir.path());
        let denied = DeniedRecord {
            chain_id: CHAIN,
            block: block(3),
            output: output_of(block(3)),
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

        let epoch = state.epoch();
        assert_eq!(state.rewind(ACTIVATION + 2).unwrap(), (0, 2));
        assert_eq!(state.denial(CHAIN, block(3)).unwrap(), None);
        assert_ne!(state.epoch(), epoch);
    }
}
