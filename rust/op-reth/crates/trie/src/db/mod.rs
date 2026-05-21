//! MDBX implementation of [`OpProofsStore`](crate::OpProofsStore).
//!
//! This module provides a complete MDBX implementation of the
//! [`OpProofsStore`](crate::OpProofsStore) trait. It uses the [`reth_db`]
//! crate for database interactions and defines the necessary tables and models for storing trie
//! branches, accounts, and storage leaves.

mod models;
pub use models::*;

mod store;
pub use store::{MdbxProofsProvider, MdbxProofsStorage};

mod cursor;
pub use cursor::{
    BlockNumberVersionedCursor, MdbxAccountCursor, MdbxStorageCursor, MdbxTrieCursor,
};

mod store_v2;
pub use store_v2::{
    MdbxProofsProviderV2, MdbxProofsStorageV2, V2AccountCursor, V2AccountTrieCursor,
    V2StorageCursor, V2StorageTrieCursor,
};
