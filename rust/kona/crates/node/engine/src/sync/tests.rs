use super::*;
use crate::EngineResetError;
use alloy_eips::BlockNumHash;
use alloy_transport::TransportErrorKind;
use kona_genesis::AltDAConfig;
use std::{
    collections::{BTreeMap, HashMap, VecDeque},
    sync::Mutex,
};

#[derive(Default)]
struct Chain {
    l1: BTreeMap<u64, BlockInfo>,
    l1_hashes: HashMap<B256, BlockInfo>,
    latest_l1: Option<BlockInfo>,
    l2: HashMap<B256, L2BlockInfo>,
    labels: HashMap<BlockNumberOrTag, L2BlockInfo>,
    l1_reads: Mutex<Vec<BlockId>>,
    l2_reads: Mutex<Vec<BlockId>>,
    number_responses: Mutex<HashMap<u64, VecDeque<Option<BlockInfo>>>>,
}

impl SyncSource for Chain {
    async fn l1_block(&self, id: BlockId) -> Result<Option<BlockInfo>, SyncStartError> {
        self.l1_reads.lock().unwrap().push(id);
        Ok(match id {
            BlockId::Hash(hash) => self.l1_hashes.get(&hash.block_hash).copied(),
            BlockId::Number(BlockNumberOrTag::Latest) => self.latest_l1,
            BlockId::Number(BlockNumberOrTag::Number(number)) => {
                if let Some(response) = self
                    .number_responses
                    .lock()
                    .unwrap()
                    .get_mut(&number)
                    .and_then(VecDeque::pop_front)
                {
                    return Ok(response);
                }
                self.l1.get(&number).copied()
            }
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

impl Chain {
    fn replace_l2(&mut self, block: L2BlockInfo) {
        self.l2.insert(block.block_info.hash, block);
        for label in self.labels.values_mut() {
            if label.block_info.hash == block.block_info.hash {
                *label = block;
            }
        }
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
            let block = BlockInfo {
                hash: hash(label, number as u64),
                number: number as u64,
                parent_hash,
                timestamp: number as u64,
            };
            parent_hash = block.hash;
            block
        })
        .collect()
}

struct Fixture {
    cfg: RollupConfig,
    source: Chain,
    blocks: Vec<L2BlockInfo>,
}

impl Fixture {
    fn new(l1: &str, l2: &str, canonical: &str, offset: usize, finalized: u8, safe: u8) -> Self {
        let old_l1 = l1_chain(l1);
        let canonical = l1_chain(canonical);
        let mut parent_hash = B256::ZERO;
        let blocks: Vec<_> = l2
            .bytes()
            .enumerate()
            .map(|(number, label)| {
                let block = L2BlockInfo {
                    block_info: BlockInfo {
                        hash: hash(label, number as u64),
                        number: number as u64,
                        parent_hash,
                        timestamp: number as u64,
                    },
                    l1_origin: old_l1[number + offset].id(),
                    seq_num: 0,
                };
                parent_hash = block.block_info.hash;
                block
            })
            .collect();
        let mut cfg = RollupConfig::default();
        cfg.genesis.l1 = old_l1[offset].id();
        cfg.genesis.l2 = blocks[0].block_info.id();
        cfg.seq_window_size = 2;
        let finalized =
            blocks.iter().find(|block| block.block_info.hash[0] == finalized).copied().unwrap();
        let safe = blocks.iter().find(|block| block.block_info.hash[0] == safe).copied().unwrap();
        let source = Chain {
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
                (BlockNumberOrTag::Finalized, finalized),
                (BlockNumberOrTag::Safe, safe),
                (BlockNumberOrTag::Latest, *blocks.last().unwrap()),
            ]),
            ..Default::default()
        };
        Self { cfg, source, blocks }
    }
}

#[derive(Debug)]
enum Expected {
    Heads { unsafe_head: u8, safe: u8 },
    WrongL1,
    WrongL2,
    TooDeep,
}

struct Case {
    name: &'static str,
    l1: &'static str,
    l2: &'static str,
    canonical: &'static str,
    finalized: u8,
    prior_safe: u8,
    origin_offset: usize,
    genesis_l1: u8,
    genesis_l2: u8,
    window: u64,
    expected: Expected,
}

impl Default for Case {
    fn default() -> Self {
        Self {
            name: "",
            l1: "",
            l2: "",
            canonical: "",
            finalized: b'A',
            prior_safe: b'A',
            origin_offset: 0,
            genesis_l1: b'a',
            genesis_l2: b'A',
            window: 2,
            expected: Expected::Heads { unsafe_head: b'A', safe: b'A' },
        }
    }
}

#[tokio::test]
async fn op_node_canonicality_vectors_with_kona_resume_policy() {
    // Canonicality/unsafe/error vectors from op-node/rollup/sync/start_test.go. Safe expectations
    // below are Kona's interim recent-window policy, not Go's prior-safe/epoch-parent policy.
    let cases = [
        Case {
            name: "already synced",
            l1: "ab",
            l2: "AB",
            canonical: "ab",
            expected: Expected::Heads { unsafe_head: b'B', safe: b'A' },
            ..Default::default()
        },
        Case {
            name: "safe after genesis",
            l1: "abcdefghijkj",
            l2: "ABCDEFGHIJKJ",
            canonical: "abcdefghijkj",
            finalized: b'B',
            prior_safe: b'D',
            expected: Expected::Heads { unsafe_head: b'J', safe: b'I' },
            ..Default::default()
        },
        Case {
            name: "small reorg long chain",
            l1: "abcdefgh",
            l2: "ABCDEFGH",
            canonical: "abcdefgx",
            finalized: b'B',
            prior_safe: b'H',
            expected: Expected::Heads { unsafe_head: b'G', safe: b'D' },
            ..Default::default()
        },
        Case {
            name: "L1 ahead",
            l1: "abcdef",
            l2: "ABCDE",
            canonical: "abcdef",
            prior_safe: b'D',
            expected: Expected::Heads { unsafe_head: b'E', safe: b'B' },
            ..Default::default()
        },
        Case {
            name: "L2 ahead after reorg",
            l1: "abcxyz",
            l2: "ABCXYZ",
            canonical: "abcx",
            finalized: b'B',
            prior_safe: b'X',
            expected: Expected::Heads { unsafe_head: b'Z', safe: b'B' },
            ..Default::default()
        },
        Case { name: "genesis", l1: "a", l2: "A", canonical: "a", ..Default::default() },
        Case {
            name: "reorg one step",
            l1: "abcdefg",
            l2: "ABCDEFG",
            canonical: "abcdefx",
            prior_safe: b'E',
            window: 3,
            expected: Expected::Heads { unsafe_head: b'F', safe: b'B' },
            ..Default::default()
        },
        Case {
            name: "reorg clips genesis",
            l1: "abc",
            l2: "ABC",
            canonical: "axy",
            prior_safe: b'B',
            ..Default::default()
        },
        Case {
            name: "reorg three steps",
            l1: "abcdefgh",
            l2: "ABCDEFGH",
            canonical: "abcdexyz",
            prior_safe: b'D',
            expected: Expected::Heads { unsafe_head: b'E', safe: b'B' },
            ..Default::default()
        },
        Case {
            name: "unexpected L1",
            l1: "abcdef",
            l2: "ABCDEF",
            canonical: "xyzwio",
            prior_safe: b'B',
            expected: Expected::WrongL1,
            ..Default::default()
        },
        Case {
            name: "unexpected L2",
            l1: "abcdef",
            l2: "ABCDEF",
            canonical: "xyzwio",
            prior_safe: b'B',
            genesis_l2: b'X',
            expected: Expected::WrongL2,
            ..Default::default()
        },
        Case {
            name: "offset genesis",
            l1: "abcdefghi",
            l2: "DEFGHI",
            canonical: "abcdefghi",
            finalized: b'E',
            prior_safe: b'H',
            origin_offset: 3,
            genesis_l1: b'd',
            genesis_l2: b'D',
            expected: Expected::Heads { unsafe_head: b'I', safe: b'F' },
            ..Default::default()
        },
        Case {
            name: "offset genesis reorg",
            l1: "abcdefgh",
            l2: "DEFGH",
            canonical: "abcdxyzw",
            finalized: b'D',
            prior_safe: b'D',
            origin_offset: 3,
            genesis_l1: b'd',
            genesis_l2: b'D',
            expected: Expected::Heads { unsafe_head: b'D', safe: b'D' },
            ..Default::default()
        },
        Case {
            name: "reorg past offset genesis",
            l1: "abcdefgh",
            l2: "DEFGH",
            canonical: "abxyzwio",
            finalized: b'D',
            prior_safe: b'D',
            origin_offset: 3,
            genesis_l1: b'd',
            genesis_l2: b'D',
            expected: Expected::WrongL1,
            ..Default::default()
        },
        Case {
            name: "prior safe does not force a long traversal",
            l1: "abcdefgh",
            l2: "ABCDEFGH",
            canonical: "abcdefgx",
            finalized: b'B',
            prior_safe: b'B',
            window: 1,
            expected: Expected::Heads { unsafe_head: b'G', safe: b'E' },
            ..Default::default()
        },
        Case {
            name: "reorg too deep",
            l1: "abcdefgh",
            l2: "ABCDEFGH",
            canonical: "abijklmn",
            finalized: b'B',
            prior_safe: b'B',
            window: 1,
            expected: Expected::TooDeep,
            ..Default::default()
        },
    ];
    for case in cases {
        let mut fixture = Fixture::new(
            case.l1,
            case.l2,
            case.canonical,
            case.origin_offset,
            case.finalized,
            case.prior_safe,
        );
        fixture.cfg.genesis.l1.hash = hash(case.genesis_l1, case.origin_offset as u64);
        fixture.cfg.genesis.l2.hash = hash(case.genesis_l2, 0);
        fixture.cfg.seq_window_size = case.window;
        let result = find_heads(&fixture.cfg, &fixture.source).await;
        match case.expected {
            Expected::Heads { unsafe_head, safe } => {
                let result = result.unwrap_or_else(|error| panic!("{}: {error}", case.name));
                assert_eq!(result.un_safe.block_info.hash[0], unsafe_head, "{} unsafe", case.name);
                assert_eq!(result.safe.block_info.hash[0], safe, "{} safe", case.name);
                assert_eq!(result.finalized, fixture.source.labels[&BlockNumberOrTag::Finalized]);
                assert!(result.safe.block_info.number <= result.un_safe.block_info.number);
                assert!(result.safe.block_info.number >= result.finalized.block_info.number);
            }
            expected => {
                let error = result.expect_err(case.name);
                assert!(
                    match expected {
                        Expected::WrongL1 =>
                            matches!(&error, SyncStartError::InvalidL1GenesisHash(..)),
                        Expected::WrongL2 =>
                            matches!(&error, SyncStartError::InvalidL2GenesisHash(..)),
                        Expected::TooDeep => matches!(&error, SyncStartError::ReorgTooDeep),
                        Expected::Heads { .. } => unreachable!(),
                    },
                    "{}: {error}",
                    case.name
                );
                assert!(!EngineResetError::SyncStart(error).is_temporary());
            }
        }
    }
}

#[tokio::test]
async fn genesis_safety_labels_keep_the_recent_window_walk_bounded_after_el_sync() {
    const OFFSET: u64 = 10_000_000;
    let mut fixture = Fixture::new("abcdefgh", "ABCDEFGH", "abcdefgh", 0, b'A', b'A');
    let mut canonical: Vec<_> = fixture.source.l1.values().copied().collect();
    for block in canonical.iter_mut().skip(1) {
        block.number += OFFSET;
    }
    // Only a recent tail is available; its first parent belongs to the omitted older history.
    canonical[1].parent_hash = hash(b'?', OFFSET);
    fixture.source.l1 = canonical.iter().map(|block| (block.number, *block)).collect();
    fixture.source.l1_hashes = canonical.iter().map(|block| (block.hash, *block)).collect();
    fixture.source.latest_l1 = canonical.last().copied();
    fixture.source.l2.clear();
    for (index, block) in fixture.blocks.iter_mut().enumerate() {
        if index > 0 {
            block.block_info.number += OFFSET;
            block.l1_origin.number += OFFSET;
        }
        if index == 1 {
            block.block_info.parent_hash = hash(b'?', OFFSET);
        }
        // The recent-window boundary is inside an epoch, not its first block or its parent.
        if index >= 4 {
            block.l1_origin = fixture.source.l1[&(OFFSET + index as u64 - 1)].id();
            block.seq_num = u64::from(index == 4);
        }
        fixture.source.replace_l2(*block);
    }
    let result = find_heads(&fixture.cfg, &fixture.source).await.unwrap();
    assert_eq!(result.un_safe, fixture.blocks[7]);
    assert_eq!(result.safe, fixture.blocks[4]);
    assert_eq!(result.safe.seq_num, 1);
    assert_eq!(result.finalized, fixture.blocks[0]);
    assert!(result.safe.block_info.number > OFFSET);
    // Three initial labels and three recent parents; no walk through millions of old blocks.
    let reads = fixture.source.l2_reads.lock().unwrap();
    assert_eq!(reads.len(), 6);
    assert!(!reads.contains(&fixture.cfg.genesis.l2.hash.into()));
}

#[tokio::test]
async fn retained_orphan_hash_does_not_prove_canonicality() {
    let fixture = Fixture::new("abcd", "ABCD", "abcx", 0, b'A', b'C');
    let orphan = fixture.blocks[3].l1_origin.hash;
    assert!(fixture.source.l1_hashes.contains_key(&orphan));
    let result = find_heads(&fixture.cfg, &fixture.source).await.unwrap();
    assert_eq!(result.un_safe, fixture.blocks[2]);
    let reads = fixture.source.l1_reads.lock().unwrap();
    assert_eq!(reads[0], BlockId::latest());
    assert_eq!(reads[1], BlockId::number(3));
    assert!(!reads.contains(&orphan.into()));
    // The numeric anchor is freshly checked even though it also exists in the per-walk cursor.
    assert_eq!(reads.last(), Some(&BlockId::number(3)));
}

#[tokio::test]
async fn missing_parent_is_retryable_without_being_wrong_chain() {
    let mut fixture = Fixture::new("abc", "ABC", "abc", 0, b'A', b'A');
    fixture.source.l2.remove(&fixture.blocks[1].block_info.hash);
    let error = find_heads(&fixture.cfg, &fixture.source).await.unwrap_err();
    assert!(
        matches!(&error, SyncStartError::BlockNotFound(id) if *id == fixture.blocks[1].block_info.hash.into())
    );
    assert!(EngineResetError::SyncStart(error).is_temporary());
}

#[tokio::test]
async fn historical_l1_hole_is_retryable_but_origins_beyond_captured_visibility_are_plausible() {
    let mut fixture = Fixture::new("abcd", "ABCD", "abcd", 0, b'A', b'A');
    fixture.source.latest_l1 = fixture.source.l1.get(&1).copied();
    let result = find_heads(&fixture.cfg, &fixture.source).await.unwrap();
    assert_eq!(result.un_safe, fixture.blocks[3]);
    assert_eq!(result.safe, fixture.blocks[0]);
    // Blocks2/3 may become visible during a walk; they cannot complete the captured L1 window.
    let reads = fixture.source.l1_reads.lock().unwrap().clone();
    assert!(!reads.contains(&BlockId::number(2)));
    assert!(!reads.contains(&BlockId::number(3)));
    assert_eq!(reads.iter().filter(|id| **id == BlockId::latest()).count(), 1);

    fixture.source.latest_l1 = fixture.source.l1.get(&3).copied();
    fixture.source.l1.remove(&3);
    let error = find_heads(&fixture.cfg, &fixture.source).await.unwrap_err();
    assert!(matches!(&error, SyncStartError::BlockNotFound(id) if *id == BlockId::number(3)));
    assert!(EngineResetError::SyncStart(error).is_temporary());
}

#[rstest::rstest]
#[case::changed(false)]
#[case::missing(true)]
#[tokio::test]
async fn changed_or_missing_anchor_retries_the_whole_walk(#[case] missing: bool) {
    let fixture = Fixture::new("abcd", "ABCD", "abcd", 0, b'A', b'C');
    let anchor = fixture.source.l1[&3];
    let changed = BlockInfo { hash: hash(b'x', 3), ..anchor };
    fixture
        .source
        .number_responses
        .lock()
        .unwrap()
        .insert(3, VecDeque::from([Some(anchor), if missing { None } else { Some(changed) }]));
    let error = find_heads(&fixture.cfg, &fixture.source).await.unwrap_err();
    assert!(matches!(&error, SyncStartError::L1ChainChanged));
    assert!(EngineResetError::SyncStart(error).is_temporary());
}

#[tokio::test]
async fn changed_anchor_does_not_turn_a_mixed_view_into_a_fatal_genesis_error() {
    let fixture = Fixture::new("abc", "ABC", "xyz", 0, b'A', b'A');
    let anchor = fixture.source.l1[&2];
    fixture.source.number_responses.lock().unwrap().insert(
        2,
        VecDeque::from([Some(anchor), Some(BlockInfo { hash: hash(b'c', 2), ..anchor })]),
    );
    assert!(matches!(
        find_heads(&fixture.cfg, &fixture.source).await,
        Err(SyncStartError::L1ChainChanged)
    ));
}

#[tokio::test]
async fn repeated_origins_preserve_sequence_and_cache_the_l1_cursor() {
    let mut fixture = Fixture::new("abcde", "ABCDE", "abcde", 0, b'A', b'C');
    let mut c = fixture.blocks[2];
    c.l1_origin = fixture.blocks[1].l1_origin;
    c.seq_num = 1;
    fixture.source.replace_l2(c);
    let mut d = fixture.blocks[3];
    d.l1_origin = BlockNumHash { number: 2, hash: hash(b'c', 2) };
    fixture.source.replace_l2(d);
    let mut e = fixture.blocks[4];
    e.l1_origin = BlockNumHash { number: 3, hash: hash(b'd', 3) };
    fixture.source.replace_l2(e);
    fixture.cfg.seq_window_size = 2;
    let result = find_heads(&fixture.cfg, &fixture.source).await.unwrap();
    assert_eq!(result.safe, fixture.blocks[0]);
    assert_eq!(
        fixture
            .source
            .l1_reads
            .lock()
            .unwrap()
            .iter()
            .filter(|id| **id == hash(b'b', 1).into())
            .count(),
        1
    );

    c.seq_num = 2;
    fixture.source.replace_l2(c);
    assert!(matches!(
        find_heads(&fixture.cfg, &fixture.source).await,
        Err(SyncStartError::InconsistentSequenceNumber)
    ));
}

#[tokio::test]
async fn changed_origin_requires_sequence_zero_and_consecutive_origin_height() {
    let mut fixture = Fixture::new("abc", "ABC", "abc", 0, b'A', b'A');
    let mut head = fixture.blocks[2];
    head.seq_num = 1;
    fixture.source.replace_l2(head);
    assert!(matches!(
        find_heads(&fixture.cfg, &fixture.source).await,
        Err(SyncStartError::NonZeroSequenceNumber)
    ));
    head.seq_num = 0;
    head.l1_origin.number = 4;
    fixture.source.replace_l2(head);
    assert!(matches!(
        find_heads(&fixture.cfg, &fixture.source).await,
        Err(SyncStartError::L1OriginMismatch)
    ));
}

#[tokio::test]
async fn malformed_parent_hash_and_height_are_not_dependency_failures() {
    let mut fixture = Fixture::new("abc", "ABC", "abc", 0, b'A', b'A');
    let parent_hash = fixture.blocks[1].block_info.hash;
    let mut parent = fixture.blocks[1];
    parent.block_info.hash = hash(b'x', 1);
    fixture.source.l2.insert(parent_hash, parent);
    let error = find_heads(&fixture.cfg, &fixture.source).await.unwrap_err();
    assert!(matches!(&error, SyncStartError::InvalidL2Response(_)));
    assert!(!EngineResetError::SyncStart(error).is_temporary());
    parent.block_info.hash = parent_hash;
    parent.block_info.number = 0;
    fixture.source.l2.insert(parent_hash, parent);
    assert!(matches!(
        find_heads(&fixture.cfg, &fixture.source).await,
        Err(SyncStartError::InvalidL2Parent)
    ));
}

#[tokio::test]
async fn malformed_l1_parent_and_missing_l1_parent_have_distinct_classification() {
    let mut fixture = Fixture::new("abc", "ABC", "abc", 0, b'A', b'A');
    let parent_hash = hash(b'b', 1);
    let parent = fixture.source.l1_hashes[&parent_hash];
    fixture.source.l1_hashes.remove(&parent_hash);
    let error = find_heads(&fixture.cfg, &fixture.source).await.unwrap_err();
    assert!(matches!(&error, SyncStartError::BlockNotFound(_)));
    assert!(EngineResetError::SyncStart(error).is_temporary());
    fixture.source.l1_hashes.insert(parent_hash, BlockInfo { number: 0, ..parent });
    assert!(matches!(
        find_heads(&fixture.cfg, &fixture.source).await,
        Err(SyncStartError::InvalidL1Parent)
    ));
    fixture.source.l1_hashes.insert(parent_hash, BlockInfo { hash: hash(b'x', 1), ..parent });
    assert!(matches!(
        find_heads(&fixture.cfg, &fixture.source).await,
        Err(SyncStartError::InvalidL1Response(_))
    ));
}

#[tokio::test]
async fn finalized_hash_and_origin_cannot_be_reorged() {
    let mut fixture = Fixture::new("abc", "ABC", "abc", 0, b'B', b'B');
    fixture.source.labels.get_mut(&BlockNumberOrTag::Finalized).unwrap().block_info.hash =
        hash(b'x', 1);
    assert!(matches!(
        find_heads(&fixture.cfg, &fixture.source).await,
        Err(SyncStartError::MismatchedFinalizedBlock(..))
    ));
    let fixture = Fixture::new("abc", "ABC", "axy", 0, b'B', b'B');
    assert!(matches!(
        find_heads(&fixture.cfg, &fixture.source).await,
        Err(SyncStartError::FinalizedL1OriginMismatch)
    ));
}

#[tokio::test]
async fn selected_safe_boundary_checks_finalized_and_genesis_hashes() {
    let mut fixture = Fixture::new("abcdef", "ABCDEF", "abcdef", 0, b'B', b'C');
    fixture.source.labels.get_mut(&BlockNumberOrTag::Finalized).unwrap().block_info.hash =
        hash(b'x', 1);
    fixture.cfg.seq_window_size = 3;
    assert!(matches!(
        find_heads(&fixture.cfg, &fixture.source).await,
        Err(SyncStartError::MismatchedFinalizedBlock(..))
    ));
    let mut fixture = Fixture::new("abcd", "ABCD", "abcd", 0, b'A', b'B');
    fixture.cfg.seq_window_size = 2;
    fixture.cfg.genesis.l2.hash = hash(b'x', 0);
    assert!(matches!(
        find_heads(&fixture.cfg, &fixture.source).await,
        Err(SyncStartError::InvalidL2GenesisHash(..))
    ));
}

#[tokio::test]
async fn invalid_initial_head_order_is_not_el_sync_recovery_policy() {
    let mut fixture = Fixture::new("abc", "ABC", "abc", 0, b'C', b'B');
    assert!(matches!(
        find_heads(&fixture.cfg, &fixture.source).await,
        Err(SyncStartError::InvalidHeadOrder)
    ));
    fixture.source.labels.insert(BlockNumberOrTag::Safe, fixture.blocks[2]);
    fixture.source.labels.insert(BlockNumberOrTag::Latest, fixture.blocks[1]);
    assert!(matches!(
        find_heads(&fixture.cfg, &fixture.source).await,
        Err(SyncStartError::InvalidHeadOrder)
    ));
}

#[tokio::test]
async fn alt_da_lookback_uses_l1_blocks_and_extends_reorg_limit() {
    let mut fixture = Fixture::new("abcdefgh", "ABCDEFGH", "abijklmn", 0, b'B', b'B');
    fixture.cfg.seq_window_size = 1;
    assert!(matches!(
        find_heads(&fixture.cfg, &fixture.source).await,
        Err(SyncStartError::ReorgTooDeep)
    ));
    fixture.cfg.alt_da_config = Some(AltDAConfig {
        da_challenge_window: Some(1),
        da_resolve_window: Some(1),
        ..Default::default()
    });
    let result = find_heads(&fixture.cfg, &fixture.source).await.unwrap();
    assert_eq!(result.un_safe, fixture.blocks[1]);
    assert_eq!(result.safe, fixture.blocks[1]);
}

#[tokio::test]
async fn reorg_at_exactly_five_lookback_windows_is_allowed() {
    let mut fixture = Fixture::new("abcdefg", "ABCDEFG", "abijklm", 0, b'B', b'B');
    fixture.cfg.seq_window_size = 1;
    let result = find_heads(&fixture.cfg, &fixture.source).await.unwrap();
    assert_eq!(result.un_safe, fixture.blocks[1]);
}

#[tokio::test]
async fn absent_labels_fall_back_but_missing_latest_and_genesis_are_retryable() {
    let mut fixture = Fixture::new("a", "A", "a", 0, b'A', b'A');
    fixture.source.labels.remove(&BlockNumberOrTag::Finalized);
    fixture.source.labels.remove(&BlockNumberOrTag::Safe);
    let result = find_heads(&fixture.cfg, &fixture.source).await.unwrap();
    assert_eq!(result.finalized, fixture.blocks[0]);
    assert_eq!(result.safe, fixture.blocks[0]);
    fixture.source.labels.remove(&BlockNumberOrTag::Latest);
    let error = find_heads(&fixture.cfg, &fixture.source).await.unwrap_err();
    assert!(matches!(&error, SyncStartError::BlockNotFound(id) if *id == BlockId::latest()));
    assert!(EngineResetError::SyncStart(error).is_temporary());
    fixture.source.labels.insert(BlockNumberOrTag::Latest, fixture.blocks[0]);
    fixture.source.l2.clear();
    assert!(matches!(
        find_heads(&fixture.cfg, &fixture.source).await,
        Err(SyncStartError::BlockNotFound(_))
    ));
}

#[tokio::test]
async fn nonzero_l2_genesis_number_is_respected() {
    let mut fixture = Fixture::new("def", "DEF", "def", 0, b'D', b'D');
    let mut blocks = fixture.blocks.clone();
    fixture.source.l2.clear();
    for block in &mut blocks {
        block.block_info.number += 10;
        fixture.source.replace_l2(*block);
    }
    fixture.cfg.genesis.l2.number = 10;
    fixture.source.labels.remove(&BlockNumberOrTag::Finalized);
    fixture.source.labels.remove(&BlockNumberOrTag::Safe);
    let result = find_heads(&fixture.cfg, &fixture.source).await.unwrap();
    assert_eq!(result.safe, blocks[0]);
    assert_eq!(result.finalized, blocks[0]);
}

#[test]
fn rpc_dependency_errors_remain_retryable() {
    let error = SyncStartError::RpcError(TransportErrorKind::custom_str("dependency unavailable"));
    assert!(EngineResetError::SyncStart(error).is_temporary());
}

struct RpcCall {
    method: &'static str,
    params: serde_json::Value,
    response: serde_json::Value,
}

impl RpcCall {
    fn result(
        method: &'static str,
        params: serde_json::Value,
        result: impl serde::Serialize,
    ) -> Self {
        Self { method, params, response: serde_json::json!({ "jsonrpc": "2.0", "result": result }) }
    }

    fn error(method: &'static str, params: serde_json::Value, message: &'static str) -> Self {
        Self {
            method,
            params,
            response: serde_json::json!({ "jsonrpc": "2.0",
            "error": { "code": -32000, "message": message } }),
        }
    }
}

struct RpcScript(std::sync::Arc<Mutex<VecDeque<RpcCall>>>);

impl RpcScript {
    fn new(calls: impl IntoIterator<Item = RpcCall>) -> Self {
        Self(std::sync::Arc::new(Mutex::new(calls.into_iter().collect())))
    }

    fn provider<N: alloy_network::Network>(&self) -> alloy_provider::RootProvider<N> {
        let calls = self.0.clone();
        let transport = tower::service_fn(move |request: alloy_json_rpc::RequestPacket| -> alloy_transport::TransportFut<'static> {
            let calls = calls.clone();
            Box::pin(async move {
                let alloy_json_rpc::RequestPacket::Single(request) = request else { panic!("unexpected batch") };
                let call = calls.lock().unwrap().pop_front().expect("unexpected RPC call");
                assert_eq!(request.method(), call.method);
                let params: serde_json::Value = serde_json::from_str(request.params().unwrap().get()).unwrap();
                assert_eq!(params, call.params, "{} params", call.method);
                let mut response = call.response;
                response["id"] = serde_json::to_value(request.id()).unwrap();
                Ok(serde_json::from_value::<alloy_json_rpc::ResponsePacket>(response).unwrap())
            })
        });
        alloy_provider::RootProvider::new(alloy_rpc_client::RpcClient::new(transport, false))
    }

    fn assert_finished(&self) {
        assert!(self.0.lock().unwrap().is_empty(), "expected RPC calls were not made");
    }
}

fn rpc_genesis() -> alloy_rpc_types_eth::Block<op_alloy_rpc_types::Transaction> {
    let header = alloy_consensus::Header { number: 10, ..Default::default() };
    alloy_rpc_types_eth::Block::new(
        alloy_rpc_types_eth::Header::new(header),
        alloy_rpc_types_eth::BlockTransactions::Full(vec![]),
    )
}

#[tokio::test]
async fn rpc_boundary_uses_numeric_canonical_anchor_and_the_callers_genesis() {
    use alloy_rpc_types_eth::{Block, BlockTransactions, Header};
    use serde_json::json;
    let l1_genesis = Block::<alloy_rpc_types_eth::Transaction>::new(
        Header::new(alloy_consensus::Header::default()),
        BlockTransactions::Hashes(vec![]),
    );
    let canonical_l1 = Block::<alloy_rpc_types_eth::Transaction>::new(
        Header::new(alloy_consensus::Header {
            number: 1,
            parent_hash: l1_genesis.header.hash,
            ..Default::default()
        }),
        BlockTransactions::Hashes(vec![]),
    );
    let orphan_hash = hash(b'x', 1);
    let genesis = rpc_genesis();
    let mut cfg = RollupConfig::default();
    cfg.genesis.l1 = BlockNumHash { number: 0, hash: l1_genesis.header.hash };
    cfg.genesis.l2 = BlockNumHash { number: 10, hash: genesis.header.hash };
    cfg.seq_window_size = 2;
    let info = kona_protocol::L1BlockInfoBedrock::new_from_number_and_block_hash(1, orphan_hash);
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
        RpcCall::result("eth_getBlockByNumber", json!(["0x1", false]), &canonical_l1),
        RpcCall::result("eth_getBlockByHash", json!([l1_genesis.header.hash, false]), &l1_genesis),
        RpcCall::result("eth_getBlockByNumber", json!(["0x1", false]), &canonical_l1),
    ]);
    let l2 = RpcScript::new([
        RpcCall::result("eth_getBlockByNumber", json!(["finalized", true]), &genesis),
        RpcCall::result("eth_getBlockByNumber", json!(["safe", true]), &genesis),
        RpcCall::result("eth_getBlockByNumber", json!(["latest", true]), unsafe_block),
        RpcCall::result("eth_getBlockByHash", json!([genesis.header.hash, true]), &genesis),
    ]);
    // The explicit config argument governs decoding, even if the client was built differently.
    let client = crate::OpEngineClient::new(
        l1.provider::<alloy_network::Ethereum>(),
        l2.provider::<op_alloy_network::Optimism>(),
        std::sync::Arc::new(RollupConfig::default()),
    );
    let result = find_starting_forkchoice(&cfg, &client).await.unwrap();
    assert_eq!(result.un_safe.block_info.id(), cfg.genesis.l2);
    assert_eq!(result.safe, result.un_safe);
    assert_eq!(result.finalized, result.un_safe);
    l1.assert_finished();
    l2.assert_finished();
}

#[rstest::rstest]
#[case::null(None)]
#[case::geth(Some("block not found"))]
#[case::erigon(Some("Unknown block"))]
#[tokio::test]
async fn rpc_missing_safety_labels_fall_back_to_the_configured_genesis(
    #[case] message: Option<&'static str>,
) {
    use serde_json::json;
    let genesis = rpc_genesis();
    let mut cfg = RollupConfig::default();
    cfg.genesis.l2 = BlockNumHash { hash: genesis.header.hash, number: 10 };
    let missing = |tag| {
        message.map_or_else(
            || RpcCall::result("eth_getBlockByNumber", json!([tag, true]), serde_json::Value::Null),
            |message| RpcCall::error("eth_getBlockByNumber", json!([tag, true]), message),
        )
    };
    let l1 = RpcScript::new([]);
    let l2 = RpcScript::new([
        missing("finalized"),
        RpcCall::result("eth_getBlockByNumber", json!(["0xa", true]), &genesis),
        missing("safe"),
        RpcCall::result("eth_getBlockByNumber", json!(["latest", true]), &genesis),
    ]);
    let client = crate::OpEngineClient::new(
        l1.provider::<alloy_network::Ethereum>(),
        l2.provider::<op_alloy_network::Optimism>(),
        std::sync::Arc::new(cfg.clone()),
    );
    let result = L2ForkchoiceState::current(&cfg, &client).await.unwrap();
    assert_eq!(result.finalized.block_info.id(), cfg.genesis.l2);
    assert_eq!(result.safe, result.finalized);
    l1.assert_finished();
    l2.assert_finished();
}

#[tokio::test]
async fn rpc_failure_does_not_masquerade_as_an_unset_finalized_label() {
    let l1 = RpcScript::new([]);
    let l2 = RpcScript::new([RpcCall::error(
        "eth_getBlockByNumber",
        serde_json::json!(["finalized", true]),
        "dependency unavailable",
    )]);
    let cfg = std::sync::Arc::new(RollupConfig::default());
    let client = crate::OpEngineClient::new(
        l1.provider::<alloy_network::Ethereum>(),
        l2.provider::<op_alloy_network::Optimism>(),
        cfg.clone(),
    );
    let error = L2ForkchoiceState::current(&cfg, &client).await.unwrap_err();
    assert!(matches!(&error, SyncStartError::RpcError(_)));
    assert!(EngineResetError::SyncStart(error).is_temporary());
    l1.assert_finished();
    l2.assert_finished();
}

#[rstest::rstest]
#[case::latest(BlockId::latest(), "eth_getBlockByNumber", serde_json::json!(["latest", true]))]
#[case::number(BlockId::number(10), "eth_getBlockByNumber", serde_json::json!(["0xa", true]))]
#[case::hash(hash(b'A', 0).into(), "eth_getBlockByHash", serde_json::json!([hash(b'A', 0), true]))]
#[tokio::test]
async fn not_found_compatibility_is_limited_to_safety_labels(
    #[case] id: BlockId,
    #[case] method: &'static str,
    #[case] params: serde_json::Value,
    #[values("block not found", "Unknown block")] message: &'static str,
) {
    let l1 = RpcScript::new([]);
    let l2 = RpcScript::new([RpcCall::error(method, params, message)]);
    let cfg = std::sync::Arc::new(RollupConfig::default());
    let client = crate::OpEngineClient::new(
        l1.provider::<alloy_network::Ethereum>(),
        l2.provider::<op_alloy_network::Optimism>(),
        cfg.clone(),
    );
    let source = EngineSyncSource { client: &client, cfg: &cfg };
    let error = source.l2_block(id).await.unwrap_err();
    assert!(matches!(&error, SyncStartError::RpcError(_)));
    assert!(EngineResetError::SyncStart(error).is_temporary());
    l1.assert_finished();
    l2.assert_finished();
}

#[tokio::test]
async fn alt_da_window_also_moves_safe_back_by_l1_blocks() {
    let mut fixture = Fixture::new("abcdefgh", "ABCDEFGH", "abcdefgh", 0, b'A', b'H');
    fixture.cfg.seq_window_size = 1;
    let result = find_heads(&fixture.cfg, &fixture.source).await.unwrap();
    assert_eq!(result.safe, fixture.blocks[5]);
    fixture.cfg.alt_da_config = Some(AltDAConfig {
        da_challenge_window: Some(1),
        da_resolve_window: Some(2),
        ..Default::default()
    });
    let result = find_heads(&fixture.cfg, &fixture.source).await.unwrap();
    assert_eq!(result.safe, fixture.blocks[3]);
    fixture.cfg.alt_da_config.as_mut().unwrap().da_resolve_window = Some(u64::MAX);
    let result = find_heads(&fixture.cfg, &fixture.source).await.unwrap();
    assert_eq!(result.safe, fixture.blocks[0]);
}
