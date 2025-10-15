//! Standalone crate for Optimism Trie Node storage.
//!
//! External storage for intermediary trie nodes that are otherwise discarded by pipeline and
//! live sync upon successful state root update. Storing these intermediary trie nodes enables
//! efficient retrieval of inputs to proof computation for duration of OP fault proof window.

#![doc(
    html_logo_url = "https://raw.githubusercontent.com/paradigmxyz/reth/main/assets/reth-docs.png",
    html_favicon_url = "https://avatars0.githubusercontent.com/u/97369466?s=256",
    issue_tracker_base_url = "https://github.com/paradigmxyz/reth/issues/"
)]
#![cfg_attr(docsrs, feature(doc_cfg))]
#![cfg_attr(not(test), warn(unused_crate_dependencies))]

pub mod api;
pub use api::{
    BlockStateDiff, OpProofsHashedCursor, OpProofsStorage, OpProofsStorageError,
    OpProofsStorageResult, OpProofsTrieCursor,
};

pub mod backfill;
pub use backfill::BackfillJob;

pub mod in_memory;
pub use in_memory::{
    InMemoryAccountCursor, InMemoryProofsStorage, InMemoryStorageCursor, InMemoryTrieCursor,
};

pub mod db;

pub mod proof;

pub mod provider;
