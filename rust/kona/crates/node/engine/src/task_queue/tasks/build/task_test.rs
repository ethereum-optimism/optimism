use crate::{
    BuildTask, BuildTaskError, EngineBuildError, EngineForkchoiceVersion, EngineState, EngineTask,
    EngineTaskError, EngineTaskErrorSeverity, ExecutableEngineTask,
    test_utils::{
        TestAttributesBuilder, TestEngineStateBuilder, test_block_info, test_engine_client,
    },
};
use alloy_rpc_types_engine::{
    ForkchoiceUpdated, INVALID_FORK_CHOICE_STATE_ERROR, INVALID_PAYLOAD_ATTRIBUTES_ERROR,
    PayloadId, PayloadStatus, PayloadStatusEnum,
};
use kona_genesis::RollupConfig;
use rstest::rstest;
use std::sync::Arc;
use tokio::sync::mpsc;

#[rstest]
#[case::success(Some(PayloadStatusEnum::Valid), true)]
#[case::missing_id(Some(PayloadStatusEnum::Valid), false)]
#[case::rpc_failure(None, false)]
#[case::invalid(Some(PayloadStatusEnum::Invalid {validation_error: String::new()}), false)]
#[case::syncing(Some(PayloadStatusEnum::Syncing), false)]
#[case::unexpected(Some(PayloadStatusEnum::Accepted), false)]
#[tokio::test]
async fn build_reports_outcome_and_does_not_retry_rejected_attributes(
    #[case] status: Option<PayloadStatusEnum>,
    #[case] payload_id_present: bool,
    #[values(EngineForkchoiceVersion::V2, EngineForkchoiceVersion::V3)]
    version: EngineForkchoiceVersion,
) {
    let parent = test_block_info(1);
    let timestamp = parent.block_info.timestamp + 2;
    let payload_id = payload_id_present.then(|| PayloadId::new([1; 8]));
    let mut cfg = RollupConfig::default();
    let method = match version {
        EngineForkchoiceVersion::V2 => {
            cfg.hardforks.ecotone_time = Some(timestamp + 1);
            "engine_forkchoiceUpdatedV2"
        }
        EngineForkchoiceVersion::V3 => {
            cfg.hardforks.ecotone_time = Some(timestamp);
            "engine_forkchoiceUpdatedV3"
        }
    };
    let cfg = Arc::new(cfg);
    let (client, l1, l2) = test_engine_client(cfg.clone());
    if let Some(status) = status.clone() {
        l2.expect(
            method,
            ForkchoiceUpdated { payload_status: PayloadStatus::from_status(status), payload_id },
        );
    } else {
        l2.expect_error(method);
    }
    let (tx, mut rx) = mpsc::channel(1);
    let task = EngineTask::Build(Box::new(BuildTask::new(
        Arc::new(client),
        cfg,
        TestAttributesBuilder::new().with_parent(parent).with_timestamp(timestamp).build(),
        tx,
    )));
    let mut state = TestEngineStateBuilder::new().with_unsafe_head(parent).build();
    let result = task.execute(&mut state).await;
    if matches!(&status, Some(PayloadStatusEnum::Invalid { .. })) {
        assert_eq!(result.unwrap_err().severity(), EngineTaskErrorSeverity::Reset);
    } else {
        result.unwrap();
    }
    match (status, rx.recv().await.unwrap()) {
        (Some(PayloadStatusEnum::Valid), Ok(id)) => assert_eq!(Some(id), payload_id),
        (
            Some(PayloadStatusEnum::Valid),
            Err(BuildTaskError::EngineBuildError(EngineBuildError::MissingPayloadId)),
        ) => assert!(payload_id.is_none()),
        (
            None,
            Err(BuildTaskError::EngineBuildError(EngineBuildError::AttributesInsertionFailed(_))),
        ) |
        (
            Some(PayloadStatusEnum::Invalid { .. }),
            Err(BuildTaskError::EngineBuildError(EngineBuildError::InvalidPayload(_))),
        ) |
        (
            Some(PayloadStatusEnum::Syncing),
            Err(BuildTaskError::EngineBuildError(EngineBuildError::EngineSyncing)),
        ) |
        (
            Some(PayloadStatusEnum::Accepted),
            Err(BuildTaskError::EngineBuildError(EngineBuildError::UnexpectedPayloadStatus(_))),
        ) => {}
        outcome => panic!("unexpected build outcome: {outcome:?}"),
    }
    l1.assert_finished();
    l2.assert_finished();
}

#[rstest]
#[case::different_height(false)]
#[case::same_height_reorg(true)]
#[tokio::test]
async fn stale_build_replies_without_calling_engine(#[case] same_height: bool) {
    let cfg = Arc::new(RollupConfig::default());
    let (client, l1, l2) = test_engine_client(cfg.clone());
    let parent = test_block_info(1);
    let mut head = test_block_info(2);
    if same_height {
        head.block_info.number = parent.block_info.number;
    }
    let (tx, mut rx) = mpsc::channel(1);
    let task = EngineTask::Build(Box::new(BuildTask::new(
        Arc::new(client),
        cfg,
        TestAttributesBuilder::new().with_parent(parent).build(),
        tx,
    )));
    let mut state = TestEngineStateBuilder::new().with_unsafe_head(head).build();
    task.execute(&mut state).await.unwrap();
    assert!(matches!(rx.recv().await.unwrap(), Err(BuildTaskError::StaleBuild)));
    assert_eq!(state.sync_state.unsafe_head(), head);
    l1.assert_finished();
    l2.assert_finished();
}

#[rstest]
#[case::invalid_forkchoice(INVALID_FORK_CHOICE_STATE_ERROR, EngineTaskErrorSeverity::Reset)]
#[case::invalid_attributes(INVALID_PAYLOAD_ATTRIBUTES_ERROR, EngineTaskErrorSeverity::Temporary)]
#[tokio::test]
async fn rpc_error_reaches_caller_and_recovery_owner(
    #[case] code: i32,
    #[case] severity: EngineTaskErrorSeverity,
) {
    let mut cfg = RollupConfig::default();
    cfg.hardforks.ecotone_time = Some(0);
    let cfg = Arc::new(cfg);
    let (client, l1, l2) = test_engine_client(cfg.clone());
    l2.expect_error_code("engine_forkchoiceUpdatedV3", i64::from(code));
    let parent = test_block_info(1);
    let (tx, mut rx) = mpsc::channel(1);
    let task = EngineTask::Build(Box::new(BuildTask::new(
        Arc::new(client),
        cfg,
        TestAttributesBuilder::new().with_parent(parent).build(),
        tx,
    )));
    let mut state = TestEngineStateBuilder::new().with_unsafe_head(parent).build();
    let result = task.execute(&mut state).await;
    if severity == EngineTaskErrorSeverity::Reset {
        assert_eq!(result.unwrap_err().severity(), severity);
    } else {
        result.unwrap();
    }
    let error = rx.recv().await.unwrap().unwrap_err();
    assert_eq!(error.severity(), severity);
    assert_eq!(error.is_invalid_attributes(), code == INVALID_PAYLOAD_ATTRIBUTES_ERROR);
    l1.assert_finished();
    l2.assert_finished();
}

#[tokio::test]
async fn critical_build_error_reaches_caller_and_engine() {
    let cfg = Arc::new(RollupConfig::default());
    let (client, l1, l2) = test_engine_client(cfg.clone());
    let parent = test_block_info(1);
    let (tx, mut rx) = mpsc::channel(1);
    let task = BuildTask::new(
        Arc::new(client),
        cfg,
        TestAttributesBuilder::new().with_parent(parent).build(),
        tx,
    );
    let mut state = TestEngineStateBuilder::new()
        .with_unsafe_head(parent)
        .with_finalized_head(test_block_info(2))
        .build();
    assert_eq!(
        task.execute(&mut state).await.unwrap_err().severity(),
        EngineTaskErrorSeverity::Critical
    );
    assert_eq!(rx.recv().await.unwrap().unwrap_err().severity(), EngineTaskErrorSeverity::Critical);
    l1.assert_finished();
    l2.assert_finished();
}

#[rstest]
#[case::stale(false)]
#[case::accepted(true)]
#[tokio::test]
async fn abandoned_build_reply_does_not_stop_engine(#[case] matching_parent: bool) {
    let mut cfg = RollupConfig::default();
    cfg.hardforks.ecotone_time = Some(0);
    let cfg = Arc::new(cfg);
    let (client, l1, l2) = test_engine_client(cfg.clone());
    let parent = test_block_info(1);
    let mut state = EngineState::default();
    if matching_parent {
        state = TestEngineStateBuilder::new().with_unsafe_head(parent).build();
        l2.expect(
            "engine_forkchoiceUpdatedV3",
            ForkchoiceUpdated {
                payload_status: PayloadStatus::from_status(PayloadStatusEnum::Valid),
                payload_id: Some(PayloadId::new([1; 8])),
            },
        );
    }
    let (tx, rx) = mpsc::channel(1);
    drop(rx);
    let task = BuildTask::new(
        Arc::new(client),
        cfg,
        TestAttributesBuilder::new().with_parent(parent).build(),
        tx,
    );
    task.execute(&mut state).await.unwrap();
    l1.assert_finished();
    l2.assert_finished();
}
