use super::*;
use crate::{EngineResetError, OpEngineClient};
use alloy_eips::BlockNumHash;
use alloy_json_rpc::{RequestPacket, ResponsePacket};
use alloy_network::Ethereum;
use alloy_provider::RootProvider;
use alloy_rpc_client::RpcClient;
use alloy_rpc_types_eth::{Block, BlockTransactions, Header};
use alloy_transport::{TransportFut, mock::Asserter};
use kona_genesis::AltDAConfig;
use op_alloy_network::Optimism;
use serde_json::{Value, json};
use std::{
    collections::{BTreeMap, HashMap, VecDeque},
    sync::{Arc, Mutex},
};

#[derive(Default)]
struct Chain {
    cfg: RollupConfig,
    blocks: Vec<L2BlockInfo>,
    l1: BTreeMap<u64, BlockInfo>,
    l1_hashes: HashMap<B256, BlockInfo>,
    latest_l1: Option<BlockInfo>,
    l2: HashMap<B256, L2BlockInfo>,
    labels: HashMap<BlockNumberOrTag, L2BlockInfo>,
    l1_reads: Mutex<Vec<BlockId>>,
    l2_reads: Mutex<Vec<BlockId>>,
    number_responses: Mutex<VecDeque<Option<BlockInfo>>>,
}

impl SyncSource for Chain {
    async fn l1_block(&self, id: BlockId) -> Result<Option<BlockInfo>, SyncStartError> {
        self.l1_reads.lock().unwrap().push(id);
        Ok(match id {
            BlockId::Hash(hash) => self.l1_hashes.get(&hash.block_hash).copied(),
            BlockId::Number(BlockNumberOrTag::Latest) => self.latest_l1,
            BlockId::Number(BlockNumberOrTag::Number(number)) => self
                .number_responses
                .lock()
                .unwrap()
                .pop_front()
                .unwrap_or_else(|| self.l1.get(&number).copied()),
            _ => panic!("unexpected L1 request {id}"),
        })
    }

    async fn l2_block(&self, id: BlockId) -> Result<Option<L2BlockInfo>, SyncStartError> {
        self.l2_reads.lock().unwrap().push(id);
        Ok(match id {
            BlockId::Hash(hash) => self.l2.get(&hash.block_hash).copied(),
            BlockId::Number(BlockNumberOrTag::Number(number)) => {
                self.l2.values().find(|block| block.block_info.number == number).copied()
            }
            BlockId::Number(tag) => self.labels.get(&tag).copied(),
        })
    }
}

fn hash(label: u8, number: u64) -> B256 {
    let mut bytes = [0; 32];
    bytes[0] = label;
    bytes[1..9].copy_from_slice(&number.to_be_bytes());
    B256::from(bytes)
}

fn l1_chain(labels: &str) -> Vec<BlockInfo> {
    let mut parent_hash = B256::ZERO;
    labels
        .bytes()
        .enumerate()
        .map(|(number, label)| {
            let block = BlockInfo::new(
                hash(label, number as u64),
                number as u64,
                parent_hash,
                number as u64,
            );
            parent_hash = block.hash;
            block
        })
        .collect()
}

impl Chain {
    fn new(old: &str, canonical: &str) -> Self {
        let old_l1 = l1_chain(old);
        let canonical = l1_chain(canonical);
        let blocks: Vec<_> = l1_chain(&old.to_ascii_uppercase())
            .into_iter()
            .zip(&old_l1)
            .map(|(block_info, origin)| L2BlockInfo {
                block_info,
                l1_origin: origin.id(),
                seq_num: 0,
            })
            .collect();
        let mut cfg = RollupConfig::default();
        cfg.genesis.l1 = old_l1[0].id();
        cfg.genesis.l2 = blocks[0].block_info.id();
        cfg.seq_window_size = 2;
        Self {
            cfg,
            // Retain the old fork by hash: existence alone must never prove canonicality.
            l1_hashes: old_l1
                .into_iter()
                .chain(canonical.iter().copied())
                .map(|block| (block.hash, block))
                .collect(),
            l1: canonical.iter().map(|block| (block.number, *block)).collect(),
            latest_l1: canonical.last().copied(),
            l2: blocks.iter().map(|block| (block.block_info.hash, *block)).collect(),
            labels: HashMap::from([
                (BlockNumberOrTag::Finalized, blocks[0]),
                (BlockNumberOrTag::Safe, blocks[0]),
                (BlockNumberOrTag::Latest, *blocks.last().unwrap()),
            ]),
            blocks,
            ..Default::default()
        }
    }

    fn replace_l2(&mut self, block: L2BlockInfo) {
        self.l2.insert(block.block_info.hash, block);
        for label in self.labels.values_mut() {
            if label.block_info.hash == block.block_info.hash {
                *label = block;
            }
        }
    }
}

#[tokio::test]
async fn genesis_safety_labels_keep_the_recent_window_walk_bounded_after_el_sync() {
    const OFFSET: u64 = 10_000_000;
    let mut chain = Chain::new("abcdefgh", "abcdefgh");
    let mut canonical: Vec<_> = chain.l1.values().copied().collect();
    for block in canonical.iter_mut().skip(1) {
        block.number += OFFSET;
    }
    // Only a recent tail is available; its first parent belongs to the omitted older history.
    canonical[1].parent_hash = hash(b'?', OFFSET);
    chain.l1 = canonical.iter().map(|block| (block.number, *block)).collect();
    chain.l1_hashes = canonical.iter().map(|block| (block.hash, *block)).collect();
    chain.latest_l1 = canonical.last().copied();
    chain.l2.clear();
    for index in 0..chain.blocks.len() {
        let mut block = chain.blocks[index];
        if index > 0 {
            block.block_info.number += OFFSET;
            block.l1_origin.number += OFFSET;
        }
        if index == 1 {
            block.block_info.parent_hash = hash(b'?', OFFSET);
        }
        // The recent-window boundary is inside an epoch, not its first block or its parent.
        if index >= 4 {
            block.l1_origin = chain.l1[&(OFFSET + index as u64 - 1)].id();
            block.seq_num = u64::from(index == 4);
        }
        chain.blocks[index] = block;
        chain.replace_l2(block);
    }
    let result = find_heads(&chain.cfg, &chain).await.unwrap();
    assert_eq!(result.un_safe, chain.blocks[7]);
    assert_eq!(result.safe, chain.blocks[4]);
    assert_eq!(result.safe.seq_num, 1);
    assert_eq!(result.finalized, chain.blocks[0]);
    assert!(result.safe.block_info.number > OFFSET);
    // Three initial labels and three recent parents; no walk through millions of old blocks.
    let reads = chain.l2_reads.lock().unwrap();
    assert_eq!(reads.len(), 6);
    assert!(!reads.contains(&chain.cfg.genesis.l2.hash.into()));
}

#[tokio::test]
async fn retained_orphan_hash_does_not_prove_canonicality() {
    let chain = Chain::new("abcdef", "abcx");
    let orphan = chain.blocks[3].l1_origin.hash;
    assert!(chain.l1_hashes.contains_key(&orphan));
    let result = find_heads(&chain.cfg, &chain).await.unwrap();
    assert_eq!(result.un_safe, chain.blocks[2]);
    let reads = chain.l1_reads.lock().unwrap();
    assert_eq!(reads[0], BlockId::latest());
    assert_eq!(reads[1], BlockId::number(3));
    assert!(!reads.contains(&orphan.into()));
    // The numeric anchor is freshly checked even though it also exists in the per-walk cursor.
    assert_eq!(reads.last(), Some(&BlockId::number(3)));
}

#[tokio::test]
async fn missing_parent_is_retryable_without_being_wrong_chain() {
    let mut chain = Chain::new("abc", "abc");
    chain.l2.remove(&chain.blocks[1].block_info.hash);
    let error = find_heads(&chain.cfg, &chain).await.unwrap_err();
    assert!(
        matches!(&error, SyncStartError::BlockNotFound(id) if *id == chain.blocks[1].block_info.hash.into())
    );
    assert!(EngineResetError::SyncStart(error).is_temporary());
}

#[tokio::test]
async fn historical_l1_hole_is_retryable_but_origins_beyond_captured_visibility_are_plausible() {
    let mut chain = Chain::new("abcd", "abcd");
    chain.latest_l1 = chain.l1.get(&1).copied();
    let result = find_heads(&chain.cfg, &chain).await.unwrap();
    assert_eq!(result.un_safe, chain.blocks[3]);
    assert_eq!(result.safe, chain.blocks[0]);
    // Blocks2/3 may become visible during a walk; they cannot complete the captured L1 window.
    let reads = chain.l1_reads.lock().unwrap().clone();
    assert!(!reads.contains(&BlockId::number(2)));
    assert!(!reads.contains(&BlockId::number(3)));
    assert_eq!(reads.iter().filter(|id| **id == BlockId::latest()).count(), 1);

    chain.latest_l1 = chain.l1.get(&3).copied();
    chain.l1.remove(&3);
    let error = find_heads(&chain.cfg, &chain).await.unwrap_err();
    assert!(matches!(&error, SyncStartError::BlockNotFound(id) if *id == BlockId::number(3)));
    assert!(EngineResetError::SyncStart(error).is_temporary());
}

#[rstest::rstest]
#[case::changed(false)]
#[case::missing(true)]
#[tokio::test]
async fn changed_or_missing_anchor_retries_the_whole_walk(#[case] missing: bool) {
    let chain = Chain::new("abcd", "abcd");
    let anchor = chain.l1[&3];
    let changed = BlockInfo { hash: hash(b'x', 3), ..anchor };
    *chain.number_responses.lock().unwrap() =
        VecDeque::from([Some(anchor), if missing { None } else { Some(changed) }]);
    let error = find_heads(&chain.cfg, &chain).await.unwrap_err();
    assert!(matches!(&error, SyncStartError::L1ChainChanged));
    assert!(EngineResetError::SyncStart(error).is_temporary());
}

#[tokio::test]
async fn changed_anchor_does_not_turn_a_mixed_view_into_a_fatal_genesis_error() {
    let chain = Chain::new("abc", "xyz");
    let anchor = chain.l1[&2];
    *chain.number_responses.lock().unwrap() =
        VecDeque::from([Some(anchor), Some(BlockInfo { hash: hash(b'c', 2), ..anchor })]);
    assert!(matches!(find_heads(&chain.cfg, &chain).await, Err(SyncStartError::L1ChainChanged)));
}

#[tokio::test]
async fn repeated_origins_preserve_sequence_and_cache_the_l1_cursor() {
    let mut chain = Chain::new("abc", "abc");
    let mut head = chain.blocks[2];
    head.l1_origin = chain.blocks[1].l1_origin;
    head.seq_num = 1;
    chain.replace_l2(head);
    let result = find_heads(&chain.cfg, &chain).await.unwrap();
    assert_eq!(result.safe, chain.blocks[0]);
    assert_eq!(
        *chain.l1_reads.lock().unwrap(),
        [BlockId::latest(), BlockId::number(1), hash(b'a', 0).into(), BlockId::number(1),]
    );
    head.seq_num = 2;
    chain.replace_l2(head);
    assert!(matches!(
        find_heads(&chain.cfg, &chain).await,
        Err(SyncStartError::InconsistentSequenceNumber)
    ));
}

#[tokio::test]
async fn changed_origin_requires_sequence_zero_and_consecutive_origin_height() {
    let mut chain = Chain::new("abc", "abc");
    let mut head = chain.blocks[2];
    head.seq_num = 1;
    chain.replace_l2(head);
    assert!(matches!(
        find_heads(&chain.cfg, &chain).await,
        Err(SyncStartError::NonZeroSequenceNumber)
    ));
    head.seq_num = 0;
    head.l1_origin.number = 4;
    chain.replace_l2(head);
    assert!(matches!(find_heads(&chain.cfg, &chain).await, Err(SyncStartError::L1OriginMismatch)));
}

#[tokio::test]
async fn malformed_parent_hash_and_height_are_not_dependency_failures() {
    let mut chain = Chain::new("abc", "abc");
    let parent_hash = chain.blocks[1].block_info.hash;
    let mut parent = chain.blocks[1];
    parent.block_info.hash = hash(b'x', 1);
    chain.l2.insert(parent_hash, parent);
    let error = find_heads(&chain.cfg, &chain).await.unwrap_err();
    assert!(matches!(&error, SyncStartError::InvalidL2Response(_)));
    parent.block_info.hash = parent_hash;
    parent.block_info.number = 0;
    chain.l2.insert(parent_hash, parent);
    assert!(matches!(find_heads(&chain.cfg, &chain).await, Err(SyncStartError::InvalidL2Parent)));
}

#[tokio::test]
async fn malformed_l1_parent_and_missing_l1_parent_have_distinct_classification() {
    let mut chain = Chain::new("abc", "abc");
    let parent_hash = hash(b'b', 1);
    let parent = chain.l1_hashes[&parent_hash];
    chain.l1_hashes.remove(&parent_hash);
    let error = find_heads(&chain.cfg, &chain).await.unwrap_err();
    assert!(matches!(&error, SyncStartError::BlockNotFound(_)));
    chain.l1_hashes.insert(parent_hash, BlockInfo { number: 0, ..parent });
    assert!(matches!(find_heads(&chain.cfg, &chain).await, Err(SyncStartError::InvalidL1Parent)));
    chain.l1_hashes.insert(parent_hash, BlockInfo { hash: hash(b'x', 1), ..parent });
    assert!(matches!(
        find_heads(&chain.cfg, &chain).await,
        Err(SyncStartError::InvalidL1Response(_))
    ));
}

#[tokio::test]
async fn safety_labels_preserve_order_and_finalized_anchors() {
    let mut chain = Chain::new("abc", "abc");
    chain.labels.insert(
        BlockNumberOrTag::Finalized,
        L2BlockInfo {
            block_info: BlockInfo { hash: hash(b'x', 1), ..chain.blocks[1].block_info },
            ..chain.blocks[1]
        },
    );
    assert!(matches!(find_heads(&chain.cfg, &chain).await, Err(SyncStartError::InvalidHeadOrder)));
    chain.labels.insert(BlockNumberOrTag::Safe, chain.blocks[1]);
    chain.labels.insert(BlockNumberOrTag::Latest, chain.blocks[0]);
    assert!(matches!(find_heads(&chain.cfg, &chain).await, Err(SyncStartError::InvalidHeadOrder)));
    chain.labels.insert(BlockNumberOrTag::Latest, chain.blocks[2]);
    assert!(matches!(
        find_heads(&chain.cfg, &chain).await,
        Err(SyncStartError::MismatchedFinalizedBlock(..))
    ));
    let mut chain = Chain::new("abc", "axy");
    chain.labels.insert(BlockNumberOrTag::Finalized, chain.blocks[1]);
    chain.labels.insert(BlockNumberOrTag::Safe, chain.blocks[1]);
    assert!(matches!(
        find_heads(&chain.cfg, &chain).await,
        Err(SyncStartError::FinalizedL1OriginMismatch)
    ));
}

#[tokio::test]
async fn configured_genesis_hashes_must_match() {
    let mut chain = Chain::new("a", "x");
    assert!(matches!(
        find_heads(&chain.cfg, &chain).await,
        Err(SyncStartError::InvalidL1GenesisHash(..))
    ));
    chain.cfg.genesis.l2.hash = hash(b'X', 0);
    assert!(matches!(
        find_heads(&chain.cfg, &chain).await,
        Err(SyncStartError::InvalidL2GenesisHash(..))
    ));
}

#[rstest::rstest]
#[case::latest(BlockNumberOrTag::Latest)]
#[case::genesis(BlockNumberOrTag::Finalized)]
#[tokio::test]
async fn missing_required_l2_blocks_are_dependencies(#[case] label: BlockNumberOrTag) {
    let mut chain = Chain::new("a", "a");
    chain.labels.remove(&label);
    if label == BlockNumberOrTag::Finalized {
        chain.l2.clear();
    }
    assert!(matches!(
        L2ForkchoiceState::load(&chain.cfg, &chain).await,
        Err(SyncStartError::BlockNotFound(_))
    ));
}

#[tokio::test]
async fn alt_da_lookback_uses_l1_blocks_and_extends_reorg_limit() {
    let mut chain = Chain::new("abcdefgh", "abijklmn");
    chain.cfg.seq_window_size = 1;
    assert!(matches!(find_heads(&chain.cfg, &chain).await, Err(SyncStartError::ReorgTooDeep)));
    chain.cfg.alt_da_config = Some(AltDAConfig {
        da_challenge_window: Some(1),
        da_resolve_window: Some(1),
        ..Default::default()
    });
    let result = find_heads(&chain.cfg, &chain).await.unwrap();
    assert_eq!(result.un_safe, chain.blocks[1]);
}

#[tokio::test]
async fn reorg_at_exactly_five_lookback_windows_is_allowed() {
    let mut chain = Chain::new("abcdefg", "abijklm");
    chain.cfg.seq_window_size = 1;
    let result = find_heads(&chain.cfg, &chain).await.unwrap();
    assert_eq!(result.un_safe, chain.blocks[1]);
}

struct RpcCall {
    method: &'static str,
    params: Value,
    response: Value,
}

impl RpcCall {
    fn result(method: &'static str, params: Value, result: impl serde::Serialize) -> Self {
        Self { method, params, response: json!({ "jsonrpc": "2.0", "result": result }) }
    }
}

struct RpcScript(Arc<Mutex<VecDeque<RpcCall>>>);

impl RpcScript {
    fn new(calls: impl IntoIterator<Item = RpcCall>) -> Self {
        Self(Arc::new(Mutex::new(calls.into_iter().collect())))
    }

    fn provider<N: alloy_network::Network>(&self) -> RootProvider<N> {
        let calls = self.0.clone();
        let transport = tower::service_fn(move |request: RequestPacket| -> TransportFut<'static> {
            let calls = calls.clone();
            Box::pin(async move {
                let RequestPacket::Single(request) = request else { panic!("unexpected batch") };
                let call = calls.lock().unwrap().pop_front().expect("unexpected RPC call");
                assert_eq!(request.method(), call.method);
                let params: Value = serde_json::from_str(request.params().unwrap().get()).unwrap();
                assert_eq!(params, call.params, "{} params", call.method);
                let mut response = call.response;
                response["id"] = serde_json::to_value(request.id()).unwrap();
                Ok(serde_json::from_value::<ResponsePacket>(response).unwrap())
            })
        });
        RootProvider::new(RpcClient::new(transport, false))
    }

    fn assert_finished(&self) {
        assert!(self.0.lock().unwrap().is_empty(), "expected RPC calls were not made");
    }
}

fn rpc_genesis() -> Block<op_alloy_rpc_types::Transaction> {
    let header = alloy_consensus::Header { number: 10, ..Default::default() };
    Block::new(Header::new(header), BlockTransactions::Full(vec![]))
}

#[tokio::test]
async fn rpc_boundary_uses_numeric_canonical_anchor_and_the_callers_genesis() {
    let l1_genesis = Block::<alloy_rpc_types_eth::Transaction>::new(
        Header::new(alloy_consensus::Header { number: 20, ..Default::default() }),
        BlockTransactions::Hashes(vec![]),
    );
    let canonical_l1 = Block::<alloy_rpc_types_eth::Transaction>::new(
        Header::new(alloy_consensus::Header {
            number: 21,
            parent_hash: l1_genesis.header.hash,
            ..Default::default()
        }),
        BlockTransactions::Hashes(vec![]),
    );
    let orphan_hash = hash(b'x', 21);
    let genesis = rpc_genesis();
    let mut cfg = RollupConfig::default();
    cfg.genesis.l1 = BlockNumHash { number: 20, hash: l1_genesis.header.hash };
    cfg.genesis.l2 = BlockNumHash { number: 10, hash: genesis.header.hash };
    cfg.seq_window_size = 2;
    let info = kona_protocol::L1BlockInfoBedrock::new_from_number_and_block_hash(21, orphan_hash);
    let deposit = op_alloy_consensus::OpTxEnvelope::Deposit(alloy_primitives::Sealed::new(
        op_alloy_consensus::TxDeposit { input: info.encode_calldata(), ..Default::default() },
    ));
    let transaction = op_alloy_rpc_types::Transaction {
        inner: alloy_rpc_types_eth::Transaction {
            inner: alloy_consensus::transaction::Recovered::new_unchecked(
                deposit,
                Default::default(),
            ),
            block_hash: None,
            block_number: Some(11),
            effective_gas_price: Some(1),
            transaction_index: Some(0),
            block_timestamp: None,
        },
        deposit_nonce: None,
        deposit_receipt_version: None,
    };
    let unsafe_block = Block::new(
        Header::new(alloy_consensus::Header {
            number: 11,
            parent_hash: genesis.header.hash,
            ..Default::default()
        }),
        BlockTransactions::Full(vec![transaction]),
    );
    let l1 = RpcScript::new([
        RpcCall::result("eth_getBlockByNumber", json!(["latest", false]), &canonical_l1),
        RpcCall::result("eth_getBlockByNumber", json!(["0x15", false]), &canonical_l1),
        RpcCall::result("eth_getBlockByHash", json!([l1_genesis.header.hash, false]), &l1_genesis),
        RpcCall::result("eth_getBlockByNumber", json!(["0x15", false]), &canonical_l1),
    ]);
    let l2 = RpcScript::new([
        RpcCall::result("eth_getBlockByNumber", json!(["finalized", true]), &genesis),
        RpcCall::result("eth_getBlockByNumber", json!(["safe", true]), &genesis),
        RpcCall::result("eth_getBlockByNumber", json!(["latest", true]), unsafe_block),
        RpcCall::result("eth_getBlockByHash", json!([genesis.header.hash, true]), &genesis),
    ]);
    // The explicit config argument governs decoding, even if the client was built differently.
    let client = OpEngineClient::new(
        l1.provider::<Ethereum>(),
        l2.provider::<Optimism>(),
        Arc::new(RollupConfig::default()),
    );
    let result = find_starting_forkchoice(&cfg, &client).await.unwrap();
    assert_eq!(result.un_safe.block_info.id(), cfg.genesis.l2);
    assert_eq!(result.safe, result.un_safe);
    assert_eq!(result.finalized, result.un_safe);
    l1.assert_finished();
    l2.assert_finished();
}

fn mock_client(
    cfg: &RollupConfig,
    rpc: &Asserter,
) -> OpEngineClient<RootProvider, RootProvider<Optimism>> {
    OpEngineClient::new(
        RootProvider::new(RpcClient::mocked(Asserter::new())),
        RootProvider::new(RpcClient::mocked(rpc.clone())),
        Arc::new(cfg.clone()),
    )
}

#[rstest::rstest]
#[case::null(None)]
#[case::geth(Some("block not found"))]
#[case::erigon(Some("Unknown block"))]
#[tokio::test]
async fn unset_safety_labels_use_configured_genesis(#[case] message: Option<&'static str>) {
    let rpc = Asserter::new();
    let genesis = rpc_genesis();
    let mut cfg = RollupConfig::default();
    cfg.genesis.l2 = BlockNumHash { hash: genesis.header.hash, number: 10 };
    let missing = || match message {
        Some(message) => rpc.push_failure_msg(message),
        None => rpc.push_success(&Value::Null),
    };
    missing();
    rpc.push_success(&genesis);
    missing();
    rpc.push_success(&genesis);
    let result = L2ForkchoiceState::current(&cfg, &mock_client(&cfg, &rpc)).await.unwrap();
    assert_eq!(result.finalized.block_info.id(), cfg.genesis.l2);
    assert_eq!(result.safe, result.finalized);
    assert!(rpc.read_q().is_empty());
}

#[rstest::rstest]
#[case::finalized(BlockId::finalized(), "dependency unavailable")]
#[case::latest(BlockId::latest(), "block not found")]
#[case::number(BlockId::number(10), "Unknown block")]
#[case::hash(hash(b'A', 0).into(), "block not found")]
#[tokio::test]
async fn other_rpc_errors_do_not_receive_label_fallback(
    #[case] id: BlockId,
    #[case] message: &'static str,
) {
    let rpc = Asserter::new();
    rpc.push_failure_msg(message);
    let cfg = RollupConfig::default();
    let client = mock_client(&cfg, &rpc);
    let error = EngineSyncSource { client: &client, cfg: &cfg }.l2_block(id).await.unwrap_err();
    assert!(matches!(error, SyncStartError::RpcError(_)));
    assert!(rpc.read_q().is_empty());
}

#[tokio::test]
async fn alt_da_window_also_moves_safe_back_by_l1_blocks() {
    let mut chain = Chain::new("abcdefgh", "abcdefgh");
    chain.cfg.seq_window_size = 1;
    let result = find_heads(&chain.cfg, &chain).await.unwrap();
    assert_eq!(result.safe, chain.blocks[5]);
    chain.cfg.alt_da_config = Some(AltDAConfig {
        da_challenge_window: Some(1),
        da_resolve_window: Some(2),
        ..Default::default()
    });
    let result = find_heads(&chain.cfg, &chain).await.unwrap();
    assert_eq!(result.safe, chain.blocks[3]);
    chain.cfg.alt_da_config.as_mut().unwrap().da_resolve_window = Some(u64::MAX);
    let result = find_heads(&chain.cfg, &chain).await.unwrap();
    assert_eq!(result.safe, chain.blocks[0]);
}
