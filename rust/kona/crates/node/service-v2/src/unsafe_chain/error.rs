//! Unsafe-chain service errors.

use crate::engine::EngineError;
use thiserror::Error;

/// A terminal unsafe-chain service error.
#[derive(Debug, Error, Clone, PartialEq, Eq)]
pub enum UnsafeChainError {
    /// A complete unsafe payload could not be applied by the engine.
    #[error(transparent)]
    Engine(#[from] EngineError),
    /// Every network payload sender was dropped before node shutdown.
    #[error("all unsafe payload senders were dropped")]
    PayloadChannelClosed,
}

/// An error submitting a network payload to the unsafe-chain service.
#[derive(Debug, Error, Clone, Copy, PartialEq, Eq)]
pub enum UnsafePayloadIngressError {
    /// The unsafe-chain service is no longer accepting network payloads.
    #[error("unsafe-chain service is unavailable")]
    Unavailable,
}
