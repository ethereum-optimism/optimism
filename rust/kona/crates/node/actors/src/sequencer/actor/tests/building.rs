use super::{
    super::{LastBuild, Publication, SealedBlock, UnsealedPayloadHandle},
    QueueSigner, TestActor, test_actor, test_actor_with_config, with_signer,
};
use crate::{
    ConductorError, EngineClientError, MockConductor, MockOrigin, MockSequencerEngineClient,
    engine::{BuildTaskError, EngineBuildError, InsertTaskError, SealTaskError},
    origin::OriginError,
    sequencer::{ActorError, MockSigner, handle::Message},
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
    sync::{Notify, mpsc, oneshot},
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

/// L1 reorging out the unsafe head's L1 origin leaves nothing to build on until the engine,
/// which follows L1, resets.
#[tokio::test(start_paused = true)]
async fn build_waits_out_a_reorged_origin() {
    let mut client = MockSequencerEngineClient::new();
    client.expect_get_unsafe_head().times(1).return_once(|| Ok(L2BlockInfo::default()));
    client.expect_start_build_block().times(0);
    let mut origin = MockOrigin::new();
    origin.expect_next().times(1).return_once(|parent| Err(OriginError::Reorged(parent.l1_origin)));
    let mut actor = test_actor();
    actor.origin = origin;
    actor.engine_client = client;

    assert_eq!(actor.build(LastBuild::NothingStarted).await.unwrap(), LastBuild::NothingStarted);
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
async fn publishing_propagates_signer_errors() {
    let mut actor = test_actor();
    actor.engine_client.expect_canonicalize_block().times(1).return_once(|_, _| Ok(()));
    actor
        .signer
        .expect_send()
        .times(1)
        .return_once(|_| Err(std::io::Error::other("signer channel closed")));

    let error = actor.publish(sealed_block(crate::test_payload(0))).await.unwrap_err();
    assert!(matches!(&error, ActorError::Signer(_)));
    assert_eq!(error.to_string(), "signer channel closed");
}

/// A sealed block, built on the default unsafe head, that the conductor has not committed.
fn sealed_block(payload: OpExecutionPayloadEnvelope) -> SealedBlock {
    SealedBlock {
        payload,
        attributes: OpAttributesWithParent::new(
            Default::default(),
            Default::default(),
            None,
            false,
        ),
        committed: false,
    }
}

/// A conductor that rejects every commit.
fn rejecting_conductor() -> MockConductor {
    let mut conductor = MockConductor::new();
    conductor.expect_commit_unsafe_payload().returning(|_| {
        Err(ConductorError::Rpc(alloy_transport::TransportErrorKind::custom_str("not leader")))
    });
    conductor
}

/// A block the conductor does not commit is neither made canonical nor gossiped, and is kept to
/// commit again a second later.
#[tokio::test(start_paused = true)]
async fn a_block_the_conductor_rejects_is_kept_unpublished() {
    let (mut actor, _, _) = test_actor_with_config(true, Some(rejecting_conductor()));
    actor.engine_client.expect_get_unsafe_head().returning(|| Ok(L2BlockInfo::default()));
    actor.engine_client.expect_canonicalize_block().times(0);
    actor.signer.expect_send().times(0);
    let payload = crate::test_payload(1);
    let sealed = payload.clone();
    actor.engine_client.expect_seal_block().times(1).return_once(move |_, _| Ok(sealed));

    let last = actor.build(started(unix_now(), Duration::ZERO)).await.unwrap();
    let LastBuild::Sealed { block, at } = &last else { panic!("kept nothing: {last:?}") };
    assert_eq!((&block.payload, block.committed, *at), (&payload, false, Instant::now()));
    // The next attempt is due a second later.
    assert!(!next_build_within(true, &last, Duration::from_millis(900)).await);
    assert!(next_build_within(true, &last, Duration::from_secs(1)).await);
}

/// A kept block is committed, made canonical and gossiped once the conductor takes it, and only
/// then does the next block start.
#[tokio::test(start_paused = true)]
async fn a_kept_block_is_published_once_the_conductor_commits_it() {
    let mut conductor = MockConductor::new();
    conductor.expect_commit_unsafe_payload().times(1).returning(|_| Ok(()));
    let (mut actor, _, _) = test_actor_with_config(true, Some(conductor));
    let payload = crate::test_payload(1);
    let mut sequence = mockall::Sequence::new();
    actor
        .engine_client
        .expect_get_unsafe_head()
        .times(1)
        .in_sequence(&mut sequence)
        .returning(|| Ok(L2BlockInfo::default()));
    let canonical = payload.clone();
    actor
        .engine_client
        .expect_canonicalize_block()
        .withf(move |payload, _| *payload == canonical)
        .times(1)
        .in_sequence(&mut sequence)
        .return_once(|_, _| Ok(()));
    let gossiped = payload.clone();
    actor
        .signer
        .expect_send()
        .withf(move |payload| *payload == gossiped)
        .times(1)
        .in_sequence(&mut sequence)
        .returning(|_| Ok(()));
    // Starting the next block stops at a temporary error.
    actor
        .engine_client
        .expect_get_unsafe_head()
        .times(1)
        .in_sequence(&mut sequence)
        .returning(|| Ok(L2BlockInfo::default()));
    actor.origin.expect_next().times(1).return_once(|_| Ok(BlockInfo::default()));
    actor.attributes_builder = TestAttributesBuilder {
        attributes: vec![Err(PipelineErrorKind::Temporary(
            BuilderError::Custom(String::new()).into(),
        ))],
    };

    let kept = LastBuild::Sealed { block: Box::new(sealed_block(payload)), at: Instant::now() };
    assert_eq!(actor.build(kept).await.unwrap(), LastBuild::NothingStarted);
}

/// A kept block the unsafe head has moved past is dropped without being committed.
#[tokio::test(start_paused = true)]
async fn a_kept_block_the_unsafe_head_moved_past_is_dropped() {
    let mut conductor = MockConductor::new();
    conductor.expect_commit_unsafe_payload().times(0);
    let (mut actor, _, _) = test_actor_with_config(true, Some(conductor));
    let mut moved = L2BlockInfo::default();
    moved.block_info.number = 1;
    actor.engine_client.expect_get_unsafe_head().times(1).return_once(move || Ok(moved));
    actor.engine_client.expect_canonicalize_block().times(0);

    let publication = actor.publish(sealed_block(crate::test_payload(1))).await.unwrap();
    assert!(matches!(publication, Publication::Dropped));
}

/// Each attempt to publish a sealed block is counted by what became of it, so an operator sees a
/// conductor that refuses the sequencer's blocks.
#[tokio::test(start_paused = true)]
async fn publications_are_counted_by_outcome() {
    let recorder = metrics_exporter_prometheus::PrometheusBuilder::new().build_recorder();
    let metrics = recorder.handle();
    let _recorder = metrics::set_default_local_recorder(&recorder);
    let mut conductor = MockConductor::new();
    let mut commits = 0;
    conductor.expect_commit_unsafe_payload().times(2).returning(move |_| {
        commits += 1;
        if commits == 1 {
            Err(ConductorError::Rpc(alloy_transport::TransportErrorKind::custom_str("not leader")))
        } else {
            Ok(())
        }
    });
    let (mut actor, _, _) = test_actor_with_config(true, Some(conductor));
    actor.engine_client.expect_get_unsafe_head().returning(|| Ok(L2BlockInfo::default()));
    actor.engine_client.expect_canonicalize_block().times(1).return_once(|_, _| Ok(()));
    actor.signer.expect_send().times(1).returning(|_| Ok(()));

    let publication = actor.publish(sealed_block(crate::test_payload(1))).await.unwrap();
    let Publication::Retry(block) = publication else { panic!("kept nothing: {publication:?}") };
    assert!(matches!(actor.publish(*block).await.unwrap(), Publication::Published));
    let rendered = metrics.render();
    for outcome in ["conductor_refused", "published"] {
        let line = format!("kona_node_sequencer_sealed_blocks_total{{outcome=\"{outcome}\"}} 1");
        assert!(rendered.contains(&line), "{rendered}");
    }
}

/// A committed block whose parent the unsafe head moved past by the time it is made canonical is
/// dropped as stale, not as one the engine rejects.
#[tokio::test(start_paused = true)]
async fn a_committed_block_the_unsafe_head_moved_past_is_dropped_as_stale() {
    let recorder = metrics_exporter_prometheus::PrometheusBuilder::new().build_recorder();
    let metrics = recorder.handle();
    let _recorder = metrics::set_default_local_recorder(&recorder);
    let (mut actor, _, _) = test_actor_with_config(true, None);
    actor.engine_client.expect_canonicalize_block().times(1).return_once(|_, _| {
        Err(EngineClientError::SealError(SealTaskError::UnsafeHeadChangedSinceBuild))
    });
    actor.signer.expect_send().times(0);
    let mut block = sealed_block(crate::test_payload(1));
    block.committed = true;

    assert!(matches!(actor.publish(block).await.unwrap(), Publication::Dropped));
    let rendered = metrics.render();
    assert!(
        rendered.contains("kona_node_sequencer_sealed_blocks_total{outcome=\"dropped_stale\"} 1"),
        "{rendered}"
    );
}

/// Once an operator overrides the conductor's leadership, for disaster recovery, the sequencer
/// publishes blocks without committing them to its conductor, as op-node.
#[tokio::test(start_paused = true)]
async fn an_overridden_leader_publishes_without_its_conductor() {
    let mut conductor = MockConductor::new();
    conductor.expect_commit_unsafe_payload().times(0);
    let (mut actor, _, _) = test_actor_with_config(true, Some(conductor));
    actor.engine_client.expect_canonicalize_block().times(1).return_once(|_, _| Ok(()));
    actor.signer.expect_send().times(1).returning(|_| Ok(()));

    let (tx, rx) = oneshot::channel();
    actor.handle_message(Message::OverrideLeader(tx)).await;
    rx.await.unwrap().unwrap();
    let publication = actor.publish(sealed_block(crate::test_payload(1))).await.unwrap();
    assert!(matches!(publication, Publication::Published));
}

/// A committed block the engine fails to make canonical is tried again as it is, without being
/// committed again. One the engine can never make canonical is dropped.
#[rstest]
#[case::temporary(
    InsertTaskError::InsertFailed(alloy_transport::TransportErrorKind::custom_str("down")),
    true
)]
#[case::rejected(
    InsertTaskError::UnexpectedPayloadStatus(alloy_rpc_types_engine::PayloadStatusEnum::Invalid {
        validation_error: String::new()
    }),
    false
)]
#[tokio::test]
async fn a_committed_block_that_fails_to_become_canonical(
    #[case] error: InsertTaskError,
    #[case] retried: bool,
) {
    let mut conductor = MockConductor::new();
    conductor.expect_commit_unsafe_payload().times(1).returning(|_| Ok(()));
    let (mut actor, _, _) = test_actor_with_config(true, Some(conductor));
    actor.engine_client.expect_get_unsafe_head().returning(|| Ok(L2BlockInfo::default()));
    actor.engine_client.expect_canonicalize_block().times(1).return_once(move |_, _| {
        Err(EngineClientError::SealError(SealTaskError::PayloadInsertionFailed(Box::new(error))))
    });
    actor.signer.expect_send().times(0);

    match actor.publish(sealed_block(crate::test_payload(1))).await.unwrap() {
        Publication::Retry(block) => assert!(retried && block.committed),
        Publication::Dropped => assert!(!retried),
        Publication::Published => panic!("published a block that is not canonical"),
    }
}

/// A full signer queue, for example during a signer outage, pauses block building without
/// blocking the actor, so admin queries such as op-conductor's `StopSequencer` are still answered.
#[tokio::test(start_paused = true)]
async fn full_signer_queue_pauses_building_but_admin_queries_are_answered() {
    let (mut actor, _, handle) = test_actor_with_config(true, None);
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
        .expect_seal_block()
        .withf(|payload_id, _| *payload_id == PayloadId::new([1; 8]))
        .times(1)
        .return_once(move |_, _| Ok(payload));
    actor.engine_client.expect_canonicalize_block().times(1).return_once(|_, _| Ok(()));
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
