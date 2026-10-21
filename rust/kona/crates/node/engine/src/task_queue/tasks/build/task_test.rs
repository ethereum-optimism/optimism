use crate::{
    BuildSealCoupling, BuildTask, BuildTaskError, EngineBuildError, EngineClient,
    EngineForkchoiceVersion, EngineState, EngineTaskExt,
    test_utils::{
        MockEngineClientBuilder, TestAttributesBuilder, TestEngineStateBuilder, test_block_info,
        test_engine_client_builder,
    },
};
use alloy_primitives::FixedBytes;
use alloy_rpc_types_engine::{ForkchoiceUpdated, PayloadId, PayloadStatus, PayloadStatusEnum};
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
}

// Wraps real errors, ignoring details so we can easily match on results.
async fn wrapped_execute<EngineClient_: EngineClient>(
    task: &BuildTask<EngineClient_>,
    state: &mut EngineState,
) -> Result<PayloadId, TestErr> {
    match task.execute(state).await {
        Ok(payload_id) => Ok(payload_id.unwrap()),
        Err(BuildTaskError::EngineBuildError(e)) => match e {
            EngineBuildError::StaleBuild => unreachable!(),
            EngineBuildError::AttributesInsertionFailed(_) => {
                Err(TestErr::AttributesInsertionFailed)
            }
            EngineBuildError::EngineSyncing => Err(TestErr::EngineSyncing),
            EngineBuildError::FinalizedAheadOfUnsafe(_, _) => Err(TestErr::FinalizedAheadOfUnsafe),
            EngineBuildError::InvalidPayload(_) => Err(TestErr::InvalidPayload),
            EngineBuildError::MissingPayloadId => Err(TestErr::MissingPayloadId),
            EngineBuildError::UnexpectedPayloadStatus(_) => Err(TestErr::Unexpected),
        },
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
) {
    let payload_id = payload_id_present.then(|| PayloadId::new([1u8; 8]));

    let parent_block = test_block_info(0);
    let unsafe_block = test_block_info(1);
    let attributes_timestamp = unsafe_block.block_info.timestamp;

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
        with_channel.then_some(tx),
        BuildSealCoupling::Atomic,
    );

    let mut state = TestEngineStateBuilder::new()
        .with_unsafe_head(unsafe_block)
        .with_safe_head(parent_block)
        .with_finalized_head(parent_block)
        .build();

    // Execute: Call execute
    let result = if with_channel {
        let outcome = task.execute(&mut state).await.unwrap();
        if expected_err.is_some() {
            assert_eq!(outcome, None);
        }
        // Translate the reply using the same matcher as a direct task result.
        match rx.recv().await.unwrap() {
            Ok(id) => Ok(id),
            Err(BuildTaskError::EngineBuildError(err)) => match err {
                EngineBuildError::AttributesInsertionFailed(_) => {
                    Err(TestErr::AttributesInsertionFailed)
                }
                EngineBuildError::EngineSyncing => Err(TestErr::EngineSyncing),
                EngineBuildError::FinalizedAheadOfUnsafe(_, _) => {
                    Err(TestErr::FinalizedAheadOfUnsafe)
                }
                EngineBuildError::InvalidPayload(_) => Err(TestErr::InvalidPayload),
                EngineBuildError::MissingPayloadId => Err(TestErr::MissingPayloadId),
                EngineBuildError::UnexpectedPayloadStatus(_) => Err(TestErr::Unexpected),
                EngineBuildError::StaleBuild => unreachable!(),
            },
        }
    } else {
        wrapped_execute(&task, &mut state).await
    };

    if expected_err.is_some() {
        assert_eq!(expected_err, result.err());
    } else {
        assert!(result.is_ok());
        assert!(payload_id.is_some(), "Payload id none when it should be some.");
        assert_eq!(result.unwrap(), payload_id.unwrap(), "Should return the correct payload ID");
    }
}

#[tokio::test]
async fn detached_build_rejects_stale_parent_without_calling_the_engine() {
    let client = test_engine_client_builder().build();
    let (sender, mut receiver) = mpsc::channel(1);
    let task = BuildTask::new(
        Arc::new(client.clone()),
        Arc::new(RollupConfig::default()),
        TestAttributesBuilder::new().with_parent(test_block_info(1)).build(),
        Some(sender),
        BuildSealCoupling::Detached,
    );
    let mut state = TestEngineStateBuilder::new().with_unsafe_head(test_block_info(2)).build();
    assert_eq!(task.execute(&mut state).await.unwrap(), None);
    assert!(matches!(
        receiver.recv().await.unwrap(),
        Err(BuildTaskError::EngineBuildError(EngineBuildError::StaleBuild))
    ));
    assert!(client.fork_choice_updated_v3_attributes().await.is_empty());
}

#[tokio::test]
async fn rejected_forkchoice_replies_and_requests_reset() {
    use crate::{EngineTaskError, EngineTaskErrorSeverity};
    use alloy_rpc_types_engine::INVALID_FORK_CHOICE_STATE_ERROR;
    let parent = test_block_info(1);
    let mut config = RollupConfig::default();
    config.hardforks.ecotone_time = Some(0);
    let client = test_engine_client_builder()
        .with_fork_choice_updated_v3_failing_non_deposit_attributes(INVALID_FORK_CHOICE_STATE_ERROR)
        .build();
    let (sender, mut receiver) = mpsc::channel(1);
    let task = BuildTask::new(
        Arc::new(client.clone()),
        Arc::new(config),
        TestAttributesBuilder::new()
            .with_parent(parent)
            .with_transactions(vec![vec![2u8].into()])
            .build(),
        Some(sender),
        BuildSealCoupling::Detached,
    );
    let mut state = TestEngineStateBuilder::new().with_unsafe_head(parent).build();
    let error = task.execute(&mut state).await.unwrap_err();
    assert_eq!(error.severity(), EngineTaskErrorSeverity::Reset);
    assert_eq!(
        receiver.recv().await.unwrap().unwrap_err().severity(),
        EngineTaskErrorSeverity::Reset
    );
    assert_eq!(client.fork_choice_updated_v3_attributes().await.len(), 1);
}
