//! Backend for the preimage server.

mod offline;
pub use offline::OfflineHostBackend;

mod online;
pub use online::{HintHandler, OnlineHostBackend, OnlineHostBackendCfg};

#[allow(clippy::redundant_pub_crate)]
// SAFETY: `pub(crate)` is required to satisfy the workspace-level `unreachable_pub` lint.
pub(crate) mod util;
