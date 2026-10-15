//! Model-based test of the proofs engine: random interleavings of commits, reorgs, reverts, sync
//! targets, flushes, failed saves, stale notifications and crashes, driven over reth's
//! persistence model and held to a reference model after every step. The deterministic boundary
//! cases live in the live target; this target is for the interleavings.

use proptest::prelude::*;

#[path = "engine_model/actions.rs"]
mod actions;
#[path = "live/chain.rs"]
#[allow(dead_code, reason = "each test target uses part of this module")]
mod chain;
#[path = "engine_model/harness.rs"]
mod harness;
#[path = "engine_model/invariants.rs"]
mod invariants;
#[path = "engine_model/model.rs"]
mod model;
#[path = "live/simulated_reth.rs"]
#[allow(dead_code, reason = "each test target uses part of this module")]
mod simulated_reth;
#[path = "live/store_hooks.rs"]
#[allow(dead_code, reason = "each test target uses part of this module")]
mod store_hooks;
#[path = "live/store_view.rs"]
#[allow(dead_code, reason = "each test target uses part of this module")]
mod store_view;
#[path = "live/waits.rs"]
#[allow(dead_code, reason = "each test target uses part of this module")]
mod waits;

use actions::{Action, Actions, Config, Rebuild};
use chain::*;
use harness::Harness;
use proptest::test_runner::contextualize_config;
use simulated_reth::RethPersistence;
use std::sync::atomic::{AtomicBool, Ordering};

/// Cases per run unless `PROPTEST_CASES` says otherwise.
const DEFAULT_CASES: u32 = 24;

/// The default count and a shrinking budget that fits the test's time limit, with the
/// `PROPTEST_*` environment variables applied on top.
fn proptest_config() -> ProptestConfig {
    contextualize_config(ProptestConfig {
        cases: DEFAULT_CASES,
        failure_persistence: None,
        max_shrink_iters: 256,
        max_shrink_time: 60_000,
        ..ProptestConfig::default()
    })
}

fn run_sequence(config: Config, actions: &[Action]) -> eyre::Result<()> {
    Harness::new(config)?.run(actions)
}

static FIRST_FAILURE_PRINTED: AtomicBool = AtomicBool::new(false);

/// Runs a case and prints the first failure as found, before shrinking, so that a run cut short
/// by a time limit still shows what failed.
fn run_case(config: Config, actions: &[Action]) -> Result<(), TestCaseError> {
    run_sequence(config, actions).map_err(|err| {
        if !FIRST_FAILURE_PRINTED.swap(true, Ordering::SeqCst) {
            eprintln!("first failing case, before shrinking: {config:?}\n{err:?}\n");
        }
        TestCaseError::fail(format!("{err:?}"))
    })
}

proptest! {
    #![proptest_config(proptest_config())]

    /// Under every cell of the matrix, the engine's tip and the store's window stay on the
    /// canonical chain and within what the model allows after each random step, every valid
    /// event is taken, every reorg on a wrong parent or below the window is refused, and the
    /// engine settles within the step bound.
    #[test]
    fn engine_and_store_follow_the_model(
        config in actions::config(),
        actions in actions::sequence(Actions::DEFAULT),
    ) {
        run_case(config, &actions)?;
    }

    /// As above, with crashes after which reth rebuilds a different chain, as a sequencer does.
    #[test]
    #[ignore = "fails on develop: after an unclean stop the store keeps blocks reth lost, and a chain rebuilt differently leaves the store's latest block off the canonical chain (ethereum-optimism/protocol-team#464; the harness's restart must gain the startup reconcile of #23358 to pass)"]
    fn engine_and_store_follow_the_model_through_crashes_rebuilding_a_different_chain(
        config in actions::config(),
        actions in actions::sequence(Actions::WITH_TRACKED_BUGS),
    ) {
        run_case(config, &actions)?;
    }
}

/// The matrix cell the replayed sequences run under.
fn replay_config(cadence: FlushCadence, persistence_threshold: u64) -> Config {
    Config {
        body_pruning: false,
        cadence,
        persistence: RethPersistence::threshold(persistence_threshold),
        verification_interval: 0,
        window: WHOLE_WINDOW,
    }
}

/// The minimal sequence the random run shrinks to for protocol-team#464: one block saved, then a
/// crash reth comes back from with a different block 1, which the store keeps serving.
#[test]
#[ignore = "fails on develop: the store's latest block 1 is not on the chain reth rebuilt after the crash (ethereum-optimism/protocol-team#464; the harness's restart must gain the startup reconcile of #23358 to pass)"]
fn store_follows_reth_through_a_crash_rebuilding_a_different_chain() -> eyre::Result<()> {
    run_sequence(
        replay_config(FlushCadence::EVERY_BLOCK, 50),
        &[Action::Commit { count: 1 }, Action::Crash(Rebuild::Different)],
    )
}

/// The ExEx acknowledges a notification's tip before the sync loop has executed the bodies
/// below it, so a reth that prunes bodies through the acknowledged height takes them from under
/// the sync loop and the catch-up stalls.
#[test]
#[ignore = "fails on develop: reth prunes the bodies the sync loop still needs because the ExEx acknowledges the notification tip, and the engine never settles (ethereum-optimism/protocol-team#478)"]
fn sync_loop_completes_while_reth_prunes_bodies_through_the_acknowledged_tip() -> eyre::Result<()> {
    let config = Config { body_pruning: true, ..replay_config(FlushCadence::EVERY_BLOCK, 7) };
    run_sequence(config, &[Action::SyncTo { count: 40 }, Action::Commit { count: 1 }])
}
