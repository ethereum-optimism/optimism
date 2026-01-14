//! Contains a concrete implementation of the [KeyValueStore] trait that stores data on disk
//! using [rocksdb].

use super::{KeyValueStore, MemoryKeyValueStore};
use alloy_primitives::B256;
use anyhow::{Result, anyhow};
use rocksdb::{DB, Options};
use std::path::PathBuf;

/// A simple, synchronous key-value store that stores data on disk.
#[derive(Debug)]
pub struct DiskKeyValueStore {
    data_directory: PathBuf,
    db: DB,
}

impl DiskKeyValueStore {
    /// Create a new [DiskKeyValueStore] with the given data directory.
    pub fn new(data_directory: PathBuf) -> Self {
        let db = DB::open(&Self::get_db_options(), data_directory.as_path())
            .unwrap_or_else(|e| panic!("Failed to open database at {data_directory:?}: {e}"));

        Self { data_directory, db }
    }

    /// Gets the [Options] for the underlying RocksDB instance.
    fn get_db_options() -> Options {
        let mut options = Options::default();
        options.set_compression_type(rocksdb::DBCompressionType::Snappy);
        options.create_if_missing(true);
        options
    }
}

impl KeyValueStore for DiskKeyValueStore {
    fn get(&self, key: alloy_primitives::B256) -> Option<Vec<u8>> {
        self.db.get(*key).ok()?
    }

    fn set(&mut self, key: alloy_primitives::B256, value: Vec<u8>) -> Result<()> {
        self.db.put(*key, value).map_err(|e| anyhow!("Failed to set key-value pair: {e}"))
    }
}

impl Drop for DiskKeyValueStore {
    fn drop(&mut self) {
        let _ = DB::destroy(&Self::get_db_options(), self.data_directory.as_path());
    }
}

impl TryFrom<DiskKeyValueStore> for MemoryKeyValueStore {
    type Error = anyhow::Error;

    fn try_from(disk_store: DiskKeyValueStore) -> Result<Self> {
        let mut memory_store = Self::new();
        let mut db_iter = disk_store.db.full_iterator(rocksdb::IteratorMode::Start);

        while let Some(Ok((key, value))) = db_iter.next() {
            memory_store.set(
                B256::try_from(key.as_ref())
                    .map_err(|e| anyhow!("Failed to convert slice to B256: {e}"))?,
                value.to_vec(),
            )?;
        }

        Ok(memory_store)
    }
}

#[cfg(test)]
mod test {
    use super::DiskKeyValueStore;
    use crate::kv::{KeyValueStore, MemoryKeyValueStore};
    use proptest::{
        arbitrary::any,
        collection::{hash_map, vec},
        proptest,
        test_runner::Config,
    };
    use std::env::temp_dir;

    proptest! {
        #![proptest_config(Config::with_cases(16))]

        /// Test that converting from a [DiskKeyValueStore] to a [MemoryKeyValueStore] is lossless.
        #[test]
        fn convert_disk_kv_to_mem_kv(k_v in hash_map(any::<[u8; 32]>(), vec(any::<u8>(), 0..128), 1..128)) {
            let tempdir = temp_dir();
            let mut disk_kv = DiskKeyValueStore::new(tempdir);
            k_v.iter().for_each(|(k, v)| {
                disk_kv.set(k.into(), v.to_vec()).unwrap();
            });

            let mem_kv = MemoryKeyValueStore::try_from(disk_kv).unwrap();
            for (k, v) in k_v {
                assert_eq!(mem_kv.get(k.into()).unwrap(), v.to_vec());
            }
        }
    }
}
