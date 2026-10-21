use crate::{
    Clock, SequencerActor,
    actors::{
        MockConductor, MockOriginSelector, MockSequencerEngineClient, MockUnsafePayloadGossipClient,
    },
};
use kona_derive::test_utils::TestAttributesBuilder;
use kona_genesis::RollupConfig;
use std::{
    sync::Arc,
    time::{Duration, SystemTime, UNIX_EPOCH},
};
use tokio::{sync::mpsc, time::Instant};

/// The time, in seconds since the UNIX epoch, that the clock of a [`test_actor`] starts at:
/// 2000-01-01T00:00:00Z.
pub(crate) const TEST_START: u64 = 946_684_800;

// Returns a test SequencerActor with mocks that can be used or overridden.
pub(crate) fn test_actor() -> SequencerActor<
    TestAttributesBuilder,
    MockConductor,
    MockOriginSelector,
    MockSequencerEngineClient,
    MockUnsafePayloadGossipClient,
> {
    // The sender is intentionally dropped, so the channel starts closed.
    // If future tests need to send messages, keep the sender instead of dropping it.
    let (_admin_api_tx, admin_api_rx) = mpsc::channel(20);
    SequencerActor::new(
        admin_api_rx,
        TestAttributesBuilder { attributes: vec![] },
        None,
        MockSequencerEngineClient::new(),
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
