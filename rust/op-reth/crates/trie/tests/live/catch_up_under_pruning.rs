//! Catches a store up from genesis over a chain longer than the ExEx's real-time threshold while
//! reth prunes the bodies the ExEx has acknowledged.
//!
//! A notification more than `REAL_TIME_BLOCKS_THRESHOLD` blocks behind reth's head makes the ExEx
//! set the engine's sync target to the notification's tip and acknowledge that tip to reth with
//! `FinishedHeight` at once; the blocks reth commits meanwhile reach the engine along the index
//! or execute path, where they raise the sync target while the engine is still behind, and are
//! acknowledged in turn. reth's pruner takes the acknowledged height as the ceiling of what it
//! may remove, so the bodies the sync loop still has to execute can disappear under it. The trie
//! crate cannot see the ExEx, so the cases model its two decisions: the sync target it sets and
//! the height it acknowledges. The pruner model prunes right up to that height and after every
//! acknowledgement, which is more than reth's pruner ever removes, so a case that passes here
//! passes under reth. The stores are MDBX, the backend a node runs; a case's cost is on reth's
//! side, where the proofs backend makes no difference.

use super::{
    BlockSpec, FlushCadence, TxSpec, chain_spec_with_address, create_mdbx_proofs_storage,
    initialize_storage, public_key_to_address,
    simulated_reth::{RethPersistence, SimulatedReth, fixed_signer, index_executed_block},
    store_view::{BlockView, ProofTarget},
};
use alloy_eips::BlockNumHash;
use alloy_primitives::{Address, U256};
use reth_chain_state::ExecutedBlock;
use reth_ethereum_primitives::Block;
use reth_optimism_trie::{
    OpProofsProviderRO, OpProofsStore, ProofWindowRange,
    engine::{EngineError, EngineHandle},
};
use reth_provider::{BlockReader, ProviderError, TransactionVariant};
use rstest::rstest;
use serial_test::serial;
use std::{
    thread,
    time::{Duration, Instant},
};

/// Blocks reth holds before the first notification: more than the ExEx's real-time threshold,
/// and a multiple of both cadences' persistence thresholds so that the engine saves all of them
/// without an idle flush.
const CATCH_UP_LENGTH: u64 = 70;

/// Blocks reth commits while the engine catches up, again a multiple of both thresholds.
const REAL_TIME_LENGTH: u64 = 5;

const _: () = assert!(
    CATCH_UP_LENGTH.is_multiple_of(FlushCadence::EVERY_FIFTH_BLOCK.persistence) &&
        REAL_TIME_LENGTH.is_multiple_of(FlushCadence::EVERY_FIFTH_BLOCK.persistence)
);

/// Blocks per static file of transactions: reth's pruner removes whole files, so this is the
/// step in which bodies disappear. Each file costs reth's persistence a write, and that
/// persistence dominates a case, so the files are as large as the slower cadence's saves.
const BLOCKS_PER_BODY_FILE: u64 = FlushCadence::EVERY_FIFTH_BLOCK.persistence;

/// How long a case lets the engine catch up before calling it stalled.
const CATCH_UP_TIMEOUT: Duration = Duration::from_secs(30);

const POLL_INTERVAL: Duration = Duration::from_millis(5);

/// What the ExEx acknowledges to reth after a notification.
#[derive(Clone, Copy, Debug)]
enum AckPolicy {
    /// The notification's tip, whether or not the store holds it, as the ExEx does today.
    NotificationTip,
    /// The newest block the store has persisted, which a pruner following it never gets ahead of.
    PersistedTip,
}

impl AckPolicy {
    const fn acknowledged(self, notified: u64, persisted: u64) -> u64 {
        match self {
            Self::NotificationTip => notified,
            Self::PersistedTip => persisted,
        }
    }
}

/// How the ExEx hands a block near reth's head to the engine.
#[derive(Clone, Copy, Debug)]
enum ProofsPath {
    Execute,
    Index,
}

impl ProofsPath {
    fn hand_over(
        self,
        engine: &EngineHandle<Block>,
        block: &ExecutedBlock,
    ) -> Result<(), EngineError> {
        match self {
            Self::Execute => engine.execute_block(block.recovered_block()),
            Self::Index => index_executed_block(engine, block),
        }
    }
}

/// reth's bodies pruner, following what the ExEx acknowledges with no distance below it.
struct BodiesPruner {
    policy: AckPolicy,
    /// The newest block whose body is gone.
    pruned_through: Option<u64>,
}

impl BodiesPruner {
    const fn new(policy: AckPolicy) -> Self {
        Self { policy, pruned_through: None }
    }

    /// Prunes what the ExEx's acknowledgement of a notification at `notified`, with the store
    /// persisted through `persisted`, lets the pruner remove.
    fn follow(&mut self, reth: &SimulatedReth, notified: u64, persisted: u64) -> eyre::Result<()> {
        let acknowledged = self.policy.acknowledged(notified, persisted);
        self.pruned_through = self.pruned_through.max(reth.prune_bodies_through(acknowledged)?);
        Ok(())
    }
}

fn recipient(number: u64) -> Address {
    Address::repeat_byte(0x40 + u8::try_from(number).expect("short chain"))
}

/// Block `number` of the chain: `number` wei to the recipient only this block pays, so that every
/// body carries a transaction for pruning to remove.
fn transfer_block(number: u64) -> BlockSpec {
    BlockSpec::new(vec![TxSpec::transfer(recipient(number), U256::from(number))])
}

/// Every address the chain touches, none of which has storage.
fn proof_targets(sender: Address) -> Vec<ProofTarget> {
    [sender, Address::ZERO]
        .into_iter()
        .chain((1..=CATCH_UP_LENGTH + REAL_TIME_LENGTH).map(recipient))
        .map(|address| ProofTarget { address, slots: Vec::new() })
        .collect()
}

fn persisted_tip<S: OpProofsStore>(store: &S) -> eyre::Result<BlockNumHash> {
    Ok(store.provider_ro()?.get_latest_block()?)
}

/// Waits for the store to persist `tip` while the pruner follows the acknowledgement of a
/// notification at `tip`, and fails naming where the store stalled.
fn wait_for_catch_up<S: OpProofsStore>(
    store: &S,
    reth: &SimulatedReth,
    pruner: &mut BodiesPruner,
    tip: BlockNumHash,
) -> eyre::Result<()> {
    let deadline = Instant::now() + CATCH_UP_TIMEOUT;
    loop {
        let persisted = persisted_tip(store)?;
        pruner.follow(reth, tip.number, persisted.number)?;
        if persisted == tip {
            return Ok(());
        }
        eyre::ensure!(
            Instant::now() < deadline,
            "the store stalled at block {} below the tip {} after reth pruned the bodies through \
             block {:?}",
            persisted.number,
            tip.number,
            pruner.pruned_through
        );
        thread::sleep(POLL_INTERVAL);
    }
}

fn run_catch_up_case(
    persistence: RethPersistence,
    cadence: FlushCadence,
    path: ProofsPath,
    policy: AckPolicy,
) -> eyre::Result<()> {
    let key_pair = fixed_signer();
    let sender = public_key_to_address(key_pair.public_key());
    let mut reth = SimulatedReth::with_body_files_of(
        chain_spec_with_address(sender),
        key_pair,
        persistence,
        BLOCKS_PER_BODY_FILE,
    )?;
    let store = create_mdbx_proofs_storage();
    initialize_storage(store.clone(), &reth.factory, reth.genesis())?;
    for number in 1..=CATCH_UP_LENGTH {
        reth.commit(&transfer_block(number))?;
    }
    let engine = reth.spawn_engine(store.clone(), cadence.persistence, cadence.backpressure);
    let mut pruner = BodiesPruner::new(policy);

    // The first notification is the whole chain, which the ExEx hands over as a sync target and
    // acknowledges at once.
    let notified = reth.head().number;
    engine.sync_to(notified)?;
    pruner.follow(&reth, notified, persisted_tip(&store)?.number)?;
    // reth keeps committing while the engine catches up; each block reaches the engine along the
    // path and is acknowledged in turn.
    for number in CATCH_UP_LENGTH + 1..=CATCH_UP_LENGTH + REAL_TIME_LENGTH {
        let block = reth.commit(&transfer_block(number))?;
        path.hand_over(&engine, &block)?;
        pruner.follow(&reth, number, persisted_tip(&store)?.number)?;
    }
    let tip = reth.head();

    wait_for_catch_up(&store, &reth, &mut pruner, tip)?;

    eyre::ensure!(pruner.pruned_through.is_some(), "reth pruned no bodies");
    let window = store.provider_ro()?.get_proof_window()?;
    assert_eq!(window, ProofWindowRange { earliest: reth.genesis(), latest: tip }, "proof window");
    let targets = proof_targets(sender);
    for number in [1, CATCH_UP_LENGTH, tip.number] {
        BlockView::read(&store, number, &targets)?
            .verify_proofs(number, reth.state_root(number))?;
    }
    Ok(())
}

/// reth refuses to serve a block below its lowest remaining transactions file, so the sync loop
/// meets a pruned body as a `BlockExpired` error rather than as an empty body.
#[test]
#[serial]
fn pruned_blocks_read_back_as_expired() -> eyre::Result<()> {
    let key_pair = fixed_signer();
    let sender = public_key_to_address(key_pair.public_key());
    let persistence = RethPersistence::threshold(7);
    let mut reth = SimulatedReth::with_body_files_of(
        chain_spec_with_address(sender),
        key_pair,
        persistence,
        1,
    )?;
    // The first persistence cycle writes blocks 1 through 3.
    for number in 1..=persistence.persistence_threshold + 1 {
        reth.commit(&transfer_block(number))?;
    }

    assert_eq!(reth.prune_bodies_through(2)?, Some(2));

    let err = reth
        .provider
        .recovered_block(1.into(), TransactionVariant::NoHash)
        .expect_err("block 1 is below the lowest remaining file");
    assert!(
        matches!(err, ProviderError::BlockExpired { requested: 1, earliest_available: 3 }),
        "got {err:?}"
    );
    assert!(reth.provider.recovered_block(3.into(), TransactionVariant::NoHash)?.is_some());
    Ok(())
}

/// With the store's persisted tip as the pruner's ceiling, pruning never outruns the sync loop
/// and the store reaches the tip under either reth threshold, cadence and path.
#[rstest]
#[serial]
fn catch_up_completes_when_reth_prunes_bodies_behind_the_persisted_tip(
    #[values(RethPersistence::threshold(7), RethPersistence::threshold(50))]
    persistence: RethPersistence,
    #[values(FlushCadence::EVERY_BLOCK, FlushCadence::EVERY_FIFTH_BLOCK)] cadence: FlushCadence,
    #[values(ProofsPath::Execute, ProofsPath::Index)] path: ProofsPath,
) -> eyre::Result<()> {
    run_catch_up_case(persistence, cadence, path, AckPolicy::PersistedTip)
}

/// With the notification's tip as the pruner's ceiling, which is what the ExEx acknowledges
/// today, reth prunes bodies the sync loop has not executed yet and the catch-up stalls
/// (ethereum-optimism/protocol-team#478).
#[rstest]
#[serial]
#[ignore = "fails on develop: the ExEx acknowledges the notification tip before the sync loop has executed the bodies below it, so reth prunes them and the catch-up stalls (ethereum-optimism/protocol-team#478)"]
fn catch_up_completes_when_reth_prunes_bodies_through_the_acknowledged_tip(
    #[values(RethPersistence::threshold(7), RethPersistence::threshold(50))]
    persistence: RethPersistence,
    #[values(FlushCadence::EVERY_BLOCK, FlushCadence::EVERY_FIFTH_BLOCK)] cadence: FlushCadence,
    #[values(ProofsPath::Execute, ProofsPath::Index)] path: ProofsPath,
) -> eyre::Result<()> {
    run_catch_up_case(persistence, cadence, path, AckPolicy::NotificationTip)
}
