#[cfg(test)]
use crate::{
    NodeActor, SequencerActorError, SequencerAdminQuery,
    actors::{
        MockOriginSelector, MockUnsafePayloadGossipClient, sequencer::tests::test_util::test_actor,
    },
};
use kona_derive::{BuilderError, PipelineErrorKind, test_utils::TestAttributesBuilder};
use kona_protocol::{BlockInfo, L2BlockInfo};
use rstest::rstest;
use std::sync::{
    Arc,
    atomic::{AtomicUsize, Ordering},
};
use tokio::{
    sync::{mpsc, oneshot},
    time::{self, Duration},
};

#[rstest]
#[case::temp(PipelineErrorKind::Temporary(BuilderError::Custom(String::new()).into()), false)]
#[case::reset(PipelineErrorKind::Reset(BuilderError::Custom(String::new()).into()), false)]
#[case::critical(PipelineErrorKind::Critical(BuilderError::Custom(String::new()).into()), true)]
#[tokio::test]
async fn test_build_unsealed_payload_prepare_payload_attributes_error(
    #[case] forced_error: PipelineErrorKind,
    #[case] expect_err: bool,
) {
    let (client, mut engine) = crate::EngineHandle::test_channel(kona_engine::EngineState {
        phase: kona_engine::EnginePhase::Ready,
        ..Default::default()
    });
    let reset = matches!(forced_error, PipelineErrorKind::Reset(_));
    let responder = tokio::spawn(async move {
        if reset {
            let _ = engine.resets.recv().await.unwrap().send(Ok(()));
        }
    });

    let l1_origin = BlockInfo::default();
    let mut origin_selector = MockOriginSelector::new();
    origin_selector.expect_next_l1_origin().times(1).return_once(move |_, _| Ok(l1_origin));

    let attributes_builder = TestAttributesBuilder { attributes: vec![Err(forced_error)] };

    let mut actor = test_actor();
    actor.origin_selector = origin_selector;
    actor.engine_client = client;
    actor.attributes_builder = attributes_builder;

    let result = actor.build_unsealed_payload().await;
    responder.await.unwrap();
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

/// A full gossip queue, for example during a signer outage, pauses block building without
/// blocking the actor, so admin queries such as op-conductor's `StopSequencer` are still answered.
#[tokio::test(start_paused = true)]
async fn full_gossip_queue_pauses_building_but_admin_queries_are_answered() {
    let mut actor = test_actor();
    let (admin_tx, admin_rx) = mpsc::channel(1);
    actor.admin_api_rx = admin_rx;

    let (handle, mut engine) = crate::EngineHandle::test_channel(kona_engine::EngineState {
        phase: kona_engine::EnginePhase::Ready,
        ..Default::default()
    });
    actor.engine_client = handle;
    let responder = tokio::spawn(async move {
        let _ = engine.resets.recv().await.unwrap().send(Ok(()));
    });

    let mut gossip = MockUnsafePayloadGossipClient::new();
    gossip.expect_has_capacity().return_const(false);
    gossip.expect_schedule_execution_payload_gossip().times(0);
    actor.unsafe_payload_gossip_client = gossip;

    // The build tick finds the queue full and returns instead of waiting for space.
    time::timeout(Duration::from_secs(10), actor.step()).await.unwrap().unwrap();
    responder.await.unwrap();

    let (tx, rx) = oneshot::channel();
    admin_tx.send(SequencerAdminQuery::StopSequencer(tx)).await.unwrap();
    time::timeout(Duration::from_secs(10), actor.step()).await.unwrap().unwrap();
    assert_eq!(rx.await.unwrap().unwrap(), L2BlockInfo::default().hash());
    assert!(!actor.is_active);
}

/// Block building resumes on the first tick after the gossip queue has room again.
#[tokio::test(start_paused = true)]
async fn building_resumes_once_the_gossip_queue_drains() {
    let mut actor = test_actor();

    let (handle, mut engine) = crate::EngineHandle::test_channel(kona_engine::EngineState {
        phase: kona_engine::EnginePhase::Ready,
        ..Default::default()
    });
    actor.engine_client = handle;
    let responder = tokio::spawn(async move {
        let _ = engine.resets.recv().await.unwrap().send(Ok(()));
    });

    // Building starts again on the second tick: the origin selector is consulted once.
    let mut origin_selector = MockOriginSelector::new();
    origin_selector.expect_next_l1_origin().times(1).return_once(|_, _| Ok(BlockInfo::default()));
    actor.origin_selector = origin_selector;
    actor.attributes_builder = TestAttributesBuilder {
        attributes: vec![Err(PipelineErrorKind::Temporary(
            BuilderError::Custom(String::new()).into(),
        ))],
    };

    // Full on the first tick, with room on the second.
    let ticks = Arc::new(AtomicUsize::new(0));
    let mut gossip = MockUnsafePayloadGossipClient::new();
    gossip.expect_has_capacity().returning(move || ticks.fetch_add(1, Ordering::SeqCst) > 0);
    actor.unsafe_payload_gossip_client = gossip;

    time::timeout(Duration::from_secs(10), actor.step()).await.unwrap().unwrap();
    responder.await.unwrap();
    time::timeout(Duration::from_secs(10), actor.step()).await.unwrap().unwrap();
}

/// Attributes prepared across an asynchronous origin read retain their original epoch.
#[tokio::test]
async fn build_captures_context_before_preparing_attributes() {
    use crate::{EngineClientError, EngineHandle, actors::engine::handle::Message};
    use kona_engine::{EnginePhase, EngineState};
    let original = EngineState { epoch: 3, phase: EnginePhase::Ready, ..Default::default() };
    let (handle, mut engine) = EngineHandle::test_channel(original);
    let state_tx = engine.state.clone();
    let mut actor = test_actor();
    actor.engine_client = handle;
    actor.origin_selector.expect_next_l1_origin().return_once(move |_, _| {
        state_tx.send_modify(|state| state.epoch += 1);
        Ok(BlockInfo::default())
    });
    actor.attributes_builder = TestAttributesBuilder { attributes: vec![Ok(Default::default())] };
    let responder = tokio::spawn(async move {
        let Message::Start(context, _, reply) = engine.commands.recv().await.unwrap() else {
            panic!("expected start")
        };
        assert_eq!(context.epoch, 3);
        reply.send(Err(EngineClientError::Stale)).unwrap();
    });
    assert!(matches!(
        actor.build_unsealed_payload().await,
        Err(SequencerActorError::EngineError(EngineClientError::Stale))
    ));
    responder.await.unwrap();
}

#[tokio::test(start_paused = true)]
async fn temporary_build_failure_waits_for_a_paced_tick() {
    use crate::{EngineClientError, EngineHandle, actors::engine::handle::Message};
    use kona_engine::{EnginePhase, EngineState};
    let (handle, mut engine) =
        EngineHandle::test_channel(EngineState { phase: EnginePhase::Ready, ..Default::default() });
    let mut actor = test_actor();
    actor.engine_client = handle;
    actor
        .origin_selector
        .expect_next_l1_origin()
        .times(1)
        .returning(|_, _| Ok(BlockInfo::default()));
    actor.attributes_builder = TestAttributesBuilder { attributes: vec![Ok(Default::default())] };
    actor.unsafe_payload_gossip_client.expect_has_capacity().return_const(true);
    let responder = tokio::spawn(async move {
        engine.resets.recv().await.unwrap().send(Ok(())).unwrap();
        let Message::Start(_, _, reply) = engine.commands.recv().await.unwrap() else {
            panic!("expected start")
        };
        reply.send(Err(EngineClientError::Temporary("syncing".into()))).unwrap();
        engine
    });
    actor.step().await.unwrap();
    let mut engine = responder.await.unwrap();
    assert!(time::timeout(Duration::from_millis(500), actor.step()).await.is_err());
    assert!(engine.commands.try_recv().is_err());
}
