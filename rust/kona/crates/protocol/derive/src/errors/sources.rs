//! Error types for sources.

use crate::{PipelineError, PipelineErrorKind, ResetError};
use alloc::string::{String, ToString};
use thiserror::Error;

/// Blob Decoding Error
#[derive(Error, Clone, Debug, PartialEq, Eq)]
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
#[derive(Error, Clone, Debug, PartialEq, Eq)]
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
    /// The blob provider returned fewer blobs than requested (under-fill).
    #[error(
        "Not enough blobs: expected blob at index {expected} but provider returned only {actual} blobs"
    )]
    NotEnoughBlobs {
        /// The blob index that was expected.
        expected: usize,
        /// The actual number of blobs returned by the provider.
        actual: usize,
    },
    /// The beacon node returned a 404 for the requested slot, indicating the slot was missed or
    /// orphaned. Blobs for missed/orphaned slots will never become available, so the pipeline
    /// must reset to move past the L1 block that referenced them.
    #[error("Blob not found at slot {slot}: {reason}")]
    BlobNotFound {
        /// The beacon slot that returned 404.
        slot: u64,
        /// The underlying error message from the beacon client.
        reason: String,
    },
    /// Error pertaining to the backend transport.
    #[error("{0}")]
    Backend(String),
}

impl From<BlobProviderError> for PipelineErrorKind {
    fn from(val: BlobProviderError) -> Self {
        match val {
            BlobProviderError::SidecarLengthMismatch(_, _) |
            BlobProviderError::SlotDerivation |
            BlobProviderError::BlobDecoding(_) => PipelineError::Provider(val.to_string()).crit(),
            BlobProviderError::NotEnoughBlobs { .. } => ResetError::BlobsUnderFill(val).reset(),
            BlobProviderError::BlobNotFound { slot, .. } => {
                ResetError::BlobsUnavailable(slot).reset()
            }
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

        let err: PipelineErrorKind = BlobProviderError::Backend("transport error".into()).into();
        assert!(matches!(err, PipelineErrorKind::Temporary(_)));

        // A 404 from the beacon node (missed/orphaned slot) must trigger a pipeline reset,
        // not a temporary retry. Without this, the safe head stalls indefinitely.
        let err: PipelineErrorKind =
            BlobProviderError::BlobNotFound { slot: 13779552, reason: "slot not found".into() }
                .into();
        assert!(
            matches!(err, PipelineErrorKind::Reset(_)),
            "BlobNotFound must map to Reset so the pipeline moves past the missed slot"
        );

        let err: PipelineErrorKind =
            BlobProviderError::NotEnoughBlobs { expected: 2, actual: 1 }.into();
        assert!(matches!(err, PipelineErrorKind::Reset(_)));
    }
}
