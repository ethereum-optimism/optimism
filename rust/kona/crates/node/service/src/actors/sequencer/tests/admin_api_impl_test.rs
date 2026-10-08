use crate::{
    ConductorError, EngineClientError,
    actors::{
        MockConductor, MockSequencerEngineClient,
        sequencer::tests::test_util::{test_actor, test_actor_with_conductor},
    },
};
use alloy_primitives::B256;
use alloy_transport::RpcError;
use kona_protocol::{BlockInfo, L2BlockInfo};
use kona_rpc::{
    AdminApiServer, AdminRpc, SequencerAdminAPIError, SequencerAdminCommand, SequencerAdminHandle,
};
use rstest::rstest;
use tokio::sync::{mpsc, oneshot};

#[rstest]
#[tokio::test]
async fn test_start_sequencer(
    #[values(true, false)] already_started: bool,
    #[values(true, false)] via_channel: bool,
) {
    let mut actor = test_actor();
    actor.update_state(|state| state.active = already_started);

    let state = actor.admin_state_receiver();
    assert_eq!(state.borrow().active, already_started);

    // start the sequencer
    let result = async {
        match via_channel {
            false => actor.start_sequencer().await,
            true => {
                let (tx, rx) = oneshot::channel();
                actor.handle_admin_command(SequencerAdminCommand::StartSequencer(tx)).await;
                rx.await.unwrap()
            }
        }
    }
    .await;
    assert!(result.is_ok());

    assert!(state.borrow().active);
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
    actor.update_state(|state| state.active = !already_stopped);

    let state = actor.admin_state_receiver();
    assert_eq!(state.borrow().active, !already_stopped);

    // stop the sequencer
    let result = async {
        match via_channel {
            false => actor.stop_sequencer().await,
            true => {
                let (tx, rx) = oneshot::channel();
                actor.handle_admin_command(SequencerAdminCommand::StopSequencer(tx)).await;
                rx.await.unwrap()
            }
        }
    }
    .await;
    assert!(result.is_ok());
    assert_eq!(result.unwrap(), expected_hash);

    assert!(!state.borrow().active);
}

#[rstest]
#[tokio::test]
async fn test_stop_sequencer_error_fetching_unsafe_head(#[values(true, false)] via_channel: bool) {
    let mut actor = test_actor();
    let state = actor.admin_state_receiver();
    let mut client = MockSequencerEngineClient::new();
    client.expect_get_unsafe_head().times(1).return_once(move || {
        assert!(!state.borrow().active, "stop must publish before reading the unsafe head");
        Err(EngineClientError::RequestError("whoops!".to_string()))
    });

    actor.engine_client = client;

    let result = async {
        match via_channel {
            false => actor.stop_sequencer().await,
            true => {
                let (tx, rx) = oneshot::channel();
                actor.handle_admin_command(SequencerAdminCommand::StopSequencer(tx)).await;
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
    assert!(!actor.state().active);
}

#[rstest]
#[tokio::test]
async fn test_set_recovery_mode(
    #[values(true, false)] starting_mode: bool,
    #[values(true, false)] mode_to_set: bool,
    #[values(true, false)] via_channel: bool,
) {
    let mut actor = test_actor();
    actor.update_state(|state| state.recovery_mode = starting_mode);

    let state = actor.admin_state_receiver();
    assert_eq!(state.borrow().recovery_mode, starting_mode);

    // set recovery mode
    let result = async {
        match via_channel {
            false => actor.set_recovery_mode(mode_to_set).await,
            true => {
                let (tx, rx) = oneshot::channel();
                actor
                    .handle_admin_command(SequencerAdminCommand::SetRecoveryMode(mode_to_set, tx))
                    .await;
                rx.await.unwrap()
            }
        }
    }
    .await;
    assert!(result.is_ok());

    assert_eq!(state.borrow().recovery_mode, mode_to_set);
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
            test_actor_with_conductor(Some(conductor))
        } else {
            let mut conductor = MockConductor::new();
            conductor.expect_override_leader().times(1).return_once(|| Ok(()));
            test_actor_with_conductor(Some(conductor))
        }
    };

    // call to override leader
    let result = async {
        match via_channel {
            false => actor.override_leader().await,
            true => {
                let (tx, rx) = oneshot::channel();
                actor.handle_admin_command(SequencerAdminCommand::OverrideLeader(tx)).await;
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
                actor
                    .handle_admin_command(SequencerAdminCommand::ResetDerivationPipeline(tx))
                    .await;
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
                actor
                    .handle_admin_command(SequencerAdminCommand::ResetDerivationPipeline(tx))
                    .await;
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
async fn test_handle_admin_command_resilient_to_dropped_receiver() {
    let mut conductor = MockConductor::new();
    conductor.expect_override_leader().times(1).returning(|| Ok(()));

    let unsafe_head = L2BlockInfo {
        block_info: BlockInfo { hash: B256::from([1u8; 32]), ..Default::default() },
        ..Default::default()
    };
    let mut client = MockSequencerEngineClient::new();
    client.expect_get_unsafe_head().times(1).returning(move || Ok(unsafe_head));
    client.expect_reset_engine_forkchoice().times(1).returning(|| Ok(()));

    let mut actor = test_actor_with_conductor(Some(conductor));
    actor.engine_client = client;

    let mut commands: Vec<SequencerAdminCommand> = Vec::new();
    {
        // immediately drop receiver
        let (tx, _rx) = oneshot::channel();
        commands.push(SequencerAdminCommand::StartSequencer(tx));
    }
    {
        // immediately drop receiver
        let (tx, _rx) = oneshot::channel();
        commands.push(SequencerAdminCommand::StopSequencer(tx));
    }
    {
        // immediately drop receiver
        let (tx, _rx) = oneshot::channel();
        commands.push(SequencerAdminCommand::SetRecoveryMode(true, tx));
    }
    {
        // immediately drop receiver
        let (tx, _rx) = oneshot::channel();
        commands.push(SequencerAdminCommand::OverrideLeader(tx));
    }
    {
        // immediately drop receiver
        let (tx, _rx) = oneshot::channel();
        commands.push(SequencerAdminCommand::ResetDerivationPipeline(tx));
    }

    // None of these should fail even if the receiver is dropped
    for command in commands {
        actor.handle_admin_command(command).await;
    }
}

/// Status reads observe completed commands without queueing a second actor request.
#[tokio::test]
async fn rpc_reads_published_state_after_commands() {
    let mut actor = test_actor_with_conductor(Some(MockConductor::new()));
    actor.update_state(|state| state.active = false);
    let (commands_tx, commands_rx) = mpsc::channel(1);
    actor.admin_command_rx = commands_rx;
    let (payloads_tx, _payloads_rx) = mpsc::channel(1);
    let rpc = AdminRpc::new(
        Some(SequencerAdminHandle::new(actor.admin_state_receiver(), commands_tx)),
        payloads_tx,
    );
    let hash = B256::repeat_byte(42);
    actor.engine_client.expect_reset_engine_forkchoice().times(1).return_once(|| Ok(()));
    actor.engine_client.expect_get_unsafe_head().times(1).return_once(move || {
        Ok(L2BlockInfo {
            block_info: BlockInfo { hash, ..Default::default() },
            ..Default::default()
        })
    });

    let ((), ()) = tokio::join!(
        async {
            assert!(!rpc.admin_sequencer_active().await.unwrap());
            assert!(rpc.admin_conductor_enabled().await.unwrap());
            assert!(!rpc.admin_recover_mode().await.unwrap());
            rpc.admin_start_sequencer().await.unwrap();
            assert!(rpc.admin_sequencer_active().await.unwrap());
            rpc.admin_set_recover_mode(true).await.unwrap();
            assert!(rpc.admin_recover_mode().await.unwrap());
            assert_eq!(rpc.admin_stop_sequencer().await.unwrap(), hash);
            assert!(!rpc.admin_sequencer_active().await.unwrap());
        },
        async {
            for _ in 0..3 {
                crate::NodeActor::step(&mut actor).await.unwrap();
            }
        }
    );
}

/// State remains authoritative even when admin RPC is disabled or has no subscribers.
#[tokio::test]
async fn state_updates_without_rpc_subscribers() {
    let mut actor = test_actor();
    actor.set_recovery_mode(true).await.unwrap();
    assert!(actor.state().recovery_mode);
    let receiver = actor.admin_state_receiver();
    assert!(receiver.borrow().recovery_mode);
    drop(receiver);
    let mut engine = MockSequencerEngineClient::new();
    engine.expect_get_unsafe_head().times(1).return_once(|| Ok(L2BlockInfo::default()));
    actor.engine_client = engine;
    actor.stop_sequencer().await.unwrap();
    assert!(!actor.state().active);
    assert!(!actor.admin_state_receiver().borrow().active);
}
