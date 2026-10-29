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

// Old chains with missing labels must recover in constant time; recent chains still derive
// from genesis. Non-zero genesis heights exercise the threshold relative to chain genesis.
#[rstest::rstest]
#[case::genesis(0, false)]
#[case::recent_chain(1, false)]
#[case::at_recovery_threshold(28, false)]
#[case::past_recovery_threshold(29, false)]
#[case::distant_unsafe_head(100_000_000, false)]
#[case::missing_labels(100_000_000, true)]
#[tokio::test]
async fn recovers_el_sync_with_genesis_labels(#[case] distance: u64, #[case] missing_labels: bool) {
    let genesis = test_l2_block(100, 50, 0, B256::ZERO);
    let mut cfg = RollupConfig { seq_window_size: 2, ..Default::default() };
    cfg.genesis.l1 = alloy_eips::BlockNumHash::new(50, l1_block(50).header.hash);
    cfg.genesis.l2 = alloy_eips::BlockNumHash::new(100, genesis.header.hash);
    let mut blocks = vec![genesis.clone()];
    if distance <= 28 {
        for i in 1..=distance {
            blocks.push(test_l2_block(100 + i, 50 + i, 0, blocks.last().unwrap().header.hash));
        }
    } else {
        blocks.push(test_l2_block(100 + distance, 50 + distance, 0, B256::ZERO));
    }
    let head = blocks.last().unwrap();
    let (client, l1, l2) = test_engine_client(Arc::new(cfg.clone()));
    if missing_labels {
        l2.expect_params(
            "eth_getBlockByNumber",
            json!(["finalized", true]),
            Option::<L2Block>::None,
        );
        l2.expect_params("eth_getBlockByNumber", json!(["0x64", true]), &genesis);
        l2.expect_params("eth_getBlockByNumber", json!(["safe", true]), Option::<L2Block>::None);
        l2.expect_params("eth_getBlockByNumber", json!(["latest", true]), head);
    } else {
        expect_heads(&l2, &genesis, &genesis, head);
    }
    if distance <= 28 {
        l1.expect_params(
            "eth_getBlockByNumber",
            json!([format!("{:#x}", 50 + distance), false]),
            l1_block(50 + distance),
        );
        for i in (0..distance).rev() {
            expect_parent(&l2, &blocks[i as usize]);
            l1.expect_params(
                "eth_getBlockByHash",
                json!([l1_block(50 + i).header.hash, false]),
                l1_block(50 + i),
            );
        }
    }
    let result = find_starting_forkchoice(&cfg, &client).await.unwrap();
    assert_eq!(result.un_safe, info(head, &cfg));
    let derived = if distance > 28 { head } else { &genesis };
    assert_eq!(result.safe, info(derived, &cfg));
    assert_eq!(result.finalized, info(derived, &cfg));
    l1.assert_finished();
    l2.assert_finished();
}

#[rstest::rstest]
#[case::behind_safe(0, 20)]
#[case::behind_finalized(20, 17)]
#[case::behind_finalized_and_safe(20, 21)]
#[tokio::test]
async fn recovers_rewound_el_labels(#[case] finalized_number: u64, #[case] safe_number: u64) {
    let mut cfg = RollupConfig::default();
    let finalized = test_l2_block(finalized_number, 100, 0, B256::ZERO);
    if finalized_number == 0 {
        cfg.genesis.l2.hash = finalized.header.hash;
    }
    let safe = test_l2_block(safe_number, 101, 0, B256::ZERO);
    let head = test_l2_block(18, 99, 0, B256::ZERO);
    let (client, l1, l2) = test_engine_client(Arc::new(cfg.clone()));
    expect_heads(&l2, &finalized, &safe, &head);
    let result = find_starting_forkchoice(&cfg, &client).await.unwrap();
    let expected = info(&head, &cfg);
    assert_eq!(
        result,
        L2ForkchoiceState { un_safe: expected, safe: expected, finalized: expected }
    );
    l1.assert_finished();
    l2.assert_finished();
}

#[rstest::rstest]
#[case::only_safe_established(0)]
#[case::finality_established(1)]
#[tokio::test]
async fn established_labels_keep_the_normal_recovery_walk(#[case] finalized: usize) {
    let mut blocks = vec![];
    for number in 0..30 {
        let parent = blocks.last().map_or(B256::ZERO, |block: &L2Block| block.header.hash);
        blocks.push(test_l2_block(number, number, 0, parent));
    }
    let mut cfg = RollupConfig { seq_window_size: 2, ..Default::default() };
    cfg.genesis.l2.hash = blocks[0].header.hash;
    let (client, l1, l2) = test_engine_client(Arc::new(cfg.clone()));
    expect_heads(&l2, &blocks[finalized], &blocks[1], &blocks[29]);
    l1.expect_params("eth_getBlockByNumber", json!(["0x1d", false]), l1_block(29));
    for number in (1..29).rev() {
        l1.expect_params(
            "eth_getBlockByHash",
            json!([l1_block(number).header.hash, false]),
            l1_block(number),
        );
    }
    for parent in (finalized..29).rev() {
        expect_parent(&l2, &blocks[parent]);
    }
    let result = find_starting_forkchoice(&cfg, &client).await.unwrap();
    assert_eq!(result.un_safe, info(&blocks[29], &cfg));
    assert_eq!(result.safe, info(&blocks[finalized], &cfg));
    assert_eq!(result.finalized, info(&blocks[finalized], &cfg));
    l1.assert_finished();
    l2.assert_finished();
}

#[tokio::test]
async fn missing_l2_parent_can_be_retried() {
    let cfg = RollupConfig::default();
    let finalized = test_l2_block(1, 99, 0, B256::ZERO);
    let head = test_l2_block(2, 100, 0, finalized.header.hash);
    let (client, l1, l2) = test_engine_client(Arc::new(cfg.clone()));
    expect_heads(&l2, &finalized, &finalized, &head);
    l1.expect_params("eth_getBlockByNumber", json!(["0x64", false]), l1_block(100));
    l2.expect_params(
        "eth_getBlockByHash",
        json!([finalized.header.hash, true]),
        Option::<L2Block>::None,
    );
    let err = find_starting_forkchoice(&cfg, &client).await.unwrap_err();
    assert!(matches!(err, SyncStartError::BlockNotFound(_)));
    assert!(crate::EngineResetError::SyncStart(err).is_temporary());

    expect_heads(&l2, &finalized, &finalized, &head);
    l1.expect_params("eth_getBlockByNumber", json!(["0x64", false]), l1_block(100));
    expect_parent(&l2, &finalized);
    l1.expect_params("eth_getBlockByHash", json!([l1_block(99).header.hash, false]), l1_block(99));
    let result = find_starting_forkchoice(&cfg, &client).await.unwrap();
    assert_eq!(result.un_safe, info(&head, &cfg));
    assert_eq!(result.safe, info(&finalized, &cfg));
    assert_eq!(result.finalized, info(&finalized, &cfg));
    l1.assert_finished();
    l2.assert_finished();
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
}
#[derive(Debug, Clone, Copy)]
enum Invalid {
    EpochSkip,
    NonzeroEpoch,
    Sequence,
    ParentHash,
}

// Both traversal paths call the production RPC client and decode real L1-info deposits.
#[rstest::rstest]
#[case(Path::ReplacedUnsafe, Invalid::EpochSkip)]
#[case(Path::ReplacedUnsafe, Invalid::NonzeroEpoch)]
#[case(Path::ReplacedUnsafe, Invalid::Sequence)]
#[case(Path::Conservative, Invalid::EpochSkip)]
#[case(Path::Conservative, Invalid::NonzeroEpoch)]
#[case(Path::Conservative, Invalid::Sequence)]
#[case(Path::Conservative, Invalid::ParentHash)]
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
    expect_parent(&l2, &parent);
    let err = find_starting_forkchoice(&cfg, &client).await.unwrap_err();
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
