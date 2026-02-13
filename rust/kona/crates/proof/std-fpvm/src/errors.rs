//! Errors for the `kona-std-fpvm` crate.

use thiserror::Error;

/// An error that can occur when reading from or writing to a file descriptor.
#[derive(Error, Debug, PartialEq, Eq)]
#[error("IO error (errno: {_0})")]
pub struct IOError(pub i32);

/// A [Result] type for the [`IOError`].
pub type IOResult<T> = Result<T, IOError>;
