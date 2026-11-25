use super::*;

#[rstest]
#[case::no_intervening_l1_update(0)]
#[case::one_intervening_l1_update(1)]
#[case::multiple_intervening_l1_updates(2)]
#[tokio::test]
async fn a_flushed_payload_keeps_its_original_derivation_source(
    #[case] intervening_l1_updates: u64,
) {
    let mut node = RecordingActor::new(engine_client_accepting_attributes());
    node.complete_el_sync_at(RESUME_HEAD).await;
    let next_source = DERIVED_FROM_L1 + 1;
    let mut competing_attributes = derived_attributes();
    competing_attributes.derived_from = Some(l1_block(next_source));
    node.actor.pipeline.attributes.push_back(competing_attributes);

    node.signal(Signal::FlushChannel).await;
    for offset in 0..intervening_l1_updates {
        node.handle(DerivationActorRequest::ProcessL1HeadUpdateRequest(Box::new(l1_block(
            next_source + offset,
        ))))
        .await;
    }
    let state_before_confirmation = node.actor.derivation_state_machine.current_state();
    node.confirm_safe_head(DERIVED_HEAD).await;

    assert_eq!(node.db.first_entry().unwrap(), record(DERIVED_FROM_L1, DERIVED_HEAD));
    assert_eq!(node.db.last_entry().unwrap(), record(DERIVED_FROM_L1, DERIVED_HEAD));
    assert_eq!(state_before_confirmation, DerivationState::AwaitingSafeHeadConfirmation);
    assert_eq!(
        node.actor.attributes_awaiting_confirmation.as_ref().unwrap().derived_from,
        Some(l1_block(next_source)),
    );
}

#[tokio::test]
async fn a_flush_while_waiting_for_l1_data_allows_derivation_after_an_l1_update() {
    let mut node = RecordingActor::new(engine_client_accepting_attributes());
    node.actor.pipeline.attributes.clear();
    node.complete_el_sync_at(RESUME_HEAD).await;

    node.signal(Signal::FlushChannel).await;

    assert_eq!(
        node.actor.derivation_state_machine.current_state(),
        DerivationState::AwaitingL1Data
    );
    node.actor.pipeline.attributes.push_back(derived_attributes());
    node.handle(DerivationActorRequest::ProcessL1HeadUpdateRequest(Box::new(l1_block(
        DERIVED_FROM_L1,
    ))))
    .await;
    node.confirm_safe_head(DERIVED_HEAD).await;
    assert_eq!(node.db.last_entry().unwrap(), record(DERIVED_FROM_L1, DERIVED_HEAD));
}
