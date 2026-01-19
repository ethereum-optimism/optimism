//! Error types for CLI utilities.

use thiserror::Error;

/// Errors that can occur in CLI operations.
#[derive(Error, Debug)]
pub enum CliError {
    /// Error when no chain config is found for the given chain ID.
    #[error("No chain config found for chain ID: {0}")]
    ChainConfigNotFound(u64),

    /// Error when no roles are found for the given chain ID.
    #[error("No roles found for chain ID: {0}")]
    RolesNotFound(u64),

    /// Error when no unsafe block signer is found for the given chain ID.
    #[error("No unsafe block signer found for chain ID: {0}")]
    UnsafeBlockSignerNotFound(u64),

    /// Error initializing metrics.
    #[error("Failed to initialize metrics")]
    MetricsInitialization(#[from] metrics_exporter_prometheus::BuildError),
}

/// Type alias for CLI results.
pub type CliResult<T> = Result<T, CliError>;
