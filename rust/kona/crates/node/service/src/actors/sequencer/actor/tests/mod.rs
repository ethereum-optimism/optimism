use super::SequencerActor;
use crate::{
    Clock, EngineClientResult, SequencerEngineClient,
    actors::{
        MockConductor, MockOriginSelector, MockSequencerEngineClient, MockUnsafePayloadGossipClient,
    },
};
use alloy_rpc_types_engine::PayloadId;
use async_trait::async_trait;
use kona_derive::test_utils::TestAttributesBuilder;
use kona_genesis::RollupConfig;
use kona_protocol::{L2BlockInfo, OpAttributesWithParent};
use kona_rpc::SequencerAdminCommand;
use op_alloy_rpc_types_engine::OpExecutionPayloadEnvelope;
use std::{
    sync::Arc,
    time::{Duration, SystemTime, UNIX_EPOCH},
};
use tokio::{sync::mpsc, time::Instant};

mod admin;
mod building;
mod conductor_commit_test;

const TEST_START: u64 = 946_684_800;

type TestSequencerActor = TestActor;

type TestActor<E = MockSequencerEngineClient> = SequencerActor<
    TestAttributesBuilder,
    MockConductor,
    MockOriginSelector,
    E,
    MockUnsafePayloadGossipClient,
>;

fn test_actor() -> TestSequencerActor {
    // Drop the sender so block-building tests have no admin requests.
    test_actor_with_config(true, false, None).0
}

fn test_actor_with_config(
    active: bool,
    recovery_mode: bool,
    conductor: Option<MockConductor>,
) -> (TestSequencerActor, mpsc::Sender<SequencerAdminCommand>) {
    test_actor_with_engine_config(
        MockSequencerEngineClient::new(),
        active,
        recovery_mode,
        conductor,
    )
}

fn test_actor_with_engine<E: SequencerEngineClient>(engine: E) -> TestActor<E> {
    test_actor_with_engine_config(engine, true, false, None).0
}

fn test_actor_with_engine_config<E: SequencerEngineClient>(
    engine: E,
    active: bool,
    recovery_mode: bool,
    conductor: Option<MockConductor>,
) -> (TestActor<E>, mpsc::Sender<SequencerAdminCommand>) {
    let (commands_tx, commands_rx) = mpsc::channel(20);
    let actor = SequencerActor::new(
        commands_rx,
        TestAttributesBuilder { attributes: vec![] },
        conductor,
        engine,
        active,
        recovery_mode,
        MockOriginSelector::new(),
        Arc::new(RollupConfig { block_time: 2, ..Default::default() }),
        MockUnsafePayloadGossipClient::new(),
        Arc::new(TokioClock::new(UNIX_EPOCH + Duration::from_secs(TEST_START))),
    );
    (actor, commands_tx)
}

/// A wall clock that starts at a given time and moves with tokio's clock, so it pauses and
/// advances with tests that pause tokio's time.
#[derive(Debug)]
pub(crate) struct TokioClock {
    start: SystemTime,
    started_at: Instant,
}

impl TokioClock {
    /// Returns a clock that reads `start` now.
    pub(crate) fn new(start: SystemTime) -> Self {
        Self { start, started_at: Instant::now() }
    }
}

impl Clock for TokioClock {
    fn now(&self) -> SystemTime {
        self.start + self.started_at.elapsed()
    }
}

/// An engine client whose seals take `seal_time` on tokio's clock, delegating everything to a mock.
#[derive(Debug)]
pub(crate) struct SlowSealEngine {
    pub(crate) inner: MockSequencerEngineClient,
    pub(crate) seal_time: Duration,
}

#[async_trait]
impl SequencerEngineClient for SlowSealEngine {
    async fn reset_engine_forkchoice(&self) -> EngineClientResult<()> {
        self.inner.reset_engine_forkchoice().await
    }

    async fn start_build_block(
        &self,
        attributes: OpAttributesWithParent,
    ) -> EngineClientResult<PayloadId> {
        self.inner.start_build_block(attributes).await
    }

    async fn seal_block(
        &self,
        payload_id: PayloadId,
        attributes: OpAttributesWithParent,
    ) -> EngineClientResult<OpExecutionPayloadEnvelope> {
        tokio::time::sleep(self.seal_time).await;
        self.inner.seal_block(payload_id, attributes).await
    }

    async fn canonicalize_block(&self, payload: OpExecutionPayloadEnvelope, parent: L2BlockInfo) -> EngineClientResult<()> {
        self.inner.canonicalize_block(payload, parent).await
    }

    async fn get_unsafe_head(&self) -> EngineClientResult<L2BlockInfo> {
        self.inner.get_unsafe_head().await
    }
}
