//! A [`rocksdb`]-backed [`SafeDbV1`] implementation.

use crate::{
    encoding::{
        decode_safe_by_l1_block_num, max_key, safe_by_l1_block_num_key, safe_by_l1_block_num_value,
    },
    error::SafeDbError,
    traits::{SafeDbV1, SafeHeadRecord},
};
use alloy_eips::BlockNumHash;
use kona_protocol::L2BlockInfo;
use rocksdb::{DB, DBRawIterator, Options, ReadOptions, WriteBatch, WriteOptions};
use std::{
    path::Path,
    sync::{PoisonError, RwLock, RwLockReadGuard, RwLockWriteGuard},
};

/// A safe-head database that persists records to a [`rocksdb`] store.
///
/// The store is held behind an [`RwLock`] so that [`SafeDb::close`] can drop it while ensuring
/// no read iterator is still borrowing it, mirroring the read/write exclusion of the Go
/// implementation.
#[derive(Debug)]
pub struct SafeDb {
    inner: RwLock<Option<DB>>,
}

impl SafeDb {
    /// Opens (creating if necessary) a safe-head database at `path`.
    pub fn new(path: impl AsRef<Path>) -> Result<Self, SafeDbError> {
        let mut options = Options::default();
        options.create_if_missing(true);
        let db = DB::open(&options, path)?;
        Ok(Self { inner: RwLock::new(Some(db)) })
    }

    fn read_guard(&self) -> RwLockReadGuard<'_, Option<DB>> {
        self.inner.read().unwrap_or_else(PoisonError::into_inner)
    }

    fn write_guard(&self) -> RwLockWriteGuard<'_, Option<DB>> {
        self.inner.write().unwrap_or_else(PoisonError::into_inner)
    }
}

/// Write options that fsync each write, matching the Go implementation's synchronous writes.
fn sync_write_options() -> WriteOptions {
    let mut options = WriteOptions::default();
    options.set_sync(true);
    options
}

/// Read options bounding iteration to the "safe head by L1 block number" column.
///
/// The upper bound is exclusive, matching the Go implementation's `IterRange`. Without these
/// bounds an iterator would walk into any other column a future schema adds.
fn column_read_options() -> ReadOptions {
    let mut options = ReadOptions::default();
    options.set_iterate_lower_bound(safe_by_l1_block_num_key(0).to_vec());
    options.set_iterate_upper_bound(max_key().to_vec());
    options
}

/// Decodes the entry the iterator is currently positioned on.
fn current_entry(iter: &DBRawIterator<'_>) -> Result<(BlockNumHash, BlockNumHash), SafeDbError> {
    let key = iter.key().ok_or(SafeDbError::InvalidEntry)?;
    let value = iter.value().ok_or(SafeDbError::InvalidEntry)?;
    decode_safe_by_l1_block_num(key, value)
}

impl SafeDbV1 for SafeDb {
    fn enabled(&self) -> bool {
        true
    }

    fn safe_head_updated(
        &self,
        safe_head: L2BlockInfo,
        l1_head: BlockNumHash,
    ) -> Result<(), SafeDbError> {
        let guard = self.write_guard();
        let db = guard.as_ref().ok_or(SafeDbError::Closed)?;
        tracing::info!(
            l2_number = safe_head.block_info.number,
            l1_number = l1_head.number,
            "Record local safe head"
        );
        let mut batch = WriteBatch::default();
        batch.put(
            safe_by_l1_block_num_key(l1_head.number),
            safe_by_l1_block_num_value(l1_head, safe_head.block_info.id()),
        );
        db.write_opt(batch, &sync_write_options())?;
        Ok(())
    }

    fn safe_head_reset(&self, safe_head: L2BlockInfo) -> Result<(), SafeDbError> {
        let guard = self.write_guard();
        let db = guard.as_ref().ok_or(SafeDbError::Closed)?;
        tracing::info!(l2_number = safe_head.block_info.number, "Resetting safe head db");

        let mut iter = db.raw_iterator_opt(column_read_options());
        iter.seek(safe_by_l1_block_num_key(safe_head.l1_origin.number));
        while iter.valid() {
            let (l1_block, l2_block) = current_entry(&iter)?;
            if l2_block.number >= safe_head.block_info.number {
                // Clone the boundary key before `prev()` invalidates the borrow.
                let boundary_key = iter.key().ok_or(SafeDbError::InvalidEntry)?.to_vec();
                iter.prev();
                let has_prev_entry = iter.valid();
                if !has_prev_entry {
                    iter.status()?;
                }

                let mut batch = WriteBatch::default();
                batch.delete_range(boundary_key.as_slice(), max_key().as_slice());
                // If a previous entry exists, the boundary L1 block is a real recorded
                // transition, so re-record the reset safe head there. If it does not, we cannot
                // know whether the reset head became safe at that L1 block or merely before our
                // records begin, so we leave it unrecorded.
                if has_prev_entry {
                    batch.put(
                        boundary_key.as_slice(),
                        safe_by_l1_block_num_value(l1_block, safe_head.block_info.id()).as_slice(),
                    );
                }
                db.write_opt(batch, &sync_write_options())?;
                return Ok(());
            }
            iter.next();
        }
        iter.status()?;
        Ok(())
    }

    fn safe_head_at_l1(&self, l1_block_num: u64) -> Result<SafeHeadRecord, SafeDbError> {
        let guard = self.read_guard();
        let db = guard.as_ref().ok_or(SafeDbError::Closed)?;
        let mut iter = db.raw_iterator_opt(column_read_options());
        // Largest key <= l1_block_num, equivalent to the Go `SeekLT(l1_block_num + 1)`.
        iter.seek_for_prev(safe_by_l1_block_num_key(l1_block_num));
        if !iter.valid() {
            iter.status()?;
            return Err(SafeDbError::NotFound);
        }
        let (l1, safe_head) = current_entry(&iter)?;
        Ok(SafeHeadRecord { l1, safe_head })
    }

    fn first_entry(&self) -> Result<SafeHeadRecord, SafeDbError> {
        let guard = self.read_guard();
        let db = guard.as_ref().ok_or(SafeDbError::Closed)?;
        let mut iter = db.raw_iterator_opt(column_read_options());
        iter.seek_to_first();
        if !iter.valid() {
            iter.status()?;
            return Err(SafeDbError::NotFound);
        }
        let (l1, safe_head) = current_entry(&iter)?;
        Ok(SafeHeadRecord { l1, safe_head })
    }

    fn last_entry(&self) -> Result<SafeHeadRecord, SafeDbError> {
        let guard = self.read_guard();
        let db = guard.as_ref().ok_or(SafeDbError::Closed)?;
        let mut iter = db.raw_iterator_opt(column_read_options());
        iter.seek_to_last();
        if !iter.valid() {
            iter.status()?;
            return Err(SafeDbError::NotFound);
        }
        let (l1, safe_head) = current_entry(&iter)?;
        Ok(SafeHeadRecord { l1, safe_head })
    }

    fn l1_at_safe_head(&self, target_l2_num: u64) -> Result<SafeHeadRecord, SafeDbError> {
        let guard = self.read_guard();
        let db = guard.as_ref().ok_or(SafeDbError::Closed)?;
        let mut iter = db.raw_iterator_opt(column_read_options());

        iter.seek_to_last();
        if !iter.valid() {
            iter.status()?;
            return Err(SafeDbError::L1AtSafeHeadNotFound);
        }
        let (mut cursor_l1, mut cursor_l2) = current_entry(&iter)?;
        if target_l2_num > cursor_l2.number {
            return Err(SafeDbError::L1AtSafeHeadNotFound);
        }
        loop {
            iter.prev();
            if !iter.valid() {
                iter.status()?;
                break;
            }
            let (prev_l1, prev_l2) = current_entry(&iter)?;
            if prev_l2.number < target_l2_num {
                return Ok(SafeHeadRecord { l1: cursor_l1, safe_head: cursor_l2 });
            }
            cursor_l1 = prev_l1;
            cursor_l2 = prev_l2;
        }
        if cursor_l2.number == target_l2_num {
            return Ok(SafeHeadRecord { l1: cursor_l1, safe_head: cursor_l2 });
        }
        Err(SafeDbError::L1AtSafeHeadUnavailable)
    }

    fn close(&self) -> Result<(), SafeDbError> {
        let mut guard = self.write_guard();
        *guard = None;
        Ok(())
    }
}
