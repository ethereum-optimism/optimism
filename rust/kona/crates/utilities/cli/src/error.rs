//! Error types for CLI utilities.

use std::io;
use thiserror::Error;

/// Error type for prometheus server initialization.
#[derive(Debug, Error)]
pub enum PrometheusError {
    /// Failed to bind to the specified address.
    #[error("failed to bind to address: {0}")]
    Bind(#[from] io::Error),
    /// Failed to set the global metrics recorder.
    #[error("failed to set global metrics recorder: {0}")]
    SetRecorder(#[from] metrics::SetRecorderError<metrics_exporter_prometheus::PrometheusRecorder>),
}

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
    #[error("Failed to initialize metrics: {0}")]
    MetricsInitialization(#[from] PrometheusError),
}

/// Type alias for CLI results.
pub type CliResult<T> = Result<T, CliError>;
