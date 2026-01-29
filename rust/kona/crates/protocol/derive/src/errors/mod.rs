//! Error types for the kona derivation pipeline.
//!
//! This module contains comprehensive error types for the derivation pipeline, organized
//! by severity and category. The error system provides detailed context for debugging
//! and enables sophisticated error handling and recovery strategies.

mod attributes;
pub use attributes::BuilderError;

mod stages;
pub use stages::BatchDecompressionError;

mod pipeline;
pub use pipeline::{PipelineEncodingError, PipelineError, PipelineErrorKind, ResetError};

mod sources;
pub use sources::{BlobDecodingError, BlobProviderError};
