//! The circuit thread.
//!
//! [`spawn`] builds the circuit on a dedicated OS thread and runs a loop that drains queued
//! messages, pushes facts through the typed input handles, runs one transaction, integrates
//! every view's delta into host-side state, publishes a [`ChainViewSnapshot`], and answers
//! queued queries from the integrated state. Queries are answered after the step so a query
//! sent after a fact observes that fact.
//!
//! `dbsp` handle calls block, so nothing here runs on a tokio worker: the tokio side talks to
//! the thread through channels only.

use std::{
    collections::{BTreeMap, HashMap, hash_map::Entry},
    hash::Hash,
    panic::{AssertUnwindSafe, catch_unwind},
    sync::mpsc as std_mpsc,
    thread,
    time::Instant,
};

use alloy_eips::BlockNumHash;
use alloy_primitives::{Address, B256};
use dbsp::{DBData, IndexedZSetReader, circuit::CircuitConfig};
use kona_protocol::BlockInfo;
use tokio::sync::{mpsc, oneshot, watch};

use crate::{
    client::{ChainViewClient, ChainViewQuery, Msg},
    facts::{
        Fact, FactError, ImportedL2Block, L1StatusKind, L2Heads, L2SafeFact, L2StatusKind,
        address_from_bytes, hash_from_bytes, l1_block_row, l1_status_row, l2_safe_row,
        l2_status_row, unsafe_block_signer_row,
    },
    handles::{CurrentSignerRow, ErrorRow, FinalizedL2Row, Handles, SafeHeadUpdateRow, ViewOutput},
    snapshot::{ChainViewSnapshot, FinalizedL2, L1Statuses, SafeHeadEntry},
};

/// How the circuit thread is run.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct ChainViewConfig {
    /// Safe-head history entries kept for `safeHeadAtL1Block`, one per derived-from L1 block;
    /// the oldest are dropped beyond this. The default covers about a day of L1 blocks.
    pub history_limit: usize,
    /// Engine-imported L2 blocks held for derivation's lookups: the newest this many, and none
    /// below the engine's finalized head. Anything not held is fetched from the L2 RPC.
    pub imported_limit: usize,
}

impl Default for ChainViewConfig {
    fn default() -> Self {
        Self { history_limit: 8192, imported_limit: 256 }
    }
}

/// DBSP worker threads. One is right for this program: its deltas are tiny and every extra
/// worker adds an exchange barrier per step.
const WORKERS: usize = 1;
/// Maximum queued messages folded into one transaction.
const BATCH_LIMIT: usize = 4096;
/// Capacity of the fact/query channel.
const CHANNEL_CAPACITY: usize = 4096;

/// Errors of the chain view.
#[derive(Debug, thiserror::Error)]
pub enum ChainViewError {
    /// The circuit could not be built or stepped.
    #[error("circuit error: {0}")]
    Circuit(#[from] dbsp::Error),
    /// A fact could not be encoded as a circuit row.
    #[error(transparent)]
    Fact(#[from] FactError),
    /// The circuit thread is gone.
    #[error("chain view is closed")]
    Closed,
    /// The fact channel is full; only `try_push` reports this.
    #[error("chain view is busy: fact channel full")]
    Full,
    /// The circuit thread panicked.
    #[error("chain view thread panicked: {0}")]
    Panicked(String),
}

/// A running chain view: the client actors use, plus the thread's exit signal.
#[derive(Debug)]
pub struct ChainViewHandle {
    /// Pushes facts and queries; clone it freely.
    pub client: ChainViewClient,
    /// Resolves when the circuit thread exits, with its final result.
    pub exit: oneshot::Receiver<Result<(), ChainViewError>>,
    thread: Option<thread::JoinHandle<()>>,
}

impl ChainViewHandle {
    /// Asks the thread to stop and waits for it. When the request cannot be queued (the channel
    /// is full or already closed) the thread is left to exit on its own rather than joined, since
    /// the join could otherwise wait on senders other actors still hold.
    pub fn shutdown(mut self) {
        let requested = self.client.try_shutdown().is_ok();
        if let Some(thread) = self.thread.take() &&
            requested
        {
            let _ = thread.join();
        }
    }
}

/// Builds the circuit on a new thread and returns once it is ready.
pub fn spawn(config: ChainViewConfig) -> Result<ChainViewHandle, ChainViewError> {
    let (tx, rx) = mpsc::channel::<Msg>(CHANNEL_CAPACITY);
    let (snapshot_tx, snapshot_rx) = watch::channel(ChainViewSnapshot::default());
    let (exit_tx, exit_rx) = oneshot::channel();
    let (ready_tx, ready_rx) = std_mpsc::channel::<Result<(), ChainViewError>>();
    let history_limit = config.history_limit.max(1);
    let imported_limit = config.imported_limit;

    let thread = thread::Builder::new()
        .name("kona-chainview".to_string())
        .spawn(move || {
            let outcome = catch_unwind(AssertUnwindSafe(|| {
                let driver = match Driver::build(history_limit, imported_limit) {
                    Ok(driver) => {
                        let _ = ready_tx.send(Ok(()));
                        driver
                    }
                    Err(e) => {
                        let _ = ready_tx.send(Err(e));
                        return Ok(());
                    }
                };
                driver.run(rx, snapshot_tx)
            }));
            let result = match outcome {
                Ok(result) => result,
                Err(payload) => Err(ChainViewError::Panicked(panic_message(payload.as_ref()))),
            };
            let _ = exit_tx.send(result);
        })
        .expect("spawn kona-chainview thread");

    match ready_rx.recv() {
        Ok(Ok(())) => {}
        Ok(Err(e)) => {
            let _ = thread.join();
            return Err(e);
        }
        Err(_) => {
            let _ = thread.join();
            return Err(ChainViewError::Panicked(
                "circuit thread died before reporting readiness".into(),
            ));
        }
    }

    Ok(ChainViewHandle {
        client: ChainViewClient::new(tx, snapshot_rx),
        exit: exit_rx,
        thread: Some(thread),
    })
}

fn panic_message(payload: &(dyn std::any::Any + Send)) -> String {
    payload
        .downcast_ref::<String>()
        .cloned()
        .or_else(|| payload.downcast_ref::<&str>().map(|s| (*s).to_string()))
        .unwrap_or_else(|| "non-string panic payload".to_string())
}

/// Weighted rows of one view, integrated across steps.
#[derive(Debug)]
struct Integrated<Row> {
    rows: HashMap<Row, i64>,
}

impl<Row: Eq + Hash + Clone> Default for Integrated<Row> {
    fn default() -> Self {
        Self { rows: HashMap::new() }
    }
}

impl<Row: Eq + Hash + Clone> Integrated<Row> {
    fn apply(&mut self, row: Row, weight: i64) {
        match self.rows.entry(row) {
            Entry::Occupied(mut entry) => {
                *entry.get_mut() += weight;
                if *entry.get() == 0 {
                    entry.remove();
                }
            }
            Entry::Vacant(entry) => {
                if weight != 0 {
                    entry.insert(weight);
                }
            }
        }
    }

    /// The single current row of a view that holds at most one row.
    fn single(&self) -> Option<&Row> {
        self.rows.iter().find(|(_, w)| **w > 0).map(|(row, _)| row)
    }
}

/// The circuit plus everything integrated from it.
pub(crate) struct Driver {
    dbsp: dbsp::DBSPHandle,
    handles: Handles,
    /// Current `l1_status` row per kind, so a replacement can retract it.
    l1_status: HashMap<L1StatusKind, BlockInfo>,
    /// The `l1_blocks` rows by height: the pipeline's origins from finalized L1 upward.
    l1_window: BTreeMap<u64, BlockInfo>,
    /// The current `unsafe_block_signer` row, so a replacement can retract it.
    signer_row: Option<(BlockInfo, Address)>,
    /// Current `l2_status` rows.
    l2_heads: Option<L2Heads>,
    /// Asserted derived blocks by L2 number, so resets can retract exactly what was pushed.
    asserted: BTreeMap<u64, L2SafeFact>,
    /// Integrated `safe_head_updates`, by derived-from L1 number; at most `history_limit`
    /// entries, the oldest dropped first.
    history: BTreeMap<u64, SafeHeadEntry>,
    history_limit: usize,
    /// Engine-imported blocks by hash, for derivation's lookups; see `prune_imported`.
    imported: HashMap<B256, ImportedL2Block>,
    imported_limit: usize,
    finalized: Integrated<FinalizedL2Row>,
    signer: Integrated<CurrentSignerRow>,
    snapshot: ChainViewSnapshot,
}

impl Driver {
    fn build(history_limit: usize, imported_limit: usize) -> Result<Self, ChainViewError> {
        let (dbsp, handles) = crate::handles::build(CircuitConfig::with_workers(WORKERS))?;
        Ok(Self {
            dbsp,
            handles,
            l1_status: HashMap::new(),
            l1_window: BTreeMap::new(),
            signer_row: None,
            l2_heads: None,
            asserted: BTreeMap::new(),
            history: BTreeMap::new(),
            history_limit,
            imported: HashMap::new(),
            imported_limit,
            finalized: Integrated::default(),
            signer: Integrated::default(),
            snapshot: ChainViewSnapshot::default(),
        })
    }

    fn run(
        mut self,
        mut rx: mpsc::Receiver<Msg>,
        snapshot_tx: watch::Sender<ChainViewSnapshot>,
    ) -> Result<(), ChainViewError> {
        let result = loop {
            let Some(first) = rx.blocking_recv() else {
                break Ok(());
            };
            let mut batch = vec![first];
            while batch.len() < BATCH_LIMIT {
                match rx.try_recv() {
                    Ok(msg) => batch.push(msg),
                    Err(_) => break,
                }
            }

            let mut queries = Vec::new();
            let mut shutdown = false;
            let mut pushed = false;
            let mut failure = None;
            for msg in batch {
                match msg {
                    Msg::Fact(fact) => match self.apply_fact(*fact) {
                        Ok(()) => pushed = true,
                        Err(e) => {
                            failure = Some(e);
                            break;
                        }
                    },
                    Msg::Query(query) => queries.push(query),
                    Msg::Shutdown => shutdown = true,
                }
            }
            if let Some(e) = failure {
                break Err(e);
            }
            if pushed {
                if let Err(e) = self.step() {
                    break Err(e);
                }
                snapshot_tx.send_replace(self.snapshot.clone());
            }
            for query in queries {
                self.answer(query);
            }
            if shutdown {
                break Ok(());
            }
        };
        let _ = self.dbsp.kill();
        result
    }

    fn answer(&self, query: ChainViewQuery) {
        match query {
            ChainViewQuery::Sync { reply } => {
                let _ = reply.send(());
            }
            ChainViewQuery::SafeHeadAtL1 { number, reply } => {
                let entry = self.history.range(..=number).next_back().map(|(_, e)| *e);
                let _ = reply.send(entry);
            }
            ChainViewQuery::ImportedL2Block { hash, reply } => {
                let _ = reply.send(self.imported.get(&hash).cloned());
            }
        }
    }

    /// Drops held blocks below the engine's finalized head, then the lowest-numbered ones
    /// until at most `imported_limit` remain.
    fn prune_imported(&mut self) {
        let finalized = self.l2_heads.map_or(0, |heads| heads.finalized_head.block_info.number);
        self.imported.retain(|_, held| held.info.block_info.number >= finalized);
        while self.imported.len() > self.imported_limit {
            let Some(lowest) = self.imported.values().map(|held| held.info.block_info.number).min()
            else {
                break;
            };
            self.imported.retain(|_, held| held.info.block_info.number != lowest);
        }
    }

    fn apply_fact(&mut self, fact: Fact) -> Result<(), ChainViewError> {
        match fact {
            Fact::L2Imported(held) => {
                self.imported.insert(held.info.block_info.hash, held);
                self.prune_imported();
            }
            Fact::L1Origin(block) => {
                if self.l1_window.get(&block.number) == Some(&block) {
                    return Ok(());
                }
                if let Some(old) = self.l1_window.insert(block.number, block) {
                    self.handles.l1_blocks.push(l1_block_row(&old)?, -1);
                }
                self.handles.l1_blocks.push(l1_block_row(&block)?, 1);
            }
            Fact::UnsafeBlockSigner { l1, signer } => {
                if self.signer_row == Some((l1, signer)) {
                    return Ok(());
                }
                if let Some((prev_l1, prev_signer)) = self.signer_row.replace((l1, signer)) {
                    self.handles
                        .unsafe_block_signer
                        .push(unsafe_block_signer_row(&prev_l1, prev_signer)?, -1);
                }
                self.handles.unsafe_block_signer.push(unsafe_block_signer_row(&l1, signer)?, 1);
            }
            Fact::L1Status { kind, block } => {
                if self.l1_status.get(&kind) == Some(&block) {
                    return Ok(());
                }
                if let Some(prev) = self.l1_status.insert(kind, block) {
                    self.handles.l1_status.push(l1_status_row(kind, &prev)?, -1);
                }
                self.handles.l1_status.push(l1_status_row(kind, &block)?, 1);
                // Blocks below finalized L1 can no longer be reorged: drop their rows. Their
                // update rows stay, the folds need the whole history.
                if kind == L1StatusKind::Finalized {
                    let kept = self.l1_window.split_off(&block.number);
                    for old in std::mem::replace(&mut self.l1_window, kept).into_values() {
                        self.handles.l1_blocks.push(l1_block_row(&old)?, -1);
                    }
                }
            }
            Fact::L2Status(heads) => {
                let heads = *heads;
                if self.l2_heads == Some(heads) {
                    return Ok(());
                }
                let prev = self.l2_heads.replace(heads);
                self.prune_imported();
                for kind in L2StatusKind::ALL {
                    let next = heads.get(kind);
                    if let Some(prev) = prev {
                        let prev = prev.get(kind);
                        if prev == next {
                            continue;
                        }
                        self.handles.l2_status.push(l2_status_row(kind, &prev)?, -1);
                    }
                    self.handles.l2_status.push(l2_status_row(kind, &next)?, 1);
                }
            }
            Fact::L2Safe(fact) => {
                let number = fact.block.block_info.number;
                if let Some(prev) = self.asserted.insert(number, fact) {
                    if prev == fact {
                        return Ok(());
                    }
                    self.handles.l2_safe_blocks.push(l2_safe_row(&prev)?, -1);
                }
                self.handles.l2_safe_blocks.push(l2_safe_row(&fact)?, 1);
            }
            Fact::L2SafeRetractAbove { l2_number, l2_hash } => {
                let mut retracted = self.asserted.split_off(&l2_number.saturating_add(1));
                // A block at the reset height with another hash was reorged out as well.
                if self.asserted.get(&l2_number).is_some_and(|f| f.block.block_info.hash != l2_hash) &&
                    let Some(fact) = self.asserted.remove(&l2_number)
                {
                    retracted.insert(l2_number, fact);
                }
                for fact in retracted.values() {
                    self.handles.l2_safe_blocks.push(l2_safe_row(fact)?, -1);
                }
            }
        }
        Ok(())
    }

    fn step(&mut self) -> Result<(), ChainViewError> {
        let started = Instant::now();
        self.dbsp.transaction()?;
        self.integrate();
        tracing::trace!(
            target: "chainview",
            elapsed_us = started.elapsed().as_micros() as u64,
            "stepped chain view"
        );
        Ok(())
    }

    /// Reads every view's delta exactly once and folds it into host state.
    fn integrate(&mut self) {
        // Consumed but not integrated: the canonical set is only needed inside the circuit.
        drain(&self.handles.l2_safe_canonical, |_, _| {});

        let history = &mut self.history;
        let mut history_deltas: Vec<(SafeHeadUpdateRow, i64)> = Vec::new();
        drain(&self.handles.safe_head_updates, |row, weight| history_deltas.push((row, weight)));
        // Retractions first, then insertions, so a replaced entry never leaves a hole.
        history_deltas.sort_by_key(|(_, weight)| *weight);
        for (row, weight) in history_deltas {
            let l1_number = u64::try_from(row.0).unwrap_or(0);
            let entry = safe_head_entry(&row);
            if weight < 0 {
                if history.get(&l1_number) == entry.as_ref() {
                    history.remove(&l1_number);
                }
            } else if let Some(entry) = entry {
                history.insert(l1_number, entry);
            }
        }
        while history.len() > self.history_limit {
            history.pop_first();
        }

        let finalized = &mut self.finalized;
        drain(&self.handles.finalized_l2, |row, weight| finalized.apply(row, weight));
        let signer = &mut self.signer;
        drain(&self.handles.current_signer, |row, weight| signer.apply(row, weight));

        let mut drops = 0u64;
        drain(&self.handles.error_view, |row: ErrorRow, weight| {
            if weight > 0 {
                drops += 1;
                tracing::warn!(target: "chainview", relation = row.0.str(), message = row.1.str(), "circuit rejected a row");
            }
        });
        self.snapshot.lateness_drops += drops;

        self.refresh_snapshot();
    }

    fn refresh_snapshot(&mut self) {
        let l1 = L1Statuses {
            head: self.l1_status.get(&L1StatusKind::Head).copied(),
            safe: self.l1_status.get(&L1StatusKind::Safe).copied(),
            finalized: self.l1_status.get(&L1StatusKind::Finalized).copied(),
            current: self.l1_status.get(&L1StatusKind::Current).copied(),
        };
        let finalized_l2 = self.finalized.single().and_then(finalized_l2);
        let signer = self.signer.single().and_then(|row| address_from_bytes(&row.0));

        // Rows at or below the engine's finalized head can never be retracted again.
        if let Some(heads) = self.l2_heads {
            let floor = heads.finalized_head.block_info.number;
            self.asserted = self.asserted.split_off(&floor);
        }

        self.snapshot.l1 = l1;
        self.snapshot.l2 = self.l2_heads;
        self.snapshot.finalized_l2 = finalized_l2;
        self.snapshot.signer = signer;
        self.snapshot.history_len = self.history.len();
        self.snapshot.l1_window_len = self.l1_window.len();
    }
}

/// Reads and consumes one view's accumulated delta.
fn drain<Row: DBData>(handle: &ViewOutput<Row>, mut f: impl FnMut(Row, i64)) {
    for snapshot in handle.take_from_all() {
        for (row, (), weight) in snapshot.consolidate().iter() {
            f(row, weight);
        }
    }
}

fn safe_head_entry(row: &SafeHeadUpdateRow) -> Option<SafeHeadEntry> {
    Some(SafeHeadEntry {
        l1: BlockNumHash { number: u64::try_from(row.0).ok()?, hash: hash_from_bytes(&row.1)? },
        l2: BlockNumHash { number: u64::try_from(row.2).ok()?, hash: hash_from_bytes(&row.3)? },
    })
}

fn finalized_l2(row: &FinalizedL2Row) -> Option<FinalizedL2> {
    Some(FinalizedL2 {
        id: BlockNumHash {
            number: u64::try_from(row.0?).ok()?,
            hash: hash_from_bytes(row.1.as_ref()?)?,
        },
        derived_from: BlockNumHash {
            number: u64::try_from(row.2?).ok()?,
            hash: hash_from_bytes(row.3.as_ref()?)?,
        },
    })
}
