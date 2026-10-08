use super::SequencerActor;
use crate::actors::{
    MockConductor, MockOriginSelector, MockSequencerEngineClient, MockUnsafePayloadGossipClient,
};
use kona_derive::test_utils::TestAttributesBuilder;
use kona_genesis::RollupConfig;
use kona_rpc::SequencerAdminCommand;
use std::sync::Arc;
use tokio::sync::mpsc;

mod admin;
mod building;

type TestSequencerActor = SequencerActor<
    TestAttributesBuilder,
    MockConductor,
    MockOriginSelector,
    MockSequencerEngineClient,
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
    let (commands_tx, commands_rx) = mpsc::channel(20);
    let actor = SequencerActor::new(
        commands_rx,
        TestAttributesBuilder { attributes: vec![] },
        conductor,
        MockSequencerEngineClient::new(),
        active,
        recovery_mode,
        MockOriginSelector::new(),
        Arc::new(RollupConfig { block_time: 2, ..Default::default() }),
        MockUnsafePayloadGossipClient::new(),
    );
    (actor, commands_tx)
}
