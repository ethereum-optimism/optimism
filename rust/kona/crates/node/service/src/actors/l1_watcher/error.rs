use std::sync::mpsc::SendError;

use crate::DerivationClientError;
use thiserror::Error;

/// The error type for the `L1WatcherActor`.
#[derive(Error, Debug)]
pub enum L1WatcherActorError<T> {
    /// Error sending the head update event.
    #[error("Error sending the head update event: {0}")]
    SendError(#[from] SendError<T>),
    /// Stream ended unexpectedly.
    #[error("Stream ended unexpectedly")]
    StreamEnded,
    /// Derivation client error, naming the chain whose derivation client failed.
    #[error("derivation client error for chain {chain_id}: {source}")]
    DerivationClientError {
        /// The id of the L2 chain served by the failing derivation client.
        chain_id: u64,
        /// The underlying derivation client error.
        #[source]
        source: DerivationClientError,
    },
}
