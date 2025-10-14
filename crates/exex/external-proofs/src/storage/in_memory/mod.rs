//! In-memory implementation of `OpProofsStorage` for testing purposes.
//!
//! This module provides a complete in-memory implementation of the
//! [`OpProofsStorage`](crate::storage::OpProofsStorage) trait that can be used for
//! testing and development. The implementation uses tokio async `RwLock`
//! for thread-safe concurrent access and stores all data in memory using `BTreeMap` collections.

mod store;
pub use store::InMemoryProofsStorage;

#[cfg(test)]
mod store_tests;
