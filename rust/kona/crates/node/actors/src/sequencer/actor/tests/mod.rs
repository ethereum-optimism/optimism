use super::Actor;
use crate::{
    MockConductor, MockOriginSelector, MockSequencerEngineClient,
    sequencer::{Handle, MockSigner, handle::Message},
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
    MockSigner,
>;

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
        MockOriginSelector::new(),
        Arc::new(RollupConfig { block_time: 2, ..Default::default() }),
        MockSigner::new(),
    );
    (actor, commands_tx, handle)
}

mod lifetime;
