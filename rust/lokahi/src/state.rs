//! What the verifier has established, and the heads the authority answers from it.

use alloy_eips::BlockNumHash;
use kona_engine::VerifierHead;
use std::collections::BTreeMap;

/// The verified result of one timestamp: every chain's local-safe block at it, all of whose
/// executing messages are valid.
#[derive(Debug, Clone, PartialEq, Eq)]
pub(crate) struct VerifiedResult {
    /// The latest L1 block any of the chains needed to derive its block: the result holds on an
    /// L1 chain that includes it.
    pub l1_inclusion: BlockNumHash,
    /// Each chain's block at the timestamp, by chain ID.
    pub heads: BTreeMap<u64, BlockNumHash>,
}

/// The verifier's state, shared with the authority RPC. It lives in memory only.
#[derive(Debug)]
pub(crate) struct Verified {
    /// The first timestamp the verifier covers.
    activation_timestamp: u64,
    /// Each chain's L2 genesis time, by chain ID.
    genesis_times: BTreeMap<u64, u64>,
    /// Verified results by timestamp. Timestamps are contiguous from activation.
    results: BTreeMap<u64, VerifiedResult>,
    /// The latest finalized L1 block seen.
    finalized_l1: Option<BlockNumHash>,
    /// Each chain's latest local-safe timestamp, as its source reported.
    local_safe_timestamps: BTreeMap<u64, u64>,
}

impl Verified {
    /// An empty state for chains with the given genesis times.
    pub(crate) const fn new(activation_timestamp: u64, genesis_times: BTreeMap<u64, u64>) -> Self {
        Self {
            activation_timestamp,
            genesis_times,
            results: BTreeMap::new(),
            finalized_l1: None,
            local_safe_timestamps: BTreeMap::new(),
        }
    }

    /// The next timestamp to verify.
    pub(crate) fn next_timestamp(&self) -> u64 {
        self.results.last_key_value().map_or(self.activation_timestamp, |(ts, _)| ts + 1)
    }

    /// Commits the result of the next timestamp.
    pub(crate) fn commit(&mut self, timestamp: u64, result: VerifiedResult) {
        debug_assert_eq!(timestamp, self.next_timestamp(), "timestamps are verified in order");
        self.results.insert(timestamp, result);
    }

    /// Records the latest finalized L1 block.
    pub(crate) const fn set_finalized_l1(&mut self, finalized_l1: BlockNumHash) {
        self.finalized_l1 = Some(finalized_l1);
    }

    /// Records a chain's latest local-safe timestamp.
    pub(crate) fn set_local_safe_timestamp(&mut self, chain_id: u64, timestamp: u64) {
        self.local_safe_timestamps.insert(chain_id, timestamp);
    }

    /// The verified safe head of `chain_id`, or `None` for a chain the verifier does not cover.
    pub(crate) fn verified_head(&self, chain_id: u64) -> Option<VerifierHead> {
        self.head_where(chain_id, |_| true)
    }

    /// The latest verified head of `chain_id` whose L1 inclusion is finalized, or `None` for a
    /// chain the verifier does not cover.
    pub(crate) fn finalized_head(&self, chain_id: u64) -> Option<VerifierHead> {
        let Some(finalized_l1) = self.finalized_l1 else {
            return self.head_where(chain_id, |_| false);
        };
        self.head_where(chain_id, |result| result.l1_inclusion.number <= finalized_l1.number)
    }

    /// The head of `chain_id` in the latest result that `accept`s, like op-supernode's
    /// `LatestVerifiedL2Block` and `VerifiedBlockAtL1`.
    fn head_where(
        &self,
        chain_id: u64,
        accept: impl Fn(&VerifiedResult) -> bool,
    ) -> Option<VerifierHead> {
        let genesis_time = *self.genesis_times.get(&chain_id)?;
        let is_pre_activation = self
            .local_safe_timestamps
            .get(&chain_id)
            .is_some_and(|&local_safe| local_safe < self.activation_timestamp);
        if is_pre_activation {
            return Some(VerifierHead::PreActivation);
        }
        let verified = self.results.iter().rev().find(|(_, result)| accept(result));
        Some(match verified {
            Some((&timestamp, result)) => {
                VerifierHead::Verified { block: result.heads[&chain_id], timestamp }
            }
            // The last timestamp before activation, clamped to genesis so the node can resolve it.
            None => VerifierHead::Anchor {
                timestamp: self.activation_timestamp.saturating_sub(1).max(genesis_time),
            },
        })
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use alloy_primitives::B256;

    const CHAIN: u64 = 10;
    const ACTIVATION: u64 = 100;

    fn block(number: u64) -> BlockNumHash {
        BlockNumHash { number, hash: B256::with_last_byte(number as u8) }
    }

    fn result(l2: u64, l1: u64) -> VerifiedResult {
        VerifiedResult { l1_inclusion: block(l1), heads: BTreeMap::from([(CHAIN, block(l2))]) }
    }

    fn state() -> Verified {
        Verified::new(ACTIVATION, BTreeMap::from([(CHAIN, ACTIVATION)]))
    }

    #[test]
    fn nothing_verified_answers_the_anchor_clamped_to_genesis() {
        let state = state();

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
        let mut state = state();
        state.set_local_safe_timestamp(CHAIN, ACTIVATION - 1);

        assert_eq!(state.verified_head(CHAIN), Some(VerifierHead::PreActivation));
    }

    #[test]
    fn verified_head_is_the_latest_result_and_finalized_the_latest_finalized_inclusion() {
        let mut state = state();
        state.commit(ACTIVATION, result(1, 5));
        state.commit(ACTIVATION + 1, result(2, 6));
        state.commit(ACTIVATION + 2, result(3, 8));
        state.set_finalized_l1(block(7));

        assert_eq!(
            state.verified_head(CHAIN),
            Some(VerifierHead::Verified { block: block(3), timestamp: ACTIVATION + 2 })
        );
        assert_eq!(
            state.finalized_head(CHAIN),
            Some(VerifierHead::Verified { block: block(2), timestamp: ACTIVATION + 1 })
        );
        assert_eq!(state.next_timestamp(), ACTIVATION + 3);
    }
}
