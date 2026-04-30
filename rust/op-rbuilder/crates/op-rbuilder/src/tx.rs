use std::{borrow::Cow, sync::Arc};

use alloy_consensus::{BlobTransactionValidationError, conditional::BlockConditionalAttributes};
use alloy_eips::{Typed2718, eip7594::BlobTransactionSidecarVariant, eip7702::SignedAuthorization};
use alloy_primitives::{Address, B256, Bytes, TxHash, TxKind, U256};
use alloy_rpc_types_eth::{AccessList, erc4337::TransactionConditional};
use reth_optimism_primitives::OpTransactionSigned;
use reth_optimism_txpool::{
    OpPooledTransaction, OpPooledTx, conditional::MaybeConditionalTransaction,
    estimated_da_size::DataAvailabilitySized, interop::MaybeInteropTransaction,
};
use reth_primitives::{Recovered, kzg::KzgSettings};
use reth_primitives_traits::InMemorySize;
use reth_transaction_pool::{EthBlobTransactionSidecar, EthPoolTransaction, PoolTransaction};

pub trait FBPoolTransaction:
    MaybeRevertingTransaction + OpPooledTx + MaybeFlashblockFilter
{
}

#[derive(Clone, Debug)]
pub struct FBPooledTransaction {
    pub inner: OpPooledTransaction,

    /// reverted hashes for the transaction. If the transaction is a bundle,
    /// this is the list of hashes of the transactions that reverted. If the
    /// transaction is not a bundle, this is `None`.
    pub reverted_hashes: Option<Vec<B256>>,

    pub flashblock_number_min: Option<u64>,
    pub flashblock_number_max: Option<u64>,
}

impl FBPoolTransaction for FBPooledTransaction {}

impl OpPooledTx for FBPooledTransaction {
    fn encoded_2718(&self) -> Cow<'_, Bytes> {
        Cow::Borrowed(self.inner.encoded_2718())
    }
}

pub trait MaybeRevertingTransaction {
    fn with_reverted_hashes(self, reverted_hashes: Vec<B256>) -> Self;
    fn reverted_hashes(&self) -> Option<Vec<B256>>;
}

impl MaybeRevertingTransaction for FBPooledTransaction {
    fn with_reverted_hashes(mut self, reverted_hashes: Vec<B256>) -> Self {
        self.reverted_hashes = Some(reverted_hashes);
        self
    }

    fn reverted_hashes(&self) -> Option<Vec<B256>> {
        self.reverted_hashes.clone()
    }
}

pub trait MaybeFlashblockFilter {
    fn with_flashblock_number_min(self, flashblock_number_min: Option<u64>) -> Self;
    fn with_flashblock_number_max(self, flashblock_number_max: Option<u64>) -> Self;
    fn flashblock_number_min(&self) -> Option<u64>;
    fn flashblock_number_max(&self) -> Option<u64>;
}

impl MaybeFlashblockFilter for FBPooledTransaction {
    fn with_flashblock_number_min(mut self, flashblock_number_min: Option<u64>) -> Self {
        self.flashblock_number_min = flashblock_number_min;
        self
    }

    fn with_flashblock_number_max(mut self, flashblock_number_max: Option<u64>) -> Self {
        self.flashblock_number_max = flashblock_number_max;
        self
    }

    fn flashblock_number_min(&self) -> Option<u64> {
        self.flashblock_number_min
    }

    fn flashblock_number_max(&self) -> Option<u64> {
        self.flashblock_number_max
    }
}

impl InMemorySize for FBPooledTransaction {
    fn size(&self) -> usize {
        self.inner.size() + core::mem::size_of::<bool>()
    }
}

impl PoolTransaction for FBPooledTransaction {
    type TryFromConsensusError =
        <op_alloy_consensus::OpPooledTransaction as TryFrom<OpTransactionSigned>>::Error;
    type Consensus = OpTransactionSigned;
    type Pooled = op_alloy_consensus::OpPooledTransaction;

    fn clone_into_consensus(&self) -> Recovered<Self::Consensus> {
        self.inner.clone_into_consensus()
    }

    fn into_consensus(self) -> Recovered<Self::Consensus> {
        self.inner.into_consensus()
    }

    fn from_pooled(tx: Recovered<Self::Pooled>) -> Self {
        let inner = OpPooledTransaction::from_pooled(tx);
        Self {
            inner,
            reverted_hashes: None,
            flashblock_number_min: None,
            flashblock_number_max: None,
        }
    }

    fn hash(&self) -> &TxHash {
        self.inner.hash()
    }

    fn sender(&self) -> Address {
        self.inner.sender()
    }

    fn sender_ref(&self) -> &Address {
        self.inner.sender_ref()
    }

    fn cost(&self) -> &U256 {
        self.inner.cost()
    }

    fn encoded_length(&self) -> usize {
        self.inner.encoded_length()
    }
}

impl Typed2718 for FBPooledTransaction {
    fn ty(&self) -> u8 {
        self.inner.ty()
    }
}

impl alloy_consensus::Transaction for FBPooledTransaction {
    fn chain_id(&self) -> Option<u64> {
        self.inner.chain_id()
    }

    fn nonce(&self) -> u64 {
        self.inner.nonce()
    }

    fn gas_limit(&self) -> u64 {
        self.inner.gas_limit()
    }

    fn gas_price(&self) -> Option<u128> {
        self.inner.gas_price()
    }

    fn max_fee_per_gas(&self) -> u128 {
        self.inner.max_fee_per_gas()
    }

    fn max_priority_fee_per_gas(&self) -> Option<u128> {
        self.inner.max_priority_fee_per_gas()
    }

    fn max_fee_per_blob_gas(&self) -> Option<u128> {
        self.inner.max_fee_per_blob_gas()
    }

    fn priority_fee_or_price(&self) -> u128 {
        self.inner.priority_fee_or_price()
    }

    fn effective_gas_price(&self, base_fee: Option<u64>) -> u128 {
        self.inner.effective_gas_price(base_fee)
    }

    fn is_dynamic_fee(&self) -> bool {
        self.inner.is_dynamic_fee()
    }

    fn kind(&self) -> TxKind {
        self.inner.kind()
    }

    fn is_create(&self) -> bool {
        self.inner.is_create()
    }

    fn value(&self) -> U256 {
        self.inner.value()
    }

    fn input(&self) -> &Bytes {
        self.inner.input()
    }

    fn access_list(&self) -> Option<&AccessList> {
        self.inner.access_list()
    }

    fn blob_versioned_hashes(&self) -> Option<&[B256]> {
        self.inner.blob_versioned_hashes()
    }

    fn authorization_list(&self) -> Option<&[SignedAuthorization]> {
        self.inner.authorization_list()
    }
}

impl EthPoolTransaction for FBPooledTransaction {
    fn take_blob(&mut self) -> EthBlobTransactionSidecar {
        EthBlobTransactionSidecar::None
    }

    fn try_into_pooled_eip4844(
        self,
        sidecar: Arc<BlobTransactionSidecarVariant>,
    ) -> Option<Recovered<Self::Pooled>> {
        self.inner.try_into_pooled_eip4844(sidecar)
    }

    fn try_from_eip4844(
        _tx: Recovered<Self::Consensus>,
        _sidecar: BlobTransactionSidecarVariant,
    ) -> Option<Self> {
        None
    }

    fn validate_blob(
        &self,
        _sidecar: &BlobTransactionSidecarVariant,
        _settings: &KzgSettings,
    ) -> Result<(), BlobTransactionValidationError> {
        Err(BlobTransactionValidationError::NotBlobTransaction(
            self.ty(),
        ))
    }
}

impl MaybeInteropTransaction for FBPooledTransaction {
    fn interop_deadline(&self) -> Option<u64> {
        self.inner.interop_deadline()
    }

    fn set_interop_deadline(&self, deadline: u64) {
        self.inner.set_interop_deadline(deadline);
    }

    fn with_interop_deadline(self, interop: u64) -> Self
    where
        Self: Sized,
    {
        self.inner.with_interop_deadline(interop).into()
    }
}

impl DataAvailabilitySized for FBPooledTransaction {
    fn estimated_da_size(&self) -> u64 {
        self.inner.estimated_da_size()
    }
}

impl From<OpPooledTransaction> for FBPooledTransaction {
    fn from(tx: OpPooledTransaction) -> Self {
        Self {
            inner: tx,
            reverted_hashes: None,
            flashblock_number_min: None,
            flashblock_number_max: None,
        }
    }
}

impl MaybeConditionalTransaction for FBPooledTransaction {
    fn set_conditional(&mut self, conditional: TransactionConditional) {
        self.inner.set_conditional(conditional);
    }

    fn conditional(&self) -> Option<&TransactionConditional> {
        self.inner.conditional()
    }

    fn has_exceeded_block_attributes(&self, block_attr: &BlockConditionalAttributes) -> bool {
        self.inner.has_exceeded_block_attributes(block_attr)
    }

    fn with_conditional(self, conditional: TransactionConditional) -> Self
    where
        Self: Sized,
    {
        FBPooledTransaction {
            inner: self.inner.with_conditional(conditional),
            reverted_hashes: self.reverted_hashes,
            flashblock_number_min: self.flashblock_number_min,
            flashblock_number_max: self.flashblock_number_max,
        }
    }
}
