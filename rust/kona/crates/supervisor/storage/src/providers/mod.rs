//! Providers for supervisor state tracking.
//!
//! This module defines and implements storage providers used by the supervisor
//! for managing L2 execution state. It includes support for reading and writing:
//! - Logs and block metadata (via [`LogProvider`])
//! - Derivation pipeline state (via [`DerivationProvider`])
//! - Chain head tracking and progression
mod derivation_provider;
pub(crate) use derivation_provider::DerivationProvider;

mod log_provider;
pub(crate) use log_provider::LogProvider;

mod head_ref_provider;
pub(crate) use head_ref_provider::SafetyHeadRefProvider;
