//! Error types for CLI utilities.

use thiserror::Error;

/// Errors that can occur in CLI operations.
#[derive(Error, Debug)]
#[non_exhaustive]
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

    /// Error installing the metrics recorder or its exporter thread.
    #[error("Failed to install the metrics recorder: {0}")]
    Metrics(String),

    /// Error spawning the thread that drives the metrics exporter.
    #[error("Failed to start the metrics exporter: {0}")]
    MetricsExporter(#[from] std::io::Error),
}

/// Type alias for CLI results.
pub type CliResult<T> = Result<T, CliError>;
