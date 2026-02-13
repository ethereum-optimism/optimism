//! Contains driver-related error types.

use kona_derive::PipelineErrorKind;
use kona_protocol::FromBlockError;
use thiserror::Error;

/// A [`Result`] type for the [`DriverError`].
pub type DriverResult<T, E> = Result<T, DriverError<E>>;

/// Driver error.
#[derive(Error, Debug)]
pub enum DriverError<E>
where
    E: core::error::Error,
{
    /// Pipeline error.
    #[error("Pipeline error: {0}")]
    Pipeline(#[from] PipelineErrorKind),
    /// An error returned by the executor.
    #[error("Executor error: {0}")]
    Executor(E),
    /// An error returned by the conversion from a block to an [`kona_protocol::L2BlockInfo`].
    #[error("From block error: {0}")]
    FromBlock(#[from] FromBlockError),
    /// Error decoding or encoding RLP.
    #[error("RLP error: {0}")]
    Rlp(alloy_rlp::Error),
}
