use super::*;

#[tokio::test]
async fn critical_pipeline_reset_failure_cannot_resume_queued_attributes() {
    let error = l2_provider_error().crit();
    let mut engine_client = MockDerivationEngineClient::new();
    engine_client.expect_send_safe_l2_signal().times(0);
    engine_client.expect_send_finalized_l2_block().times(0);
    let mut node = RecordingActor::new(engine_client);
    node.actor.pipeline.attributes.clear();
    node.complete_el_sync_at(DERIVED_HEAD).await;
    node.actor.pipeline.attributes.push_back(derived_attributes());
    let expected_error = error.to_string();
    node.actor.pipeline.signal_error = Some(error);
    let previous_state = node.actor.derivation_state_machine.current_state();
    let previous_head = node.actor.derivation_state_machine.last_confirmed_safe_head();

    node.request_tx
        .send(DerivationActorRequest::ProcessEngineSignalRequest(Box::new(Signal::Reset(
            kona_derive::ResetSignal { l2_safe_head: l2_block(RESUME_HEAD) },
        ))))
        .await
        .unwrap();
    let actual_error = node.actor.step().await.unwrap_err();

    assert!(matches!(&actual_error, DerivationError::Pipeline(_)));
    assert_eq!(actual_error.to_string(), expected_error);
    assert_eq!(node.actor.derivation_state_machine.current_state(), previous_state);
    assert_eq!(node.actor.derivation_state_machine.last_confirmed_safe_head(), previous_head);
    assert_eq!(node.actor.pipeline.attributes.len(), 1);
    assert!(node.actor.attributes_awaiting_confirmation.is_none());
    assert!(matches!(node.db.last_entry(), Err(SafeDbError::NotFound)));
    assert!(node.actor.finalizer.try_finalize_next(l1_block(DERIVED_FROM_L1)).is_none());
}
