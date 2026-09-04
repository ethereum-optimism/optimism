//! Tests for the [`EngineActor`]'s derivation-bound signals.

use crate::{
    EngineActor, EngineActorRequest, EngineError, NodeActor,
    actors::engine::client::MockEngineDerivationClient,
};
use alloy_primitives::B256;
use kona_engine::{
    Engine, EngineState, EngineSyncStateUpdate, NoopBlockSink, test_utils::MockEngineClient,
};
use kona_genesis::RollupConfig;
use kona_protocol::{BlockInfo, L2BlockInfo};
use std::sync::Arc;
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
    state.sync_state = state.sync_state.apply_update(EngineSyncStateUpdate {
        unsafe_head: Some(local_safe),
        local_safe_head: Some(local_safe),
        finalized_head: Some(cross_safe),
    });
    state.sync_state = state.sync_state.apply_cross_safe_promotion(promoter.promote(cross_safe));

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

    let mut derivation_client = MockEngineDerivationClient::new();
    derivation_client
        .expect_send_new_engine_local_safe_head()
        .withf(move |head: &L2BlockInfo| *head == local_safe)
        .times(1)
        .returning(|_| Ok(()));

    let cfg = Arc::new(RollupConfig::default());
    let (request_tx, request_rx) = mpsc::channel::<EngineActorRequest>(1);
    let mut actor = EngineActor::new(
        Arc::new(MockEngineClient::builder().with_config(cfg.clone()).build()),
        cfg,
        derivation_client,
        engine,
        None,
        request_rx,
        Arc::new(NoopBlockSink),
    );

    // The actor drains, pushes the confirmation, then waits for a request; closing the channel
    // ends the step.
    drop(request_tx);
    assert!(matches!(actor.step().await, Err(EngineError::ChannelClosed)));
}
