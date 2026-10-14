use crate::{
    Conductor, ConductorError, EngineClientError, EngineClientResult, MockConductor, MockOrigin,
    MockSequencerEngineClient, SequencerEngineClient,
    engine::{BuildTaskError, EngineBuildError},
    sequencer::{ActorError, Builder, HandleError, MockSigner},
};
use alloy_rpc_types_engine::PayloadId;
use async_trait::async_trait;
use kona_derive::test_utils::TestAttributesBuilder;
use kona_genesis::RollupConfig;
use kona_protocol::{L2BlockInfo, OpAttributesWithParent};
use op_alloy_rpc_types_engine::OpExecutionPayloadEnvelope;
use std::{
    future::{pending, ready},
    sync::{
        Arc,
        atomic::{AtomicUsize, Ordering},
    },
};
use tokio::{
    sync::{Notify, mpsc, oneshot},
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

/// An engine request the test answers, or holds to observe the sequencer abandoning it.
#[derive(Debug)]
enum EngineRequest {
    Reset(oneshot::Sender<EngineClientResult<()>>),
    Build(oneshot::Sender<EngineClientResult<PayloadId>>),
}

/// Forwards the sequencer's engine requests to the test.
#[derive(Debug)]
struct ForwardingEngine(mpsc::Sender<EngineRequest>);

impl ForwardingEngine {
    async fn request<T>(
        &self,
        request: impl FnOnce(oneshot::Sender<EngineClientResult<T>>) -> EngineRequest,
    ) -> EngineClientResult<T> {
        let (tx, rx) = oneshot::channel();
        self.0.send(request(tx)).await.unwrap();
        rx.await.unwrap()
    }
}

#[async_trait]
impl SequencerEngineClient for ForwardingEngine {
    async fn reset_engine_forkchoice(&self) -> EngineClientResult<()> {
        self.request(EngineRequest::Reset).await
    }

    async fn start_build_block(
        &self,
        _attributes: OpAttributesWithParent,
    ) -> EngineClientResult<PayloadId> {
        self.request(EngineRequest::Build).await
    }

    async fn seal_block(
        &self,
        _payload_id: PayloadId,
        _attributes: OpAttributesWithParent,
    ) -> EngineClientResult<OpExecutionPayloadEnvelope> {
        panic!("no block is built before the sequencer stops")
    }

    async fn canonicalize_block(
        &self,
        _payload: OpExecutionPayloadEnvelope,
        _attributes: OpAttributesWithParent,
    ) -> EngineClientResult<()> {
        panic!("no block is built before the sequencer stops")
    }

    async fn get_unsafe_head(&self) -> EngineClientResult<L2BlockInfo> {
        Ok(L2BlockInfo::default())
    }
}

#[test]
fn construction_and_build_do_not_require_a_runtime() {
    let builder = Builder::new();
    let handle = builder.handle();
    let initial = false;
    assert_eq!(handle.is_active().unwrap(), initial);
    let task = builder.build(
        TestAttributesBuilder { attributes: vec![] },
        MockSequencerEngineClient::new(),
        MockOrigin::new(),
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
    let builder = Builder::new();
    let handle = builder.handle();
    let (engine, mut requests) = mpsc::channel(1);
    let task = tokio::spawn(builder.build(
        TestAttributesBuilder { attributes: vec![] },
        ForwardingEngine(engine),
        MockOrigin::new(),
        config(),
        None::<MockConductor>,
        MockSigner::new(),
    ));
    let EngineRequest::Reset(reply) = requests.recv().await.unwrap() else {
        panic!("expected initial reset");
    };
    task.abort();
    assert!(task.await.unwrap_err().is_cancelled());
    assert!(reply.is_closed());
    assert!(matches!(handle.is_active(), Err(HandleError::RequestError(_))));
}

#[tokio::test(start_paused = true)]
async fn sequencing_continues_without_command_handles() {
    let mut engine = MockSequencerEngineClient::new();
    engine.expect_reset_engine_forkchoice().times(1).return_once(|| Ok(()));
    engine.expect_get_unsafe_head().returning(|| Ok(L2BlockInfo::default()));
    let builds = Arc::new(AtomicUsize::new(0));
    let observed_builds = builds.clone();
    let reached_builds = Arc::new(Notify::new());
    let notify_after_builds = reached_builds.clone();
    // The engine is syncing, so each build waits a block time for the next.
    engine.expect_start_build_block().returning(move |_| {
        if observed_builds.fetch_add(1, Ordering::Relaxed) == 2 {
            notify_after_builds.notify_one();
        }
        Err(EngineClientError::StartBuildError(BuildTaskError::EngineBuildError(
            EngineBuildError::EngineSyncing,
        )))
    });
    let mut origin = MockOrigin::new();
    origin.expect_next().returning(|_| Ok(Default::default()));
    // The actor may build again before it is aborted.
    let attributes = (0..10).map(|_| Ok(Default::default())).collect();
    let mut signer = MockSigner::new();
    signer.expect_ready().returning(|| Ok(()));
    let builder = Builder::new();
    let handle = builder.handle();
    let task = tokio::spawn(builder.build(
        TestAttributesBuilder { attributes },
        engine,
        origin,
        config(),
        None::<MockConductor>,
        signer,
    ));
    handle.start().await.unwrap();
    drop(handle);
    time::timeout(Duration::from_secs(5), reached_builds.notified()).await.unwrap();
    task.abort();
    assert!(task.await.unwrap_err().is_cancelled());
    assert!(builds.load(Ordering::Relaxed) >= 3);
}

#[tokio::test]
async fn stopped_actor_without_command_handles_stays_pending() {
    let mut engine = MockSequencerEngineClient::new();
    engine.expect_reset_engine_forkchoice().times(1).return_once(|| Ok(()));
    let task = Builder::new().build(
        TestAttributesBuilder { attributes: vec![] },
        engine,
        MockOrigin::new(),
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
    let builder = Builder::new();
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
        MockOrigin::new(),
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
    let (engine, mut requests) = mpsc::channel(1);
    let mut origin = MockOrigin::new();
    origin.expect_next().times(1).return_once(|_| Ok(Default::default()));
    let mut signer = MockSigner::new();
    signer.expect_ready().times(1).returning(|| Ok(()));
    let builder = Builder::new();
    let handle = builder.handle();
    let task = tokio::spawn(builder.build(
        TestAttributesBuilder { attributes: vec![Ok(Default::default())] },
        ForwardingEngine(engine),
        origin,
        config(),
        None::<MockConductor>,
        signer,
    ));
    let EngineRequest::Reset(reset) = requests.recv().await.unwrap() else {
        panic!("expected initial reset");
    };
    reset.send(Ok(())).unwrap();
    handle.start().await.unwrap();
    let EngineRequest::Build(build) = requests.recv().await.unwrap() else {
        panic!("expected block build");
    };
    task.abort();
    assert!(task.await.unwrap_err().is_cancelled());
    assert!(build.is_closed());
}

#[tokio::test]
async fn aborting_lifetime_interrupts_an_in_flight_message() {
    let started = Arc::new(Notify::new());
    let builder = Builder::new();
    let handle = builder.handle();
    let mut engine = MockSequencerEngineClient::new();
    engine.expect_reset_engine_forkchoice().times(1).return_once(|| Ok(()));
    let task = tokio::spawn(builder.build(
        TestAttributesBuilder { attributes: vec![] },
        engine,
        MockOrigin::new(),
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
