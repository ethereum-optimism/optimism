#![doc = include_str!("../README.md")]
#![doc(
    html_logo_url = "https://raw.githubusercontent.com/ethereum-optimism/optimism/develop/rust/kona/assets/square.png",
    html_favicon_url = "https://raw.githubusercontent.com/ethereum-optimism/optimism/develop/rust/kona/assets/favicon.ico",
    issue_tracker_base_url = "https://github.com/ethereum-optimism/optimism/issues/"
)]
#![cfg_attr(docsrs, feature(doc_cfg))]

mod archive;
mod checksum;
mod encoding;
mod error;
mod kv;
mod logs;
mod verified;

#[cfg(feature = "rocksdb")]
mod rocks;

pub use archive::{ArchivedOutput, OutputArchive};
pub use checksum::{ChecksumArgs, MessageChecksum, log_hash, log_to_log_hash};
pub use error::StoreError;
pub use kv::{Entry, Kv, MemoryKv, WriteBatch};
pub use logs::{BlockSeal, ContainsQuery, LogStore, LogsDb, OpenedBlock, StoredExecutingMessage};
pub use verified::{InvalidHead, PendingTransition, RoundResult, VerifiedResult, VerifiedStore};

#[cfg(feature = "rocksdb")]
pub use rocks::RocksKv;

#[cfg(feature = "rocksdb")]
mod on_disk {
    use crate::{
        archive::OutputArchive, error::StoreError, logs::LogStore, rocks::RocksKv,
        verified::VerifiedStore,
    };
    use alloy_primitives::ChainId;
    use std::path::Path;

    /// The verified store on disk.
    pub type RocksVerifiedStore = VerifiedStore<RocksKv>;
    /// One chain's log store on disk.
    pub type RocksLogStore = LogStore<RocksKv>;
    /// The invalidated-output archive on disk.
    pub type RocksOutputArchive = OutputArchive<RocksKv>;

    /// Directory name of the verified store within an interop data directory.
    pub const VERIFIED_DIR: &str = "verified";
    /// Directory name holding the per-chain log stores.
    pub const LOGS_DIR: &str = "logs";
    /// Directory name of the invalidated-output archive.
    pub const ARCHIVE_DIR: &str = "invalidated-outputs";

    /// Opens the verified store under `data_dir`.
    pub fn open_verified_store(
        data_dir: impl AsRef<Path>,
    ) -> Result<RocksVerifiedStore, StoreError> {
        VerifiedStore::new(RocksKv::open(data_dir.as_ref().join(VERIFIED_DIR))?)
    }

    /// Opens `chain_id`'s log store under `data_dir`.
    ///
    /// Each chain gets its own database, so one chain's log history can be cleared and
    /// re-backfilled without touching another's.
    pub fn open_log_store(
        data_dir: impl AsRef<Path>,
        chain_id: ChainId,
    ) -> Result<RocksLogStore, StoreError> {
        let path = data_dir.as_ref().join(LOGS_DIR).join(format!("chain-{chain_id}"));
        LogStore::new(chain_id, RocksKv::open(path)?)
    }

    /// Opens the invalidated-output archive under `data_dir`.
    ///
    /// Its own database: it is the only store here whose loss is unrecoverable, so its files are
    /// separable for backup and cannot be discarded along with a re-derivable one.
    pub fn open_output_archive(
        data_dir: impl AsRef<Path>,
    ) -> Result<RocksOutputArchive, StoreError> {
        Ok(OutputArchive::new(RocksKv::open(data_dir.as_ref().join(ARCHIVE_DIR))?))
    }
}

#[cfg(feature = "rocksdb")]
pub use on_disk::{
    ARCHIVE_DIR, LOGS_DIR, RocksLogStore, RocksOutputArchive, RocksVerifiedStore, VERIFIED_DIR,
    open_log_store, open_output_archive, open_verified_store,
};
