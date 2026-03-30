//! Contains a concrete implementation of the [`KeyValueStore`] trait that stores data on disk
//! using a directory-based layout compatible with op-challenger's `DataFormatDirectory`.

use super::{DataFormat, FORMAT_FILENAME, KeyValueStore};
use crate::{HostError, Result};
use alloy_primitives::{B256, hex};
use std::{
    fs,
    io::Write,
    path::{Path, PathBuf},
};

/// A key-value store that writes preimages as hex-encoded files in subdirectories.
///
/// Layout is compatible with op-challenger's `directoryKV`:
/// - Key `0x0123456789...abc` maps to `<dir>/0123/456789...abc.txt`
/// - Values are hex-encoded on disk
/// - A `kvformat` marker file containing `"directory"` is written for op-challenger compatibility
#[derive(Debug)]
pub struct DirectoryKeyValueStore {
    data_directory: PathBuf,
}

impl DirectoryKeyValueStore {
    /// Create a new [`DirectoryKeyValueStore`] with the given data directory.
    pub fn new(data_directory: &Path) -> Self {
        fs::create_dir_all(data_directory)
            .unwrap_or_else(|e| panic!("Failed to create directory {data_directory:?}: {e}"));

        let format_path = data_directory.join(FORMAT_FILENAME);
        if !format_path.exists() {
            fs::write(&format_path, DataFormat::Directory.as_str())
                .unwrap_or_else(|e| panic!("Failed to write kvformat marker: {e}"));
        }

        Self { data_directory: data_directory.to_path_buf() }
    }

    /// Returns the file path for the given key.
    ///
    /// The hex key (without `0x` prefix) is split into a 4-char directory prefix and the
    /// remainder as the filename with `.txt` extension. This matches op-challenger's layout.
    fn key_path(&self, key: B256) -> PathBuf {
        let hex_key = format!("{key:x}");
        let (dir_part, file_part) = hex_key.split_at(4);
        self.data_directory.join(dir_part).join(format!("{file_part}.txt"))
    }
}

impl KeyValueStore for DirectoryKeyValueStore {
    fn get(&self, key: B256) -> Option<Vec<u8>> {
        let path = self.key_path(key);
        let data = fs::read_to_string(&path).ok()?;
        match hex::decode(&data) {
            Ok(value) => Some(value),
            Err(e) => {
                tracing::warn!(key = %key, path = %path.display(), error = %e, "Corrupt preimage file, ignoring");
                None
            }
        }
    }

    fn set(&mut self, key: B256, value: Vec<u8>) -> Result<()> {
        let path = self.key_path(key);
        let parent = path.parent().ok_or_else(|| {
            HostError::KeyValueSetFailed(format!("no parent directory for {path:?}"))
        })?;
        fs::create_dir_all(parent).map_err(|e| {
            HostError::KeyValueSetFailed(format!("failed to create directory {parent:?}: {e}"))
        })?;

        // Write to a temp file and rename for atomicity — a crash during fs::write could leave
        // a partially written (corrupt) file, but rename is atomic on POSIX.
        let mut tmp = tempfile::NamedTempFile::new_in(parent).map_err(|e| {
            HostError::KeyValueSetFailed(format!("failed to create temp file in {parent:?}: {e}"))
        })?;
        tmp.write_all(hex::encode(&value).as_bytes())
            .map_err(|e| HostError::KeyValueSetFailed(format!("failed to write temp file: {e}")))?;
        tmp.persist(&path).map_err(|e| {
            HostError::KeyValueSetFailed(format!("failed to rename temp file to {path:?}: {e}"))
        })?;
        Ok(())
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

    proptest! {
        #![proptest_config(Config::with_cases(16))]

        #[test]
        fn directory_kv_roundtrip(k_v in hash_map(any::<[u8; 32]>(), vec(any::<u8>(), 0..128), 1..128)) {
            let tempdir = tempfile::TempDir::new().unwrap();
            let mut kv = DirectoryKeyValueStore::new(tempdir.path());

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
        let tempdir = tempfile::TempDir::new().unwrap();
        let _kv = DirectoryKeyValueStore::new(tempdir.path());

        let marker = std::fs::read_to_string(tempdir.path().join("kvformat")).unwrap();
        assert_eq!(marker, "directory");
    }

    #[test]
    fn key_path_layout() {
        let tempdir = tempfile::TempDir::new().unwrap();
        let kv = DirectoryKeyValueStore::new(tempdir.path());

        let key = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
            .parse::<B256>()
            .unwrap();
        let path = kv.key_path(key);

        let expected = tempdir
            .path()
            .join("0123")
            .join("456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef.txt");
        assert_eq!(path, expected);
    }
}
