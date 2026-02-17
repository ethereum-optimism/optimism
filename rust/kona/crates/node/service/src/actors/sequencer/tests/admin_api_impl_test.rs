use crate::{
    ConductorError, EngineClientError, SequencerAdminQuery,
    actors::{MockConductor, MockSequencerEngineClient, sequencer::tests::test_util::test_actor},
};
use alloy_primitives::B256;
use alloy_transport::RpcError;
use kona_protocol::{BlockInfo, L2BlockInfo};
use kona_rpc::SequencerAdminAPIError;
use rstest::rstest;
use tokio::sync::oneshot;

#[rstest]
#[tokio::test]
async fn test_is_sequencer_active(
    #[values(true, false)] active: bool,
    #[values(true, false)] via_channel: bool,
) {
    let mut actor = test_actor();
    actor.is_active = active;

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
    let mut actor = test_actor();
    if conductor_exists {
        actor.conductor = Some(MockConductor::new())
    };

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
    let mut actor = test_actor();
    actor.in_recovery_mode = recovery_mode;

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
    let mut actor = test_actor();
    actor.is_active = already_started;

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

    let mut client = MockSequencerEngineClient::new();
    client.expect_get_unsafe_head().times(1).return_once(move || Ok(unsafe_head));

    let mut actor = test_actor();
    actor.engine_client = client;
    actor.is_active = !already_stopped;

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
    let mut client = MockSequencerEngineClient::new();
    client
        .expect_get_unsafe_head()
        .times(1)
        .return_once(|| Err(EngineClientError::RequestError("whoops!".to_string())));

    let mut actor = test_actor();
    actor.engine_client = client;

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
        SequencerAdminAPIError::ErrorAfterSequencerWasStopped(_)
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
    let mut actor = test_actor();
    actor.in_recovery_mode = starting_mode;

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
            test_actor()
        } else if conductor_error {
            let mut conductor = MockConductor::new();
            conductor.expect_override_leader().times(1).return_once(move || {
                Err(ConductorError::Rpc(RpcError::local_usage_str(conductor_error_string)))
            });
            let mut actor = test_actor();
            actor.conductor = Some(conductor);
            actor
        } else {
            let mut conductor = MockConductor::new();
            conductor.expect_override_leader().times(1).return_once(|| Ok(()));
            let mut actor = test_actor();
            actor.conductor = Some(conductor);
            actor
        }
    };

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
    let mut client = MockSequencerEngineClient::new();
    client.expect_reset_engine_forkchoice().times(1).return_once(|| Ok(()));

    let mut actor = test_actor();
    actor.engine_client = client;

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
    let mut client = MockSequencerEngineClient::new();
    client
        .expect_reset_engine_forkchoice()
        .times(1)
        .return_once(|| Err(EngineClientError::RequestError("reset failed".to_string())));

    let mut actor = test_actor();
    actor.engine_client = client;

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
    let mut client = MockSequencerEngineClient::new();
    client.expect_get_unsafe_head().times(1).returning(move || Ok(unsafe_head));
    client.expect_reset_engine_forkchoice().times(1).returning(|| Ok(()));

    let mut actor = test_actor();
    actor.conductor = Some(conductor);
    actor.engine_client = client;

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
