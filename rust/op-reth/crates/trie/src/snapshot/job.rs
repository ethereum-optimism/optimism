//! [`SnapshotInitJob`] — chunked builder for the one-time trie-state snapshot.

use super::SnapshotError;
use crate::{
    OpProofsHashedAccountCursorFactory, OpProofsProviderRO, OpProofsSnapshotInitProvider,
    SnapshotInitAnchor, SnapshotInitStatus, SnapshotTrieCursorFactory, db::StorageTrieKey,
    initialize::CompletionEstimatable,
};
use alloy_eips::BlockNumHash;
use alloy_primitives::{B256, BlockNumber};
use reth_primitives_traits::AlloyBlockHeader;
use reth_provider::{BlockHashReader, HeaderProvider, ProviderError};
use reth_trie::{
    BranchNodeCompact, HashedPostState, Nibbles, StateRoot, StoredNibbles,
    hashed_cursor::{HashedCursor, HashedPostStateCursorFactory},
    trie_cursor::TrieCursor,
};
use std::time::Instant;
use tracing::info;

/// Default rows copied per chunked init transaction. Used by
/// [`SnapshotInitJob::new`]; callers can override via
/// [`SnapshotInitJob::with_chunk_size`].
const SNAPSHOT_INIT_CHUNK_SIZE: usize = 50_000;

/// Storage-trie chunk grouped by hashed address. Each inner vec is the
/// per-address payload accepted by
/// [`OpProofsSnapshotInitProvider::store_storage_trie_snapshot_branches`].
type StorageChunk = Vec<(B256, Vec<(Nibbles, Option<BranchNodeCompact>)>)>;

/// Output of a successful [`SnapshotInitJob::run`] call.
///
/// Shape mirrors [`crate::api::SnapshotInitAnchor`]: the anchor `block` plus
/// the lifecycle `status` (always [`SnapshotInitStatus::Completed`] on success).
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct SnapshotInitOutcome {
    /// Block the snapshot's trie state corresponds to.
    pub block: BlockNumHash,
    /// Lifecycle status — always [`SnapshotInitStatus::Completed`] for a
    /// successful run.
    pub status: SnapshotInitStatus,
    /// Number of account trie nodes copied during this run (does **not**
    /// include rows already present from a prior resumable run).
    pub account_nodes_copied: u64,
    /// Number of storage trie nodes copied during this run.
    pub storage_nodes_copied: u64,
}

/// Builds the one-time trie-state snapshot.
#[derive(Debug)]
pub struct SnapshotInitJob<P, S: crate::OpProofsBackfillStore + Send> {
    /// Reth DB provider (used to look up the target block's hash + header).
    provider: P,
    /// Op-reth proofs storage that owns the snapshot tables.
    storage: S,
    /// Rows committed per chunked rw-tx during the drain phases. Larger
    /// values trade peak memory for fewer commits.
    chunk_size: usize,
}

impl<P, S: crate::OpProofsBackfillStore + Send> SnapshotInitJob<P, S> {
    /// Build a job with the default chunk size.
    pub const fn new(provider: P, storage: S) -> Self {
        Self { provider, storage, chunk_size: SNAPSHOT_INIT_CHUNK_SIZE }
    }

    /// Override the per-tx chunk size. Useful for operators tuning memory
    /// usage on small/large databases, or for tests that want to drive the
    /// chunk loop in a few iterations.
    pub const fn with_chunk_size(mut self, chunk_size: usize) -> Self {
        self.chunk_size = chunk_size;
        self
    }
}

impl<P, S> SnapshotInitJob<P, S>
where
    P: HeaderProvider + BlockHashReader + Send,
    S: crate::OpProofsBackfillStore + Send,
{
    /// Build a snapshot at `target_block`, validating against the reth header.
    ///
    /// `target_block` must fall inside the proofs window's `[earliest, latest]`.
    /// Auto-resumes a partial `Building` snapshot if the existing anchor
    /// matches; refuses to run if a `Ready` snapshot exists at a different
    /// anchor (the caller must drop it first).
    pub fn run(&self, target_block: BlockNumber) -> Result<SnapshotInitOutcome, SnapshotError> {
        let start = Instant::now();

        let target = self.prepare_anchor(target_block)?;
        // Read the init anchor once: lifecycle classification uses status/block,
        // and the drain phases use the destination-table resume keys.
        let init_anchor =
            self.storage.snapshot_initialization_provider()?.snapshot_init_anchor()?;
        self.start_or_resume(target, &init_anchor)?;

        let expected_root = self.expected_state_root(target_block)?;

        let copy_start = Instant::now();
        let account_nodes_copied =
            self.drain_account_trie(target_block, init_anchor.last_account_trie_key)?;
        let storage_nodes_copied =
            self.drain_storage_trie(target_block, init_anchor.last_storage_trie_key)?;
        let copy_elapsed = copy_start.elapsed();

        let validate_start = Instant::now();
        self.validate_state_root(target_block, expected_root)?;
        let validate_elapsed = validate_start.elapsed();

        self.finalize_ready()?;

        info!(
            target: "reth::op-proofs::snapshot-init",
            block = target_block,
            account_nodes_copied,
            storage_nodes_copied,
            copy_elapsed = ?copy_elapsed,
            validate_elapsed = ?validate_elapsed,
            total_elapsed = ?start.elapsed(),
            "Snapshot init complete"
        );

        Ok(SnapshotInitOutcome {
            block: target,
            status: SnapshotInitStatus::Completed,
            account_nodes_copied,
            storage_nodes_copied,
        })
    }

    /// Validate `target_block` is inside the proofs window and resolve its hash.
    fn prepare_anchor(&self, target_block: BlockNumber) -> Result<BlockNumHash, SnapshotError> {
        let ro = self.storage.provider_ro()?;
        let window = ro.get_proof_window()?;
        if target_block < window.earliest.number || target_block > window.latest.number {
            return Err(SnapshotError::SnapshotInitTargetOutsideWindow {
                target_block,
                earliest: window.earliest.number,
                latest: window.latest.number,
            });
        }

        let target_hash = self
            .provider
            .block_hash(target_block)?
            .ok_or_else(|| ProviderError::HeaderNotFound(target_block.into()))?;
        Ok(BlockNumHash::new(target_block, target_hash))
    }

    /// Decide whether this run starts fresh or resumes an in-flight build,
    /// and plant a new `Building` row when fresh.
    ///
    /// Errors on drift (`InProgress` at a different target) or refusal
    /// (`Completed` snapshot already exists).
    fn start_or_resume(
        &self,
        target: BlockNumHash,
        init_anchor: &SnapshotInitAnchor,
    ) -> Result<(), SnapshotError> {
        // Invariant: `InProgress`/`Completed` always carry an anchor block —
        // `set_snapshot_init_anchor` plants status + block atomically.
        let resume = match init_anchor.status {
            SnapshotInitStatus::NotStarted => false,
            SnapshotInitStatus::InProgress => {
                let b = init_anchor.block.expect("InProgress implies anchor planted");
                if b == target {
                    true
                } else {
                    return Err(SnapshotError::SnapshotResumeDriftDetected {
                        anchor_block: b.number,
                        reason: "snapshot init target does not match the in-progress snapshot anchor",
                    });
                }
            }
            SnapshotInitStatus::Completed => {
                let b = init_anchor.block.expect("Completed implies anchor planted");
                return Err(SnapshotError::SnapshotAlreadyExists {
                    existing_block: b.number,
                    existing_status: SnapshotInitStatus::Completed,
                });
            }
        };

        if !resume {
            let sp = self.storage.snapshot_initialization_provider()?;
            sp.set_snapshot_init_anchor(target)?;
            OpProofsSnapshotInitProvider::commit(sp)?;
        }
        info!(
            target: "reth::op-proofs::snapshot-init",
            block = target.number,
            resume,
            "Starting snapshot init"
        );
        Ok(())
    }

    /// Look up the expected state root for `target_block` from reth's headers.
    fn expected_state_root(&self, target_block: BlockNumber) -> Result<B256, SnapshotError> {
        Ok(self
            .provider
            .header_by_number(target_block)?
            .ok_or_else(|| ProviderError::HeaderNotFound(target_block.into()))?
            .state_root())
    }

    /// Drain the history-aware account trie cursor at `target_block` into
    /// `V2AccountsTrieSnapshot`, one chunk per rw-tx. Resumes past whatever's
    /// currently in the snapshot.
    ///
    /// Returns the number of rows copied during *this* call (excluding rows
    /// already present from prior runs).
    fn drain_account_trie(
        &self,
        target_block: BlockNumber,
        mut resume_after: Option<StoredNibbles>,
    ) -> Result<u64, SnapshotError> {
        let phase_start = Instant::now();
        let mut initial_progress: Option<f64> = None;
        let mut copied = 0u64;

        // `resume_after` is the destination table's last key at run start
        // (read once in `run`); thereafter we advance it locally as we
        // commit each chunk.
        loop {
            let chunk = {
                let ro = self.storage.provider_ro()?;
                let mut cursor = ro.account_trie_cursor(target_block)?;
                collect_account_chunk(&mut cursor, resume_after.clone(), self.chunk_size)?
            };
            if chunk.is_empty() {
                break;
            }
            let n = chunk.len() as u64;
            let last_key = chunk.last().expect("non-empty").0.clone();
            let sp = self.storage.snapshot_initialization_provider()?;
            sp.store_account_trie_snapshot_branches(chunk)?;
            OpProofsSnapshotInitProvider::commit(sp)?;
            copied += n;
            log_phase_progress("accounts", &last_key, &mut initial_progress, phase_start, copied);
            resume_after = Some(last_key);
        }
        Ok(copied)
    }

    /// Walk hashed accounts at `target_block`, drain each account's historical
    /// storage trie cursor into `V2StoragesTrieSnapshot`, one chunk per rw-tx.
    ///
    /// Resume tracks the last `StorageTrieKey` written.
    fn drain_storage_trie(
        &self,
        target_block: BlockNumber,
        mut resume_after: Option<StorageTrieKey>,
    ) -> Result<u64, SnapshotError> {
        let phase_start = Instant::now();
        let mut initial_progress: Option<f64> = None;
        let mut copied = 0u64;

        // `resume_after` is the destination table's last key at run start
        // (read once in `run`); thereafter we advance it locally as we
        // commit each chunk.
        loop {
            let chunk = {
                let ro = self.storage.provider_ro()?;
                collect_storage_chunk(&ro, target_block, resume_after.clone(), self.chunk_size)?
            };
            if chunk.is_empty() {
                break;
            }
            let n: u64 = chunk.iter().map(|(_, nodes)| nodes.len() as u64).sum();
            // The chunk's last storage-trie key is the last (addr, path) pair we
            // just committed. Storage progress tracks the hashed address, so
            // this is what feeds `estimate_progress`.
            let (last_addr, last_path_group) = chunk.last().expect("non-empty");
            let last_key = StorageTrieKey::new(
                *last_addr,
                StoredNibbles(last_path_group.last().expect("non-empty group").0),
            );
            let sp = self.storage.snapshot_initialization_provider()?;
            for (addr, nodes) in chunk {
                sp.store_storage_trie_snapshot_branches(addr, nodes)?;
            }
            OpProofsSnapshotInitProvider::commit(sp)?;
            copied += n;
            log_phase_progress("storages", &last_key, &mut initial_progress, phase_start, copied);
            resume_after = Some(last_key);
        }
        Ok(copied)
    }

    /// Compute the state root from the snapshot tables and the live hashed
    /// leaves and compare against `expected_root`.
    ///
    /// On mismatch the meta is **not** advanced — it stays at `Building` so
    /// a re-run can diagnose / resume / `snapshot-drop`.
    fn validate_state_root(
        &self,
        target_block: BlockNumber,
        expected_root: B256,
    ) -> Result<(), SnapshotError> {
        let sp = self.storage.snapshot_provider_ro()?;
        let state_sorted = HashedPostState::default().into_sorted();
        let computed_root = StateRoot::new(
            SnapshotTrieCursorFactory::new(sp.clone()),
            HashedPostStateCursorFactory::new(
                OpProofsHashedAccountCursorFactory::new(sp.clone(), target_block),
                &state_sorted,
            ),
        )
        .root()?;

        if computed_root != expected_root {
            return Err(SnapshotError::StateRootMismatch {
                block_number: target_block,
                computed: computed_root,
                expected: expected_root,
            });
        }
        Ok(())
    }

    /// Flip status to `Ready` and commit in a final rw-tx.
    fn finalize_ready(&self) -> Result<(), SnapshotError> {
        let sp = self.storage.snapshot_initialization_provider()?;
        sp.commit_snapshot()?;
        OpProofsSnapshotInitProvider::commit(sp)?;
        Ok(())
    }
}

/// Drain up to `max_entries` rows from a `TrieCursor` strictly after `resume_after`.
fn collect_account_chunk<C: TrieCursor>(
    cursor: &mut C,
    resume_after: Option<StoredNibbles>,
    max_entries: usize,
) -> Result<Vec<(StoredNibbles, BranchNodeCompact)>, SnapshotError> {
    if max_entries == 0 {
        return Ok(Vec::new());
    }
    let mut next = match resume_after {
        None => cursor.seek(Nibbles::default())?,
        Some(after) => {
            // `seek` returns the first key >= `after`. If it matches exactly,
            // skip past it; otherwise we're already past.
            match cursor.seek(after.0)? {
                Some((k, _)) if k == after.0 => cursor.next()?,
                other => other,
            }
        }
    };
    let mut out = Vec::with_capacity(max_entries);
    while let Some((k, v)) = next {
        if out.len() >= max_entries {
            break;
        }
        out.push((StoredNibbles(k), v));
        next = cursor.next()?;
    }
    Ok(out)
}

/// Collect a chunk of storage-trie entries grouped by hashed address.
///
/// Walks hashed accounts at `target_block` and drains each account's storage
/// trie cursor up to `max_entries` total nodes (counted across all groups),
/// returning `Vec<(address, nodes_for_that_address)>`. Each inner vector is
/// the per-address payload accepted by
/// [`OpProofsSnapshotInitProvider::store_storage_trie_snapshot_branches`].
///
/// Resume semantics: `resume_after` is the last [`StorageTrieKey`] already
/// written to the snapshot. We seek the account cursor to its address,
/// position that account's storage cursor past its path, drain, then advance
/// to the next account.
fn collect_storage_chunk<P>(
    proofs_ro: &P,
    target_block: BlockNumber,
    resume_after: Option<StorageTrieKey>,
    max_entries: usize,
) -> Result<StorageChunk, SnapshotError>
where
    P: OpProofsProviderRO,
{
    if max_entries == 0 {
        return Ok(Vec::new());
    }
    let mut out: StorageChunk = Vec::new();
    let mut total = 0usize;

    let (start_addr, mut path_resume) = match resume_after {
        None => (B256::ZERO, None),
        Some(k) => (k.hashed_address, Some(k.path)),
    };

    let mut acc_cursor = proofs_ro.account_hashed_cursor(target_block)?;
    let mut next_account = acc_cursor.seek(start_addr)?;

    while let Some((addr, _account)) = next_account {
        let mut stor_cursor = proofs_ro.storage_trie_cursor(addr, target_block)?;

        // Position past any pending path resume (only applies to the first
        // account on this call — subsequent accounts always start at the
        // beginning of their trie).
        let mut next_stor = match path_resume.take() {
            None => stor_cursor.seek(Nibbles::default())?,
            Some(p) => match stor_cursor.seek(p.0)? {
                Some((k, _)) if k == p.0 => stor_cursor.next()?,
                other => other,
            },
        };

        let mut group: Vec<(Nibbles, Option<BranchNodeCompact>)> = Vec::new();
        while let Some((path, node)) = next_stor {
            if total >= max_entries {
                if !group.is_empty() {
                    out.push((addr, group));
                }
                return Ok(out);
            }
            group.push((path, Some(node)));
            total += 1;
            next_stor = stor_cursor.next()?;
        }
        if !group.is_empty() {
            out.push((addr, group));
        }

        next_account = acc_cursor.next()?;
    }

    Ok(out)
}

impl CompletionEstimatable for StorageTrieKey {
    /// Address dominates ordering, so progress along the storage-trie scan
    /// tracks the hashed address.
    fn estimate_progress(&self) -> f64 {
        self.hashed_address.estimate_progress()
    }
}

/// Emit an `info!` line with chunk count, cumulative count, progress %, and ETA.
///
/// Modeled on the loop in [`crate::initialize::InitializationJob`]: `last_key`'s
/// position in keyspace gives a 0–1 progress fraction, and we extrapolate ETA
/// from how far we moved since the phase started. `initial_progress` is
/// captured on the first call so the rate excludes resume offsets.
fn log_phase_progress<K: CompletionEstimatable>(
    phase: &'static str,
    last_key: &K,
    initial_progress: &mut Option<f64>,
    phase_start: Instant,
    cumulative: u64,
) {
    let progress = last_key.estimate_progress();
    let initial = *initial_progress.get_or_insert(progress);
    let elapsed_secs = phase_start.elapsed().as_secs_f64();

    let rate = if elapsed_secs.is_normal() { (progress - initial) / elapsed_secs } else { 0.0 };
    let eta_secs = if rate.is_normal() && rate > 0.0 { (1.0 - progress) / rate } else { 0.0 };

    info!(
        target: "reth::op-proofs::snapshot-init",
        phase,
        cumulative,
        progress_pct = format_args!("{:.2}", progress * 100.0),
        eta_secs = format_args!("{eta_secs:.0}"),
        "Snapshot init progress"
    );
}
