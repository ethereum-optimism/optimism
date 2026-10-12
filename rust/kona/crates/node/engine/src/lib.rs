#![doc = include_str!("../README.md")]
#![doc(
    html_logo_url = "https://raw.githubusercontent.com/ethereum-optimism/optimism/develop/rust/kona/assets/square.png",
    html_favicon_url = "https://raw.githubusercontent.com/ethereum-optimism/optimism/develop/rust/kona/assets/favicon.ico",
    issue_tracker_base_url = "https://github.com/ethereum-optimism/optimism/issues/"
)]
#![cfg_attr(docsrs, feature(doc_cfg))]

//! ## Module Organization
//!
//! - **Client** - HTTP client for Engine API communication via [`EngineClient`]
//! - **Sync** - Sync starting point discovery via [`find_starting_forkchoice`]
//! - **Versions** - Engine API version selection via [`EngineForkchoiceVersion`],
//!   [`EngineNewPayloadVersion`], [`EngineGetPayloadVersion`]
//! - **Attributes** - Payload attribute validation via [`AttributesMatch`]
//! - **Block sink** - Canonical block notifications via [`ImportedBlockSink`]
//! - **Kinds** - Engine client type identification via [`EngineKind`]
//! - **Metrics** - Optional Prometheus metrics collection via [`Metrics`]

#[macro_use]
extern crate tracing;

mod attributes;
pub use attributes::{AttributesMatch, AttributesMismatch};

mod block_sink;
pub use block_sink::{ImportedBlockSink, NoopBlockSink};

mod client;
pub use client::{EngineClient, EngineClientBuilder, EngineClientError};

mod versions;
pub use versions::{EngineForkchoiceVersion, EngineGetPayloadVersion, EngineNewPayloadVersion};

mod kinds;
pub use kinds::EngineKind;

mod metrics;
pub use metrics::Metrics;

mod sync;
pub use sync::{L2ForkchoiceState, SyncStartError, find_starting_forkchoice};

#[cfg(any(test, feature = "test-utils"))]
/// Utilities that are useful when creating unit tests using structs within this library.
pub mod test_utils;
