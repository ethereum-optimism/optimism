//! The pure half of a verification round: the observation it operates on, and the decision it
//! reaches.
//!
//! Nothing in this module performs I/O or touches a store. A round observes the world once, into a
//! [`RoundObservation`], and every decision is then a function of that value alone. That is what
//! makes the decision testable without a node, and what makes a round reproducible: re-running
//! [`check_preconditions`] and [`decide_verified_result`] on a recorded observation must reach the
//! same decision the live node reached.

use crate::verified::{RoundResult, VerifiedResult};
use alloy_eips::BlockNumHash;
use alloy_primitives::ChainId;
use std::collections::BTreeMap;

/// One chain's contribution to a round: the L2 block at the round's timestamp, and the L1 block
/// that block was derived from.
///
/// The two come out of a single atomic read of the chain's engine state, so the pairing is not an
/// assembly of two separately-sampled halves.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct ChainFrontier {
    /// The chain's L2 block at the round's timestamp.
    pub block: BlockNumHash,
    /// The L1 block that L2 block was derived from.
    pub l1_inclusion: BlockNumHash,
}

/// A consistent snapshot of everything a round decides from.
///
/// Captured once, at the top of the round, so the decision cannot see a chain advance underneath
/// it. Every field is plain data: no handles, no channels, nothing that could be read again.
#[derive(Debug, Clone, PartialEq, Eq, Default)]
pub struct RoundObservation {
    /// The most recently committed frontier, or [`None`] when nothing is committed yet.
    pub last_verified: Option<VerifiedResult>,
    /// The timestamp this round would verify.
    pub next_timestamp: u64,
    /// Every chain's frontier at [`Self::next_timestamp`], or [`None`] when at least one chain
    /// could not answer for that timestamp yet.
    ///
    /// All-or-nothing on purpose: the round verifies a timestamp across the whole dependency set,
    /// and a partial frontier would verify some chains' messages against an absent view of
    /// others'.
    pub frontier: Option<BTreeMap<ChainId, ChainFrontier>>,
    /// Whether every L1 block this round's frontier relies on is still canonical.
    pub l1_consistent: bool,
    /// Whether the *committed* frontier's L1 inclusion has left the canonical L1 chain.
    ///
    /// Tracked apart from [`Self::l1_consistent`] because the two call for opposite actions: a
    /// stale frontier head is waited out, a stale committed head has to be rewound.
    pub l1_needs_rewind: bool,
}

impl RoundObservation {
    /// Returns whether every chain answered for this round's timestamp.
    pub const fn chains_ready(&self) -> bool {
        self.frontier.is_some()
    }

    /// Returns the L1 block that included the whole frontier: the highest of the per-chain L1
    /// inclusions.
    ///
    /// The maximum, not the minimum: this names the L1 block by which every chain's block at this
    /// timestamp had been derived. Chains derive at their own pace, so one chain reaching the
    /// timestamp from a later L1 block sets the bound for the set.
    pub fn l1_inclusion(&self) -> Option<BlockNumHash> {
        let frontier = self.frontier.as_ref()?;
        frontier.values().map(|chain| chain.l1_inclusion).max_by_key(|l1| l1.number)
    }

    /// Returns the L1 blocks a round at this observation depends on being canonical.
    pub fn frontier_l1_blocks(&self) -> Vec<BlockNumHash> {
        self.frontier
            .as_ref()
            .map(|frontier| frontier.values().map(|chain| chain.l1_inclusion).collect())
            .unwrap_or_default()
    }
}

/// What a round decided to do.
#[derive(Debug, Default, Clone, Copy, PartialEq, Eq, Hash, PartialOrd, Ord)]
pub enum Decision {
    /// Do nothing this round and observe again later.
    #[default]
    Wait,
    /// Commit the round's frontier as verified.
    Advance,
    /// Replace the blocks the round found invalid.
    Invalidate,
    /// Drop committed frontiers that rest on an L1 block that is no longer canonical.
    Rewind,
}

impl Decision {
    /// Returns the decision's name, as it appears in logs and metric labels.
    pub const fn as_str(&self) -> &'static str {
        match self {
            Self::Wait => "wait",
            Self::Advance => "advance",
            Self::Invalidate => "invalidate",
            Self::Rewind => "rewind",
        }
    }

    /// Every decision a round can reach.
    ///
    /// Used to pre-register metric label series so a decision that has not happened yet still
    /// reads as zero rather than as an absent series.
    pub const ALL: [Self; 4] = [Self::Wait, Self::Advance, Self::Invalidate, Self::Rewind];

    /// Returns whether applying this decision has durable side effects, and therefore has to be
    /// written to the write-ahead log before any of them begins.
    pub const fn is_effectful(&self) -> bool {
        match self {
            Self::Wait => false,
            Self::Advance | Self::Invalidate | Self::Rewind => true,
        }
    }
}

impl core::fmt::Display for Decision {
    fn fmt(&self, f: &mut core::fmt::Formatter<'_>) -> core::fmt::Result {
        f.write_str(self.as_str())
    }
}

/// A decision together with the round result it was reached from.
#[derive(Debug, Clone, PartialEq, Eq, Default)]
pub struct StepOutput {
    /// What to do.
    pub decision: Decision,
    /// The round's result, empty for a decision that carries none.
    pub result: RoundResult,
}

impl StepOutput {
    /// Returns a decision that carries no result.
    pub fn bare(decision: Decision) -> Self {
        Self { decision, result: RoundResult::default() }
    }
}

/// Decides whether the observation alone already settles the round, before any verification runs.
///
/// Returns [`None`] when verification should proceed. Ordering is load-bearing: a stale *committed*
/// L1 inclusion is a rewind and takes precedence over a stale frontier one, which is merely waited
/// out.
pub fn check_preconditions(observation: &RoundObservation) -> Option<StepOutput> {
    if !observation.chains_ready() {
        return Some(StepOutput::bare(Decision::Wait));
    }
    if observation.l1_needs_rewind {
        return Some(StepOutput::bare(Decision::Rewind));
    }
    if !observation.l1_consistent {
        return Some(StepOutput::bare(Decision::Wait));
    }
    None
}

/// Decides what to do with a completed verification result.
pub fn decide_verified_result(result: RoundResult) -> StepOutput {
    if result.verified.l2_heads.is_empty() {
        return StepOutput::bare(Decision::Wait);
    }
    let decision = if result.is_valid() { Decision::Advance } else { Decision::Invalidate };
    StepOutput { decision, result }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::verified::InvalidHead;
    use alloy_primitives::B256;

    fn l1(number: u64) -> BlockNumHash {
        BlockNumHash { number, hash: B256::repeat_byte(number as u8) }
    }

    fn l2(number: u64) -> BlockNumHash {
        BlockNumHash { number, hash: B256::repeat_byte(0x80 | number as u8) }
    }

    fn frontier(chains: [(ChainId, u64, u64); 2]) -> BTreeMap<ChainId, ChainFrontier> {
        chains
            .into_iter()
            .map(|(chain_id, l2_number, l1_number)| {
                (chain_id, ChainFrontier { block: l2(l2_number), l1_inclusion: l1(l1_number) })
            })
            .collect()
    }

    fn ready() -> RoundObservation {
        RoundObservation {
            last_verified: None,
            next_timestamp: 100,
            frontier: Some(frontier([(901, 10, 5), (902, 12, 7)])),
            l1_consistent: true,
            l1_needs_rewind: false,
        }
    }

    fn advance_result() -> RoundResult {
        RoundResult {
            verified: VerifiedResult {
                timestamp: 100,
                l1_inclusion: l1(7),
                l2_heads: BTreeMap::from([(901, l2(10)), (902, l2(12))]),
            },
            invalid_heads: BTreeMap::new(),
        }
    }

    #[test]
    fn unready_chains_wait() {
        let observation = RoundObservation { frontier: None, ..ready() };
        assert_eq!(check_preconditions(&observation), Some(StepOutput::bare(Decision::Wait)));
    }

    #[test]
    fn a_stale_committed_l1_inclusion_rewinds() {
        let observation =
            RoundObservation { l1_needs_rewind: true, l1_consistent: false, ..ready() };
        assert_eq!(check_preconditions(&observation), Some(StepOutput::bare(Decision::Rewind)));
    }

    #[test]
    fn a_stale_frontier_l1_inclusion_only_waits() {
        // The committed frontier is still canonical; only a chain's own L1 head is behind. Waiting
        // lets that chain catch up, where rewinding would throw away verified history for it.
        let observation = RoundObservation { l1_consistent: false, ..ready() };
        assert_eq!(check_preconditions(&observation), Some(StepOutput::bare(Decision::Wait)));
    }

    #[test]
    fn a_ready_consistent_observation_proceeds_to_verification() {
        assert_eq!(check_preconditions(&ready()), None);
    }

    #[test]
    fn unready_chains_take_precedence_over_a_rewind() {
        // Without a frontier there is nothing to rewind *to* that this round observed, so the
        // round waits and re-observes rather than acting on a half-read world.
        let observation = RoundObservation { frontier: None, l1_needs_rewind: true, ..ready() };
        assert_eq!(check_preconditions(&observation), Some(StepOutput::bare(Decision::Wait)));
    }

    #[test]
    fn an_empty_result_waits() {
        let output = decide_verified_result(RoundResult::default());
        assert_eq!(output, StepOutput::bare(Decision::Wait));
    }

    #[test]
    fn an_all_valid_result_advances() {
        let output = decide_verified_result(advance_result());
        assert_eq!(output.decision, Decision::Advance);
        assert_eq!(output.result, advance_result());
    }

    #[test]
    fn a_result_with_an_invalid_head_invalidates() {
        let mut result = advance_result();
        result.invalid_heads.insert(
            902,
            InvalidHead {
                block: l2(12),
                state_root: B256::repeat_byte(0xaa),
                message_passer_storage_root: B256::repeat_byte(0xbb),
            },
        );
        let output = decide_verified_result(result.clone());
        assert_eq!(output.decision, Decision::Invalidate);
        // The invalidation carries the whole round, valid heads included: the frontier it commits
        // is the one the replacement is measured against.
        assert_eq!(output.result, result);
    }

    #[test]
    fn the_l1_inclusion_is_the_highest_per_chain_inclusion() {
        assert_eq!(ready().l1_inclusion(), Some(l1(7)));
        assert_eq!(RoundObservation { frontier: None, ..ready() }.l1_inclusion(), None);
    }

    #[test]
    fn the_frontier_l1_blocks_are_every_chains_inclusion() {
        let mut blocks = ready().frontier_l1_blocks();
        blocks.sort_by_key(|block| block.number);
        assert_eq!(blocks, vec![l1(5), l1(7)]);
        assert!(RoundObservation { frontier: None, ..ready() }.frontier_l1_blocks().is_empty());
    }

    #[test]
    fn only_waiting_is_free_of_side_effects() {
        assert!(!Decision::Wait.is_effectful());
        for decision in [Decision::Advance, Decision::Invalidate, Decision::Rewind] {
            assert!(decision.is_effectful(), "{decision} has durable side effects");
        }
    }

    #[test]
    fn every_decision_has_a_stable_name() {
        let names: Vec<_> = Decision::ALL.iter().map(|d| d.to_string()).collect();
        assert_eq!(names, vec!["wait", "advance", "invalidate", "rewind"]);
    }
}
