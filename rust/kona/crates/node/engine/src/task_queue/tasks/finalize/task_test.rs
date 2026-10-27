use crate::{
    ConsolidateInput, ConsolidateTask, Engine, EngineTask, EngineTaskError,
    EngineTaskErrorSeverity, EngineTaskExt, FinalizeBlockId, FinalizeTask, FinalizeTaskError,
    InsertTask, NoopBlockSink,
    test_utils::{TestEngineStateBuilder, test_block_info, test_engine_client},
};
use alloy_eips::BlockNumHash;
use alloy_primitives::{Address, Sealed, b256};
use alloy_rpc_types_engine::{
    ExecutionPayloadV1, ForkchoiceUpdated, PayloadStatus, PayloadStatusEnum,
};
use alloy_rpc_types_eth::{Block, BlockTransactions, Header};
use kona_genesis::RollupConfig;
use kona_protocol::{BlockInfo, L1BlockInfoBedrock, L2BlockInfo};
use op_alloy_consensus::{OpTxEnvelope, TxDeposit};
use op_alloy_rpc_types_engine::OpExecutionPayloadEnvelope;
use std::sync::Arc;

/// When the engine receives a `ByHash` finalize request for a block hash it doesn't have,
/// [`FinalizeTask`] must defer finalization rather than silently finalize whatever it happens
/// to have at the same height. Delegated polling retries the pinned target.
///
/// Assert the exact by-hash RPC lookup, so a by-number lookup cannot silently finalize a
/// different canonical block at the same height.
#[tokio::test]
async fn finalize_task_by_hash_defers_when_engine_lacks_hash() {
    const N: u64 = 10;
    let hash_a = b256!("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa");
    let hash_b = b256!("bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb");

    let cfg = Arc::new(RollupConfig::default());
    let (engine_client, l1, l2) = test_engine_client(cfg.clone());
    l2.expect_params(
        "eth_getBlockByHash",
        serde_json::json!([hash_b, true]),
        Option::<alloy_rpc_types_eth::Block<op_alloy_rpc_types::Transaction>>::None,
    );

    // Place the safe head at N so the sanity check (`safe_head.number >= block_id.number`) passes
    // and `execute()` reaches the lookup we care about.
    let safe_head = L2BlockInfo {
        block_info: BlockInfo {
            number: N,
            hash: hash_a,
            parent_hash: Default::default(),
            timestamp: N * 2,
        },
        l1_origin: BlockNumHash::default(),
        seq_num: 0,
    };
    let previous_finalized = test_block_info(N - 1);
    let mut state = TestEngineStateBuilder::new()
        .with_unsafe_head(safe_head)
        .with_safe_head(safe_head)
        .with_finalized_head(previous_finalized)
        .build();

    let task = FinalizeTask::new(
        Arc::new(engine_client),
        cfg,
        FinalizeBlockId::ByHash(BlockNumHash { number: N, hash: hash_b }),
    );

    task.execute(&mut state).await.unwrap();
    assert_eq!(state.sync_state.finalized_head(), previous_finalized);
    l1.assert_finished();
    l2.assert_finished();
}

fn rpc_block(number: u64) -> Block<op_alloy_rpc_types::Transaction> {
    let header = Header::new(alloy_consensus::Header {
        number,
        timestamp: number * 2,
        ..Default::default()
    });
    let transaction = OpTxEnvelope::Deposit(Sealed::new(TxDeposit {
        input: L1BlockInfoBedrock::default().encode_calldata(),
        ..Default::default()
    }));
    let transaction = op_alloy_rpc_types::Transaction::from_transaction(
        alloy_consensus::transaction::Recovered::new_unchecked(transaction, Address::ZERO),
        Default::default(),
    );
    Block::new(header, BlockTransactions::Full(vec![transaction]))
}

#[tokio::test]
async fn missing_delegated_finalized_block_releases_queue_and_recovers_on_next_poll() {
    let cfg = Arc::new(RollupConfig::default());
    let (client, l1, l2) = test_engine_client(cfg.clone());
    let client = Arc::new(client);
    let previous_finalized = test_block_info(0);
    let state = TestEngineStateBuilder::new()
        .with_unsafe_head(test_block_info(20))
        .with_safe_head(test_block_info(10))
        .with_finalized_head(previous_finalized)
        .build();
    let (state_tx, state_rx) = tokio::sync::watch::channel(state);
    let mut engine = Engine::new(state, state_tx);
    let finalized_block = rpc_block(15);
    let finalized = L2BlockInfo::from_block_and_genesis(
        &finalized_block.clone().into_consensus(),
        &cfg.genesis,
    )
    .unwrap();
    let target = FinalizeBlockId::ByHash(finalized.block_info.id());
    l2.expect(
        "engine_forkchoiceUpdatedV3",
        ForkchoiceUpdated::new(PayloadStatus::from_status(PayloadStatusEnum::Syncing)),
    );
    l2.expect_params(
        "eth_getBlockByHash",
        serde_json::json!([finalized.hash(), true]),
        Option::<Block<op_alloy_rpc_types::Transaction>>::None,
    );
    engine.enqueue(EngineTask::Consolidate(Box::new(ConsolidateTask::new(
        client.clone(),
        cfg.clone(),
        ConsolidateInput::BlockInfo(test_block_info(30)),
        Arc::new(NoopBlockSink),
    ))));
    engine.enqueue(EngineTask::Finalize(Box::new(FinalizeTask::new(
        client.clone(),
        cfg.clone(),
        target,
    ))));
    tokio::time::timeout(std::time::Duration::from_secs(1), engine.drain())
        .await
        .expect("missing data must yield, not retry inline")
        .unwrap();
    assert!(engine.is_empty(), "deferred finalization must not block new unsafe imports");
    assert_eq!(state_rx.borrow().sync_state.finalized_head(), previous_finalized);

    // An unsafe import can run before the next delegated poll retries the same target.
    let mut imported_block = rpc_block(31)
        .into_consensus()
        .map_transactions(|transaction| transaction.inner.inner.into_inner());
    imported_block.header.parent_hash = engine.state().sync_state.unsafe_head().hash();
    let payload = ExecutionPayloadV1::from_block_slow(&imported_block);
    let imported: op_alloy_consensus::OpBlock =
        OpExecutionPayloadEnvelope::V1(payload.clone()).try_into_block().unwrap();
    l2.expect("engine_newPayloadV1", PayloadStatus::from_status(PayloadStatusEnum::Valid));
    l2.expect(
        "engine_forkchoiceUpdatedV3",
        ForkchoiceUpdated::new(PayloadStatus::from_status(PayloadStatusEnum::Valid)),
    );
    l2.expect_params(
        "eth_getBlockByHash",
        serde_json::json!([finalized.hash(), true]),
        finalized_block,
    );
    l2.expect_params(
        "engine_forkchoiceUpdatedV3",
        serde_json::json!([
            alloy_rpc_types_engine::ForkchoiceState {
                head_block_hash: imported.header.hash_slow(),
                safe_block_hash: engine.state().sync_state.safe_head().hash(),
                finalized_block_hash: finalized.hash(),
            },
            null
        ]),
        ForkchoiceUpdated::new(PayloadStatus::from_status(PayloadStatusEnum::Valid)),
    );
    engine.enqueue(EngineTask::Insert(Box::new(InsertTask::new(
        client.clone(),
        cfg.clone(),
        OpExecutionPayloadEnvelope::V1(payload),
        false,
        Arc::new(NoopBlockSink),
    ))));
    engine.enqueue(EngineTask::Finalize(Box::new(FinalizeTask::new(client, cfg, target))));
    engine.drain().await.unwrap();
    assert!(engine.is_empty());
    assert_eq!(state_rx.borrow().sync_state.unsafe_head().block_info.number, 31);
    assert_eq!(state_rx.borrow().sync_state.finalized_head(), finalized);
    l1.assert_finished();
    l2.assert_finished();
}

#[tokio::test]
async fn malformed_hash_pinned_finalized_block_is_critical() {
    for wrong_number in [false, true] {
        let cfg = Arc::new(RollupConfig::default());
        let (client, l1, l2) = test_engine_client(cfg.clone());
        let block = rpc_block(10);
        let actual =
            L2BlockInfo::from_block_and_genesis(&block.clone().into_consensus(), &cfg.genesis)
                .unwrap()
                .block_info
                .id();
        let expected = if wrong_number {
            BlockNumHash { number: 11, hash: actual.hash }
        } else {
            BlockNumHash {
                number: 10,
                hash: b256!("bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"),
            }
        };
        l2.expect_params("eth_getBlockByHash", serde_json::json!([expected.hash, true]), block);
        let mut state = TestEngineStateBuilder::new()
            .with_unsafe_head(test_block_info(20))
            .with_finalized_head(test_block_info(0))
            .build();
        let previous_finalized = state.sync_state.finalized_head();
        let err = FinalizeTask::new(Arc::new(client), cfg, FinalizeBlockId::ByHash(expected))
            .execute(&mut state)
            .await
            .unwrap_err();
        assert!(
            matches!(err, FinalizeTaskError::BlockIdentityMismatch { expected: e, actual: a } if e == expected && a == actual)
        );
        assert_eq!(err.severity(), EngineTaskErrorSeverity::Critical);
        assert_eq!(state.sync_state.finalized_head(), previous_finalized);
        l1.assert_finished();
        l2.assert_finished();
    }
}

#[tokio::test]
async fn missing_number_only_finalized_block_remains_critical() {
    let cfg = Arc::new(RollupConfig::default());
    let (client, l1, l2) = test_engine_client(cfg.clone());
    l2.expect_params(
        "eth_getBlockByNumber",
        serde_json::json!(["0xa", true]),
        Option::<Block<op_alloy_rpc_types::Transaction>>::None,
    );
    let mut state = TestEngineStateBuilder::new()
        .with_unsafe_head(test_block_info(20))
        .with_finalized_head(test_block_info(0))
        .build();
    let err = FinalizeTask::new(Arc::new(client), cfg, FinalizeBlockId::ByNumber(10))
        .execute(&mut state)
        .await
        .unwrap_err();
    assert!(matches!(err, FinalizeTaskError::BlockNotFound(10)));
    assert_eq!(err.severity(), EngineTaskErrorSeverity::Critical);
    l1.assert_finished();
    l2.assert_finished();
}
