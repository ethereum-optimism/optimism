use crate::{
    RollupNodeApiServer, RollupRpc, SafeHeadResponse,
    safe_db_error::SAFE_DB_ERROR_CODE,
    test_utils::{l1_block, l1_watcher_at, l2_block, query_client},
};
use alloy_primitives::U64;
use jsonrpsee::types::ErrorCode;
use kona_engine::EngineState;
use kona_safedb::{SafeDatabase, SafeDb};
use rstest::rstest;
use std::sync::Arc;
use tempfile::TempDir;

const RECORDED_L1: u64 = 100;
const RECORDED_L2: u64 = 20;

fn rpc(safe_db: Option<Arc<dyn SafeDb>>) -> RollupRpc {
    let config = Arc::new(kona_genesis::RollupConfig::default());
    let (_, state) = tokio::sync::watch::channel(EngineState::default());
    RollupRpc::new(
        "test-version".into(),
        config.clone(),
        state,
        query_client(config, None),
        l1_watcher_at(None),
        safe_db,
    )
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

    let response = rpc(db).op_safe_head_at_l1_block(U64::from(queried)).await.unwrap();

    assert_eq!(response.l1_block, l1_block(RECORDED_L1).id());
    assert_eq!(response.safe_head, l2_block(RECORDED_L2).block_info.id());
}

/// op-node's rollup client recognizes a not-found answer by its message, so the message is
/// exactly the one op-node sends.
#[rstest]
#[case::below_all_records(RECORDED_L1 - 1)]
#[case::zero(0)]
#[tokio::test]
async fn an_l1_block_below_every_record_is_not_found(#[case] queried: u64) {
    let (_dir, db) = recorded();

    let err = rpc(db).op_safe_head_at_l1_block(U64::from(queried)).await.unwrap_err();

    assert_eq!(err.code(), SAFE_DB_ERROR_CODE);
    assert_eq!(err.message(), "not found");
}

#[rstest]
#[case::at_the_record("0x64")]
#[case::above_the_record("0x96")]
#[case::maximum("0xffffffffffffffff")]
#[tokio::test]
async fn the_rpc_accepts_hex_encoded_block_numbers(#[case] queried: &str) {
    let (_directory, database) = recorded();

    let response = rpc(database)
        .into_rpc()
        .call::<_, SafeHeadResponse>("optimism_safeHeadAtL1Block", jsonrpsee::rpc_params![queried])
        .await
        .unwrap();

    assert_eq!(response.l1_block, l1_block(RECORDED_L1).id());
    assert_eq!(response.safe_head, l2_block(RECORDED_L2).block_info.id());
}

#[rstest]
#[case::earliest("earliest")]
#[case::finalized("finalized")]
#[case::latest("latest")]
#[case::pending("pending")]
#[case::safe("safe")]
#[case::overflow("0x10000000000000000")]
#[tokio::test]
async fn the_rpc_rejects_tags_and_overflow_as_invalid_parameters(#[case] queried: &str) {
    let request = serde_json::json!({
        "id": 1,
        "jsonrpc": "2.0",
        "method": "optimism_safeHeadAtL1Block",
        "params": [queried],
    });

    let (response, _) =
        rpc(None).into_rpc().raw_json_request(&request.to_string(), 1).await.unwrap();
    let response: serde_json::Value = serde_json::from_str(response.get()).unwrap();

    assert_eq!(response["error"]["code"], ErrorCode::InvalidParams.code());
}

#[tokio::test]
async fn a_disabled_database_is_reported_like_op_node_does() {
    let err = rpc(None).op_safe_head_at_l1_block(U64::from(RECORDED_L1)).await.unwrap_err();

    assert_eq!(err.code(), SAFE_DB_ERROR_CODE);
    assert!(err.message().contains("safe head database not enabled"), "{}", err.message());
    assert!(!err.message().contains("not found"), "a disabled database is not an empty one");
}
