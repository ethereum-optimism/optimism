//! op-node parity for pipeline reset failures. op-node's `DerivationPipeline.Step` reruns
//! `initialReset` for the pending safe head, its deriver turns a temporary error into
//! `EngineTemporaryErrorEvent`, and the step scheduler retries after backoff. Only
//! `ErrCritical` stops the node.

use super::*;

/// A reset whose pipeline signal fails once with a temporary provider error neither stops the
/// actor nor resumes derivation; the scheduled retry adopts the reset head and resumes.
#[tokio::test(start_paused = true)]
async fn temporary_reset_failure_is_retried_like_op_node() {
    let mut engine_client = MockDerivationEngineClient::new();
    engine_client.expect_send_safe_l2_signal().times(1).returning(|_| Ok(()));
    engine_client.expect_send_finalized_l2_block().times(0);
    let mut node = RecordingActor::new(engine_client);
    node.actor.pipeline.attributes.clear();
    node.complete_el_sync_at(DERIVED_HEAD).await;
    node.actor.pipeline.attributes.push_back(derived_attributes());
    node.actor.pipeline.signal_error = Some(l2_provider_error().temp());
    let previous_state = node.actor.derivation_state_machine.current_state();
    let previous_head = node.actor.derivation_state_machine.last_confirmed_safe_head();

    node.request_tx
        .send(DerivationActorRequest::ProcessEngineSignalRequest(Box::new(Signal::Reset(
            kona_derive::ResetSignal { l2_safe_head: l2_block(RESUME_HEAD) },
        ))))
        .await
        .unwrap();
    node.actor.step().await.expect("a temporary reset failure must not stop the actor");

    assert_eq!(node.actor.derivation_state_machine.current_state(), previous_state);
    assert_eq!(node.actor.derivation_state_machine.last_confirmed_safe_head(), previous_head);
    assert_eq!(node.actor.pipeline.attributes.len(), 1);
    assert!(node.actor.attributes_awaiting_confirmation.is_none());
    assert!(matches!(node.db.last_entry(), Err(SafeDbError::NotFound)));

    // No request is queued, so the next step is the scheduled retry.
    node.actor.step().await.expect("the retried reset must succeed");

    assert_eq!(
        node.actor.derivation_state_machine.last_confirmed_safe_head(),
        l2_block(RESUME_HEAD)
    );
    assert!(node.actor.attributes_awaiting_confirmation.is_some());
}

/// A critical reset failure stops the actor, as `ErrCritical` stops op-node.
#[tokio::test]
async fn critical_reset_failure_stops_the_actor_like_op_node() {
    let mut engine_client = MockDerivationEngineClient::new();
    engine_client.expect_send_safe_l2_signal().times(0);
    engine_client.expect_send_finalized_l2_block().times(0);
    let mut node = RecordingActor::new(engine_client);
    node.actor.pipeline.attributes.clear();
    node.complete_el_sync_at(DERIVED_HEAD).await;
    node.actor.pipeline.attributes.push_back(derived_attributes());
    node.actor.pipeline.signal_error = Some(l2_provider_error().crit());

    node.request_tx
        .send(DerivationActorRequest::ProcessEngineSignalRequest(Box::new(Signal::Reset(
            kona_derive::ResetSignal { l2_safe_head: l2_block(RESUME_HEAD) },
        ))))
        .await
        .unwrap();

    assert!(matches!(
        node.actor.step().await,
        Err(DerivationError::Pipeline(PipelineErrorKind::Critical(PipelineError::Provider(_))))
    ));
    assert!(node.actor.attributes_awaiting_confirmation.is_none());
}
