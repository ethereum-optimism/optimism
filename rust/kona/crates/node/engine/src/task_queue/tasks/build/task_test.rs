use crate::{
    BuildTask, BuildTaskError, EngineBuildError, EngineForkchoiceVersion, EngineState,
    EngineTaskExt,
    test_utils::{
        TestAttributesBuilder, TestEngineStateBuilder, test_block_info, test_engine_client,
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
}

// Wraps real errors, ignoring details so we can easily match on results.
async fn wrapped_execute(task: &BuildTask, state: &mut EngineState) -> Result<PayloadId, TestErr> {
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
        },
        Err(BuildTaskError::MpscSend(_)) => Err(TestErr::MpscSend),
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

    let method = match fcu_version {
        EngineForkchoiceVersion::V2 => {
            cfg.hardforks.ecotone_time = Some(attributes_timestamp + 1);
            "engine_forkchoiceUpdatedV2"
        }
        EngineForkchoiceVersion::V3 => {
            cfg.hardforks.ecotone_time = Some(attributes_timestamp);
            "engine_forkchoiceUpdatedV3"
        }
    };
    let cfg = Arc::new(cfg);
    let (engine_client, l1, l2) = test_engine_client(cfg.clone());
    if let Some(status) = fcu_status {
        l2.expect(method, fcu_for_payload(payload_id, status));
    } else {
        l2.expect_error(method);
    }

    let attributes = TestAttributesBuilder::new()
        .with_parent(parent_block)
        .with_timestamp(attributes_timestamp)
        .build();

    let (tx, mut rx) = mpsc::channel(1);

    let task = BuildTask::new(
        Arc::new(engine_client.clone()),
        cfg,
        attributes.clone(),
        with_channel.then_some(tx),
    );

    let mut state = TestEngineStateBuilder::new()
        .with_unsafe_head(unsafe_block)
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
        if task.payload_id_tx.is_some() {
            let res = rx.recv().await;
            assert!(res.is_some(), "channel result is None");
            assert_eq!(
                res.unwrap(),
                payload_id.unwrap(),
                "channel should have received correct payload id"
            );
        }
    }
    l1.assert_finished();
    l2.assert_finished();
}
