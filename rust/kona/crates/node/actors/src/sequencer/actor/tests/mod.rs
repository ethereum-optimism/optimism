use super::Actor;
use crate::{
    MockConductor, MockOrigin, MockSequencerEngineClient,
    sequencer::{Handle, MockSigner, Signer, handle::Message},
};
use async_trait::async_trait;
use kona_derive::test_utils::TestAttributesBuilder;
use kona_genesis::RollupConfig;
use op_alloy_rpc_types_engine::OpExecutionPayloadEnvelope;
use std::sync::Arc;
use tokio::sync::{mpsc, watch};

mod admin;
mod building;

type TestActor =
    Actor<TestAttributesBuilder, MockConductor, MockOrigin, MockSequencerEngineClient, MockSigner>;

fn test_actor() -> TestActor {
    // Drop the sender so block-building tests have no admin requests.
    test_actor_with_config(true, None).0
}

fn test_actor_with_config(
    active: bool,
    conductor: Option<MockConductor>,
) -> (TestActor, mpsc::Sender<Message>, Handle) {
    let (commands_tx, commands_rx) = mpsc::channel(20);
    let (is_active_tx, is_active_rx) = watch::channel(active);
    let handle = Handle::new(is_active_rx, commands_tx.clone());
    let actor = Actor::new(
        commands_rx,
        is_active_tx,
        TestAttributesBuilder { attributes: vec![] },
        conductor,
        MockSequencerEngineClient::new(),
        MockOrigin::new(),
        Arc::new(RollupConfig { block_time: 2, ..Default::default() }),
        MockSigner::new(),
    );
    (actor, commands_tx, handle)
}

mod lifetime;

/// A signer that queues payloads on a channel, and has room exactly when the channel does.
#[derive(Debug)]
struct QueueSigner(mpsc::Sender<OpExecutionPayloadEnvelope>);

#[async_trait]
impl Signer for QueueSigner {
    type Error = std::io::Error;

    async fn send(&self, payload: OpExecutionPayloadEnvelope) -> Result<(), Self::Error> {
        self.0.send(payload).await.map_err(std::io::Error::other)
    }

    async fn ready(&self) -> Result<(), Self::Error> {
        self.0.reserve().await.map(drop).map_err(std::io::Error::other)
    }
}

/// Replaces the actor's signer.
fn with_signer<S: Signer>(
    actor: TestActor,
    signer: S,
) -> Actor<TestAttributesBuilder, MockConductor, MockOrigin, MockSequencerEngineClient, S> {
    Actor {
        messages_rx: actor.messages_rx,
        is_active: actor.is_active,
        is_active_tx: actor.is_active_tx,
        attributes_builder: actor.attributes_builder,
        conductor: actor.conductor,
        engine_client: actor.engine_client,
        origin: actor.origin,
        rollup_config: actor.rollup_config,
        signer,
    }
}
