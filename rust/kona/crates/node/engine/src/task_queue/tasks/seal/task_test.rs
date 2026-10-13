use crate::{
    EngineTaskExt, SealTask, SealTaskError,
    test_utils::{
        TestAttributesBuilder, TestEngineStateBuilder, test_block_info, test_engine_client,
    },
};
use alloy_rpc_types_engine::PayloadId;
use kona_genesis::RollupConfig;
use rstest::rstest;
use std::sync::Arc;
use tokio::sync::mpsc;

/// The two paths the unsafe-head check can steer `execute` into: aborting the seal as stale, or
/// proceeding to the payload fetch — whose injected RPC failure surfaces as
/// [`SealTaskError::GetPayloadFailed`].
#[derive(Debug, PartialEq, Eq)]
enum SealOutcome {
    AbortedAsStale,
    ProceededToSeal,
}

fn classify(err: &SealTaskError) -> SealOutcome {
    match err {
        SealTaskError::UnsafeHeadChangedSinceBuild => SealOutcome::AbortedAsStale,
        SealTaskError::GetPayloadFailed(_) => SealOutcome::ProceededToSeal,
        other => panic!("unexpected seal error: {other:?}"),
    }
}

#[rstest]
#[case::moved_unsafe_head(false, SealOutcome::AbortedAsStale)]
#[case::current_unsafe_head(true, SealOutcome::ProceededToSeal)]
#[tokio::test]
async fn unsafe_head_check_variants(
    #[case] unsafe_head_at_parent: bool,
    #[case] expected: SealOutcome,
    #[values(true, false)] with_channel: bool,
) {
    let parent_block = test_block_info(10);
    let unsafe_head = if unsafe_head_at_parent { parent_block } else { test_block_info(15) };

    let attributes = TestAttributesBuilder::new().with_parent(parent_block).build();
    let mut state = TestEngineStateBuilder::new().with_unsafe_head(unsafe_head).build();

    let (tx, mut rx) = mpsc::channel(1);
    let cfg = Arc::new(RollupConfig::default());
    let (client, l1, l2) = test_engine_client(cfg.clone());
    if unsafe_head_at_parent {
        l2.expect_error("engine_getPayloadV2");
    }
    let task = SealTask::new(
        Arc::new(client),
        cfg,
        PayloadId::new([1u8; 8]),
        attributes,
        with_channel.then_some(tx),
    );

    let result = task.execute(&mut state).await;

    if with_channel {
        // With a result channel, the task itself succeeds and the error is relayed to the caller.
        result.expect("task with a result channel should succeed");
        let relayed = rx.recv().await.expect("channel should receive the seal result");
        assert_eq!(classify(&relayed.expect_err("seal should fail against the mock")), expected);
    } else {
        assert_eq!(classify(&result.expect_err("seal should fail against the mock")), expected);
    }
    l1.assert_finished();
    l2.assert_finished();
}

/// Exercise version selection and decoding through the production client and OP Alloy extension.
#[rstest]
#[case(5, "engine_getPayloadV2")]
#[case(15, "engine_getPayloadV3")]
#[case(25, "engine_getPayloadV4")]
#[case(35, "engine_getPayloadV5")]
#[tokio::test]
async fn payload_fetch_selects_version_and_decodes_reply(
    #[case] timestamp: u64,
    #[case] method: &'static str,
) {
    use alloy_primitives::B256;
    use alloy_rpc_types_engine::{
        ExecutionPayloadEnvelopeV2, ExecutionPayloadFieldV2, ExecutionPayloadV2, ExecutionPayloadV3,
    };
    use op_alloy_rpc_types_engine::{
        OpExecutionPayloadEnvelope, OpExecutionPayloadEnvelopeV3, OpExecutionPayloadEnvelopeV4,
        OpExecutionPayloadV4,
    };

    let mut cfg = RollupConfig::default();
    cfg.hardforks.ecotone_time = Some(10);
    cfg.hardforks.isthmus_time = Some(20);
    cfg.hardforks.karst_time = Some(30);
    let cfg = Arc::new(cfg);
    let (client, l1, l2) = test_engine_client(cfg.clone());
    let id = PayloadId::new([1; 8]);
    let params = serde_json::json!([id]);
    let root = B256::repeat_byte(0x42);
    let block = alloy_consensus::Block::<op_alloy_consensus::OpTxEnvelope>::default();
    let expected = if timestamp < 10 {
        let payload = ExecutionPayloadV2::from_block_slow(&block);
        l2.expect_params(
            method,
            params,
            ExecutionPayloadEnvelopeV2 {
                execution_payload: ExecutionPayloadFieldV2::V2(payload.clone()),
                block_value: Default::default(),
            },
        );
        OpExecutionPayloadEnvelope::V2(payload)
    } else if timestamp < 20 {
        let payload = ExecutionPayloadV3::from_block_slow(&block);
        l2.expect_params(
            method,
            params,
            OpExecutionPayloadEnvelopeV3 {
                execution_payload: payload.clone(),
                block_value: Default::default(),
                blobs_bundle: Default::default(),
                should_override_builder: false,
                parent_beacon_block_root: root,
            },
        );
        OpExecutionPayloadEnvelope::V3 { payload, parent_beacon_block_root: root }
    } else {
        let payload = OpExecutionPayloadV4::from_v3_with_withdrawals_root(
            ExecutionPayloadV3::from_block_slow(&block),
            root,
        );
        l2.expect_params(
            method,
            params,
            OpExecutionPayloadEnvelopeV4 {
                execution_payload: payload.clone(),
                block_value: Default::default(),
                blobs_bundle: Default::default(),
                should_override_builder: false,
                parent_beacon_block_root: root,
                execution_requests: vec![],
            },
        );
        OpExecutionPayloadEnvelope::V4 { payload, parent_beacon_block_root: root }
    };
    let actual = super::task::get_payload(&client, &cfg, id, timestamp).await.unwrap();
    assert_eq!(actual, expected);
    l1.assert_finished();
    l2.assert_finished();
}

/// The sequencer commits a sealed payload to the conductor before importing it, so sealing must
/// not change forkchoice.
#[tokio::test]
async fn seal_keeps_forkchoice_unchanged() {
    use super::super::canonicalize::tests::{PayloadFixture, payload_fixture};
    use alloy_rpc_types_engine::{ExecutionPayloadEnvelopeV2, ExecutionPayloadFieldV2};

    let PayloadFixture { payload: expected, cfg } = payload_fixture();
    let op_alloy_rpc_types_engine::OpExecutionPayloadEnvelope::V1(payload) = expected.clone()
    else {
        panic!("fixture must be V1");
    };
    let (engine, l1, l2) = test_engine_client(cfg.clone());
    l2.expect(
        "engine_getPayloadV2",
        ExecutionPayloadEnvelopeV2 {
            execution_payload: ExecutionPayloadFieldV2::V1(payload),
            block_value: Default::default(),
        },
    );
    let (tx, mut rx) = mpsc::channel(1);
    let mut state = crate::EngineState::default();
    let original = state;
    let task = SealTask::new(
        Arc::new(engine),
        cfg,
        PayloadId::new([1; 8]),
        TestAttributesBuilder::new().with_parent(kona_protocol::L2BlockInfo::default()).build(),
        Some(tx),
    );
    task.execute(&mut state).await.unwrap();
    assert_eq!(rx.recv().await.unwrap().unwrap(), expected);
    assert_eq!(state, original);
    l1.assert_finished();
    l2.assert_finished();
}
