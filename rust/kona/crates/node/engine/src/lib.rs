#![doc = include_str!("../README.md")]
#![doc(
    html_logo_url = "https://raw.githubusercontent.com/ethereum-optimism/optimism/develop/rust/kona/assets/square.png",
    html_favicon_url = "https://raw.githubusercontent.com/ethereum-optimism/optimism/develop/rust/kona/assets/favicon.ico",
    issue_tracker_base_url = "https://github.com/ethereum-optimism/optimism/issues/"
)]
#![cfg_attr(docsrs, feature(doc_cfg))]

//! ## Architecture
//!
//! The engine performs operations atomically, one at a time:
//!
//! ```text
//! ┌─────────────┐    ┌──────────────┐    ┌─────────────┐
//! │   Engine    │◄───┤    Engine    │◄───┤  Engine     │
//! │   Client    │    │              │    │  Operations │
//! └─────────────┘    └──────────────┘    └─────────────┘
//!        │                   │                   │
//!        ▼                   ▼                   ▼
//! ┌─────────────┐    ┌──────────────┐    ┌─────────────┐
//! │ Engine API  │    │ Engine State │    │ Rollup      │
//! │ (HTTP/JWT)  │    │   Updates    │    │ Config      │
//! └─────────────┘    └──────────────┘    └─────────────┘
//! ```
//!
//! ## Module Organization
//!
//! - **Operations** - Engine operations and state publication via [`Engine`]
//! - **Client** - HTTP client for Engine API communication via [`EngineClient`]
//! - **State** - Engine state management and synchronization via [`EngineState`]
//! - **Versions** - Engine API version selection via [`EngineForkchoiceVersion`],
//!   [`EngineNewPayloadVersion`], [`EngineGetPayloadVersion`]
//! - **Attributes** - Payload attribute validation via [`AttributesMatch`]
//! - **Kinds** - Engine client type identification via [`EngineKind`]
//! - **Metrics** - Optional Prometheus metrics collection via [`Metrics`]

#[macro_use]
extern crate tracing;

mod engine;
pub(crate) use engine::{
    BuildSealCoupling, BuildTask, ConsolidateTask, FinalizeTask, InsertTask, SealTask,
    SynchronizeTask,
};
pub use engine::{
    BuildTaskError, ConsolidateInput, ConsolidateTaskError, Engine, EngineBuildError,
    EngineResetError, EngineTaskError, EngineTaskErrorSeverity, FinalizeBlockId, FinalizeTaskError,
    InsertTaskError, SealTaskError, SynchronizeTaskError,
};

mod attributes;
pub use attributes::{AttributesMatch, AttributesMismatch};

mod block_sink;
pub use block_sink::{ImportedBlockSink, NoopBlockSink};

mod client;
pub use client::{EngineClient, EngineClientBuilder, EngineClientError};

mod request;
pub use request::{EngineRequestError, EngineRequestResult};

mod versions;
pub use versions::{EngineForkchoiceVersion, EngineGetPayloadVersion, EngineNewPayloadVersion};

mod state;
pub use state::{EngineState, EngineSyncState, EngineSyncStateUpdate};

mod kinds;
pub use kinds::EngineKind;

mod metrics;
pub use metrics::Metrics;

mod sync;
pub use sync::{L2ForkchoiceState, SyncStartError, find_starting_forkchoice};

#[cfg(any(test, feature = "test-utils"))]
/// Utilities that are useful when creating unit tests using structs within this library.
pub mod test_utils;
