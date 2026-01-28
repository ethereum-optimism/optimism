use alloc::vec::Vec;
use alloy_consensus::{
    Sealed, SignableTransaction, Signed, TxEip1559, TxEip4844, TypedTransaction,
};
use alloy_eips::eip7702::SignedAuthorization;
use alloy_network_primitives::TransactionBuilder7702;
use alloy_primitives::{Address, Signature, TxKind, U256};
use alloy_rpc_types_eth::{AccessList, TransactionInput, TransactionRequest};
use op_alloy_consensus::{OpTxEnvelope, OpTypedTransaction, TxDeposit};
use serde::{Deserialize, Serialize};

/// Builder for [`OpTypedTransaction`].
#[derive(
    Clone,
    Debug,
    Default,
    PartialEq,
    Eq,
    Hash,
    derive_more::From,
    derive_more::AsRef,
    derive_more::AsMut,
    Serialize,
    Deserialize,
)]
#[serde(transparent)]
pub struct OpTransactionRequest(TransactionRequest);

impl OpTransactionRequest {
    /// Sets the `from` field in the call to the provided address
    #[inline]
    pub const fn from(mut self, from: Address) -> Self {
        self.0.from = Some(from);
        self
    }

    /// Sets the transactions type for the transactions.
    #[doc(alias = "tx_type")]
    pub const fn transaction_type(mut self, transaction_type: u8) -> Self {
        self.0.transaction_type = Some(transaction_type);
        self
    }

    /// Sets the gas limit for the transaction.
    pub const fn gas_limit(mut self, gas_limit: u64) -> Self {
        self.0.gas = Some(gas_limit);
        self
    }

    /// Sets the nonce for the transaction.
    pub const fn nonce(mut self, nonce: u64) -> Self {
        self.0.nonce = Some(nonce);
        self
    }

    /// Sets the maximum fee per gas for the transaction.
    pub const fn max_fee_per_gas(mut self, max_fee_per_gas: u128) -> Self {
        self.0.max_fee_per_gas = Some(max_fee_per_gas);
        self
    }

    /// Sets the maximum priority fee per gas for the transaction.
    pub const fn max_priority_fee_per_gas(mut self, max_priority_fee_per_gas: u128) -> Self {
        self.0.max_priority_fee_per_gas = Some(max_priority_fee_per_gas);
        self
    }

    /// Sets the recipient address for the transaction.
    #[inline]
    pub const fn to(mut self, to: Address) -> Self {
        self.0.to = Some(TxKind::Call(to));
        self
    }

    /// Sets the value (amount) for the transaction.
    pub const fn value(mut self, value: U256) -> Self {
        self.0.value = Some(value);
        self
    }

    /// Sets the access list for the transaction.
    pub fn access_list(mut self, access_list: AccessList) -> Self {
        self.0.access_list = Some(access_list);
        self
    }

    /// Sets the input data for the transaction.
    pub fn input(mut self, input: TransactionInput) -> Self {
        self.0.input = input;
        self
    }

    /// Builds [`OpTypedTransaction`] from this builder. See [`TransactionRequest::build_typed_tx`]
    /// for more info.
    ///
    /// Note that EIP-4844 transactions are not supported by Optimism and will be converted into
    /// EIP-1559 transactions.
    pub fn build_typed_tx(self) -> Result<OpTypedTransaction, Self> {
        let tx = self.0.build_typed_tx().map_err(Self)?;
        match tx {
            TypedTransaction::Legacy(tx) => Ok(OpTypedTransaction::Legacy(tx)),
            TypedTransaction::Eip1559(tx) => Ok(OpTypedTransaction::Eip1559(tx)),
            TypedTransaction::Eip2930(tx) => Ok(OpTypedTransaction::Eip2930(tx)),
            TypedTransaction::Eip4844(tx) => {
                let tx: TxEip4844 = tx.into();
                Ok(OpTypedTransaction::Eip1559(TxEip1559 {
                    chain_id: tx.chain_id,
                    nonce: tx.nonce,
                    gas_limit: tx.gas_limit,
                    max_priority_fee_per_gas: tx.max_priority_fee_per_gas,
                    max_fee_per_gas: tx.max_fee_per_gas,
                    to: TxKind::Call(tx.to),
                    value: tx.value,
                    access_list: tx.access_list,
                    input: tx.input,
                }))
            }
            TypedTransaction::Eip7702(tx) => Ok(OpTypedTransaction::Eip7702(tx)),
        }
    }
}

impl From<OpTransactionRequest> for TransactionRequest {
    fn from(value: OpTransactionRequest) -> Self {
        value.0
    }
}

impl From<TxDeposit> for OpTransactionRequest {
    fn from(tx: TxDeposit) -> Self {
        let TxDeposit {
            source_hash: _,
            from,
            to,
            mint: _,
            value,
            gas_limit,
            is_system_transaction: _,
            input,
        } = tx;

        Self(TransactionRequest {
            from: Some(from),
            to: Some(to),
            value: Some(value),
            gas: Some(gas_limit),
            input: input.into(),
            ..Default::default()
        })
    }
}

impl From<Sealed<TxDeposit>> for OpTransactionRequest {
    fn from(value: Sealed<TxDeposit>) -> Self {
        value.into_inner().into()
    }
}

impl<T> From<Signed<T, Signature>> for OpTransactionRequest
where
    T: SignableTransaction<Signature> + Into<TransactionRequest>,
{
    fn from(value: Signed<T, Signature>) -> Self {
        #[cfg(feature = "k256")]
        let from = value.recover_signer().ok();
        #[cfg(not(feature = "k256"))]
        let from = None;

        let mut inner: TransactionRequest = value.strip_signature().into();
        inner.from = from;

        Self(inner)
    }
}

impl From<OpTypedTransaction> for OpTransactionRequest {
    fn from(tx: OpTypedTransaction) -> Self {
        match tx {
            OpTypedTransaction::Legacy(tx) => Self(tx.into()),
            OpTypedTransaction::Eip2930(tx) => Self(tx.into()),
            OpTypedTransaction::Eip1559(tx) => Self(tx.into()),
            OpTypedTransaction::Eip7702(tx) => Self(tx.into()),
            OpTypedTransaction::Deposit(tx) => tx.into(),
        }
    }
}

impl From<OpTxEnvelope> for OpTransactionRequest {
    fn from(value: OpTxEnvelope) -> Self {
        match value {
            OpTxEnvelope::Eip2930(tx) => tx.into(),
            OpTxEnvelope::Eip1559(tx) => tx.into(),
            OpTxEnvelope::Eip7702(tx) => tx.into(),
            OpTxEnvelope::Deposit(tx) => tx.into(),
            _ => Default::default(),
        }
    }
}

impl TransactionBuilder7702 for OpTransactionRequest {
    fn authorization_list(&self) -> Option<&Vec<SignedAuthorization>> {
        self.as_ref().authorization_list()
    }

    fn set_authorization_list(&mut self, authorization_list: Vec<SignedAuthorization>) {
        self.as_mut().set_authorization_list(authorization_list);
    }
}
