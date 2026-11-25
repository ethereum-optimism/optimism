//! Contains the error type of [`OpStackRequest::Open`](crate::OpStackRequest::Open).

use crate::EngineBuildError;
use alloy_primitives::B256;
use alloy_transport::{RpcError, TransportErrorKind};
use kona_protocol::FromBlockError;
use thiserror::Error;

/// An error opening a block on a parent with [`OpStackRequest::Open`](crate::OpStackRequest::Open).
#[derive(Debug, Error)]
pub enum OpenBlockError {
    /// The execution layer has no block with the parent's hash.
    #[error("Parent block {0} not found")]
    ParentNotFound(B256),
    /// Reading the parent block from the execution layer failed.
    #[error("Failed to read the parent block: {0}")]
    ParentLookup(#[from] RpcError<TransportErrorKind>),
    /// The parent block is not a block of this chain.
    #[error(transparent)]
    InvalidParent(#[from] FromBlockError),
    /// The execution layer did not start the build.
    #[error(transparent)]
    Build(#[from] EngineBuildError),
}
