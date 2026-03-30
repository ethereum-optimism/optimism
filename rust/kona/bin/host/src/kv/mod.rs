//! This module contains the [`KeyValueStore`] trait and concrete implementations of it.

use crate::Result;
use alloy_primitives::B256;
use std::{path::Path, sync::Arc};
use tokio::sync::RwLock;

mod mem;
pub use mem::MemoryKeyValueStore;

mod disk;
pub use disk::DiskKeyValueStore;

mod directory;
pub use directory::DirectoryKeyValueStore;

mod split;
pub use split::SplitKeyValueStore;

/// The filename used to record the storage format, for compatibility with op-challenger.
const FORMAT_FILENAME: &str = "kvformat";

/// The storage format for on-disk preimage data.
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq, clap::ValueEnum, serde::Serialize)]
pub enum DataFormat {
    /// Files stored in subdirectories with hex-encoded values.
    /// Compatible with op-challenger's `DataFormatDirectory`.
    #[default]
    Directory,
    /// RocksDB-backed storage.
    Rocksdb,
}

impl DataFormat {
    /// Returns the string identifier written to the `kvformat` marker file.
    const fn as_str(self) -> &'static str {
        match self {
            Self::Directory => "directory",
            Self::Rocksdb => "rocksdb",
        }
    }
}

/// Reads the `kvformat` marker file from the given directory. If the marker exists and contains
/// a supported format, returns that format. Otherwise, returns `default_format`. The marker file
/// is written by the individual store implementations (`DirectoryKeyValueStore`,
/// `DiskKeyValueStore`) when they initialize.
pub(crate) fn detect_data_format(data_dir: &Path, default_format: DataFormat) -> DataFormat {
    let format_path = data_dir.join(FORMAT_FILENAME);
    std::fs::read_to_string(&format_path).map_or(default_format, |contents| {
        match contents.as_str() {
            "directory" => DataFormat::Directory,
            "rocksdb" => DataFormat::Rocksdb,
            other => {
                tracing::warn!(format = other, "Unknown kvformat marker, using CLI default");
                default_format
            }
        }
    })
}

/// A type alias for a shared key-value store.
pub type SharedKeyValueStore = Arc<RwLock<dyn KeyValueStore + Send + Sync>>;

/// Creates a [`SharedKeyValueStore`] backed by disk or memory.
///
/// If `data_dir` is provided, the format is auto-detected from any existing `kvformat` marker file,
/// falling back to `default_format`. Otherwise a [`MemoryKeyValueStore`] is used.
pub(crate) fn create_key_value_store<L>(
    local_kv_store: L,
    data_dir: Option<&Path>,
    default_format: DataFormat,
) -> SharedKeyValueStore
where
    L: KeyValueStore + Send + Sync + 'static,
{
    match data_dir {
        Some(data_dir) => {
            let format = detect_data_format(data_dir, default_format);
            match format {
                DataFormat::Directory => {
                    let dir_kv_store = DirectoryKeyValueStore::new(data_dir);
                    Arc::new(RwLock::new(SplitKeyValueStore::new(local_kv_store, dir_kv_store)))
                }
                DataFormat::Rocksdb => {
                    let disk_kv_store = DiskKeyValueStore::new(data_dir.to_path_buf());
                    Arc::new(RwLock::new(SplitKeyValueStore::new(local_kv_store, disk_kv_store)))
                }
            }
        }
        None => {
            let mem_kv_store = MemoryKeyValueStore::new();
            Arc::new(RwLock::new(SplitKeyValueStore::new(local_kv_store, mem_kv_store)))
        }
    }
}

/// Describes the interface of a simple, synchronous key-value store.
pub trait KeyValueStore {
    /// Get the value associated with the given key.
    fn get(&self, key: B256) -> Option<Vec<u8>>;

    /// Set the value associated with the given key.
    fn set(&mut self, key: B256, value: Vec<u8>) -> Result<()>;
}

#[cfg(test)]
mod test {
    use super::*;
    use std::fs;

    #[test]
    fn detect_reads_existing_directory_marker() {
        let dir = tempfile::TempDir::new().unwrap();
        fs::write(dir.path().join("kvformat"), "directory").unwrap();

        let format = detect_data_format(dir.path(), DataFormat::Rocksdb);
        assert_eq!(format, DataFormat::Directory);
    }

    #[test]
    fn detect_reads_existing_rocksdb_marker() {
        let dir = tempfile::TempDir::new().unwrap();
        fs::write(dir.path().join("kvformat"), "rocksdb").unwrap();

        let format = detect_data_format(dir.path(), DataFormat::Directory);
        assert_eq!(format, DataFormat::Rocksdb);
    }

    #[test]
    fn detect_falls_back_to_default_when_no_marker() {
        let dir = tempfile::TempDir::new().unwrap();

        assert_eq!(detect_data_format(dir.path(), DataFormat::Directory), DataFormat::Directory);
        assert_eq!(detect_data_format(dir.path(), DataFormat::Rocksdb), DataFormat::Rocksdb);
    }

    #[test]
    fn detect_falls_back_to_default_for_unknown_format() {
        let dir = tempfile::TempDir::new().unwrap();
        fs::write(dir.path().join("kvformat"), "pebble").unwrap();

        let format = detect_data_format(dir.path(), DataFormat::Directory);
        assert_eq!(format, DataFormat::Directory);
    }
}
