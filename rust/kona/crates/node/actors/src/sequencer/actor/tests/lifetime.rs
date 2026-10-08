use crate::{
    Conductor, ConductorError, EngineClientError, MockConductor, MockOriginSelector,
    MockSequencerEngineClient, MockUnsafePayloadGossipClient, QueuedSequencerEngineClient,
    sequencer::{ActorError, Builder, Capacity, HandleError, State},
};
use async_trait::async_trait;
use kona_derive::test_utils::TestAttributesBuilder;
use kona_engine::EngineActorRequest;
use kona_genesis::RollupConfig;
use kona_protocol::L2BlockInfo;
use op_alloy_rpc_types_engine::OpExecutionPayloadEnvelope;
use std::{
    future::{pending, ready},
    sync::{
        Arc,
        atomic::{AtomicUsize, Ordering},
    },
};
use tokio::{
    sync::{Notify, mpsc, watch},
    time::{self, Duration},
};
use tokio_util::sync::CancellationToken;

fn config() -> Arc<RollupConfig> {
    Arc::new(RollupConfig { block_time: 2, ..Default::default() })
}

#[derive(Debug)]
struct PendingConductor(Arc<Notify>);

#[async_trait]
impl Conductor for PendingConductor {
    async fn commit_unsafe_payload(
        &self,
        _payload: &OpExecutionPayloadEnvelope,
    ) -> Result<(), ConductorError> {
        panic!("a stopped sequencer must not commit payloads")
    }

    async fn override_leader(&self) -> Result<(), ConductorError> {
        self.0.notify_one();
        pending().await
    }
}

#[test]
fn construction_and_build_do_not_require_a_runtime() {
    let builder =
        Builder::new(Capacity::try_from(1).unwrap(), Some(MockConductor::new()), false, true);
    let handle = builder.handle();
    let initial = State { active: false, conductor_enabled: true, recovery_mode: true };
    assert_eq!(handle.snapshot().unwrap(), initial);
    let task = builder.build(
        TestAttributesBuilder { attributes: vec![] },
        MockSequencerEngineClient::new(),
        MockOriginSelector::new(),
        config(),
        MockUnsafePayloadGossipClient::new(),
        CancellationToken::new(),
    );
    assert_eq!(handle.snapshot().unwrap(), initial);
    drop(task);
    assert!(matches!(handle.snapshot(), Err(HandleError::RequestError(_))));
}

#[tokio::test]
async fn cancellation_before_startup_skips_the_engine_reset() {
    let cancellation = CancellationToken::new();
    cancellation.cancel();
    let task = Builder::new(Capacity::try_from(1).unwrap(), None::<MockConductor>, true, false)
        .build(
            TestAttributesBuilder { attributes: vec![] },
            MockSequencerEngineClient::new(),
            MockOriginSelector::new(),
            config(),
            MockUnsafePayloadGossipClient::new(),
            cancellation,
        );
    task.await.unwrap();
}

#[tokio::test]
async fn cancellation_interrupts_a_pending_initial_reset() {
    let builder = Builder::new(Capacity::try_from(1).unwrap(), None::<MockConductor>, true, false);
    let handle = builder.handle();
    let (engine_actor_request_tx, mut requests) = mpsc::channel(1);
    let (_, unsafe_head_rx) = watch::channel(L2BlockInfo::default());
    let cancellation = CancellationToken::new();
    let task = tokio::spawn(builder.build(
        TestAttributesBuilder { attributes: vec![] },
        QueuedSequencerEngineClient { engine_actor_request_tx, unsafe_head_rx },
        MockOriginSelector::new(),
        config(),
        MockUnsafePayloadGossipClient::new(),
        cancellation.clone(),
    ));
    let EngineActorRequest::Reset(request) = requests.recv().await.unwrap() else {
        panic!("expected initial reset");
    };
    cancellation.cancel();
    task.await.unwrap().unwrap();
    assert!(request.result_tx.is_closed());
    assert!(matches!(handle.snapshot(), Err(HandleError::RequestError(_))));
}

#[tokio::test(start_paused = true)]
async fn sequencing_continues_without_command_handles() {
    let mut engine = MockSequencerEngineClient::new();
    engine.expect_reset_engine_forkchoice().times(1).return_once(|| Ok(()));
    let ticks = Arc::new(AtomicUsize::new(0));
    let observed_ticks = ticks.clone();
    let cancellation = CancellationToken::new();
    let cancel_after_ticks = cancellation.clone();
    let mut gossip = MockUnsafePayloadGossipClient::new();
    gossip.expect_has_capacity().times(3).returning(move || {
        if observed_ticks.fetch_add(1, Ordering::Relaxed) == 2 {
            cancel_after_ticks.cancel();
        }
        false
    });
    let task = Builder::new(Capacity::try_from(1).unwrap(), None::<MockConductor>, true, false)
        .build(
            TestAttributesBuilder { attributes: vec![] },
            engine,
            MockOriginSelector::new(),
            config(),
            gossip,
            cancellation,
        );
    time::timeout(Duration::from_secs(5), task).await.unwrap().unwrap();
    assert_eq!(ticks.load(Ordering::Relaxed), 3);
}

#[tokio::test]
async fn stopped_actor_without_command_handles_waits_for_cancellation() {
    let mut engine = MockSequencerEngineClient::new();
    engine.expect_reset_engine_forkchoice().times(1).return_once(|| Ok(()));
    let cancellation = CancellationToken::new();
    let task = Builder::new(Capacity::try_from(1).unwrap(), None::<MockConductor>, false, false)
        .build(
            TestAttributesBuilder { attributes: vec![] },
            engine,
            MockOriginSelector::new(),
            config(),
            MockUnsafePayloadGossipClient::new(),
            cancellation.clone(),
        );
    tokio::pin!(task);
    tokio::select! {
        biased;
        result = &mut task => panic!("stopped actor exited: {result:?}"),
        _ = ready(()) => {}
    }
    cancellation.cancel();
    task.await.unwrap();
}

#[tokio::test]
async fn failed_startup_rejects_a_previously_queued_command() {
    let builder = Builder::new(Capacity::try_from(1).unwrap(), None::<MockConductor>, false, false);
    let handle = builder.handle();
    let command = handle.start();
    tokio::pin!(command);
    tokio::select! {
        biased;
        result = &mut command => panic!("command completed before build: {result:?}"),
        _ = ready(()) => {}
    }
    let mut engine = MockSequencerEngineClient::new();
    engine
        .expect_reset_engine_forkchoice()
        .times(1)
        .return_once(|| Err(EngineClientError::RequestError("startup reset failed".to_owned())));
    let task = builder.build(
        TestAttributesBuilder { attributes: vec![] },
        engine,
        MockOriginSelector::new(),
        config(),
        MockUnsafePayloadGossipClient::new(),
        CancellationToken::new(),
    );
    assert!(matches!(task.await, Err(ActorError::EngineError(_))));
    assert!(matches!(command.await, Err(HandleError::RequestError(_))));
    assert!(matches!(handle.snapshot(), Err(HandleError::RequestError(_))));
}

#[tokio::test]
async fn cancellation_interrupts_an_in_flight_block_build() {
    let (engine_actor_request_tx, mut requests) = mpsc::channel(1);
    let (_, unsafe_head_rx) = watch::channel(L2BlockInfo::default());
    let mut origin = MockOriginSelector::new();
    origin.expect_next_l1_origin().times(1).return_once(|_, _| Ok(Default::default()));
    let mut gossip = MockUnsafePayloadGossipClient::new();
    gossip.expect_has_capacity().times(1).return_const(true);
    let cancellation = CancellationToken::new();
    let task = tokio::spawn(
        Builder::new(Capacity::try_from(1).unwrap(), None::<MockConductor>, true, false).build(
            TestAttributesBuilder { attributes: vec![Ok(Default::default())] },
            QueuedSequencerEngineClient { engine_actor_request_tx, unsafe_head_rx },
            origin,
            config(),
            gossip,
            cancellation.clone(),
        ),
    );
    let EngineActorRequest::Reset(reset) = requests.recv().await.unwrap() else {
        panic!("expected initial reset");
    };
    reset.result_tx.send(Ok(())).await.unwrap();
    let EngineActorRequest::Build(build) = requests.recv().await.unwrap() else {
        panic!("expected block build");
    };
    cancellation.cancel();
    task.await.unwrap().unwrap();
    assert!(build.result_tx.is_closed());
}

#[tokio::test]
async fn cancellation_interrupts_an_in_flight_message() {
    let started = Arc::new(Notify::new());
    let builder = Builder::new(
        Capacity::try_from(1).unwrap(),
        Some(PendingConductor(started.clone())),
        false,
        false,
    );
    let handle = builder.handle();
    let mut engine = MockSequencerEngineClient::new();
    engine.expect_reset_engine_forkchoice().times(1).return_once(|| Ok(()));
    let cancellation = CancellationToken::new();
    let task = tokio::spawn(builder.build(
        TestAttributesBuilder { attributes: vec![] },
        engine,
        MockOriginSelector::new(),
        config(),
        MockUnsafePayloadGossipClient::new(),
        cancellation.clone(),
    ));
    let command = tokio::spawn(async move { handle.override_leader().await });
    started.notified().await;
    cancellation.cancel();
    task.await.unwrap().unwrap();
    assert!(matches!(command.await.unwrap(), Err(HandleError::RequestError(_))));
}
