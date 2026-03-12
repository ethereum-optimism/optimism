//! Result types for the `kona-derive` pipeline.

use crate::PipelineErrorKind;

/// A result type for the derivation pipeline stages.
pub type PipelineResult<T> = Result<T, PipelineErrorKind>;
