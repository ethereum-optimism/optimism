//! Host library component of the kona-sp1 proof.

pub mod logger;
pub mod metrics;
pub mod network;
pub mod witness_generation;
pub use logger::setup_logger;

/// Builds an environment-variable name from a service prefix and suffix.
pub fn prefixed_env_var(prefix: &str, suffix: &str) -> String {
    format!("{prefix}_{suffix}")
}
