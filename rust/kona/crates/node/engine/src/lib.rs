#![doc = include_str!("../README.md")]
#![doc(
    html_logo_url = "https://raw.githubusercontent.com/ethereum-optimism/optimism/develop/rust/kona/assets/square.png",
    html_favicon_url = "https://raw.githubusercontent.com/ethereum-optimism/optimism/develop/rust/kona/assets/favicon.ico",
    issue_tracker_base_url = "https://github.com/ethereum-optimism/optimism/issues/"
)]
#![cfg_attr(docsrs, feature(doc_cfg))]

//! ## Architecture
//!
//! This crate provides ordinary Engine API operations, their applied state, and the canonical
//! reset walk. The node service owns operation serialization, reset epochs, retries and replies.
//! Operations report their actual failure stage and retain partial accepted state for publication.
//!
//! - **Operations** — [`start_build`], [`get_payload`], [`insert_payload`], [`consolidate`],
//!   [`follow_safe`], [`finalize`] and [`synchronize`].
//! - **Client** — injected L1/L2 providers through [`EngineClient`].
//! - **State** — one snapshot of heads, readiness and reset epoch in [`EngineState`].
//! - **Validation** — payload matching through [`AttributesMatch`] and Engine API version
//!   selection.
//! - **Recovery** — the canonical [`find_starting_forkchoice`] walk, called by the reset owner.

#[macro_use]
extern crate tracing;

mod operations;
pub use operations::{
    BuildError, ConsolidateError, Consolidation, FinalizationSource, FinalizeError,
    ForkchoiceError, ForkchoiceStatus, ImportMode, InsertError, ResetError, consolidate, finalize,
    follow_safe, get_payload, insert_payload, start_build, synchronize,
};

mod attributes;
pub use attributes::{AttributesMatch, AttributesMismatch};

mod block_sink;
pub use block_sink::{ImportedBlockSink, NoopBlockSink};

mod client;
pub use client::{EngineClient, OpEngineClient};

mod providers;
pub use providers::{HyperAuthClient, l1_provider, l2_provider, rpc_client};

mod versions;
pub use versions::{EngineForkchoiceVersion, EngineGetPayloadVersion};

mod state;
pub use state::{EnginePhase, EngineState, EngineSyncState, EngineSyncStateUpdate};

mod metrics;
pub use metrics::Metrics;

mod sync;
pub use sync::{L2ForkchoiceState, SyncStartError, SyncStartPolicy, find_starting_forkchoice};

#[cfg(any(test, feature = "test-utils"))]
/// Utilities that are useful when creating unit tests using structs within this library.
pub mod test_utils;
