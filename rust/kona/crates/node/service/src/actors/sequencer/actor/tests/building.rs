use super::{SlowSealEngine, TEST_START, TestActor, test_actor, test_actor_with_engine};
use crate::{
    NodeActor, SequencerActorError, SequencerEngineClient,
    actors::{MockOriginSelector, MockSequencerEngineClient, MockUnsafePayloadGossipClient},
};
use alloy_rpc_types_engine::{ExecutionPayloadV1, PayloadId};
use kona_derive::{BuilderError, PipelineErrorKind, test_utils::TestAttributesBuilder};
use kona_protocol::{BlockInfo, L2BlockInfo};
use kona_rpc::SequencerAdminCommand;
use op_alloy_rpc_types_engine::OpExecutionPayloadEnvelope;
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

/// An unsafe head built at `timestamp`.
fn block_at(timestamp: u64) -> L2BlockInfo {
    let mut block = L2BlockInfo::default();
    block.block_info.timestamp = timestamp;
    block
}

/// A sealed payload, as returned by the engine.
fn payload() -> OpExecutionPayloadEnvelope {
    let block = alloy_consensus::Block::<op_alloy_consensus::OpTxEnvelope>::default();
    OpExecutionPayloadEnvelope::V1(ExecutionPayloadV1::from_block_slow(&block))
}

/// Returns an engine that is reset once, then builds one block on top of each of `heads`, in order.
fn mock_engine(heads: Vec<L2BlockInfo>) -> MockSequencerEngineClient {
    let builds = heads.len();
    let mut heads = heads.into_iter();
    let mut engine = MockSequencerEngineClient::new();
    engine.expect_reset_engine_forkchoice().times(1).return_once(|| Ok(()));
    engine
        .expect_get_unsafe_head()
        .times(builds)
        .returning(move || Ok(heads.next().expect("no more unsafe heads")));
    engine.expect_start_build_block().times(builds).returning(|_| Ok(PayloadId::new([1; 8])));
    engine
}

/// Sets up `actor` to build `blocks` blocks. The gossip queue has room on the first
/// `ticks_with_room` ticks only, so a later tick returns before sealing.
fn expect_builds<E: SequencerEngineClient>(
    actor: &mut TestActor<E>,
    blocks: usize,
    ticks_with_room: usize,
) {
    let mut origin_selector = MockOriginSelector::new();
    origin_selector
        .expect_next_l1_origin()
        .times(blocks)
        .returning(|_, _| Ok(BlockInfo::default()));
    actor.origin_selector = origin_selector;
    actor.attributes_builder =
        TestAttributesBuilder { attributes: (0..blocks).map(|_| Ok(Default::default())).collect() };

    let ticks = Arc::new(AtomicUsize::new(0));
    let mut gossip = MockUnsafePayloadGossipClient::new();
    gossip
        .expect_has_capacity()
        .returning(move || ticks.fetch_add(1, Ordering::SeqCst) < ticks_with_room);
    gossip.expect_schedule_execution_payload_gossip().returning(|_| Ok(()));
    actor.unsafe_payload_gossip_client = gossip;
}

/// Steps `actor` once and returns how much time passed on tokio's clock.
async fn step_time<E: SequencerEngineClient + 'static>(actor: &mut TestActor<E>) -> Duration {
    let start = time::Instant::now();
    time::timeout(Duration::from_secs(10), actor.step())
        .await
        .expect("the tick did not fire")
        .expect("the step failed");
    start.elapsed()
}

/// After building a block, the next build tick fires when the next block is due by the actor's
/// clock, which starts at `TEST_START`: neither earlier, nor after one block time (2s) if the block
/// is due at another time.
///
/// With `T` as `TEST_START`, `t` as the time since the test started, and `P` as the parent's
/// timestamp:
///
/// | t       | Tick | What happens                                   | Block due at |
/// |---------|------|------------------------------------------------|--------------|
/// | 0       | 1    | Reset; start building A on a parent built at P | A: P+2       |
/// | P+2 - T | 2    | A is due; nothing has been sealed yet, so no   |              |
/// |         |      | time is taken off for sealing                  |              |
///
/// When A is already overdue (`P+2 < T`), tick 2 fires straight away.
#[rstest]
#[case::due_later(TEST_START + 1, Duration::from_secs(3))]
#[case::overdue(TEST_START - 3, Duration::ZERO)]
#[tokio::test(start_paused = true)]
async fn next_block_is_built_when_it_is_due(#[case] parent_timestamp: u64, #[case] wait: Duration) {
    let mut actor = test_actor();
    actor.engine_client = mock_engine(vec![block_at(parent_timestamp)]);
    expect_builds(&mut actor, 1, 1);

    // Reset the engine and start building on the first tick, which fires immediately.
    step_time(&mut actor).await;

    assert_eq!(step_time(&mut actor).await, wait, "the next tick fired at the wrong time");
}

/// Each tick seals the block built since the previous tick, then starts building the next one. The
/// next tick is scheduled for when the next block is due, less the time the last seal took, so the
/// next seal finishes when its block is due. A seal that takes longer than the time left means the
/// next tick fires straight away.
///
/// With `T` as `TEST_START`, `t` as the time since the test started, and `s` as the seal time:
///
/// | t       | Tick | What happens                                   | Block due at |
/// |---------|------|------------------------------------------------|--------------|
/// | 0       | 1    | Reset; start building A on a parent built at T | A: T+2       |
/// | 2       | 2    | A is due; seal A, which takes `s`              |              |
/// | 2 + s   | 2    | Start building B on A                          | B: T+4       |
/// | 4 - s   | 3    | Seal B, so that it finishes at t=4             |              |
///
/// The test checks the wait between the end of tick 2 and tick 3: `2 - 2s`, or none if that is
/// negative. Tick 1 had nothing to seal, so tick 2 takes no time off for sealing.
#[rstest]
#[case::wait_less(Duration::from_millis(500), Duration::from_secs(1))]
#[case::no_time_left(Duration::from_millis(1_500), Duration::ZERO)]
#[case::longer_than_block_time(Duration::from_millis(2_500), Duration::ZERO)]
#[tokio::test(start_paused = true)]
async fn next_block_starts_earlier_by_the_seal_time(
    #[case] seal_time: Duration,
    #[case] wait: Duration,
) {
    // The first block builds on a parent at `TEST_START`, and is sealed into the parent of the
    // second block, due at `TEST_START + 2`.
    let mut engine = mock_engine(vec![block_at(TEST_START), block_at(TEST_START + 2)]);
    engine.expect_seal_block().times(1).return_once(|_, _| Ok(payload()));
    engine.expect_canonicalize_block().times(1).returning(|_, _| Ok(()));
    let mut actor = test_actor_with_engine(SlowSealEngine { inner: engine, seal_time });
    expect_builds(&mut actor, 2, 2);

    // Reset the engine and start building the first block, immediately.
    step_time(&mut actor).await;
    // When the first block is due, seal it and start building the second.
    assert_eq!(step_time(&mut actor).await, Duration::from_secs(2) + seal_time);

    // The second block is due 2s after the first, less the time the seal took.
    assert_eq!(step_time(&mut actor).await, wait, "the next tick fired at the wrong time");
}

#[rstest]
#[case::stale_build(crate::EngineClientError::StartBuildError(
    kona_engine::BuildTaskError::StaleBuild
))]
#[case::engine_recovery(crate::EngineClientError::ResponseError("reset cleared the build".into()))]
#[case::invalid_forkchoice(crate::EngineClientError::StartBuildError(kona_engine::BuildTaskError::EngineBuildError(
    kona_engine::EngineBuildError::AttributesInsertionFailed(Arc::new(alloy_transport::RpcError::ErrorResp(
        alloy_json_rpc::ErrorPayload {
            code: i64::from(alloy_rpc_types_engine::INVALID_FORK_CHOICE_STATE_ERROR),
            message: "inconsistent forkchoice".into(), data: None,
        }
    )))
)))]
#[tokio::test(start_paused = true)]
async fn rejected_build_retries_on_new_head_and_keeps_admin_responsive(
    #[case] error: crate::EngineClientError,
) {
    let (mut actor, commands) = super::test_actor_with_config(true, false, None);
    actor.started = true;
    let mut old_head = block_at(TEST_START);
    old_head.block_info.hash = alloy_primitives::B256::repeat_byte(1);
    let mut new_head = old_head;
    new_head.block_info.hash = alloy_primitives::B256::repeat_byte(2);
    let mut engine = MockSequencerEngineClient::new();
    let mut sequence = mockall::Sequence::new();
    engine
        .expect_get_unsafe_head()
        .times(1)
        .in_sequence(&mut sequence)
        .return_once(move || Ok(old_head));
    engine
        .expect_start_build_block()
        .withf(move |attrs| attrs.parent == old_head)
        .times(1)
        .in_sequence(&mut sequence)
        .return_once(move |_| Err(error));
    engine
        .expect_get_unsafe_head()
        .times(1)
        .in_sequence(&mut sequence)
        .return_once(move || Ok(new_head));
    engine
        .expect_start_build_block()
        .withf(move |attrs| attrs.parent == new_head)
        .times(1)
        .in_sequence(&mut sequence)
        .return_once(|_| Ok(PayloadId::new([1; 8])));
    // StopSequencer returns the current head without advancing the pending build.
    engine
        .expect_get_unsafe_head()
        .times(1)
        .in_sequence(&mut sequence)
        .return_once(move || Ok(new_head));
    actor.engine_client = engine;
    expect_builds(&mut actor, 2, 2);

    assert_eq!(step_time(&mut actor).await, Duration::ZERO);
    assert!(actor.in_flight.is_none());
    assert!(actor.state().active);
    assert_eq!(step_time(&mut actor).await, Duration::from_secs(1));
    assert!(matches!(&actor.in_flight, Some(super::super::InFlightBlock::Building(handle))
        if handle.attributes_with_parent.parent == new_head));

    let (tx, rx) = oneshot::channel();
    commands.send(SequencerAdminCommand::StopSequencer(tx)).await.unwrap();
    assert_eq!(step_time(&mut actor).await, Duration::ZERO);
    assert_eq!(rx.await.unwrap().unwrap(), new_head.hash());
    assert!(!actor.state().active);
}

#[rstest]
#[case::dead_engine(crate::EngineClientError::RequestError("engine stopped".into()))]
#[case::critical_build(crate::EngineClientError::StartBuildError(
    kona_engine::BuildTaskError::EngineBuildError(
        kona_engine::EngineBuildError::FinalizedAheadOfUnsafe(1, 2)
    )
))]
#[tokio::test(start_paused = true)]
async fn terminal_build_failures_still_stop_sequencing(#[case] error: crate::EngineClientError) {
    let mut actor = test_actor();
    actor.started = true;
    let mut engine = MockSequencerEngineClient::new();
    engine.expect_get_unsafe_head().times(1).return_once(|| Ok(block_at(TEST_START)));
    engine.expect_start_build_block().times(1).return_once(move |_| Err(error));
    actor.engine_client = engine;
    expect_builds(&mut actor, 1, 1);
    assert!(actor.step().await.is_err());
}
