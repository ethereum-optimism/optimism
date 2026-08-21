//! The key/value backend seam the interop stores are written against.
//!
//! Every store here is a record layout plus an append cursor over an ordered byte-keyed map.
//! Naming that map explicitly keeps the choice of storage engine a contained one: [`RocksKv`] is
//! the on-disk backend, [`MemoryKv`] is the one tests and callers that want no files use, and a
//! third backend is a new implementation of this trait rather than a rewrite of the stores.
//!
//! [`RocksKv`]: crate::RocksKv

use crate::error::StoreError;
use std::{
    collections::BTreeMap,
    fmt::Debug,
    sync::{PoisonError, RwLock},
};

/// A single mutation in a [`WriteBatch`].
#[derive(Debug, Clone, PartialEq, Eq)]
enum Op {
    /// Write `value` at `key`.
    Put(Vec<u8>, Vec<u8>),
    /// Remove `key`.
    Delete(Vec<u8>),
    /// Remove every key in `[start, end)`.
    DeleteRange(Vec<u8>, Vec<u8>),
}

/// A set of mutations applied atomically and durably by [`Kv::write`].
///
/// Atomicity is what makes the stores' invariants recoverable: a sealed block and the cursor
/// that names it, or a WAL slot and the record it describes, land together or not at all.
#[derive(Debug, Default, Clone, PartialEq, Eq)]
pub struct WriteBatch {
    ops: Vec<Op>,
}

impl WriteBatch {
    /// Returns an empty batch.
    pub const fn new() -> Self {
        Self { ops: Vec::new() }
    }

    /// Queues a write of `value` at `key`.
    pub fn put(&mut self, key: impl Into<Vec<u8>>, value: impl Into<Vec<u8>>) {
        self.ops.push(Op::Put(key.into(), value.into()));
    }

    /// Queues a removal of `key`.
    pub fn delete(&mut self, key: impl Into<Vec<u8>>) {
        self.ops.push(Op::Delete(key.into()));
    }

    /// Queues a removal of every key in `[start, end)`.
    pub fn delete_range(&mut self, start: impl Into<Vec<u8>>, end: impl Into<Vec<u8>>) {
        self.ops.push(Op::DeleteRange(start.into(), end.into()));
    }

    /// Returns whether the batch has no mutations queued.
    pub fn is_empty(&self) -> bool {
        self.ops.is_empty()
    }

    /// Lowers the batch into a [`rocksdb::WriteBatch`], preserving mutation order.
    #[cfg(feature = "rocksdb")]
    pub(crate) fn into_rocksdb(self) -> rocksdb::WriteBatch {
        let mut batch = rocksdb::WriteBatch::default();
        for op in self.ops {
            match op {
                Op::Put(key, value) => batch.put(key, value),
                Op::Delete(key) => batch.delete(key),
                Op::DeleteRange(start, end) => batch.delete_range(start, end),
            }
        }
        batch
    }
}

/// An ordered, byte-keyed store with atomic, durable batch writes.
///
/// Keys are compared lexicographically, so the stores encode their ordering fields as
/// big-endian integers to make lexicographic order match numeric order.
pub trait Kv: Debug + Send + Sync + 'static {
    /// Returns the value at `key`, or [`None`] if the key is absent.
    fn get(&self, key: &[u8]) -> Result<Option<Vec<u8>>, StoreError>;

    /// Applies every mutation in `batch` atomically, returning only once the write is durable.
    fn write(&self, batch: WriteBatch) -> Result<(), StoreError>;

    /// Returns the first entry with a key in `[start, end)`, in ascending key order.
    fn first_in(
        &self,
        start: &[u8],
        end: &[u8],
    ) -> Result<Option<(Vec<u8>, Vec<u8>)>, StoreError>;

    /// Returns the last entry with a key in `[start, end)`, in ascending key order.
    fn last_in(&self, start: &[u8], end: &[u8]) -> Result<Option<(Vec<u8>, Vec<u8>)>, StoreError>;

    /// Returns every entry with a key in `[start, end)`, in ascending key order.
    ///
    /// Callers materialise the whole range, so this is only for scans over a store that is
    /// small by construction.
    fn range(&self, start: &[u8], end: &[u8]) -> Result<Vec<(Vec<u8>, Vec<u8>)>, StoreError>;
}

/// An in-memory [`Kv`].
///
/// Writes are atomic but not durable: nothing survives the process. It is the backend for unit
/// tests and for callers that want the store semantics without files.
#[derive(Debug, Default)]
pub struct MemoryKv {
    entries: RwLock<BTreeMap<Vec<u8>, Vec<u8>>>,
}

impl MemoryKv {
    /// Returns an empty store.
    pub fn new() -> Self {
        Self::default()
    }
}

impl Kv for MemoryKv {
    fn get(&self, key: &[u8]) -> Result<Option<Vec<u8>>, StoreError> {
        let entries = self.entries.read().unwrap_or_else(PoisonError::into_inner);
        Ok(entries.get(key).cloned())
    }

    fn write(&self, batch: WriteBatch) -> Result<(), StoreError> {
        let mut entries = self.entries.write().unwrap_or_else(PoisonError::into_inner);
        for op in batch.ops {
            match op {
                Op::Put(key, value) => {
                    entries.insert(key, value);
                }
                Op::Delete(key) => {
                    entries.remove(&key);
                }
                Op::DeleteRange(start, end) => {
                    let doomed: Vec<_> = entries.range(start..end).map(|(k, _)| k.clone()).collect();
                    for key in doomed {
                        entries.remove(&key);
                    }
                }
            }
        }
        Ok(())
    }

    fn first_in(
        &self,
        start: &[u8],
        end: &[u8],
    ) -> Result<Option<(Vec<u8>, Vec<u8>)>, StoreError> {
        let entries = self.entries.read().unwrap_or_else(PoisonError::into_inner);
        Ok(entries.range(start.to_vec()..end.to_vec()).next().map(|(k, v)| (k.clone(), v.clone())))
    }

    fn last_in(&self, start: &[u8], end: &[u8]) -> Result<Option<(Vec<u8>, Vec<u8>)>, StoreError> {
        let entries = self.entries.read().unwrap_or_else(PoisonError::into_inner);
        Ok(entries
            .range(start.to_vec()..end.to_vec())
            .next_back()
            .map(|(k, v)| (k.clone(), v.clone())))
    }

    fn range(&self, start: &[u8], end: &[u8]) -> Result<Vec<(Vec<u8>, Vec<u8>)>, StoreError> {
        let entries = self.entries.read().unwrap_or_else(PoisonError::into_inner);
        Ok(entries
            .range(start.to_vec()..end.to_vec())
            .map(|(k, v)| (k.clone(), v.clone()))
            .collect())
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn batch_writes_are_visible_and_ordered() {
        let kv = MemoryKv::new();
        let mut batch = WriteBatch::new();
        assert!(batch.is_empty());
        batch.put(vec![0, 2], vec![2]);
        batch.put(vec![0, 1], vec![1]);
        batch.put(vec![0, 3], vec![3]);
        assert!(!batch.is_empty());
        kv.write(batch).unwrap();

        assert_eq!(kv.get(&[0, 2]).unwrap(), Some(vec![2]));
        assert_eq!(kv.first_in(&[0, 0], &[1]).unwrap().unwrap().1, vec![1]);
        assert_eq!(kv.last_in(&[0, 0], &[1]).unwrap().unwrap().1, vec![3]);
        assert_eq!(kv.range(&[0, 0], &[1]).unwrap().len(), 3);
    }

    #[test]
    fn delete_range_is_half_open() {
        let kv = MemoryKv::new();
        let mut batch = WriteBatch::new();
        for i in 0..5u8 {
            batch.put(vec![0, i], vec![i]);
        }
        kv.write(batch).unwrap();

        let mut batch = WriteBatch::new();
        batch.delete_range(vec![0, 1], vec![0, 3]);
        batch.delete(vec![0, 4]);
        kv.write(batch).unwrap();

        let remaining: Vec<_> =
            kv.range(&[0, 0], &[1]).unwrap().into_iter().map(|(k, _)| k[1]).collect();
        assert_eq!(remaining, vec![0, 3]);
    }

    #[test]
    fn range_bounds_exclude_other_columns() {
        let kv = MemoryKv::new();
        let mut batch = WriteBatch::new();
        batch.put(vec![0, 9], vec![1]);
        batch.put(vec![1, 0], vec![2]);
        kv.write(batch).unwrap();

        assert_eq!(kv.range(&[0], &[1]).unwrap().len(), 1);
        assert_eq!(kv.last_in(&[0], &[1]).unwrap().unwrap().0, vec![0, 9]);
    }
}
