use crate::primitives::{CustomTransaction, TxPayment};
use alloy_eips::{eip2930::AccessList, Typed2718};
use alloy_evm::{FromRecoveredTx, FromTxWithEncoded, IntoTxEnv};
use alloy_op_evm::block::OpTxEnv;
use alloy_primitives::{Address, Bytes, TxKind, B256, U256};
use op_alloy_consensus::OpTxEnvelope;
use op_revm::OpTransaction;
use reth_ethereum::evm::{primitives::TransactionEnv, revm::context::TxEnv};

/// An Optimism transaction extended by [`PaymentTxEnv`] that can be fed to [`Evm`].
///
/// [`Evm`]: alloy_evm::Evm
#[derive(Clone, Debug)]
pub enum CustomTxEnv {
    Op(OpTransaction<TxEnv>),
    Payment(PaymentTxEnv),
}

/// A transaction environment is a set of information related to an Ethereum transaction that can be
/// fed to [`Evm`] for execution.
///
/// [`Evm`]: alloy_evm::Evm
#[derive(Clone, Debug, Default)]
pub struct PaymentTxEnv(pub TxEnv);

impl revm::context::Transaction for CustomTxEnv {
    type AccessListItem<'a>
        = <TxEnv as revm::context::Transaction>::AccessListItem<'a>
    where
        Self: 'a;
    type Authorization<'a>
        = <TxEnv as revm::context::Transaction>::Authorization<'a>
    where
        Self: 'a;

    fn tx_type(&self) -> u8 {
        match self {
            Self::Op(tx) => tx.tx_type(),
            Self::Payment(tx) => tx.tx_type(),
        }
    }

    fn caller(&self) -> Address {
        match self {
            Self::Op(tx) => tx.caller(),
            Self::Payment(tx) => tx.caller(),
        }
    }

    fn gas_limit(&self) -> u64 {
        match self {
            Self::Op(tx) => tx.gas_limit(),
            Self::Payment(tx) => tx.gas_limit(),
        }
    }

    fn value(&self) -> U256 {
        match self {
            Self::Op(tx) => tx.value(),
            Self::Payment(tx) => tx.value(),
        }
    }

    fn input(&self) -> &Bytes {
        match self {
            Self::Op(tx) => tx.input(),
            Self::Payment(tx) => tx.input(),
        }
    }

    fn nonce(&self) -> u64 {
        match self {
            Self::Op(tx) => revm::context::Transaction::nonce(tx),
            Self::Payment(tx) => revm::context::Transaction::nonce(tx),
        }
    }

    fn kind(&self) -> TxKind {
        match self {
            Self::Op(tx) => tx.kind(),
            Self::Payment(tx) => tx.kind(),
        }
    }

    fn chain_id(&self) -> Option<u64> {
        match self {
            Self::Op(tx) => tx.chain_id(),
            Self::Payment(tx) => tx.chain_id(),
        }
    }

    fn gas_price(&self) -> u128 {
        match self {
            Self::Op(tx) => tx.gas_price(),
            Self::Payment(tx) => tx.gas_price(),
        }
    }

    fn access_list(&self) -> Option<impl Iterator<Item = Self::AccessListItem<'_>>> {
        Some(match self {
            Self::Op(tx) => tx.base.access_list.iter(),
            Self::Payment(tx) => tx.0.access_list.iter(),
        })
    }

    fn blob_versioned_hashes(&self) -> &[B256] {
        match self {
            Self::Op(tx) => tx.blob_versioned_hashes(),
            Self::Payment(tx) => tx.blob_versioned_hashes(),
        }
    }

    fn max_fee_per_blob_gas(&self) -> u128 {
        match self {
            Self::Op(tx) => tx.max_fee_per_blob_gas(),
            Self::Payment(tx) => tx.max_fee_per_blob_gas(),
        }
    }

    fn authorization_list_len(&self) -> usize {
        match self {
            Self::Op(tx) => tx.authorization_list_len(),
            Self::Payment(tx) => tx.authorization_list_len(),
        }
    }

    fn authorization_list(&self) -> impl Iterator<Item = Self::Authorization<'_>> {
        match self {
            Self::Op(tx) => tx.base.authorization_list.iter(),
            Self::Payment(tx) => tx.0.authorization_list.iter(),
        }
    }

    fn max_priority_fee_per_gas(&self) -> Option<u128> {
        match self {
            Self::Op(tx) => tx.max_priority_fee_per_gas(),
            Self::Payment(tx) => tx.max_priority_fee_per_gas(),
        }
    }
}

impl revm::context::Transaction for PaymentTxEnv {
    type AccessListItem<'a>
        = <TxEnv as revm::context::Transaction>::AccessListItem<'a>
    where
        Self: 'a;
    type Authorization<'a>
        = <TxEnv as revm::context::Transaction>::Authorization<'a>
    where
        Self: 'a;

    fn tx_type(&self) -> u8 {
        self.0.tx_type()
    }

    fn caller(&self) -> Address {
        self.0.caller()
    }

    fn gas_limit(&self) -> u64 {
        self.0.gas_limit()
    }

    fn value(&self) -> U256 {
        self.0.value()
    }

    fn input(&self) -> &Bytes {
        self.0.input()
    }

    fn nonce(&self) -> u64 {
        revm::context::Transaction::nonce(&self.0)
    }

    fn kind(&self) -> TxKind {
        self.0.kind()
    }

    fn chain_id(&self) -> Option<u64> {
        self.0.chain_id()
    }

    fn gas_price(&self) -> u128 {
        self.0.gas_price()
    }

    fn access_list(&self) -> Option<impl Iterator<Item = Self::AccessListItem<'_>>> {
        self.0.access_list()
    }

    fn blob_versioned_hashes(&self) -> &[B256] {
        self.0.blob_versioned_hashes()
    }

    fn max_fee_per_blob_gas(&self) -> u128 {
        self.0.max_fee_per_blob_gas()
    }

    fn authorization_list_len(&self) -> usize {
        self.0.authorization_list_len()
    }

    fn authorization_list(&self) -> impl Iterator<Item = Self::Authorization<'_>> {
        self.0.authorization_list()
    }

    fn max_priority_fee_per_gas(&self) -> Option<u128> {
        self.0.max_priority_fee_per_gas()
    }
}

impl TransactionEnv for PaymentTxEnv {
    fn set_gas_limit(&mut self, gas_limit: u64) {
        self.0.set_gas_limit(gas_limit);
    }

    fn nonce(&self) -> u64 {
        self.0.nonce()
    }

    fn set_nonce(&mut self, nonce: u64) {
        self.0.set_nonce(nonce);
    }

    fn set_access_list(&mut self, access_list: AccessList) {
        self.0.set_access_list(access_list);
    }
}

impl TransactionEnv for CustomTxEnv {
    fn set_gas_limit(&mut self, gas_limit: u64) {
        match self {
            Self::Op(tx) => tx.set_gas_limit(gas_limit),
            Self::Payment(tx) => tx.set_gas_limit(gas_limit),
        }
    }

    fn nonce(&self) -> u64 {
        match self {
            Self::Op(tx) => tx.nonce(),
            Self::Payment(tx) => tx.nonce(),
        }
    }

    fn set_nonce(&mut self, nonce: u64) {
        match self {
            Self::Op(tx) => tx.set_nonce(nonce),
            Self::Payment(tx) => tx.set_nonce(nonce),
        }
    }

    fn set_access_list(&mut self, access_list: AccessList) {
        match self {
            Self::Op(tx) => tx.set_access_list(access_list),
            Self::Payment(tx) => tx.set_access_list(access_list),
        }
    }
}

impl FromRecoveredTx<TxPayment> for TxEnv {
    fn from_recovered_tx(tx: &TxPayment, caller: Address) -> Self {
        let TxPayment {
            chain_id,
            nonce,
            gas_limit,
            max_fee_per_gas,
            max_priority_fee_per_gas,
            to,
            value,
        } = tx;
        Self {
            tx_type: tx.ty(),
            caller,
            gas_limit: *gas_limit,
            gas_price: *max_fee_per_gas,
            gas_priority_fee: Some(*max_priority_fee_per_gas),
            kind: TxKind::Call(*to),
            value: *value,
            nonce: *nonce,
            chain_id: Some(*chain_id),
            ..Default::default()
        }
    }
}

impl FromTxWithEncoded<TxPayment> for TxEnv {
    fn from_encoded_tx(tx: &TxPayment, sender: Address, _encoded: Bytes) -> Self {
        Self::from_recovered_tx(tx, sender)
    }
}

impl FromRecoveredTx<OpTxEnvelope> for CustomTxEnv {
    fn from_recovered_tx(tx: &OpTxEnvelope, sender: Address) -> Self {
        Self::Op(OpTransaction::from_recovered_tx(tx, sender))
    }
}

impl FromTxWithEncoded<OpTxEnvelope> for CustomTxEnv {
    fn from_encoded_tx(tx: &OpTxEnvelope, sender: Address, encoded: Bytes) -> Self {
        Self::Op(OpTransaction::from_encoded_tx(tx, sender, encoded))
    }
}

impl FromRecoveredTx<CustomTransaction> for CustomTxEnv {
    fn from_recovered_tx(tx: &CustomTransaction, sender: Address) -> Self {
        match tx {
            CustomTransaction::Op(tx) => Self::from_recovered_tx(tx, sender),
            CustomTransaction::Payment(tx) => {
                Self::Payment(PaymentTxEnv(TxEnv::from_recovered_tx(tx.tx(), sender)))
            }
        }
    }
}

impl FromTxWithEncoded<CustomTransaction> for CustomTxEnv {
    fn from_encoded_tx(tx: &CustomTransaction, sender: Address, encoded: Bytes) -> Self {
        match tx {
            CustomTransaction::Op(tx) => Self::from_encoded_tx(tx, sender, encoded),
            CustomTransaction::Payment(tx) => {
                Self::Payment(PaymentTxEnv(TxEnv::from_encoded_tx(tx.tx(), sender, encoded)))
            }
        }
    }
}

impl IntoTxEnv<Self> for CustomTxEnv {
    fn into_tx_env(self) -> Self {
        self
    }
}

impl OpTxEnv for CustomTxEnv {
    fn encoded_bytes(&self) -> Option<&Bytes> {
        match self {
            Self::Op(tx) => tx.encoded_bytes(),
            Self::Payment(_) => None,
        }
    }
}
