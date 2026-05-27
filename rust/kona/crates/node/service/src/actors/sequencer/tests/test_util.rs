use crate::{
    SequencerActor,
    actors::{
        MockConductor, MockOriginSelector, MockSequencerEngineClient, MockUnsafePayloadGossipClient,
    },
};
use kona_derive::test_utils::TestAttributesBuilder;
use kona_genesis::RollupConfig;
use std::sync::Arc;
use tokio::sync::mpsc;

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
    )
}
