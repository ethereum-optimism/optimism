//! Error types for OP EVM execution.

use alloy_evm::InvalidTxError;
use core::fmt;
use op_revm::OpTransactionError;
use revm::context_interface::result::{EVMError, InvalidTransaction};

/// Newtype wrapper around [`OpTransactionError`] that allows implementing foreign traits.
#[derive(Debug)]
pub struct OpTxError(pub OpTransactionError);

impl fmt::Display for OpTxError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        self.0.fmt(f)
    }
}

impl core::error::Error for OpTxError {
    fn source(&self) -> Option<&(dyn core::error::Error + 'static)> {
        self.0.source()
    }
}

impl InvalidTxError for OpTxError {
    fn as_invalid_tx_err(&self) -> Option<&InvalidTransaction> {
        match &self.0 {
            OpTransactionError::Base(tx) => Some(tx),
            _ => None,
        }
    }
}

/// Maps an [`EVMError<DB, OpTransactionError>`] to [`EVMError<DB, OpTxError>`].
pub fn map_op_err<DB>(err: EVMError<DB, OpTransactionError>) -> EVMError<DB, OpTxError> {
    match err {
        EVMError::Transaction(e) => EVMError::Transaction(OpTxError(e)),
        EVMError::Database(e) => EVMError::Database(e),
        EVMError::Header(e) => EVMError::Header(e),
        EVMError::Custom(e) => EVMError::Custom(e),
    }
}
