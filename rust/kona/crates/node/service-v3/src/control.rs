//! Shared task-control errors.

use thiserror::Error;

/// An error returned by a cloneable service control handle.
#[derive(Debug, Error, Clone, PartialEq, Eq)]
pub enum ControlError {
    /// The service is no longer receiving control requests.
    #[error("service is unavailable")]
    Unavailable,
    /// The service stopped before acknowledging a request.
    #[error("service dropped its response")]
    ResponseDropped,
}
