#![doc = include_str!("../README.md")]
#![doc(
    html_logo_url = "https://raw.githubusercontent.com/ethereum-optimism/optimism/develop/rust/kona/assets/square.png",
    html_favicon_url = "https://raw.githubusercontent.com/ethereum-optimism/optimism/develop/rust/kona/assets/favicon.ico",
    issue_tracker_base_url = "https://github.com/ethereum-optimism/optimism/issues/"
)]
#![cfg_attr(docsrs, feature(doc_cfg))]

//! ## Architecture
//!
//! The engine operates as a task-driven system where operations are queued and executed atomically:
//!
//! ```text
//! ┌─────────────┐    ┌──────────────┐    ┌─────────────┐
//! │   Engine    │◄───┤  Task Queue  │◄───┤  Engine     │
//! │   Client    │    │   (Priority) │    │  Tasks      │
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
//! - **Task Queue** - Core engine task queue and execution logic via [`Engine`]
//! - **Client** - HTTP client for Engine API communication via [`EngineClient`]
//! - **State** - Engine state management and synchronization via [`EngineState`]
//! - **Versions** - Engine API version selection via [`EngineForkchoiceVersion`],
//!   [`EngineNewPayloadVersion`], [`EngineGetPayloadVersion`]
//! - **Attributes** - Payload attribute validation via [`AttributesMatch`]
//! - **Kinds** - Engine client type identification via [`EngineKind`]
//! - **Query** - Engine query interface via [`EngineQueries`]
//! - **Metrics** - Optional Prometheus metrics collection via [`Metrics`]

#[macro_use]
extern crate tracing;

mod task_queue;
pub use task_queue::{
    BuildTask, BuildTaskError, ConsolidateInput, ConsolidateTask, ConsolidateTaskError, Engine,
    EngineBuildError, EngineResetError, EngineTask, EngineTaskError, EngineTaskErrorSeverity,
    EngineTaskErrors, EngineTaskExt, FinalizeTask, FinalizeTaskError, InsertTask, InsertTaskError,
    SealTask, SealTaskError, SynchronizeTask, SynchronizeTaskError,
};

mod attributes;
pub use attributes::{AttributesMatch, AttributesMismatch};

mod client;
pub use client::{
    EngineClient, EngineClientBuilder, EngineClientBuilderError, EngineClientError,
    HyperAuthClient, OpEngineClient,
};

mod rollup_boost;
pub use rollup_boost::{
    FlashblocksClientArgs, FlashblocksWebsocketConfig, RollupBoostServer, RollupBoostServerArgs,
    RollupBoostServerError,
};

mod versions;
pub use versions::{EngineForkchoiceVersion, EngineGetPayloadVersion, EngineNewPayloadVersion};

mod state;
pub use state::{EngineState, EngineSyncState, EngineSyncStateUpdate};

mod kinds;
pub use kinds::EngineKind;

mod query;
pub use query::{EngineQueries, EngineQueriesError, EngineQuerySender};

mod metrics;
pub use metrics::Metrics;

mod sync;
pub use sync::{L2ForkchoiceState, SyncStartError, find_starting_forkchoice};

#[cfg(any(test, feature = "test-utils"))]
/// Utilities that are useful when creating unit tests using structs within this library.
pub mod test_utils;
