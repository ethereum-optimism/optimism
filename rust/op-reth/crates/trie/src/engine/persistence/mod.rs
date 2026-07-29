//! Persistence layer for the live trie engine.
//!
//! `PersistenceHandle` is the caller-side handle used by the engine to
//! send save/unwind requests. `PersistenceService` is the background
//! worker that executes those requests sequentially on its own thread.

pub mod error;
pub mod handle;
#[cfg(feature = "metrics")]
pub mod metrics;
pub mod service;

pub use handle::PersistenceHandle;
