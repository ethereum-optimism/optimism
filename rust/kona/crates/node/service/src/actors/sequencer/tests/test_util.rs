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
use tokio_util::sync::CancellationToken;

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
    SequencerActor {
        admin_api_rx,
        attributes_builder: TestAttributesBuilder { attributes: vec![] },
        cancellation_token: CancellationToken::new(),
        conductor: None,
        engine_client: MockSequencerEngineClient::new(),
        is_active: true,
        in_recovery_mode: false,
        origin_selector: MockOriginSelector::new(),
        rollup_config: Arc::new(RollupConfig::default()),
        unsafe_payload_gossip_client: MockUnsafePayloadGossipClient::new(),
    }
}
