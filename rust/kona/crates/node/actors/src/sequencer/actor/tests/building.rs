use super::{test_actor, test_actor_with_config};
use crate::{
    MockOriginSelector, MockSequencerEngineClient, MockUnsafePayloadGossipClient,
    sequencer::{ActorError, handle::Message},
};
use kona_derive::{BuilderError, PipelineErrorKind, test_utils::TestAttributesBuilder};
use kona_protocol::{BlockInfo, L2BlockInfo};
use rstest::rstest;
use std::sync::{
    Arc,
    atomic::{AtomicUsize, Ordering},
};
use tokio::{
    sync::oneshot,
    time::{self, Duration},
};

#[rstest]
#[case::temp(PipelineErrorKind::Temporary(BuilderError::Custom(String::new()).into()), false)]
#[case::reset(PipelineErrorKind::Reset(BuilderError::Custom(String::new()).into()), false)]
#[case::critical(PipelineErrorKind::Critical(BuilderError::Custom(String::new()).into()), true)]
#[tokio::test(start_paused = true)]
async fn build_handles_payload_attributes_errors(
    #[case] forced_error: PipelineErrorKind,
    #[case] expect_err: bool,
) {
    let mut client = MockSequencerEngineClient::new();

    let unsafe_head = L2BlockInfo::default();
    client.expect_get_unsafe_head().times(1).return_once(move || Ok(unsafe_head));
    // Must not be called on critical error
    client.expect_start_build_block().times(0);
    let resets = usize::from(matches!(&forced_error, PipelineErrorKind::Reset(_)));
    client.expect_reset_engine_forkchoice().times(resets).returning(|| Ok(()));

    let l1_origin = BlockInfo::default();
    let mut origin_selector = MockOriginSelector::new();
    origin_selector.expect_next_l1_origin().times(1).return_once(move |_, _| Ok(l1_origin));

    let attributes_builder = TestAttributesBuilder { attributes: vec![Err(forced_error)] };

    let mut actor = test_actor();
    actor.origin_selector = origin_selector;
    actor.engine_client = client;
    actor.attributes_builder = attributes_builder;

    actor.unsafe_payload_gossip_client.expect_has_capacity().return_const(true);
    let result = actor.build().await;
    if expect_err {
        assert!(result.is_err());
        assert!(matches!(
            result.unwrap_err(),
            ActorError::AttributesBuilder(PipelineErrorKind::Critical(_))
        ));
    } else {
        assert!(result.is_ok());
    }
}

/// A full gossip queue, for example during a signer outage, pauses block building without
/// blocking the actor, so admin queries such as op-conductor's `StopSequencer` are still answered.
#[tokio::test(start_paused = true)]
async fn full_gossip_queue_pauses_building_but_admin_queries_are_answered() {
    let (mut actor, _, handle) = test_actor_with_config(true, false, None);
    let mut engine = MockSequencerEngineClient::new();
    // No block is built or sealed while the queue is full.
    engine.expect_start_build_block().times(0);
    engine.expect_get_unsafe_head().times(1).return_once(|| Ok(L2BlockInfo::default()));
    actor.engine_client = engine;

    let mut gossip = MockUnsafePayloadGossipClient::new();
    gossip.expect_has_capacity().return_const(false);
    gossip.expect_schedule_execution_payload_gossip().times(0);
    actor.unsafe_payload_gossip_client = gossip;

    // Building returns instead of waiting for space in the gossip queue.
    time::timeout(Duration::from_secs(10), actor.build()).await.unwrap().unwrap();

    let (tx, rx) = oneshot::channel();
    time::timeout(Duration::from_secs(10), actor.handle_message(Message::StopSequencer(tx)))
        .await
        .unwrap();
    assert_eq!(rx.await.unwrap().unwrap(), L2BlockInfo::default().hash());
    assert!(!handle.snapshot().unwrap().active);
}

/// Block building resumes after the gossip queue has room again.
#[tokio::test(start_paused = true)]
async fn building_resumes_once_the_gossip_queue_drains() {
    let mut actor = test_actor();

    let mut engine = MockSequencerEngineClient::new();
    engine.expect_get_unsafe_head().times(1).return_once(|| Ok(L2BlockInfo::default()));
    actor.engine_client = engine;

    // Building starts again on the second attempt: the origin selector is consulted once.
    let mut origin_selector = MockOriginSelector::new();
    origin_selector.expect_next_l1_origin().times(1).return_once(|_, _| Ok(BlockInfo::default()));
    actor.origin_selector = origin_selector;
    actor.attributes_builder = TestAttributesBuilder {
        attributes: vec![Err(PipelineErrorKind::Temporary(
            BuilderError::Custom(String::new()).into(),
        ))],
    };

    // Full on the first attempt, with room on the second.
    let ticks = Arc::new(AtomicUsize::new(0));
    let mut gossip = MockUnsafePayloadGossipClient::new();
    gossip.expect_has_capacity().returning(move || ticks.fetch_add(1, Ordering::SeqCst) > 0);
    actor.unsafe_payload_gossip_client = gossip;

    time::timeout(Duration::from_secs(10), actor.build()).await.unwrap().unwrap();
    time::timeout(Duration::from_secs(10), actor.build()).await.unwrap().unwrap();
}
