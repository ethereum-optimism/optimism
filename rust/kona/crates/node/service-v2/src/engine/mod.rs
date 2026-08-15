//! Semantic execution-engine access and forkchoice reconciliation.
//!
//! [`crate::engine::EngineService`] is the exclusive owner of raw Engine API calls and
//! authoritative head state. Unsafe and safe chain services interact with it only through
//! [`crate::engine::EngineClient`].

mod api;
#[cfg(test)]
pub(crate) use api::EngineRequest;
pub use api::{BuiltUnsafePayload, EngineClient, SafeChainUpdate};

mod config;
pub use config::EngineConfig;

mod error;
pub use error::{EngineError, EngineResult, EngineServiceError};

mod service;
pub(crate) use service::ENGINE_RETRY_DELAY;
pub use service::{DEFAULT_ENGINE_REQUEST_CAPACITY, EngineService};
