use crate::{
    BuildSealCoupling::{self, Atomic, Detached},
    EngineTaskExt, InsertTaskError, SealTask, SealTaskError,
    test_utils::{
        TestAttributesBuilder, TestEngineStateBuilder, test_block_info, test_engine_client_builder,
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
/// proceeding to the payload fetch — which the unconfigured mock client fails, surfacing as
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
    let task = SealTask::new(
        Arc::new(test_engine_client_builder().build()),
        Arc::new(RollupConfig::default()),
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
    let client = Arc::new(
        test_engine_client_builder()
            .with_execution_payload_v3(OpExecutionPayloadEnvelopeV3 {
                execution_payload: ExecutionPayloadV3::from_block_slow(&block),
                block_value: Default::default(),
                blobs_bundle: Default::default(),
                should_override_builder: false,
                parent_beacon_block_root: Default::default(),
            })
            .with_new_payload_v3_response(PayloadStatus::from_status(PayloadStatusEnum::Invalid {
                validation_error: "invalid".into(),
            }))
            .with_fork_choice_updated_v3_response(ForkchoiceUpdated {
                payload_status: PayloadStatus::from_status(PayloadStatusEnum::Valid),
                payload_id: Some(PayloadId::new([2u8; 8])),
            })
            .build(),
    );
    let mut state = TestEngineStateBuilder::new().with_unsafe_head(parent).build();
    let task = SealTask::new(
        client.clone(),
        Arc::new(cfg),
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
    let rebuilds: Vec<_> =
        client.fork_choice_updated_v3_attributes().await.into_iter().flatten().collect();
    if expected == Rejection::ReplacementFailed {
        assert_eq!(rebuilds.len(), 1, "one deposits-only rebuild");
        assert_eq!(rebuilds[0].transactions, Some(vec![deposit]));
    } else {
        assert!(rebuilds.is_empty(), "nothing replaces the payload");
    }
}
