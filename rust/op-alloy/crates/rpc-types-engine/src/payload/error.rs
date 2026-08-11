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
    /// A V3 or V4 payload is missing its parent beacon block root.
    #[error("missing parent beacon block root")]
    MissingParentBeaconBlockRoot,
    /// A V1 or V2 payload unexpectedly includes a parent beacon block root.
    #[error("unexpected parent beacon block root")]
    UnexpectedParentBeaconBlockRoot,
    /// A V4 payload is missing its execution request fields.
    #[error("missing EL requests")]
    MissingELRequests,
    /// A pre-V4 payload unexpectedly includes execution request fields.
    #[error("unexpected EL requests")]
    UnexpectedELRequests,
    /// L1 [`PayloadError`] that can also occur on L2.
    #[error(transparent)]
    Eth(#[from] PayloadError),
}
