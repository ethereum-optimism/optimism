use super::{super::*, test_actor, test_actor_with_config};
use crate::actors::{
    MockConductor, MockOriginSelector, MockSequencerEngineClient, MockUnsafePayloadGossipClient,
};
use alloy_consensus::{Block, SignableTransaction, TxEip1559, TxEnvelope};
use alloy_eips::eip2718::Encodable2718;
use alloy_primitives::{Sealed, Signature};
use alloy_rpc_types_engine::{ExecutionPayloadV1, PayloadStatusEnum};
use alloy_transport::{RpcError, TransportErrorKind};
use kona_derive::test_utils::TestAttributesBuilder;
use mockall::Sequence;
use op_alloy_consensus::{OpTxEnvelope, TxDeposit};
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

fn pending_sealed_block() -> ConductorCommittedBlock {
    ConductorCommittedBlock::new(sealed_payload(2), &pending_handle(), true)
}

#[rstest]
#[case::building(InFlightBlock::Building(pending_handle()))]
#[case::committing(InFlightBlock::Committing(pending_sealed_block()))]
#[case::committed(InFlightBlock::Committed(pending_sealed_block()))]
#[tokio::test(start_paused = true)]
async fn stop_then_start_builds_on_the_new_head(#[case] in_flight: InFlightBlock) {
    let mut conductor = MockConductor::new();
    conductor.expect_leader().times(1).return_once(|| Ok(true));
    conductor.expect_commit_unsafe_payload().times(0);
    let (mut actor, commands) = test_actor_with_config(true, false, Some(conductor));
    actor.started = true;
    actor.in_flight = Some(in_flight);
    let old_head = L2BlockInfo::default();
    let new_head = L2BlockInfo {
        block_info: BlockInfo {
            hash: B256::repeat_byte(42),
            number: 10,
            timestamp: super::TEST_START,
            ..Default::default()
        },
        ..Default::default()
    };
    let mut sequence = Sequence::new();
    actor
        .engine_client
        .expect_get_unsafe_head()
        .times(1)
        .in_sequence(&mut sequence)
        .return_once(move || Ok(old_head));
    // The node follows a new leader while stopped; start and the next build see its head.
    actor
        .engine_client
        .expect_get_unsafe_head()
        .times(2)
        .in_sequence(&mut sequence)
        .returning(move || Ok(new_head));
    actor.engine_client.expect_seal_block().times(0);
    actor.engine_client.expect_canonicalize_block().times(0);
    actor
        .engine_client
        .expect_start_build_block()
        .withf(move |attrs| attrs.parent == new_head)
        .times(1)
        .in_sequence(&mut sequence)
        .return_once(|_| Ok(PayloadId::new([2; 8])));
    actor
        .origin_selector
        .expect_next_l1_origin()
        .withf(move |head, _| *head == new_head)
        .times(1)
        .return_once(|_, _| Ok(BlockInfo::default()));
    actor.attributes_builder = TestAttributesBuilder { attributes: vec![Ok(Default::default())] };
    actor.unsafe_payload_gossip_client.expect_has_capacity().return_const(true);
    actor.unsafe_payload_gossip_client.expect_schedule_execution_payload_gossip().times(0);

    let (reply, stopped) = tokio::sync::oneshot::channel();
    commands.send(SequencerAdminCommand::StopSequencer(reply)).await.unwrap();
    actor.step().await.unwrap();
    assert_eq!(stopped.await.unwrap().unwrap(), old_head.hash());
    assert!(!actor.state().active);
    assert!(actor.in_flight.is_none());

    let (reply, started) = tokio::sync::oneshot::channel();
    commands.send(SequencerAdminCommand::StartSequencer(new_head.hash(), reply)).await.unwrap();
    actor.step().await.unwrap();
    started.await.unwrap().unwrap();
    assert!(actor.state().active);
    assert!(actor.in_flight.is_none());

    tick(&mut actor).await;
    assert!(matches!(&actor.in_flight, Some(InFlightBlock::Building(handle))
        if handle.payload_id == PayloadId::new([2; 8])
            && handle.attributes_with_parent.parent == new_head));
}

#[rstest]
#[case::building(InFlightBlock::Building(pending_handle()))]
#[case::committing(InFlightBlock::Committing(pending_sealed_block()))]
#[case::committed(InFlightBlock::Committed(pending_sealed_block()))]
#[tokio::test(start_paused = true)]
async fn stop_clears_pending_work_even_if_head_read_fails(
    #[case] in_flight: InFlightBlock,
    #[values(true, false)] already_stopped: bool,
) {
    let (mut actor, commands) = test_actor_with_config(!already_stopped, false, None);
    actor.started = true;
    actor.in_flight = Some(in_flight);
    actor
        .engine_client
        .expect_get_unsafe_head()
        .times(2)
        .returning(|| Err(EngineClientError::RequestError("head unavailable".into())));
    actor.engine_client.expect_start_build_block().times(0);
    actor.engine_client.expect_seal_block().times(0);
    actor.engine_client.expect_canonicalize_block().times(0);
    actor.unsafe_payload_gossip_client.expect_has_capacity().times(0);
    actor.unsafe_payload_gossip_client.expect_schedule_execution_payload_gossip().times(0);

    // A failed response read and repeated stops must still discard all pending work.
    for _ in 0..2 {
        let (reply, stopped) = tokio::sync::oneshot::channel();
        commands.send(SequencerAdminCommand::StopSequencer(reply)).await.unwrap();
        actor.step().await.unwrap();
        assert!(matches!(stopped.await.unwrap(), Err(SequencerAdminAPIError::UnsafeHeadError(_))));
        assert!(!actor.state().active);
        assert!(actor.in_flight.is_none());
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
        .times(1)
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
            Some(InFlightBlock::Committing(block)) if block.envelope == sealed_payload(2)
        ));
    }
}

#[tokio::test]
async fn conductor_recovery_retries_the_same_envelope_before_canonicalization_and_gossip() {
    let mut actor = actor_with_pending_handle();
    let mut sequence = Sequence::new();
    let mut conductor = MockConductor::new();
    actor
        .engine_client
        .expect_seal_block()
        .withf(|id, _| *id == PayloadId::new([1; 8]))
        .times(1)
        .in_sequence(&mut sequence)
        .return_once(|_, _| Ok(sealed_payload(2)));
    for succeeds in [false, true] {
        let expected = sealed_payload(2);
        conductor
            .expect_commit_unsafe_payload()
            .withf(move |p| p == &expected)
            .times(1)
            .in_sequence(&mut sequence)
            .return_once(move |_| {
                if succeeds {
                    Ok(())
                } else {
                    Err(crate::actors::ConductorError::Rpc(TransportErrorKind::custom(
                        std::io::Error::new(
                            std::io::ErrorKind::TimedOut,
                            "commit applied, response timed out",
                        ),
                    )))
                }
            });
    }
    actor.conductor = Some(conductor);
    let payload = sealed_payload(2);
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
    assert!(
        matches!(&actor.in_flight, Some(InFlightBlock::Committing(block)) if block.envelope == sealed_payload(2))
    );
    tick(&mut actor).await;
    assert!(actor.in_flight.is_none());
}

#[tokio::test]
async fn local_override_retries_the_same_block_without_another_conductor_commit() {
    let mut conductor = MockConductor::new();
    conductor.expect_commit_unsafe_payload().times(1).return_once(|_| {
        Err(crate::actors::ConductorError::Rpc(RpcError::local_usage_str("quorum unavailable")))
    });
    conductor.expect_leader().times(1).return_once(|| Ok(false));
    let (mut actor, _) = test_actor_with_config(true, false, Some(conductor));
    actor.started = true;
    actor.in_flight = Some(InFlightBlock::Building(pending_handle()));
    actor.unsafe_payload_gossip_client.expect_has_capacity().return_const(true);
    actor.engine_client.expect_get_unsafe_head().returning(|| Ok(L2BlockInfo::default()));
    actor
        .origin_selector
        .expect_next_l1_origin()
        .returning(|_, _| Err(L1OriginSelectorError::NotEnoughData(BlockInfo::default())));
    let mut sequence = Sequence::new();
    actor
        .engine_client
        .expect_seal_block()
        .withf(|id, _| *id == PayloadId::new([1; 8]))
        .times(1)
        .in_sequence(&mut sequence)
        .return_once(|_, _| Ok(sealed_payload(2)));
    let payload = sealed_payload(2);
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
    assert!(
        matches!(&actor.in_flight, Some(InFlightBlock::Committing(block)) if block.envelope == sealed_payload(2))
    );
    actor.override_leader().await.unwrap();
    tick(&mut actor).await;
    assert!(actor.in_flight.is_none());
    assert!(actor.admin_state_receiver().borrow().conductor_enabled);
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
#[case::uncommitted_stale(true, false)]
#[case::uncommitted_invalid(false, false)]
#[case::conductor_stale(true, true)]
#[case::conductor_invalid(false, true)]
#[tokio::test]
async fn stale_or_invalid_payload_fails_if_conductor_committed(
    #[case] stale: bool,
    #[case] committed: bool,
) {
    let mut actor = test_actor();
    actor.started = true;
    actor.in_flight = Some(InFlightBlock::Committed(ConductorCommittedBlock {
        envelope: sealed_payload(2),
        parent: L2BlockInfo::default(),
        tx_count: 0,
        committed,
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
    let result = actor.step().await;
    if committed {
        assert!(matches!(
            result,
            Err(SequencerActorError::EngineError(EngineClientError::CanonicalizeError(_)))
        ));
    } else {
        result.unwrap();
    }
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

/// Reset must retain the exact conductor-committed envelope for another canonicalization attempt.
#[rstest]
#[case::reset_succeeds(true)]
#[case::reset_fails(false)]
#[tokio::test(start_paused = true)]
async fn invalid_forkchoice_resets_before_retrying_committed_payload(#[case] reset_succeeds: bool) {
    let mut actor = test_actor();
    actor.started = true;
    actor.conductor = Some(MockConductor::new());
    let payload = sealed_payload(2);
    actor.in_flight = Some(InFlightBlock::Committed(ConductorCommittedBlock {
        envelope: payload.clone(),
        parent: L2BlockInfo::default(),
        tx_count: 0,
        committed: true,
    }));
    actor.engine_client.expect_seal_block().times(0);
    let mut sequence = Sequence::new();
    actor
        .engine_client
        .expect_canonicalize_block()
        .times(1)
        .in_sequence(&mut sequence)
        .return_once(|_, _| {
            Err(EngineClientError::CanonicalizeError(
                CanonicalizeTaskError::PayloadInsertionFailed(Box::new(
                    InsertTaskError::ForkchoiceUpdateFailed(
                        SynchronizeTaskError::InvalidForkchoiceState,
                    ),
                )),
            ))
        });
    actor
        .engine_client
        .expect_reset_engine_forkchoice()
        .times(1)
        .in_sequence(&mut sequence)
        .return_once(move || {
            if reset_succeeds {
                Ok(())
            } else {
                Err(EngineClientError::ResetForkchoiceError("reset failed".into()))
            }
        });
    actor.unsafe_payload_gossip_client.expect_has_capacity().return_const(true);

    let result = actor.step().await;
    if !reset_succeeds {
        assert!(matches!(
            result,
            Err(SequencerActorError::EngineError(EngineClientError::ResetForkchoiceError(_)))
        ));
        return;
    }
    result.unwrap();
    assert!(matches!(actor.in_flight, Some(InFlightBlock::Committed(_))));

    let expected = payload.clone();
    actor
        .engine_client
        .expect_canonicalize_block()
        .withf(move |envelope, parent| envelope == &expected && *parent == L2BlockInfo::default())
        .times(1)
        .in_sequence(&mut sequence)
        .return_once(|_, _| Ok(()));
    let expected = payload.clone();
    actor
        .unsafe_payload_gossip_client
        .expect_schedule_execution_payload_gossip()
        .withf(move |envelope| envelope == &expected)
        .times(1)
        .return_once(|_| Ok(()));
    actor
        .engine_client
        .expect_get_unsafe_head()
        .times(1)
        .return_once(|| Ok(L2BlockInfo::default()));
    actor
        .origin_selector
        .expect_next_l1_origin()
        .times(1)
        .return_once(|_, _| Err(L1OriginSelectorError::NotEnoughData(BlockInfo::default())));
    actor.engine_client.expect_start_build_block().times(0);

    tick(&mut actor).await;
    assert!(actor.in_flight.is_none());
}

#[tokio::test(start_paused = true)]
async fn changed_parent_after_reset_cannot_replace_a_conductor_committed_payload() {
    let mut actor = test_actor();
    actor.started = true;
    actor.in_flight = Some(InFlightBlock::Committed(ConductorCommittedBlock {
        envelope: sealed_payload(2),
        parent: L2BlockInfo::default(),
        tx_count: 0,
        committed: true,
    }));
    actor.unsafe_payload_gossip_client.expect_has_capacity().return_const(true);
    actor.unsafe_payload_gossip_client.expect_schedule_execution_payload_gossip().times(0);
    actor.engine_client.expect_start_build_block().times(0);
    let mut sequence = Sequence::new();
    actor
        .engine_client
        .expect_canonicalize_block()
        .times(1)
        .in_sequence(&mut sequence)
        .return_once(|_, _| {
            Err(EngineClientError::CanonicalizeError(
                CanonicalizeTaskError::PayloadInsertionFailed(Box::new(
                    InsertTaskError::ForkchoiceUpdateFailed(
                        SynchronizeTaskError::InvalidForkchoiceState,
                    ),
                )),
            ))
        });
    actor
        .engine_client
        .expect_reset_engine_forkchoice()
        .times(1)
        .in_sequence(&mut sequence)
        .return_once(|| Ok(()));
    actor
        .engine_client
        .expect_canonicalize_block()
        .times(1)
        .in_sequence(&mut sequence)
        .return_once(|_, _| {
            Err(EngineClientError::CanonicalizeError(
                CanonicalizeTaskError::UnsafeHeadChangedSinceBuild,
            ))
        });

    tick(&mut actor).await;
    assert!(matches!(actor.in_flight, Some(InFlightBlock::Committed(_))));
    actor.build_ticker.reset_immediately();
    assert!(matches!(
        actor.step().await,
        Err(SequencerActorError::EngineError(EngineClientError::CanonicalizeError(
            CanonicalizeTaskError::UnsafeHeadChangedSinceBuild
        )))
    ));
    assert!(actor.in_flight.is_none());
}

/// A conductor-committed INVALID payload cannot be replaced at the same height, including when
/// the payload contains only deposits.
#[rstest]
#[case::deposits_only(true)]
#[case::with_user_transaction(false)]
#[tokio::test(start_paused = true)]
async fn invalid_sequenced_payload_stops_without_rebuilding(#[case] deposits_only: bool) {
    let mut block = Block::<OpTxEnvelope>::default();
    block.header.number = 1;
    block.header.timestamp = 2;
    block.body.transactions.push(OpTxEnvelope::Deposit(Sealed::new(TxDeposit {
        input: kona_protocol::L1BlockInfoBedrock::default().encode_calldata(),
        ..Default::default()
    })));
    if !deposits_only {
        block.body.transactions.push(OpTxEnvelope::Eip1559(
            TxEip1559::default().into_signed(Signature::test_signature()),
        ));
    }
    assert_eq!(block.body.transactions.iter().all(OpTxEnvelope::is_deposit), deposits_only);
    let transactions = block.body.transactions.iter().map(|tx| tx.encoded_2718().into()).collect();
    let rejected = OpExecutionPayloadEnvelope::V1(ExecutionPayloadV1::from_block_slow(&block));
    let mut actor = test_actor();
    actor.started = true;
    let mut handle = pending_handle();
    handle.attributes_with_parent.attributes.transactions = Some(transactions);
    actor.in_flight = Some(InFlightBlock::Building(handle));
    actor.unsafe_payload_gossip_client.expect_has_capacity().return_const(true);
    actor.unsafe_payload_gossip_client.expect_schedule_execution_payload_gossip().times(0);
    actor.engine_client.expect_reset_engine_forkchoice().times(0);
    let mut sequence = Sequence::new();
    let sealed = rejected.clone();
    actor
        .engine_client
        .expect_seal_block()
        .withf(|id, _| *id == PayloadId::new([1; 8]))
        .times(1)
        .in_sequence(&mut sequence)
        .return_once(move |_, _| Ok(sealed));
    let mut conductor = MockConductor::new();
    let expected = rejected.clone();
    conductor
        .expect_commit_unsafe_payload()
        .withf(move |payload| payload == &expected)
        .times(1)
        .in_sequence(&mut sequence)
        .return_once(|_| Ok(()));
    actor
        .engine_client
        .expect_canonicalize_block()
        .withf(move |payload, _| payload == &rejected)
        .times(1)
        .in_sequence(&mut sequence)
        .return_once(|_, _| {
            Err(EngineClientError::CanonicalizeError(
                CanonicalizeTaskError::PayloadInsertionFailed(Box::new(
                    InsertTaskError::UnexpectedPayloadStatus(PayloadStatusEnum::Invalid {
                        validation_error: "invalid sequenced block".into(),
                    }),
                )),
            ))
        });
    actor.engine_client.expect_get_unsafe_head().times(0);
    actor.engine_client.expect_start_build_block().times(0);
    actor.conductor = Some(conductor);

    assert!(matches!(
        actor.step().await,
        Err(SequencerActorError::EngineError(EngineClientError::CanonicalizeError(
            CanonicalizeTaskError::PayloadInsertionFailed(_)
        )))
    ));
}
