use super::synchronize;
use crate::{
    EngineState, EngineSyncStateUpdate, EngineTaskError, EngineTaskErrorSeverity, ForkchoiceStatus,
    SynchronizeTaskError,
    test_utils::{TestEngineStateBuilder, test_block_info, test_engine_client},
};
use alloy_rpc_types_engine::{ForkchoiceUpdated, PayloadStatus, PayloadStatusEnum};
use kona_genesis::RollupConfig;
use std::sync::Arc;

fn response(status: PayloadStatusEnum) -> ForkchoiceUpdated {
    ForkchoiceUpdated::new(PayloadStatus::from_status(status))
}

#[tokio::test]
async fn syncing_retries_identical_target_until_valid() {
    let (client, l1, l2) = test_engine_client(Arc::new(RollupConfig::default()));
    let mut state = TestEngineStateBuilder::new()
        .with_unsafe_head(test_block_info(20))
        .with_safe_head(test_block_info(0))
        .with_finalized_head(test_block_info(0))
        .build();
    let update =
        EngineSyncStateUpdate { safe_head: Some(test_block_info(10)), ..Default::default() };
    let target = state.sync_state.apply_update(update).create_forkchoice_state();
    for status in [PayloadStatusEnum::Syncing, PayloadStatusEnum::Valid] {
        l2.expect_params(
            "engine_forkchoiceUpdatedV3",
            serde_json::json!([target, null]),
            response(status),
        );
    }
    synchronize(&client, &mut state, update).await.unwrap();
    assert_eq!(state.forkchoice_status, ForkchoiceStatus::Syncing(target));
    synchronize(&client, &mut state, update).await.unwrap();
    assert_eq!(state.forkchoice_status, ForkchoiceStatus::Applied(target));
    synchronize(&client, &mut state, update).await.unwrap();
    l1.assert_finished();
    l2.assert_finished();
}

#[tokio::test]
async fn changed_target_supersedes_pending_forkchoice() {
    let (client, l1, l2) = test_engine_client(Arc::new(RollupConfig::default()));
    let mut state = TestEngineStateBuilder::new()
        .with_unsafe_head(test_block_info(20))
        .with_safe_head(test_block_info(0))
        .with_finalized_head(test_block_info(0))
        .build();
    for (safe, status) in [(10, PayloadStatusEnum::Syncing), (15, PayloadStatusEnum::Valid)] {
        let update =
            EngineSyncStateUpdate { safe_head: Some(test_block_info(safe)), ..Default::default() };
        let target = state.sync_state.apply_update(update).create_forkchoice_state();
        l2.expect_params(
            "engine_forkchoiceUpdatedV3",
            serde_json::json!([target, null]),
            response(status),
        );
        synchronize(&client, &mut state, update).await.unwrap();
    }
    let target = state.sync_state.create_forkchoice_state();
    assert_eq!(state.forkchoice_status, ForkchoiceStatus::Applied(target));
    synchronize(&client, &mut state, EngineSyncStateUpdate::default()).await.unwrap();
    l1.assert_finished();
    l2.assert_finished();
}

#[tokio::test]
async fn default_state_is_sent_once_and_local_metadata_does_not_repeat_fcu() {
    let (client, l1, l2) = test_engine_client(Arc::new(RollupConfig::default()));
    let mut state = EngineState::default();
    assert_eq!(state.forkchoice_status, ForkchoiceStatus::Unsent);
    l2.expect("engine_forkchoiceUpdatedV3", response(PayloadStatusEnum::Valid));
    synchronize(&client, &mut state, EngineSyncStateUpdate::default()).await.unwrap();
    synchronize(&client, &mut state, EngineSyncStateUpdate::default()).await.unwrap();
    let local_safe = test_block_info(1);
    synchronize(
        &client,
        &mut state,
        EngineSyncStateUpdate { local_safe_head: Some(local_safe), ..Default::default() },
    )
    .await
    .unwrap();
    assert_eq!(state.sync_state.local_safe_head(), local_safe);
    l1.assert_finished();
    l2.assert_finished();
}

#[tokio::test]
async fn invalid_forkchoice_is_not_acknowledged_as_syncing() {
    let (client, l1, l2) = test_engine_client(Arc::new(RollupConfig::default()));
    let mut state = EngineState::default();
    l2.expect_error_code("engine_forkchoiceUpdatedV3", -38002);
    let err = synchronize(&client, &mut state, EngineSyncStateUpdate::default()).await.unwrap_err();
    assert!(matches!(err, SynchronizeTaskError::InvalidForkchoiceState));
    assert_eq!(err.severity(), EngineTaskErrorSeverity::Reset);
    assert_eq!(state.forkchoice_status, ForkchoiceStatus::Unsent);
    l1.assert_finished();
    l2.assert_finished();
}

#[tokio::test]
async fn applied_forkchoice_marks_sync_complete_without_repeating_fcu() {
    let (client, l1, l2) = test_engine_client(Arc::new(RollupConfig::default()));
    let mut state = TestEngineStateBuilder::new().with_unsafe_head(test_block_info(20)).build();
    state.el_sync_finished = false;
    synchronize(&client, &mut state, EngineSyncStateUpdate::default()).await.unwrap();
    assert!(state.el_sync_finished);
    l1.assert_finished();
    l2.assert_finished();
}
