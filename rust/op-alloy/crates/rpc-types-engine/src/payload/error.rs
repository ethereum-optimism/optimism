//! Optimism payload errors.

use alloy_rpc_types_engine::PayloadError;

/// Extends [`PayloadError`] for Optimism.
#[derive(Debug, thiserror::Error)]
pub enum OpPayloadError {
    /// Non-empty list of L1 withdrawals (Shanghai).
    #[error("non-empty L1 withdrawals")]
    NonEmptyL1Withdrawals,
    /// Contains unsupported blob transaction type EIP-4844.
    #[error("contains blob transaction")]
    BlobTransaction,
    /// Non-empty list of execution layer requests.
    #[error("non-empty EL requests")]
    NonEmptyELRequests,
    /// Non-empty list of blob versioned hashes.
    #[error("non-empty blob versioned hashes")]
    NonEmptyBlobVersionedHashes,
    /// L1 [`PayloadError`] that can also occur on L2.
    #[error(transparent)]
    Eth(#[from] PayloadError),
}
