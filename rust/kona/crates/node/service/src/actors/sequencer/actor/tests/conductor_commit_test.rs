use super::{super::*, test_actor};
use crate::actors::{
    MockConductor, MockOriginSelector, MockSequencerEngineClient, MockUnsafePayloadGossipClient,
};
use alloy_consensus::{Block, TxEnvelope};
use alloy_rpc_types_engine::{ExecutionPayloadV1, PayloadStatusEnum};
use alloy_transport::{RpcError, TransportErrorKind};
use kona_derive::test_utils::TestAttributesBuilder;
use mockall::Sequence;
use rstest::rstest;

fn sealed_payload(timestamp: u64) -> OpExecutionPayloadEnvelope {
    let mut block = Block::<TxEnvelope>::default();
    block.header.number = 1;
    block.header.timestamp = timestamp;
    OpExecutionPayloadEnvelope::V1(ExecutionPayloadV1::from_block_slow(&block))
}

type TestActor = SequencerActor<
    TestAttributesBuilder,
    MockConductor,
    MockOriginSelector,
    MockSequencerEngineClient,
    MockUnsafePayloadGossipClient,
>;

/// Returns a started test actor with `pending_handle()` waiting to be sealed on the next tick and
/// no L1 origin available for the build job that follows a successful seal.
fn actor_with_pending_handle() -> TestActor {
    let mut actor = test_actor();
    actor.started = true;
    actor.in_flight = Some(InFlightBlock::Building(pending_handle()));
    actor.unsafe_payload_gossip_client.expect_has_capacity().return_const(true);
    actor.engine_client.expect_get_unsafe_head().returning(|| Ok(L2BlockInfo::default()));
    actor
        .origin_selector
        .expect_next_l1_origin()
        .returning(|_, _| Err(L1OriginSelectorError::NotEnoughData(BlockInfo::default())));
    actor
}

/// Runs one build tick.
async fn tick(actor: &mut TestActor) {
    actor.build_ticker.reset_immediately();
    actor.step().await.unwrap();
}

fn pending_handle() -> UnsealedPayloadHandle {
    UnsealedPayloadHandle {
        payload_id: PayloadId::new([1; 8]),
        attributes_with_parent: OpAttributesWithParent::new(
            OpPayloadAttributes::default(),
            L2BlockInfo::default(),
            None,
            false,
        ),
    }
}

#[rstest]
#[case::rejected(false)]
#[case::timeout(true)]
#[tokio::test]
async fn commit_failure_withholds_canonicalization_gossip_and_next_build(#[case] timeout: bool) {
    let mut actor = test_actor();
    actor.started = true;
    actor.in_flight = Some(InFlightBlock::Building(pending_handle()));
    actor
        .engine_client
        .expect_seal_block()
        .withf(|id, _| *id == PayloadId::new([1; 8]))
        .times(2)
        .returning(|_, _| Ok(sealed_payload(2)));
    actor.engine_client.expect_canonicalize_block().times(0);
    actor.engine_client.expect_start_build_block().times(0);
    actor.unsafe_payload_gossip_client.expect_schedule_execution_payload_gossip().times(0);
    actor.unsafe_payload_gossip_client.expect_has_capacity().return_const(true);
    let mut conductor = MockConductor::new();
    conductor.expect_commit_unsafe_payload().times(2).returning(move |_| {
        let error = if timeout {
            TransportErrorKind::custom(std::io::Error::new(
                std::io::ErrorKind::TimedOut,
                "commit timed out",
            ))
        } else {
            RpcError::local_usage_str("commit rejected")
        };
        Err(crate::actors::ConductorError::Rpc(error))
    });
    actor.conductor = Some(conductor);

    for _ in 0..2 {
        actor.build_ticker.reset_immediately();
        actor.step().await.unwrap();
        assert!(matches!(
            &actor.in_flight,
            Some(InFlightBlock::Building(handle)) if handle.payload_id == PayloadId::new([1; 8])
        ));
    }
}

#[tokio::test]
async fn conductor_recovery_reseals_the_build_job_before_commit_canonicalization_and_gossip() {
    let mut actor = actor_with_pending_handle();
    let mut sequence = Sequence::new();
    let mut conductor = MockConductor::new();
    for (timestamp, succeeds) in [(2, false), (3, true)] {
        actor
            .engine_client
            .expect_seal_block()
            .withf(|id, _| *id == PayloadId::new([1; 8]))
            .times(1)
            .in_sequence(&mut sequence)
            .return_once(move |_, _| Ok(sealed_payload(timestamp)));
        let expected = sealed_payload(timestamp);
        conductor
            .expect_commit_unsafe_payload()
            .withf(move |p| p == &expected)
            .times(1)
            .in_sequence(&mut sequence)
            .return_once(move |_| {
                if succeeds {
                    Ok(())
                } else {
                    Err(crate::actors::ConductorError::Rpc(RpcError::local_usage_str(
                        "commit failed",
                    )))
                }
            });
    }
    actor.conductor = Some(conductor);
    let payload = sealed_payload(3);
    let expected = payload.clone();
    actor
        .engine_client
        .expect_canonicalize_block()
        .withf(move |p, parent| p == &expected && *parent == L2BlockInfo::default())
        .times(1)
        .in_sequence(&mut sequence)
        .return_once(|_, _| Ok(()));
    actor
        .unsafe_payload_gossip_client
        .expect_schedule_execution_payload_gossip()
        .withf(move |p| p == &payload)
        .times(1)
        .in_sequence(&mut sequence)
        .return_once(|_| Ok(()));

    tick(&mut actor).await;
    assert!(matches!(actor.in_flight, Some(InFlightBlock::Building(_))));
    tick(&mut actor).await;
    assert!(actor.in_flight.is_none());
}

#[tokio::test]
async fn canonicalization_retry_keeps_the_committed_payload() {
    let mut actor = actor_with_pending_handle();
    let payload = sealed_payload(2);
    let mut sequence = Sequence::new();
    let sealed = payload.clone();
    actor
        .engine_client
        .expect_seal_block()
        .times(1)
        .in_sequence(&mut sequence)
        .return_once(move |_, _| Ok(sealed));
    let mut conductor = MockConductor::new();
    conductor
        .expect_commit_unsafe_payload()
        .times(1)
        .in_sequence(&mut sequence)
        .return_once(|_| Ok(()));
    actor.conductor = Some(conductor);
    let expected = payload.clone();
    actor
        .engine_client
        .expect_canonicalize_block()
        .withf(move |p, _| p == &expected)
        .times(1)
        .in_sequence(&mut sequence)
        .return_once(|_, _| {
            Err(EngineClientError::CanonicalizeError(
                CanonicalizeTaskError::PayloadInsertionFailed(Box::new(
                    InsertTaskError::ForkchoiceUpdateFailed(
                        SynchronizeTaskError::ForkchoiceUpdateFailed(RpcError::local_usage_str(
                            "temporary FCU failure",
                        )),
                    ),
                )),
            ))
        });
    let expected = payload.clone();
    actor
        .engine_client
        .expect_canonicalize_block()
        .withf(move |p, _| p == &expected)
        .times(1)
        .in_sequence(&mut sequence)
        .return_once(|_, _| Ok(()));
    actor
        .unsafe_payload_gossip_client
        .expect_schedule_execution_payload_gossip()
        .withf(move |p| p == &payload)
        .times(1)
        .in_sequence(&mut sequence)
        .return_once(|_| Ok(()));

    tick(&mut actor).await;
    assert!(matches!(actor.in_flight, Some(InFlightBlock::Committed(_))));
    tick(&mut actor).await;
    assert!(actor.in_flight.is_none());
}

#[rstest]
#[case::stale(true)]
#[case::invalid(false)]
#[tokio::test]
async fn stale_or_invalid_committed_payload_is_dropped(#[case] stale: bool) {
    let mut actor = test_actor();
    actor.started = true;
    actor.in_flight = Some(InFlightBlock::Committed(CommittedBlock {
        envelope: sealed_payload(2),
        parent: L2BlockInfo::default(),
        tx_count: 0,
    }));
    actor.engine_client.expect_seal_block().times(0);
    actor.engine_client.expect_canonicalize_block().times(1).return_once(move |_, _| {
        Err(EngineClientError::CanonicalizeError(if stale {
            CanonicalizeTaskError::UnsafeHeadChangedSinceBuild
        } else {
            CanonicalizeTaskError::PayloadInsertionFailed(Box::new(
                InsertTaskError::UnexpectedPayloadStatus(PayloadStatusEnum::Invalid {
                    validation_error: "invalid".into(),
                }),
            ))
        }))
    });
    actor.engine_client.expect_start_build_block().times(0);
    actor.unsafe_payload_gossip_client.expect_schedule_execution_payload_gossip().times(0);
    actor.unsafe_payload_gossip_client.expect_has_capacity().return_const(true);
    actor.step().await.unwrap();
    assert!(actor.in_flight.is_none());
}

#[tokio::test]
async fn sequencing_without_a_conductor_canonicalizes_before_gossip() {
    let payload = sealed_payload(2);
    let mut actor = actor_with_pending_handle();
    let mut sequence = Sequence::new();
    let sealed = payload.clone();
    actor
        .engine_client
        .expect_seal_block()
        .times(1)
        .in_sequence(&mut sequence)
        .return_once(move |_, _| Ok(sealed));
    actor
        .engine_client
        .expect_canonicalize_block()
        .times(1)
        .in_sequence(&mut sequence)
        .return_once(|_, _| Ok(()));
    actor
        .unsafe_payload_gossip_client
        .expect_schedule_execution_payload_gossip()
        .withf(move |p| p == &payload)
        .times(1)
        .in_sequence(&mut sequence)
        .return_once(|_| Ok(()));
    tick(&mut actor).await;
    assert!(actor.in_flight.is_none());
}

#[tokio::test]
async fn failed_payload_fetch_drops_the_expired_build_job() {
    let mut actor = test_actor();
    actor.started = true;
    actor.in_flight = Some(InFlightBlock::Building(pending_handle()));
    actor.engine_client.expect_seal_block().times(1).return_once(|_, _| {
        Err(EngineClientError::SealError(SealTaskError::GetPayloadFailed(
            RpcError::local_usage_str("unknown payload"),
        )))
    });
    actor.engine_client.expect_canonicalize_block().times(0);
    actor.engine_client.expect_start_build_block().times(0);
    actor.unsafe_payload_gossip_client.expect_schedule_execution_payload_gossip().times(0);
    actor.unsafe_payload_gossip_client.expect_has_capacity().return_const(true);
    actor.step().await.unwrap();
    assert!(actor.in_flight.is_none());
}
