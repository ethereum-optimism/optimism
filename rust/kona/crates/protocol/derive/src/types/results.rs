//! Result types for the `kona-derive` pipeline.

use crate::PipelineErrorKind;

/// A result type for the derivation pipeline stages.
pub type PipelineResult<T> = Result<T, PipelineErrorKind>;

/// A pipeline error.
#[derive(Debug, PartialEq, Eq)]
pub enum StepResult {
    /// Attributes were successfully prepared.
    PreparedAttributes,
    /// Origin was advanced.
    AdvancedOrigin,
    /// Origin advance failed.
    OriginAdvanceErr(PipelineErrorKind),
    /// Step failed.
    StepFailed(PipelineErrorKind),
}
