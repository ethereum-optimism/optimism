//! The [`rocksdb`]-backed [`Kv`] implementation.

use crate::{error::StoreError, kv::Kv};
use rocksdb::{DB, Options, ReadOptions, WriteOptions};
use std::{
    path::Path,
    sync::{PoisonError, RwLock, RwLockReadGuard, RwLockWriteGuard},
};

/// A [`Kv`] that persists to a [`rocksdb`] database.
///
/// The handle is held behind an [`RwLock`] so [`RocksKv::close`] can drop it while ensuring no
/// read iterator is still borrowing it.
#[derive(Debug)]
pub struct RocksKv {
    inner: RwLock<Option<DB>>,
}

impl RocksKv {
    /// Opens (creating if necessary) a database at `path`.
    pub fn open(path: impl AsRef<Path>) -> Result<Self, StoreError> {
        let mut options = Options::default();
        options.create_if_missing(true);
        let db = DB::open(&options, path)?;
        Ok(Self { inner: RwLock::new(Some(db)) })
    }

    /// Drops the database handle. Subsequent operations return [`StoreError::Closed`].
    ///
    /// Every write is already fsynced, so closing loses nothing.
    pub fn close(&self) {
        let mut guard = self.write_guard();
        *guard = None;
    }

    fn read_guard(&self) -> RwLockReadGuard<'_, Option<DB>> {
        self.inner.read().unwrap_or_else(PoisonError::into_inner)
    }

    fn write_guard(&self) -> RwLockWriteGuard<'_, Option<DB>> {
        self.inner.write().unwrap_or_else(PoisonError::into_inner)
    }

    /// Write options that fsync, so a record is durable once [`Kv::write`] returns. The stores
    /// order their side effects around that guarantee.
    fn sync_write_options() -> WriteOptions {
        let mut options = WriteOptions::default();
        options.set_sync(true);
        options
    }

    /// Read options bounding iteration to `[start, end)`, so an iterator cannot walk out of the
    /// column it was asked about.
    fn bounded_read_options(start: &[u8], end: &[u8]) -> ReadOptions {
        let mut options = ReadOptions::default();
        options.set_iterate_lower_bound(start.to_vec());
        options.set_iterate_upper_bound(end.to_vec());
        options
    }
}

impl Kv for RocksKv {
    fn get(&self, key: &[u8]) -> Result<Option<Vec<u8>>, StoreError> {
        let guard = self.read_guard();
        let db = guard.as_ref().ok_or(StoreError::Closed)?;
        Ok(db.get(key)?)
    }

    fn write(&self, batch: crate::kv::WriteBatch) -> Result<(), StoreError> {
        let guard = self.write_guard();
        let db = guard.as_ref().ok_or(StoreError::Closed)?;
        db.write_opt(batch.into_rocksdb(), &Self::sync_write_options())?;
        Ok(())
    }

    fn first_in(
        &self,
        start: &[u8],
        end: &[u8],
    ) -> Result<Option<(Vec<u8>, Vec<u8>)>, StoreError> {
        let guard = self.read_guard();
        let db = guard.as_ref().ok_or(StoreError::Closed)?;
        let mut iter = db.raw_iterator_opt(Self::bounded_read_options(start, end));
        iter.seek_to_first();
        if !iter.valid() {
            iter.status()?;
            return Ok(None);
        }
        Ok(Some((iter.key().unwrap_or_default().to_vec(), iter.value().unwrap_or_default().to_vec())))
    }

    fn last_in(&self, start: &[u8], end: &[u8]) -> Result<Option<(Vec<u8>, Vec<u8>)>, StoreError> {
        let guard = self.read_guard();
        let db = guard.as_ref().ok_or(StoreError::Closed)?;
        let mut iter = db.raw_iterator_opt(Self::bounded_read_options(start, end));
        iter.seek_to_last();
        if !iter.valid() {
            iter.status()?;
            return Ok(None);
        }
        Ok(Some((iter.key().unwrap_or_default().to_vec(), iter.value().unwrap_or_default().to_vec())))
    }

    fn range(&self, start: &[u8], end: &[u8]) -> Result<Vec<(Vec<u8>, Vec<u8>)>, StoreError> {
        let guard = self.read_guard();
        let db = guard.as_ref().ok_or(StoreError::Closed)?;
        let mut iter = db.raw_iterator_opt(Self::bounded_read_options(start, end));
        iter.seek_to_first();
        let mut out = Vec::new();
        while iter.valid() {
            let Some(key) = iter.key() else { break };
            let Some(value) = iter.value() else { break };
            out.push((key.to_vec(), value.to_vec()));
            iter.next();
        }
        iter.status()?;
        Ok(out)
    }
}
