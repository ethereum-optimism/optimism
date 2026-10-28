use super::*;
use crate::{
    ChainIdAndOutput, DecimalChainId, OutputV0Response, SuperRootApiServer,
    SuperRootAtTimestampResponse,
    test_utils::{
        BlockOutput, GENESIS_TIME, l1_block, l1_watcher_at, l2_block, query_client,
        recorded_query_client, timestamp_of,
    },
};
use alloy_eips::BlockNumHash;
use alloy_primitives::{B256, U64};
use jsonrpsee::{core::RpcResult, types::ErrorCode};
use kona_engine::{EngineState, EngineSyncStateUpdate};
use kona_genesis::RollupConfig;
use kona_interop::{OutputRootWithChain, SuperRoot};
use kona_protocol::OutputRoot;
use kona_safedb::{SafeDatabase, SafeDb, SafeHeadRecord};
use rstest::rstest;
use std::sync::Arc;
use tempfile::TempDir;

const CHAIN_ID: u64 = 901;
/// The safe head trails local-safe, so a swap of the two labels fails the tests.
const SAFE: u64 = 4;
const LOCAL_SAFE: u64 = 5;
const FINALIZED: u64 = 3;
const CURRENT_L1: u64 = 130;

fn rollup_config() -> Arc<RollupConfig> {
    let mut config = RollupConfig {
        block_time: timestamp_of(1) - GENESIS_TIME,
        l2_chain_id: CHAIN_ID.into(),
        ..Default::default()
    };
    config.genesis.l2 = l2_block(0).block_info.id();
    config.genesis.l2_time = GENESIS_TIME;
    Arc::new(config)
}

fn output(block: u64) -> OutputRoot {
    OutputRoot {
        state_root: B256::with_last_byte(0x40 + block as u8),
        bridge_storage_root: B256::with_last_byte(0x60 + block as u8),
        block_hash: l2_block(block).block_info.hash,
    }
}

fn engine_state() -> EngineState {
    let sync_state = EngineState::default().sync_state.apply_update(EngineSyncStateUpdate {
        unsafe_head: Some(l2_block(LOCAL_SAFE + 2)),
        local_safe_head: Some(l2_block(LOCAL_SAFE)),
        safe_head: Some(l2_block(SAFE)),
        finalized_head: Some(l2_block(FINALIZED)),
    });
    EngineState { sync_state, el_sync_finished: true, ..Default::default() }
}

/// Blocks 1, 3 and 5 became safe at L1 blocks 100, 110 and 120.
fn recorded() -> (TempDir, Option<Arc<dyn SafeDb>>) {
    let dir = TempDir::new().unwrap();
    let db = SafeDatabase::new(dir.path()).unwrap();
    for (l2, l1) in [(1, 100), (3, 110), (5, 120)] {
        db.safe_head_updated(l2_block(l2), l1_block(l1).id()).unwrap();
    }
    (dir, Some(Arc::new(db)))
}

fn rpc(safe_db: Option<Arc<dyn SafeDb>>, timestamp: u64) -> SuperRootRpc {
    let config = rollup_config();
    let number = config.block_number_from_timestamp(timestamp.max(GENESIS_TIME));
    let block_output = (timestamp >= GENESIS_TIME && number <= LOCAL_SAFE)
        .then(|| BlockOutput { block: l2_block(number), output: output(number) });
    let record = if number == 0 {
        Some(SafeHeadRecord {
            l1: BlockNumHash { hash: B256::ZERO, number: 0 },
            safe_head: config.genesis.l2,
        })
    } else {
        safe_db.as_ref().and_then(|database| database.l1_at_safe_head(number).ok())
    };
    let (_, state) = tokio::sync::watch::channel(engine_state());
    SuperRootRpc::new(
        recorded_query_client(config.clone(), block_output, record),
        state,
        l1_watcher_at(Some(l1_block(CURRENT_L1))),
        config,
        safe_db,
    )
}

async fn at(
    safe_db: Option<Arc<dyn SafeDb>>,
    timestamp: u64,
) -> RpcResult<SuperRootAtTimestampResponse> {
    rpc(safe_db, timestamp).superroot_at_timestamp(U64::from(timestamp)).await
}

#[rstest]
#[case::on_the_block(timestamp_of(3), 3, 110)]
#[case::between_blocks(timestamp_of(3) + 1, 3, 110)]
#[case::above_safe_at_local_safe(timestamp_of(LOCAL_SAFE), LOCAL_SAFE, 120)]
#[tokio::test]
async fn a_local_safe_block_is_answered_with_the_l1_block_that_made_it_safe(
    #[case] timestamp: u64,
    #[case] block: u64,
    #[case] required_l1: u64,
) {
    let (_dir, db) = recorded();

    let response = at(db, timestamp).await.unwrap();

    let output_root = output(block).hash();
    let super_root =
        SuperRoot::new(timestamp, vec![OutputRootWithChain::new(CHAIN_ID, output_root)]);
    let data = response.data.expect("a local-safe block has data");
    assert_eq!(data.verified_required_l1, l1_block(required_l1).id());
    assert_eq!(data.super_root, super_root.hash());
    assert_eq!(data.super_v1.timestamp, U64::from(timestamp));
    assert_eq!(
        data.super_v1.chains,
        vec![ChainIdAndOutput { chain_id: DecimalChainId(CHAIN_ID), output: output_root }]
    );
    let optimistic = &response.optimistic_at_timestamp[&DecimalChainId(CHAIN_ID)];
    assert_eq!(optimistic.required_l1, l1_block(required_l1).id());
    assert_eq!(optimistic.output_root, output_root);
    assert_eq!(optimistic.output, OutputV0Response::from(&output(block)));
    assert_eq!(response.chain_ids, vec![DecimalChainId(CHAIN_ID)]);
    assert_eq!(response.current_l1, l1_block(CURRENT_L1).id());
    assert_eq!(response.current_local_safe_timestamp, timestamp_of(LOCAL_SAFE));
    assert_eq!(response.current_safe_timestamp, timestamp_of(SAFE));
    assert_eq!(response.current_finalized_timestamp, timestamp_of(FINALIZED));
}

/// A block recorded as safe together with a later one reads as safe at that later block's
/// L1 block, which is the earliest L1 block at which the recorded head had reached it.
#[tokio::test]
async fn a_block_without_its_own_record_reads_from_the_next_record() {
    let (_dir, db) = recorded();

    let response = at(db, timestamp_of(2)).await.unwrap();

    assert_eq!(response.data.unwrap().verified_required_l1, l1_block(110).id());
}

#[tokio::test]
async fn the_genesis_block_needs_no_l1_data() {
    let (_dir, db) = recorded();

    let response = at(db, GENESIS_TIME).await.unwrap();

    assert_eq!(
        response.data.unwrap().verified_required_l1,
        BlockNumHash { hash: B256::ZERO, number: 0 }
    );
}

#[tokio::test]
async fn a_block_beyond_local_safe_is_omitted() {
    let (_dir, db) = recorded();

    let response = at(db, timestamp_of(LOCAL_SAFE + 1)).await.unwrap();

    assert!(response.data.is_none());
    assert!(response.optimistic_at_timestamp.is_empty());
    assert_eq!(response.chain_ids, vec![DecimalChainId(CHAIN_ID)]);
    assert_eq!(response.current_l1, l1_block(CURRENT_L1).id());
}

/// The engine can reset between the two reads: the local-safe head then names a different
/// block at the same height, and the output just fetched belongs to neither record.
#[tokio::test]
async fn a_local_safe_head_replaced_between_reads_is_rejected() {
    let (_dir, db) = recorded();
    let mut replacement = l2_block(LOCAL_SAFE);
    replacement.block_info.parent_hash = B256::repeat_byte(0xee);
    let config = rollup_config();
    let (_, state) = tokio::sync::watch::channel(engine_state());
    let rpc = SuperRootRpc::new(
        recorded_query_client(
            config.clone(),
            Some(BlockOutput { block: replacement, output: output(LOCAL_SAFE) }),
            Some(SafeHeadRecord {
                l1: l1_block(120).id(),
                safe_head: l2_block(LOCAL_SAFE).block_info.id(),
            }),
        ),
        state,
        l1_watcher_at(Some(l1_block(CURRENT_L1))),
        config,
        db,
    );

    let error = rpc.superroot_at_timestamp(U64::from(timestamp_of(LOCAL_SAFE))).await.unwrap_err();

    assert_eq!(error.code(), -32000);
    assert!(error.message().contains("changed"));
}

#[tokio::test]
async fn a_timestamp_before_genesis_is_rejected() {
    let (_dir, db) = recorded();

    let err = at(db, GENESIS_TIME - 1).await.unwrap_err();

    assert_eq!(err.code(), ErrorCode::InvalidParams.code());
}

#[tokio::test]
async fn a_disabled_database_fails_a_local_safe_query_like_op_node_does() {
    let err = at(None, timestamp_of(3)).await.unwrap_err();

    assert!(err.message().contains("safe head database not enabled"), "{}", err.message());
}

/// The wire shape must be byte-compatible with op-node's, since the Go clients decode both.
#[tokio::test]
async fn the_response_serializes_in_op_node_wire_shape() {
    let (_dir, db) = recorded();
    let timestamp = timestamp_of(3);

    let response = at(db, timestamp).await.unwrap();

    let output_root = output(3).hash();
    let super_root =
        SuperRoot::new(timestamp, vec![OutputRootWithChain::new(CHAIN_ID, output_root)]);
    let expected = serde_json::json!({
        "current_l1": { "hash": l1_block(CURRENT_L1).hash, "number": CURRENT_L1 },
        "safe_timestamp": timestamp_of(SAFE),
        "local_safe_timestamp": timestamp_of(LOCAL_SAFE),
        "finalized_timestamp": timestamp_of(FINALIZED),
        "optimistic_at_timestamp": {
            "901": {
                "output": {
                    "stateRoot": output(3).state_root,
                    "messagePasserStorageRoot": output(3).bridge_storage_root,
                    "blockHash": output(3).block_hash,
                },
                "output_root": output_root,
                "required_l1": { "hash": l1_block(110).hash, "number": 110 },
            }
        },
        "chain_ids": ["901"],
        "data": {
            "verified_required_l1": { "hash": l1_block(110).hash, "number": 110 },
            "super": {
                "timestamp": format!("0x{timestamp:x}"),
                "chains": [{ "ChainID": "901", "Output": output_root }],
            },
            "super_root": super_root.hash(),
        },
    });
    assert_eq!(serde_json::to_value(&response).unwrap(), expected);
    assert_eq!(serde_json::from_value::<SuperRootAtTimestampResponse>(expected).unwrap(), response);
}

#[tokio::test]
async fn genesis_needs_no_database() {
    let response = at(None, GENESIS_TIME).await.unwrap();

    assert_eq!(response.data.unwrap().verified_required_l1.number, 0);
}

#[tokio::test]
async fn progress_beyond_local_safe_needs_no_database() {
    let response = at(None, timestamp_of(LOCAL_SAFE + 1)).await.unwrap();

    assert!(response.data.is_none());
    assert_eq!(response.current_l1, l1_block(CURRENT_L1).id());
}

#[tokio::test]
async fn an_output_read_failure_returns_an_internal_error() {
    let (_directory, database) = recorded();
    let config = rollup_config();
    let (engine, l1, l2) = kona_engine::test_utils::test_engine_client(config.clone());
    l2.expect_error("eth_getBlockByNumber");
    let (_, state) = tokio::sync::watch::channel(engine_state());
    let rpc = SuperRootRpc::new(
        engine.query_client(),
        state,
        l1_watcher_at(Some(l1_block(CURRENT_L1))),
        config,
        database,
    );

    let error = rpc.superroot_at_timestamp(U64::from(timestamp_of(3))).await.unwrap_err();

    assert_eq!(error.code(), ErrorCode::InternalError.code());
    l1.assert_finished();
    l2.assert_finished();
}

#[tokio::test]
async fn an_unavailable_l1_watcher_returns_an_internal_error() {
    let config = rollup_config();
    let (_, state) = tokio::sync::watch::channel(engine_state());
    let (sender, receiver) = tokio::sync::mpsc::channel(1);
    drop(receiver);
    let rpc = SuperRootRpc::new(query_client(config.clone()), state, sender, config, None);

    let error = rpc.superroot_at_timestamp(U64::from(timestamp_of(3))).await.unwrap_err();

    assert_eq!(error.code(), ErrorCode::InternalError.code());
}

mod reorg_tests;
