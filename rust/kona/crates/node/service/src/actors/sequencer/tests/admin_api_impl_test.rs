use super::{
    super::{SequencerHandle, SequencerState, handle::Message},
    test_util::test_actor_with_handle,
};
use crate::{ConductorError, actors::MockConductor};
use alloy_primitives::B256;
use alloy_transport::RpcError;
use kona_protocol::{BlockInfo, L2BlockInfo};
use kona_rpc::{SequencerAdminAPIClient, SequencerAdminAPIError};
use rstest::rstest;
use tokio::sync::oneshot;

#[rstest]
#[case::running(true, false)]
#[case::recovering(false, true)]
#[tokio::test]
async fn constructor_publishes_configured_state(#[case] active: bool, #[case] recovery: bool) {
    let (_actor, handle) = test_actor_with_handle(active, recovery, Some(MockConductor::new()));
    assert_eq!(handle.is_sequencer_active().await.unwrap(), active);
    assert_eq!(handle.is_recovery_mode().await.unwrap(), recovery);
    assert!(handle.is_conductor_enabled().await.unwrap());
}

#[rstest]
#[case::already_running(true)]
#[case::stopped(false)]
#[tokio::test]
async fn start_publishes_active_before_acknowledgement(#[case] active: bool) {
    let (mut actor, handle) = test_actor_with_handle(active, false, None);
    let ((), ()) = tokio::join!(
        async {
            handle.start_sequencer().await.unwrap();
            assert!(handle.is_sequencer_active().await.unwrap());
        },
        async {
            let command = actor.commands.recv().await.unwrap();
            actor.handle_command(command).await;
        }
    );
    assert!(actor.state.active);
}

#[rstest]
#[case::running(true)]
#[case::already_stopped(false)]
#[tokio::test]
async fn stop_publishes_inactive_and_returns_current_engine_head(#[case] active: bool) {
    let head = L2BlockInfo {
        block_info: BlockInfo { hash: B256::repeat_byte(7), ..Default::default() },
        ..Default::default()
    };
    let mut state = kona_engine::EngineState::default();
    state.sync_state = state.sync_state.apply_update(kona_engine::EngineSyncStateUpdate {
        unsafe_head: Some(head),
        ..Default::default()
    });
    let (mut actor, handle) = test_actor_with_handle(active, false, None);
    actor.engine_client = crate::EngineHandle::test_channel(state).0;
    let ((), ()) = tokio::join!(
        async {
            assert_eq!(handle.stop_sequencer().await.unwrap(), head.hash());
            assert!(!handle.is_sequencer_active().await.unwrap());
        },
        async {
            let command = actor.commands.recv().await.unwrap();
            actor.handle_command(command).await;
        }
    );
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
            actor.handle_command(command).await;
        }
    );
}

#[rstest]
#[case::absent(false, false)]
#[case::success(true, false)]
#[case::dependency_error(true, true)]
#[tokio::test]
async fn override_leader_preserves_dependency_results(
    #[case] configured: bool,
    #[case] fails: bool,
) {
    let conductor = configured.then(|| {
        let mut conductor = MockConductor::new();
        conductor.expect_override_leader().times(1).return_once(move || {
            if fails {
                Err(ConductorError::Rpc(RpcError::local_usage_str("override unavailable")))
            } else {
                Ok(())
            }
        });
        conductor
    });
    let (mut actor, handle) = test_actor_with_handle(true, false, conductor);
    let (result, ()) = tokio::join!(handle.override_leader(), async {
        let command = actor.commands.recv().await.unwrap();
        actor.handle_command(command).await;
    });
    assert_eq!(result.is_ok(), configured && !fails);
    if let Err(error) = result {
        assert!(matches!(error, SequencerAdminAPIError::LeaderOverrideError(_)));
        assert!(error.to_string().contains(if configured {
            "override unavailable"
        } else {
            "No conductor configured"
        }));
    }
    assert_eq!(handle.is_conductor_enabled().await.unwrap(), configured);
}

#[tokio::test]
async fn reads_bypass_a_full_command_queue_and_reject_a_closed_publisher() {
    let initial = SequencerState { active: false, recovery_mode: false };
    let (handle, owner) = SequencerHandle::test_channel(initial, true);
    let commands = futures::future::join_all((0..32).map(|_| handle.start_sequencer()));
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
    let (result, ()) = tokio::join!(handle.start_sequencer(), async {
        drop(owner.commands.recv().await.unwrap());
    });
    assert!(matches!(result, Err(SequencerAdminAPIError::ResponseError)));
    drop(owner.commands);
    assert!(matches!(handle.start_sequencer().await, Err(SequencerAdminAPIError::RequestError(_))));
}

#[tokio::test]
async fn dropped_mutation_replies_do_not_cancel_accepted_commands() {
    let mut conductor = MockConductor::new();
    conductor.expect_override_leader().times(1).return_once(|| Ok(()));
    let (mut actor, handle) = test_actor_with_handle(false, false, Some(conductor));
    let (reply, result) = oneshot::channel();
    drop(result);
    actor.handle_command(Message::Start(reply)).await;
    assert!(handle.is_sequencer_active().await.unwrap());
    let (reply, result) = oneshot::channel();
    drop(result);
    actor.handle_command(Message::Stop(reply)).await;
    assert!(!handle.is_sequencer_active().await.unwrap());
    let (reply, result) = oneshot::channel();
    drop(result);
    actor.handle_command(Message::SetRecoveryMode(true, reply)).await;
    assert!(handle.is_recovery_mode().await.unwrap());
    let (reply, result) = oneshot::channel();
    drop(result);
    actor.handle_command(Message::OverrideLeader(reply)).await;
    assert!(handle.is_conductor_enabled().await.unwrap());
}
