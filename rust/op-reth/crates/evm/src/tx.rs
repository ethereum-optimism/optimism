//! [`OpTx`] newtype wrapper around [`OpTransaction<TxEnv>`].

use alloy_consensus::{
    Signed, TxEip1559, TxEip2930, TxEip4844, TxEip4844Variant, TxEip7702, TxLegacy,
};
use alloy_eips::{Encodable2718, Typed2718, eip7594::Encodable7594};
use alloy_evm::{FromRecoveredTx, FromTxWithEncoded, IntoTxEnv};
use alloy_op_evm::block::OpTxEnv;
use alloy_primitives::{Address, B256, Bytes, TxKind, U256};
use core::ops::{Deref, DerefMut};
use op_alloy_consensus::{OpTxEnvelope, TxDeposit};
use op_revm::{OpTransaction, transaction::deposit::DepositTransactionParts};
use reth_evm::TransactionEnv;
use revm::context::TxEnv;

/// Helper to convert a deposit transaction into a [`TxEnv`].
fn deposit_tx_env(tx: &TxDeposit, caller: Address) -> TxEnv {
    TxEnv {
        tx_type: tx.ty(),
        caller,
        gas_limit: tx.gas_limit,
        kind: tx.to,
        value: tx.value,
        data: tx.input.clone(),
        ..Default::default()
    }
}

/// Newtype wrapper around [`OpTransaction<TxEnv>`] that allows implementing foreign traits.
#[derive(Clone, Debug, Default)]
pub struct OpTx(pub OpTransaction<TxEnv>);

impl From<OpTx> for OpTransaction<TxEnv> {
    fn from(tx: OpTx) -> Self {
        tx.0
    }
}

impl Deref for OpTx {
    type Target = OpTransaction<TxEnv>;

    fn deref(&self) -> &Self::Target {
        &self.0
    }
}

impl DerefMut for OpTx {
    fn deref_mut(&mut self) -> &mut Self::Target {
        &mut self.0
    }
}

impl IntoTxEnv<Self> for OpTx {
    fn into_tx_env(self) -> Self {
        self
    }
}

impl OpTxEnv for OpTx {
    fn encoded_bytes(&self) -> Option<&Bytes> {
        self.0.enveloped_tx.as_ref()
    }
}

impl revm::context::Transaction for OpTx {
    type AccessListItem<'a>
        = <OpTransaction<TxEnv> as revm::context::Transaction>::AccessListItem<'a>
    where
        Self: 'a;
    type Authorization<'a>
        = <OpTransaction<TxEnv> as revm::context::Transaction>::Authorization<'a>
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

impl FromRecoveredTx<OpTxEnvelope> for OpTx {
    fn from_recovered_tx(tx: &OpTxEnvelope, sender: Address) -> Self {
        let encoded = tx.encoded_2718();
        Self::from_encoded_tx(tx, sender, encoded.into())
    }
}

impl FromTxWithEncoded<OpTxEnvelope> for OpTx {
    fn from_encoded_tx(tx: &OpTxEnvelope, caller: Address, encoded: Bytes) -> Self {
        match tx {
            OpTxEnvelope::Legacy(tx) => Self::from_encoded_tx(tx, caller, encoded),
            OpTxEnvelope::Eip1559(tx) => Self::from_encoded_tx(tx, caller, encoded),
            OpTxEnvelope::Eip2930(tx) => Self::from_encoded_tx(tx, caller, encoded),
            OpTxEnvelope::Eip7702(tx) => Self::from_encoded_tx(tx, caller, encoded),
            OpTxEnvelope::Deposit(tx) => Self::from_encoded_tx(tx.inner(), caller, encoded),
        }
    }
}

/// Generates [`FromRecoveredTx`] and [`FromTxWithEncoded`] impls for [`OpTx`] from a
/// `Signed<$tx>` and bare `$tx` type. The bare type conversion creates the [`TxEnv`] via
/// [`FromRecoveredTx`] and wraps it in an [`OpTransaction`].
macro_rules! impl_from_tx {
    ($($tx:ty),+ $(,)?) => {
        $(
            impl FromRecoveredTx<Signed<$tx>> for OpTx {
                fn from_recovered_tx(tx: &Signed<$tx>, sender: Address) -> Self {
                    let encoded = tx.encoded_2718();
                    Self::from_encoded_tx(tx, sender, encoded.into())
                }
            }

            impl FromTxWithEncoded<Signed<$tx>> for OpTx {
                fn from_encoded_tx(tx: &Signed<$tx>, caller: Address, encoded: Bytes) -> Self {
                    Self::from_encoded_tx(tx.tx(), caller, encoded)
                }
            }

            impl FromTxWithEncoded<$tx> for OpTx {
                fn from_encoded_tx(tx: &$tx, caller: Address, encoded: Bytes) -> Self {
                    let base = TxEnv::from_recovered_tx(tx, caller);
                    Self(OpTransaction {
                        base,
                        enveloped_tx: Some(encoded),
                        deposit: Default::default(),
                    })
                }
            }
        )+
    };
}

impl_from_tx!(TxLegacy, TxEip2930, TxEip1559, TxEip4844, TxEip7702);

/// `TxEip4844Variant<T>` conversion is not necessary for `OpTx`, but it's useful
/// sugar for Foundry.
impl<T> FromRecoveredTx<Signed<TxEip4844Variant<T>>> for OpTx
where
    T: Encodable7594 + Send + Sync,
{
    fn from_recovered_tx(tx: &Signed<TxEip4844Variant<T>>, sender: Address) -> Self {
        let encoded = tx.encoded_2718();
        Self::from_encoded_tx(tx, sender, encoded.into())
    }
}

impl<T> FromTxWithEncoded<Signed<TxEip4844Variant<T>>> for OpTx {
    fn from_encoded_tx(tx: &Signed<TxEip4844Variant<T>>, caller: Address, encoded: Bytes) -> Self {
        Self::from_encoded_tx(tx.tx(), caller, encoded)
    }
}

impl<T> FromTxWithEncoded<TxEip4844Variant<T>> for OpTx {
    fn from_encoded_tx(tx: &TxEip4844Variant<T>, caller: Address, encoded: Bytes) -> Self {
        let base = TxEnv::from_recovered_tx(tx, caller);
        Self(OpTransaction { base, enveloped_tx: Some(encoded), deposit: Default::default() })
    }
}

impl FromRecoveredTx<TxDeposit> for OpTx {
    fn from_recovered_tx(tx: &TxDeposit, sender: Address) -> Self {
        let encoded = tx.encoded_2718();
        Self::from_encoded_tx(tx, sender, encoded.into())
    }
}

impl FromTxWithEncoded<TxDeposit> for OpTx {
    fn from_encoded_tx(tx: &TxDeposit, caller: Address, encoded: Bytes) -> Self {
        let base = deposit_tx_env(tx, caller);
        let deposit = DepositTransactionParts {
            source_hash: tx.source_hash,
            mint: Some(tx.mint),
            is_system_transaction: tx.is_system_transaction,
        };
        Self(OpTransaction { base, enveloped_tx: Some(encoded), deposit })
    }
}

#[cfg(feature = "rpc")]
impl<Block: alloy_evm::env::BlockEnvironment> alloy_evm::rpc::TryIntoTxEnv<OpTx, Block>
    for op_alloy_rpc_types::OpTransactionRequest
{
    type Err = alloy_evm::rpc::EthTxEnvError;

    fn try_into_tx_env<Spec>(
        self,
        evm_env: &alloy_evm::EvmEnv<Spec, Block>,
    ) -> Result<OpTx, Self::Err> {
        let inner: OpTransaction<TxEnv> = self.try_into_tx_env(evm_env)?;
        Ok(OpTx(inner))
    }
}

impl TransactionEnv for OpTx {
    fn set_gas_limit(&mut self, gas_limit: u64) {
        self.0.base.gas_limit = gas_limit;
    }

    fn nonce(&self) -> u64 {
        self.0.base.nonce
    }

    fn set_nonce(&mut self, nonce: u64) {
        self.0.base.nonce = nonce;
    }

    fn set_access_list(&mut self, access_list: alloy_eips::eip2930::AccessList) {
        self.0.base.access_list = access_list;
    }
}
