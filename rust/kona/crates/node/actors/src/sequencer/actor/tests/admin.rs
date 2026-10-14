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
    let (mut actor, _, handle) = test_actor_with_config(already_started, None);
    actor.engine_client.expect_get_unsafe_head().returning(|| Ok(L2BlockInfo::default()));
    assert_eq!(handle.is_active().unwrap(), already_started);
    command(&mut actor, |tx| Message::StartSequencer(B256::ZERO, tx)).await.unwrap();
    assert!(handle.is_active().unwrap());
}

/// A queued stop wins over the initially ready build tick.
#[rstest]
#[tokio::test(start_paused = true)]
async fn stop_sequencer(#[values(true, false)] already_stopped: bool) {
    let (mut actor, commands, handle) = test_actor_with_config(!already_stopped, None);
    let hash = B256::repeat_byte(1);
    actor.engine_client.expect_get_unsafe_head().times(1).return_once(move || {
        Ok(L2BlockInfo {
            block_info: BlockInfo { hash, ..Default::default() },
            ..Default::default()
        })
    });
    // Block building must not run while the stop is queued.
    actor.signer.expect_ready().times(0);
    assert_eq!(handle.is_active().unwrap(), !already_stopped);
    actor.engine_client.expect_reset_engine_forkchoice().times(1).return_once(|| Ok(()));
    let (tx, rx) = oneshot::channel();
    commands.send(Message::StopSequencer(tx)).await.unwrap();
    let task = tokio::spawn(actor.run());
    assert_eq!(time::timeout(Duration::from_secs(10), rx).await.unwrap().unwrap().unwrap(), hash);
    assert!(!handle.is_active().unwrap());
    task.abort();
    assert!(task.await.unwrap_err().is_cancelled());
}

/// Behind op-conductor, the sequencer starts only while its conductor reports leadership, and on
/// its own unsafe head only, as op-node.
#[rstest]
#[case::leader(Ok(true), B256::ZERO, None)]
#[case::not_leader(Ok(false), B256::ZERO, Some("sequencer is not the leader"))]
#[case::leader_check_failed(
    Err(ConductorError::Rpc(RpcError::local_usage_str("conductor down"))),
    B256::ZERO,
    Some("sequencer leader check failed")
)]
#[case::other_head(Ok(true), B256::repeat_byte(1), Some("block hash does not match"))]
#[tokio::test(start_paused = true)]
async fn start_checks_leadership_and_the_unsafe_head(
    #[case] leader: Result<bool, ConductorError>,
    #[case] unsafe_head: B256,
    #[case] refusal: Option<&str>,
) {
    let mut conductor = MockConductor::new();
    conductor.expect_leader().times(1).return_once(move || leader);
    let (mut actor, _, handle) = test_actor_with_config(false, Some(conductor));
    actor.engine_client.expect_get_unsafe_head().returning(|| Ok(L2BlockInfo::default()));

    let result = command(&mut actor, |tx| Message::StartSequencer(unsafe_head, tx)).await;
    match refusal {
        None => result.unwrap(),
        Some(refusal) => assert!(result.unwrap_err().to_string().contains(refusal)),
    }
    assert_eq!(handle.is_active().unwrap(), refusal.is_none());
}

#[tokio::test(start_paused = true)]
async fn stop_is_published_before_a_failed_unsafe_head_read() {
    let (mut actor, _, handle) = test_actor_with_config(true, None);
    let observed_state = handle.clone();
    actor.engine_client.expect_get_unsafe_head().times(1).return_once(move || {
        assert!(!observed_state.is_active().unwrap());
        Err(EngineClientError::RequestError("whoops!".to_string()))
    });
    let error = command(&mut actor, Message::StopSequencer).await.unwrap_err();
    assert!(matches!(error, HandleError::ErrorAfterSequencerWasStopped(_)));
    assert!(!handle.is_active().unwrap());
}

/// Once an operator overrides the conductor's leadership, for disaster recovery, the sequencer
/// starts without asking its conductor whether it leads, as op-node.
#[tokio::test(start_paused = true)]
async fn an_overridden_leader_starts_without_its_conductor() {
    let mut conductor = MockConductor::new();
    conductor.expect_leader().times(0);
    let (mut actor, _, handle) = test_actor_with_config(false, Some(conductor));
    actor.engine_client.expect_get_unsafe_head().returning(|| Ok(L2BlockInfo::default()));

    command(&mut actor, Message::OverrideLeader).await.unwrap();
    command(&mut actor, |tx| Message::StartSequencer(B256::ZERO, tx)).await.unwrap();
    assert!(handle.is_active().unwrap());
}

#[tokio::test(start_paused = true)]
async fn overriding_the_leader_needs_a_conductor() {
    let (mut actor, _, _) = test_actor_with_config(true, None);
    let error = command(&mut actor, Message::OverrideLeader).await.unwrap_err();
    assert!(matches!(error, HandleError::LeaderOverrideError(_)));
    assert!(error.to_string().contains("No conductor configured"));
}

#[tokio::test(start_paused = true)]
async fn accepted_commands_run_when_response_receivers_are_dropped() {
    let (mut actor, _, handle) = test_actor_with_config(true, Some(MockConductor::new()));
    actor
        .engine_client
        .expect_get_unsafe_head()
        .times(1)
        .return_once(|| Ok(L2BlockInfo::default()));
    let mut commands = Vec::new();
    let (tx, _) = oneshot::channel();
    commands.push(Message::StartSequencer(B256::ZERO, tx));
    let (tx, _) = oneshot::channel();
    commands.push(Message::StopSequencer(tx));
    let (tx, _) = oneshot::channel();
    commands.push(Message::OverrideLeader(tx));
    for command in commands {
        time::timeout(Duration::from_secs(10), actor.handle_message(command)).await.unwrap();
    }
    assert!(!handle.is_active().unwrap());
}

#[tokio::test(start_paused = true)]
async fn handle_reads_published_state_after_commands() {
    let mut conductor = MockConductor::new();
    conductor.expect_leader().times(1).return_once(|| Ok(true));
    let (mut actor, _, handle) = test_actor_with_config(false, Some(conductor));
    let hash = B256::repeat_byte(42);
    actor.engine_client.expect_get_unsafe_head().times(2).returning(move || {
        Ok(L2BlockInfo {
            block_info: BlockInfo { hash, ..Default::default() },
            ..Default::default()
        })
    });
    assert!(!handle.is_active().unwrap());
    actor.engine_client.expect_reset_engine_forkchoice().times(1).return_once(|| Ok(()));
    // The signer never has room, so no block is built.
    let (signer, _queued) = tokio::sync::mpsc::channel(1);
    signer.try_send(crate::test_payload(0)).unwrap();
    let actor = super::with_signer(actor, super::QueueSigner(signer));
    let task = tokio::spawn(actor.run());
    handle.start(hash).await.unwrap();
    assert!(handle.is_active().unwrap());
    assert_eq!(handle.stop().await.unwrap(), hash);
    assert!(!handle.is_active().unwrap());
    task.abort();
    assert!(task.await.unwrap_err().is_cancelled());
}

#[tokio::test(start_paused = true)]
async fn state_updates_without_rpc_subscribers() {
    let (mut actor, commands, handle) = test_actor_with_config(true, None);
    drop(handle);
    assert_eq!(actor.is_active_tx.receiver_count(), 0);
    actor.engine_client.expect_get_unsafe_head().times(2).returning(|| Ok(L2BlockInfo::default()));
    command(&mut actor, Message::StopSequencer).await.unwrap();
    let handle = Handle::new(actor.is_active_tx.subscribe(), commands.clone());
    assert!(!handle.is_active().unwrap());
    drop(handle);
    command(&mut actor, |tx| Message::StartSequencer(B256::ZERO, tx)).await.unwrap();
    let handle = Handle::new(actor.is_active_tx.subscribe(), commands);
    assert!(handle.is_active().unwrap());
}

#[tokio::test(start_paused = true)]
async fn failed_startup_does_not_apply_a_queued_command() {
    let (mut actor, commands, handle) = test_actor_with_config(false, None);
    let observed_state = handle.clone();
    actor.engine_client.expect_reset_engine_forkchoice().times(1).return_once(move || {
        assert!(!observed_state.is_active().unwrap());
        Err(EngineClientError::RequestError("startup reset failed".to_string()))
    });
    let (tx, mut rx) = oneshot::channel();
    commands.send(Message::StartSequencer(B256::ZERO, tx)).await.unwrap();
    assert!(actor.run().await.is_err());
    assert!(matches!(handle.is_active(), Err(HandleError::RequestError(_))));
    assert!(matches!(rx.try_recv(), Err(oneshot::error::TryRecvError::Closed)));
}

/// Only the series for the sequencer's current state reads 1, so a dashboard of the gauge shows the
/// state after it changes.
#[test]
fn the_state_gauge_reads_the_current_state_only() {
    let recorder = metrics_exporter_prometheus::PrometheusBuilder::new().build_recorder();
    let metrics = recorder.handle();
    let (mut actor, _, _) = test_actor_with_config(true, None);
    metrics::with_local_recorder(&recorder, || {
        actor.update_state();
        actor.is_active = false;
        actor.update_state();
    });
    let rendered = metrics.render();
    assert!(rendered.contains("kona_node_sequencer_state{active=\"true\"} 0"), "{rendered}");
    assert!(rendered.contains("kona_node_sequencer_state{active=\"false\"} 1"), "{rendered}");
}
