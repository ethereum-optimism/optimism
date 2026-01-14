//! Error types for the `kona-interop` crate.

use crate::InteropProvider;
use alloy_primitives::{Address, B256};
use core::fmt::Debug;
use kona_registry::HashMap;
use thiserror::Error;

/// An error type for the [MessageGraph] struct.
///
/// [MessageGraph]: crate::MessageGraph
#[derive(Debug, Clone, PartialEq, Eq, Error)]
pub enum MessageGraphError<E: Debug> {
    /// Dependency set is impossibly empty
    #[error("Dependency set is impossibly empty")]
    EmptyDependencySet,
    /// Missing a [RollupConfig] for a chain ID
    ///
    /// [RollupConfig]: kona_genesis::RollupConfig
    #[error("Missing a RollupConfig for chain ID {0}")]
    MissingRollupConfig(u64),
    /// Interop provider error
    #[error("Interop provider: {0}")]
    InteropProviderError(#[from] E),
    /// Remote message not found
    #[error("Remote message not found on chain ID {chain_id} with message hash {message_hash}")]
    RemoteMessageNotFound {
        /// The remote chain ID
        chain_id: u64,
        /// The message hash
        message_hash: B256,
    },
    /// Invalid message origin
    #[error("Invalid message origin. Expected {expected}, got {actual}")]
    InvalidMessageOrigin {
        /// The expected message origin
        expected: Address,
        /// The actual message origin
        actual: Address,
    },
    /// Invalid message payload hash
    #[error("Invalid message hash. Expected {expected}, got {actual}")]
    InvalidMessageHash {
        /// The expected message hash
        expected: B256,
        /// The actual message hash
        actual: B256,
    },
    /// Invalid message timestamp
    #[error("Invalid message timestamp. Expected {expected}, got {actual}")]
    InvalidMessageTimestamp {
        /// The expected timestamp
        expected: u64,
        /// The actual timestamp
        actual: u64,
    },
    /// Interop has not been activated for at least one block on the initiating message's chain.
    #[error(
        "Interop has not been active for at least one block on initiating message's chain. Activation time: {activation_time}, initiating message time: {initiating_message_time}"
    )]
    InitiatedTooEarly {
        /// The timestamp of the interop activation
        activation_time: u64,
        /// The timestamp of the initiating message
        initiating_message_time: u64,
    },
    /// Message is in the future
    #[error("Message is in the future. Expected timestamp to be <= {max}, got {actual}")]
    MessageInFuture {
        /// The expected max timestamp
        max: u64,
        /// The actual timestamp
        actual: u64,
    },
    /// Message has exceeded the expiry window.
    #[error(
        "Message has exceeded the expiry window. Initiating Timestamp: {initiating_timestamp}, Executing Timestamp: {executing_timestamp}"
    )]
    MessageExpired {
        /// The timestamp of the initiating message
        initiating_timestamp: u64,
        /// The timestamp of the executing message
        executing_timestamp: u64,
    },
    /// Invalid messages were found
    #[error("Invalid messages found on chains: {0:?}")]
    InvalidMessages(HashMap<u64, MessageGraphError<E>>),
}

/// A [Result] alias for the [MessageGraphError] type.
#[allow(type_alias_bounds)]
pub type MessageGraphResult<T, P: InteropProvider> =
    core::result::Result<T, MessageGraphError<P::Error>>;

/// An error type for the [SuperRoot] struct's serialization and deserialization.
///
/// [SuperRoot]: crate::SuperRoot
#[derive(Debug, Clone, Error)]
pub enum SuperRootError {
    /// Invalid super root version byte
    #[error("Invalid super root version byte")]
    InvalidVersionByte,
    /// Unexpected encoded super root length
    #[error("Unexpected encoded super root length")]
    UnexpectedLength,
    /// Slice conversion error
    #[error("Slice conversion error: {0}")]
    SliceConversionError(#[from] core::array::TryFromSliceError),
}

/// A [Result] alias for the [SuperRootError] type.
pub type SuperRootResult<T> = core::result::Result<T, SuperRootError>;

/// Errors that can occur during interop validation.
#[derive(Debug, Error, PartialEq, Eq)]
pub enum InteropValidationError {
    /// Interop is not enabled on one or both chains at the required timestamp.
    #[error("interop not enabled")]
    InteropNotEnabled,

    /// Executing timestamp is earlier than the initiating timestamp.
    #[error(
        "executing timestamp is earlier than initiating timestamp, executing: {executing}, initiating: {initiating}"
    )]
    InvalidTimestampInvariant {
        /// Executing timestamp of the message
        executing: u64,
        /// Initiating timestamp of the message
        initiating: u64,
    },

    /// Timestamp is outside the allowed interop expiry window.
    #[error("timestamp outside allowed interop window, timestamp: {0}")]
    InvalidInteropTimestamp(u64),
}
