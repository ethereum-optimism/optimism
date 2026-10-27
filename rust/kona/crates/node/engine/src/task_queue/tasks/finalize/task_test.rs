use crate::{
    EngineTaskError, EngineTaskErrorSeverity, EngineTaskExt, FinalizeBlockId, FinalizeTask,
    FinalizeTaskError,
    test_utils::{TestEngineStateBuilder, test_block_info, test_engine_client},
};
use alloy_eips::BlockNumHash;
use alloy_primitives::b256;
use kona_genesis::RollupConfig;
use kona_protocol::{BlockInfo, L2BlockInfo};
use std::sync::Arc;

/// When the engine receives a `ByHash` finalize request for a block hash it doesn't have,
/// [`FinalizeTask`] must fail with [`FinalizeTaskError::BlockNotFound`] rather than silently
/// finalize whatever it happens to have at the same height.
///
/// Assert the exact by-hash RPC lookup, so a by-number lookup cannot silently finalize a
/// different canonical block at the same height.
#[tokio::test]
async fn finalize_task_by_hash_errors_when_engine_lacks_hash() {
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
    let mut state = TestEngineStateBuilder::new()
        .with_unsafe_head(safe_head)
        .with_safe_head(safe_head)
        .with_finalized_head(test_block_info(N - 1))
        .build();

    let task = FinalizeTask::new(
        Arc::new(engine_client),
        cfg,
        FinalizeBlockId::ByHash(BlockNumHash { number: N, hash: hash_b }),
    );

    let result = task.execute(&mut state).await;

    assert!(
        matches!(result, Err(FinalizeTaskError::BlockNotFound(n)) if n == N),
        "expected BlockNotFound({N}) — got {result:?}. The by-hash lookup must fail loudly when \
         the engine lacks the requested hash; instead, the task either succeeded (finalizing the \
         wrong block) or surfaced a different error."
    );
    l1.assert_finished();
    l2.assert_finished();
}

/// A real L2 block and its rollup metadata, including an L1-info deposit.
struct FinalizedBlock {
    block: alloy_rpc_types_eth::Block<op_alloy_rpc_types::Transaction>,
    info: L2BlockInfo,
}

impl FinalizedBlock {
    fn new(number: u64) -> Self {
        use alloy_primitives::{Address, Sealed};
        use alloy_rpc_types_eth::{Block, BlockTransactions, Header};
        use op_alloy_consensus::{OpTxEnvelope, TxDeposit};

        let header = Header::new(alloy_consensus::Header {
            number,
            timestamp: number * 2,
            ..Default::default()
        });
        let tx = OpTxEnvelope::Deposit(Sealed::new(TxDeposit {
            input: kona_protocol::L1BlockInfoBedrock::default().encode_calldata(),
            ..Default::default()
        }));
        let tx = op_alloy_rpc_types::Transaction::from_transaction(
            alloy_consensus::transaction::Recovered::new_unchecked(tx, Address::ZERO),
            Default::default(),
        );
        let block = Block::new(header, BlockTransactions::Full(vec![tx]));
        let mut cfg = RollupConfig::default();
        if number == cfg.genesis.l2.number {
            cfg.genesis.l2.hash = block.header.hash;
        }
        let info =
            L2BlockInfo::from_block_and_genesis(&block.clone().into_consensus(), &cfg.genesis)
                .unwrap();
        Self { block, info }
    }
}

#[rstest::rstest]
#[case::by_hash(true)]
#[case::by_number(false)]
#[tokio::test]
async fn finalize_task_ignores_older_target_without_rpc(#[case] by_hash: bool) {
    let cfg = Arc::new(RollupConfig::default());
    let (client, l1, l2) = test_engine_client(cfg.clone());
    let finalized = test_block_info(10);
    let mut state = TestEngineStateBuilder::new()
        .with_unsafe_head(test_block_info(30))
        .with_safe_head(test_block_info(20))
        .with_finalized_head(finalized)
        .build();
    let initial_state = state.sync_state;
    let target = if by_hash {
        FinalizeBlockId::ByHash(test_block_info(5).block_info.id())
    } else {
        FinalizeBlockId::ByNumber(5)
    };

    FinalizeTask::new(Arc::new(client), cfg, target).execute(&mut state).await.unwrap();

    assert_eq!(state.sync_state, initial_state);
    l1.assert_finished();
    l2.assert_finished();
}

#[rstest::rstest]
#[case::by_hash(true)]
#[case::by_number(false)]
#[tokio::test]
async fn finalize_task_identical_target_preserves_finalized(#[case] by_hash: bool) {
    let cfg = Arc::new(RollupConfig::default());
    let (client, l1, l2) = test_engine_client(cfg.clone());
    let FinalizedBlock { block, info: finalized } = FinalizedBlock::new(10);
    let mut state = TestEngineStateBuilder::new()
        .with_unsafe_head(test_block_info(30))
        .with_safe_head(test_block_info(20))
        .with_finalized_head(finalized)
        .build();
    let initial_state = state.sync_state;
    let target = if by_hash {
        l2.expect_params("eth_getBlockByHash", serde_json::json!([finalized.hash(), true]), block);
        FinalizeBlockId::ByHash(finalized.block_info.id())
    } else {
        l2.expect_params("eth_getBlockByNumber", serde_json::json!(["0xa", true]), block);
        FinalizeBlockId::ByNumber(10)
    };

    FinalizeTask::new(Arc::new(client), cfg, target).execute(&mut state).await.unwrap();

    assert_eq!(state.sync_state, initial_state);
    l1.assert_finished();
    l2.assert_finished();
}

#[rstest::rstest]
#[case::by_hash(true)]
#[case::by_number(false)]
#[tokio::test]
async fn finalize_task_rejects_conflicting_finalized_hash(#[case] by_hash: bool) {
    let cfg = Arc::new(RollupConfig::default());
    let (client, l1, l2) = test_engine_client(cfg.clone());
    let FinalizedBlock { block, info: requested } = FinalizedBlock::new(10);
    let finalized = test_block_info(10);
    assert_ne!(requested.hash(), finalized.hash());
    let mut state = TestEngineStateBuilder::new()
        .with_unsafe_head(test_block_info(30))
        .with_safe_head(test_block_info(20))
        .with_finalized_head(finalized)
        .build();
    let initial_state = state.sync_state;
    let target = if by_hash {
        // Reject the pinned contradiction before any lookup, including a missing local block.
        FinalizeBlockId::ByHash(requested.block_info.id())
    } else {
        l2.expect_params("eth_getBlockByNumber", serde_json::json!(["0xa", true]), block);
        FinalizeBlockId::ByNumber(10)
    };

    let error =
        FinalizeTask::new(Arc::new(client), cfg, target).execute(&mut state).await.unwrap_err();

    assert!(matches!(error, FinalizeTaskError::ConflictingFinalizedBlock {
        number: 10, current_hash, requested_hash
    } if current_hash == finalized.hash() && requested_hash == requested.hash()));
    assert_eq!(error.severity(), EngineTaskErrorSeverity::Critical);
    assert_eq!(state.sync_state, initial_state);
    l1.assert_finished();
    l2.assert_finished();
}

#[rstest::rstest]
#[case::advancing(false)]
#[case::initial_genesis(true)]
#[tokio::test]
async fn finalize_task_advances_or_initializes_finalized(#[case] genesis: bool) {
    use alloy_rpc_types_engine::{
        ForkchoiceState, ForkchoiceUpdated, PayloadStatus, PayloadStatusEnum,
    };

    let number = if genesis { 0 } else { 15 };
    let FinalizedBlock { block, info: requested } = FinalizedBlock::new(number);
    let mut config = RollupConfig::default();
    // Genesis has no L1-info deposit requirement, but its hash must match the rollup config.
    if genesis {
        config.genesis.l2 = requested.block_info.id();
    }
    let cfg = Arc::new(config);
    let (client, l1, l2) = test_engine_client(cfg.clone());
    let old_finalized = if genesis { L2BlockInfo::default() } else { test_block_info(10) };
    let safe = if genesis { requested } else { test_block_info(20) };
    let mut state = TestEngineStateBuilder::new()
        .with_unsafe_head(safe)
        .with_safe_head(safe)
        .with_finalized_head(old_finalized)
        .build();
    l2.expect_params(
        "eth_getBlockByNumber",
        serde_json::json!([format!("{number:#x}"), true]),
        block,
    );
    l2.expect_params(
        "engine_forkchoiceUpdatedV3",
        serde_json::json!([
            ForkchoiceState {
                head_block_hash: safe.hash(),
                safe_block_hash: safe.hash(),
                finalized_block_hash: requested.hash(),
            },
            null
        ]),
        ForkchoiceUpdated::new(PayloadStatus::from_status(PayloadStatusEnum::Valid)),
    );

    FinalizeTask::new(Arc::new(client), cfg, FinalizeBlockId::ByNumber(number))
        .execute(&mut state)
        .await
        .unwrap();

    assert_eq!(state.sync_state.finalized_head().block_info, requested.block_info);
    l1.assert_finished();
    l2.assert_finished();
}

#[tokio::test]
async fn finalize_task_still_rejects_target_ahead_of_safe() {
    let cfg = Arc::new(RollupConfig::default());
    let (client, l1, l2) = test_engine_client(cfg.clone());
    let mut state = TestEngineStateBuilder::new()
        .with_unsafe_head(test_block_info(30))
        .with_safe_head(test_block_info(20))
        .with_finalized_head(test_block_info(10))
        .build();
    let initial_state = state.sync_state;

    let error = FinalizeTask::new(Arc::new(client), cfg, FinalizeBlockId::ByNumber(21))
        .execute(&mut state)
        .await
        .unwrap_err();

    assert!(matches!(error, FinalizeTaskError::BlockNotSafe));
    assert_eq!(state.sync_state, initial_state);
    l1.assert_finished();
    l2.assert_finished();
}

#[tokio::test]
async fn finalize_task_rejects_by_number_response_below_current_finalized() {
    let cfg = Arc::new(RollupConfig::default());
    let (client, l1, l2) = test_engine_client(cfg.clone());
    let FinalizedBlock { block, .. } = FinalizedBlock::new(5);
    let mut state = TestEngineStateBuilder::new()
        .with_unsafe_head(test_block_info(30))
        .with_safe_head(test_block_info(20))
        .with_finalized_head(test_block_info(10))
        .build();
    let initial_state = state.sync_state;
    l2.expect_params("eth_getBlockByNumber", serde_json::json!(["0xf", true]), block);

    let error = FinalizeTask::new(Arc::new(client), cfg, FinalizeBlockId::ByNumber(15))
        .execute(&mut state)
        .await
        .unwrap_err();

    assert!(matches!(error, FinalizeTaskError::BlockNotFound(15)));
    assert_eq!(error.severity(), EngineTaskErrorSeverity::Critical);
    assert_eq!(state.sync_state, initial_state);
    l1.assert_finished();
    l2.assert_finished();
}
