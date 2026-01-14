use std::sync::mpsc::SendError;

use alloy_eips::BlockId;
use alloy_transport::TransportError;
use thiserror::Error;

/// The error type for the `L1WatcherActor`.
#[derive(Error, Debug)]
pub enum L1WatcherActorError<T> {
    /// Error sending the head update event.
    #[error("Error sending the head update event: {0}")]
    SendError(#[from] SendError<T>),
    /// Error in the transport layer.
    #[error("Transport error: {0}")]
    Transport(#[from] TransportError),
    /// The L1 block was not found.
    #[error("L1 block not found: {0}")]
    L1BlockNotFound(BlockId),
    /// Stream ended unexpectedly.
    #[error("Stream ended unexpectedly")]
    StreamEnded,
}
