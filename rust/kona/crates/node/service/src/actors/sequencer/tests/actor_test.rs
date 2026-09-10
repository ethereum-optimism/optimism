use super::{CanonicalizationErrorAction, PendingPayload, canonicalization_error_action};
#[cfg(test)]
use crate::{
    ConductorError, L1OriginSelectorError, SequencerActorError,
    actors::{
        MockConductor, MockOriginSelector, MockSequencerEngineClient,
        sequencer::{actor::UnsealedPayloadHandle, tests::test_util::test_actor},
    },
};
use alloy_primitives::{Address, B256, Bloom, Bytes, U256};
use alloy_rpc_types_engine::{ExecutionPayloadV1, PayloadId, PayloadStatusEnum};
use alloy_transport::RpcError;
use kona_derive::{BuilderError, PipelineErrorKind, test_utils::TestAttributesBuilder};
use kona_engine::{InsertTaskError, SynchronizeTaskError};
use kona_protocol::{BlockInfo, L2BlockInfo, OpAttributesWithParent};
use mockall::Sequence;
use op_alloy_rpc_types_engine::{OpExecutionPayloadEnvelope, OpPayloadAttributes};
use rstest::rstest;

#[rstest]
#[case::temp(PipelineErrorKind::Temporary(BuilderError::Custom(String::new()).into()), false)]
#[case::reset(PipelineErrorKind::Reset(BuilderError::Custom(String::new()).into()), false)]
#[case::critical(PipelineErrorKind::Critical(BuilderError::Custom(String::new()).into()), true)]
#[tokio::test]
async fn test_build_unsealed_payload_prepare_payload_attributes_error(
    #[case] forced_error: PipelineErrorKind,
    #[case] expect_err: bool,
) {
    let mut client = MockSequencerEngineClient::new();

    let unsafe_head = L2BlockInfo::default();
    client.expect_get_unsafe_head().times(1).return_once(move || Ok(unsafe_head));
    // Must not be called on critical error
    client.expect_start_build_block().times(0);
    if let PipelineErrorKind::Reset(_) = &forced_error {
        client.expect_reset_engine_forkchoice().times(1).return_once(move || Ok(()));
    }

    let l1_origin = BlockInfo::default();
    let mut origin_selector = MockOriginSelector::new();
    origin_selector.expect_next_l1_origin().times(1).return_once(move |_, _| Ok(l1_origin));

    let attributes_builder = TestAttributesBuilder { attributes: vec![Err(forced_error)] };

    let mut actor = test_actor();
    actor.origin_selector = origin_selector;
    actor.engine_client = client;
    actor.attributes_builder = attributes_builder;

    let result = actor.build_unsealed_payload().await;
    if expect_err {
        assert!(result.is_err());
        assert!(matches!(
            result.unwrap_err(),
            SequencerActorError::AttributesBuilder(PipelineErrorKind::Critical(_))
        ));
    } else {
        assert!(result.is_ok());
    }
}

fn test_payload() -> OpExecutionPayloadEnvelope {
    OpExecutionPayloadEnvelope::V1(ExecutionPayloadV1 {
        parent_hash: B256::ZERO,
        fee_recipient: Address::ZERO,
        state_root: B256::ZERO,
        receipts_root: B256::ZERO,
        logs_bloom: Bloom::ZERO,
        prev_randao: B256::ZERO,
        block_number: 1,
        gas_limit: 30_000_000,
        gas_used: 0,
        timestamp: 2,
        extra_data: Bytes::new(),
        base_fee_per_gas: U256::from(1),
        block_hash: B256::ZERO,
        transactions: Vec::new(),
    })
}

fn test_unsealed_payload() -> UnsealedPayloadHandle {
    UnsealedPayloadHandle {
        payload_id: PayloadId::default(),
        attributes_with_parent: OpAttributesWithParent::new(
            OpPayloadAttributes::default(),
            L2BlockInfo::default(),
            None,
            false,
        ),
    }
}

#[tokio::test]
async fn test_rejected_conductor_commit_is_not_canonicalized_or_gossiped() {
    let payload = test_payload();
    let mut sequence = Sequence::new();
    let mut engine = MockSequencerEngineClient::new();
    engine.expect_seal_block().times(1).in_sequence(&mut sequence).return_once({
        let payload = payload.clone();
        move |_, _| Ok(payload)
    });
    engine.expect_canonicalize_block().times(0);

    let mut conductor = MockConductor::new();
    conductor
        .expect_commit_unsafe_payload()
        .times(1)
        .in_sequence(&mut sequence)
        .return_once(|_| Err(ConductorError::Rpc(RpcError::local_usage_str("leadership lost"))));

    let mut actor = test_actor();
    actor.engine_client = engine;
    actor.conductor = Some(conductor);
    actor.unsafe_payload_gossip_client.expect_schedule_execution_payload_gossip().times(0);
    actor.pending_payload = Some(PendingPayload::Unsealed(Box::new(test_unsealed_payload())));

    assert!(actor.handle_build_tick().await.is_ok());
    match actor.pending_payload.as_ref() {
        Some(PendingPayload::Sealed(retained)) => assert_eq!(retained.as_ref(), &payload),
        other => panic!("expected exact sealed payload to be retained, got {other:?}"),
    }
}

#[tokio::test]
async fn test_failed_conductor_commit_retries_exact_sealed_payload() {
    let payload = test_payload();
    let mut sequence = Sequence::new();
    let mut engine = MockSequencerEngineClient::new();
    engine.expect_seal_block().times(1).in_sequence(&mut sequence).return_once({
        let payload = payload.clone();
        move |_, _| Ok(payload)
    });
    engine
        .expect_canonicalize_block()
        .withf({
            let payload = payload.clone();
            move |actual| actual == &payload
        })
        .times(1)
        .return_once(|_| Ok(()));
    engine.expect_get_unsafe_head().times(1).return_once(|| Ok(L2BlockInfo::default()));
    engine.expect_reset_engine_forkchoice().times(1).return_once(|| Ok(()));

    let mut conductor = MockConductor::new();
    conductor
        .expect_commit_unsafe_payload()
        .withf({
            let payload = payload.clone();
            move |actual| actual == &payload
        })
        .times(1)
        .in_sequence(&mut sequence)
        .return_once(|_| Err(ConductorError::Rpc(RpcError::local_usage_str("response lost"))));
    conductor
        .expect_commit_unsafe_payload()
        .withf({
            let payload = payload.clone();
            move |actual| actual == &payload
        })
        .times(1)
        .in_sequence(&mut sequence)
        .return_once(|_| Ok(()));

    let mut origin_selector = MockOriginSelector::new();
    origin_selector
        .expect_next_l1_origin()
        .times(1)
        .return_once(|head, _| Err(L1OriginSelectorError::OriginNotFound(head.l1_origin.hash)));

    let mut actor = test_actor();
    actor.engine_client = engine;
    actor.conductor = Some(conductor);
    actor.origin_selector = origin_selector;
    actor
        .unsafe_payload_gossip_client
        .expect_schedule_execution_payload_gossip()
        .withf({
            let payload = payload.clone();
            move |actual| actual == &payload
        })
        .times(1)
        .in_sequence(&mut sequence)
        .return_once(|_| Ok(()));
    actor.pending_payload = Some(PendingPayload::Unsealed(Box::new(test_unsealed_payload())));

    assert!(actor.handle_build_tick().await.is_ok());
    match actor.pending_payload.as_ref() {
        Some(PendingPayload::Sealed(retained)) => assert_eq!(retained.as_ref(), &payload),
        other => panic!("expected exact sealed payload to be retained, got {other:?}"),
    }

    assert!(actor.handle_build_tick().await.is_ok());
    assert!(actor.pending_payload.is_none());
}

#[tokio::test]
async fn test_temporary_canonicalization_failure_retries_exact_committed_payload() {
    let payload = test_payload();
    let mut sequence = Sequence::new();
    let mut engine = MockSequencerEngineClient::new();
    engine.expect_seal_block().times(1).return_once({
        let payload = payload.clone();
        move |_, _| Ok(payload)
    });
    engine.expect_canonicalize_block().times(1).in_sequence(&mut sequence).return_once(|_| {
        Err(crate::EngineClientError::CanonicalizeError(InsertTaskError::InsertFailed(
            RpcError::local_usage_str("temporary engine failure"),
        )))
    });
    engine
        .expect_canonicalize_block()
        .withf({
            let payload = payload.clone();
            move |actual| actual == &payload
        })
        .times(1)
        .in_sequence(&mut sequence)
        .return_once(|_| Ok(()));
    engine.expect_get_unsafe_head().times(1).return_once(|| Ok(L2BlockInfo::default()));
    engine.expect_reset_engine_forkchoice().times(1).return_once(|| Ok(()));

    let mut conductor = MockConductor::new();
    conductor.expect_commit_unsafe_payload().times(1).return_once(|_| Ok(()));

    let mut origin_selector = MockOriginSelector::new();
    origin_selector
        .expect_next_l1_origin()
        .times(1)
        .return_once(|head, _| Err(L1OriginSelectorError::OriginNotFound(head.l1_origin.hash)));

    let mut actor = test_actor();
    actor.engine_client = engine;
    actor.conductor = Some(conductor);
    actor.origin_selector = origin_selector;
    actor
        .unsafe_payload_gossip_client
        .expect_schedule_execution_payload_gossip()
        .withf({
            let payload = payload.clone();
            move |actual| actual == &payload
        })
        .times(1)
        .return_once(|_| Ok(()));
    actor.pending_payload = Some(PendingPayload::Unsealed(Box::new(test_unsealed_payload())));

    assert!(actor.handle_build_tick().await.is_ok());
    match actor.pending_payload.as_ref() {
        Some(PendingPayload::Committed(retained)) => assert_eq!(retained.as_ref(), &payload),
        other => panic!("expected exact committed payload to be retained, got {other:?}"),
    }

    assert!(actor.handle_build_tick().await.is_ok());
    assert!(actor.pending_payload.is_none());
}

#[test]
fn transient_new_payload_and_forkchoice_errors_retry_committed_payload() {
    let errors = [
        InsertTaskError::InsertFailed(RpcError::local_usage_str("new payload unavailable")),
        InsertTaskError::UnexpectedPayloadStatus(PayloadStatusEnum::Accepted),
        InsertTaskError::ForkchoiceUpdateFailed(SynchronizeTaskError::ForkchoiceUpdateFailed(
            RpcError::local_usage_str("forkchoice unavailable"),
        )),
        InsertTaskError::ForkchoiceUpdateFailed(SynchronizeTaskError::UnexpectedPayloadStatus(
            PayloadStatusEnum::Accepted,
        )),
    ];

    for err in errors {
        assert_eq!(canonicalization_error_action(&err), CanonicalizationErrorAction::Retry);
    }
}

#[test]
fn permanently_invalid_new_payload_and_forkchoice_statuses_are_fatal() {
    let errors = [
        InsertTaskError::UnexpectedPayloadStatus(PayloadStatusEnum::Invalid {
            validation_error: "invalid payload".to_string(),
        }),
        InsertTaskError::ForkchoiceUpdateFailed(SynchronizeTaskError::UnexpectedPayloadStatus(
            PayloadStatusEnum::Invalid { validation_error: "invalid forkchoice".to_string() },
        )),
    ];

    for err in errors {
        assert_eq!(canonicalization_error_action(&err), CanonicalizationErrorAction::Fatal);
    }
}

#[tokio::test]
async fn invalid_conductor_approved_payload_stops_instead_of_retrying() {
    let payload = test_payload();
    let mut engine = MockSequencerEngineClient::new();
    engine.expect_canonicalize_block().times(1).return_once(|_| {
        Err(crate::EngineClientError::CanonicalizeError(InsertTaskError::UnexpectedPayloadStatus(
            PayloadStatusEnum::Invalid { validation_error: "invalid payload".to_string() },
        )))
    });

    let mut actor = test_actor();
    actor.engine_client = engine;
    actor.unsafe_payload_gossip_client.expect_schedule_execution_payload_gossip().times(0);
    actor.pending_payload = Some(PendingPayload::Committed(Box::new(payload)));

    assert!(matches!(
        actor.handle_build_tick().await,
        Err(SequencerActorError::EngineError(crate::EngineClientError::CanonicalizeError(
            InsertTaskError::UnexpectedPayloadStatus(PayloadStatusEnum::Invalid { .. })
        )))
    ));
    assert!(actor.pending_payload.is_none(), "invalid payload must not be retained for retry");
}

#[tokio::test]
async fn test_stale_payload_without_conductor_is_dropped_without_gossip() {
    let payload = test_payload();
    let mut engine = MockSequencerEngineClient::new();
    engine.expect_canonicalize_block().times(1).return_once(|_| {
        Err(crate::EngineClientError::CanonicalizeError(InsertTaskError::StalePayload {
            parent: B256::ZERO,
            unsafe_head: B256::repeat_byte(1),
        }))
    });

    let mut actor = test_actor();
    actor.engine_client = engine;
    actor.unsafe_payload_gossip_client.expect_schedule_execution_payload_gossip().times(0);
    actor.pending_payload = Some(PendingPayload::Committed(Box::new(payload)));

    assert!(actor.handle_build_tick().await.is_ok());
    assert!(actor.pending_payload.is_none());
}

#[tokio::test]
async fn stale_conductor_approved_payload_stops_before_building_replacement() {
    let payload = test_payload();
    let mut engine = MockSequencerEngineClient::new();
    engine.expect_canonicalize_block().times(1).return_once(|_| {
        Err(crate::EngineClientError::CanonicalizeError(InsertTaskError::StalePayload {
            parent: B256::ZERO,
            unsafe_head: B256::repeat_byte(1),
        }))
    });

    let mut actor = test_actor();
    actor.engine_client = engine;
    actor.conductor = Some(MockConductor::new());
    actor.unsafe_payload_gossip_client.expect_schedule_execution_payload_gossip().times(0);
    actor.pending_payload = Some(PendingPayload::Committed(Box::new(payload)));

    assert!(matches!(
        actor.handle_build_tick().await,
        Err(SequencerActorError::EngineError(crate::EngineClientError::CanonicalizeError(
            InsertTaskError::StalePayload { .. }
        )))
    ));
    assert!(actor.pending_payload.is_none());
}

#[tokio::test]
async fn test_accepted_conductor_commit_is_canonicalized_before_gossip() {
    let mut sequence = Sequence::new();
    let mut engine = MockSequencerEngineClient::new();
    engine
        .expect_seal_block()
        .times(1)
        .in_sequence(&mut sequence)
        .return_once(|_, _| Ok(test_payload()));

    let mut conductor = MockConductor::new();
    conductor
        .expect_commit_unsafe_payload()
        .times(1)
        .in_sequence(&mut sequence)
        .return_once(|_| Ok(()));

    engine.expect_canonicalize_block().times(1).in_sequence(&mut sequence).return_once(|_| Ok(()));

    let mut actor = test_actor();
    actor.engine_client = engine;
    actor.conductor = Some(conductor);
    actor
        .unsafe_payload_gossip_client
        .expect_schedule_execution_payload_gossip()
        .times(1)
        .in_sequence(&mut sequence)
        .return_once(|_| Ok(()));

    let payload = actor.seal_payload(&test_unsealed_payload()).await.unwrap();
    let payload = actor.commit_payload_or_backoff(Box::new(payload)).await.unwrap();
    assert!(actor.canonicalize_and_gossip_payload(&payload).await.is_ok());
}
