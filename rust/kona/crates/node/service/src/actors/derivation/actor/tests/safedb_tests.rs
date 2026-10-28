use super::*;

#[rstest]
#[case::no_intervening_l1_update(0)]
#[case::one_intervening_l1_update(1)]
#[case::multiple_intervening_l1_updates(2)]
#[tokio::test]
async fn a_flushed_payload_keeps_its_original_derivation_source(
    #[case] intervening_l1_updates: u64,
    #[values(true, false)] is_last_in_span: bool,
) {
    let mut node = RecordingActor::new(engine_client_accepting_attributes());
    Arc::make_mut(&mut node.actor.pipeline.rollup_config).hardforks.holocene_time = Some(0);
    node.actor.pipeline.attributes.front_mut().unwrap().is_last_in_span = is_last_in_span;
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

/// Holocene partial-span validity follows L1 inclusion even when the L2 payload timestamp
/// lags activation. Each accepted singular block is independently recorded and finalizable.
#[tokio::test]
async fn a_holocene_prefix_is_recorded_and_finalizable_independently() {
    let mut engine_client = engine_client_accepting_attributes();
    engine_client
        .expect_send_finalized_l2_block()
        .withf(|id| matches!(id, FinalizeBlockId::ByNumber(DERIVED_HEAD)))
        .once()
        .returning(|_| Ok(()));
    let mut node = RecordingActor::new(engine_client);
    Arc::make_mut(&mut node.actor.pipeline.rollup_config).hardforks.holocene_time = Some(1);
    let attributes = node.actor.pipeline.attributes.front_mut().unwrap();
    attributes.is_last_in_span = false;
    attributes.derived_from.as_mut().unwrap().timestamp = 1;
    assert_eq!(attributes.attributes.payload_attributes.timestamp, 0);
    node.complete_el_sync_at(RESUME_HEAD).await;
    node.confirm_safe_head(DERIVED_HEAD).await;
    assert_eq!(node.db.last_entry().unwrap(), record(DERIVED_FROM_L1, DERIVED_HEAD));
    node.finalize_l1(DERIVED_FROM_L1).await;
}
