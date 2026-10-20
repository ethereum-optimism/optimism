use super::{
    super::{
        actor::{PendingBuild, UnsealedPayloadHandle},
        handle::Message,
    },
    test_util::{install_engine, known_head, run_command, test_actor_with_handle},
};
use crate::{
    BuildToken, ConductorError, EngineClientError, EngineContext, NodeActor, Sealed,
    actors::{MockConductor, engine::handle::Message as EngineMessage},
};
use alloy_rpc_types_engine::{ExecutionPayloadV1, PayloadId};
use alloy_transport::RpcError;
use kona_protocol::OpAttributesWithParent;
use kona_rpc::SequencerAdminAPIClient;
use op_alloy_consensus::OpBlock;
use op_alloy_rpc_types_engine::{OpExecutionPayloadEnvelope, OpPayloadAttributes};
use rstest::rstest;
use tokio::{
    sync::oneshot,
    time::{self, Duration},
};

fn payload(marker: u8) -> OpExecutionPayloadEnvelope {
    let mut payload = ExecutionPayloadV1::from_block_slow(&OpBlock::default());
    payload.parent_hash = known_head().hash();
    payload.block_number = known_head().block_info.number + 1;
    payload.timestamp = known_head().block_info.timestamp + 2;
    payload.extra_data = vec![marker].into();
    payload.block_hash = payload.clone().into_block_raw().unwrap().header.hash_slow();
    OpExecutionPayloadEnvelope::V1(payload)
}

fn pending(payload: &OpExecutionPayloadEnvelope, committed: bool) -> UnsealedPayloadHandle {
    let context = EngineContext { epoch: 0, parent: known_head() };
    let mut attributes = OpPayloadAttributes::default();
    attributes.payload_attributes.timestamp = payload.timestamp();
    UnsealedPayloadHandle {
        job: if committed {
            PendingBuild::Committed(Box::new(Sealed::test(context, payload.clone())))
        } else {
            PendingBuild::Building(BuildToken::test(
                context,
                PayloadId::new([1; 8]),
                payload.timestamp(),
            ))
        },
        attributes_with_parent: OpAttributesWithParent::new(attributes, known_head(), None, false),
    }
}

#[tokio::test(start_paused = true)]
async fn commit_failure_keeps_the_job_reseals_after_one_second_and_publishes_nothing() {
    let first = payload(1);
    let second = payload(2);
    let mut conductor = MockConductor::new();
    let mut sequence = mockall::Sequence::new();
    let expected = first.clone();
    conductor
        .expect_commit_unsafe_payload()
        .times(1)
        .in_sequence(&mut sequence)
        .withf(move |payload| payload == &expected)
        .return_once(|_| {
            Err(ConductorError::Rpc(RpcError::local_usage_str("acknowledgement lost")))
        });
    let expected = second.clone();
    conductor
        .expect_commit_unsafe_payload()
        .times(1)
        .in_sequence(&mut sequence)
        .withf(move |payload| payload == &expected)
        .return_once(|_| Ok(()));
    let (mut actor, handle) = test_actor_with_handle(false, false, Some(conductor));
    actor.state.active = true;
    actor.publish();
    let mut engine = install_engine(&mut actor);
    actor.test_pending(pending(&first, false));
    actor.origin_selector.expect_next_l1_origin().times(0);
    let expected = second.clone();
    actor
        .unsafe_payload_gossip_client
        .expect_schedule_execution_payload_gossip()
        .times(1)
        .withf(move |payload| payload == &expected)
        .return_once(|_| Ok(()));
    actor.unsafe_payload_gossip_client.expect_has_capacity().return_const(true);
    let expected = second.clone();
    let peer = tokio::spawn(async move {
        let EngineMessage::Seal(_, reply) = engine.commands.recv().await.unwrap() else {
            panic!("expected seal")
        };
        reply
            .send(Ok(Sealed::test(EngineContext { epoch: 0, parent: known_head() }, first)))
            .unwrap();
        let EngineMessage::Seal(_, reply) = engine.commands.recv().await.unwrap() else {
            panic!("must reseal before acknowledgement")
        };
        reply
            .send(Ok(Sealed::test(EngineContext { epoch: 0, parent: known_head() }, second)))
            .unwrap();
        let EngineMessage::Canonicalize(sealed, reply) = engine.commands.recv().await.unwrap()
        else {
            panic!("expected acknowledged import")
        };
        assert_eq!(sealed.payload(), &expected);
        // Queue Stop while import is in flight. A new build must not overtake it.
        let stop = handle.stop_sequencer();
        tokio::pin!(stop);
        assert!(futures::poll!(stop.as_mut()).is_pending());
        let mut head = known_head();
        head.block_info.hash = expected.block_hash();
        head.block_info.number = expected.block_number();
        head.block_info.timestamp = expected.timestamp();
        engine.state.send_modify(|state| {
            state.sync_state = state.sync_state.apply_update(kona_engine::EngineSyncStateUpdate {
                unsafe_head: Some(head),
                ..Default::default()
            })
        });
        reply.send(Ok(expected.clone())).unwrap();
        assert_eq!(stop.await.unwrap(), expected.block_hash());
        assert!(engine.commands.try_recv().is_err());
    });
    actor.step().await.unwrap();
    assert!(actor.has_pending());
    assert_eq!(actor.engine_client.state().sync_state.unsafe_head(), known_head());
    assert!(time::timeout(Duration::from_millis(500), actor.step()).await.is_err());
    actor.step().await.unwrap();
    peer.await.unwrap();
    assert!(!actor.has_pending());
    assert!(!actor.state.active);
}

#[rstest]
#[case::managed(false)]
#[case::local_override(true)]
#[tokio::test]
async fn acknowledged_import_retry_uses_only_the_exact_envelope(#[case] local_override: bool) {
    let payload = payload(3);
    let mut conductor = MockConductor::new();
    let expected = payload.clone();
    conductor
        .expect_commit_unsafe_payload()
        .times(if local_override { 0 } else { 1 })
        .withf(move |payload| payload == &expected)
        .return_once(|_| Ok(()));
    conductor.expect_leader().times(0);
    let (mut actor, handle) = test_actor_with_handle(false, false, Some(conductor));
    let mut engine = install_engine(&mut actor);
    if local_override {
        run_command(&mut actor, handle.override_leader()).await.unwrap();
    }
    let expected = payload.clone();
    actor
        .unsafe_payload_gossip_client
        .expect_schedule_execution_payload_gossip()
        .times(1)
        .withf(move |payload| payload == &expected)
        .return_once(|_| Ok(()));
    let expected = payload.clone();
    let peer = tokio::spawn(async move {
        let EngineMessage::Seal(_, reply) = engine.commands.recv().await.unwrap() else {
            panic!("expected seal")
        };
        reply
            .send(Ok(Sealed::test(
                EngineContext { epoch: 0, parent: known_head() },
                expected.clone(),
            )))
            .unwrap();
        for temporary in [true, false] {
            let EngineMessage::Canonicalize(sealed, reply) = engine.commands.recv().await.unwrap()
            else {
                panic!("must retry import without retrieval")
            };
            assert_eq!(sealed.payload(), &expected);
            reply
                .send(if temporary {
                    Err(EngineClientError::Temporary("EL unavailable".into()))
                } else {
                    Ok(expected.clone())
                })
                .unwrap();
        }
    });
    let mut pending = pending(&payload, false);
    assert!(matches!(
        actor.publish_payload(&mut pending).await,
        Err(crate::SequencerActorError::EngineError(EngineClientError::Temporary(_)))
    ));
    assert!(matches!(pending.job, PendingBuild::Committed(_)));
    assert!(actor.publish_payload(&mut pending).await.unwrap());
    peer.await.unwrap();
}

#[tokio::test]
async fn unmanaged_publication_imports_without_a_conductor() {
    let payload = payload(4);
    let (mut actor, _handle) = test_actor_with_handle(true, false, None);
    let mut engine = install_engine(&mut actor);
    actor
        .unsafe_payload_gossip_client
        .expect_schedule_execution_payload_gossip()
        .times(1)
        .return_once(|_| Ok(()));
    let expected = payload.clone();
    let peer = tokio::spawn(async move {
        let EngineMessage::Seal(_, reply) = engine.commands.recv().await.unwrap() else {
            panic!("expected seal")
        };
        reply
            .send(Ok(Sealed::test(
                EngineContext { epoch: 0, parent: known_head() },
                expected.clone(),
            )))
            .unwrap();
        let EngineMessage::Canonicalize(sealed, reply) = engine.commands.recv().await.unwrap()
        else {
            panic!("expected import")
        };
        assert_eq!(sealed.payload(), &expected);
        reply.send(Ok(expected)).unwrap();
    });
    assert!(actor.publish_payload(&mut pending(&payload, false)).await.unwrap());
    peer.await.unwrap();
}

#[rstest]
#[case::expired(EngineClientError::Expired)]
#[case::stale(EngineClientError::Stale)]
#[case::invalid(EngineClientError::Rejected("invalid".into()))]
#[tokio::test]
async fn unusable_jobs_are_discarded_without_retrieval_loops_or_publication(
    #[case] error: EngineClientError,
) {
    let (mut actor, _handle) = test_actor_with_handle(true, false, None);
    let mut engine = install_engine(&mut actor);
    actor.test_pending(pending(&payload(5), false));
    actor.origin_selector.expect_next_l1_origin().times(0);
    actor.unsafe_payload_gossip_client.expect_has_capacity().return_const(true);
    actor.unsafe_payload_gossip_client.expect_schedule_execution_payload_gossip().times(0);
    let peer = tokio::spawn(async move {
        let EngineMessage::Seal(_, reply) = engine.commands.recv().await.unwrap() else {
            panic!("expected seal")
        };
        reply.send(Err(error)).unwrap();
        engine
    });
    actor.step().await.unwrap();
    let mut engine = peer.await.unwrap();
    assert!(!actor.has_pending());
    assert!(engine.commands.try_recv().is_err());
}

#[tokio::test]
async fn same_head_reset_between_retrieval_and_commit_forbids_publication() {
    let (mut actor, _handle) = test_actor_with_handle(false, false, Some(MockConductor::new()));
    let mut engine = install_engine(&mut actor);
    actor.unsafe_payload_gossip_client.expect_schedule_execution_payload_gossip().times(0);
    let payload = payload(6);
    let expected = payload.clone();
    let peer = tokio::spawn(async move {
        let EngineMessage::Seal(_, reply) = engine.commands.recv().await.unwrap() else {
            panic!("expected seal")
        };
        engine.state.send_modify(|state| state.epoch += 1);
        reply
            .send(Ok(Sealed::test(EngineContext { epoch: 0, parent: known_head() }, expected)))
            .unwrap();
    });
    assert!(matches!(
        actor.publish_payload(&mut pending(&payload, false)).await,
        Err(crate::SequencerActorError::EngineError(EngineClientError::Stale))
    ));
    peer.await.unwrap();
}

#[rstest]
#[case::building(false, false)]
#[case::reset_committed(true, true)]
#[tokio::test]
async fn stop_clears_uncommitted_or_reset_jobs_before_restart(
    #[case] committed: bool,
    #[case] reset: bool,
) {
    let (mut actor, handle) = test_actor_with_handle(true, false, None);
    let mut engine = install_engine(&mut actor);
    actor.test_pending(pending(&payload(7), committed));
    if reset {
        engine.state.send_modify(|state| state.epoch += 1);
    }
    actor.unsafe_payload_gossip_client.expect_schedule_execution_payload_gossip().times(0);
    assert_eq!(
        run_command(&mut actor, handle.stop_sequencer()).await.unwrap(),
        known_head().hash()
    );
    assert!(!actor.has_pending());
    run_command(&mut actor, handle.start_sequencer(known_head().hash())).await.unwrap();
    assert!(!actor.has_pending());
    assert!(engine.commands.try_recv().is_err());
}

#[rstest]
#[case::reply_waited(false, false)]
#[case::reply_abandoned(true, false)]
#[case::local_override(false, true)]
#[tokio::test(start_paused = true)]
async fn stop_retries_exact_committed_import_and_gossips_before_acknowledging(
    #[case] abandoned: bool,
    #[case] local_override: bool,
) {
    let payload = payload(8);
    let mut conductor = MockConductor::new();
    conductor.expect_leader().times(if local_override { 0 } else { 2 }).returning(|| Ok(true));
    let (mut actor, handle) = test_actor_with_handle(false, false, Some(conductor));
    actor.state.active = true;
    actor.publish();
    let mut engine = install_engine(&mut actor);
    actor.test_pending(pending(&payload, true));
    if local_override {
        run_command(&mut actor, handle.override_leader()).await.unwrap();
    }
    let gossiped = std::sync::Arc::new(std::sync::atomic::AtomicBool::new(false));
    let seen = gossiped.clone();
    let expected = payload.clone();
    actor
        .unsafe_payload_gossip_client
        .expect_schedule_execution_payload_gossip()
        .times(1)
        .withf(move |payload| payload == &expected)
        .return_once(move |_| {
            seen.store(true, std::sync::atomic::Ordering::SeqCst);
            Ok(())
        });
    let expected = payload.clone();
    let reads = handle.clone();
    let peer = tokio::spawn(async move {
        for temporary in [true, false] {
            let EngineMessage::Canonicalize(sealed, reply) = engine.commands.recv().await.unwrap()
            else {
                panic!("expected exact import")
            };
            assert_eq!(sealed.payload(), &expected);
            assert!(
                reads.is_sequencer_active().await.unwrap(),
                "inactive publication must wait for settlement"
            );
            if temporary {
                reply.send(Err(EngineClientError::Temporary("retry".into()))).unwrap();
            } else {
                let mut head = known_head();
                head.block_info.hash = expected.block_hash();
                head.block_info.number = expected.block_number();
                head.block_info.timestamp = expected.timestamp();
                engine.state.send_modify(|state| {
                    state.sync_state =
                        state.sync_state.apply_update(kona_engine::EngineSyncStateUpdate {
                            unsafe_head: Some(head),
                            ..Default::default()
                        })
                });
                reply.send(Ok(expected.clone())).unwrap();
            }
        }
    });
    let (reply, result) = oneshot::channel();
    let result = if abandoned {
        drop(result);
        None
    } else {
        Some(result)
    };
    actor.handle_command(Message::Stop(reply)).await.unwrap();
    if let Some(result) = result {
        assert_eq!(result.await.unwrap().unwrap(), payload.block_hash());
    }
    peer.await.unwrap();
    assert!(gossiped.load(std::sync::atomic::Ordering::SeqCst));
    assert!(!handle.is_sequencer_active().await.unwrap());
    assert!(!actor.has_pending());
}

#[rstest]
#[case::lost_leadership(false, false)]
#[case::failed_query(true, false)]
#[case::loss_during_retry(false, true)]
#[tokio::test(start_paused = true)]
async fn stop_ends_settlement_on_leadership_loss_or_query_failure(
    #[case] fails: bool,
    #[case] retry: bool,
) {
    let mut conductor = MockConductor::new();
    let mut sequence = mockall::Sequence::new();
    if retry {
        conductor.expect_leader().times(1).in_sequence(&mut sequence).return_once(|| Ok(true));
    }
    conductor.expect_leader().times(1).in_sequence(&mut sequence).return_once(move || {
        if fails {
            Err(ConductorError::Rpc(RpcError::local_usage_str("quorum unavailable")))
        } else {
            Ok(false)
        }
    });
    let (mut actor, handle) = test_actor_with_handle(false, false, Some(conductor));
    actor.state.active = true;
    actor.publish();
    let mut engine = install_engine(&mut actor);
    actor.test_pending(pending(&payload(9), true));
    actor.unsafe_payload_gossip_client.expect_schedule_execution_payload_gossip().times(0);
    let peer = async {
        if retry {
            let EngineMessage::Canonicalize(_, reply) = engine.commands.recv().await.unwrap()
            else {
                panic!("expected initial import")
            };
            reply.send(Err(EngineClientError::Temporary("retry".into()))).unwrap();
        }
    };
    let (result, ()) = tokio::join!(run_command(&mut actor, handle.stop_sequencer()), peer);
    assert_eq!(result.unwrap(), known_head().hash());
    assert!(!actor.has_pending());
    assert!(engine.commands.try_recv().is_err());
}

/// A held commit lets the test enqueue Stop before the positive acknowledgement arrives.
#[derive(Debug)]
struct HeldCommit {
    entered: std::sync::Arc<tokio::sync::Notify>,
    release: std::sync::Arc<tokio::sync::Notify>,
}

#[async_trait::async_trait]
impl crate::Conductor for HeldCommit {
    async fn leader(&self) -> Result<bool, ConductorError> {
        Ok(true)
    }

    async fn commit_unsafe_payload(
        &self,
        _: &OpExecutionPayloadEnvelope,
    ) -> Result<(), ConductorError> {
        self.entered.notify_one();
        self.release.notified().await;
        Ok(())
    }
}

#[tokio::test]
async fn stop_queued_during_commit_settles_the_late_ack_before_any_new_build() {
    let payload = payload(10);
    let entered = std::sync::Arc::new(tokio::sync::Notify::new());
    let release = std::sync::Arc::new(tokio::sync::Notify::new());
    let mut state =
        kona_engine::EngineState { phase: kona_engine::EnginePhase::Ready, ..Default::default() };
    state.sync_state = state.sync_state.apply_update(kona_engine::EngineSyncStateUpdate {
        unsafe_head: Some(known_head()),
        ..Default::default()
    });
    let (engine_handle, mut engine) = crate::EngineHandle::test_channel(state);
    let mut gossip = crate::actors::MockUnsafePayloadGossipClient::new();
    gossip.expect_has_capacity().return_const(true);
    let gossiped = std::sync::Arc::new(std::sync::atomic::AtomicBool::new(false));
    let seen = gossiped.clone();
    let expected = payload.clone();
    gossip
        .expect_schedule_execution_payload_gossip()
        .times(1)
        .withf(move |payload| payload == &expected)
        .return_once(move |_| {
            seen.store(true, std::sync::atomic::Ordering::SeqCst);
            Ok(())
        });
    let mut origins = crate::actors::MockOriginSelector::new();
    origins.expect_next_l1_origin().times(0);
    let (mut actor, handle) = crate::SequencerActor::new(
        kona_derive::test_utils::TestAttributesBuilder { attributes: vec![] },
        Some(HeldCommit { entered: entered.clone(), release: release.clone() }),
        engine_handle,
        true,
        false,
        origins,
        std::sync::Arc::new(kona_genesis::RollupConfig { block_time: 2, ..Default::default() }),
        gossip,
    );
    actor.state.active = true;
    actor.publish();
    actor.test_pending(pending(&payload, false));
    let expected = payload.clone();
    let peer = async move {
        let EngineMessage::Seal(_, reply) = engine.commands.recv().await.unwrap() else {
            panic!("expected seal")
        };
        reply
            .send(Ok(Sealed::test(
                EngineContext { epoch: 0, parent: known_head() },
                expected.clone(),
            )))
            .unwrap();
        let EngineMessage::Canonicalize(sealed, reply) = engine.commands.recv().await.unwrap()
        else {
            panic!("expected late acknowledged import")
        };
        assert_eq!(sealed.payload(), &expected);
        let mut head = known_head();
        head.block_info.hash = expected.block_hash();
        head.block_info.number = expected.block_number();
        head.block_info.timestamp = expected.timestamp();
        engine.state.send_modify(|state| {
            state.sync_state = state.sync_state.apply_update(kona_engine::EngineSyncStateUpdate {
                unsafe_head: Some(head),
                ..Default::default()
            })
        });
        reply.send(Ok(expected)).unwrap();
        engine
    };
    let stopping = async {
        entered.notified().await;
        let stop = handle.stop_sequencer();
        tokio::pin!(stop);
        assert!(futures::poll!(stop.as_mut()).is_pending());
        release.notify_one();
        assert_eq!(stop.await.unwrap(), payload.block_hash());
        assert!(gossiped.load(std::sync::atomic::Ordering::SeqCst));
        assert!(!handle.is_sequencer_active().await.unwrap());
    };
    let (step, (), mut engine) = time::timeout(Duration::from_secs(10), async {
        tokio::join!(actor.step(), stopping, peer)
    })
    .await
    .unwrap();
    step.unwrap();
    assert!(!actor.has_pending());
    assert!(engine.commands.try_recv().is_err());
}

#[derive(Debug)]
struct HeldOrigin {
    entered: std::sync::Arc<tokio::sync::Notify>,
    release: std::sync::Arc<tokio::sync::Notify>,
}

#[async_trait::async_trait]
impl crate::OriginSelector for HeldOrigin {
    async fn next_l1_origin(
        &mut self,
        _: kona_protocol::L2BlockInfo,
        _: bool,
    ) -> Result<kona_protocol::BlockInfo, crate::L1OriginSelectorError> {
        self.entered.notify_one();
        self.release.notified().await;
        Ok(kona_protocol::BlockInfo::default())
    }
}

#[tokio::test]
async fn stop_then_start_during_preparation_discards_attributes_even_when_state_matches() {
    let entered = std::sync::Arc::new(tokio::sync::Notify::new());
    let release = std::sync::Arc::new(tokio::sync::Notify::new());
    let mut state =
        kona_engine::EngineState { phase: kona_engine::EnginePhase::Ready, ..Default::default() };
    state.sync_state = state.sync_state.apply_update(kona_engine::EngineSyncStateUpdate {
        unsafe_head: Some(known_head()),
        ..Default::default()
    });
    let (engine_handle, mut engine) = crate::EngineHandle::test_channel(state);
    let (mut actor, handle) = crate::SequencerActor::new(
        kona_derive::test_utils::TestAttributesBuilder {
            attributes: vec![Ok(OpPayloadAttributes::default())],
        },
        None::<MockConductor>,
        engine_handle,
        true,
        false,
        HeldOrigin { entered: entered.clone(), release: release.clone() },
        std::sync::Arc::new(kona_genesis::RollupConfig { block_time: 2, ..Default::default() }),
        crate::actors::MockUnsafePayloadGossipClient::new(),
    );
    let controls = async {
        entered.notified().await;
        let stop = handle.stop_sequencer();
        let start = handle.start_sequencer(known_head().hash());
        tokio::pin!(stop, start);
        assert!(futures::poll!(stop.as_mut()).is_pending());
        assert!(futures::poll!(start.as_mut()).is_pending());
        release.notify_one();
        assert_eq!(stop.await.unwrap(), known_head().hash());
        start.await.unwrap();
    };
    let (prepared, ()) = time::timeout(Duration::from_secs(10), async {
        tokio::join!(actor.build_unsealed_payload(), controls)
    })
    .await
    .unwrap();
    assert!(prepared.unwrap().is_none());
    assert!(handle.is_sequencer_active().await.unwrap());
    assert!(!actor.has_pending());
    assert!(engine.commands.try_recv().is_err(), "old preparation must not reach start_build");
}
