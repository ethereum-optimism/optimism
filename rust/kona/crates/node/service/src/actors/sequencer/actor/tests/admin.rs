use super::{TestSequencerActor, test_actor_with_config};
use crate::{ConductorError, EngineClientError, NodeActor, actors::MockConductor};
use alloy_primitives::B256;
use alloy_transport::RpcError;
use kona_protocol::{BlockInfo, L2BlockInfo};
use kona_rpc::{
    AdminApiServer, AdminRpc, SequencerAdminAPIError, SequencerAdminCommand, SequencerAdminHandle,
};
use rstest::rstest;
use tokio::{
    sync::{mpsc, oneshot},
    time::{self, Duration},
};

#[tokio::test]
async fn stalled_leader_check_times_out_without_blocking_queued_stop() {
    use crate::{ConductorClient, SequencerActor};
    use jsonrpsee::{RpcModule, server::ServerBuilder};
    use std::sync::Arc;
    use tokio::sync::Notify;

    let entered = Arc::new(Notify::new());
    let release = Arc::new(Notify::new());
    let server = ServerBuilder::default().build("127.0.0.1:0").await.unwrap();
    let url = format!("http://{}", server.local_addr().unwrap()).parse().unwrap();
    let mut module = RpcModule::new((entered.clone(), release.clone()));
    module
        .register_async_method("conductor_leader", |_, state, _| async move {
            state.0.notify_one();
            state.1.notified().await;
            true
        })
        .unwrap();
    let server = server.start(module);
    let client = ConductorClient::new_http(url).with_timeout(Duration::from_millis(200));
    let (template, _) = test_actor_with_config(false, false, None);
    let (commands, rx) = mpsc::channel(2);
    let mut actor = SequencerActor::new(
        rx,
        template.attributes_builder,
        Some(client),
        template.engine_client,
        false,
        false,
        template.origin_selector,
        template.rollup_config,
        template.unsafe_payload_gossip_client,
        template.clock,
    );
    let head = B256::repeat_byte(42);
    actor.engine_client.expect_reset_engine_forkchoice().times(1).return_once(|| Ok(()));
    // Only stop may read the head: the failed leadership check cannot activate.
    actor.engine_client.expect_get_unsafe_head().times(1).return_once(move || {
        Ok(L2BlockInfo {
            block_info: BlockInfo { hash: head, ..Default::default() },
            ..Default::default()
        })
    });
    let (reply, started) = oneshot::channel();
    commands.send(SequencerAdminCommand::StartSequencer(head, reply)).await.unwrap();
    let (reply, stopped) = oneshot::channel();
    commands.send(SequencerAdminCommand::StopSequencer(reply)).await.unwrap();
    {
        let step = actor.step();
        tokio::pin!(step);
        tokio::select! {
            _ = entered.notified() => {},
            result = &mut step => panic!("start returned before the stalled leader request: {result:?}"),
        }
        time::timeout(Duration::from_secs(5), step).await.unwrap().unwrap();
    }
    assert!(started.await.unwrap().unwrap_err().to_string().contains("timed out"));
    assert!(!actor.admin_state_receiver().borrow().active);
    time::timeout(Duration::from_secs(5), actor.step()).await.unwrap().unwrap();
    assert_eq!(stopped.await.unwrap().unwrap(), head);
    assert!(!actor.admin_state_receiver().borrow().active);
    release.notify_waiters();
    server.stop().unwrap();
    server.stopped().await;
}

fn admin_actor(
    active: bool,
    recovery_mode: bool,
    conductor: Option<MockConductor>,
) -> (TestSequencerActor, mpsc::Sender<SequencerAdminCommand>) {
    let (mut actor, commands) = test_actor_with_config(active, recovery_mode, conductor);
    actor.engine_client.expect_reset_engine_forkchoice().times(1).return_once(|| Ok(()));
    (actor, commands)
}

async fn command<T>(
    actor: &mut TestSequencerActor,
    commands: &mpsc::Sender<SequencerAdminCommand>,
    build: impl FnOnce(oneshot::Sender<Result<T, SequencerAdminAPIError>>) -> SequencerAdminCommand,
) -> Result<T, SequencerAdminAPIError> {
    let (tx, rx) = oneshot::channel();
    commands.send(build(tx)).await.unwrap();
    time::timeout(Duration::from_secs(10), actor.step()).await.unwrap().unwrap();
    rx.await.unwrap()
}

#[rstest]
#[case::standalone(false)]
#[case::leader(true)]
#[tokio::test(start_paused = true)]
async fn start_sequencer(#[case] managed: bool) {
    let conductor = managed.then(|| {
        let mut conductor = MockConductor::new();
        conductor.expect_leader().times(1).return_once(|| Ok(true));
        conductor
    });
    let (mut actor, commands) = admin_actor(false, false, conductor);
    let hash = B256::repeat_byte(42);
    actor.engine_client.expect_get_unsafe_head().times(1).return_once(move || {
        Ok(L2BlockInfo {
            block_info: BlockInfo { hash, ..Default::default() },
            ..Default::default()
        })
    });
    let state = actor.admin_state_receiver();
    command(&mut actor, &commands, |reply| SequencerAdminCommand::StartSequencer(hash, reply))
        .await
        .unwrap();
    assert!(state.borrow().active);
}

#[rstest]
#[case::follower(false, "sequencer is not the leader")]
#[case::query_error(true, "sequencer leader check failed")]
#[tokio::test(start_paused = true)]
async fn start_requires_leadership(#[case] fails: bool, #[case] message: &str) {
    let mut conductor = MockConductor::new();
    conductor.expect_leader().times(1).return_once(move || {
        if fails {
            Err(ConductorError::Rpc(RpcError::local_usage_str("leader unavailable")))
        } else {
            Ok(false)
        }
    });
    let (mut actor, commands) = admin_actor(false, false, Some(conductor));
    actor.engine_client.expect_get_unsafe_head().times(0);
    let error = command(&mut actor, &commands, |reply| {
        SequencerAdminCommand::StartSequencer(B256::ZERO, reply)
    })
    .await
    .unwrap_err();
    assert!(error.to_string().contains(message));
    if fails {
        assert!(error.to_string().contains("leader unavailable"));
    }
    assert!(!actor.admin_state_receiver().borrow().active);
}

#[rstest]
#[case::unknown_head(B256::ZERO, false, "no prestate")]
#[case::wrong_head(B256::repeat_byte(1), false, "block hash does not match")]
#[case::head_query_error(B256::repeat_byte(42), true, "failed to read unsafe head")]
#[tokio::test(start_paused = true)]
async fn start_requires_expected_unsafe_head(
    #[case] actual: B256,
    #[case] fails: bool,
    #[case] message: &str,
) {
    let (mut actor, commands) = admin_actor(false, false, None);
    actor.engine_client.expect_get_unsafe_head().times(1).return_once(move || {
        if fails {
            return Err(EngineClientError::RequestError("head unavailable".to_owned()));
        }
        Ok(L2BlockInfo {
            block_info: BlockInfo { hash: actual, ..Default::default() },
            ..Default::default()
        })
    });
    let error = command(&mut actor, &commands, |reply| {
        SequencerAdminCommand::StartSequencer(B256::repeat_byte(42), reply)
    })
    .await
    .unwrap_err();
    assert!(error.to_string().contains(message));
    assert!(!actor.admin_state_receiver().borrow().active);
}

#[tokio::test(start_paused = true)]
async fn start_rejects_an_already_active_sequencer() {
    let (mut actor, commands) = admin_actor(true, false, None);
    actor.engine_client.expect_get_unsafe_head().times(0);
    let error = command(&mut actor, &commands, |reply| {
        SequencerAdminCommand::StartSequencer(B256::ZERO, reply)
    })
    .await
    .unwrap_err();
    assert!(error.to_string().contains("sequencer already running"));
    assert!(actor.admin_state_receiver().borrow().active);
}

/// A queued stop wins over the initially ready build tick.
#[rstest]
#[tokio::test(start_paused = true)]
async fn stop_sequencer(#[values(true, false)] already_stopped: bool) {
    let (mut actor, commands) = admin_actor(!already_stopped, false, None);
    let hash = B256::repeat_byte(1);
    actor.engine_client.expect_get_unsafe_head().times(1).return_once(move || {
        Ok(L2BlockInfo {
            block_info: BlockInfo { hash, ..Default::default() },
            ..Default::default()
        })
    });
    // Block building must not run while the stop is queued.
    actor.unsafe_payload_gossip_client.expect_has_capacity().times(0);
    let state = actor.admin_state_receiver();
    assert_eq!(state.borrow().active, !already_stopped);
    assert_eq!(
        command(&mut actor, &commands, SequencerAdminCommand::StopSequencer).await.unwrap(),
        hash
    );
    assert!(!state.borrow().active);
}

#[tokio::test(start_paused = true)]
async fn stop_is_published_before_a_failed_unsafe_head_read() {
    let (mut actor, commands) = admin_actor(true, false, None);
    let state = actor.admin_state_receiver();
    let observed_state = state.clone();
    actor.engine_client.expect_get_unsafe_head().times(1).return_once(move || {
        assert!(!observed_state.borrow().active);
        Err(EngineClientError::RequestError("whoops!".to_string()))
    });
    let error =
        command(&mut actor, &commands, SequencerAdminCommand::StopSequencer).await.unwrap_err();
    assert!(matches!(error, SequencerAdminAPIError::ErrorAfterSequencerWasStopped(_)));
    assert!(!state.borrow().active);
}

#[rstest]
#[tokio::test(start_paused = true)]
async fn set_recovery_mode(
    #[values(true, false)] starting_mode: bool,
    #[values(true, false)] mode: bool,
) {
    let (mut actor, commands) = admin_actor(true, starting_mode, None);
    let state = actor.admin_state_receiver();
    assert_eq!(state.borrow().recovery_mode, starting_mode);
    command(&mut actor, &commands, |tx| SequencerAdminCommand::SetRecoveryMode(mode, tx))
        .await
        .unwrap();
    assert_eq!(state.borrow().recovery_mode, mode);
}

#[rstest]
#[case::no_conductor(false, false)]
#[case::overridden(true, false)]
#[case::conductor_error(true, true)]
#[tokio::test(start_paused = true)]
async fn override_leader(#[case] configured: bool, #[case] fail: bool) {
    let conductor = configured.then(|| {
        let mut conductor = MockConductor::new();
        conductor.expect_override_leader().times(1).return_once(move || {
            if fail {
                Err(ConductorError::Rpc(RpcError::local_usage_str("test conductor error")))
            } else {
                Ok(())
            }
        });
        conductor
    });
    let (mut actor, commands) = admin_actor(true, false, conductor);
    let result = command(&mut actor, &commands, SequencerAdminCommand::OverrideLeader).await;
    if !configured || fail {
        let error = result.unwrap_err();
        assert!(matches!(error, SequencerAdminAPIError::LeaderOverrideError(_)));
        let expected = if configured { "test conductor error" } else { "No conductor configured" };
        assert!(error.to_string().contains(expected));
    } else {
        result.unwrap();
    }
}

#[tokio::test(start_paused = true)]
async fn accepted_commands_run_when_response_receivers_are_dropped() {
    let mut conductor = MockConductor::new();
    conductor.expect_leader().times(1).return_once(|| Ok(true));
    conductor.expect_override_leader().times(1).return_once(|| Ok(()));
    let (mut actor, commands_tx) = test_actor_with_config(true, false, Some(conductor));
    actor.engine_client.expect_reset_engine_forkchoice().times(1).returning(|| Ok(()));
    actor
        .engine_client
        .expect_get_unsafe_head()
        .times(1)
        .return_once(|| Ok(L2BlockInfo::default()));
    let mut commands = Vec::new();
    let (tx, _) = oneshot::channel();
    commands.push(SequencerAdminCommand::StartSequencer(B256::ZERO, tx));
    let (tx, _) = oneshot::channel();
    commands.push(SequencerAdminCommand::StopSequencer(tx));
    let (tx, _) = oneshot::channel();
    commands.push(SequencerAdminCommand::SetRecoveryMode(true, tx));
    let (tx, _) = oneshot::channel();
    commands.push(SequencerAdminCommand::OverrideLeader(tx));
    let state = actor.admin_state_receiver();
    for command in commands {
        commands_tx.send(command).await.unwrap();
        time::timeout(Duration::from_secs(10), actor.step()).await.unwrap().unwrap();
    }
    assert!(!state.borrow().active);
    assert!(state.borrow().recovery_mode);
}

#[tokio::test(start_paused = true)]
async fn rpc_reads_published_state_after_commands() {
    let mut conductor = MockConductor::new();
    conductor.expect_leader().times(1).return_once(|| Ok(true));
    let (mut actor, commands) = admin_actor(false, false, Some(conductor));
    let (payloads_tx, _) = mpsc::channel(1);
    let rpc = AdminRpc::new(
        Some(SequencerAdminHandle::new(actor.admin_state_receiver(), commands)),
        mpsc::channel(1).0,
        payloads_tx,
    );
    let hash = B256::repeat_byte(42);
    actor.engine_client.expect_get_unsafe_head().times(2).returning(move || {
        Ok(L2BlockInfo {
            block_info: BlockInfo { hash, ..Default::default() },
            ..Default::default()
        })
    });
    assert!(!rpc.admin_sequencer_active().await.unwrap());
    assert!(rpc.admin_conductor_enabled().await.unwrap());
    assert!(!rpc.admin_recover_mode().await.unwrap());
    let (result, step) = tokio::join!(biased; rpc.admin_start_sequencer(hash), actor.step());
    step.unwrap();
    result.unwrap();
    assert!(rpc.admin_sequencer_active().await.unwrap());
    let (result, step) = tokio::join!(biased; rpc.admin_set_recover_mode(true), actor.step());
    step.unwrap();
    result.unwrap();
    assert!(rpc.admin_recover_mode().await.unwrap());
    let (result, step) = tokio::join!(biased; rpc.admin_stop_sequencer(), actor.step());
    step.unwrap();
    assert_eq!(result.unwrap(), hash);
    assert!(!rpc.admin_sequencer_active().await.unwrap());
}

#[tokio::test(start_paused = true)]
async fn state_updates_without_rpc_subscribers() {
    let (mut actor, commands) = admin_actor(true, false, None);
    assert_eq!(actor.state.receiver_count(), 0);
    command(&mut actor, &commands, |tx| SequencerAdminCommand::SetRecoveryMode(true, tx))
        .await
        .unwrap();
    let receiver = actor.admin_state_receiver();
    assert!(receiver.borrow().recovery_mode);
    drop(receiver);
    actor
        .engine_client
        .expect_get_unsafe_head()
        .times(1)
        .return_once(|| Ok(L2BlockInfo::default()));
    command(&mut actor, &commands, SequencerAdminCommand::StopSequencer).await.unwrap();
    assert!(!actor.admin_state_receiver().borrow().active);
}

#[tokio::test(start_paused = true)]
async fn failed_startup_does_not_apply_a_queued_command() {
    let (mut actor, commands) = test_actor_with_config(false, false, None);
    actor
        .engine_client
        .expect_reset_engine_forkchoice()
        .times(1)
        .return_once(|| Err(EngineClientError::RequestError("startup reset failed".to_string())));
    let (tx, mut rx) = oneshot::channel();
    commands.send(SequencerAdminCommand::StartSequencer(B256::ZERO, tx)).await.unwrap();
    assert!(actor.step().await.is_err());
    assert!(!actor.admin_state_receiver().borrow().active);
    assert!(matches!(rx.try_recv(), Err(oneshot::error::TryRecvError::Empty)));
}
