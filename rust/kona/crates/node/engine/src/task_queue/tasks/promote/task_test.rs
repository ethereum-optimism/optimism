//! Lagging-cross tests: the cross-safe head trails local-safe because promotion is withheld.
//!
//! Standalone kona-node always has local-safe == cross-safe, so it cannot catch a head being
//! classified as the wrong one. Withholding promotion is the only shape that can.

use crate::{
    CrossSafePromoter, Engine, EngineState, EngineSyncStateUpdate, EngineTaskExt, LocalSafeHead,
    PromoteCrossSafeTask, SynchronizeTask,
    test_utils::{MockEngineClient, test_engine_client_builder},
};
use alloy_primitives::B256;
use alloy_rpc_types_engine::{
    ForkchoiceState, ForkchoiceUpdated, PayloadStatus, PayloadStatusEnum,
};
use kona_genesis::RollupConfig;
use kona_protocol::{BlockInfo, L2BlockInfo};
use std::sync::Arc;
use tokio::sync::watch;

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

fn client(cfg: Arc<RollupConfig>) -> Arc<MockEngineClient> {
    Arc::new(
        test_engine_client_builder()
            .with_config(cfg)
            .with_fork_choice_updated_v3_response(ForkchoiceUpdated::new(PayloadStatus::new(
                PayloadStatusEnum::Valid,
                None,
            )))
            .build(),
    )
}

/// Builds an engine whose cross-safe head is fed only by promotions, seeded with every head at
/// `genesis`, and returns its state plus the unique promoter.
fn externally_promoted(genesis: L2BlockInfo) -> (EngineState, CrossSafePromoter) {
    let (state_tx, _state_rx) = watch::channel(EngineState::default());
    let (len_tx, _len_rx) = watch::channel(0usize);
    let (engine, promoter) = Engine::<MockEngineClient>::with_external_cross_safe(
        EngineState::default(),
        state_tx,
        len_tx,
    );

    let mut state = *engine.state();
    state.sync_state = state.sync_state.apply_update(EngineSyncStateUpdate {
        unsafe_head: Some(genesis),
        local_safe_head: Some(LocalSafeHead::unpaired(genesis)),
        finalized_head: Some(genesis),
    });
    state.sync_state = state.sync_state.apply_cross_safe_promotion(promoter.promote(genesis));

    (state, promoter)
}

async fn advance_local_safe(
    client: Arc<MockEngineClient>,
    cfg: Arc<RollupConfig>,
    state: &mut EngineState,
    head: L2BlockInfo,
) {
    SynchronizeTask::new(
        client,
        cfg,
        EngineSyncStateUpdate {
            unsafe_head: Some(head),
            local_safe_head: Some(LocalSafeHead::unpaired(head)),
            ..EngineSyncStateUpdate::NONE
        },
    )
    .execute(state)
    .await
    .unwrap();
}

fn last_fcu(states: &[ForkchoiceState]) -> ForkchoiceState {
    *states.last().expect("no forkchoice update was issued")
}

#[tokio::test]
async fn local_safe_advances_while_withheld_promotion_holds_the_fcu_safe_label() {
    let cfg = Arc::new(RollupConfig::default());
    let client = client(cfg.clone());
    let (genesis, b1, b2) = (block(0), block(1), block(2));
    let (mut state, _promoter) = externally_promoted(genesis);

    advance_local_safe(client.clone(), cfg.clone(), &mut state, b1).await;
    advance_local_safe(client.clone(), cfg.clone(), &mut state, b2).await;

    assert_eq!(state.sync_state.local_safe_head(), b2);
    assert_eq!(state.sync_state.cross_safe_head(), genesis);

    let fcu = last_fcu(&client.fork_choice_states().await);
    assert_eq!(fcu.head_block_hash, b2.block_info.hash);
    assert_eq!(fcu.safe_block_hash, genesis.block_info.hash);
}

#[tokio::test]
async fn promotion_moves_the_fcu_safe_label_without_touching_local_safe() {
    let cfg = Arc::new(RollupConfig::default());
    let client = client(cfg.clone());
    let (genesis, b1, b2) = (block(0), block(1), block(2));
    let (mut state, promoter) = externally_promoted(genesis);

    advance_local_safe(client.clone(), cfg.clone(), &mut state, b1).await;
    advance_local_safe(client.clone(), cfg.clone(), &mut state, b2).await;

    PromoteCrossSafeTask::new(client.clone(), cfg.clone(), promoter.promote(b1))
        .execute(&mut state)
        .await
        .unwrap();

    assert_eq!(state.sync_state.cross_safe_head(), b1);
    assert_eq!(state.sync_state.local_safe_head(), b2, "promotion must not move local-safe");

    let fcu = last_fcu(&client.fork_choice_states().await);
    assert_eq!(fcu.head_block_hash, b2.block_info.hash);
    assert_eq!(fcu.safe_block_hash, b1.block_info.hash);
}

#[tokio::test]
async fn promotion_moves_backwards_but_is_clamped_at_the_finalized_head() {
    let cfg = Arc::new(RollupConfig::default());
    let client = client(cfg.clone());
    let (genesis, b1, b2, b3) = (block(0), block(1), block(2), block(3));
    let (mut state, promoter) = externally_promoted(genesis);

    advance_local_safe(client.clone(), cfg.clone(), &mut state, b3).await;
    PromoteCrossSafeTask::new(client.clone(), cfg.clone(), promoter.promote(b3))
        .execute(&mut state)
        .await
        .unwrap();

    // A rewind lowers cross-safe: backward promotions are legal above the finalized head.
    PromoteCrossSafeTask::new(client.clone(), cfg.clone(), promoter.promote(b2))
        .execute(&mut state)
        .await
        .unwrap();
    assert_eq!(state.sync_state.cross_safe_head(), b2);

    SynchronizeTask::new(
        client.clone(),
        cfg.clone(),
        EngineSyncStateUpdate { finalized_head: Some(b1), ..EngineSyncStateUpdate::NONE },
    )
    .execute(&mut state)
    .await
    .unwrap();

    PromoteCrossSafeTask::new(client.clone(), cfg.clone(), promoter.promote(genesis))
        .execute(&mut state)
        .await
        .unwrap();

    assert_eq!(state.sync_state.cross_safe_head(), b1, "clamped at the finalized head");
    assert_eq!(last_fcu(&client.fork_choice_states().await).safe_block_hash, b1.block_info.hash);
}

#[tokio::test]
async fn standalone_advances_cross_safe_with_local_safe_in_a_single_fcu() {
    let cfg = Arc::new(RollupConfig::default());
    let client = client(cfg.clone());
    let b1 = block(1);

    // `Engine::new` is the standalone construction: no promoter exists, so the cross-safe head is
    // fed from local-safe advancement.
    let (state_tx, _state_rx) = watch::channel(EngineState::default());
    let (len_tx, _len_rx) = watch::channel(0usize);
    let engine = Engine::<MockEngineClient>::new(EngineState::default(), state_tx, len_tx);
    let mut state = *engine.state();

    advance_local_safe(client.clone(), cfg.clone(), &mut state, b1).await;

    assert_eq!(state.sync_state.local_safe_head(), b1);
    assert_eq!(state.sync_state.cross_safe_head(), b1);

    let fcus = client.fork_choice_states().await;
    assert_eq!(fcus.len(), 1, "the local-safe advance must not cost an extra forkchoice update");
    assert_eq!(fcus[0].head_block_hash, b1.block_info.hash);
    assert_eq!(fcus[0].safe_block_hash, b1.block_info.hash);
}

#[tokio::test]
async fn promotion_ahead_of_local_safe_is_held_at_local_safe() {
    let cfg = Arc::new(RollupConfig::default());
    let client = client(cfg.clone());
    let (genesis, b1, b3) = (block(0), block(1), block(3));
    let (mut state, promoter) = externally_promoted(genesis);

    advance_local_safe(client.clone(), cfg.clone(), &mut state, b1).await;

    // The verifier names a block this engine has not derived locally yet. Cross-safe is local-safe
    // *and* cross-verified, so the promotion is held at local-safe. Unclamped, the cross-safe head
    // would move to b3 and the forkchoice update would carry `safeBlockHash` ahead of
    // `headBlockHash`, which the EL rejects with `INVALID_FORK_CHOICE_STATE` — a `Reset` severity
    // that costs a full engine reset.
    PromoteCrossSafeTask::new(client.clone(), cfg.clone(), promoter.promote(b3))
        .execute(&mut state)
        .await
        .unwrap();

    assert_eq!(
        state.sync_state.cross_safe_head(),
        b1,
        "a promotion ahead of local-safe must be held at the local-safe head"
    );

    let fcu = last_fcu(&client.fork_choice_states().await);
    assert_eq!(fcu.head_block_hash, b1.block_info.hash);
    assert_eq!(
        fcu.safe_block_hash, b1.block_info.hash,
        "the forkchoice update must never report safe ahead of head"
    );
}

#[tokio::test]
async fn an_externally_promoted_engine_still_emits_its_initial_forkchoice_update() {
    let cfg = Arc::new(RollupConfig::default());
    let client = client(cfg.clone());

    // A freshly built interop engine: nothing has happened yet, so a synchronize task carries no
    // change. The initial forkchoice update still has to go out.
    let (state_tx, _state_rx) = watch::channel(EngineState::default());
    let (len_tx, _len_rx) = watch::channel(0usize);
    let (engine, _promoter) = Engine::<MockEngineClient>::with_external_cross_safe(
        EngineState::default(),
        state_tx,
        len_tx,
    );
    let mut state = *engine.state();

    SynchronizeTask::new(client.clone(), cfg.clone(), EngineSyncStateUpdate::NONE)
        .execute(&mut state)
        .await
        .unwrap();

    // The skip-if-unchanged guard used to read `state.sync_state != Default::default()` as "the
    // initial forkchoice state has been emitted". `with_external_cross_safe` sets
    // `cross_safe_source` to `Promoted`, which makes that comparison true from birth, so the guard
    // took the early return on the very first task and the initial forkchoice update was never
    // emitted for exactly the interop configuration that needs it.
    assert_eq!(
        client.fork_choice_states().await.len(),
        1,
        "the initial forkchoice update must be emitted even though the update carries no change"
    );
    assert!(state.forkchoice_emitted, "the initial emission must be recorded");
}

#[tokio::test]
async fn a_reset_below_the_cross_safe_head_holds_cross_safe_at_the_walkback_point() {
    let cfg = Arc::new(RollupConfig::default());
    let client = client(cfg.clone());
    let (genesis, b2, b5) = (block(0), block(2), block(5));
    let (mut state, promoter) = externally_promoted(genesis);

    advance_local_safe(client.clone(), cfg.clone(), &mut state, b5).await;
    PromoteCrossSafeTask::new(client.clone(), cfg.clone(), promoter.promote(b5))
        .execute(&mut state)
        .await
        .unwrap();
    assert_eq!(state.sync_state.cross_safe_head(), b5);

    // The update `Engine::reset` applies: `find_starting_forkchoice` walks the local-safe head
    // back to the last block a full sequencing window behind the unsafe head, so it lands below
    // the cross-safe head an external verifier had already promoted. No promotion accompanies the
    // reset, so nothing but `apply_update` itself can hold the cross-safe head down.
    SynchronizeTask::new(
        client.clone(),
        cfg.clone(),
        EngineSyncStateUpdate {
            unsafe_head: Some(b2),
            local_safe_head: Some(LocalSafeHead::unpaired(b2)),
            finalized_head: Some(genesis),
        },
    )
    .execute(&mut state)
    .await
    .unwrap();

    assert_eq!(
        state.sync_state.cross_safe_head(),
        b2,
        "a rewind below the cross-safe head must hold it at the rewound local-safe head"
    );

    let fcu = last_fcu(&client.fork_choice_states().await);
    assert_eq!(fcu.head_block_hash, b2.block_info.hash);
    assert_eq!(
        fcu.safe_block_hash, b2.block_info.hash,
        "the forkchoice update must never report safe ahead of head"
    );
}
