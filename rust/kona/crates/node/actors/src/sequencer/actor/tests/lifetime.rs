use crate::{
    Conductor, ConductorError, EngineClientError, EngineClientResult, MockConductor, MockOrigin,
    MockSequencerEngineClient, SequencerEngineClient,
    engine::{BuildTaskError, EngineBuildError},
    sequencer::{Builder, HandleError, MockSigner},
};
use alloy_primitives::B256;
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

    async fn leader(&self) -> Result<bool, ConductorError> {
        self.0.notify_one();
        pending().await
    }
}

/// A build request, which the test answers, or holds to observe the sequencer abandoning it.
type BuildRequest = oneshot::Sender<EngineClientResult<PayloadId>>;

/// Forwards the sequencer's build requests to the test.
#[derive(Debug)]
struct ForwardingEngine(mpsc::Sender<BuildRequest>);

#[async_trait]
impl SequencerEngineClient for ForwardingEngine {
    async fn start_build_block(
        &self,
        _attributes: OpAttributesWithParent,
    ) -> EngineClientResult<PayloadId> {
        let (tx, rx) = oneshot::channel();
        self.0.send(tx).await.unwrap();
        rx.await.unwrap()
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

#[tokio::test(start_paused = true)]
async fn sequencing_continues_without_command_handles() {
    let mut engine = MockSequencerEngineClient::new();
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
    handle.start(B256::ZERO).await.unwrap();
    drop(handle);
    time::timeout(Duration::from_secs(5), reached_builds.notified()).await.unwrap();
    task.abort();
    assert!(task.await.unwrap_err().is_cancelled());
    assert!(builds.load(Ordering::Relaxed) >= 3);
}

#[tokio::test]
async fn stopped_actor_without_command_handles_stays_pending() {
    let task = Builder::new().build(
        TestAttributesBuilder { attributes: vec![] },
        MockSequencerEngineClient::new(),
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
    handle.start(B256::ZERO).await.unwrap();
    let build = requests.recv().await.unwrap();
    task.abort();
    assert!(task.await.unwrap_err().is_cancelled());
    assert!(build.is_closed());
}

#[tokio::test]
async fn aborting_lifetime_interrupts_an_in_flight_message() {
    let started = Arc::new(Notify::new());
    let builder = Builder::new();
    let handle = builder.handle();
    let task = tokio::spawn(builder.build(
        TestAttributesBuilder { attributes: vec![] },
        MockSequencerEngineClient::new(),
        MockOrigin::new(),
        config(),
        Some(PendingConductor(started.clone())),
        MockSigner::new(),
    ));
    let command = tokio::spawn(async move { handle.start(B256::ZERO).await });
    started.notified().await;
    task.abort();
    assert!(task.await.unwrap_err().is_cancelled());
    assert!(matches!(command.await.unwrap(), Err(HandleError::RequestError(_))));
}
