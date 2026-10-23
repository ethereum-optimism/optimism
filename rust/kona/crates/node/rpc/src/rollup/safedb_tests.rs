use crate::{
    RollupNodeApiServer, RollupRpc,
    safe_db_error::SAFE_DB_ERROR_CODE,
    test_utils::{StubEngineClient, l1_block, l1_watcher_at, l2_block},
};
use alloy_eips::BlockNumberOrTag;
use jsonrpsee::types::ErrorCode;
use kona_engine::EngineState;
use kona_safedb::{SafeDatabase, SafeDb};
use rstest::rstest;
use std::{collections::BTreeMap, sync::Arc};
use tempfile::TempDir;

const RECORDED_L1: u64 = 100;
const RECORDED_L2: u64 = 20;

fn rpc(safe_db: Option<Arc<dyn SafeDb>>) -> RollupRpc<StubEngineClient> {
    let engine = StubEngineClient::new(EngineState::default(), BTreeMap::new());
    RollupRpc::new("test-version", engine, l1_watcher_at(None), safe_db)
}

fn recorded() -> (TempDir, Option<Arc<dyn SafeDb>>) {
    let dir = TempDir::new().unwrap();
    let db = SafeDatabase::new(dir.path()).unwrap();
    db.safe_head_updated(l2_block(RECORDED_L2), l1_block(RECORDED_L1).id()).unwrap();
    (dir, Some(Arc::new(db)))
}

#[rstest]
#[case::at_the_record(RECORDED_L1)]
#[case::above_the_record(RECORDED_L1 + 50)]
#[tokio::test]
async fn answers_with_the_highest_record_at_or_below_the_l1_block(#[case] queried: u64) {
    let (_dir, db) = recorded();

    let response =
        rpc(db).op_safe_head_at_l1_block(BlockNumberOrTag::Number(queried)).await.unwrap();

    assert_eq!(response.l1_block, l1_block(RECORDED_L1).id());
    assert_eq!(response.safe_head, l2_block(RECORDED_L2).block_info.id());
}

/// op-node's rollup client recognizes a not-found answer by its message, so the message is
/// exactly the one op-node sends.
#[rstest]
#[case::below_all_records(BlockNumberOrTag::Number(RECORDED_L1 - 1))]
#[case::earliest(BlockNumberOrTag::Earliest)]
#[tokio::test]
async fn an_l1_block_below_every_record_is_not_found(#[case] queried: BlockNumberOrTag) {
    let (_dir, db) = recorded();

    let err = rpc(db).op_safe_head_at_l1_block(queried).await.unwrap_err();

    assert_eq!(err.code(), SAFE_DB_ERROR_CODE);
    assert_eq!(err.message(), "not found");
}

#[tokio::test]
async fn a_tag_that_needs_chain_context_is_rejected() {
    let (_dir, db) = recorded();

    let err = rpc(db).op_safe_head_at_l1_block(BlockNumberOrTag::Latest).await.unwrap_err();

    assert_eq!(err.code(), ErrorCode::InvalidParams.code());
}

#[tokio::test]
async fn a_disabled_database_is_reported_like_op_node_does() {
    let err = rpc(None)
        .op_safe_head_at_l1_block(BlockNumberOrTag::Number(RECORDED_L1))
        .await
        .unwrap_err();

    assert_eq!(err.code(), SAFE_DB_ERROR_CODE);
    assert!(err.message().contains("safe head database not enabled"), "{}", err.message());
    assert!(!err.message().contains("not found"), "a disabled database is not an empty one");
}
