use super::Actor;
use crate::{
    MockConductor, MockOriginSelector, MockSequencerEngineClient, MockUnsafePayloadGossipClient,
    sequencer::{State, handle::Message},
};
use kona_derive::test_utils::TestAttributesBuilder;
use kona_genesis::RollupConfig;
use std::sync::Arc;
use tokio::sync::{mpsc, watch};

mod admin;
mod building;

type TestActor = Actor<
    TestAttributesBuilder,
    MockConductor,
    MockOriginSelector,
    MockSequencerEngineClient,
    MockUnsafePayloadGossipClient,
>;

fn test_actor() -> TestActor {
    // Drop the sender so block-building tests have no admin requests.
    test_actor_with_config(true, false, None).0
}

fn test_actor_with_config(
    active: bool,
    recovery_mode: bool,
    conductor: Option<MockConductor>,
) -> (TestActor, mpsc::Sender<Message>) {
    let (commands_tx, commands_rx) = mpsc::channel(20);
    let state = State { active, recovery_mode, conductor_enabled: conductor.is_some() };
    let (published, _) = watch::channel(state);
    let actor = Actor::new(
        commands_rx,
        published,
        state,
        TestAttributesBuilder { attributes: vec![] },
        conductor,
        MockSequencerEngineClient::new(),
        MockOriginSelector::new(),
        Arc::new(RollupConfig { block_time: 2, ..Default::default() }),
        MockUnsafePayloadGossipClient::new(),
    );
    (actor, commands_tx)
}

mod lifetime;
