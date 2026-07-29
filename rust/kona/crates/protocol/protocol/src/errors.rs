//! Error types for protocol conversions.

use crate::DecodeError;
use alloy_primitives::B256;
use op_alloy_consensus::EIP1559ParamError;

/// An error encountered during OP [`Block`](alloy_consensus::Block) conversion.
#[derive(Debug, Eq, PartialEq, thiserror::Error)]
pub enum OpBlockConversionError {
    /// Invalid genesis hash.
    #[error("Invalid genesis hash. Expected {0}, got {1}")]
    InvalidGenesisHash(B256, B256),
    /// Invalid transaction type.
    #[error("First payload transaction has unexpected type: {0}")]
    InvalidTxType(u8),
    /// L1 Info error
    #[error("Failed to decode L1 info: {0}")]
    L1InfoError(#[from] DecodeError),
    /// Missing system config in genesis block.
    #[error("Missing system config in genesis block")]
    MissingSystemConfigGenesis,
    /// Empty transactions.
    #[error("Empty transactions in payload. Block hash: {0}")]
    EmptyTransactions(B256),
    /// EIP-1559 parameter decoding error.
    #[error("Failed to decode EIP-1559 parameters from header's `extraData` field.")]
    Eip1559DecodeError(#[from] EIP1559ParamError),
    /// Activation-block gas limit is below the fork's one-time upgrade gas.
    #[error("Activation block gas limit {gas_limit} is below upgrade gas {upgrade_gas}")]
    GasLimitBelowUpgradeGas {
        /// The block header's gas limit.
        gas_limit: u64,
        /// The fork's one-time upgrade gas.
        upgrade_gas: u64,
    },
}
