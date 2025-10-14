//! MDBX tables and models for external proofs storage.
use alloy_primitives::B256;
use reth_db_api::{
    table::{DupSort, TableInfo},
    tables, TableSet, TableType, TableViewer,
};
use reth_primitives_traits::Account;
use reth_trie::{BranchNodeCompact, StoredNibbles};
use std::fmt;

mod block;
pub use block::*;
mod version;
pub use version::*;

mod storage;
pub use storage::*;

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
        type Key = StorageTrieSubKey;
        type Value = VersionedValue<BranchNodeCompact>;
        type SubKey = u64; // block number
    }

    /// Stores versioned account state across block history.
    ///
    /// Each entry maps a hashed account address to its serialized account data (balance, nonce,
    /// code hash, storage root).
    table HashedAccountHistory {
        type Key = B256;
        type Value = VersionedValue<MaybeDeleted<Account>>;
        type SubKey = u64; // block number
    }

    /// Stores versioned storage state across block history.
    ///
    /// Each entry maps a composite key of (hashed address, storage key) to its stored value.
    /// Used for reconstructing contract storage at any historical block height.
    table HashedStorageHistory {
        type Key = HashedStorageSubKey;
        type Value = VersionedValue<B256>;
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
}
