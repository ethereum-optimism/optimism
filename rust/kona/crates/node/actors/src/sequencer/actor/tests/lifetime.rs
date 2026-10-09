use crate::{
    Conductor, ConductorError, EngineClientError, MockConductor, MockOriginSelector,
    MockSequencerEngineClient, QueuedSequencerEngineClient,
    sequencer::{ActorError, Builder, Capacity, HandleError, MockSigner},
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
    let builder = Builder::new(Capacity::try_from(1).unwrap());
    let handle = builder.handle();
    let initial = false;
    assert_eq!(handle.is_active().unwrap(), initial);
    let task = builder.build(
        TestAttributesBuilder { attributes: vec![] },
        MockSequencerEngineClient::new(),
        MockOriginSelector::new(),
        config(),
        Some(MockConductor::new()),
        MockSigner::new(),
    );
    assert_eq!(handle.is_active().unwrap(), initial);
    drop(task);
    assert!(matches!(handle.is_active(), Err(HandleError::RequestError(_))));
}

#[tokio::test]
async fn aborting_lifetime_interrupts_a_pending_initial_reset() {
    let builder = Builder::new(Capacity::try_from(1).unwrap());
    let handle = builder.handle();
    let (engine_actor_request_tx, mut requests) = mpsc::channel(1);
    let (_, unsafe_head_rx) = watch::channel(L2BlockInfo::default());
    let task = tokio::spawn(builder.build(
        TestAttributesBuilder { attributes: vec![] },
        QueuedSequencerEngineClient { engine_actor_request_tx, unsafe_head_rx },
        MockOriginSelector::new(),
        config(),
        None::<MockConductor>,
        MockSigner::new(),
    ));
    let EngineActorRequest::Reset(request) = requests.recv().await.unwrap() else {
        panic!("expected initial reset");
    };
    task.abort();
    assert!(task.await.unwrap_err().is_cancelled());
    assert!(request.result_tx.is_closed());
    assert!(matches!(handle.is_active(), Err(HandleError::RequestError(_))));
}

#[tokio::test(start_paused = true)]
async fn sequencing_continues_without_command_handles() {
    let mut engine = MockSequencerEngineClient::new();
    engine.expect_reset_engine_forkchoice().times(1).return_once(|| Ok(()));
    let ticks = Arc::new(AtomicUsize::new(0));
    let observed_ticks = ticks.clone();
    let reached_ticks = Arc::new(Notify::new());
    let notify_after_ticks = reached_ticks.clone();
    let mut signer = MockSigner::new();
    signer.expect_has_capacity().times(3).returning(move || {
        if observed_ticks.fetch_add(1, Ordering::Relaxed) == 2 {
            notify_after_ticks.notify_one();
        }
        false
    });
    let builder = Builder::new(Capacity::try_from(1).unwrap());
    let handle = builder.handle();
    let task = tokio::spawn(builder.build(
        TestAttributesBuilder { attributes: vec![] },
        engine,
        MockOriginSelector::new(),
        config(),
        None::<MockConductor>,
        signer,
    ));
    handle.start().await.unwrap();
    drop(handle);
    time::timeout(Duration::from_secs(5), reached_ticks.notified()).await.unwrap();
    task.abort();
    assert!(task.await.unwrap_err().is_cancelled());
    assert_eq!(ticks.load(Ordering::Relaxed), 3);
}

#[tokio::test]
async fn stopped_actor_without_command_handles_stays_pending() {
    let mut engine = MockSequencerEngineClient::new();
    engine.expect_reset_engine_forkchoice().times(1).return_once(|| Ok(()));
    let task = Builder::new(Capacity::try_from(1).unwrap()).build(
        TestAttributesBuilder { attributes: vec![] },
        engine,
        MockOriginSelector::new(),
        config(),
        None::<MockConductor>,
        MockSigner::new(),
    );
    let mut task = Box::pin(task);
    tokio::select! {
        biased;
        result = &mut task => panic!("stopped actor exited: {result:?}"),
        _ = ready(()) => {}
    }
    drop(task);
}

#[tokio::test]
async fn failed_startup_rejects_a_previously_queued_command() {
    let builder = Builder::new(Capacity::try_from(1).unwrap());
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
        None::<MockConductor>,
        MockSigner::new(),
    );
    assert!(matches!(task.await, Err(ActorError::EngineError(_))));
    assert!(matches!(command.await, Err(HandleError::RequestError(_))));
    assert!(matches!(handle.is_active(), Err(HandleError::RequestError(_))));
}

#[tokio::test]
async fn aborting_lifetime_interrupts_an_in_flight_block_build() {
    let (engine_actor_request_tx, mut requests) = mpsc::channel(1);
    let (_, unsafe_head_rx) = watch::channel(L2BlockInfo::default());
    let mut origin = MockOriginSelector::new();
    origin.expect_next_l1_origin().times(1).return_once(|_| Ok(Default::default()));
    let mut signer = MockSigner::new();
    signer.expect_has_capacity().times(1).return_const(true);
    let builder = Builder::new(Capacity::try_from(1).unwrap());
    let handle = builder.handle();
    let task = tokio::spawn(builder.build(
        TestAttributesBuilder { attributes: vec![Ok(Default::default())] },
        QueuedSequencerEngineClient { engine_actor_request_tx, unsafe_head_rx },
        origin,
        config(),
        None::<MockConductor>,
        signer,
    ));
    let EngineActorRequest::Reset(reset) = requests.recv().await.unwrap() else {
        panic!("expected initial reset");
    };
    reset.result_tx.send(Ok(())).await.unwrap();
    handle.start().await.unwrap();
    let EngineActorRequest::Build(build) = requests.recv().await.unwrap() else {
        panic!("expected block build");
    };
    task.abort();
    assert!(task.await.unwrap_err().is_cancelled());
    assert!(build.result_tx.is_closed());
}

#[tokio::test]
async fn aborting_lifetime_interrupts_an_in_flight_message() {
    let started = Arc::new(Notify::new());
    let builder = Builder::new(Capacity::try_from(1).unwrap());
    let handle = builder.handle();
    let mut engine = MockSequencerEngineClient::new();
    engine.expect_reset_engine_forkchoice().times(1).return_once(|| Ok(()));
    let task = tokio::spawn(builder.build(
        TestAttributesBuilder { attributes: vec![] },
        engine,
        MockOriginSelector::new(),
        config(),
        Some(PendingConductor(started.clone())),
        MockSigner::new(),
    ));
    let command = tokio::spawn(async move { handle.override_leader().await });
    started.notified().await;
    task.abort();
    assert!(task.await.unwrap_err().is_cancelled());
    assert!(matches!(command.await.unwrap(), Err(HandleError::RequestError(_))));
}
