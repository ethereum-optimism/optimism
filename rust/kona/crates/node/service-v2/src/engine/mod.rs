//! Semantic execution-engine access and forkchoice reconciliation.
//!
//! This module is the exclusive owner of raw Engine API mutations. Unsafe and safe chain
//! workflows interact with it through narrow semantic operations rather than Engine API calls.

mod api;
pub use api::{EngineClient, SafeChainUpdate};

mod driver;
pub use driver::EngineDriver;

mod error;
pub use error::{EngineError, EngineResult, EngineServiceError};

mod service;
pub use service::{DEFAULT_ENGINE_REQUEST_CAPACITY, EngineService};

#[cfg(test)]
mod tests;
