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
pub use change_set::*;
pub mod kv;
pub use kv::*;
mod key;
pub use key::*;
mod value;
pub use value::*;

use alloy_primitives::{B256, BlockNumber};
use reth_db::{
    BlockNumberList, TableSet, TableType, TableViewer,
    table::{DupSort, TableInfo},
    tables,
};
use reth_primitives_traits::{Account, StorageEntry};
use reth_trie_common::{BranchNodeCompact, StorageTrieEntry, StoredNibbles, StoredNibblesSubKey};
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

    // ==================== V2 Tables ====================
    //
    // The v2 schema uses the 3-table-per-data-type pattern. All v2 tables are
    // prefixed with `V2` to clearly distinguish them from v1 tables and to ensure
    // each store only reads/writes its own tables.
    //
    //   - **Current state** tables hold the latest values for fast reads.
    //   - **ChangeSet** tables group changes by block number for efficient pruning/unwinding.
    //   - **History** tables store sharded bitmaps for historical lookups.
    //

    // -------------------- Proof Window --------------------

    /// V2 proof window tracking (independent of the v1 [`ProofWindow`] table).
    table V2ProofWindow {
        type Key = ProofWindowKey;
        type Value = BlockNumberHash;
    }

    // -------------------- Hashed Accounts --------------------

    /// Sharded history index for hashed accounts.
    ///
    /// Maps `ShardedKey<B256>` (hashed address + highest block number in shard)
    /// to a bitmap of block numbers that modified this account. Used for historical
    /// lookups: find the relevant block in the bitmap, then read the changeset.
    table V2HashedAccountsHistory {
        type Key = HashedAccountShardedKey;
        type Value = BlockNumberList;
    }

    /// Account changesets grouped by block number.
    ///
    /// Each entry stores the hashed address and the account state **before** the
    /// block was applied (`None` if the account didn't exist). Grouped by block
    /// number for efficient pruning (delete all entries for a block in one
    /// operation) and unwinding (restore old values on reorg).
    table V2HashedAccountChangeSets {
        type Key = BlockNumber;
        type Value = HashedAccountBeforeTx;
        type SubKey = B256;
    }

    /// Current state of all accounts, keyed by `keccak256(address)`.
    ///
    /// Holds the latest account data (nonce, balance, code hash, storage root).
    /// Primary read target for state root computation and proof generation —
    /// no version lookup needed.
    table V2HashedAccounts {
        type Key = B256;
        type Value = Account;
    }

    // -------------------- Hashed Storages --------------------

    /// Sharded history index for storage slots.
    ///
    /// Composite key of `(hashed_address, hashed_storage_key, highest_block_number)`.
    /// Maps to a bitmap of block numbers that modified this storage slot.
    table V2HashedStoragesHistory {
        type Key = HashedStorageShardedKey;
        type Value = BlockNumberList;
    }

    /// Storage changesets grouped by block number and account.
    ///
    /// Composite key of `(block_number, hashed_address)`. Each entry stores the
    /// hashed storage key and value **before** the block was applied.
    /// A value of [`U256::ZERO`](alloy_primitives::U256::ZERO) means the slot
    /// did not exist (needs to be removed on unwind).
    table V2HashedStorageChangeSets {
        type Key = BlockNumberHashedAddress;
        type Value = StorageEntry;
        type SubKey = B256;
    }

    /// Current storage values, keyed by hashed address with hashed storage key
    /// as the `DupSort` subkey.
    ///
    /// Holds the latest storage slot values for each account. Primary read target
    /// for storage proof generation.
    table V2HashedStorages {
        type Key = B256;
        type Value = StorageEntry;
        type SubKey = B256;
    }

    // -------------------- Account Trie --------------------

    /// Sharded history index for the account state trie.
    ///
    /// Maps `ShardedKey<StoredNibbles>` (trie path + highest block number in shard)
    /// to a bitmap of block numbers that modified this path.
    table V2AccountsTrieHistory {
        type Key = AccountTrieShardedKey;
        type Value = BlockNumberList;
    }

    /// Account trie changesets grouped by block number.
    ///
    /// Each entry stores the trie path and the branch node value **before** the
    /// block was applied (`None` if the node didn't exist). Enables efficient
    /// pruning and unwinding of trie state.
    table V2AccountTrieChangeSets {
        type Key = BlockNumber;
        type Value = TrieChangeSetsEntry;
        type SubKey = StoredNibblesSubKey;
    }

    /// Current state of the account Merkle Patricia Trie.
    ///
    /// Maps trie paths to the latest branch node. Primary read target during
    /// proof generation — no version lookup needed.
    table V2AccountsTrie {
        type Key = StoredNibbles;
        type Value = BranchNodeCompact;
    }

    // -------------------- Storage Trie --------------------

    /// Sharded history index for per-account storage tries.
    ///
    /// Composite key of `(hashed_address, trie_path, highest_block_number)`.
    /// Maps to a bitmap of block numbers that modified this storage trie node.
    table V2StoragesTrieHistory {
        type Key = StorageTrieShardedKey;
        type Value = BlockNumberList;
    }

    /// Storage trie changesets grouped by block number and account.
    ///
    /// Composite key of `(block_number, hashed_address)`. Each entry stores the
    /// trie path and the branch node value **before** the block was applied.
    table V2StorageTrieChangeSets {
        type Key = BlockNumberHashedAddress;
        type Value = TrieChangeSetsEntry;
        type SubKey = StoredNibblesSubKey;
    }

    /// Current state of each account's storage Merkle Patricia Trie.
    ///
    /// Keyed by hashed account address, with the trie path as the `DupSort` subkey.
    /// Holds the latest branch node for each path in each account's storage trie.
    table V2StoragesTrie {
        type Key = B256;
        type Value = StorageTrieEntry;
        type SubKey = StoredNibblesSubKey;
    }
}
