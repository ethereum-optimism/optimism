//! The severity of an error from an engine operation.

use derive_more::Display;

/// The severity of an error from an engine operation.
///
/// This determines how the engine actor handles the error.
#[derive(Debug, PartialEq, Eq, Display, Clone, Copy)]
pub enum EngineTaskErrorSeverity {
    /// The error is temporary, and retrying the operation may succeed.
    #[display("temporary")]
    Temporary,
    /// The error is critical and is propagated to the engine actor.
    #[display("critical")]
    Critical,
    /// The error indicates that the engine should be reset.
    #[display("reset")]
    Reset,
    /// The error indicates that the engine should be flushed.
    #[display("flush")]
    Flush,
}

/// The interface for an error from an engine operation.
///
/// An engine operation error should have an associated severity level to specify how to handle the
/// error.
pub trait EngineTaskError {
    /// The severity of the error.
    fn severity(&self) -> EngineTaskErrorSeverity;
}
