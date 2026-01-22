//! MDBX implementation of [`OpProofsStore`](crate::OpProofsStore).
//!
//! This module provides a complete MDBX implementation of the
//! [`OpProofsStore`](crate::OpProofsStore) trait. It uses the [`reth_db`] crate for
//! database interactions and defines the necessary tables and models for storing trie branches,
//! accounts, and storage leaves.

mod block;
pub use block::*;
mod version;
pub use version::*;
mod storage;
pub use storage::*;
mod change_set;
pub(crate) mod kv;
pub use change_set::*;
pub use kv::*;

use alloy_primitives::B256;
use reth_db::{
    table::{DupSort, TableInfo},
    tables, TableSet, TableType, TableViewer,
};
use reth_primitives_traits::Account;
use reth_trie_common::{BranchNodeCompact, StoredNibbles};
use std::fmt;

tables! {
    /// Stores historical branch nodes for the account state trie.
    ///
    /// Each entry maps a compact-encoded trie path (`StoredNibbles`) to its versioned branch node.
    /// Multiple versions of the same node are stored using the block number as a subkey.
    table AccountTrieHistory {
        type Key = StoredNibbles;
        type Value = VersionedValue<BranchNodeCompact>;
        type SubKey = u64; // block number
    }

    /// Stores historical branch nodes for the storage trie of each account.
    ///
    /// Each entry is identified by a composite key combining the account’s hashed address and the
    /// compact-encoded trie path. Versions are tracked using block numbers as subkeys.
    table StorageTrieHistory {
        type Key = StorageTrieKey;
        type Value = VersionedValue<BranchNodeCompact>;
        type SubKey = u64; // block number
    }

    /// Stores versioned account state across block history.
    ///
    /// Each entry maps a hashed account address to its serialized account data (balance, nonce,
    /// code hash, storage root).
    table HashedAccountHistory {
        type Key = B256;
        type Value = VersionedValue<Account>;
        type SubKey = u64; // block number
    }

    /// Stores versioned storage state across block history.
    ///
    /// Each entry maps a composite key of (hashed address, storage key) to its stored value.
    /// Used for reconstructing contract storage at any historical block height.
    table HashedStorageHistory {
        type Key = HashedStorageKey;
        type Value = VersionedValue<StorageValue>;
        type SubKey = u64; // block number
    }

    /// Tracks the active proof window in the external historical storage.
    ///
    /// Stores the earliest and latest block numbers (and corresponding hashes)
    /// for which historical trie data is retained.
    table ProofWindow {
      type Key = ProofWindowKey;
      type Value = BlockNumberHash;
    }

    /// A reverse mapping of block numbers to a keys of the tables.
    /// This is used for efficiently locating data by block number.
    table BlockChangeSet {
        type Key = u64; // Block number
        type Value = ChangeSet;
    }
}
