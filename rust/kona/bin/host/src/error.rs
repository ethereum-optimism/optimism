//! Error types for the host binary.

use alloy_rlp::Error as RlpError;
use alloy_transport::TransportError;
use kona_preimage::errors::PreimageOracleError;
use std::array::TryFromSliceError;
use thiserror::Error;

/// Result type for host operations.
pub type Result<T> = std::result::Result<T, HostError>;

/// Error type for host operations.
#[derive(Debug, Error)]
pub enum HostError {
    /// A custom error message.
    #[error("{0}")]
    Custom(String),

    /// Block not found error.
    #[error("Block not found")]
    BlockNotFound,

    /// Invalid hint data length.
    #[error("Invalid hint data length")]
    InvalidHintDataLength,

    /// Precompile not accelerated.
    #[error("Precompile not accelerated")]
    PrecompileNotAccelerated,

    /// Failed precompile execution.
    #[error("Failed precompile execution: {0}")]
    PrecompileExecutionFailed(String),

    /// No rollup config found for chain ID.
    #[error("No rollup config found for chain ID: {0}")]
    NoRollupConfig(u64),

    /// Output root mismatch.
    #[error("Output root does not match L2 head")]
    OutputRootMismatch,

    /// Agreed pre-state hash mismatch.
    #[error("Agreed pre-state hash does not match")]
    AgreedPreStateHashMismatch,

    /// Expected blob count mismatch.
    #[error("Expected {expected} blob(s), got {actual}")]
    BlobCountMismatch {
        /// Expected blob count.
        expected: usize,
        /// Actual blob count.
        actual: usize,
    },

    /// Expected sidecar count mismatch.
    #[error("Expected {expected} sidecar(s), got {actual}")]
    SidecarCountMismatch {
        /// Expected sidecar count.
        expected: usize,
        /// Actual sidecar count.
        actual: usize,
    },

    /// No artifacts found for safe head.
    #[error("No artifacts found for the safe head")]
    NoArtifactsForSafeHead,

    /// Failed to fetch blob sidecars.
    #[error("Failed to fetch blob sidecars: {0}")]
    BlobSidecarFetchFailed(String),

    /// Failed to set key-value pair.
    #[error("Failed to set key-value pair: {0}")]
    KeyValueSetFailed(String),

    /// Failed to convert slice to B256.
    #[error("Failed to convert slice to B256: {0}")]
    B256ConversionFailed(String),

    /// Failed to fetch header RLP.
    #[error("Failed to fetch header RLP: {0}")]
    HeaderRlpFetchFailed(String),

    /// Error fetching code hash preimage.
    #[error("Error fetching code hash preimage: {0}")]
    CodeHashPreimageFetchFailed(String),

    /// Transport error.
    #[error("Transport error: {0}")]
    Transport(#[from] TransportError),

    /// RLP decoding error.
    #[error("RLP decoding error: {0}")]
    Rlp(#[from] RlpError),

    /// TryFromSlice error.
    #[error("TryFromSlice error: {0}")]
    TryFromSlice(#[from] TryFromSliceError),

    /// Serde JSON error.
    #[error("Serde JSON error: {0}")]
    SerdeJson(#[from] serde_json::Error),

    /// RocksDB error.
    #[error("RocksDB error: {0}")]
    RocksDb(String),

    /// Preimage oracle error.
    #[error("Preimage oracle error: {0}")]
    PreimageOracle(#[from] PreimageOracleError),

    /// Kona derive error.
    #[error("Kona derive error: {0}")]
    KonaDerive(String),

    /// Kona executor error.
    #[error("Kona executor error: {0}")]
    KonaExecutor(String),

    /// IO error.
    #[error("IO error: {0}")]
    Io(#[from] std::io::Error),
}

impl From<rocksdb::Error> for HostError {
    fn from(err: rocksdb::Error) -> Self {
        Self::RocksDb(err.to_string())
    }
}

impl From<kona_derive::PipelineError> for HostError {
    fn from(err: kona_derive::PipelineError) -> Self {
        Self::KonaDerive(err.to_string())
    }
}

impl From<kona_executor::ExecutorError> for HostError {
    fn from(err: kona_executor::ExecutorError) -> Self {
        Self::KonaExecutor(err.to_string())
    }
}
