//! Eth API extension.

use crate::{
    OpEthApiError, SequencerClient,
    error::TxConditionalErr,
    eth::bundle::{Bundle, BundleExpiry, BundleResult, OpEthBundleApiServer, PendingBundles},
};
use alloy_consensus::{BlockHeader, transaction::TxHashRef};
use alloy_eips::BlockNumberOrTag;
use alloy_primitives::{B256, Bytes, StorageKey, U256};
use alloy_rpc_types_eth::erc4337::{AccountStorage, TransactionConditional};
use jsonrpsee_core::RpcResult;
use reth_optimism_txpool::{
    conditional::MaybeConditionalTransaction, revert_protection::MaybeRevertProtectedTransaction,
};
use reth_primitives_traits::{Recovered, SealedHeader};
use reth_rpc_eth_api::L2EthApiExtServer;
use reth_rpc_eth_types::{EthApiError, utils::recover_raw_transaction};
use reth_storage_api::{BlockReaderIdExt, StateProviderFactory};
use reth_transaction_pool::{
    AddedTransactionOutcome, PoolTransaction, TransactionOrigin, TransactionPool,
};
use std::sync::Arc;
use tokio::sync::Semaphore;

/// Maximum execution const for conditional transactions.
const MAX_CONDITIONAL_EXECUTION_COST: u64 = 5000;

const MAX_CONCURRENT_CONDITIONAL_VALIDATIONS: usize = 3;

/// OP-Reth `Eth` API extensions implementation: `eth_sendRawTransactionConditional` and
/// `eth_sendBundle`.
///
/// Separate from [`super::OpEthApi`] to allow to enable it conditionally,
#[derive(Clone, Debug)]
pub struct OpEthExtApi<Pool, Provider> {
    /// Sequencer client, configured to forward submitted transactions to sequencer of given OP
    /// network.
    sequencer_client: Option<SequencerClient>,
    /// Expiries of the bundles submitted through `eth_sendBundle`, see [`PendingBundles`].
    pending_bundles: PendingBundles,
    inner: Arc<OpEthExtApiInner<Pool, Provider>>,
}

impl<Pool, Provider> OpEthExtApi<Pool, Provider>
where
    Provider: BlockReaderIdExt + StateProviderFactory + Clone + 'static,
{
    /// Creates a new [`OpEthExtApi`].
    pub fn new(sequencer_client: Option<SequencerClient>, pool: Pool, provider: Provider) -> Self {
        let inner = Arc::new(OpEthExtApiInner::new(pool, provider));
        Self { sequencer_client, pending_bundles: PendingBundles::default(), inner }
    }

    /// Shares the submitted bundles' expiries with the `eth` API, so its
    /// `eth_getTransactionReceipt` can report an expired bundle. See
    /// [`super::OpEthApiBuilder::with_pending_bundles`].
    pub fn with_pending_bundles(mut self, pending_bundles: PendingBundles) -> Self {
        self.pending_bundles = pending_bundles;
        self
    }

    /// Returns the submitted bundles' expiries.
    pub const fn pending_bundles(&self) -> &PendingBundles {
        &self.pending_bundles
    }

    /// Returns the latest header, or a header-not-found error if the chain has none.
    fn latest_header(&self) -> Result<SealedHeader<Provider::Header>, OpEthApiError> {
        let header_not_found = || {
            OpEthApiError::Eth(EthApiError::HeaderNotFound(alloy_eips::BlockId::Number(
                BlockNumberOrTag::Latest,
            )))
        };
        self.provider()
            .latest_header()
            .map_err(|_| header_not_found())?
            .ok_or_else(header_not_found)
    }

    /// Returns the configured sequencer client, if any.
    const fn sequencer_client(&self) -> Option<&SequencerClient> {
        self.sequencer_client.as_ref()
    }

    #[inline]
    fn pool(&self) -> &Pool {
        self.inner.pool()
    }

    #[inline]
    fn provider(&self) -> &Provider {
        self.inner.provider()
    }

    /// Validates the conditional's `known accounts` settings against the current state.
    async fn validate_known_accounts(
        &self,
        condition: &TransactionConditional,
    ) -> Result<(), TxConditionalErr> {
        if condition.known_accounts.is_empty() {
            return Ok(());
        }

        let _permit =
            self.inner.validation_semaphore.acquire().await.map_err(TxConditionalErr::internal)?;

        let state = self
            .provider()
            .state_by_block_number_or_tag(BlockNumberOrTag::Latest)
            .map_err(TxConditionalErr::internal)?;

        for (address, storage) in &condition.known_accounts {
            match storage {
                AccountStorage::Slots(slots) => {
                    for (slot, expected_value) in slots {
                        let current = state
                            .storage(*address, StorageKey::from(*slot))
                            .map_err(TxConditionalErr::internal)?
                            .unwrap_or_default();

                        if current != U256::from_be_bytes(**expected_value) {
                            return Err(TxConditionalErr::StorageValueMismatch);
                        }
                    }
                }
                AccountStorage::RootHash(expected_root) => {
                    let actual_root = state
                        .storage_root(*address, Default::default())
                        .map_err(TxConditionalErr::internal)?;

                    if *expected_root != actual_root {
                        return Err(TxConditionalErr::StorageRootMismatch);
                    }
                }
            }
        }

        Ok(())
    }
}

#[async_trait::async_trait]
impl<Pool, Provider> L2EthApiExtServer for OpEthExtApi<Pool, Provider>
where
    Provider: BlockReaderIdExt + StateProviderFactory + Clone + 'static,
    Pool: TransactionPool<Transaction: MaybeConditionalTransaction> + 'static,
{
    async fn send_raw_transaction_conditional(
        &self,
        bytes: Bytes,
        condition: TransactionConditional,
    ) -> RpcResult<B256> {
        // calculate and validate cost
        let cost = condition.cost();
        if cost > MAX_CONDITIONAL_EXECUTION_COST {
            return Err(TxConditionalErr::ConditionalCostExceeded.into());
        }

        let recovered_tx = recover_raw_transaction(&bytes).map_err(|_| {
            OpEthApiError::Eth(reth_rpc_eth_types::EthApiError::FailedToDecodeSignedTransaction)
        })?;

        let mut tx = <Pool as TransactionPool>::Transaction::from_pooled(recovered_tx);

        // get current header
        let header = self.latest_header()?;

        // Ensure that the condition can still be met by checking the max bounds
        if condition.has_exceeded_block_number(header.number()) ||
            condition.has_exceeded_timestamp(header.timestamp())
        {
            return Err(TxConditionalErr::InvalidCondition.into());
        }

        // Validate Account
        self.validate_known_accounts(&condition).await?;

        if let Some(sequencer) = self.sequencer_client() {
            // If we have a sequencer client, forward the transaction
            let _ = sequencer
                .forward_raw_transaction_conditional(bytes.as_ref(), condition)
                .await
                .map_err(OpEthApiError::Sequencer)?;
            Ok(*tx.hash())
        } else {
            // otherwise, add to pool with the appended conditional
            tx.set_conditional(condition);
            let AddedTransactionOutcome { hash, .. } =
                self.pool().add_transaction(TransactionOrigin::Private, tx).await.map_err(|e| {
                    OpEthApiError::Eth(reth_rpc_eth_types::EthApiError::PoolError(e.into()))
                })?;

            Ok(hash)
        }
    }
}

#[async_trait::async_trait]
impl<Pool, Provider> OpEthBundleApiServer for OpEthExtApi<Pool, Provider>
where
    Provider: BlockReaderIdExt + StateProviderFactory + Clone + 'static,
    Pool: TransactionPool<Transaction: MaybeConditionalTransaction + MaybeRevertProtectedTransaction>
        + 'static,
{
    async fn send_bundle(&self, bundle: Bundle) -> RpcResult<BundleResult> {
        let raw_tx = bundle.single_tx()?;
        let recovered_tx: Recovered<<Pool::Transaction as PoolTransaction>::Pooled> =
            recover_raw_transaction(raw_tx)
                .map_err(|_| OpEthApiError::Eth(EthApiError::FailedToDecodeSignedTransaction))?;
        let bundle_hash = *recovered_tx.tx_hash();

        // Validate the execution window against the chain head and turn it into the conditional
        // the builder checks and the pool maintenance expires.
        let header = self.latest_header()?;
        let conditional = bundle.conditional(header.number(), header.timestamp())?;
        let expiry = BundleExpiry::from(&conditional);

        if let Some(sequencer) = self.sequencer_client() {
            // Only the sequencer builds blocks, so it is the one to hold the bundle.
            let result =
                sequencer.forward_bundle(&bundle).await.map_err(OpEthApiError::Sequencer)?;
            // Remembered here too, so this node's `eth_getTransactionReceipt` can report expiry.
            self.pending_bundles().insert(result.bundle_hash, expiry);
            return Ok(result);
        }

        let tx = <Pool as TransactionPool>::Transaction::from_pooled(recovered_tx)
            .with_conditional(conditional)
            .with_revert_protected(bundle.is_revert_protected(&bundle_hash));
        self.pool()
            .add_transaction(TransactionOrigin::Private, tx)
            .await
            .map_err(|e| OpEthApiError::Eth(EthApiError::PoolError(e.into())))?;
        // Recorded after the pool accepted it: a resubmission replaces the old window.
        self.pending_bundles().insert(bundle_hash, expiry);

        Ok(BundleResult { bundle_hash })
    }
}

#[derive(Debug)]
struct OpEthExtApiInner<Pool, Provider> {
    /// The transaction pool of the node.
    pool: Pool,
    /// The provider type used to interact with the node.
    provider: Provider,
    /// The semaphore used to limit the number of concurrent conditional validations.
    validation_semaphore: Semaphore,
}

impl<Pool, Provider> OpEthExtApiInner<Pool, Provider> {
    fn new(pool: Pool, provider: Provider) -> Self {
        Self {
            pool,
            provider,
            validation_semaphore: Semaphore::new(MAX_CONCURRENT_CONDITIONAL_VALIDATIONS),
        }
    }

    #[inline]
    const fn pool(&self) -> &Pool {
        &self.pool
    }

    #[inline]
    const fn provider(&self) -> &Provider {
        &self.provider
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::eth::bundle::{BundleError, MAX_BLOCK_RANGE_BLOCKS};
    use alloy_consensus::{Header, SignableTransaction, TxEip1559};
    use alloy_eips::eip2718::Encodable2718;
    use alloy_primitives::{Address, TxHash, TxKind, U256};
    use alloy_signer::SignerSync;
    use alloy_signer_local::PrivateKeySigner;
    use jsonrpsee_types::error::INVALID_PARAMS_CODE;
    use reth_optimism_txpool::OpPooledTransaction;
    use reth_provider::test_utils::MockEthProvider;
    use reth_transaction_pool::{
        CoinbaseTipOrdering, Pool, PoolConfig, blobstore::InMemoryBlobStore,
        noop::MockTransactionValidator,
    };

    const LATEST_BLOCK: u64 = 100;
    const LATEST_TIMESTAMP: u64 = 1_700_000_000;

    /// A pool of [`OpPooledTransaction`] backed by the always-valid mock validator, so a bundle
    /// lands in the pool with the metadata `send_bundle` attached.
    type TestPool = Pool<
        MockTransactionValidator<OpPooledTransaction>,
        CoinbaseTipOrdering<OpPooledTransaction>,
        InMemoryBlobStore,
    >;

    fn ext_api() -> OpEthExtApi<TestPool, MockEthProvider> {
        let provider = MockEthProvider::default();
        // `latest_header` resolves through the mock's block store, so add a (bodiless) block.
        provider.add_block(
            B256::with_last_byte(1),
            alloy_consensus::Block {
                header: Header {
                    number: LATEST_BLOCK,
                    timestamp: LATEST_TIMESTAMP,
                    ..Default::default()
                },
                body: Default::default(),
            },
        );
        let pool = Pool::new(
            MockTransactionValidator::default(),
            CoinbaseTipOrdering::default(),
            InMemoryBlobStore::default(),
            PoolConfig::default(),
        );
        OpEthExtApi::new(None, pool, provider)
    }

    /// A signed, EIP-2718 encoded EIP-1559 transfer.
    struct RawTx {
        bytes: Bytes,
        hash: TxHash,
    }

    fn raw_tx(signer: &PrivateKeySigner, nonce: u64) -> RawTx {
        let tx = TxEip1559 {
            chain_id: 10,
            nonce,
            gas_limit: 21_000,
            max_fee_per_gas: 1_000,
            max_priority_fee_per_gas: 1,
            to: TxKind::Call(Address::repeat_byte(0x22)),
            value: U256::from(1),
            ..Default::default()
        };
        let signature = signer.sign_hash_sync(&tx.signature_hash()).unwrap();
        let signed = tx.into_signed(signature);
        let hash = *signed.hash();
        let bytes = op_alloy_consensus::OpPooledTransaction::Eip1559(signed).encoded_2718().into();
        RawTx { bytes, hash }
    }

    fn bundle(raw: &RawTx) -> Bundle {
        Bundle { txs: vec![raw.bytes.clone()], ..Default::default() }
    }

    #[tokio::test]
    async fn send_bundle_adds_revert_protected_conditional_tx() {
        let api = ext_api();
        let raw = raw_tx(&PrivateKeySigner::random(), 0);

        let result = api
            .send_bundle(Bundle { max_block_number: Some(LATEST_BLOCK + 5), ..bundle(&raw) })
            .await
            .unwrap();
        assert_eq!(result.bundle_hash, raw.hash);

        let pooled = api.pool().get(&raw.hash).expect("bundle tx is in the pool");
        assert_eq!(pooled.origin, TransactionOrigin::Private);
        assert!(pooled.transaction.is_revert_protected());
        let conditional = pooled.transaction.conditional().expect("conditional attached");
        assert_eq!(conditional.block_number_min, None);
        assert_eq!(conditional.block_number_max, Some(LATEST_BLOCK + 5));

        // The window is remembered for `eth_getTransactionReceipt`.
        assert!(!api.pending_bundles().is_expired(&raw.hash, LATEST_BLOCK + 4, 0));
        assert!(api.pending_bundles().is_expired(&raw.hash, LATEST_BLOCK + 5, 0));
    }

    #[tokio::test]
    async fn send_bundle_defaults_expiry_and_honours_reverting_tx_hashes() {
        let api = ext_api();
        let raw = raw_tx(&PrivateKeySigner::random(), 0);

        let result = api
            .send_bundle(Bundle { reverting_tx_hashes: Some(vec![raw.hash]), ..bundle(&raw) })
            .await
            .unwrap();
        assert_eq!(result.bundle_hash, raw.hash);

        let pooled = api.pool().get(&raw.hash).expect("bundle tx is in the pool");
        assert!(!pooled.transaction.is_revert_protected());
        let conditional = pooled.transaction.conditional().expect("conditional attached");
        assert_eq!(conditional.block_number_max, Some(LATEST_BLOCK + MAX_BLOCK_RANGE_BLOCKS));
        assert!(api.pending_bundles().is_expired(
            &raw.hash,
            LATEST_BLOCK + MAX_BLOCK_RANGE_BLOCKS,
            0
        ));
    }

    #[tokio::test]
    async fn send_bundle_rejects_invalid_bundles_before_the_pool() {
        let api = ext_api();
        let raw = raw_tx(&PrivateKeySigner::random(), 0);

        let err = api.send_bundle(Bundle::default()).await.unwrap_err();
        assert_eq!(err.code(), INVALID_PARAMS_CODE);
        assert_eq!(err.message(), BundleError::NotExactlyOneTx { count: 0 }.to_string());

        let err = api
            .send_bundle(Bundle { max_block_number: Some(LATEST_BLOCK), ..bundle(&raw) })
            .await
            .unwrap_err();
        assert_eq!(err.code(), INVALID_PARAMS_CODE);
        assert_eq!(
            err.message(),
            BundleError::MaxBlockInPast { max: LATEST_BLOCK, current: LATEST_BLOCK }.to_string()
        );

        let err = api
            .send_bundle(Bundle {
                max_block_number: Some(LATEST_BLOCK + 1),
                max_timestamp: Some(LATEST_TIMESTAMP + 1),
                ..bundle(&raw)
            })
            .await
            .unwrap_err();
        assert_eq!(err.message(), BundleError::RangesMutuallyExclusive.to_string());

        let garbage = Bundle { txs: vec![Bytes::from_static(&[0x02, 0x00])], ..Default::default() };
        let err = api.send_bundle(garbage).await.unwrap_err();
        assert_eq!(err.code(), INVALID_PARAMS_CODE);

        assert!(api.pool().get(&raw.hash).is_none(), "rejected bundles never reach the pool");
        assert!(api.pending_bundles().get(&raw.hash).is_none(), "nor are they remembered");
    }

    #[tokio::test]
    async fn resubmitting_a_bundle_replaces_its_window_once_the_old_copy_is_gone() {
        let api = ext_api();
        let raw = raw_tx(&PrivateKeySigner::random(), 0);
        let first = Bundle { max_block_number: Some(LATEST_BLOCK + 2), ..bundle(&raw) };
        let second = Bundle { max_block_number: Some(LATEST_BLOCK + 8), ..bundle(&raw) };

        api.send_bundle(first).await.unwrap();
        assert!(api.pending_bundles().is_expired(&raw.hash, LATEST_BLOCK + 2, 0));

        // While the first copy is pending the pool rejects the duplicate, and the remembered
        // window stays the one the pool holds.
        let err = api.send_bundle(second.clone()).await.unwrap_err();
        assert!(err.message().contains("already known"), "{err:?}");
        assert!(api.pending_bundles().is_expired(&raw.hash, LATEST_BLOCK + 2, 0));

        // Once it is gone (expired and evicted), a resubmission starts a new window.
        api.pool().remove_transactions(vec![raw.hash]);
        api.send_bundle(second).await.unwrap();
        assert!(!api.pending_bundles().is_expired(&raw.hash, LATEST_BLOCK + 2, 0));
        assert!(api.pending_bundles().is_expired(&raw.hash, LATEST_BLOCK + 8, 0));
    }
}
