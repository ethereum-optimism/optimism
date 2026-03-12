//! Node actor path: wraps the real [`DerivationActor`] with mock providers.
//!
//! Constructs a real derivation pipeline backed by [`MockL1Chain`] and a
//! test L2 chain provider, then drives it step-by-step through the
//! [`DerivationActor`].

mod test_engine_client;
mod test_l2_provider;
mod test_pipeline;

pub use test_engine_client::*;
pub use test_l2_provider::*;
pub use test_pipeline::*;

use kona_node_service::{
    DerivationActor, DerivationActorBuilder, DerivationActorRequest, DerivationError, NodeActor,
};
use kona_protocol::{BlockInfo, L2BlockInfo};
use tokio::sync::mpsc;

/// A test wrapper around the real [`DerivationActor`] that provides
/// step-by-step control for deterministic testing.
pub struct TestNodeActor {
    /// The real derivation actor.
    actor: DerivationActor<TestEngineClient, TestPipeline>,
    /// Channel to send requests to the actor.
    request_tx: mpsc::Sender<DerivationActorRequest>,
}

impl TestNodeActor {
    /// Creates a new [`TestNodeActor`] from the given components.
    pub async fn new(engine_client: TestEngineClient, pipeline: TestPipeline) -> Self {
        let (request_tx, request_rx) = mpsc::channel(64);

        let builder = DerivationActorBuilder {
            engine_client,
            inbound_request_rx: request_rx,
            pipeline,
        };

        let ((), actor) = DerivationActor::init(builder)
            .await
            .expect("failed to init derivation actor");

        Self { actor, request_tx }
    }

    /// Steps the derivation actor once (processes one request from the channel).
    pub async fn step(&mut self) -> Result<(), DerivationError> {
        self.actor.step().await
    }

    /// Notifies the actor that the EL sync is complete with the given safe head.
    pub async fn notify_el_sync_complete(&self, safe_head: L2BlockInfo) {
        self.request_tx
            .send(DerivationActorRequest::ProcessEngineSyncCompletionRequest(
                Box::new(safe_head),
            ))
            .await
            .expect("failed to send sync completion request");
    }

    /// Notifies the actor of a new L1 head.
    pub async fn notify_l1_head(&self, l1_head: BlockInfo) {
        self.request_tx
            .send(DerivationActorRequest::ProcessL1HeadUpdateRequest(
                Box::new(l1_head),
            ))
            .await
            .expect("failed to send L1 head update");
    }

    /// Confirms a new safe head from the engine.
    pub async fn confirm_safe_head(&self, safe_head: L2BlockInfo) {
        self.request_tx
            .send(DerivationActorRequest::ProcessEngineSafeHeadUpdateRequest(
                Box::new(safe_head),
            ))
            .await
            .expect("failed to send safe head confirmation");
    }

    /// Sends a finalized L1 block notification.
    pub async fn notify_finalized_l1(&self, finalized: BlockInfo) {
        self.request_tx
            .send(DerivationActorRequest::ProcessFinalizedL1Block(
                Box::new(finalized),
            ))
            .await
            .expect("failed to send finalized L1 block");
    }
}

impl std::fmt::Debug for TestNodeActor {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        f.debug_struct("TestNodeActor").finish()
    }
}
