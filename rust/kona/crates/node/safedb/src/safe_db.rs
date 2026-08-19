//! A [`rocksdb`]-backed [`SafeDb`] implementation.

use crate::{
    encoding::SafeByL1BlockNum,
    error::SafeDbError,
    traits::{SafeDb, SafeHeadRecord},
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
/// The store is held behind an [`RwLock`] so that [`SafeDatabase::close`] can drop it while
/// ensuring no read iterator is still borrowing it.
#[derive(Debug)]
pub struct SafeDatabase {
    inner: RwLock<Option<DB>>,
}

impl SafeDatabase {
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

    /// Write options that fsync each write, so a recorded safe head survives a crash.
    fn sync_write_options() -> WriteOptions {
        let mut options = WriteOptions::default();
        options.set_sync(true);
        options
    }

    /// Read options bounding iteration to the "safe head by L1 block number" column.
    ///
    /// The upper bound is exclusive. Without these bounds an iterator would walk into any other
    /// column a future schema adds.
    fn column_read_options() -> ReadOptions {
        let mut options = ReadOptions::default();
        options.set_iterate_lower_bound(SafeByL1BlockNum::key(0).to_vec());
        options.set_iterate_upper_bound(SafeByL1BlockNum::max_key().to_vec());
        options
    }

    /// Decodes the entry the iterator is currently positioned on.
    fn current_entry(iter: &DBRawIterator<'_>) -> Result<SafeHeadRecord, SafeDbError> {
        let key = iter.key().ok_or(SafeDbError::InvalidEntry)?;
        let value = iter.value().ok_or(SafeDbError::InvalidEntry)?;
        SafeByL1BlockNum::decode(key, value)
    }
}

impl SafeDb for SafeDatabase {
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
            SafeByL1BlockNum::key(l1_head.number),
            SafeByL1BlockNum::value(l1_head, safe_head.block_info.id()),
        );
        db.write_opt(batch, &Self::sync_write_options())?;
        Ok(())
    }

    fn safe_head_reset(&self, safe_head: L2BlockInfo) -> Result<(), SafeDbError> {
        let guard = self.write_guard();
        let db = guard.as_ref().ok_or(SafeDbError::Closed)?;
        tracing::info!(l2_number = safe_head.block_info.number, "Resetting safe head db");

        let mut iter = db.raw_iterator_opt(Self::column_read_options());
        iter.seek(SafeByL1BlockNum::key(safe_head.l1_origin.number));
        while iter.valid() {
            let record = Self::current_entry(&iter)?;
            if record.safe_head.number >= safe_head.block_info.number {
                // Clone the boundary key before `prev()` invalidates the borrow.
                let boundary_key = iter.key().ok_or(SafeDbError::InvalidEntry)?.to_vec();
                iter.prev();
                let has_prev_entry = iter.valid();
                if !has_prev_entry {
                    iter.status()?;
                }

                let mut batch = WriteBatch::default();
                batch.delete_range(boundary_key.as_slice(), SafeByL1BlockNum::max_key().as_slice());
                // If a previous entry exists, the boundary L1 block is a real recorded
                // transition, so re-record the reset safe head there. If it does not, we cannot
                // know whether the reset head became safe at that L1 block or merely before our
                // records begin, so we leave it unrecorded.
                if has_prev_entry {
                    batch.put(
                        boundary_key.as_slice(),
                        SafeByL1BlockNum::value(record.l1, safe_head.block_info.id()).as_slice(),
                    );
                }
                db.write_opt(batch, &Self::sync_write_options())?;
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
        let mut iter = db.raw_iterator_opt(Self::column_read_options());
        // Largest key <= l1_block_num.
        iter.seek_for_prev(SafeByL1BlockNum::key(l1_block_num));
        if !iter.valid() {
            iter.status()?;
            return Err(SafeDbError::NotFound);
        }
        Self::current_entry(&iter)
    }

    fn first_entry(&self) -> Result<SafeHeadRecord, SafeDbError> {
        let guard = self.read_guard();
        let db = guard.as_ref().ok_or(SafeDbError::Closed)?;
        let mut iter = db.raw_iterator_opt(Self::column_read_options());
        iter.seek_to_first();
        if !iter.valid() {
            iter.status()?;
            return Err(SafeDbError::NotFound);
        }
        Self::current_entry(&iter)
    }

    fn last_entry(&self) -> Result<SafeHeadRecord, SafeDbError> {
        let guard = self.read_guard();
        let db = guard.as_ref().ok_or(SafeDbError::Closed)?;
        let mut iter = db.raw_iterator_opt(Self::column_read_options());
        iter.seek_to_last();
        if !iter.valid() {
            iter.status()?;
            return Err(SafeDbError::NotFound);
        }
        Self::current_entry(&iter)
    }

    fn l1_at_safe_head(&self, target_l2_num: u64) -> Result<SafeHeadRecord, SafeDbError> {
        let guard = self.read_guard();
        let db = guard.as_ref().ok_or(SafeDbError::Closed)?;
        let mut iter = db.raw_iterator_opt(Self::column_read_options());

        iter.seek_to_last();
        if !iter.valid() {
            iter.status()?;
            return Err(SafeDbError::L1AtSafeHeadNotFound);
        }
        let mut cursor = Self::current_entry(&iter)?;
        if target_l2_num > cursor.safe_head.number {
            return Err(SafeDbError::L1AtSafeHeadNotFound);
        }
        loop {
            iter.prev();
            if !iter.valid() {
                iter.status()?;
                break;
            }
            let prev = Self::current_entry(&iter)?;
            if prev.safe_head.number < target_l2_num {
                return Ok(cursor);
            }
            cursor = prev;
        }
        if cursor.safe_head.number == target_l2_num {
            return Ok(cursor);
        }
        Err(SafeDbError::L1AtSafeHeadUnavailable)
    }

    fn close(&self) -> Result<(), SafeDbError> {
        let mut guard = self.write_guard();
        *guard = None;
        Ok(())
    }
}
