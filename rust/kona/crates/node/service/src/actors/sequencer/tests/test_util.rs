use super::super::SequencerHandle;
use crate::{
    SequencerActor,
    actors::{MockConductor, MockOriginSelector, MockUnsafePayloadGossipClient},
};
use kona_derive::test_utils::TestAttributesBuilder;
use kona_genesis::RollupConfig;
use std::sync::Arc;

pub(crate) type TestActor = SequencerActor<
    TestAttributesBuilder,
    MockConductor,
    MockOriginSelector,
    MockUnsafePayloadGossipClient,
>;

pub(crate) fn test_actor() -> TestActor {
    test_actor_with_handle(true, false, None).0
}

pub(crate) fn test_actor_with_handle(
    active: bool,
    recovery_mode: bool,
    conductor: Option<MockConductor>,
) -> (TestActor, SequencerHandle) {
    SequencerActor::new(
        TestAttributesBuilder { attributes: vec![] },
        conductor,
        crate::EngineHandle::test_channel(kona_engine::EngineState::default()).0,
        active,
        recovery_mode,
        MockOriginSelector::new(),
        // tokio::time::interval requires a non-zero period.
        Arc::new(RollupConfig { block_time: 2, ..Default::default() }),
        MockUnsafePayloadGossipClient::new(),
    )
}
