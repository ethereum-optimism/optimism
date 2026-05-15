use crate::tx::MaybeFlashblockFilter;
use alloy_consensus::{
    EthereumTxEnvelope, TxEip4844, TxEip4844WithSidecar, TxType, error::ValueError,
    transaction::Recovered,
};
use alloy_eips::{
    Typed2718,
    eip2930::AccessList,
    eip4844::{BlobTransactionValidationError, env_settings::KzgSettings},
    eip7594::BlobTransactionSidecarVariant,
    eip7702::SignedAuthorization,
};
use alloy_primitives::{Address, B256, Bytes, TxHash, TxKind, U256};
use reth::primitives::TransactionSigned;
use reth_primitives_traits::{InMemorySize, SignedTransaction};
use reth_transaction_pool::{
    EthBlobTransactionSidecar, EthPoolTransaction, PoolTransaction, TransactionOrigin,
    ValidPoolTransaction,
    identifier::TransactionId,
    test_utils::{MockTransaction, MockTransactionFactory},
};
use std::{sync::Arc, time::Instant};

/// A factory for creating and managing various types of mock transactions.
#[derive(Debug, Default)]
pub struct MockFbTransactionFactory {
    pub(crate) factory: MockTransactionFactory,
}

// === impl MockTransactionFactory ===

impl MockFbTransactionFactory {
    /// Generates a transaction ID for the given [`MockTransaction`].
    pub fn tx_id(&mut self, tx: &MockFbTransaction) -> TransactionId {
        self.factory.tx_id(&tx.inner)
    }

    /// Validates a [`MockTransaction`] and returns a [`MockValidFbTx`].
    pub fn validated(&mut self, transaction: MockFbTransaction) -> MockValidFbTx {
        self.validated_with_origin(TransactionOrigin::External, transaction)
    }

    /// Validates a [`MockTransaction`] and returns a shared [`Arc<MockValidFbTx>`].
    pub fn validated_arc(&mut self, transaction: MockFbTransaction) -> Arc<MockValidFbTx> {
        Arc::new(self.validated(transaction))
    }

    /// Converts the transaction into a validated transaction with a specified origin.
    pub fn validated_with_origin(
        &mut self,
        origin: TransactionOrigin,
        transaction: MockFbTransaction,
    ) -> MockValidFbTx {
        MockValidFbTx {
            propagate: false,
            transaction_id: self.tx_id(&transaction),
            transaction,
            timestamp: Instant::now(),
            origin,
            authority_ids: None,
        }
    }

    /// Creates a validated legacy [`MockTransaction`].
    pub fn create_legacy(&mut self) -> MockValidFbTx {
        self.validated(MockFbTransaction {
            inner: MockTransaction::legacy(),
            reverted_hashes: None,
            flashblock_number_max: None,
            flashblock_number_min: None,
        })
    }

    /// Creates a validated legacy [`MockTransaction`].
    pub fn create_legacy_fb(&mut self, min: Option<u64>, max: Option<u64>) -> MockValidFbTx {
        self.validated(MockFbTransaction {
            inner: MockTransaction::legacy(),
            reverted_hashes: None,
            flashblock_number_max: max,
            flashblock_number_min: min,
        })
    }

    /// Creates a validated EIP-1559 [`MockTransaction`].
    pub fn create_eip1559(&mut self) -> MockValidFbTx {
        self.validated(MockFbTransaction {
            inner: MockTransaction::eip1559(),
            reverted_hashes: None,
            flashblock_number_max: None,
            flashblock_number_min: None,
        })
    }

    /// Creates a validated EIP-4844 [`MockTransaction`].
    pub fn create_eip4844(&mut self) -> MockValidFbTx {
        self.validated(MockFbTransaction {
            inner: MockTransaction::eip4844(),
            reverted_hashes: None,
            flashblock_number_max: None,
            flashblock_number_min: None,
        })
    }
}

#[derive(Debug, Clone, Eq, PartialEq)]
pub struct MockFbTransaction {
    pub inner: MockTransaction,
    /// reverted hashes for the transaction. If the transaction is a bundle,
    /// this is the list of hashes of the transactions that reverted. If the
    /// transaction is not a bundle, this is `None`.
    pub reverted_hashes: Option<Vec<B256>>,

    pub flashblock_number_min: Option<u64>,
    pub flashblock_number_max: Option<u64>,
}

/// A validated transaction in the transaction pool, using [`MockTransaction`] as the transaction
/// type.
///
/// This type is an alias for [`ValidPoolTransaction<MockTransaction>`].
pub type MockValidFbTx = ValidPoolTransaction<MockFbTransaction>;

impl PoolTransaction for MockFbTransaction {
    type TryFromConsensusError = ValueError<EthereumTxEnvelope<TxEip4844>>;

    type Consensus = TransactionSigned;

    type Pooled = PooledTransactionVariant;

    fn into_consensus(self) -> Recovered<Self::Consensus> {
        self.inner.into()
    }

    fn from_pooled(pooled: Recovered<Self::Pooled>) -> Self {
        Self {
            inner: pooled.into(),
            reverted_hashes: None,
            flashblock_number_min: None,
            flashblock_number_max: None,
        }
    }

    fn hash(&self) -> &TxHash {
        self.inner.get_hash()
    }

    fn sender(&self) -> Address {
        *self.inner.get_sender()
    }

    fn sender_ref(&self) -> &Address {
        self.inner.get_sender()
    }

    // Having `get_cost` from `make_setters_getters` would be cleaner but we didn't
    // want to also generate the error-prone cost setters. For now cost should be
    // correct at construction and auto-updated per field update via `update_cost`,
    // not to be manually set.
    fn cost(&self) -> &U256 {
        match &self.inner {
            MockTransaction::Legacy { cost, .. }
            | MockTransaction::Eip2930 { cost, .. }
            | MockTransaction::Eip1559 { cost, .. }
            | MockTransaction::Eip4844 { cost, .. }
            | MockTransaction::Eip7702 { cost, .. } => cost,
        }
    }

    /// Returns the encoded length of the transaction.
    fn encoded_length(&self) -> usize {
        self.inner.size()
    }
}

impl InMemorySize for MockFbTransaction {
    fn size(&self) -> usize {
        *self.inner.get_size()
    }
}

impl Typed2718 for MockFbTransaction {
    fn ty(&self) -> u8 {
        match self.inner {
            MockTransaction::Legacy { .. } => TxType::Legacy.into(),
            MockTransaction::Eip1559 { .. } => TxType::Eip1559.into(),
            MockTransaction::Eip4844 { .. } => TxType::Eip4844.into(),
            MockTransaction::Eip2930 { .. } => TxType::Eip2930.into(),
            MockTransaction::Eip7702 { .. } => TxType::Eip7702.into(),
        }
    }
}

impl alloy_consensus::Transaction for MockFbTransaction {
    fn chain_id(&self) -> Option<u64> {
        match &self.inner {
            MockTransaction::Legacy { chain_id, .. } => *chain_id,
            MockTransaction::Eip1559 { chain_id, .. }
            | MockTransaction::Eip4844 { chain_id, .. }
            | MockTransaction::Eip2930 { chain_id, .. }
            | MockTransaction::Eip7702 { chain_id, .. } => Some(*chain_id),
        }
    }

    fn nonce(&self) -> u64 {
        *self.inner.get_nonce()
    }

    fn gas_limit(&self) -> u64 {
        *self.inner.get_gas_limit()
    }

    fn gas_price(&self) -> Option<u128> {
        match &self.inner {
            MockTransaction::Legacy { gas_price, .. }
            | MockTransaction::Eip2930 { gas_price, .. } => Some(*gas_price),
            _ => None,
        }
    }

    fn max_fee_per_gas(&self) -> u128 {
        match &self.inner {
            MockTransaction::Legacy { gas_price, .. }
            | MockTransaction::Eip2930 { gas_price, .. } => *gas_price,
            MockTransaction::Eip1559 {
                max_fee_per_gas, ..
            }
            | MockTransaction::Eip4844 {
                max_fee_per_gas, ..
            }
            | MockTransaction::Eip7702 {
                max_fee_per_gas, ..
            } => *max_fee_per_gas,
        }
    }

    fn max_priority_fee_per_gas(&self) -> Option<u128> {
        match &self.inner {
            MockTransaction::Legacy { .. } | MockTransaction::Eip2930 { .. } => None,
            MockTransaction::Eip1559 {
                max_priority_fee_per_gas,
                ..
            }
            | MockTransaction::Eip4844 {
                max_priority_fee_per_gas,
                ..
            }
            | MockTransaction::Eip7702 {
                max_priority_fee_per_gas,
                ..
            } => Some(*max_priority_fee_per_gas),
        }
    }

    fn max_fee_per_blob_gas(&self) -> Option<u128> {
        match &self.inner {
            MockTransaction::Eip4844 {
                max_fee_per_blob_gas,
                ..
            } => Some(*max_fee_per_blob_gas),
            _ => None,
        }
    }

    fn priority_fee_or_price(&self) -> u128 {
        match &self.inner {
            MockTransaction::Legacy { gas_price, .. }
            | MockTransaction::Eip2930 { gas_price, .. } => *gas_price,
            MockTransaction::Eip1559 {
                max_priority_fee_per_gas,
                ..
            }
            | MockTransaction::Eip4844 {
                max_priority_fee_per_gas,
                ..
            }
            | MockTransaction::Eip7702 {
                max_priority_fee_per_gas,
                ..
            } => *max_priority_fee_per_gas,
        }
    }

    fn effective_gas_price(&self, base_fee: Option<u64>) -> u128 {
        base_fee.map_or_else(
            || self.max_fee_per_gas(),
            |base_fee| {
                // if the tip is greater than the max priority fee per gas, set it to the max
                // priority fee per gas + base fee
                let tip = self.max_fee_per_gas().saturating_sub(base_fee as u128);
                if let Some(max_tip) = self.max_priority_fee_per_gas() {
                    if tip > max_tip {
                        max_tip + base_fee as u128
                    } else {
                        // otherwise return the max fee per gas
                        self.max_fee_per_gas()
                    }
                } else {
                    self.max_fee_per_gas()
                }
            },
        )
    }

    fn is_dynamic_fee(&self) -> bool {
        !matches!(
            self.inner,
            MockTransaction::Legacy { .. } | MockTransaction::Eip2930 { .. }
        )
    }

    fn kind(&self) -> TxKind {
        match &self.inner {
            MockTransaction::Legacy { to, .. }
            | MockTransaction::Eip1559 { to, .. }
            | MockTransaction::Eip2930 { to, .. } => *to,
            MockTransaction::Eip4844 { to, .. } | MockTransaction::Eip7702 { to, .. } => {
                TxKind::Call(*to)
            }
        }
    }

    fn is_create(&self) -> bool {
        match &self.inner {
            MockTransaction::Legacy { to, .. }
            | MockTransaction::Eip1559 { to, .. }
            | MockTransaction::Eip2930 { to, .. } => to.is_create(),
            MockTransaction::Eip4844 { .. } | MockTransaction::Eip7702 { .. } => false,
        }
    }

    fn value(&self) -> U256 {
        match &self.inner {
            MockTransaction::Legacy { value, .. }
            | MockTransaction::Eip1559 { value, .. }
            | MockTransaction::Eip2930 { value, .. }
            | MockTransaction::Eip4844 { value, .. }
            | MockTransaction::Eip7702 { value, .. } => *value,
        }
    }

    fn input(&self) -> &Bytes {
        self.inner.get_input()
    }

    fn access_list(&self) -> Option<&AccessList> {
        match &self.inner {
            MockTransaction::Legacy { .. } => None,
            MockTransaction::Eip1559 {
                access_list: accesslist,
                ..
            }
            | MockTransaction::Eip4844 {
                access_list: accesslist,
                ..
            }
            | MockTransaction::Eip2930 {
                access_list: accesslist,
                ..
            }
            | MockTransaction::Eip7702 {
                access_list: accesslist,
                ..
            } => Some(accesslist),
        }
    }

    fn blob_versioned_hashes(&self) -> Option<&[B256]> {
        match &self.inner {
            MockTransaction::Eip4844 {
                blob_versioned_hashes,
                ..
            } => Some(blob_versioned_hashes),
            _ => None,
        }
    }

    fn authorization_list(&self) -> Option<&[SignedAuthorization]> {
        match &self.inner {
            MockTransaction::Eip7702 {
                authorization_list, ..
            } => Some(authorization_list),
            _ => None,
        }
    }
}

impl EthPoolTransaction for MockFbTransaction {
    fn take_blob(&mut self) -> EthBlobTransactionSidecar {
        match &self.inner {
            MockTransaction::Eip4844 { sidecar, .. } => {
                EthBlobTransactionSidecar::Present(sidecar.clone())
            }
            _ => EthBlobTransactionSidecar::None,
        }
    }

    fn try_into_pooled_eip4844(
        self,
        sidecar: Arc<BlobTransactionSidecarVariant>,
    ) -> Option<Recovered<Self::Pooled>> {
        let (tx, signer) = self.into_consensus().into_parts();
        tx.try_into_pooled_eip4844(Arc::unwrap_or_clone(sidecar))
            .map(|tx| tx.with_signer(signer))
            .ok()
    }

    fn try_from_eip4844(
        tx: Recovered<Self::Consensus>,
        sidecar: BlobTransactionSidecarVariant,
    ) -> Option<Self> {
        let (tx, signer) = tx.into_parts();
        tx.try_into_pooled_eip4844(sidecar)
            .map(|tx| tx.with_signer(signer))
            .ok()
            .map(Self::from_pooled)
    }

    fn validate_blob(
        &self,
        _blob: &BlobTransactionSidecarVariant,
        _settings: &KzgSettings,
    ) -> Result<(), alloy_eips::eip4844::BlobTransactionValidationError> {
        match &self.inner {
            MockTransaction::Eip4844 { .. } => Ok(()),
            _ => Err(BlobTransactionValidationError::NotBlobTransaction(
                self.inner.tx_type(),
            )),
        }
    }
}

pub type PooledTransactionVariant =
    alloy_consensus::EthereumTxEnvelope<TxEip4844WithSidecar<BlobTransactionSidecarVariant>>;

impl MaybeFlashblockFilter for MockFbTransaction {
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
