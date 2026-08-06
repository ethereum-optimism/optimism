//! MDBX implementation of [`OpProofsStore`](crate::OpProofsStore).
//!
//! This module provides a complete MDBX implementation of the
//! [`OpProofsStore`](crate::OpProofsStore) trait. It uses the [`reth_db`] crate for
//! database interactions and defines the necessary tables and models for storing trie branches,
//! accounts, and storage leaves.

mod block;
pub use block::*;
mod storage;
pub use storage::*;
mod key;
pub use key::*;
mod value;
pub use value::*;
mod snapshot;
pub use snapshot::*;

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
    // The v2 schema uses the 3-table-per-data-type pattern. Table names retain
    // their `V2` prefix for compatibility with existing proofs databases.
    //
    //   - **Current state** tables hold the latest values for fast reads.
    //   - **ChangeSet** tables group changes by block number for efficient pruning/unwinding.
    //   - **History** tables store sharded bitmaps for historical lookups.
    //

    // -------------------- Proof Window --------------------

    /// Tracks the active proof window.
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

    /// Snapshot of [`V2AccountsTrie`] reflecting trie state at block x.
    ///
    /// Same schema as [`V2AccountsTrie`]. Populated by `SnapshotInitJob`.
    table V2AccountsTrieSnapshot {
        type Key = StoredNibbles;
        type Value = BranchNodeCompact;
    }

    /// Snapshot of [`V2StoragesTrie`] reflecting trie state at block x.
    ///
    /// Same schema as [`V2StoragesTrie`].
    table V2StoragesTrieSnapshot {
        type Key = B256;
        type Value = StorageTrieEntry;
        type SubKey = StoredNibblesSubKey;
    }

    /// Snapshot of [`V2HashedAccounts`] reflecting hashed-account leaves at the
    /// snapshot anchor block.
    table V2HashedAccountsSnapshot {
        type Key = B256;
        type Value = Account;
    }

    /// Snapshot of [`V2HashedStorages`] reflecting hashed-storage leaves at the
    /// snapshot anchor block. Same shape as [`V2HashedStorages`].
    table V2HashedStoragesSnapshot {
        type Key = B256;
        type Value = StorageEntry;
        type SubKey = B256;
    }

    /// Single-row metadata for the snapshot: which block its trie state
    /// reflects, and whether it's [`SnapshotStatus::Ready`] for reads.
    table V2SnapshotMeta {
        type Key = SnapshotMetaKey;
        type Value = SnapshotMeta;
    }
}
