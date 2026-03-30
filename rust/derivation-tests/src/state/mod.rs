//! In-memory state database with trie root and proof computation.

mod db;
pub(crate) mod roots;

pub use db::{AccountState, StateSnapshot, TestStateDb, rebuild_cache_db};
pub use roots::{
    TrieNodeStore, compute_receipts_root, compute_transactions_root, generate_account_proof,
    storage_root_from_hashed_state,
};
