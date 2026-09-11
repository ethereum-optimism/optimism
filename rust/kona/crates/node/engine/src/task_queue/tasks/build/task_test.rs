use crate::{
    BuildSealCoupling, BuildTask, BuildTaskError, EngineBuildError, EngineClient,
    EngineForkchoiceVersion, EngineState, EngineTaskError, EngineTaskErrorSeverity, EngineTaskExt,
    test_utils::{
        MockEngineClientBuilder, TestAttributesBuilder, TestEngineStateBuilder, test_block_info,
        test_engine_client_builder,
    },
};
use alloy_json_rpc::ErrorPayload;
use alloy_primitives::FixedBytes;
use alloy_rpc_types_engine::{
    ForkchoiceUpdated, INVALID_FORK_CHOICE_STATE_ERROR, PayloadId, PayloadStatus, PayloadStatusEnum,
};
use kona_genesis::RollupConfig;
use rstest::rstest;
use std::sync::Arc;
use thiserror::Error;
use tokio::sync::mpsc;

fn fcu_for_payload(payload_id: Option<PayloadId>, status: PayloadStatusEnum) -> ForkchoiceUpdated {
    ForkchoiceUpdated {
        payload_status: PayloadStatus { status, latest_valid_hash: Some(FixedBytes([2u8; 32])) },
        payload_id,
    }
}

fn configure_fcu(
    b: MockEngineClientBuilder,
    fcu_version: EngineForkchoiceVersion,
    fcu_response: ForkchoiceUpdated,
    cfg: &mut RollupConfig,
    attributes_timestamp: u64,
) -> MockEngineClientBuilder {
    match fcu_version {
        EngineForkchoiceVersion::V2 => {
            // Ecotone not yet active
            cfg.hardforks.ecotone_time = Some(attributes_timestamp + 1);
            b.with_fork_choice_updated_v2_response(fcu_response)
        }
        EngineForkchoiceVersion::V3 => {
            // Ecotone is active
            cfg.hardforks.ecotone_time = Some(attributes_timestamp);
            b.with_fork_choice_updated_v3_response(fcu_response)
        }
    }
}

#[derive(Debug, Error, PartialEq, Eq)]
enum TestErr {
    #[error("AttributesInsertionFailed.")]
    AttributesInsertionFailed,
    #[error("EngineSyncing.")]
    EngineSyncing,
    #[error("FinalizedAheadOfUnsafe.")]
    FinalizedAheadOfUnsafe,
    #[error("InvalidPayload.")]
    InvalidPayload,
    #[error("MissingPayloadId.")]
    MissingPayloadId,
    #[error("UnexpectedPayloadStatus.")]
    Unexpected,
    #[error("MpscSend.")]
    MpscSend,
    #[error("InvalidForkchoiceState.")]
    InvalidForkchoiceState,
    #[error("UnsafeHeadChangedSinceBuild.")]
    UnsafeHeadChangedSinceBuild,
}

// Wraps real errors, ignoring details so we can easily match on results.
async fn wrapped_execute<EngineClient_: EngineClient>(
    task: &BuildTask<EngineClient_>,
    state: &mut EngineState,
) -> Result<PayloadId, TestErr> {
    match task.execute(state).await {
        Ok(payload_id) => Ok(payload_id),
        Err(BuildTaskError::EngineBuildError(e)) => match e {
            EngineBuildError::AttributesInsertionFailed(_) => {
                Err(TestErr::AttributesInsertionFailed)
            }
            EngineBuildError::EngineSyncing => Err(TestErr::EngineSyncing),
            EngineBuildError::FinalizedAheadOfUnsafe(_, _) => Err(TestErr::FinalizedAheadOfUnsafe),
            EngineBuildError::InvalidPayload(_) => Err(TestErr::InvalidPayload),
            EngineBuildError::MissingPayloadId => Err(TestErr::MissingPayloadId),
            EngineBuildError::UnexpectedPayloadStatus(_) => Err(TestErr::Unexpected),
            EngineBuildError::InvalidForkchoiceState => Err(TestErr::InvalidForkchoiceState),
        },
        Err(BuildTaskError::MpscSend(_)) => Err(TestErr::MpscSend),
        Err(BuildTaskError::UnsafeHeadChangedSinceBuild) => {
            Err(TestErr::UnsafeHeadChangedSinceBuild)
        }
    }
}

#[rstest]
#[case::success(Some(PayloadStatusEnum::Valid), true, None)]
#[case::missing_id(Some(PayloadStatusEnum::Valid), false, Some(TestErr::MissingPayloadId))]
#[case::fcu_fail(None, false, Some(TestErr::AttributesInsertionFailed))]
#[case::fcu_status_fail(Some(PayloadStatusEnum::Invalid{validation_error: String::new()}), false, Some(TestErr::InvalidPayload))]
#[case::fcu_status_fail(Some(PayloadStatusEnum::Syncing), false, Some(TestErr::EngineSyncing))]
#[case::fcu_status_fail(Some(PayloadStatusEnum::Accepted), false, Some(TestErr::Unexpected))]
#[tokio::test]
async fn test_execute_variants(
    // NB: none = failure
    #[case] fcu_status: Option<PayloadStatusEnum>,
    // NB: none = failure
    #[case] payload_id_present: bool,
    // NB: none = success
    #[case] expected_err: Option<TestErr>,
    #[values(true, false)] with_channel: bool,
    #[values(EngineForkchoiceVersion::V2, EngineForkchoiceVersion::V3)]
    fcu_version: EngineForkchoiceVersion,
    // A detached build whose parent is the unsafe head must behave exactly like an atomic one; the
    // staleness check is covered separately below. Without this, an inverted check would pass.
    #[values(BuildSealCoupling::Atomic, BuildSealCoupling::Detached)] coupling: BuildSealCoupling,
) {
    let payload_id = payload_id_present.then(|| PayloadId::new([1u8; 8]));

    let parent_block = test_block_info(0);
    let attributes_timestamp = parent_block.block_info.timestamp;

    let mut cfg = RollupConfig::default();

    // Configure client with FCU response. If none, it will err on call, which is also a test case.
    let engine_client = fcu_status
        .map_or_else(test_engine_client_builder, |status| {
            configure_fcu(
                test_engine_client_builder(),
                fcu_version,
                fcu_for_payload(payload_id, status),
                &mut cfg,
                attributes_timestamp,
            )
        })
        .build();

    let attributes = TestAttributesBuilder::new()
        .with_parent(parent_block)
        .with_timestamp(attributes_timestamp)
        .build();

    let (tx, mut rx) = mpsc::channel(1);

    let task = BuildTask::new(
        Arc::new(engine_client.clone()),
        Arc::new(cfg),
        attributes.clone(),
        coupling,
        with_channel.then_some(tx),
    );

    let mut state = TestEngineStateBuilder::new()
        .with_unsafe_head(parent_block)
        .with_safe_head(parent_block)
        .with_finalized_head(parent_block)
        .build();

    // Execute: Call execute
    let result = wrapped_execute(&task, &mut state).await;

    if expected_err.is_some() {
        assert_eq!(expected_err, result.err());
    } else {
        assert!(result.is_ok());
        assert!(payload_id.is_some(), "Payload id none when it should be some.");
        assert_eq!(result.unwrap(), payload_id.unwrap(), "Should return the correct payload ID");

        // test channel payload send
        if task.result_tx.is_some() {
            let res = rx.recv().await;
            assert!(res.is_some(), "channel result is None");
            assert_eq!(
                res.unwrap().expect("build succeeded"),
                payload_id.unwrap(),
                "channel should have received correct payload id"
            );
        }
    }
}

#[tokio::test]
async fn invalid_forkchoice_state_requests_a_reset() {
    // The pre-block-creation forkchoice update is rejected by the EL when the safe head is no
    // longer an ancestor of the parent the attributes build on. Retrying it can never succeed, so
    // it must surface as a reset rather than a temporary error the task queue spins on.
    let parent_block = test_block_info(0);
    let engine_client = test_engine_client_builder()
        .with_fork_choice_updated_error(ErrorPayload {
            code: INVALID_FORK_CHOICE_STATE_ERROR as i64,
            message: "invalid forkchoice state".into(),
            data: None,
        })
        .build();

    let attributes = TestAttributesBuilder::new()
        .with_parent(parent_block)
        .with_timestamp(parent_block.block_info.timestamp)
        .build();

    let task = BuildTask::new(
        Arc::new(engine_client),
        Arc::new(RollupConfig::default()),
        attributes,
        BuildSealCoupling::Atomic,
        None,
    );
    let mut state = TestEngineStateBuilder::new().with_unsafe_head(parent_block).build();

    let err = task.execute(&mut state).await.expect_err("forkchoice update is rejected");
    assert_eq!(err.severity(), EngineTaskErrorSeverity::Reset);
}

#[tokio::test]
async fn stale_sequencer_build_is_rejected_without_a_forkchoice_update() {
    // A sequencer picks the parent from its own snapshot of the unsafe head and only then asks the
    // engine to build. If derivation reorged the chain in the meantime, the forkchoice update this
    // build would send has a safe block that is no longer an ancestor of the head, which the
    // execution layer rejects. Reject the job here instead, so the caller re-builds on the new
    // head.
    // Same height, different hash: exactly the shape a force-included derived block takes.
    let stale_parent = test_block_info(42);
    let unsafe_head = test_block_info(42);

    // No forkchoice response is configured: the mock errors if the task calls the engine at all.
    let engine_client = test_engine_client_builder().build();
    let attributes = TestAttributesBuilder::new()
        .with_parent(stale_parent)
        .with_timestamp(unsafe_head.block_info.timestamp)
        .build();

    let (tx, mut rx) = mpsc::channel(1);
    let task = BuildTask::new(
        Arc::new(engine_client),
        Arc::new(RollupConfig::default()),
        attributes,
        BuildSealCoupling::Detached,
        Some(tx),
    );
    let mut state = TestEngineStateBuilder::new().with_unsafe_head(unsafe_head).build();

    assert_eq!(
        wrapped_execute(&task, &mut state).await.err(),
        Some(TestErr::UnsafeHeadChangedSinceBuild)
    );
    assert!(
        matches!(rx.recv().await, Some(Err(BuildTaskError::UnsafeHeadChangedSinceBuild))),
        "the caller must be told to re-build rather than left waiting"
    );
}
