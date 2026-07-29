//! Commonly used types that contain metadata about a transaction.

use alloy_consensus::transaction::TransactionInfo;

/// Additional receipt metadata required for deposit transactions.
///
/// These fields are used to provide additional context for deposit transactions in RPC responses
#[derive(Debug, Clone, Copy, Default, Eq, PartialEq)]
pub struct OpDepositInfo {
    /// Nonce for deposit transactions. Only present in RPC responses.
    pub deposit_nonce: Option<u64>,
    /// Deposit receipt version for deposit transactions post-canyon
    pub deposit_receipt_version: Option<u64>,
}

/// Additional fields in the context of a block that contains this transaction and its deposit
/// metadata if the transaction is a deposit.
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq)]
pub struct OpTransactionInfo {
    /// Additional transaction information.
    pub inner: TransactionInfo,
    /// Additional metadata for deposit transactions.
    pub deposit_meta: OpDepositInfo,
}

impl OpTransactionInfo {
    /// Creates a new [`OpTransactionInfo`] with the given [`TransactionInfo`] and
    /// [`OpDepositInfo`].
    pub const fn new(inner: TransactionInfo, deposit_meta: OpDepositInfo) -> Self {
        Self { inner, deposit_meta }
    }
}
