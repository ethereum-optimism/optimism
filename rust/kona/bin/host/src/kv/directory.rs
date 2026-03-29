//! Contains a concrete implementation of the [`KeyValueStore`] trait that stores data on disk
//! using a directory-based layout compatible with op-program's `DataFormatDirectory`.

use super::KeyValueStore;
use crate::{HostError, Result};
use alloy_primitives::{hex, B256};
use std::{fs, path::PathBuf};

/// The filename used to record the storage format, matching op-program's convention.
const FORMAT_FILENAME: &str = "kvformat";

/// The format identifier written to the marker file, matching op-program's `DataFormatDirectory`.
const FORMAT_VALUE: &str = "directory";

/// A key-value store that writes preimages as hex-encoded files in subdirectories.
///
/// Layout matches op-program's `directoryKV`:
/// - Key `0x0123456789...abc` maps to `<dir>/0123/456789...abc.txt`
/// - Values are hex-encoded on disk
/// - A `kvformat` marker file containing `"directory"` is written for op-challenger compatibility
#[derive(Debug)]
pub struct DirectoryKeyValueStore {
    data_directory: PathBuf,
}

impl DirectoryKeyValueStore {
    /// Create a new [`DirectoryKeyValueStore`] with the given data directory.
    pub fn new(data_directory: PathBuf) -> Self {
        fs::create_dir_all(&data_directory)
            .unwrap_or_else(|e| panic!("Failed to create directory {data_directory:?}: {e}"));

        let format_path = data_directory.join(FORMAT_FILENAME);
        if !format_path.exists() {
            fs::write(&format_path, FORMAT_VALUE)
                .unwrap_or_else(|e| panic!("Failed to write kvformat marker: {e}"));
        }

        Self { data_directory }
    }

    /// Returns the file path for the given key.
    ///
    /// Matches op-program's `directoryKV.pathKey`: the hex key (without `0x` prefix) is split
    /// into a 4-char directory prefix and the remainder as the filename with `.txt` extension.
    fn key_path(&self, key: B256) -> PathBuf {
        let hex_key = format!("{key:x}");
        let (dir_part, file_part) = hex_key.split_at(4);
        self.data_directory.join(dir_part).join(format!("{file_part}.txt"))
    }
}

impl KeyValueStore for DirectoryKeyValueStore {
    fn get(&self, key: B256) -> Option<Vec<u8>> {
        let data = fs::read_to_string(self.key_path(key)).ok()?;
        hex::decode(data).ok()
    }

    fn set(&mut self, key: B256, value: Vec<u8>) -> Result<()> {
        let path = self.key_path(key);
        if let Some(parent) = path.parent() {
            fs::create_dir_all(parent).map_err(|e| {
                HostError::KeyValueSetFailed(format!("failed to create directory {parent:?}: {e}"))
            })?;
        }
        fs::write(&path, hex::encode(&value))
            .map_err(|e| HostError::KeyValueSetFailed(format!("failed to write {path:?}: {e}")))
    }
}

#[cfg(test)]
mod test {
    use super::DirectoryKeyValueStore;
    use crate::kv::KeyValueStore;
    use alloy_primitives::B256;
    use proptest::{
        arbitrary::any,
        collection::{hash_map, vec},
        proptest,
        test_runner::Config,
    };
    use std::env::temp_dir;

    proptest! {
        #![proptest_config(Config::with_cases(16))]

        #[test]
        fn directory_kv_roundtrip(k_v in hash_map(any::<[u8; 32]>(), vec(any::<u8>(), 0..128), 1..128)) {
            let tempdir = temp_dir();
            let mut kv = DirectoryKeyValueStore::new(tempdir);

            for (k, v) in &k_v {
                kv.set((*k).into(), v.clone()).unwrap();
            }

            for (k, v) in &k_v {
                let key: B256 = (*k).into();
                assert_eq!(kv.get(key).unwrap(), *v);
            }
        }
    }

    #[test]
    fn writes_kvformat_marker() {
        let tempdir = temp_dir().join("kona_test_kvformat");
        let _kv = DirectoryKeyValueStore::new(tempdir.clone());

        let marker = std::fs::read_to_string(tempdir.join("kvformat")).unwrap();
        assert_eq!(marker, "directory");
    }

    #[test]
    fn key_path_layout() {
        let tempdir = temp_dir().join("kona_test_layout");
        let kv = DirectoryKeyValueStore::new(tempdir);

        let key = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
            .parse::<B256>()
            .unwrap();
        let path = kv.key_path(key);

        assert!(path.to_str().unwrap().contains("0123"));
        assert!(path.file_name().unwrap().to_str().unwrap().ends_with(".txt"));
    }
}
