//! What a step can do to reth, the ExEx and the node, and the configuration a run samples.

use super::{FlushCadence, WHOLE_WINDOW, simulated_reth::RethPersistence};
use proptest::{collection::vec, prelude::*, sample::select};

/// Steps per case at most.
const MAX_STEPS: usize = 24;

/// A proof window the chains here outgrow, so that pruning moves the earliest block.
const PRUNED_WINDOW: u64 = 4;

/// How reth rebuilds the blocks a crash lost: from the same consensus-layer chain, as a verifier
/// does, or as different blocks, as a sequencer does.
#[derive(Clone, Copy, Debug)]
pub(crate) enum Rebuild {
    Different,
    Same,
}

#[derive(Clone, Copy, Debug)]
pub(crate) enum Action {
    /// A one-block reorg on a parent no block has, at a fork point the engine knows, which it
    /// must refuse.
    BogusReorg { offset: u64 },
    /// reth commits `count` blocks, each delivered to the engine as a live notification.
    Commit { count: u64 },
    /// reth and the engine stop uncleanly, losing what either held in memory, and restart.
    Crash(Rebuild),
    /// The next save fails at its write transaction, and enough blocks follow to start it.
    FailNextSave,
    /// Waits for the save in flight, if any.
    Flush,
    /// reth replaces its newest `depth` in-memory blocks with `length` new ones and delivers
    /// the reorg.
    Reorg { depth: u64, length: u64 },
    /// reth drops its newest `depth` in-memory blocks and delivers the revert.
    Revert { depth: u64 },
    /// reth commits `depth` blocks and reorgs them away before their notifications reach the
    /// engine, then the reorg's does.
    StaleCommit { depth: u64, length: u64 },
    /// reth commits `count` blocks the engine only learns of through a sync target, as the ExEx
    /// hands over a notification far behind reth's head.
    SyncTo { count: u64 },
}

/// Which extra actions a run may take.
#[derive(Clone, Copy, Debug)]
pub(crate) struct Actions {
    /// A crash after which reth rebuilds a different chain (ethereum-optimism/protocol-team#464).
    pub(crate) crash_rebuilding_a_different_chain: bool,
}

impl Actions {
    pub(crate) const DEFAULT: Self = Self { crash_rebuilding_a_different_chain: false };
    pub(crate) const WITH_TRACKED_BUGS: Self = Self { crash_rebuilding_a_different_chain: true };
}

fn rebuild(actions: Actions) -> BoxedStrategy<Rebuild> {
    if actions.crash_rebuilding_a_different_chain {
        prop_oneof![Just(Rebuild::Same), Just(Rebuild::Different)].boxed()
    } else {
        Just(Rebuild::Same).boxed()
    }
}

fn action(actions: Actions) -> impl Strategy<Value = Action> {
    prop_oneof![
        6 => (1..=4u64).prop_map(|count| Action::Commit { count }),
        3 => (1..=8u64, 1..=4u64).prop_map(|(depth, length)| Action::Reorg { depth, length }),
        2 => (1..=4u64).prop_map(|depth| Action::Revert { depth }),
        3 => (1..=12u64).prop_map(|count| Action::SyncTo { count }),
        3 => Just(Action::Flush),
        1 => Just(Action::FailNextSave),
        2 => (1..=3u64, 1..=3u64).prop_map(|(depth, length)| Action::StaleCommit { depth, length }),
        1 => (0..=8u64).prop_map(|offset| Action::BogusReorg { offset }),
        1 => rebuild(actions).prop_map(Action::Crash),
    ]
}

pub(crate) fn sequence(actions: Actions) -> impl Strategy<Value = Vec<Action>> {
    vec(action(actions), 1..=MAX_STEPS)
}

/// One cell of the matrix.
#[derive(Clone, Copy, Debug)]
pub(crate) struct Config {
    /// reth prunes block bodies through what the ExEx acknowledges, the notification's tip.
    pub(crate) body_pruning: bool,
    pub(crate) cadence: FlushCadence,
    pub(crate) persistence: RethPersistence,
    /// Every block at a multiple of this is executed rather than indexed; 0 indexes every block.
    pub(crate) verification_interval: u64,
    pub(crate) window: u64,
}

impl Config {
    pub(crate) const fn executes(&self, number: u64) -> bool {
        self.verification_interval > 0 && number.is_multiple_of(self.verification_interval)
    }
}

/// reth's persistence settings: the thresholds of develop and of reth 2.6, the latter also with
/// its default masking window, which reth only accepts below the threshold.
fn persistence() -> impl Strategy<Value = RethPersistence> {
    select(vec![7u64, 50]).prop_flat_map(|threshold| {
        let masking = if threshold == 50 { vec![0u64, 30] } else { vec![0u64] };
        select(masking)
            .prop_map(move |masking| RethPersistence::threshold(threshold).masking(masking))
    })
}

pub(crate) fn config() -> impl Strategy<Value = Config> {
    (
        select(vec![FlushCadence::EVERY_BLOCK, FlushCadence::EVERY_FIFTH_BLOCK]),
        persistence(),
        select(vec![0u64, 1, 4]),
        select(vec![WHOLE_WINDOW, PRUNED_WINDOW]),
    )
        .prop_map(|(cadence, persistence, verification_interval, window)| Config {
            body_pruning: false,
            cadence,
            persistence,
            verification_interval,
            window,
        })
}
