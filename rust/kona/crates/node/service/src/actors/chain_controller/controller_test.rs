//! Tests for the [`ChainController`]'s derivation-bound signals.

use crate::{
    ChainController, ChainControllerError, ChainControllerRequest, NodeActor,
    actors::chain_controller::derivation_client::MockChainControllerDerivationClient,
};
use alloy_eips::BlockNumHash;
use alloy_primitives::B256;
use kona_engine::{
    Engine, EngineState, EngineSyncStateUpdate, LocalSafeHead, test_utils::MockEngineClient,
};
use kona_genesis::RollupConfig;
use kona_protocol::{BlockInfo, L2BlockInfo};
use kona_safedb::{DisabledDatabase, SafeDb, SafeDbError, SafeHeadRecord, SharedSafeDb};
use std::sync::{Arc, Mutex};
use tokio::sync::{mpsc, watch};

fn block(number: u64) -> L2BlockInfo {
    L2BlockInfo {
        block_info: BlockInfo {
            number,
            hash: B256::repeat_byte(number as u8),
            parent_hash: B256::repeat_byte(number.saturating_sub(1) as u8),
            timestamp: number * 2,
        },
        ..Default::default()
    }
}

/// Builds an engine whose local-safe head is at `local_safe` while its cross-safe head is held at
/// `cross_safe` by withholding promotion.
fn lagging_cross_engine(
    local_safe: L2BlockInfo,
    cross_safe: L2BlockInfo,
) -> Engine<MockEngineClient> {
    let (state_tx, _state_rx) = watch::channel(EngineState::default());
    let (len_tx, _len_rx) = watch::channel(0usize);
    let (engine, promoter) = Engine::<MockEngineClient>::with_external_cross_safe(
        EngineState::default(),
        state_tx,
        len_tx,
    );

    let mut state = *engine.state();
    state.sync_state = state.apply_sync_update(EngineSyncStateUpdate {
        unsafe_head: Some(local_safe),
        local_safe_head: Some(LocalSafeHead::unpaired(local_safe)),
        finalized_head: Some(cross_safe),
    });
    state.sync_state = state.apply_cross_safe_promotion(promoter.promote(cross_safe));

    let (state_tx, _state_rx) = watch::channel(state);
    let (len_tx, _len_rx) = watch::channel(0usize);
    Engine::with_external_cross_safe(state, state_tx, len_tx).0
}

/// The depth-1 lockstep confirmation must carry the local-safe head. Feeding it from cross-safe
/// deadlocks under interop, and since both heads are the same type nothing but a lagging-cross
/// test catches the mix-up.
#[tokio::test]
async fn lockstep_confirmation_carries_local_safe_not_cross_safe() {
    let (cross_safe, local_safe) = (block(3), block(7));
    let engine = lagging_cross_engine(local_safe, cross_safe);
    assert_eq!(engine.state().sync_state.cross_safe_head(), cross_safe);

    let mut derivation_client = MockChainControllerDerivationClient::new();
    derivation_client
        .expect_send_new_engine_local_safe_head()
        .withf(move |head: &L2BlockInfo| *head == local_safe)
        .times(1)
        .returning(|_| Ok(()));

    let cfg = Arc::new(RollupConfig::default());
    let (request_tx, request_rx) = mpsc::channel::<ChainControllerRequest>(1);
    let mut actor = ChainController::new(
        Arc::new(MockEngineClient::builder().with_config(cfg.clone()).build()),
        cfg,
        derivation_client,
        engine,
        None,
        request_rx,
        Arc::new(DisabledDatabase),
    );

    // The actor drains, pushes the confirmation, then waits for a request; closing the channel
    // ends the step.
    drop(request_tx);
    assert!(matches!(actor.step().await, Err(ChainControllerError::ChannelClosed)));
}

/// A [`SafeDb`] that remembers what it was asked to do, and can be made to fail.
///
/// The recording half is the point: the contract this controller has to keep — one record per
/// advance, ascending in L1, nothing written for a head with no L1 origin — is about the *calls*,
/// and a real database would answer queries identically whether or not those calls were right.
#[derive(Debug, Default)]
struct RecordingSafeDb {
    enabled: bool,
    fail: bool,
    updates: Mutex<Vec<SafeHeadRecord>>,
    resets: Mutex<Vec<u64>>,
}

impl RecordingSafeDb {
    fn enabled() -> Arc<Self> {
        Arc::new(Self { enabled: true, ..Default::default() })
    }

    fn failing() -> Arc<Self> {
        Arc::new(Self { enabled: true, fail: true, ..Default::default() })
    }

    fn disabled() -> Arc<Self> {
        Arc::new(Self::default())
    }

    fn updates(&self) -> Vec<SafeHeadRecord> {
        self.updates.lock().unwrap().clone()
    }

    fn resets(&self) -> Vec<u64> {
        self.resets.lock().unwrap().clone()
    }
}

impl SafeDb for RecordingSafeDb {
    fn enabled(&self) -> bool {
        self.enabled
    }

    fn safe_head_updated(
        &self,
        safe_head: L2BlockInfo,
        l1_head: BlockNumHash,
    ) -> Result<(), SafeDbError> {
        self.updates
            .lock()
            .unwrap()
            .push(SafeHeadRecord { l1: l1_head, safe_head: safe_head.block_info.id() });
        if self.fail { Err(SafeDbError::NotFound) } else { Ok(()) }
    }

    fn safe_head_reset(&self, safe_head: L2BlockInfo) -> Result<(), SafeDbError> {
        self.resets.lock().unwrap().push(safe_head.block_info.number);
        if self.fail { Err(SafeDbError::NotFound) } else { Ok(()) }
    }

    fn safe_head_at_l1(&self, _l1_block_num: u64) -> Result<SafeHeadRecord, SafeDbError> {
        Err(SafeDbError::NotFound)
    }

    fn first_entry(&self) -> Result<SafeHeadRecord, SafeDbError> {
        Err(SafeDbError::NotFound)
    }

    fn last_entry(&self) -> Result<SafeHeadRecord, SafeDbError> {
        Err(SafeDbError::NotFound)
    }

    fn l1_at_safe_head(&self, _target_l2_num: u64) -> Result<SafeHeadRecord, SafeDbError> {
        Err(SafeDbError::NotFound)
    }

    fn close(&self) -> Result<(), SafeDbError> {
        Ok(())
    }
}

/// An L1 block at `number`.
fn l1(number: u64) -> BlockInfo {
    BlockInfo { number, hash: B256::repeat_byte(0x80 | number as u8), ..Default::default() }
}

/// The pairing of the L2 block at `number` with the L1 block at `origin`.
fn paired(number: u64, origin: u64) -> LocalSafeHead {
    LocalSafeHead::derived_from(block(number), l1(origin))
}

/// A controller that records into `safe_db`, over an engine these tests never step.
///
/// The recorder is a function of the pairing it is handed and of what it has already written, so
/// the engine behind it is irrelevant here; the request channel's sender is dropped for the same
/// reason.
fn recording_controller(
    safe_db: SharedSafeDb,
) -> ChainController<MockEngineClient, MockChainControllerDerivationClient> {
    let cfg = Arc::new(RollupConfig::default());
    let (_request_tx, request_rx) = mpsc::channel::<ChainControllerRequest>(1);
    let (state_tx, _state_rx) = watch::channel(EngineState::default());
    let (len_tx, _len_rx) = watch::channel(0usize);
    ChainController::new(
        Arc::new(MockEngineClient::builder().with_config(cfg.clone()).build()),
        cfg,
        MockChainControllerDerivationClient::new(),
        Engine::new(EngineState::default(), state_tx, len_tx),
        None,
        request_rx,
        safe_db,
    )
}

/// The pairing is what makes the record useful, so a head that has one is recorded — and recorded
/// once, because a drain that observed no advance has nothing new to store.
#[test]
fn a_paired_local_safe_head_is_recorded_exactly_once() {
    let db = RecordingSafeDb::enabled();
    let mut controller = recording_controller(db.clone());

    controller.record_local_safe_head(paired(7, 4));
    controller.record_local_safe_head(paired(7, 4));

    let updates = db.updates();
    assert_eq!(updates.len(), 1, "an unchanged head must not be rewritten");
    assert_eq!(updates[0].safe_head, block(7).block_info.id());
    assert_eq!(updates[0].l1, l1(4).id());
}

/// Two L2 blocks derived from the same L1 block are two records under one key, and the later one
/// has to win: it is the safe head as of that L1 block.
#[test]
fn a_further_l2_block_at_the_same_l1_block_is_recorded() {
    let db = RecordingSafeDb::enabled();
    let mut controller = recording_controller(db.clone());

    controller.record_local_safe_head(paired(7, 4));
    controller.record_local_safe_head(paired(8, 4));

    let updates = db.updates();
    assert_eq!(updates.len(), 2);
    assert_eq!(updates[1].safe_head, block(8).block_info.id());
    assert_eq!(updates[1].l1, l1(4).id());
}

/// A reset walkback and the derivation-delegation path both write heads whose L1 origin they never
/// knew. Recording one would claim it was derived from block 0, which a later history query would
/// hand back as fact.
#[test]
fn an_unpaired_local_safe_head_is_not_recorded() {
    let db = RecordingSafeDb::enabled();
    let mut controller = recording_controller(db.clone());

    controller.record_local_safe_head(LocalSafeHead::unpaired(block(7)));

    assert!(db.updates().is_empty(), "an unpaired head has no L1 key to record it under");
}

/// The database's contract is that records arrive in ascending L1 order. A pairing below the last
/// recorded L1 block is a replay or the tail of a reset, and writing it would put an entry back
/// that a rewind removed on purpose.
#[test]
fn a_pairing_below_the_last_recorded_l1_block_is_not_recorded() {
    let db = RecordingSafeDb::enabled();
    let mut controller = recording_controller(db.clone());

    controller.record_local_safe_head(paired(7, 6));
    controller.record_local_safe_head(paired(8, 3));

    assert_eq!(db.updates().len(), 1, "a rewinding L1 origin must not be recorded");
}

/// A reset disowns every L2 block above its target, and the records naming them go with it. The
/// suppression of already-written pairings is cleared too: the next advance may well be at an L1
/// block the rewind just emptied.
#[test]
fn a_rewind_reaches_the_database_and_clears_the_suppression() {
    let db = RecordingSafeDb::enabled();
    let mut controller = recording_controller(db.clone());
    controller.record_local_safe_head(paired(9, 6));

    controller.rewind_safe_db(block(4));
    assert_eq!(db.resets(), vec![4], "the rewind must reach the database");

    // Re-derivation lands at an L1 block below the one last written, which before the rewind
    // would have been suppressed as non-ascending.
    controller.record_local_safe_head(paired(5, 5));

    let updates = db.updates();
    assert_eq!(updates.len(), 2, "the advance after a rewind must be recorded");
    assert_eq!(updates[1].l1, l1(5).id());
}

/// A write that failed was not stored, so the controller must not remember it as stored: the next
/// drain is the only chance to record that head again.
#[test]
fn a_failed_write_is_retried_on_the_next_drain() {
    let db = RecordingSafeDb::failing();
    let mut controller = recording_controller(db.clone());

    controller.record_local_safe_head(paired(7, 4));
    controller.record_local_safe_head(paired(7, 4));

    assert_eq!(db.updates().len(), 2, "a failed record must not be treated as written");
}

/// A host that does not record derivation pays nothing for the recorder: it does not even reach
/// for the database.
#[test]
fn a_disabled_database_is_never_written_to() {
    let db = RecordingSafeDb::disabled();
    let mut controller = recording_controller(db.clone());

    controller.record_local_safe_head(paired(7, 4));
    controller.rewind_safe_db(block(4));

    assert!(db.updates().is_empty(), "a disabled database records nothing");
    assert!(db.resets().is_empty(), "a disabled database rewinds nothing");
}
