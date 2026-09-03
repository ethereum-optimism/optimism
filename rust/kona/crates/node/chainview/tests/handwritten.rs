//! The chain view's finality chain written by hand against the DBSP Rust API, checked against
//! the compiled SQL circuit on the same inputs.
//!
//! This is a measurement, not a shipping path: it is what `l2_safe_canonical`,
//! `finalized_candidate` and `finalized_l2` in `chainview.sql` look like without the compiler.
//! The differential tests feed identical rows to both circuits and require identical
//! `finalized_l2` output after every step.

use std::collections::BTreeMap;

use alloy_eips::BlockNumHash;
use alloy_primitives::B256;
use dbsp::{
    DBSPHandle, IndexedZSetReader, OutputHandle, RootCircuit, Runtime, ZSetHandle,
    circuit::CircuitConfig,
    operator::Max,
    typed_batch::OrdZSet,
    utils::{Tup0, Tup2, Tup4},
};
use kona_chainview::{
    L1StatusKind, L2SafeFact, L2StatusKind,
    facts::{l1_block_row, l1_status_row, l2_safe_row, l2_status_row},
    handles::{FinalizedL2Row, Handles, L1BlockRow, L1StatusRow, L2SafeRow, L2StatusRow},
};
use kona_protocol::{BlockInfo, L2BlockInfo};

/// The input handles of the hand-written circuit.
struct Inputs {
    l1_blocks: ZSetHandle<L1BlockRow>,
    l1_status: ZSetHandle<L1StatusRow>,
    l2_status: ZSetHandle<L2StatusRow>,
    l2_safe_blocks: ZSetHandle<L2SafeRow>,
}

// ---- hand-written circuit: begin ----
fn build_by_hand(
    circuit: &mut RootCircuit,
) -> Result<(Inputs, OutputHandle<OrdZSet<FinalizedL2Row>>), anyhow::Error> {
    let (l1_blocks, l1_blocks_in) = circuit.add_input_zset::<L1BlockRow>();
    let (l1_status, l1_status_in) = circuit.add_input_zset::<L1StatusRow>();
    let (l2_status, l2_status_in) = circuit.add_input_zset::<L2StatusRow>();
    let (safe, safe_in) = circuit.add_input_zset::<L2SafeRow>();

    // MAX(number) per status kind, keyed by the unit key so the singletons can be joined.
    // Not modelled: the SQL's COALESCE(.., -1); an absent status yields no row here.
    let finalized_l1 =
        l1_status.filter(|r| r.0.str() == "finalized").map_index(|r| (Tup0(), r.1)).aggregate(Max);
    let engine_safe =
        l2_status.filter(|r| r.0.str() == "safe").map_index(|r| (Tup0(), r.1)).aggregate(Max);
    let engine_finalized =
        l2_status.filter(|r| r.0.str() == "finalized").map_index(|r| (Tup0(), r.1)).aggregate(Max);

    // l2_safe_canonical: rows with no L1 block of another hash at their derived-from height.
    let safe_by_height = safe.map_index(|s| (s.8, s.clone()));
    let l1_hash_by_number = l1_blocks.map_index(|b| (b.0, b.1.clone()));
    let contradicted = safe_by_height
        .join(&l1_hash_by_number, |_, s, hash| Tup2::new(s.9 != *hash, s.clone()))
        .filter(|t| t.0)
        .map(|t| t.1.clone())
        .distinct();
    let canonical = safe.minus(&contradicted);

    // finalized_candidate: ARG_MAX(.., seq) over canonical rows derived from at or below
    // finalized L1 and at or below the engine's safe head.
    let below_finalized_l1 = canonical
        .map_index(|s| (Tup0(), s.clone()))
        .join(&finalized_l1, |_, s, f| Tup2::new(s.8 <= *f, s.clone()))
        .filter(|t| t.0)
        .map(|t| t.1.clone());
    let eligible = below_finalized_l1
        .map_index(|s| (Tup0(), s.clone()))
        .join(&engine_safe, |_, s, es| Tup2::new(s.1 <= *es, s.clone()))
        .filter(|t| t.0)
        .map(|t| t.1.clone());
    let candidate = eligible.map_index(|s| (Tup0(), Tup2::new(s.0, s.clone()))).aggregate(Max);

    // finalized_l2: the candidate, if above the engine's finalized head.
    let finalized_l2 = candidate
        .join(&engine_finalized, |_, c, ef| Tup2::new(c.1.1 > *ef, c.1.clone()))
        .filter(|t| t.0)
        .map(|t| Tup4::new(Some(t.1.1), Some(t.1.2.clone()), Some(t.1.8), Some(t.1.9.clone())));

    Ok((
        Inputs {
            l1_blocks: l1_blocks_in,
            l1_status: l1_status_in,
            l2_status: l2_status_in,
            l2_safe_blocks: safe_in,
        },
        finalized_l2.output(),
    ))
}
// ---- hand-written circuit: end ----

fn l1_hash(number: u64, fork: u8) -> B256 {
    let mut bytes = [0u8; 32];
    bytes[0] = fork;
    bytes[24..].copy_from_slice(&number.to_be_bytes());
    B256::from(bytes)
}

fn l1_on(number: u64, fork: u8) -> BlockInfo {
    BlockInfo {
        hash: l1_hash(number, fork),
        number,
        parent_hash: if number == 0 { B256::ZERO } else { l1_hash(number - 1, fork) },
        timestamp: 12 * number,
    }
}

fn l1(number: u64) -> BlockInfo {
    l1_on(number, 0)
}

fn l2(number: u64) -> L2BlockInfo {
    let mut bytes = [0xaau8; 32];
    bytes[24..].copy_from_slice(&number.to_be_bytes());
    L2BlockInfo {
        block_info: BlockInfo {
            hash: B256::from(bytes),
            number,
            parent_hash: B256::ZERO,
            timestamp: 2 * number,
        },
        l1_origin: BlockNumHash { number: number / 6, hash: l1_hash(number / 6, 0) },
        seq_num: number % 6,
    }
}

/// Both circuits, fed identically.
struct Pair {
    sql: DBSPHandle,
    handles: Handles,
    hand: DBSPHandle,
    inputs: Inputs,
    hand_out: OutputHandle<OrdZSet<FinalizedL2Row>>,
    sql_acc: BTreeMap<FinalizedL2Row, i64>,
    hand_acc: BTreeMap<FinalizedL2Row, i64>,
    heads: Vec<L2StatusRow>,
    seq: u64,
}

impl Pair {
    fn start() -> Self {
        let (sql, handles) = kona_chainview::build(CircuitConfig::with_workers(1)).expect("sql");
        let (hand, (inputs, hand_out)) =
            Runtime::init_circuit(CircuitConfig::with_workers(1), build_by_hand).expect("hand");
        Self {
            sql,
            handles,
            hand,
            inputs,
            hand_out,
            sql_acc: BTreeMap::new(),
            hand_acc: BTreeMap::new(),
            heads: Vec::new(),
            seq: 0,
        }
    }

    fn l1_blocks(&self, blocks: impl IntoIterator<Item = BlockInfo>, weight: i64) {
        for block in blocks {
            let row = l1_block_row(&block).expect("row");
            self.handles.l1_blocks.push(row.clone(), weight);
            self.inputs.l1_blocks.push(row, weight);
        }
    }

    fn finalized_l1(&self, block: BlockInfo) {
        let row = l1_status_row(L1StatusKind::Finalized, &block).expect("row");
        self.handles.l1_status.push(row.clone(), 1);
        self.inputs.l1_status.push(row, 1);
    }

    /// Replaces the engine's four head rows: unsafe = safe + 4, local-safe = safe.
    fn l2_heads(&mut self, safe: u64, finalized: u64) {
        for row in self.heads.drain(..) {
            self.handles.l2_status.push(row.clone(), -1);
            self.inputs.l2_status.push(row, -1);
        }
        let labels = [
            (L2StatusKind::Unsafe, safe + 4),
            (L2StatusKind::LocalSafe, safe),
            (L2StatusKind::Safe, safe),
            (L2StatusKind::Finalized, finalized),
        ];
        for (kind, number) in labels {
            let row = l2_status_row(kind, &l2(number)).expect("row");
            self.handles.l2_status.push(row.clone(), 1);
            self.inputs.l2_status.push(row.clone(), 1);
            self.heads.push(row);
        }
    }

    fn confirm(&mut self, l2_number: u64, derived_from: BlockInfo) -> L2SafeRow {
        self.seq += 1;
        let fact = L2SafeFact { seq: self.seq, block: l2(l2_number), derived_from };
        let row = l2_safe_row(&fact).expect("row");
        self.handles.l2_safe_blocks.push(row.clone(), 1);
        self.inputs.l2_safe_blocks.push(row.clone(), 1);
        row
    }

    fn retract(&self, row: L2SafeRow) {
        self.handles.l2_safe_blocks.push(row.clone(), -1);
        self.inputs.l2_safe_blocks.push(row, -1);
    }

    /// Steps both circuits, folds their deltas, and requires the integrated views to agree.
    /// Returns the finalized L2 number both name, if any.
    fn step(&mut self) -> Option<u64> {
        self.sql.transaction().expect("sql step");
        self.hand.transaction().expect("hand step");
        for snapshot in self.handles.finalized_l2.take_from_all() {
            for (row, (), weight) in snapshot.consolidate().iter() {
                *self.sql_acc.entry(row).or_insert(0) += weight;
            }
        }
        for batch in self.hand_out.take_from_all() {
            for (row, (), weight) in batch.iter() {
                *self.hand_acc.entry(row).or_insert(0) += weight;
            }
        }
        self.sql_acc.retain(|_, w| *w != 0);
        self.hand_acc.retain(|_, w| *w != 0);
        assert_eq!(self.sql_acc, self.hand_acc, "compiled and hand-written circuits disagree");
        assert!(self.sql_acc.len() <= 1, "finalized_l2 has at most one row");
        self.sql_acc.keys().next().and_then(|row| row.0).map(|n| n as u64)
    }
}

#[test]
fn finality_follows_derived_from_below_finalized_l1() {
    let mut p = Pair::start();
    p.l1_blocks((1..=20).map(l1), 1);
    p.l2_heads(30, 0);
    p.confirm(11, l1(5));
    p.confirm(12, l1(6));
    p.confirm(13, l1(7));
    assert_eq!(p.step(), None, "nothing is finalized before an L1 block is");
    p.finalized_l1(l1(6));
    assert_eq!(p.step(), Some(12));
    p.finalized_l1(l1(7));
    assert_eq!(p.step(), Some(13));
}

#[test]
fn finality_never_exceeds_the_engine_safe_head() {
    let mut p = Pair::start();
    p.l1_blocks((1..=20).map(l1), 1);
    p.l2_heads(11, 0);
    p.confirm(11, l1(5));
    p.confirm(12, l1(6));
    p.finalized_l1(l1(10));
    assert_eq!(p.step(), Some(11), "block 12 is above the engine safe head");
    p.l2_heads(12, 0);
    assert_eq!(p.step(), Some(12));
}

#[test]
fn an_l1_reorg_retracts_derived_blocks_on_the_dropped_branch() {
    let mut p = Pair::start();
    p.l1_blocks((1..=15).map(l1), 1);
    p.l2_heads(40, 0);
    p.confirm(20, l1(10));
    let on_dropped_branch = p.confirm(21, l1(12));
    p.finalized_l1(l1(11));
    assert_eq!(p.step(), Some(20));

    // Blocks 12..=15 are replaced by another branch; 21 was derived from the dropped one.
    p.l1_blocks((12..=15).map(l1), -1);
    p.l1_blocks((12..=15).map(|n| l1_on(n, 1)), 1);
    p.finalized_l1(l1_on(13, 1));
    assert_eq!(p.step(), Some(20), "21 is contradicted by the new block at height 12");

    // The pipeline re-derives 21 from the replacement block; the old row is retracted.
    p.retract(on_dropped_branch);
    p.confirm(21, l1_on(12, 1));
    assert_eq!(p.step(), Some(21));
}

#[test]
fn a_pipeline_reset_retracts_blocks_above_the_reset_head() {
    let mut p = Pair::start();
    p.l1_blocks((1..=20).map(l1), 1);
    p.l2_heads(30, 0);
    let rows: Vec<_> = (11..=14).map(|n| p.confirm(n, l1(n - 6))).collect();
    p.finalized_l1(l1(8));
    assert_eq!(p.step(), Some(14));

    // Reset to safe head 12: rows above it are retracted; finality below it keeps its progress.
    for row in rows.into_iter().skip(2) {
        p.retract(row);
    }
    p.l2_heads(12, 0);
    assert_eq!(p.step(), Some(12));
}
