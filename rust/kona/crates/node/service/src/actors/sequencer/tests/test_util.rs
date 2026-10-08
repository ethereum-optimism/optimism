use crate::{
    Clock, EngineClientResult, SequencerActor, SequencerEngineClient,
    actors::{
        MockConductor, MockOriginSelector, MockSequencerEngineClient, MockUnsafePayloadGossipClient,
    },
};
use alloy_rpc_types_engine::PayloadId;
use async_trait::async_trait;
use kona_derive::test_utils::TestAttributesBuilder;
use kona_genesis::RollupConfig;
use kona_protocol::{L2BlockInfo, OpAttributesWithParent};
use op_alloy_rpc_types_engine::OpExecutionPayloadEnvelope;
use std::{
    sync::Arc,
    time::{Duration, SystemTime, UNIX_EPOCH},
};
use tokio::{sync::mpsc, time::Instant};

/// The time, in seconds since the UNIX epoch, that the clock of a [`test_actor`] starts at:
/// 2000-01-01T00:00:00Z.
pub(crate) const TEST_START: u64 = 946_684_800;

/// A [`SequencerActor`] built from mocks, with the engine client `E`.
pub(crate) type TestActor<E = MockSequencerEngineClient> = SequencerActor<
    TestAttributesBuilder,
    MockConductor,
    MockOriginSelector,
    E,
    MockUnsafePayloadGossipClient,
>;

pub(crate) type TestSequencerActor = TestActor;

// Returns a test SequencerActor with mocks that can be used or overridden.
pub(crate) fn test_actor() -> TestActor {
    test_actor_with_engine(MockSequencerEngineClient::new())
}

/// Returns a test [`SequencerActor`] like [`test_actor`], with the given engine client.
pub(crate) fn test_actor_with_conductor(conductor: Option<MockConductor>) -> TestSequencerActor {
    test_actor_with_engine_and_conductor(MockSequencerEngineClient::new(), conductor)
}

pub(crate) fn test_actor_with_engine<E: SequencerEngineClient>(engine_client: E) -> TestActor<E> {
    test_actor_with_engine_and_conductor(engine_client, None)
}

fn test_actor_with_engine_and_conductor<E: SequencerEngineClient>(engine_client: E, conductor: Option<MockConductor>) -> TestActor<E> {
    // The sender is intentionally dropped, so the channel starts closed.
    // If future tests need to send messages, keep the sender instead of dropping it.
    let (_admin_api_tx, admin_api_rx) = mpsc::channel(20);
    SequencerActor::new(
        admin_api_rx,
        TestAttributesBuilder { attributes: vec![] },
        conductor,
        engine_client,
        true,
        false,
        MockOriginSelector::new(),
        // tokio::time::interval requires a non-zero period; the default block_time is 0.
        Arc::new(RollupConfig { block_time: 2, ..Default::default() }),
        MockUnsafePayloadGossipClient::new(),
        Arc::new(TokioClock::new(UNIX_EPOCH + Duration::from_secs(TEST_START))),
    )
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

    async fn seal_and_canonicalize_block(
        &self,
        payload_id: PayloadId,
        attributes: OpAttributesWithParent,
    ) -> EngineClientResult<OpExecutionPayloadEnvelope> {
        tokio::time::sleep(self.seal_time).await;
        self.inner.seal_and_canonicalize_block(payload_id, attributes).await
    }

    async fn get_unsafe_head(&self) -> EngineClientResult<L2BlockInfo> {
        self.inner.get_unsafe_head().await
    }
}
