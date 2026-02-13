//! Contains error types for system config updates.

use alloy_primitives::B256;
use derive_more::From;

/// An error for processing the [`crate::SystemConfig`] update log.
#[derive(Debug, From, thiserror::Error, Clone, Copy, PartialEq, Eq)]
#[cfg_attr(feature = "serde", derive(serde::Serialize, serde::Deserialize))]
pub enum SystemConfigUpdateError {
    /// An error occurred while processing the update log.
    #[error("Log processing error: {0}")]
    LogProcessing(LogProcessingError),
    /// A batcher update error.
    #[error("Batcher update error: {0}")]
    Batcher(BatcherUpdateError),
    /// A gas config update error.
    #[error("Gas config update error: {0}")]
    GasConfig(GasConfigUpdateError),
    /// A gas limit update error.
    #[error("Gas limit update error: {0}")]
    GasLimit(GasLimitUpdateError),
    /// An EIP-1559 parameter update error.
    #[error("EIP-1559 parameter update error: {0}")]
    Eip1559(EIP1559UpdateError),
    /// An operator fee parameter update error.
    #[error("Operator fee parameter update error: {0}")]
    OperatorFee(OperatorFeeUpdateError),
    /// An unsafe block signer update error.
    #[error("Unsafe block signer update error: {0}")]
    UnsafeBlockSigner(UnsafeBlockSignerUpdateError),
    /// A min base fee parameter update error.
    #[error("Min base fee parameter update error: {0}")]
    MinBaseFee(MinBaseFeeUpdateError),
    /// A da footprint gas scalar update error.
    #[error("DA footprint gas scalar update error: {0}")]
    DaFootprintGasScalar(DaFootprintGasScalarUpdateError),
}

/// An error occurred while processing the update log.
#[derive(Debug, From, thiserror::Error, Clone, Copy, PartialEq, Eq)]
#[cfg_attr(feature = "serde", derive(serde::Serialize, serde::Deserialize))]
pub enum LogProcessingError {
    /// Received an incorrect number of log topics.
    #[error("Invalid config update log: invalid topic length: {0}")]
    InvalidTopicLen(usize),
    /// The log topic is invalid.
    #[error("Invalid config update log: invalid topic")]
    InvalidTopic,
    /// The config update log version is unsupported.
    #[error("Invalid config update log: unsupported version: {0}")]
    UnsupportedVersion(B256),
    /// Failed to decode the update type from the config update log.
    #[error("Failed to decode config update log: update type")]
    UpdateTypeDecodingError,
    /// An invalid system config update type.
    #[error("Invalid system config update type: {0}")]
    InvalidSystemConfigUpdateType(u64),
}

/// An error for updating the batcher address on the [`crate::SystemConfig`].
#[derive(Debug, thiserror::Error, Clone, Copy, PartialEq, Eq)]
#[cfg_attr(feature = "serde", derive(serde::Serialize, serde::Deserialize))]
pub enum BatcherUpdateError {
    /// Invalid data length.
    #[error("Invalid config update log: invalid data length: {0}")]
    InvalidDataLen(usize),
    /// Failed to decode the data pointer argument from the batcher update log.
    #[error("Failed to decode batcher update log: data pointer")]
    PointerDecodingError,
    /// The data pointer is invalid.
    #[error("Invalid config update log: invalid data pointer: {0}")]
    InvalidDataPointer(u64),
    /// Failed to decode the data length argument from the batcher update log.
    #[error("Failed to decode batcher update log: data length")]
    LengthDecodingError,
    /// The data length is invalid.
    #[error("Invalid config update log: invalid data length: {0}")]
    InvalidDataLength(u64),
    /// Failed to decode the batcher address argument from the batcher update log.
    #[error("Failed to decode batcher update log: batcher address")]
    BatcherAddressDecodingError,
}

/// An error for updating the unsafe block signer address on the [`crate::SystemConfig`].
#[derive(Debug, thiserror::Error, Clone, Copy, PartialEq, Eq)]
#[cfg_attr(feature = "serde", derive(serde::Serialize, serde::Deserialize))]
pub enum UnsafeBlockSignerUpdateError {
    /// Invalid data length.
    #[error("Invalid config update log: invalid data length: {0}")]
    InvalidDataLen(usize),
    /// Failed to decode the data pointer argument from the update log.
    #[error("Failed to decode unsafe block signer update log: data pointer")]
    PointerDecodingError,
    /// The data pointer is invalid.
    #[error("Invalid config update log: invalid data pointer: {0}")]
    InvalidDataPointer(u64),
    /// Failed to decode the data length argument from the update log.
    #[error("Failed to decode unsafe block signer update log: data length")]
    LengthDecodingError,
    /// The data length is invalid.
    #[error("Invalid config update log: invalid data length: {0}")]
    InvalidDataLength(u64),
    /// Failed to decode the unsafe block signer address argument from the update log.
    #[error("Failed to decode unsafe block signer update log: unsafe block signer address")]
    UnsafeBlockSignerAddressDecodingError,
}

/// An error for updating the gas config on the [`crate::SystemConfig`].
#[derive(Debug, thiserror::Error, Clone, Copy, PartialEq, Eq)]
#[cfg_attr(feature = "serde", derive(serde::Serialize, serde::Deserialize))]
pub enum GasConfigUpdateError {
    /// Invalid data length.
    #[error("Invalid config update log: invalid data length: {0}")]
    InvalidDataLen(usize),
    /// Failed to decode the data pointer argument from the gas config update log.
    #[error("Failed to decode gas config update log: data pointer")]
    PointerDecodingError,
    /// The data pointer is invalid.
    #[error("Invalid config update log: invalid data pointer: {0}")]
    InvalidDataPointer(u64),
    /// Failed to decode the data length argument from the gas config update log.
    #[error("Failed to decode gas config update log: data length")]
    LengthDecodingError,
    /// The data length is invalid.
    #[error("Invalid config update log: invalid data length: {0}")]
    InvalidDataLength(u64),
    /// Failed to decode the overhead argument from the gas config update log.
    #[error("Failed to decode gas config update log: overhead")]
    OverheadDecodingError,
    /// Failed to decode the scalar argument from the gas config update log.
    #[error("Failed to decode gas config update log: scalar")]
    ScalarDecodingError,
}

/// An error for updating the min base fee on the [`crate::SystemConfig`].
#[derive(Debug, thiserror::Error, Clone, Copy, PartialEq, Eq)]
#[cfg_attr(feature = "serde", derive(serde::Serialize, serde::Deserialize))]
pub enum MinBaseFeeUpdateError {
    /// Invalid data length.
    #[error("Invalid config update log: invalid data length: {0}")]
    InvalidDataLen(usize),
    /// Failed to decode the data pointer argument from the min base fee update log.
    #[error("Failed to decode gas limit update log: data pointer")]
    PointerDecodingError,
    /// The data pointer is invalid.
    #[error("Invalid config update log: invalid data pointer: {0}")]
    InvalidDataPointer(u64),
    /// Failed to decode the data length argument from the min base fee update log.
    #[error("Failed to decode gas limit update log: data length")]
    LengthDecodingError,
    /// The data length is invalid.
    #[error("Invalid config update log: invalid data length: {0}")]
    InvalidDataLength(u64),
    /// Failed to decode the min base fee argument from the min base fee update log.
    #[error("Failed to decode min base fee update log: min base fee")]
    MinBaseFeeDecodingError,
}

/// An error for updating the da footprint gas scalar on the [`crate::SystemConfig`].
#[derive(Debug, thiserror::Error, Clone, Copy, PartialEq, Eq)]
#[cfg_attr(feature = "serde", derive(serde::Serialize, serde::Deserialize))]
pub enum DaFootprintGasScalarUpdateError {
    /// Invalid data length.
    #[error("Invalid config update log: invalid data length: {0}")]
    InvalidDataLen(usize),
    /// Failed to decode the data pointer argument from the min base fee update log.
    #[error("Failed to decode gas limit update log: data pointer")]
    PointerDecodingError,
    /// The data pointer is invalid.
    #[error("Invalid config update log: invalid data pointer: {0}")]
    InvalidDataPointer(u64),
    /// Failed to decode the data length argument from the min base fee update log.
    #[error("Failed to decode gas limit update log: data length")]
    LengthDecodingError,
    /// The data length is invalid.
    #[error("Invalid config update log: invalid data length: {0}")]
    InvalidDataLength(u64),
    /// Failed to decode the da footprint gas scalar argument from the da footprint gas scalar
    /// update log.
    #[error("Failed to decode da footprint gas scalar update log: da footprint gas scalar")]
    DaFootprintGasScalarDecodingError,
}

/// An error for updating the gas limit on the [`crate::SystemConfig`].
#[derive(Debug, thiserror::Error, Clone, Copy, PartialEq, Eq)]
#[cfg_attr(feature = "serde", derive(serde::Serialize, serde::Deserialize))]
pub enum GasLimitUpdateError {
    /// Invalid data length.
    #[error("Invalid config update log: invalid data length: {0}")]
    InvalidDataLen(usize),
    /// Failed to decode the data pointer argument from the gas limit update log.
    #[error("Failed to decode gas limit update log: data pointer")]
    PointerDecodingError,
    /// The data pointer is invalid.
    #[error("Invalid config update log: invalid data pointer: {0}")]
    InvalidDataPointer(u64),
    /// Failed to decode the data length argument from the gas limit update log.
    #[error("Failed to decode gas limit update log: data length")]
    LengthDecodingError,
    /// The data length is invalid.
    #[error("Invalid config update log: invalid data length: {0}")]
    InvalidDataLength(u64),
    /// Failed to decode the gas limit argument from the gas limit update log.
    #[error("Failed to decode gas limit update log: gas limit")]
    GasLimitDecodingError,
}

/// An error for updating the EIP-1559 parameters on the [`crate::SystemConfig`].
#[derive(Debug, thiserror::Error, Clone, Copy, PartialEq, Eq)]
#[cfg_attr(feature = "serde", derive(serde::Serialize, serde::Deserialize))]
pub enum EIP1559UpdateError {
    /// Invalid data length.
    #[error("Invalid config update log: invalid data length: {0}")]
    InvalidDataLen(usize),
    /// Failed to decode the data pointer argument from the eip 1559 update log.
    #[error("Failed to decode eip1559 parameter update log: data pointer")]
    PointerDecodingError,
    /// The data pointer is invalid.
    #[error("Invalid config update log: invalid data pointer: {0}")]
    InvalidDataPointer(u64),
    /// Failed to decode the data length argument from the eip 1559 update log.
    #[error("Failed to decode eip1559 parameter update log: data length")]
    LengthDecodingError,
    /// The data length is invalid.
    #[error("Invalid config update log: invalid data length: {0}")]
    InvalidDataLength(u64),
    /// Failed to decode the eip1559 params argument from the eip 1559 update log.
    #[error("Failed to decode eip1559 parameter update log: eip1559 parameters")]
    EIP1559DecodingError,
}

/// An error for updating the operator fee parameters on the [`crate::SystemConfig`].
#[derive(Debug, thiserror::Error, Clone, Copy, PartialEq, Eq)]
#[cfg_attr(feature = "serde", derive(serde::Serialize, serde::Deserialize))]
pub enum OperatorFeeUpdateError {
    /// Invalid data length.
    #[error("Invalid config update log: invalid data length: {0}")]
    InvalidDataLen(usize),
    /// Failed to decode the data pointer argument from the operator fee update log.
    #[error("Failed to decode operator fee parameter update log: data pointer")]
    PointerDecodingError,
    /// The data pointer is invalid.
    #[error("Invalid config update log: invalid data pointer: {0}")]
    InvalidDataPointer(u64),
    /// Failed to decode the data length argument from the operator fee update log.
    #[error("Failed to decode operator fee parameter update log: data length")]
    LengthDecodingError,
    /// The data length is invalid.
    #[error("Invalid config update log: invalid data length: {0}")]
    InvalidDataLength(u64),
    /// Failed to decode the scalar argument from the update log.
    #[error("Failed to decode operator fee parameter update log: scalar")]
    ScalarDecodingError,
    /// Failed to decode the constant argument from the update log.
    #[error("Failed to decode operator fee parameter update log: constant")]
    ConstantDecodingError,
}
