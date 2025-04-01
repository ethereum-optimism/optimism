//! RPC errors specific to OP.

use alloy_rpc_types_eth::{error::EthRpcErrorCode, BlockError};
use alloy_transport::{RpcError, TransportErrorKind};
use jsonrpsee_types::error::{INTERNAL_ERROR_CODE, INVALID_PARAMS_CODE};
use op_revm::{OpHaltReason, OpTransactionError};
use reth_optimism_evm::OpBlockExecutionError;
use reth_rpc_eth_api::AsEthApiError;
use reth_rpc_eth_types::{error::api::FromEvmHalt, EthApiError};
use reth_rpc_server_types::result::{internal_rpc_err, rpc_err};
use revm::context_interface::result::{EVMError, InvalidTransaction};
use std::fmt::Display;

/// Optimism specific errors, that extend [`EthApiError`].
#[derive(Debug, thiserror::Error)]
pub enum OpEthApiError {
    /// L1 ethereum error.
    #[error(transparent)]
    Eth(#[from] EthApiError),
    /// EVM error originating from invalid optimism data.
    #[error(transparent)]
    Evm(#[from] OpBlockExecutionError),
    /// Thrown when calculating L1 gas fee.
    #[error("failed to calculate l1 gas fee")]
    L1BlockFeeError,
    /// Thrown when calculating L1 gas used
    #[error("failed to calculate l1 gas used")]
    L1BlockGasError,
    /// Wrapper for [`revm_primitives::InvalidTransaction`](InvalidTransaction).
    #[error(transparent)]
    InvalidTransaction(#[from] OpInvalidTransactionError),
    /// Sequencer client error.
    #[error(transparent)]
    Sequencer(#[from] SequencerClientError),
}

impl AsEthApiError for OpEthApiError {
    fn as_err(&self) -> Option<&EthApiError> {
        match self {
            Self::Eth(err) => Some(err),
            _ => None,
        }
    }
}

impl From<OpEthApiError> for jsonrpsee_types::error::ErrorObject<'static> {
    fn from(err: OpEthApiError) -> Self {
        match err {
            OpEthApiError::Eth(err) => err.into(),
            OpEthApiError::InvalidTransaction(err) => err.into(),
            OpEthApiError::Evm(_) |
            OpEthApiError::L1BlockFeeError |
            OpEthApiError::L1BlockGasError => internal_rpc_err(err.to_string()),
            OpEthApiError::Sequencer(err) => err.into(),
        }
    }
}

/// Optimism specific invalid transaction errors
#[derive(thiserror::Error, Debug)]
pub enum OpInvalidTransactionError {
    /// A deposit transaction was submitted as a system transaction post-regolith.
    #[error("no system transactions allowed after regolith")]
    DepositSystemTxPostRegolith,
    /// A deposit transaction halted post-regolith
    #[error("deposit transaction halted after regolith")]
    HaltedDepositPostRegolith,
    /// Transaction conditional errors.
    #[error(transparent)]
    TxConditionalErr(#[from] TxConditionalErr),
}

impl From<OpInvalidTransactionError> for jsonrpsee_types::error::ErrorObject<'static> {
    fn from(err: OpInvalidTransactionError) -> Self {
        match err {
            OpInvalidTransactionError::DepositSystemTxPostRegolith |
            OpInvalidTransactionError::HaltedDepositPostRegolith => {
                rpc_err(EthRpcErrorCode::TransactionRejected.code(), err.to_string(), None)
            }
            OpInvalidTransactionError::TxConditionalErr(_) => err.into(),
        }
    }
}

impl TryFrom<OpTransactionError> for OpInvalidTransactionError {
    type Error = InvalidTransaction;

    fn try_from(err: OpTransactionError) -> Result<Self, Self::Error> {
        match err {
            OpTransactionError::DepositSystemTxPostRegolith => {
                Ok(Self::DepositSystemTxPostRegolith)
            }
            OpTransactionError::HaltedDepositPostRegolith => Ok(Self::HaltedDepositPostRegolith),
            OpTransactionError::Base(err) => Err(err),
        }
    }
}

/// Transaction conditional related errors.
#[derive(Debug, thiserror::Error)]
pub enum TxConditionalErr {
    /// Transaction conditional cost exceeded maximum allowed
    #[error("conditional cost exceeded maximum allowed")]
    ConditionalCostExceeded,
    /// Invalid conditional parameters
    #[error("invalid conditional parameters")]
    InvalidCondition,
    /// Internal error
    #[error("internal error: {0}")]
    Internal(String),
    /// Thrown if the conditional's storage value doesn't match the latest state's.
    #[error("storage value mismatch")]
    StorageValueMismatch,
    /// Thrown when the conditional's storage root doesn't match the latest state's root.
    #[error("storage root mismatch")]
    StorageRootMismatch,
}

impl TxConditionalErr {
    /// Creates an internal error variant
    pub fn internal<E: Display>(err: E) -> Self {
        Self::Internal(err.to_string())
    }
}

impl From<TxConditionalErr> for jsonrpsee_types::error::ErrorObject<'static> {
    fn from(err: TxConditionalErr) -> Self {
        let code = match &err {
            TxConditionalErr::Internal(_) => INTERNAL_ERROR_CODE,
            _ => INVALID_PARAMS_CODE,
        };

        jsonrpsee_types::error::ErrorObject::owned(code, err.to_string(), None::<String>)
    }
}

/// Error type when interacting with the Sequencer
#[derive(Debug, thiserror::Error)]
pub enum SequencerClientError {
    /// Wrapper around an [`RpcError<TransportErrorKind>`].
    #[error(transparent)]
    HttpError(#[from] RpcError<TransportErrorKind>),
    /// Thrown when serializing transaction to forward to sequencer
    #[error("invalid sequencer transaction")]
    InvalidSequencerTransaction,
}

impl From<SequencerClientError> for jsonrpsee_types::error::ErrorObject<'static> {
    fn from(err: SequencerClientError) -> Self {
        jsonrpsee_types::error::ErrorObject::owned(
            INTERNAL_ERROR_CODE,
            err.to_string(),
            None::<String>,
        )
    }
}

impl From<BlockError> for OpEthApiError {
    fn from(error: BlockError) -> Self {
        Self::Eth(error.into())
    }
}

impl<T> From<EVMError<T, OpTransactionError>> for OpEthApiError
where
    T: Into<EthApiError>,
{
    fn from(error: EVMError<T, OpTransactionError>) -> Self {
        match error {
            EVMError::Transaction(err) => match err.try_into() {
                Ok(err) => Self::InvalidTransaction(err),
                Err(err) => Self::Eth(EthApiError::InvalidTransaction(err.into())),
            },
            EVMError::Database(err) => Self::Eth(err.into()),
            EVMError::Header(err) => Self::Eth(err.into()),
            EVMError::Custom(err) => Self::Eth(EthApiError::EvmCustom(err)),
        }
    }
}

impl FromEvmHalt<OpHaltReason> for OpEthApiError {
    fn from_evm_halt(halt: OpHaltReason, gas_limit: u64) -> Self {
        match halt {
            OpHaltReason::FailedDeposit => {
                OpInvalidTransactionError::HaltedDepositPostRegolith.into()
            }
            OpHaltReason::Base(halt) => EthApiError::from_evm_halt(halt, gas_limit).into(),
        }
    }
}
