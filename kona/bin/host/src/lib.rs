#![doc = include_str!("../README.md")]
#![cfg_attr(docsrs, feature(doc_cfg, doc_auto_cfg))]

mod server;
pub use server::{PreimageServer, PreimageServerError};

mod kv;
pub use kv::{
    DiskKeyValueStore, KeyValueStore, MemoryKeyValueStore, SharedKeyValueStore, SplitKeyValueStore,
};

mod backend;
pub use backend::{HintHandler, OfflineHostBackend, OnlineHostBackend, OnlineHostBackendCfg};

pub mod eth;

#[cfg(feature = "single")]
pub mod single;

#[cfg(feature = "interop")]
pub mod interop;
