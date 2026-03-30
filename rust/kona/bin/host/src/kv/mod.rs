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
/// a supported format, returns that format. Otherwise, returns `default_format` and writes the
/// marker so that future opens (including by op-challenger) detect the format automatically.
pub fn detect_data_format(data_dir: &Path, default_format: DataFormat) -> DataFormat {
    let format_path = data_dir.join(FORMAT_FILENAME);
    match std::fs::read_to_string(&format_path) {
        Ok(contents) => match contents.as_str() {
            "directory" => DataFormat::Directory,
            "rocksdb" => DataFormat::Rocksdb,
            other => {
                tracing::warn!(format = other, "Unknown kvformat marker, using CLI default");
                default_format
            }
        },
        Err(_) => default_format,
    }
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

#[cfg(test)]
mod test {
    use super::*;
    use std::{env::temp_dir, fs};

    fn test_dir(name: &str) -> std::path::PathBuf {
        let dir = temp_dir().join(format!("kona_detect_format_{name}"));
        let _ = fs::remove_dir_all(&dir);
        fs::create_dir_all(&dir).unwrap();
        dir
    }

    #[test]
    fn detect_reads_existing_directory_marker() {
        let dir = test_dir("read_dir");
        fs::write(dir.join("kvformat"), "directory").unwrap();

        let format = detect_data_format(&dir, DataFormat::Rocksdb);
        assert_eq!(format, DataFormat::Directory);
    }

    #[test]
    fn detect_reads_existing_rocksdb_marker() {
        let dir = test_dir("read_rocksdb");
        fs::write(dir.join("kvformat"), "rocksdb").unwrap();

        let format = detect_data_format(&dir, DataFormat::Directory);
        assert_eq!(format, DataFormat::Rocksdb);
    }

    #[test]
    fn detect_falls_back_to_default_when_no_marker() {
        let dir = test_dir("no_marker");

        assert_eq!(detect_data_format(&dir, DataFormat::Directory), DataFormat::Directory);
        assert_eq!(detect_data_format(&dir, DataFormat::Rocksdb), DataFormat::Rocksdb);
    }

    #[test]
    fn detect_falls_back_to_default_for_unknown_format() {
        let dir = test_dir("unknown");
        fs::write(dir.join("kvformat"), "pebble").unwrap();

        let format = detect_data_format(&dir, DataFormat::Directory);
        assert_eq!(format, DataFormat::Directory);
    }
}
