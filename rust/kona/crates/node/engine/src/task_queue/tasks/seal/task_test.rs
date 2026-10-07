use crate::{
    BuildSealCoupling::{self, Atomic, Detached},
    EngineTaskExt, InsertTaskError, SealTask, SealTaskError,
    test_utils::{
        TestAttributesBuilder, TestEngineStateBuilder, test_block_info, test_engine_client,
    },
};
use alloy_primitives::Bytes;
use alloy_rpc_types_engine::{
    ExecutionPayloadV3, ForkchoiceUpdated, PayloadId, PayloadStatus, PayloadStatusEnum,
};
use kona_genesis::RollupConfig;
use op_alloy_consensus::{DEPOSIT_TX_TYPE_ID, OpTxEnvelope};
use op_alloy_rpc_types_engine::OpExecutionPayloadEnvelopeV3;
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
#[case::detached_with_moved_unsafe_head(Detached, false, SealOutcome::AbortedAsStale)]
#[case::detached_with_current_unsafe_head(Detached, true, SealOutcome::ProceededToSeal)]
#[case::atomic_with_reorged_unsafe_head(Atomic, false, SealOutcome::ProceededToSeal)]
#[case::atomic_with_current_unsafe_head(Atomic, true, SealOutcome::ProceededToSeal)]
#[tokio::test]
async fn unsafe_head_check_variants(
    #[case] coupling: BuildSealCoupling,
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
    if unsafe_head_at_parent || coupling == Atomic {
        l2.expect_error("engine_getPayloadV2");
    }
    let task = SealTask::new(
        Arc::new(client),
        cfg,
        PayloadId::new([1u8; 8]),
        attributes,
        false,
        coupling,
        with_channel.then_some(tx),
        Arc::new(crate::NoopBlockSink),
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
    let attributes = TestAttributesBuilder::new().with_timestamp(timestamp).build();
    let client = Arc::new(client);
    let task = SealTask::new(
        client.clone(),
        cfg.clone(),
        id,
        attributes.clone(),
        false,
        Detached,
        None,
        Arc::new(crate::NoopBlockSink),
    );
    let actual = task.seal_payload(&cfg, &client, id, attributes).await.unwrap();
    assert_eq!(actual, expected);
    l1.assert_finished();
    l2.assert_finished();
}

/// How a seal task ends when the engine rejects its payload at newPayload.
#[derive(Debug, PartialEq, Eq)]
enum Rejection {
    /// The task keeps the engine's rejection, so nothing replaces the payload.
    Kept,
    /// The rejected payload held only deposits, which is critical.
    DepositsOnlyRejected,
    /// The task tried to replace the payload with its deposits-only block, which the mock rejects.
    ReplacementFailed,
}

/// Only derived payloads under Holocene are replaced with their deposits-only block, as op-node
/// does. A sequencer drops its own rejected payload and builds the next one.
#[rstest]
#[case::derived(true, true, false, Rejection::ReplacementFailed)]
#[case::sequenced(false, true, false, Rejection::Kept)]
#[case::derived_before_holocene(true, false, false, Rejection::Kept)]
#[case::derived_deposits_only(true, true, true, Rejection::DepositsOnlyRejected)]
#[case::sequenced_deposits_only(false, true, true, Rejection::DepositsOnlyRejected)]
#[tokio::test]
async fn rejected_payloads_are_replaced_only_if_derived_under_holocene(
    #[case] is_attributes_derived: bool,
    #[case] holocene: bool,
    #[case] deposits_only: bool,
    #[case] expected: Rejection,
) {
    let parent = test_block_info(10);
    let deposit = Bytes::from(vec![DEPOSIT_TX_TYPE_ID, 0x01]);
    let mut transactions = vec![deposit.clone()];
    if !deposits_only {
        transactions.push(Bytes::from(vec![0x02]));
    }
    let attributes =
        TestAttributesBuilder::new().with_parent(parent).with_transactions(transactions).build();
    let mut cfg = RollupConfig::default();
    cfg.hardforks.ecotone_time = Some(0);
    cfg.hardforks.holocene_time = holocene.then_some(0);

    let mut block = alloy_consensus::Block::<OpTxEnvelope>::default();
    block.header.number = parent.block_info.number + 1;
    let cfg = Arc::new(cfg);
    let (client, l1, l2) = test_engine_client(cfg.clone());
    let client = Arc::new(client);
    let payload = OpExecutionPayloadEnvelopeV3 {
        execution_payload: ExecutionPayloadV3::from_block_slow(&block),
        block_value: Default::default(), blobs_bundle: Default::default(),
        should_override_builder: false, parent_beacon_block_root: Default::default(),
    };
    let invalid = PayloadStatus::from_status(PayloadStatusEnum::Invalid { validation_error: "invalid".into() });
    l2.expect("engine_getPayloadV3", payload.clone());
    l2.expect("engine_newPayloadV3", invalid.clone());
    if expected == Rejection::ReplacementFailed {
        let forkchoice = TestEngineStateBuilder::new().with_unsafe_head(parent).build()
            .sync_state.create_forkchoice_state();
        l2.expect_params("engine_forkchoiceUpdatedV3", serde_json::json!([
            forkchoice, attributes.as_deposits_only().attributes
        ]), ForkchoiceUpdated {
            payload_status: PayloadStatus::from_status(PayloadStatusEnum::Valid),
            payload_id: Some(PayloadId::new([2; 8])),
        });
        l2.expect("engine_getPayloadV3", payload);
        l2.expect("engine_newPayloadV3", invalid);
    }
    let mut state = TestEngineStateBuilder::new().with_unsafe_head(parent).build();
    let task = SealTask::new(
        client.clone(),
        cfg,
        PayloadId::new([1u8; 8]),
        attributes,
        is_attributes_derived,
        Atomic,
        None,
        Arc::new(crate::NoopBlockSink),
    );

    let err = task.execute(&mut state).await.expect_err("the engine rejects every payload");

    let rejection = match &err {
        SealTaskError::PayloadInsertionFailed(insert)
            if matches!(**insert, InsertTaskError::UnexpectedPayloadStatus(_)) =>
        {
            Rejection::Kept
        }
        SealTaskError::DepositOnlyPayloadFailed => Rejection::DepositsOnlyRejected,
        SealTaskError::DepositOnlyPayloadReattemptFailed => Rejection::ReplacementFailed,
        other => panic!("unexpected seal error: {other:?}"),
    };
    assert_eq!(rejection, expected);
    l1.assert_finished();
    l2.assert_finished();
}
