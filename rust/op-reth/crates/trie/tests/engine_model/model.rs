//! The reference model: the canonical chain as the consensus layer holds it, and where the engine
//! and its store must end up once they settle. It knows nothing of the engine's internals; what
//! it cannot predict exactly, because a save or the sync loop runs in the background, it bounds,
//! and the harness narrows the bound with what it observes.

use alloy_eips::{NumHash, eip1898::BlockWithParent};
use alloy_primitives::B256;
use std::ops::RangeInclusive;

/// A canonical block: what the store must agree with at its height.
#[derive(Clone, Copy, Debug)]
pub(super) struct ModelBlock {
    pub(super) hash: B256,
    pub(super) parent: B256,
    /// reth's persisted head just before reth committed the block; reth's persisted head never
    /// falls, so every save of the block sees at least this.
    pub(super) persisted_before_commit: u64,
    /// The serial of the transfer the block carries; rebuilding it on the same parent gives the
    /// same block.
    pub(super) serial: u64,
    pub(super) state_root: B256,
}

/// Where the engine may be: a tip it has reached for certain and the target its sync loop works
/// toward, so its actual tip lies between the two.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub(super) struct EngineCandidate {
    pub(super) target: u64,
    pub(super) tip: NumHash,
}

/// The engine's reply to a block from a live notification.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub(super) enum Reply {
    /// A parent-hash mismatch naming the engine's tip at the height below the block.
    Refused { tip_hash: B256 },
    /// `Ok`, whether the block was buffered, ignored or taken as a sync target.
    Taken,
}

pub(super) struct Model {
    /// Every state the engine may be in; an observation keeps the ones it confirms. No
    /// candidate's way to its settled tip passes through another's settled tip, so the first
    /// settled tip the harness observes is the one the engine stays at.
    candidates: Vec<EngineCandidate>,
    /// The canonical chain, genesis first.
    canonical: Vec<ModelBlock>,
    /// The highest head the chain has had, which no save's tip exceeds.
    highest_head: u64,
    /// The lowest block whose hash the engine has held since it started, through its tip.
    known_from: u64,
    /// The store has persisted at least through this block.
    persisted_at_least: u64,
    /// The proof window the store keeps.
    window: u64,
}

impl Model {
    pub(super) fn new(genesis: ModelBlock, window: u64) -> Self {
        Self {
            candidates: vec![EngineCandidate { target: 0, tip: NumHash::new(0, genesis.hash) }],
            canonical: vec![genesis],
            highest_head: 0,
            known_from: 1,
            persisted_at_least: 0,
            window,
        }
    }

    pub(super) const fn head(&self) -> u64 {
        self.canonical.len() as u64 - 1
    }

    pub(super) fn block(&self, number: u64) -> &ModelBlock {
        &self.canonical[number as usize]
    }

    pub(super) fn num_hash(&self, number: u64) -> NumHash {
        NumHash::new(number, self.block(number).hash)
    }

    pub(super) fn block_with_parent(&self, number: u64) -> BlockWithParent {
        BlockWithParent::new(self.block(number).parent, self.num_hash(number))
    }

    pub(super) fn is_canonical(&self, block: NumHash) -> bool {
        block.number <= self.head() && self.block(block.number).hash == block.hash
    }

    pub(super) fn push(&mut self, block: ModelBlock) {
        self.canonical.push(block);
        self.highest_head = self.highest_head.max(self.head());
    }

    /// Keeps the chain through `head`.
    pub(super) fn truncate(&mut self, head: u64) {
        self.canonical.truncate(head as usize + 1);
    }

    /// The serials of the blocks above `number`, the order to rebuild them in.
    pub(super) fn serials_above(&self, number: u64) -> Vec<u64> {
        self.canonical[number as usize + 1..].iter().map(|block| block.serial).collect()
    }

    pub(super) const fn window(&self) -> u64 {
        self.window
    }

    /// Where the store's earliest block may lie while its latest block is `latest`. Every save
    /// with tip `t` prunes to `t - window`, or not past reth's persisted head when the pruner
    /// caps it there. The save that wrote `latest` had a tip at or above it and saw reth
    /// persisted at least through `persisted_before_commit` of `latest`, and the earliest block
    /// never moves down, which gives the lower end. No save's tip exceeds the highest head,
    /// which gives the upper end.
    pub(super) fn earliest_bounds(&self, latest: u64) -> RangeInclusive<u64> {
        let persisted_before_the_save = self.block(latest).persisted_before_commit;
        let lowest = latest.saturating_sub(self.window).min(persisted_before_the_save);
        let highest = self.highest_head.saturating_sub(self.window);
        lowest..=highest
    }

    pub(super) const fn persisted_at_least(&self) -> u64 {
        self.persisted_at_least
    }

    /// The store was seen persisted through `latest`.
    pub(super) fn note_persisted(&mut self, latest: u64) {
        self.persisted_at_least = self.persisted_at_least.max(latest);
    }

    pub(super) fn candidates(&self) -> &[EngineCandidate] {
        &self.candidates
    }

    /// Fork points whose hash the engine holds for certain: the blocks it took in since it
    /// started, and genesis while the window is never pruned.
    pub(super) fn known_fork_points(&self) -> Vec<u64> {
        let mut points: Vec<u64> = self
            .highest_reach()
            .map(|reach| (self.known_from..=reach).collect())
            .unwrap_or_default();
        if self.window >= self.head() {
            points.push(0);
        }
        points
    }

    /// How far the sync loop can carry `candidate` over a chain whose head was `head`: to its
    /// target while its tip is canonical, nowhere once the tip is on an abandoned chain.
    fn reach(&self, candidate: &EngineCandidate, head: u64) -> u64 {
        if self.is_canonical(candidate.tip) {
            candidate.target.min(head).max(candidate.tip.number)
        } else {
            candidate.tip.number
        }
    }

    fn highest_reach(&self) -> Option<u64> {
        self.candidates.iter().map(|candidate| self.reach(candidate, self.head())).max()
    }

    /// The tip `candidate` settles on once the sync loop has done what it can.
    fn settled_tip(&self, candidate: &EngineCandidate) -> NumHash {
        if self.is_canonical(candidate.tip) {
            self.num_hash(self.reach(candidate, self.head()))
        } else {
            candidate.tip
        }
    }

    /// The tips the engine may settle on.
    pub(super) fn settled_tips(&self) -> Vec<NumHash> {
        let mut tips = Vec::new();
        for candidate in &self.candidates {
            let tip = self.settled_tip(candidate);
            if !tips.contains(&tip) {
                tips.push(tip);
            }
        }
        tips
    }

    /// Whether `tip` lies on some candidate's way to settling: the candidate's tip, or a
    /// canonical block between it and the candidate's reach. The sync loop only moves forward
    /// along the canonical chain, so a tip off every path is a wrong engine state at once.
    pub(super) fn on_path(&self, tip: NumHash) -> bool {
        let head = self.head();
        self.candidates.iter().any(|candidate| {
            tip == candidate.tip ||
                (self.is_canonical(tip) &&
                    (candidate.tip.number..=self.reach(candidate, head))
                        .contains(&tip.number))
        })
    }

    /// Keeps the candidates that settle on `observed` and pins them to it.
    pub(super) fn observe_settled(&mut self, observed: NumHash) -> eyre::Result<()> {
        let head = self.head();
        let settled: Vec<EngineCandidate> = self
            .candidates
            .iter()
            .filter(|candidate| self.settled_tip(candidate) == observed)
            .map(|candidate| EngineCandidate {
                target: candidate.target.min(head).max(observed.number),
                tip: observed,
            })
            .collect();
        eyre::ensure!(
            !settled.is_empty(),
            "the engine settled at {observed:?}, which none of {:?} predicts",
            self.candidates
        );
        self.replace_candidates(settled);
        Ok(())
    }

    /// The engine replied `reply` to `block` from a live notification. It buffers a block that
    /// is the next one and builds on its tip, on the execute path only while the parent's state
    /// is still in reth; it refuses the next block when the block does not build on its tip,
    /// naming the tip; it raises its sync target for a block beyond the next one; it ignores a
    /// block at or below its tip.
    pub(super) fn deliver_block(
        &mut self,
        block: NumHash,
        parent: B256,
        parent_available: bool,
        reply: Reply,
    ) -> eyre::Result<()> {
        let consistent: Vec<EngineCandidate> = self
            .candidates
            .iter()
            .flat_map(|candidate| self.deliveries(candidate, block, parent, parent_available))
            .filter(|(_, predicted)| *predicted == reply)
            .map(|(candidate, _)| candidate)
            .collect();
        eyre::ensure!(
            !consistent.is_empty(),
            "the engine replied {reply:?} to block {} on parent {parent}, which no state it \
             could be in allows: {:?}",
            block.number,
            self.candidates
        );
        self.replace_candidates(consistent);
        Ok(())
    }

    /// What `candidate` becomes on `block`, with the reply each outcome gives, for every tip the
    /// candidate's sync loop may have reached by then.
    fn deliveries(
        &self,
        candidate: &EngineCandidate,
        block: NumHash,
        parent: B256,
        parent_available: bool,
    ) -> Vec<(EngineCandidate, Reply)> {
        let number = block.number;
        let lowest = candidate.tip.number;
        if number <= lowest {
            return vec![(*candidate, Reply::Taken)];
        }
        let reach = self.reach(candidate, self.head());
        let mut next = Vec::new();
        if lowest < number - 1 {
            let target = candidate.target.max(number);
            next.push((EngineCandidate { target, ..*candidate }, Reply::Taken));
        }
        if number - 1 <= reach {
            let tip_below =
                if lowest == number - 1 { candidate.tip } else { self.num_hash(number - 1) };
            if tip_below.hash == parent {
                let tip = if parent_available { block } else { tip_below };
                next.push((EngineCandidate { target: candidate.target, tip }, Reply::Taken));
            } else {
                let refused = EngineCandidate { target: candidate.target, tip: tip_below };
                next.push((refused, Reply::Refused { tip_hash: tip_below.hash }));
            }
        }
        if reach >= number {
            next.push((*candidate, Reply::Taken));
        }
        next
    }

    /// The engine replied `Ok` to a reorg from `first` to a chain ending at `last`, delivered
    /// after reth switched to it while its head stood at `head_before`.
    pub(super) fn reorg(&mut self, first: u64, last: NumHash, head_before: u64) {
        self.unwind_from(first, last, head_before);
    }

    /// The engine replied `Ok` to a revert of the blocks from `first` up, delivered after reth
    /// dropped them while its head stood at `head_before`.
    pub(super) fn revert(&mut self, first: u64, head_before: u64) {
        self.unwind_from(first, self.num_hash(first - 1), head_before);
    }

    /// The engine skipped the unwind where `first` was above its tip; elsewhere the store and
    /// the buffer unwound to the block below `first` and the tip is `tip_after`.
    fn unwind_from(&mut self, first: u64, tip_after: NumHash, head_before: u64) {
        let widest_head = head_before.max(self.head());
        let mut next = Vec::new();
        for candidate in &self.candidates {
            if candidate.tip.number < first {
                next.push(*candidate);
            }
            if self.reach(candidate, widest_head) >= first {
                next.push(EngineCandidate {
                    target: candidate.target.min(first - 1),
                    tip: tip_after,
                });
            }
        }
        self.replace_candidates(next);
        self.known_from = self.known_from.min(first);
        self.persisted_at_least = self.persisted_at_least.min(first - 1);
    }

    pub(super) fn sync_to(&mut self, target: u64) {
        for candidate in &mut self.candidates {
            candidate.target = candidate.target.max(target);
        }
    }

    /// The engine's reply to a reorg at `first` whose first block builds on no block it holds:
    /// refused where its tip was at or above `first`, which takes a candidate reaching that far,
    /// and skipped where its tip was below.
    pub(super) fn bogus_reorg(&mut self, first: u64, refused: bool) -> eyre::Result<()> {
        let head = self.head();
        let consistent: Vec<EngineCandidate> = self
            .candidates
            .iter()
            .copied()
            .filter(|candidate| {
                if refused {
                    self.reach(candidate, head) >= first
                } else {
                    candidate.tip.number < first
                }
            })
            .collect();
        eyre::ensure!(
            !consistent.is_empty(),
            "the engine {} a reorg at block {first} on a parent no block has, which no state it \
             could be in allows: {:?}",
            if refused { "refused" } else { "skipped" },
            self.candidates
        );
        self.replace_candidates(consistent);
        Ok(())
    }

    /// A fresh engine over the store reports `tip`, which the store had persisted: nothing is
    /// buffered and nothing below the tip is known.
    pub(super) fn restart(&mut self, tip: NumHash) -> eyre::Result<()> {
        eyre::ensure!(
            tip.number >= self.persisted_at_least,
            "a fresh engine reports tip {tip:?} below the persisted bound {}",
            self.persisted_at_least
        );
        eyre::ensure!(
            self.highest_reach().is_some_and(|reach| tip.number <= reach),
            "a fresh engine reports tip {tip:?} above anything the old engine could have held \
             ({:?})",
            self.candidates
        );
        self.candidates = vec![EngineCandidate { target: tip.number, tip }];
        self.known_from = tip.number + 1;
        self.persisted_at_least = tip.number;
        Ok(())
    }

    fn replace_candidates(&mut self, next: Vec<EngineCandidate>) {
        let mut unique: Vec<EngineCandidate> = Vec::with_capacity(next.len());
        for candidate in next {
            if !unique.contains(&candidate) {
                unique.push(candidate);
            }
        }
        assert!(!unique.is_empty(), "a step left the engine with no possible state");
        self.candidates = unique;
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use rstest::rstest;

    const WINDOW: u64 = 1000;

    /// The canonical block at `number`: hash and parent are the block numbers.
    fn canonical(number: u64) -> NumHash {
        NumHash::new(number, B256::with_last_byte(number as u8))
    }

    /// A block at `number` on an abandoned chain.
    fn stale(number: u64) -> NumHash {
        NumHash::new(number, B256::with_last_byte(0xA0 + number as u8))
    }

    fn block(number: u64) -> ModelBlock {
        ModelBlock {
            hash: canonical(number).hash,
            parent: canonical(number.saturating_sub(1)).hash,
            persisted_before_commit: 0,
            serial: number,
            state_root: B256::ZERO,
        }
    }

    /// A chain through `head` with the engine in `candidates`.
    fn model(head: u64, candidates: &[EngineCandidate]) -> Model {
        let mut model = Model::new(block(0), WINDOW);
        for number in 1..=head {
            model.push(block(number));
        }
        model.candidates = candidates.to_vec();
        model
    }

    const fn at(tip: NumHash, target: u64) -> EngineCandidate {
        EngineCandidate { target, tip }
    }

    #[rstest]
    #[case::at_or_below_the_tip(3, &[at(canonical(3), 3)], 2, true, &[at(canonical(3), 3)])]
    #[case::next_block(4, &[at(canonical(3), 3)], 4, true, &[at(canonical(4), 3)])]
    #[case::next_block_without_parent_state(4, &[at(canonical(3), 3)], 4, false, &[at(canonical(3), 3)])]
    #[case::beyond_the_next_block(5, &[at(canonical(3), 3)], 5, true, &[at(canonical(3), 5)])]
    #[case::while_syncing(7, &[at(canonical(3), 6)], 7, true, &[at(canonical(3), 7), at(canonical(7), 6)])]
    fn deliver_block_taken(
        #[case] head: u64,
        #[case] before: &[EngineCandidate],
        #[case] number: u64,
        #[case] parent_available: bool,
        #[case] after: &[EngineCandidate],
    ) {
        let mut model = model(head, before);
        model
            .deliver_block(
                canonical(number),
                canonical(number - 1).hash,
                parent_available,
                Reply::Taken,
            )
            .unwrap();
        assert_eq!(model.candidates(), after);
    }

    #[rstest]
    #[case::refused_citing_the_tip(Reply::Refused { tip_hash: stale(3).hash }, true)]
    #[case::refused_citing_another_block(Reply::Refused { tip_hash: canonical(3).hash }, false)]
    #[case::taken(Reply::Taken, false)]
    fn deliver_block_on_a_stale_tip(#[case] reply: Reply, #[case] consistent: bool) {
        let mut model = model(4, &[at(stale(3), 3)]);
        let result = model.deliver_block(canonical(4), canonical(3).hash, true, reply);
        assert_eq!(result.is_ok(), consistent, "{result:?}");
    }

    #[rstest]
    #[case::skipped_below_the_tip(&[at(canonical(2), 2)], &[at(canonical(2), 2)])]
    #[case::applied_at_the_tip(&[at(canonical(5), 5)], &[at(canonical(5), 3)])]
    #[case::either_while_syncing(&[at(canonical(2), 5)], &[at(canonical(2), 5), at(canonical(5), 3)])]
    fn reorg_from_block_four(
        #[case] before: &[EngineCandidate],
        #[case] after: &[EngineCandidate],
    ) {
        let mut model = model(5, before);
        model.reorg(4, canonical(5), 5);
        assert_eq!(model.candidates(), after);
    }

    #[test]
    fn revert_from_block_four_moves_the_tip_below_it() {
        let mut model = model(3, &[at(canonical(5), 5)]);
        model.revert(4, 5);
        assert_eq!(model.candidates(), &[at(canonical(3), 3)]);
    }

    #[test]
    fn settling_keeps_the_candidates_that_reach_the_observed_tip() {
        let mut model = model(7, &[at(canonical(3), 7), at(canonical(7), 6)]);
        assert_eq!(model.settled_tips(), vec![canonical(7)]);
        assert!(model.on_path(canonical(5)));
        assert!(!model.on_path(stale(5)));
        model.observe_settled(canonical(7)).unwrap();
        assert_eq!(model.candidates(), &[at(canonical(7), 7)]);
        assert!(model.observe_settled(canonical(6)).is_err());
    }

    #[rstest]
    #[case::at_the_tip(3, true)]
    #[case::above_the_reach(4, false)]
    #[case::below_the_persisted_bound(1, false)]
    fn restart_bounds(#[case] tip: u64, #[case] consistent: bool) {
        let mut model = model(3, &[at(canonical(3), 3)]);
        model.note_persisted(2);
        assert_eq!(model.restart(canonical(tip)).is_ok(), consistent);
    }

    /// A chain through `head` under `window`, whose block `n` was committed with reth persisted
    /// through `min(n, persisted)`.
    fn model_with_reth_persisted(window: u64, head: u64, persisted: u64) -> Model {
        let mut model = Model::new(block(0), window);
        for number in 1..=head {
            let persisted_before_commit = persisted.min(number);
            model.push(ModelBlock { persisted_before_commit, ..block(number) });
        }
        model
    }

    #[rstest]
    #[case::whole_window(WINDOW, 7, 7, 7, 0..=0)]
    #[case::reth_persisted_through_the_prune_target(4, 8, 8, 8, 4..=4)]
    #[case::reth_persisted_below_the_prune_target(4, 8, 8, 3, 3..=4)]
    #[case::latest_unwound_below_the_highest_head(4, 8, 5, 8, 1..=4)]
    #[case::latest_within_a_window_of_genesis(4, 8, 2, 8, 0..=4)]
    fn earliest_bounds(
        #[case] window: u64,
        #[case] head: u64,
        #[case] latest: u64,
        #[case] persisted: u64,
        #[case] expected: RangeInclusive<u64>,
    ) {
        let model = model_with_reth_persisted(window, head, persisted);
        assert_eq!(model.earliest_bounds(latest), expected);
    }

    #[test]
    fn earliest_bounds_keep_the_highest_head_through_a_reorg() {
        let mut model = model_with_reth_persisted(4, 8, 5);
        model.truncate(5);
        model.push(ModelBlock { persisted_before_commit: 5, ..block(6) });
        assert_eq!(model.earliest_bounds(6), 2..=4);
    }
}
