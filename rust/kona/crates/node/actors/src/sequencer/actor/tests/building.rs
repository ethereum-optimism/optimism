use super::{
    super::{LastBuild, UnsealedPayloadHandle},
    QueueSigner, TestActor, test_actor, test_actor_with_config, with_signer,
};
use crate::{
    EngineClientError, MockOrigin, MockSequencerEngineClient,
    engine::{BuildTaskError, EngineBuildError},
    sequencer::{ActorError, MockSigner},
};
use alloy_rpc_types_engine::PayloadId;
use kona_derive::{BuilderError, PipelineErrorKind, test_utils::TestAttributesBuilder};
use kona_protocol::{BlockInfo, L2BlockInfo, OpAttributesWithParent};
use op_alloy_rpc_types_engine::OpExecutionPayloadEnvelope;
use rstest::rstest;
use std::{
    sync::Arc,
    time::{SystemTime, UNIX_EPOCH},
};
use tokio::{
    sync::{Notify, mpsc},
    time::{self, Duration, Instant},
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
    let mut origin = MockOrigin::new();
    origin.expect_next().times(1).return_once(move |_| Ok(l1_origin));

    let attributes_builder = TestAttributesBuilder { attributes: vec![Err(forced_error)] };

    let mut actor = test_actor();
    actor.origin = origin;
    actor.engine_client = client;
    actor.attributes_builder = attributes_builder;

    let result = actor.build(LastBuild::NothingStarted).await;
    if expect_err {
        assert!(result.is_err());
        assert!(matches!(
            result.unwrap_err(),
            ActorError::AttributesBuilder(PipelineErrorKind::Critical(_))
        ));
    } else {
        assert_eq!(result.unwrap(), LastBuild::NothingStarted);
    }
}

/// The engine reports build failures instead of retrying them. A refused build is reported, so that
/// the next is timed from the refusal, and only critical failures stop the sequencer.
#[rstest]
#[case::invalid(EngineBuildError::InvalidPayload(String::new()), false)]
#[case::syncing(EngineBuildError::EngineSyncing, false)]
#[case::critical(EngineBuildError::FinalizedAheadOfUnsafe(1, 2), true)]
#[tokio::test(start_paused = true)]
async fn build_handles_start_build_errors(#[case] error: EngineBuildError, #[case] fatal: bool) {
    let mut client = MockSequencerEngineClient::new();
    client.expect_get_unsafe_head().times(1).return_once(|| Ok(L2BlockInfo::default()));
    client.expect_start_build_block().times(1).return_once(move |_| {
        Err(EngineClientError::StartBuildError(BuildTaskError::EngineBuildError(error)))
    });
    let mut origin = MockOrigin::new();
    origin.expect_next().times(1).return_once(|_| Ok(BlockInfo::default()));

    let mut actor = test_actor();
    actor.engine_client = client;
    actor.origin = origin;
    actor.attributes_builder = TestAttributesBuilder { attributes: vec![Ok(Default::default())] };

    let result = actor.build(LastBuild::NothingStarted).await;
    if fatal {
        assert!(matches!(
            result,
            Err(ActorError::EngineError(EngineClientError::StartBuildError(_)))
        ));
        return;
    }
    assert_eq!(result.unwrap(), LastBuild::Refused(Instant::now()));
}

#[tokio::test]
async fn sealing_propagates_signer_errors() {
    let mut actor = test_actor();
    let block = alloy_consensus::Block::<op_alloy_consensus::OpTxEnvelope>::default();
    let payload = OpExecutionPayloadEnvelope::V1(
        alloy_rpc_types_engine::ExecutionPayloadV1::from_block_slow(&block),
    );
    actor
        .engine_client
        .expect_seal_and_canonicalize_block()
        .times(1)
        .return_once(|_, _| Ok(payload));
    actor
        .signer
        .expect_send()
        .times(1)
        .return_once(|_| Err(std::io::Error::other("signer channel closed")));
    let handle = UnsealedPayloadHandle {
        payload_id: Default::default(),
        attributes_with_parent: OpAttributesWithParent::new(
            Default::default(),
            Default::default(),
            None,
            false,
        ),
    };

    let error = actor.seal_and_commit_payload_if_applicable(&handle).await.unwrap_err();
    assert!(matches!(&error, ActorError::Signer(_)));
    assert_eq!(error.to_string(), "signer channel closed");
}

/// A full signer queue, for example during a signer outage, pauses block building without
/// blocking the actor, so admin queries such as op-conductor's `StopSequencer` are still answered.
#[tokio::test(start_paused = true)]
async fn full_signer_queue_pauses_building_but_admin_queries_are_answered() {
    let (mut actor, _, handle) = test_actor_with_config(true, None);
    actor.engine_client.expect_reset_engine_forkchoice().times(1).return_once(|| Ok(()));
    // No block is built while the queue is full.
    actor.engine_client.expect_start_build_block().times(0);
    actor
        .engine_client
        .expect_get_unsafe_head()
        .times(1)
        .return_once(|| Ok(L2BlockInfo::default()));
    let (signer, _queued) = mpsc::channel(1);
    signer.try_send(crate::test_payload(0)).unwrap();
    let task = tokio::spawn(with_signer(actor, QueueSigner(signer)).run());

    // Several block times pass without a build.
    time::sleep(Duration::from_secs(10)).await;
    let stopped = time::timeout(Duration::from_secs(1), handle.stop()).await.unwrap().unwrap();
    assert_eq!(stopped, L2BlockInfo::default().hash());
    assert!(!handle.is_active().unwrap());
    task.abort();
}

/// A build waits for the signer's queue to have room, and starts as soon as it does.
#[tokio::test(start_paused = true)]
async fn building_starts_as_soon_as_the_signer_queue_drains() {
    // The handle and command sender are dropped, so the actor gets no commands.
    let (mut actor, _, _) = test_actor_with_config(true, None);
    actor.engine_client.expect_reset_engine_forkchoice().times(1).return_once(|| Ok(()));
    actor
        .engine_client
        .expect_get_unsafe_head()
        .times(1)
        .return_once(|| Ok(L2BlockInfo::default()));
    let started = Arc::new(Notify::new());
    let notify = started.clone();
    actor.origin.expect_next().times(1).return_once(move |_| {
        notify.notify_one();
        Ok(BlockInfo::default())
    });
    actor.attributes_builder = TestAttributesBuilder {
        attributes: vec![Err(PipelineErrorKind::Temporary(
            BuilderError::Custom(String::new()).into(),
        ))],
    };
    let (signer, mut queued) = mpsc::channel(1);
    signer.try_send(crate::test_payload(0)).unwrap();
    let task = tokio::spawn(with_signer(actor, QueueSigner(signer)).run());

    // However long the queue stays full, no block is built.
    assert!(time::timeout(Duration::from_secs(10), started.notified()).await.is_err());
    // Once the signer takes the queued payload, building starts at once, not at the next tick.
    queued.recv().await.unwrap();
    time::timeout(Duration::from_millis(1), started.notified()).await.unwrap();
    task.abort();
}

/// A ready signer, for timing tests.
fn ready_signer() -> MockSigner {
    let mut signer = MockSigner::new();
    signer.expect_ready().returning(|| Ok(()));
    signer
}

const BLOCK_TIME: Duration = Duration::from_secs(2);

/// Whether `next_build` returns within `within`.
async fn next_build_within(active: bool, last: &LastBuild, within: Duration) -> bool {
    let signer = ready_signer();
    time::timeout(within, TestActor::next_build(active, &signer, last, BLOCK_TIME)).await.is_ok()
}

#[tokio::test(start_paused = true)]
async fn no_build_while_stopped() {
    assert!(!next_build_within(false, &LastBuild::NothingStarted, Duration::from_secs(60)).await);
}

#[tokio::test(start_paused = true)]
async fn builds_at_once_when_nothing_waits_to_be_sealed() {
    assert!(next_build_within(true, &LastBuild::NothingStarted, Duration::ZERO).await);
}

#[tokio::test(start_paused = true)]
async fn retries_a_block_time_after_a_refusal() {
    let refused = LastBuild::Refused(Instant::now());
    assert!(!next_build_within(true, &refused, BLOCK_TIME * 3 / 4).await);
    assert!(next_build_within(true, &refused, BLOCK_TIME / 4 + Duration::from_millis(1)).await);
}

/// A started block whose parent has the given timestamp.
fn started(parent_timestamp: u64, seal_duration: Duration) -> LastBuild {
    let mut parent = L2BlockInfo::default();
    parent.block_info.timestamp = parent_timestamp;
    let block = UnsealedPayloadHandle {
        payload_id: PayloadId::new([1; 8]),
        attributes_with_parent: OpAttributesWithParent::new(
            Default::default(),
            parent,
            None,
            false,
        ),
    };
    LastBuild::Started { block: Box::new(block), seal_duration }
}

fn unix_now() -> u64 {
    SystemTime::now().duration_since(UNIX_EPOCH).unwrap().as_secs()
}

#[tokio::test(start_paused = true)]
async fn seals_a_started_block_when_it_is_due_as_early_as_the_last_seal_took() {
    // Due in 9 to 10 seconds: a block time after its parent, less the second the seal took, in
    // whole seconds.
    let soon = started(unix_now() + 9, Duration::from_secs(1));
    assert!(!next_build_within(true, &soon, Duration::from_secs(8)).await);
    assert!(next_build_within(true, &soon, Duration::from_secs(10)).await);
    // A block whose time has passed is sealed at once.
    let late = started(unix_now() - 3, Duration::ZERO);
    assert!(next_build_within(true, &late, Duration::ZERO).await);
}

/// The build waits for both the signer to have room and the build to be due, however they fall.
#[tokio::test(start_paused = true)]
async fn waits_for_room_even_after_the_build_is_due() {
    type QueueActor = super::super::Actor<
        TestAttributesBuilder,
        crate::MockConductor,
        MockOrigin,
        MockSequencerEngineClient,
        QueueSigner,
    >;
    let (signer, mut queued) = mpsc::channel(1);
    signer.try_send(crate::test_payload(0)).unwrap();
    let signer = QueueSigner(signer);
    let last = LastBuild::Refused(Instant::now());
    let next = QueueActor::next_build(true, &signer, &last, BLOCK_TIME);
    tokio::pin!(next);

    // The build is due after a block time, but the signer is still full.
    assert!(time::timeout(BLOCK_TIME * 2, &mut next).await.is_err());
    queued.recv().await.unwrap();
    time::timeout(Duration::from_millis(1), &mut next).await.unwrap().unwrap();
}

/// A build seals the block the last one started, then starts the next.
#[tokio::test(start_paused = true)]
async fn build_seals_the_block_the_last_build_started() {
    let mut actor = test_actor();
    let sealed = crate::test_payload(1);
    let payload = sealed.clone();
    actor
        .engine_client
        .expect_seal_and_canonicalize_block()
        .withf(|payload_id, _| *payload_id == PayloadId::new([1; 8]))
        .times(1)
        .return_once(move |_, _| Ok(payload));
    actor
        .signer
        .expect_send()
        .withf(move |payload| *payload == sealed)
        .times(1)
        .returning(|_| Ok(()));
    // Starting the next block stops at a temporary error.
    actor
        .engine_client
        .expect_get_unsafe_head()
        .times(1)
        .return_once(|| Ok(L2BlockInfo::default()));
    actor.origin.expect_next().times(1).return_once(|_| Ok(BlockInfo::default()));
    actor.attributes_builder = TestAttributesBuilder {
        attributes: vec![Err(PipelineErrorKind::Temporary(
            BuilderError::Custom(String::new()).into(),
        ))],
    };

    let last = actor.build(started(unix_now(), Duration::ZERO)).await.unwrap();
    assert_eq!(last, LastBuild::NothingStarted);
}
