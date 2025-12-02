//! Error types for sources.

use crate::{PipelineError, PipelineErrorKind};
use alloc::string::{String, ToString};
use thiserror::Error;

/// Blob Decoding Error
#[derive(Error, Debug, PartialEq, Eq)]
pub enum BlobDecodingError {
    /// Invalid field element
    #[error("Invalid field element")]
    InvalidFieldElement,
    /// Invalid encoding version
    #[error("Invalid encoding version")]
    InvalidEncodingVersion,
    /// Invalid length
    #[error("Invalid length")]
    InvalidLength,
    /// Missing Data
    #[error("Missing data")]
    MissingData,
}

/// An error returned by the [`BlobProviderError`].
#[derive(Error, Debug, PartialEq, Eq)]
pub enum BlobProviderError {
    /// The number of specified blob hashes did not match the number of returned sidecars.
    #[error("Blob sidecar length mismatch: expected {0}, got {1}")]
    SidecarLengthMismatch(usize, usize),
    /// Slot derivation error.
    #[error("Failed to derive slot")]
    SlotDerivation,
    /// Blob decoding error.
    #[error("Blob decoding error: {0}")]
    BlobDecoding(#[from] BlobDecodingError),
    /// Error pertaining to the backend transport.
    #[error("{0}")]
    Backend(String),
}

impl From<BlobProviderError> for PipelineErrorKind {
    fn from(val: BlobProviderError) -> Self {
        match val {
            BlobProviderError::SidecarLengthMismatch(_, _) => {
                PipelineError::Provider(val.to_string()).crit()
            }
            BlobProviderError::SlotDerivation => PipelineError::Provider(val.to_string()).crit(),
            BlobProviderError::BlobDecoding(_) => PipelineError::Provider(val.to_string()).crit(),
            BlobProviderError::Backend(_) => PipelineError::Provider(val.to_string()).temp(),
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use core::error::Error;

    #[test]
    fn test_blob_decoding_error_source() {
        let err: BlobProviderError = BlobDecodingError::InvalidFieldElement.into();
        assert!(err.source().is_some());
    }

    #[test]
    fn test_from_blob_provider_error() {
        let err: PipelineErrorKind = BlobProviderError::SlotDerivation.into();
        assert!(matches!(err, PipelineErrorKind::Critical(_)));

        let err: PipelineErrorKind = BlobProviderError::SidecarLengthMismatch(1, 2).into();
        assert!(matches!(err, PipelineErrorKind::Critical(_)));

        let err: PipelineErrorKind =
            BlobProviderError::BlobDecoding(BlobDecodingError::InvalidFieldElement).into();
        assert!(matches!(err, PipelineErrorKind::Critical(_)));
    }
}
