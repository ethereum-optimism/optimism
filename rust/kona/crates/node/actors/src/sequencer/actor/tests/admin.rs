use super::{TestActor, test_actor_with_config};
use crate::{
    ConductorError, EngineClientError, MockConductor,
    sequencer::{Handle, HandleError, handle::Message},
};
use alloy_primitives::B256;
use alloy_transport::RpcError;
use kona_protocol::{BlockInfo, L2BlockInfo};
use rstest::rstest;
use tokio::{
    sync::oneshot,
    time::{self, Duration},
};
use tokio_util::sync::CancellationToken;

async fn command<T>(
    actor: &mut TestActor,
    build: impl FnOnce(oneshot::Sender<Result<T, HandleError>>) -> Message,
) -> Result<T, HandleError> {
    let (tx, rx) = oneshot::channel();
    time::timeout(Duration::from_secs(10), actor.handle_message(build(tx))).await.unwrap();
    rx.await.unwrap()
}

#[rstest]
#[tokio::test(start_paused = true)]
async fn start_sequencer(#[values(true, false)] already_started: bool) {
    let (mut actor, _) = test_actor_with_config(already_started, false, None);
    let state = actor.admin_state_receiver();
    assert_eq!(state.borrow().active, already_started);
    command(&mut actor, Message::StartSequencer).await.unwrap();
    assert!(state.borrow().active);
}

/// A queued stop wins over the initially ready build tick.
#[rstest]
#[tokio::test(start_paused = true)]
async fn stop_sequencer(#[values(true, false)] already_stopped: bool) {
    let (mut actor, commands) = test_actor_with_config(!already_stopped, false, None);
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
    actor.engine_client.expect_reset_engine_forkchoice().times(1).return_once(|| Ok(()));
    let (tx, rx) = oneshot::channel();
    commands.send(Message::StopSequencer(tx)).await.unwrap();
    let cancellation = CancellationToken::new();
    let task = tokio::spawn(actor.run(cancellation.clone()));
    assert_eq!(time::timeout(Duration::from_secs(10), rx).await.unwrap().unwrap().unwrap(), hash);
    assert!(!state.borrow().active);
    cancellation.cancel();
    task.await.unwrap().unwrap();
}

#[tokio::test(start_paused = true)]
async fn stop_is_published_before_a_failed_unsafe_head_read() {
    let (mut actor, _) = test_actor_with_config(true, false, None);
    let state = actor.admin_state_receiver();
    let observed_state = state.clone();
    actor.engine_client.expect_get_unsafe_head().times(1).return_once(move || {
        assert!(!observed_state.borrow().active);
        Err(EngineClientError::RequestError("whoops!".to_string()))
    });
    let error = command(&mut actor, Message::StopSequencer).await.unwrap_err();
    assert!(matches!(error, HandleError::ErrorAfterSequencerWasStopped(_)));
    assert!(!state.borrow().active);
}

#[rstest]
#[tokio::test(start_paused = true)]
async fn set_recovery_mode(
    #[values(true, false)] starting_mode: bool,
    #[values(true, false)] mode: bool,
) {
    let (mut actor, _) = test_actor_with_config(true, starting_mode, None);
    let state = actor.admin_state_receiver();
    assert_eq!(state.borrow().recovery_mode, starting_mode);
    command(&mut actor, |tx| Message::SetRecoveryMode(mode, tx)).await.unwrap();
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
    let (mut actor, _) = test_actor_with_config(true, false, conductor);
    let result = command(&mut actor, Message::OverrideLeader).await;
    if !configured || fail {
        let error = result.unwrap_err();
        assert!(matches!(error, HandleError::LeaderOverrideError(_)));
        let expected = if configured { "test conductor error" } else { "No conductor configured" };
        assert!(error.to_string().contains(expected));
    } else {
        result.unwrap();
    }
}

#[tokio::test(start_paused = true)]
async fn accepted_commands_run_when_response_receivers_are_dropped() {
    let mut conductor = MockConductor::new();
    conductor.expect_override_leader().times(1).return_once(|| Ok(()));
    let (mut actor, _) = test_actor_with_config(true, false, Some(conductor));
    actor
        .engine_client
        .expect_get_unsafe_head()
        .times(1)
        .return_once(|| Ok(L2BlockInfo::default()));
    let mut commands = Vec::new();
    let (tx, _) = oneshot::channel();
    commands.push(Message::StartSequencer(tx));
    let (tx, _) = oneshot::channel();
    commands.push(Message::StopSequencer(tx));
    let (tx, _) = oneshot::channel();
    commands.push(Message::SetRecoveryMode(true, tx));
    let (tx, _) = oneshot::channel();
    commands.push(Message::OverrideLeader(tx));
    let state = actor.admin_state_receiver();
    for command in commands {
        time::timeout(Duration::from_secs(10), actor.handle_message(command)).await.unwrap();
    }
    assert!(!state.borrow().active);
    assert!(state.borrow().recovery_mode);
}

#[tokio::test(start_paused = true)]
async fn handle_reads_published_state_after_commands() {
    let (mut actor, commands) = test_actor_with_config(false, false, Some(MockConductor::new()));
    let handle = Handle::new(actor.admin_state_receiver(), commands);
    let hash = B256::repeat_byte(42);
    actor.engine_client.expect_get_unsafe_head().times(1).return_once(move || {
        Ok(L2BlockInfo {
            block_info: BlockInfo { hash, ..Default::default() },
            ..Default::default()
        })
    });
    assert!(!handle.snapshot().unwrap().active);
    assert!(handle.snapshot().unwrap().conductor_enabled);
    assert!(!handle.snapshot().unwrap().recovery_mode);
    actor.engine_client.expect_reset_engine_forkchoice().times(1).return_once(|| Ok(()));
    actor.unsafe_payload_gossip_client.expect_has_capacity().return_const(false);
    let cancellation = CancellationToken::new();
    let task = tokio::spawn(actor.run(cancellation.clone()));
    handle.start().await.unwrap();
    assert!(handle.snapshot().unwrap().active);
    handle.set_recovery_mode(true).await.unwrap();
    assert!(handle.snapshot().unwrap().recovery_mode);
    assert_eq!(handle.stop().await.unwrap(), hash);
    assert!(!handle.snapshot().unwrap().active);
    cancellation.cancel();
    task.await.unwrap().unwrap();
}

#[tokio::test(start_paused = true)]
async fn state_updates_without_rpc_subscribers() {
    let (mut actor, _) = test_actor_with_config(true, false, None);
    assert_eq!(actor.published.receiver_count(), 0);
    command(&mut actor, |tx| Message::SetRecoveryMode(true, tx)).await.unwrap();
    let receiver = actor.admin_state_receiver();
    assert!(receiver.borrow().recovery_mode);
    drop(receiver);
    actor
        .engine_client
        .expect_get_unsafe_head()
        .times(1)
        .return_once(|| Ok(L2BlockInfo::default()));
    command(&mut actor, Message::StopSequencer).await.unwrap();
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
    commands.send(Message::StartSequencer(tx)).await.unwrap();
    let state = actor.admin_state_receiver();
    assert!(actor.run(CancellationToken::new()).await.is_err());
    assert!(!state.borrow().active);
    assert!(matches!(rx.try_recv(), Err(oneshot::error::TryRecvError::Closed)));
}
