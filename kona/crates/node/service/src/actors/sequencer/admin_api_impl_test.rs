use crate::{
    BlockEngineError, ConductorError, SequencerActorBuilder, SequencerAdminQuery,
    actors::{
        MockBlockBuildingClient, MockConductor, MockOriginSelector, MockUnsafePayloadGossipClient,
    },
};
use alloy_primitives::B256;
use alloy_transport::RpcError;
use kona_derive::test_utils::TestAttributesBuilder;
use kona_genesis::RollupConfig;
use kona_protocol::{BlockInfo, L2BlockInfo};
use kona_rpc::{SequencerAdminAPIError, StopSequencerError};
use rstest::rstest;
use std::{sync::Arc, vec};
use tokio::sync::{mpsc, oneshot};
use tokio_util::sync::CancellationToken;

// Returns a test SequencerActorBuilder with mocks that can be used or overridden.
fn test_builder() -> SequencerActorBuilder<
    TestAttributesBuilder,
    MockBlockBuildingClient,
    MockConductor,
    MockOriginSelector,
    MockUnsafePayloadGossipClient,
> {
    let (_admin_api_tx, admin_api_rx) = mpsc::channel(20);
    SequencerActorBuilder::new()
        .with_active_status(true)
        .with_admin_api_receiver(admin_api_rx)
        .with_attributes_builder(TestAttributesBuilder { attributes: vec![] })
        .with_block_building_client(MockBlockBuildingClient::new())
        .with_cancellation_token(CancellationToken::new())
        .with_origin_selector(MockOriginSelector::new())
        .with_recovery_mode_status(false)
        .with_rollup_config(Arc::new(RollupConfig::default()))
        .with_unsafe_payload_gossip_client(MockUnsafePayloadGossipClient::new())
}

#[rstest]
#[tokio::test]
async fn test_is_sequencer_active(
    #[values(true, false)] active: bool,
    #[values(true, false)] via_channel: bool,
) {
    let mut actor = test_builder().with_active_status(active).build().unwrap();

    let result = async {
        match via_channel {
            false => actor.is_sequencer_active().await,
            true => {
                let (tx, rx) = oneshot::channel();
                actor.handle_admin_query(SequencerAdminQuery::SequencerActive(tx)).await;
                rx.await.unwrap()
            }
        }
    }
    .await;

    assert!(result.is_ok());
    assert_eq!(active, result.unwrap());
}

#[rstest]
#[tokio::test]
async fn test_is_conductor_enabled(
    #[values(true, false)] conductor_exists: bool,
    #[values(true, false)] via_channel: bool,
) {
    let mut actor = {
        if conductor_exists {
            test_builder().with_conductor(MockConductor::new())
        } else {
            test_builder()
        }
    }
    .build()
    .unwrap();

    let result = async {
        match via_channel {
            false => actor.is_conductor_enabled().await,
            true => {
                let (tx, rx) = oneshot::channel();
                actor.handle_admin_query(SequencerAdminQuery::ConductorEnabled(tx)).await;
                rx.await.unwrap()
            }
        }
    }
    .await;

    assert!(result.is_ok());
    assert_eq!(conductor_exists, result.unwrap());
}

#[rstest]
#[tokio::test]
async fn test_in_recovery_mode(
    #[values(true, false)] recovery_mode: bool,
    #[values(true, false)] via_channel: bool,
) {
    let mut actor = test_builder().with_recovery_mode_status(recovery_mode).build().unwrap();

    let result = async {
        match via_channel {
            false => actor.in_recovery_mode().await,
            true => {
                let (tx, rx) = oneshot::channel();
                actor.handle_admin_query(SequencerAdminQuery::RecoveryMode(tx)).await;
                rx.await.unwrap()
            }
        }
    }
    .await;

    assert!(result.is_ok());
    assert_eq!(recovery_mode, result.unwrap());
}

#[rstest]
#[tokio::test]
async fn test_start_sequencer(
    #[values(true, false)] already_started: bool,
    #[values(true, false)] via_channel: bool,
) {
    let mut actor = test_builder().with_active_status(already_started).build().unwrap();

    // verify starting state
    let result = actor.is_sequencer_active().await;
    assert!(result.is_ok());
    assert_eq!(result.unwrap(), already_started);

    // start the sequencer
    let result = async {
        match via_channel {
            false => actor.start_sequencer().await,
            true => {
                let (tx, rx) = oneshot::channel();
                actor.handle_admin_query(SequencerAdminQuery::StartSequencer(tx)).await;
                rx.await.unwrap()
            }
        }
    }
    .await;
    assert!(result.is_ok());

    // verify it is started
    let result = actor.is_sequencer_active().await;
    assert!(result.is_ok());
    assert!(result.unwrap());
}

#[rstest]
#[tokio::test]
async fn test_stop_sequencer_success(
    #[values(true, false)] already_stopped: bool,
    #[values(true, false)] via_channel: bool,
) {
    let unsafe_head = L2BlockInfo {
        block_info: BlockInfo { hash: B256::from([1u8; 32]), ..Default::default() },
        ..Default::default()
    };
    let expected_hash = unsafe_head.hash();

    let mut client = MockBlockBuildingClient::new();
    client.expect_get_unsafe_head().times(1).return_once(move || Ok(unsafe_head));

    let mut actor = test_builder()
        .with_block_building_client(client)
        .with_active_status(!already_stopped)
        .build()
        .unwrap();

    // verify starting state
    let result = actor.is_sequencer_active().await;
    assert!(result.is_ok());
    assert_eq!(result.unwrap(), !already_stopped);

    // stop the sequencer
    let result = async {
        match via_channel {
            false => actor.stop_sequencer().await,
            true => {
                let (tx, rx) = oneshot::channel();
                actor.handle_admin_query(SequencerAdminQuery::StopSequencer(tx)).await;
                rx.await.unwrap()
            }
        }
    }
    .await;
    assert!(result.is_ok());
    assert_eq!(result.unwrap(), expected_hash);

    // verify ending state
    let result = actor.is_sequencer_active().await;
    assert!(result.is_ok());
    assert!(!result.unwrap());
}

#[rstest]
#[tokio::test]
async fn test_stop_sequencer_error_fetching_unsafe_head(#[values(true, false)] via_channel: bool) {
    let mut client = MockBlockBuildingClient::new();
    client
        .expect_get_unsafe_head()
        .times(1)
        .return_once(|| Err(BlockEngineError::RequestError("whoops!".to_string())));

    let mut actor = test_builder().with_block_building_client(client).build().unwrap();

    let result = async {
        match via_channel {
            false => actor.stop_sequencer().await,
            true => {
                let (tx, rx) = oneshot::channel();
                actor.handle_admin_query(SequencerAdminQuery::StopSequencer(tx)).await;
                rx.await.unwrap()
            }
        }
    }
    .await;
    assert!(result.is_err());

    assert!(matches!(
        result.unwrap_err(),
        SequencerAdminAPIError::StopError(StopSequencerError::ErrorAfterSequencerWasStopped(_))
    ));
    assert!(!actor.is_active);
}

#[rstest]
#[tokio::test]
async fn test_set_recovery_mode(
    #[values(true, false)] starting_mode: bool,
    #[values(true, false)] mode_to_set: bool,
    #[values(true, false)] via_channel: bool,
) {
    let mut actor = test_builder().with_recovery_mode_status(starting_mode).build().unwrap();

    // verify starting state
    let result = actor.in_recovery_mode().await;
    assert!(result.is_ok());
    assert_eq!(result.unwrap(), starting_mode);

    // set recovery mode
    let result = async {
        match via_channel {
            false => actor.set_recovery_mode(mode_to_set).await,
            true => {
                let (tx, rx) = oneshot::channel();
                actor
                    .handle_admin_query(SequencerAdminQuery::SetRecoveryMode(mode_to_set, tx))
                    .await;
                rx.await.unwrap()
            }
        }
    }
    .await;
    assert!(result.is_ok());

    // verify it is set
    let result = actor.in_recovery_mode().await;
    assert!(result.is_ok());
    assert_eq!(result.unwrap(), mode_to_set);
}

#[rstest]
#[tokio::test]
async fn test_override_leader(
    #[values(true, false)] conductor_configured: bool,
    #[values(true, false)] conductor_error: bool,
    #[values(true, false)] via_channel: bool,
) {
    // mock error string returned by conductor, if configured (to differentiate between error
    // returned if not configured)
    let conductor_error_string = "test: error within conductor";

    let mut actor = {
        // wire up conductor absence/presence and response error/success
        if !conductor_configured {
            test_builder()
        } else if conductor_error {
            let mut conductor = MockConductor::new();
            conductor.expect_override_leader().times(1).return_once(move || {
                Err(ConductorError::Rpc(RpcError::local_usage_str(conductor_error_string)))
            });
            test_builder().with_conductor(conductor)
        } else {
            let mut conductor = MockConductor::new();
            conductor.expect_override_leader().times(1).return_once(|| Ok(()));
            test_builder().with_conductor(conductor)
        }
    }
    .build()
    .unwrap();

    // call to override leader
    let result = async {
        match via_channel {
            false => actor.override_leader().await,
            true => {
                let (tx, rx) = oneshot::channel();
                actor.handle_admin_query(SequencerAdminQuery::OverrideLeader(tx)).await;
                rx.await.unwrap()
            }
        }
    }
    .await;

    // verify result
    if !conductor_configured || conductor_error {
        assert!(result.is_err());
        assert_eq!(
            conductor_configured,
            result.err().unwrap().to_string().contains(conductor_error_string)
        );
    } else {
        assert!(result.is_ok())
    }
}

#[rstest]
#[tokio::test]
async fn test_reset_derivation_pipeline_success(#[values(true, false)] via_channel: bool) {
    let mut client = MockBlockBuildingClient::new();
    client.expect_reset_engine_forkchoice().times(1).return_once(|| Ok(()));

    let mut actor = test_builder().with_block_building_client(client).build().unwrap();

    let result = async {
        match via_channel {
            false => actor.reset_derivation_pipeline().await,
            true => {
                let (tx, rx) = oneshot::channel();
                actor.handle_admin_query(SequencerAdminQuery::ResetDerivationPipeline(tx)).await;
                rx.await.unwrap()
            }
        }
    }
    .await;

    assert!(result.is_ok());
}

#[rstest]
#[tokio::test]
async fn test_reset_derivation_pipeline_error(#[values(true, false)] via_channel: bool) {
    let mut client = MockBlockBuildingClient::new();
    client
        .expect_reset_engine_forkchoice()
        .times(1)
        .return_once(|| Err(BlockEngineError::RequestError("reset failed".to_string())));

    let mut actor = test_builder().with_block_building_client(client).build().unwrap();

    let result = async {
        match via_channel {
            false => actor.reset_derivation_pipeline().await,
            true => {
                let (tx, rx) = oneshot::channel();
                actor.handle_admin_query(SequencerAdminQuery::ResetDerivationPipeline(tx)).await;
                rx.await.unwrap()
            }
        }
    }
    .await;

    assert!(result.is_err());
    assert!(result.unwrap_err().to_string().contains("Failed to reset engine"));
}

#[rstest]
#[tokio::test]
async fn test_handle_admin_query_resilient_to_dropped_receiver() {
    let mut conductor = MockConductor::new();
    conductor.expect_override_leader().times(1).returning(|| Ok(()));

    let unsafe_head = L2BlockInfo {
        block_info: BlockInfo { hash: B256::from([1u8; 32]), ..Default::default() },
        ..Default::default()
    };
    let mut client = MockBlockBuildingClient::new();
    client.expect_get_unsafe_head().times(1).returning(move || Ok(unsafe_head));
    client.expect_reset_engine_forkchoice().times(1).returning(|| Ok(()));

    let mut actor = test_builder()
        .with_conductor(conductor)
        .with_block_building_client(client)
        .build()
        .unwrap();

    let mut queries: Vec<SequencerAdminQuery> = Vec::new();
    {
        // immediately drop receiver
        let (tx, _rx) = oneshot::channel();
        queries.push(SequencerAdminQuery::SequencerActive(tx));
    }
    {
        // immediately drop receiver
        let (tx, _rx) = oneshot::channel();
        queries.push(SequencerAdminQuery::StartSequencer(tx));
    }
    {
        // immediately drop receiver
        let (tx, _rx) = oneshot::channel();
        queries.push(SequencerAdminQuery::StopSequencer(tx));
    }
    {
        // immediately drop receiver
        let (tx, _rx) = oneshot::channel();
        queries.push(SequencerAdminQuery::ConductorEnabled(tx));
    }
    {
        // immediately drop receiver
        let (tx, _rx) = oneshot::channel();
        queries.push(SequencerAdminQuery::RecoveryMode(tx));
    }
    {
        // immediately drop receiver
        let (tx, _rx) = oneshot::channel();
        queries.push(SequencerAdminQuery::SetRecoveryMode(true, tx));
    }
    {
        // immediately drop receiver
        let (tx, _rx) = oneshot::channel();
        queries.push(SequencerAdminQuery::OverrideLeader(tx));
    }
    {
        // immediately drop receiver
        let (tx, _rx) = oneshot::channel();
        queries.push(SequencerAdminQuery::ResetDerivationPipeline(tx));
    }

    // None of these should fail even if the receiver is dropped
    for query in queries {
        actor.handle_admin_query(query).await;
    }
}
