//! This module contains the [`KeyValueStore`] trait and concrete implementations of it.

use crate::Result;
use alloy_primitives::B256;
use std::sync::Arc;
use tokio::sync::RwLock;

mod mem;
pub use mem::MemoryKeyValueStore;

mod disk;
pub use disk::DiskKeyValueStore;

mod directory;
pub use directory::DirectoryKeyValueStore;

mod split;
pub use split::SplitKeyValueStore;

/// The storage format for on-disk preimage data.
#[derive(Debug, Clone, Copy, Default, clap::ValueEnum)]
pub enum DataFormat {
    /// Files stored in subdirectories with hex-encoded values.
    /// Compatible with op-program's `DataFormatDirectory`.
    #[default]
    Directory,
    /// RocksDB-backed storage.
    Rocksdb,
}

/// A type alias for a shared key-value store.
pub type SharedKeyValueStore = Arc<RwLock<dyn KeyValueStore + Send + Sync>>;

/// Describes the interface of a simple, synchronous key-value store.
pub trait KeyValueStore {
    /// Get the value associated with the given key.
    fn get(&self, key: B256) -> Option<Vec<u8>>;

    /// Set the value associated with the given key.
    fn set(&mut self, key: B256, value: Vec<u8>) -> Result<()>;
}
