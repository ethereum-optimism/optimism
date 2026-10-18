use super::{
    super::{SequencerHandle, SequencerState, handle::Message},
    test_util::{install_engine, known_head, run_command, test_actor_with_handle},
};
use crate::{ConductorError, actors::MockConductor};
use alloy_primitives::B256;
use alloy_transport::RpcError;

use kona_rpc::{SequencerAdminAPIClient, SequencerAdminAPIError};
use rstest::rstest;
use tokio::sync::oneshot;

#[rstest]
#[case::unmanaged(true, false, true)]
#[case::managed(true, true, false)]
#[case::stopped(false, false, false)]
#[tokio::test]
async fn constructor_requires_explicit_managed_activation(
    #[case] configured_active: bool,
    #[case] managed: bool,
    #[case] expected: bool,
) {
    let (_actor, handle) =
        test_actor_with_handle(configured_active, true, managed.then(MockConductor::new));
    assert_eq!(handle.is_sequencer_active().await.unwrap(), expected);
    assert!(handle.is_recovery_mode().await.unwrap());
    assert_eq!(handle.is_conductor_enabled().await.unwrap(), managed);
}

#[tokio::test]
async fn start_validates_head_and_publishes_before_acknowledgement() {
    let (mut actor, handle) = test_actor_with_handle(false, false, None);
    assert!(matches!(run_command(&mut actor, handle.start_sequencer(B256::ZERO)).await,
        Err(SequencerAdminAPIError::NoPrestate(expected)) if expected == B256::ZERO));
    let engine = install_engine(&mut actor);
    engine.state.send_modify(|state| {
        state.sync_state = state.sync_state.apply_update(kona_engine::EngineSyncStateUpdate {
            unsafe_head: Some(kona_protocol::L2BlockInfo {
                block_info: kona_protocol::BlockInfo {
                    hash: B256::ZERO,
                    number: 99,
                    ..Default::default()
                },
                ..Default::default()
            }),
            ..Default::default()
        })
    });
    assert!(matches!(run_command(&mut actor, handle.start_sequencer(B256::ZERO)).await,
        Err(SequencerAdminAPIError::NoPrestate(expected)) if expected == B256::ZERO));
    engine.state.send_modify(|state| {
        state.sync_state = state.sync_state.apply_update(kona_engine::EngineSyncStateUpdate {
            unsafe_head: Some(known_head()),
            ..Default::default()
        })
    });
    assert!(matches!(run_command(&mut actor, handle.start_sequencer(B256::ZERO)).await,
        Err(SequencerAdminAPIError::HeadMismatch { head, received }) if head == known_head() && received == B256::ZERO));
    run_command(&mut actor, handle.start_sequencer(known_head().hash())).await.unwrap();
    assert!(handle.is_sequencer_active().await.unwrap());
    assert!(matches!(
        run_command(&mut actor, handle.start_sequencer(known_head().hash())).await,
        Err(SequencerAdminAPIError::AlreadyRunning)
    ));
}

#[rstest]
#[case::leader(true, false)]
#[case::follower(false, false)]
#[case::unavailable(false, true)]
#[tokio::test]
async fn start_checks_leadership_before_already_running(#[case] leader: bool, #[case] fails: bool) {
    let mut conductor = MockConductor::new();
    conductor.expect_leader().times(1).return_once(move || {
        if fails {
            Err(ConductorError::Rpc(RpcError::local_usage_str("unavailable")))
        } else {
            Ok(leader)
        }
    });
    let (mut actor, handle) = test_actor_with_handle(false, false, Some(conductor));
    actor.state.active = true;
    actor.publish();
    let result = run_command(&mut actor, handle.start_sequencer(B256::ZERO)).await;
    match (leader, fails) {
        (_, true) => assert!(matches!(result, Err(SequencerAdminAPIError::LeaderCheckFailed(_)))),
        (false, false) => assert!(matches!(result, Err(SequencerAdminAPIError::NotLeader))),
        (true, false) => assert!(matches!(result, Err(SequencerAdminAPIError::AlreadyRunning))),
    }
}

#[tokio::test]
async fn start_reads_the_head_after_the_leadership_query() {
    let (mut actor, handle) = test_actor_with_handle(false, false, None);
    let engine = install_engine(&mut actor);
    let mut new_head = known_head();
    new_head.block_info.hash = B256::repeat_byte(8);
    let mut conductor = MockConductor::new();
    conductor.expect_leader().times(1).return_once(move || {
        engine.state.send_modify(|state| {
            state.sync_state = state.sync_state.apply_update(kona_engine::EngineSyncStateUpdate {
                unsafe_head: Some(new_head),
                ..Default::default()
            })
        });
        Ok(true)
    });
    actor.conductor = Some(conductor);
    let result = run_command(&mut actor, handle.start_sequencer(known_head().hash())).await;
    assert!(
        matches!(result, Err(SequencerAdminAPIError::HeadMismatch { head, .. }) if head == new_head)
    );
    assert!(!handle.is_sequencer_active().await.unwrap());
}

#[tokio::test]
async fn stop_publishes_inactive_returns_head_and_restart_is_fresh() {
    let (mut actor, handle) = test_actor_with_handle(true, false, None);
    let _engine = install_engine(&mut actor);
    assert_eq!(
        run_command(&mut actor, handle.stop_sequencer()).await.unwrap(),
        known_head().hash()
    );
    assert!(!handle.is_sequencer_active().await.unwrap());
    assert!(matches!(
        run_command(&mut actor, handle.stop_sequencer()).await,
        Err(SequencerAdminAPIError::NotRunning)
    ));
    run_command(&mut actor, handle.start_sequencer(known_head().hash())).await.unwrap();
    assert!(handle.is_sequencer_active().await.unwrap());
}

#[rstest]
#[case::enable(false, true)]
#[case::disable(true, false)]
#[case::unchanged(true, true)]
#[tokio::test]
async fn recovery_mode_is_published_before_acknowledgement(
    #[case] initial: bool,
    #[case] mode: bool,
) {
    let (mut actor, handle) = test_actor_with_handle(true, initial, None);
    let ((), ()) = tokio::join!(
        async {
            handle.set_recovery_mode(mode).await.unwrap();
            assert_eq!(handle.is_recovery_mode().await.unwrap(), mode);
        },
        async {
            let command = actor.commands.recv().await.unwrap();
            actor.handle_command(command).await.unwrap();
        }
    );
}

#[rstest]
#[case::unmanaged(false)]
#[case::managed(true)]
#[tokio::test]
async fn override_is_local_idempotent_and_preserves_start_validation(#[case] configured: bool) {
    let conductor = configured.then(|| {
        let mut conductor = MockConductor::new();
        conductor.expect_leader().times(0);
        conductor.expect_commit_unsafe_payload().times(0);
        conductor
    });
    let (mut actor, handle) = test_actor_with_handle(false, false, conductor);
    run_command(&mut actor, handle.override_leader()).await.unwrap();
    run_command(&mut actor, handle.override_leader()).await.unwrap();
    assert!(!handle.is_sequencer_active().await.unwrap(), "override does not activate sequencing");
    assert_eq!(handle.is_conductor_enabled().await.unwrap(), configured);
    assert!(matches!(
        run_command(&mut actor, handle.start_sequencer(B256::ZERO)).await,
        Err(SequencerAdminAPIError::NoPrestate(_))
    ));
    let _engine = install_engine(&mut actor);
    assert!(matches!(
        run_command(&mut actor, handle.start_sequencer(B256::ZERO)).await,
        Err(SequencerAdminAPIError::HeadMismatch { .. })
    ));
    run_command(&mut actor, handle.start_sequencer(known_head().hash())).await.unwrap();
    assert!(handle.is_sequencer_active().await.unwrap());
    assert_eq!(handle.is_conductor_enabled().await.unwrap(), configured);
}

#[tokio::test]
async fn dropped_override_reply_preserves_local_permission_for_this_process_only() {
    let mut conductor = MockConductor::new();
    conductor.expect_leader().times(0);
    conductor.expect_commit_unsafe_payload().times(0);
    let (mut actor, handle) = test_actor_with_handle(false, false, Some(conductor));
    let _engine = install_engine(&mut actor);
    let (reply, result) = oneshot::channel();
    drop(result);
    actor.handle_command(Message::OverrideLeader(reply)).await.unwrap();
    run_command(&mut actor, handle.start_sequencer(known_head().hash())).await.unwrap();
    assert!(handle.is_sequencer_active().await.unwrap());
    assert!(handle.is_conductor_enabled().await.unwrap());
    let (fresh, handle) = test_actor_with_handle(true, false, Some(MockConductor::new()));
    assert!(!fresh.conductor_overridden);
    assert!(!handle.is_sequencer_active().await.unwrap());
}

#[tokio::test]
async fn reads_bypass_a_full_command_queue_and_reject_a_closed_publisher() {
    let initial = SequencerState { active: false, recovery_mode: false };
    let (handle, owner) = SequencerHandle::test_channel(initial, true);
    let commands = futures::future::join_all((0..32).map(|_| handle.start_sequencer(B256::ZERO)));
    tokio::pin!(commands);
    assert!(futures::poll!(commands.as_mut()).is_pending());
    assert_eq!(owner.commands.len(), 32);
    assert!(!handle.is_sequencer_active().await.unwrap());
    assert!(!handle.is_recovery_mode().await.unwrap());
    assert!(handle.is_conductor_enabled().await.unwrap());
    owner.state.send_replace(SequencerState { active: true, recovery_mode: true });
    assert!(handle.is_sequencer_active().await.unwrap());
    assert!(handle.is_recovery_mode().await.unwrap());
    // Close with an unread last publication, so merely borrowing stale state would be wrong.
    owner.state.send_replace(initial);
    drop(owner.state);
    assert!(matches!(
        handle.is_sequencer_active().await,
        Err(SequencerAdminAPIError::ResponseError)
    ));
    assert!(matches!(handle.is_recovery_mode().await, Err(SequencerAdminAPIError::ResponseError)));
    assert!(matches!(
        handle.is_conductor_enabled().await,
        Err(SequencerAdminAPIError::ResponseError)
    ));
}

#[tokio::test]
async fn closed_command_and_reply_channels_return_the_existing_errors() {
    let (handle, mut owner) = SequencerHandle::test_channel(
        SequencerState { active: false, recovery_mode: false },
        false,
    );
    let (result, ()) = tokio::join!(handle.start_sequencer(B256::ZERO), async {
        drop(owner.commands.recv().await.unwrap());
    });
    assert!(matches!(result, Err(SequencerAdminAPIError::ResponseError)));
    drop(owner.commands);
    assert!(matches!(
        handle.start_sequencer(B256::ZERO).await,
        Err(SequencerAdminAPIError::RequestError(_))
    ));
}

#[tokio::test]
async fn dropped_mutation_replies_do_not_cancel_accepted_commands() {
    let mut conductor = MockConductor::new();
    conductor.expect_leader().times(1).return_once(|| Ok(true));
    let (mut actor, handle) = test_actor_with_handle(false, false, Some(conductor));
    let _engine = install_engine(&mut actor);
    let (reply, result) = oneshot::channel();
    drop(result);
    actor.handle_command(Message::Start(known_head().hash(), reply)).await.unwrap();
    assert!(handle.is_sequencer_active().await.unwrap());
    let (reply, result) = oneshot::channel();
    drop(result);
    actor.handle_command(Message::Stop(reply)).await.unwrap();
    assert!(!handle.is_sequencer_active().await.unwrap());
    let (reply, result) = oneshot::channel();
    drop(result);
    actor.handle_command(Message::SetRecoveryMode(true, reply)).await.unwrap();
    assert!(handle.is_recovery_mode().await.unwrap());
    let (reply, result) = oneshot::channel();
    drop(result);
    actor.handle_command(Message::OverrideLeader(reply)).await.unwrap();
    assert!(handle.is_conductor_enabled().await.unwrap());
}

#[tokio::test]
async fn registered_start_requires_hash_and_preserves_domain_and_owner_errors() {
    use jsonrpsee::core::server::MethodsError;
    use kona_rpc::{SequencerAdminApiServer, SequencerAdminRpc};
    let (mut actor, handle) = test_actor_with_handle(false, false, None);
    let _engine = install_engine(&mut actor);
    let rpc = SequencerAdminRpc::new(handle).into_rpc();
    let error =
        rpc.call::<_, ()>("admin_startSequencer", jsonrpsee::rpc_params![]).await.unwrap_err();
    assert!(matches!(error, MethodsError::JsonRpc(error) if error.code() == -32602));
    assert!(actor.commands.is_empty(), "invalid parameters cannot enter the owner queue");
    let (result, ()) = tokio::join!(
        rpc.call::<_, ()>("admin_startSequencer", jsonrpsee::rpc_params![B256::ZERO]),
        async {
            let command = actor.commands.recv().await.unwrap();
            actor.handle_command(command).await.unwrap();
        }
    );
    let error = result.unwrap_err();
    assert!(
        matches!(error, MethodsError::JsonRpc(error) if error.code() == -32000 && error.message().starts_with("block hash does not match: head "))
    );
    drop(actor);
    let error = rpc
        .call::<_, ()>("admin_startSequencer", jsonrpsee::rpc_params![known_head().hash()])
        .await
        .unwrap_err();
    assert!(matches!(error, MethodsError::JsonRpc(error) if error.code() == -32603));
}
