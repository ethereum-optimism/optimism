#[cfg(test)]
use crate::{
    NodeActor, SequencerActorError, SequencerAdminQuery,
    actors::{
        MockOriginSelector, MockSequencerEngineClient, MockUnsafePayloadGossipClient,
        sequencer::tests::test_util::test_actor,
    },
};
use alloy_rpc_types_engine::PayloadId;
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

/// A full gossip queue, for example during a signer outage, pauses block building without
/// blocking the actor, so admin queries such as op-conductor's `StopSequencer` are still answered.
#[tokio::test(start_paused = true)]
async fn full_gossip_queue_pauses_building_but_admin_queries_are_answered() {
    let mut actor = test_actor();
    let (admin_tx, admin_rx) = mpsc::channel(1);
    actor.admin_api_rx = admin_rx;

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
    admin_tx.send(SequencerAdminQuery::StopSequencer(tx)).await.unwrap();
    time::timeout(Duration::from_secs(10), actor.step()).await.unwrap().unwrap();
    assert_eq!(rx.await.unwrap().unwrap(), L2BlockInfo::default().hash());
    assert!(!actor.is_active);
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

/// After building a block, the next build tick waits until that block is due, rather than firing
/// straight away.
#[tokio::test(start_paused = true)]
async fn next_block_is_not_built_before_it_is_due() {
    let mut actor = test_actor();

    // The test's time starts at 2000-01-01T00:00:00Z, when the parent was built, so with a 2s block
    // time the next block is due in exactly 2s.
    let mut unsafe_head = L2BlockInfo::default();
    unsafe_head.block_info.timestamp = 946_684_800;

    let mut engine = MockSequencerEngineClient::new();
    engine.expect_reset_engine_forkchoice().times(1).return_once(|| Ok(()));
    engine.expect_get_unsafe_head().times(1).return_once(move || Ok(unsafe_head));
    engine.expect_start_build_block().times(1).return_once(|_| Ok(PayloadId::new([1; 8])));
    actor.engine_client = engine;

    let mut origin_selector = MockOriginSelector::new();
    origin_selector.expect_next_l1_origin().times(1).return_once(|_, _| Ok(BlockInfo::default()));
    actor.origin_selector = origin_selector;
    actor.attributes_builder = TestAttributesBuilder { attributes: vec![Ok(Default::default())] };

    // Room on the first tick, to start building; none on the second, so it returns before sealing.
    let ticks = Arc::new(AtomicUsize::new(0));
    let mut gossip = MockUnsafePayloadGossipClient::new();
    gossip.expect_has_capacity().returning(move || ticks.fetch_add(1, Ordering::SeqCst) == 0);
    actor.unsafe_payload_gossip_client = gossip;

    // Reset the engine and start building on the first tick, which fires immediately.
    time::timeout(Duration::from_secs(10), actor.step())
        .await
        .expect("the first tick did not fire")
        .expect("the first step failed");

    let step = actor.step();
    tokio::pin!(step);
    assert!(
        time::timeout(Duration::from_millis(500), &mut step).await.is_err(),
        "the next tick fired before the next block was due"
    );
    time::timeout(Duration::from_millis(1_500), &mut step)
        .await
        .expect("the next tick did not fire once the next block was due")
        .expect("the next step failed");
}
