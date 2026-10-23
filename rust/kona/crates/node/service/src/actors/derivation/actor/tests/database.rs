use super::*;
use kona_derive::ResetSignal;
use mockall::{mock, predicate::eq};

mock! {
    #[derive(Debug)]
    SafeDatabase {}

    impl SafeDb for SafeDatabase {
        fn clear(&self) -> Result<(), SafeDbError>;
        fn first_entry(&self) -> Result<SafeHeadRecord, SafeDbError>;
        fn l1_at_safe_head(&self, target_l2_num: u64) -> Result<SafeHeadRecord, SafeDbError>;
        fn last_entry(&self) -> Result<SafeHeadRecord, SafeDbError>;
        fn safe_head_at_l1(&self, l1_block_num: u64) -> Result<SafeHeadRecord, SafeDbError>;
        fn safe_head_reset(&self, safe_head: L2BlockInfo) -> Result<(), SafeDbError>;
        fn safe_head_updated(
            &self,
            safe_head: L2BlockInfo,
            l1_head: BlockNumHash,
        ) -> Result<(), SafeDbError>;
    }
}

fn actor_with_database(
    database: Option<Arc<dyn SafeDb>>,
) -> DerivationActor<MockDerivationEngineClient, StubPipeline> {
    let (_request_sender, request_receiver) = mpsc::channel(1);
    DerivationActor::new(
        MockDerivationEngineClient::new(),
        request_receiver,
        StubPipeline {
            origin: BlockInfo::default(),
            reset_error: None,
            rollup_config: Arc::new(RollupConfig::default()),
            step_result: StepResult::StepFailed(PipelineError::Eof.temp()),
        },
        database,
        watch::channel(None).0,
    )
}

fn tip_record(number: u64) -> Result<SafeHeadRecord, SafeDbError> {
    Ok(record(DERIVED_FROM_L1, number))
}

#[test]
fn a_confirmed_block_is_written_directly_to_its_derivation_source() {
    let mut database = MockSafeDatabase::new();
    database
        .expect_safe_head_updated()
        .with(eq(l2_block(DERIVED_HEAD)), eq(l1_block(DERIVED_FROM_L1).id()))
        .once()
        .returning(|_, _| Ok(()));
    let mut actor = actor_with_database(Some(Arc::new(database)));
    actor.attributes_awaiting_confirmation = Some(derived_attributes());

    actor.settle_derived_attributes(l2_block(DERIVED_HEAD)).unwrap();

    assert_eq!(actor.finalizer.try_finalize_next(l1_block(DERIVED_FROM_L1)), Some(DERIVED_HEAD));
}

#[test]
fn a_failed_write_is_returned_without_retry_or_finalization() {
    let mut database = MockSafeDatabase::new();
    database
        .expect_safe_head_updated()
        .with(eq(l2_block(DERIVED_HEAD)), eq(l1_block(DERIVED_FROM_L1).id()))
        .once()
        .returning(|_, _| Err(SafeDbError::InvalidEntry));
    let mut actor = actor_with_database(Some(Arc::new(database)));
    actor.attributes_awaiting_confirmation = Some(derived_attributes());

    let result = actor.settle_derived_attributes(l2_block(DERIVED_HEAD));

    assert!(matches!(result, Err(DerivationError::SafeDbError(SafeDbError::InvalidEntry))));
    assert!(actor.finalizer.try_finalize_next(l1_block(DERIVED_FROM_L1)).is_none());
}

#[rstest]
#[case::empty_history(Err(SafeDbError::NotFound))]
#[case::tip_at_resume_head(tip_record(RESUME_HEAD))]
#[case::tip_above_resume_head(tip_record(RESUME_HEAD + 10))]
#[case::tip_read_fails(Err(SafeDbError::InvalidEntry))]
fn truncation_resets_when_the_tip_is_not_known_to_be_below_the_resume_head(
    #[case] tip: Result<SafeHeadRecord, SafeDbError>,
) {
    let mut database = MockSafeDatabase::new();
    database.expect_last_entry().once().return_once(move || tip);
    database.expect_safe_head_reset().with(eq(l2_block(RESUME_HEAD))).once().returning(|_| Ok(()));
    let actor = actor_with_database(Some(Arc::new(database)));

    actor.truncate_safe_db(l2_block(RESUME_HEAD), "engine reset").unwrap();
}

#[test]
fn truncation_clears_history_when_the_tip_is_below_the_resume_head() {
    let mut database = MockSafeDatabase::new();
    database.expect_last_entry().once().return_once(|| tip_record(RESUME_HEAD - 5));
    database.expect_clear().once().returning(|| Ok(()));
    let actor = actor_with_database(Some(Arc::new(database)));

    actor.truncate_safe_db(l2_block(RESUME_HEAD), "engine reset").unwrap();
}

#[test]
fn a_failed_clear_is_returned_without_retry() {
    let mut database = MockSafeDatabase::new();
    database.expect_last_entry().once().return_once(|| tip_record(RESUME_HEAD - 5));
    database.expect_clear().once().returning(|| Err(SafeDbError::InvalidEntry));
    let actor = actor_with_database(Some(Arc::new(database)));

    let result = actor.truncate_safe_db(l2_block(RESUME_HEAD), "engine reset");

    assert!(matches!(result, Err(SafeDbError::InvalidEntry)));
}

fn engine_reset_request() -> DerivationActorRequest {
    DerivationActorRequest::ProcessEngineSignalRequest(Box::new(Signal::Reset(ResetSignal {
        l2_safe_head: l2_block(RESUME_HEAD),
    })))
}

fn sync_completion_request() -> DerivationActorRequest {
    DerivationActorRequest::ProcessEngineSyncCompletionRequest(Box::new(l2_block(RESUME_HEAD)))
}

#[rstest]
#[case::engine_reset(engine_reset_request())]
#[case::el_sync_completion(sync_completion_request())]
#[tokio::test]
async fn a_failed_truncation_stops_request_processing(#[case] request: DerivationActorRequest) {
    let mut database = MockSafeDatabase::new();
    database.expect_last_entry().once().returning(|| Err(SafeDbError::NotFound));
    database
        .expect_safe_head_reset()
        .with(eq(l2_block(RESUME_HEAD)))
        .once()
        .returning(|_| Err(SafeDbError::InvalidEntry));
    let mut actor = actor_with_database(Some(Arc::new(database)));

    let result = actor.handle_derivation_actor_request(request).await;

    assert!(matches!(result, Err(DerivationError::SafeDbError(SafeDbError::InvalidEntry))));
}

#[test]
fn an_absent_database_skips_writes_and_truncation() {
    let mut actor = actor_with_database(None);
    actor.attributes_awaiting_confirmation = Some(derived_attributes());

    actor.settle_derived_attributes(l2_block(DERIVED_HEAD)).unwrap();
    actor.truncate_safe_db(l2_block(RESUME_HEAD), "engine reset").unwrap();

    assert_eq!(actor.finalizer.try_finalize_next(l1_block(DERIVED_FROM_L1)), Some(DERIVED_HEAD));
}
