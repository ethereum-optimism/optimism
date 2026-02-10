use crate::{
    SequencerActor,
    actors::{
        MockConductor, MockOriginSelector, MockSequencerEngineClient, MockUnsafePayloadGossipClient,
    },
};
use kona_derive::test_utils::TestAttributesBuilder;
use kona_genesis::RollupConfig;
use std::{sync::Arc, time::Duration};
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
    let rollup_config = Arc::new(RollupConfig { block_time: 2, ..Default::default() });
    let build_ticker = tokio::time::interval(Duration::from_secs(rollup_config.block_time));
    SequencerActor {
        admin_api_rx,
        attributes_builder: TestAttributesBuilder { attributes: vec![] },
        conductor: None,
        engine_client: MockSequencerEngineClient::new(),
        is_active: true,
        in_recovery_mode: false,
        origin_selector: MockOriginSelector::new(),
        rollup_config,
        unsafe_payload_gossip_client: MockUnsafePayloadGossipClient::new(),
        build_ticker,
        next_payload_to_seal: None,
        last_seal_duration: Duration::from_secs(0),
        needs_reset: false,
    }
}
