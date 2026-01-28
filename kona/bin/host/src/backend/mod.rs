//! Backend for the preimage server.

mod offline;
pub use offline::OfflineHostBackend;

mod online;
pub use online::{HintHandler, OnlineHostBackend, OnlineHostBackendCfg};

pub(crate) mod util;
