//! Regression vectors from op-node/rollup/sync/start_test.go, plus malformed traversal edges.

use super::*;
use crate::test_utils::{RpcMock, test_engine_client, test_l2_block};
use alloy_rpc_types_eth::Block;
use serde_json::json;
use std::sync::Arc;

type L1Block = Block<alloy_rpc_types_eth::Transaction>;
type L2Block = Block<op_alloy_rpc_types::Transaction>;

fn l1_block(number: u64) -> L1Block {
    let mut block = L1Block::default();
    block.header.inner.number = number;
    block.header.hash = B256::with_last_byte(number as u8);
    block.header.inner.parent_hash = B256::with_last_byte(number.saturating_sub(1) as u8);
    block
}

fn info(block: &L2Block, cfg: &RollupConfig) -> L2BlockInfo {
    L2BlockInfo::from_block_and_genesis(&block.clone().into_consensus(), &cfg.genesis).unwrap()
}

fn expect_heads(mock: &RpcMock, finalized: &L2Block, safe: &L2Block, head: &L2Block) {
    for (tag, block) in [("finalized", finalized), ("safe", safe), ("latest", head)] {
        mock.expect_params("eth_getBlockByNumber", json!([tag, true]), block);
    }
}

fn expect_parent(mock: &RpcMock, parent: &L2Block) {
    mock.expect_params("eth_getBlockByHash", json!([parent.header.hash, true]), parent);
}

// Same histories and expected labels as op-node's small-reorg, L1-ahead and L2-ahead vectors.
// The final case keeps the previous safe head below the lookback boundary.
#[rstest::rstest]
#[case::small_reorg(8, 8, Some(7), 1, 7, (6, 2), &[(7,false),(6,true),(5,true),(4,true),(3,true)])]
#[case::l1_ahead(5, 6, None, 0, 3, (4, 0), &[(4,false),(3,true),(2,true),(1,true)])]
#[case::l2_ahead_after_reorg(6, 4, None, 1, 3, (5, 1), &[(5,false),(4,false),(3,false),(2,true),(1,true)])]
#[case::prior_safe_limits_recovery(12, 12, None, 1, 3, (11, 2), &[(11,false),(10,true),(9,true),(8,true),(7,true),(6,true),(5,true),(4,true),(3,true)])]
#[tokio::test]
async fn matches_op_node_sync_start_vectors(
    #[case] l2_len: usize,
    #[case] l1_len: usize,
    #[case] replaced: Option<usize>,
    #[case] finalized: usize,
    #[case] safe: usize,
    #[case] expected: (usize, usize),
    #[case] reads: &[(usize, bool)],
) {
    let (expected_unsafe, expected_safe) = expected;
    let mut blocks = vec![];
    for i in 0..l2_len {
        let parent = blocks.last().map_or(B256::ZERO, |b: &L2Block| b.header.hash);
        blocks.push(test_l2_block(i as u64, i as u64, 0, parent));
    }
    let mut cfg = RollupConfig { seq_window_size: 2, ..Default::default() };
    cfg.genesis.l2.hash = blocks[0].header.hash;
    let (client, l1, l2) = test_engine_client(Arc::new(cfg.clone()));
    expect_heads(&l2, &blocks[finalized], &blocks[safe], blocks.last().unwrap());
    for &(number, by_hash) in reads {
        let mut block = (number < l1_len).then(|| l1_block(number as u64));
        if replaced == Some(number) {
            block.as_mut().unwrap().header.hash = B256::repeat_byte(0xff);
        }
        let (method, id) = if by_hash {
            ("eth_getBlockByHash", json!(B256::with_last_byte(number as u8)))
        } else {
            ("eth_getBlockByNumber", json!(format!("{number:#x}")))
        };
        l1.expect_params(method, json!([id, false]), block);
    }
    for parent in (expected_safe..l2_len - 1).rev() {
        expect_parent(&l2, &blocks[parent]);
    }
    let result = find_starting_forkchoice(&cfg, &client).await.unwrap();
    assert_eq!(result.un_safe, info(&blocks[expected_unsafe], &cfg));
    assert_eq!(result.safe, info(&blocks[expected_safe], &cfg));
    assert_eq!(result.finalized, info(&blocks[finalized], &cfg));
    l1.assert_finished();
    l2.assert_finished();
}

#[rstest::rstest]
#[case::sequencing_window(false, 1)]
#[case::alt_da_window(true, 0)]
#[tokio::test]
async fn safe_head_is_last_block_before_epoch_boundary(
    #[case] alt_da: bool,
    #[case] expected: usize,
) {
    let mut blocks = vec![];
    for i in 0..8 {
        let parent = blocks.last().map_or(B256::ZERO, |b: &L2Block| b.header.hash);
        blocks.push(test_l2_block(i, i / 2, i % 2, parent));
    }
    let mut cfg = RollupConfig { seq_window_size: 1, ..Default::default() };
    if alt_da {
        cfg.alt_da_config = Some(kona_genesis::AltDAConfig {
            da_challenge_window: Some(1),
            da_resolve_window: Some(1),
            ..Default::default()
        });
    }
    cfg.genesis.l2.hash = blocks[0].header.hash;
    let (client, l1, l2) = test_engine_client(Arc::new(cfg.clone()));
    expect_heads(&l2, &blocks[0], &blocks[7], &blocks[7]);
    l1.expect_params("eth_getBlockByNumber", json!(["0x3", false]), l1_block(3));
    for number in (if alt_da { 0 } else { 1 }..=2).rev() {
        l1.expect_params(
            "eth_getBlockByHash",
            json!([B256::with_last_byte(number), false]),
            l1_block(number as u64),
        );
    }
    for parent in (expected..7).rev() {
        expect_parent(&l2, &blocks[parent]);
    }
    let result = find_starting_forkchoice(&cfg, &client).await.unwrap();
    assert_eq!(result.un_safe, info(&blocks[7], &cfg));
    assert_eq!(result.safe, info(&blocks[expected], &cfg));
    l1.assert_finished();
    l2.assert_finished();
}

#[derive(Debug, Clone, Copy)]
enum Path {
    ReplacedUnsafe,
    Conservative,
    Checkpoint,
}
#[derive(Debug, Clone, Copy)]
enum Invalid {
    EpochSkip,
    NonzeroEpoch,
    Sequence,
    ParentHash,
}

// Each traversal calls the production RPC client and decodes real L1-info deposits. A check
// that exists only in the conservative path cannot protect the retained-checkpoint path.
#[rstest::rstest]
#[case(Path::ReplacedUnsafe, Invalid::EpochSkip)]
#[case(Path::ReplacedUnsafe, Invalid::NonzeroEpoch)]
#[case(Path::ReplacedUnsafe, Invalid::Sequence)]
#[case(Path::Conservative, Invalid::EpochSkip)]
#[case(Path::Conservative, Invalid::NonzeroEpoch)]
#[case(Path::Conservative, Invalid::Sequence)]
#[case(Path::Conservative, Invalid::ParentHash)]
#[case(Path::Checkpoint, Invalid::EpochSkip)]
#[case(Path::Checkpoint, Invalid::NonzeroEpoch)]
#[case(Path::Checkpoint, Invalid::Sequence)]
#[case(Path::Checkpoint, Invalid::ParentHash)]
#[tokio::test]
async fn rejects_inconsistent_edges_on_every_path(#[case] path: Path, #[case] invalid: Invalid) {
    let cfg = RollupConfig::default();
    let finalized = test_l2_block(1, 99, 0, B256::ZERO);
    let parent = test_l2_block(2, 100, 0, finalized.header.hash);
    let (origin, seq) = match invalid {
        Invalid::EpochSkip => (102, 0),
        Invalid::NonzeroEpoch => (101, 1),
        Invalid::Sequence => (100, 2),
        Invalid::ParentHash => (101, 0),
    };
    let child = test_l2_block(3, origin, seq, parent.header.hash);
    let mut l1_origin = l1_block(origin);
    if matches!(invalid, Invalid::ParentHash) {
        l1_origin.header.inner.parent_hash = B256::repeat_byte(0xff);
    }
    if matches!(path, Path::ReplacedUnsafe) {
        l1_origin.header.hash = B256::repeat_byte(0xff);
    }
    let (client, l1, l2) = test_engine_client(Arc::new(cfg.clone()));
    expect_heads(&l2, &finalized, &parent, &child);
    l1.expect_params("eth_getBlockByNumber", json!([format!("{origin:#x}"), false]), &l1_origin);
    let checkpoint = SafeCheckpoint {
        safe: info(&parent, &cfg),
        source: BlockInfo { number: 103, hash: l1_block(103).header.hash, ..Default::default() },
        replay_source: None,
    };
    if matches!(path, Path::Checkpoint) {
        l1.expect_params("eth_getBlockByNumber", json!(["latest", false]), l1_block(105));
        l1.expect_params(
            "eth_getBlockByNumber",
            json!([format!("{origin:#x}"), false]),
            &l1_origin,
        );
        l1.expect_params("eth_getBlockByNumber", json!(["0x67", false]), l1_block(103));
        l2.expect_params("eth_getBlockByNumber", json!(["0x2", false]), &parent);
        l2.expect_params("eth_getBlockByNumber", json!(["0x3", false]), &child);
    }
    expect_parent(&l2, &parent);
    let checkpoints = if matches!(path, Path::Checkpoint) { vec![checkpoint] } else { vec![] };
    let err =
        find_starting_forkchoice_with_checkpoints(&cfg, &client, &checkpoints).await.unwrap_err();
    assert!(
        match invalid {
            Invalid::EpochSkip | Invalid::ParentHash =>
                matches!(err, SyncStartError::L1OriginMismatch),
            Invalid::NonzeroEpoch => matches!(err, SyncStartError::NonZeroSequenceNumber),
            Invalid::Sequence => matches!(err, SyncStartError::InconsistentSequenceNumber),
        },
        "unexpected error: {err}"
    );
    l1.assert_finished();
    l2.assert_finished();
}

#[test]
fn epoch_checks_compare_full_ids_and_reject_overflow() {
    let mut parent = L2BlockInfo::default();
    let mut child = parent;
    child.l1_origin.hash = B256::repeat_byte(1); // Same height, different hash is not a new epoch.
    assert!(matches!(
        check_epoch_and_sequence(child, parent, None),
        Err(SyncStartError::L1OriginMismatch)
    ));
    parent.l1_origin.number = u64::MAX;
    assert!(matches!(
        check_epoch_and_sequence(child, parent, None),
        Err(SyncStartError::L1OriginMismatch)
    ));
    child = parent;
    parent.seq_num = u64::MAX;
    assert!(matches!(
        check_epoch_and_sequence(child, parent, None),
        Err(SyncStartError::InconsistentSequenceNumber)
    ));
}

#[test]
fn alt_da_extends_sync_lookback() {
    let mut cfg = RollupConfig { seq_window_size: 2, ..Default::default() };
    cfg.alt_da_config = Some(kona_genesis::AltDAConfig {
        da_challenge_address: Some(alloy_primitives::Address::repeat_byte(1)),
        da_challenge_window: Some(3),
        da_resolve_window: Some(4),
        ..Default::default()
    });
    assert_eq!(sync_lookback(&cfg), 7);
    cfg.seq_window_size = 10;
    assert_eq!(sync_lookback(&cfg), 10);
    cfg.seq_window_size = 2;
    cfg.alt_da_config = None;
    assert_eq!(sync_lookback(&cfg), 2);
}

#[tokio::test]
async fn missing_l1_parent_retries_from_a_fresh_chain_view() {
    let cfg = RollupConfig::default();
    let (client, l1, l2) = test_engine_client(Arc::new(cfg.clone()));
    let mut walker = SyncWalker::new(&cfg, &client);
    let mut head = L2BlockInfo::default();
    head.l1_origin.number = 101;
    head.l1_origin.hash = l1_block(101).header.hash;
    l1.expect_params("eth_getBlockByNumber", json!(["0x65", false]), l1_block(101));
    assert_eq!(walker.origin(head).await.unwrap(), OriginStatus::Canonical);
    head.l1_origin.number = 100;
    head.l1_origin.hash = l1_block(100).header.hash;
    l1.expect_params(
        "eth_getBlockByHash",
        json!([head.l1_origin.hash, false]),
        Option::<L1Block>::None,
    );
    let err = walker.origin(head).await.unwrap_err();
    assert!(matches!(err, SyncStartError::ChainViewChanged));
    assert!(crate::EngineResetError::SyncStart(err).is_temporary());
    l1.assert_finished();
    l2.assert_finished();
}

#[rstest::rstest]
#[case::ahead(false)]
#[case::disappeared(true)]
#[tokio::test]
async fn checkpoint_accepts_future_unsafe_origin_but_retries_a_disappeared_one(
    #[case] disappeared: bool,
) {
    let cfg = RollupConfig::default();
    let finalized = test_l2_block(1, 99, 0, B256::ZERO);
    let safe = test_l2_block(2, 100, 0, finalized.header.hash);
    let head = test_l2_block(3, 101, 0, safe.header.hash);
    let (client, l1, l2) = test_engine_client(Arc::new(cfg.clone()));
    expect_heads(&l2, &finalized, &safe, &head);
    l1.expect_params("eth_getBlockByNumber", json!(["0x65", false]), Option::<L1Block>::None);
    l1.expect_params(
        "eth_getBlockByNumber",
        json!(["latest", false]),
        l1_block(if disappeared { 101 } else { 100 }),
    );
    l1.expect_params("eth_getBlockByNumber", json!(["0x65", false]), Option::<L1Block>::None);
    if !disappeared {
        l1.expect_params("eth_getBlockByNumber", json!(["0x64", false]), l1_block(100));
        for (number, block) in [(2, &safe), (3, &head)] {
            l2.expect_params("eth_getBlockByNumber", json!([format!("{number:#x}"), false]), block);
        }
        expect_parent(&l2, &safe);
        l2.expect_params(
            "engine_forkchoiceUpdatedV3",
            json!([{
            "headBlockHash": head.header.hash, "safeBlockHash": safe.header.hash,
            "finalizedBlockHash": finalized.header.hash
        }, null]),
            alloy_rpc_types_engine::ForkchoiceUpdated::new(
                alloy_rpc_types_engine::PayloadStatus::from_status(
                    alloy_rpc_types_engine::PayloadStatusEnum::Valid,
                ),
            ),
        );
        l1.expect_params("eth_getBlockByNumber", json!(["0x64", false]), l1_block(100));
    }
    let checkpoint = SafeCheckpoint {
        safe: info(&safe, &cfg),
        source: BlockInfo { number: 100, hash: l1_block(100).header.hash, ..Default::default() },
        replay_source: None,
    };
    let result = find_starting_forkchoice_with_checkpoints(&cfg, &client, &[checkpoint]).await;
    if disappeared {
        assert!(matches!(result, Err(SyncStartError::ChainViewChanged)));
    } else {
        let result = result.unwrap();
        assert_eq!(result.un_safe, info(&head, &cfg));
        assert_eq!(result.safe, checkpoint.safe);
    }
    l1.assert_finished();
    l2.assert_finished();
}

#[tokio::test]
async fn rejects_reorgs_deeper_than_five_lookback_windows() {
    let mut blocks = vec![];
    for number in 0..9 {
        let parent = blocks.last().map_or(B256::ZERO, |b: &L2Block| b.header.hash);
        blocks.push(test_l2_block(number, number, 0, parent));
    }
    let mut cfg = RollupConfig { seq_window_size: 1, ..Default::default() };
    cfg.genesis.l2.hash = blocks[0].header.hash;
    let (client, l1, l2) = test_engine_client(Arc::new(cfg.clone()));
    expect_heads(&l2, &blocks[0], &blocks[0], &blocks[8]);
    for number in (2..=8).rev() {
        let mut replacement = l1_block(number);
        replacement.header.hash = B256::with_last_byte(number as u8 + 100);
        replacement.header.inner.parent_hash = B256::with_last_byte(number as u8 + 99);
        l1.expect_params(
            "eth_getBlockByNumber",
            json!([format!("{number:#x}"), false]),
            replacement,
        );
    }
    for parent in (2..8).rev() {
        expect_parent(&l2, &blocks[parent]);
    }
    let err = find_starting_forkchoice(&cfg, &client).await.unwrap_err();
    assert!(matches!(err, SyncStartError::ReorgTooDeep));
    assert!(!crate::EngineResetError::SyncStart(err).is_temporary());
    l1.assert_finished();
    l2.assert_finished();
}

#[tokio::test]
async fn checkpoint_is_retried_after_an_ahead_suffix_hides_a_replaced_origin() {
    let cfg = RollupConfig { seq_window_size: 2, ..Default::default() };
    let mut blocks = vec![];
    for number in 1..=5 {
        let parent = blocks.last().map_or(B256::ZERO, |b: &L2Block| b.header.hash);
        blocks.push(test_l2_block(number, number + 97, 0, parent));
    }
    let (client, l1, l2) = test_engine_client(Arc::new(cfg.clone()));
    expect_heads(&l2, &blocks[0], &blocks[1], &blocks[4]);
    let mut replacement = l1_block(100);
    replacement.header.hash = B256::repeat_byte(0xff);
    let missing = Option::<L1Block>::None;
    l1.expect_params("eth_getBlockByNumber", json!(["0x66", false]), missing.as_ref());
    l1.expect_params("eth_getBlockByNumber", json!(["latest", false]), &replacement);
    l1.expect_params("eth_getBlockByNumber", json!(["0x66", false]), missing.as_ref());
    l1.expect_params("eth_getBlockByNumber", json!(["0x63", false]), l1_block(99));
    for block in [&blocks[1], &blocks[4]] {
        l2.expect_params(
            "eth_getBlockByNumber",
            json!([format!("{:#x}", block.header.number), false]),
            block,
        );
    }
    // The first probe and then the conservative walk both encounter the hidden replacement.
    for pass in 0..2 {
        if pass == 1 {
            l1.expect_params("eth_getBlockByNumber", json!(["0x66", false]), missing.as_ref());
        }
        for parent in [&blocks[3], &blocks[2]] {
            expect_parent(&l2, parent);
        }
        l1.expect_params("eth_getBlockByNumber", json!(["0x65", false]), missing.as_ref());
        l1.expect_params("eth_getBlockByNumber", json!(["0x64", false]), &replacement);
    }
    expect_parent(&l2, &blocks[1]);
    l1.expect_params("eth_getBlockByHash", json!([l1_block(99).header.hash, false]), l1_block(99));
    l1.expect_params("eth_getBlockByNumber", json!(["latest", false]), &replacement);
    for _ in 0..2 {
        l1.expect_params("eth_getBlockByNumber", json!(["0x63", false]), l1_block(99));
    }
    for _ in 0..2 {
        l2.expect_params("eth_getBlockByNumber", json!(["0x2", false]), &blocks[1]);
    }
    l2.expect_params(
        "engine_forkchoiceUpdatedV3",
        json!([{
        "headBlockHash": blocks[1].header.hash, "safeBlockHash": blocks[1].header.hash,
        "finalizedBlockHash": blocks[0].header.hash
    }, null]),
        alloy_rpc_types_engine::ForkchoiceUpdated::new(
            alloy_rpc_types_engine::PayloadStatus::from_status(
                alloy_rpc_types_engine::PayloadStatusEnum::Valid,
            ),
        ),
    );
    l1.expect_params("eth_getBlockByNumber", json!(["0x64", false]), replacement);
    let safe = info(&blocks[1], &cfg);
    let checkpoint = SafeCheckpoint {
        safe,
        source: BlockInfo { number: 99, hash: l1_block(99).header.hash, ..Default::default() },
        replay_source: None,
    };
    let result =
        find_starting_forkchoice_with_checkpoints(&cfg, &client, &[checkpoint]).await.unwrap();
    assert_eq!(result.un_safe, safe);
    assert_eq!(result.safe, safe);
    l1.assert_finished();
    l2.assert_finished();
}
