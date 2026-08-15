//! Engine domain: execution ownership, unsafe acquisition, and forkchoice reconciliation.
//!
//! [`crate::engine::EngineService`] exclusively owns raw Engine API calls and all unsafe-chain
//! behavior. The only capability exposed to Derivation is [`crate::engine::EngineHandle`], whose
//! API consists of safe and finalized
//! updates.

mod admin;
pub use admin::EngineAdminAdapter;

mod api;
#[cfg(test)]
pub(crate) use api::EngineRequest;
pub use api::{EngineHandle, EngineRpcAdapter, SafeChainUpdate};

mod config;
pub use config::EngineConfig;

mod error;
pub use error::{EngineError, EngineResult, EngineServiceError};

mod network;
mod runtime;
pub(crate) use runtime::EngineRuntimeConfig;

mod service;
pub use service::{DEFAULT_ENGINE_REQUEST_CAPACITY, EngineService};
pub(crate) use service::{ENGINE_RETRY_DELAY, EngineController, EngineStarted};

mod signer;
mod unsafe_chain;
pub use unsafe_chain::SequencerConfig;
