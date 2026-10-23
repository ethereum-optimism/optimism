use super::*;
use alloy_primitives::B256;
use assert_matches::assert_matches;
use kona_protocol::BlockInfo;
use kona_safedb::SafeHeadRecord;
use mockall::{Sequence, mock, predicate::eq};
use rstest::rstest;

const DERIVED_FROM_L1: u64 = 100;
const RESUME_HEAD: u64 = 20;
const SAFE_HEAD_L1_ORIGIN: u64 = 42;

mock! {
    #[derive(Debug)]
    SafeDatabase {}

    impl SafeDb for SafeDatabase {
        fn clear(&self) -> Result<(), SafeDbError>;
        fn close(&self) -> Result<(), SafeDbError>;
        fn enabled(&self) -> bool;
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

fn derived_from() -> BlockNumHash {
    BlockNumHash { hash: B256::repeat_byte(1), number: DERIVED_FROM_L1 }
}

fn l2_block(number: u64) -> L2BlockInfo {
    L2BlockInfo {
        block_info: BlockInfo { hash: B256::repeat_byte(2), number, ..Default::default() },
        l1_origin: BlockNumHash { hash: B256::ZERO, number: SAFE_HEAD_L1_ORIGIN },
        seq_num: 0,
    }
}

fn tip_record(number: u64) -> Result<SafeHeadRecord, SafeDbError> {
    Ok(SafeHeadRecord { l1: derived_from(), safe_head: l2_block(number).block_info.id() })
}

fn enabled_database() -> MockSafeDatabase {
    let mut database = MockSafeDatabase::new();
    database.expect_enabled().once().return_const(true);
    database
}

#[tokio::test]
async fn a_record_is_keyed_by_the_l1_block_the_head_was_derived_from() {
    let mut database = enabled_database();
    database
        .expect_safe_head_updated()
        .with(eq(l2_block(RESUME_HEAD + 1)), eq(derived_from()))
        .once()
        .returning(|_, _| Ok(()));
    let recorder = SafeHeadRecorder::new(Arc::new(database));

    recorder.record(l2_block(RESUME_HEAD + 1), derived_from()).await.unwrap();
}

#[rstest]
#[case::empty_history(Err(SafeDbError::NotFound))]
#[case::tip_at_resume_head(tip_record(RESUME_HEAD))]
#[case::tip_above_resume_head(tip_record(RESUME_HEAD + 10))]
#[case::tip_read_fails(Err(SafeDbError::Closed))]
#[tokio::test]
async fn a_truncation_resets_when_the_tip_is_not_known_to_be_below_the_resume_head(
    #[case] tip: Result<SafeHeadRecord, SafeDbError>,
) {
    let mut database = enabled_database();
    database.expect_last_entry().once().return_once(move || tip);
    database.expect_safe_head_reset().with(eq(l2_block(RESUME_HEAD))).once().returning(|_| Ok(()));
    let recorder = SafeHeadRecorder::new(Arc::new(database));

    recorder.truncate(l2_block(RESUME_HEAD), TruncationReason::EngineReset).await.unwrap();
}

#[tokio::test]
async fn a_truncation_clears_history_when_the_tip_is_below_the_resume_head() {
    let mut database = enabled_database();
    database.expect_last_entry().once().return_once(|| tip_record(RESUME_HEAD - 5));
    database.expect_clear().once().returning(|| Ok(()));
    let recorder = SafeHeadRecorder::new(Arc::new(database));

    recorder.truncate(l2_block(RESUME_HEAD), TruncationReason::EngineReset).await.unwrap();
}

#[rstest]
#[case::first_retry(1)]
#[case::last_retry(WRITE_RETRIES)]
#[tokio::test(start_paused = true)]
async fn a_failed_write_is_retried_until_it_succeeds(#[case] failed_attempts: usize) {
    let mut database = enabled_database();
    let mut sequence = Sequence::new();
    database
        .expect_safe_head_updated()
        .with(eq(l2_block(RESUME_HEAD + 1)), eq(derived_from()))
        .times(failed_attempts)
        .in_sequence(&mut sequence)
        .returning(|_, _| Err(SafeDbError::Closed));
    database
        .expect_safe_head_updated()
        .with(eq(l2_block(RESUME_HEAD + 1)), eq(derived_from()))
        .once()
        .in_sequence(&mut sequence)
        .returning(|_, _| Ok(()));
    let recorder = SafeHeadRecorder::new(Arc::new(database));

    recorder.record(l2_block(RESUME_HEAD + 1), derived_from()).await.unwrap();
}

#[tokio::test(start_paused = true)]
async fn a_write_that_exhausts_its_retries_returns_the_database_error() {
    let mut database = enabled_database();
    database
        .expect_safe_head_updated()
        .with(eq(l2_block(RESUME_HEAD + 1)), eq(derived_from()))
        .times(WRITE_RETRIES + 1)
        .returning(|_, _| Err(SafeDbError::Closed));
    let recorder = SafeHeadRecorder::new(Arc::new(database));

    let result = recorder.record(l2_block(RESUME_HEAD + 1), derived_from()).await;

    assert_matches!(result, Err(SafeDbError::Closed));
}

#[tokio::test]
async fn a_disabled_database_is_never_read_or_written() {
    let mut database = MockSafeDatabase::new();
    database.expect_enabled().times(2).return_const(false);
    let recorder = SafeHeadRecorder::new(Arc::new(database));

    recorder.record(l2_block(RESUME_HEAD + 1), derived_from()).await.unwrap();
    recorder.truncate(l2_block(RESUME_HEAD), TruncationReason::ElSyncCompleted).await.unwrap();
}
