//! MDBX implementation of [`OpProofsStore`](crate::OpProofsStore).
//!
//! This module provides a complete MDBX implementation of the
//! [`OpProofsStore`](crate::OpProofsStore) trait. It uses the [`reth_db`]
//! crate for database interactions and defines the necessary tables and models for storing trie
//! branches, accounts, and storage leaves.

mod models;
pub use models::*;

mod store_v2;
pub use store_v2::{
    MdbxAccountCursor, MdbxAccountTrieCursor, MdbxAccountTrieSnapshotCursor,
    MdbxHashedAccountSnapshotCursor, MdbxHashedStorageSnapshotCursor, MdbxProofsProvider,
    MdbxProofsStorage, MdbxStorageCursor, MdbxStorageTrieCursor, MdbxStorageTrieSnapshotCursor,
};
