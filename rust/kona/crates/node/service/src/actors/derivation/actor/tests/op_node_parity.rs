//! op-node parity for pipeline reset failures. op-node's `DerivationPipeline.Step` reruns
//! `initialReset` for the pending safe head. Its deriver turns a temporary error into
//! `EngineTemporaryErrorEvent`, which the step scheduler retries after backoff, and a reset
//! error into `ResetEvent`, which starts a new engine reset. Only `ErrCritical` stops the node.

use super::*;

fn engine_reset(head: u64) -> DerivationActorRequest {
    DerivationActorRequest::ProcessEngineSignalRequest(Box::new(Signal::Reset(
        kona_derive::ResetSignal { l2_safe_head: l2_block(head) },
    )))
}

fn l1_head_update(number: u64) -> DerivationActorRequest {
    DerivationActorRequest::ProcessL1HeadUpdateRequest(Box::new(l1_block(number)))
}

fn el_sync_completed(head: u64) -> DerivationActorRequest {
    engine_sync_completed_at(l2_block(head))
}

/// A reset whose pipeline signal fails once with a temporary provider error neither stops the
/// actor nor lets derivation resume, whichever actor requested the reset. A request that would
/// otherwise start deriving holds until the scheduled retry adopts the reset head.
#[rstest]
#[case::l1_head_while_awaiting_l1_data(Some(DERIVED_HEAD), l1_head_update(DERIVED_FROM_L1))]
#[case::el_sync_completion_at_startup(None, el_sync_completed(DERIVED_HEAD))]
#[tokio::test(start_paused = true)]
async fn temporary_reset_failure_holds_derivation_until_the_retry_like_op_node(
    #[case] el_sync_completed_before_reset_at: Option<u64>,
    #[case] request_during_backoff: DerivationActorRequest,
) {
    let mut engine_client = MockDerivationEngineClient::new();
    engine_client.expect_send_safe_l2_signal().times(1).returning(|_| Ok(()));
    engine_client.expect_send_finalized_l2_block().times(0);
    let mut node = RecordingActor::new(engine_client);
    node.actor.pipeline.attributes.clear();
    if let Some(head) = el_sync_completed_before_reset_at {
        node.complete_el_sync_at(head).await;
    }
    node.actor.pipeline.attributes.push_back(derived_attributes());
    node.actor.pipeline.signal_error = Some(l2_provider_error().temp());
    let previous_state = node.actor.derivation_state_machine.current_state();
    let previous_head = node.actor.derivation_state_machine.last_confirmed_safe_head();

    node.handle(engine_reset(RESUME_HEAD)).await;

    assert_eq!(node.actor.derivation_state_machine.current_state(), previous_state);
    assert_eq!(node.actor.derivation_state_machine.last_confirmed_safe_head(), previous_head);
    assert!(node.actor.attributes_awaiting_confirmation.is_none());
    assert!(matches!(node.db.last_entry(), Err(SafeDbError::NotFound)));

    node.handle(request_during_backoff).await;

    assert_eq!(
        node.actor.derivation_state_machine.current_state(),
        DerivationState::AwaitingL1Data
    );
    assert_eq!(node.actor.pipeline.attributes.len(), 1, "the pipeline must not be stepped");
    assert!(node.actor.attributes_awaiting_confirmation.is_none());

    // No request is queued, so the next step is the scheduled retry.
    node.actor.step().await.expect("the retried reset must succeed");

    assert_eq!(
        node.actor.derivation_state_machine.last_confirmed_safe_head(),
        l2_block(RESUME_HEAD)
    );
    assert!(node.actor.attributes_awaiting_confirmation.is_some());
}

/// A reset head the pipeline can no longer walk back from, because the L2 chain reorged under
/// it, asks the engine for a fresh head rather than stopping the actor. Derivation holds until
/// the fresh reset succeeds.
#[tokio::test]
async fn reorged_reset_head_requests_a_fresh_engine_reset_like_op_node() {
    let mut engine_client = MockDerivationEngineClient::new();
    engine_client.expect_reset_engine_forkchoice().times(1).returning(|| Ok(()));
    engine_client.expect_send_safe_l2_signal().times(1).returning(|_| Ok(()));
    engine_client.expect_send_finalized_l2_block().times(0);
    let mut node = RecordingActor::new(engine_client);
    node.actor.pipeline.attributes.clear();
    node.complete_el_sync_at(DERIVED_HEAD).await;
    node.actor.pipeline.attributes.push_back(derived_attributes());
    let reorged_parent = l2_block(RESUME_HEAD).block_info.parent_hash;
    node.actor.pipeline.signal_error =
        Some(ResetError::BlockNotFound(reorged_parent.into()).reset());

    node.handle(engine_reset(RESUME_HEAD)).await;
    node.handle(l1_head_update(DERIVED_FROM_L1)).await;

    assert_eq!(
        node.actor.derivation_state_machine.current_state(),
        DerivationState::AwaitingL1Data
    );
    assert_eq!(node.actor.pipeline.attributes.len(), 1, "the pipeline must not be stepped");
    assert!(node.actor.attributes_awaiting_confirmation.is_none());

    node.handle(engine_reset(RESUME_HEAD - 1)).await;

    assert_eq!(
        node.actor.derivation_state_machine.last_confirmed_safe_head(),
        l2_block(RESUME_HEAD - 1)
    );
    assert!(node.actor.attributes_awaiting_confirmation.is_some());
}
