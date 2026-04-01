#![doc = include_str!("../README.md")]
#![cfg_attr(docsrs, feature(doc_cfg))]

mod error;
pub use error::{HostError, Result};

mod server;
pub use server::{PreimageServer, PreimageServerError};

pub(crate) mod kv;
pub use kv::{
    DataFormat, DirectoryKeyValueStore, DiskKeyValueStore, KeyValueStore, MemoryKeyValueStore,
    SharedKeyValueStore, SplitKeyValueStore,
};

mod backend;
pub use backend::{HintHandler, OfflineHostBackend, OnlineHostBackend, OnlineHostBackendCfg};

pub mod eth;

#[cfg(feature = "single")]
pub mod single;

#[cfg(feature = "interop")]
pub mod interop;
