use super::test_actor;
use crate::{
    MockOriginSelector, MockSequencerEngineClient, MockUnsafePayloadGossipClient, NodeActor,
    SequencerActorError, sequencer::SequencerAdminCommand,
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
#[tokio::test(start_paused = true)]
async fn step_handles_payload_attributes_errors(
    #[case] forced_error: PipelineErrorKind,
    #[case] expect_err: bool,
) {
    let mut client = MockSequencerEngineClient::new();

    let unsafe_head = L2BlockInfo::default();
    client.expect_get_unsafe_head().times(1).return_once(move || Ok(unsafe_head));
    // Must not be called on critical error
    client.expect_start_build_block().times(0);
    let resets = if matches!(&forced_error, PipelineErrorKind::Reset(_)) { 2 } else { 1 };
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
    let result = actor.step().await;
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
    actor.admin_command_rx = admin_rx;

    let mut engine = MockSequencerEngineClient::new();
    engine.expect_reset_engine_forkchoice().times(1).return_once(|| Ok(()));
    // No block is built or sealed while the queue is full.
    engine.expect_start_build_block().times(0);
    engine.expect_get_unsafe_head().times(1).return_once(|| Ok(L2BlockInfo::default()));
    actor.engine_client = engine;

    let mut gossip = MockUnsafePayloadGossipClient::new();
    gossip.expect_has_capacity().return_const(false);
    gossip.expect_schedule_execution_payload_gossip().times(0);
    actor.unsafe_payload_gossip_client = gossip;

    // The build tick finds the queue full and returns instead of waiting for space.
    time::timeout(Duration::from_secs(10), actor.step()).await.unwrap().unwrap();

    let (tx, rx) = oneshot::channel();
    admin_tx.send(SequencerAdminCommand::StopSequencer(tx)).await.unwrap();
    time::timeout(Duration::from_secs(10), actor.step()).await.unwrap().unwrap();
    assert_eq!(rx.await.unwrap().unwrap(), L2BlockInfo::default().hash());
    assert!(!actor.state().active);
}

/// Block building resumes on the first tick after the gossip queue has room again.
#[tokio::test(start_paused = true)]
async fn building_resumes_once_the_gossip_queue_drains() {
    let mut actor = test_actor();

    let mut engine = MockSequencerEngineClient::new();
    engine.expect_reset_engine_forkchoice().times(1).return_once(|| Ok(()));
    engine.expect_get_unsafe_head().times(1).return_once(|| Ok(L2BlockInfo::default()));
    actor.engine_client = engine;

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
    time::timeout(Duration::from_secs(10), actor.step()).await.unwrap().unwrap();
}
