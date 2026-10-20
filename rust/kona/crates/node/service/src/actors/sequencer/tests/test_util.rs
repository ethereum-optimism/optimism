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

/// Run one public command against the actual owner handler.
pub(crate) async fn run_command<T>(
    actor: &mut TestActor,
    request: impl std::future::Future<Output = Result<T, kona_rpc::SequencerAdminAPIError>>,
) -> Result<T, kona_rpc::SequencerAdminAPIError> {
    tokio::join!(request, async {
        let command = actor.commands.recv().await.unwrap();
        actor.handle_command(command).await.unwrap();
    })
    .0
}

pub(crate) fn known_head() -> kona_protocol::L2BlockInfo {
    kona_protocol::L2BlockInfo {
        block_info: kona_protocol::BlockInfo {
            hash: alloy_primitives::B256::repeat_byte(7),
            number: 12,
            timestamp: 100,
            ..Default::default()
        },
        ..Default::default()
    }
}

pub(crate) fn install_engine(actor: &mut TestActor) -> crate::actors::engine::handle::TestEngine {
    let mut state =
        kona_engine::EngineState { phase: kona_engine::EnginePhase::Ready, ..Default::default() };
    state.sync_state = state.sync_state.apply_update(kona_engine::EngineSyncStateUpdate {
        unsafe_head: Some(known_head()),
        ..Default::default()
    });
    let (handle, engine) = crate::EngineHandle::test_channel(state);
    actor.engine_client = handle;
    engine
}
