use kona_supervisor_storage::StorageError;
use thiserror::Error;

use crate::syncnode::ManagedNodeError;

/// Error type for reorg handling
#[derive(Debug, Error)]
pub enum ReorgHandlerError {
    /// Indicates managed node not found for the chain.
    #[error("managed node not found for chain: {0}")]
    ManagedNodeMissing(u64),

    /// Indicates an error occurred while interacting with the managed node.
    #[error(transparent)]
    ManagedNodeError(#[from] ManagedNodeError),

    /// Indicates an error occurred while interacting with the database.
    #[error(transparent)]
    StorageError(#[from] StorageError),

    /// Indicates an error occurred while interacting with the l1 RPC client.
    #[error("failed to interact with l1 RPC client: {0}")]
    RPCError(String),

    /// Indicates an error occurred while finding rewind target for reorg.
    /// This can happen if the rewind target block is pre-interop.
    #[error("rewind target is pre-interop")]
    RewindTargetPreInterop,
}
