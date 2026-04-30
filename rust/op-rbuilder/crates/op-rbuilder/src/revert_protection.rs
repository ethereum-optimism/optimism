use std::{sync::Arc, time::Instant};

use crate::{
    metrics::OpRBuilderMetrics,
    primitives::bundle::{Bundle, BundleResult},
    tx::{FBPooledTransaction, MaybeFlashblockFilter, MaybeRevertingTransaction},
};
use alloy_json_rpc::RpcObject;
use alloy_primitives::B256;
use jsonrpsee::{
    core::{RpcResult, async_trait},
    proc_macros::rpc,
};
use moka::future::Cache;
use reth::rpc::api::eth::{RpcReceipt, helpers::FullEthApi};
use reth_optimism_txpool::{OpPooledTransaction, conditional::MaybeConditionalTransaction};
use reth_provider::StateProviderFactory;
use reth_rpc_eth_types::{EthApiError, utils::recover_raw_transaction};
use reth_transaction_pool::{PoolTransaction, TransactionOrigin, TransactionPool};
use tracing::error;

// Namespace overrides for revert protection support
#[cfg_attr(not(test), rpc(server, namespace = "eth"))]
#[cfg_attr(test, rpc(server, client, namespace = "eth"))]
pub trait EthApiExt<R: RpcObject> {
    #[method(name = "sendBundle")]
    async fn send_bundle(&self, tx: Bundle) -> RpcResult<BundleResult>;

    #[method(name = "getTransactionReceipt")]
    async fn transaction_receipt(&self, hash: B256) -> RpcResult<Option<R>>;
}

pub struct RevertProtectionExt<Pool, Provider, Eth> {
    pool: Pool,
    provider: Provider,
    eth_api: Eth,
    metrics: Arc<OpRBuilderMetrics>,
    reverted_cache: Cache<B256, ()>,
}

impl<Pool, Provider, Eth> RevertProtectionExt<Pool, Provider, Eth>
where
    Pool: Clone,
    Provider: Clone,
    Eth: Clone,
{
    pub fn new(
        pool: Pool,
        provider: Provider,
        eth_api: Eth,
        reverted_cache: Cache<B256, ()>,
    ) -> Self {
        Self {
            pool,
            provider,
            eth_api,
            metrics: Arc::new(OpRBuilderMetrics::default()),
            reverted_cache,
        }
    }
}

#[async_trait]
impl<Pool, Provider, Eth> EthApiExtServer<RpcReceipt<Eth::NetworkTypes>>
    for RevertProtectionExt<Pool, Provider, Eth>
where
    Pool: TransactionPool<Transaction = FBPooledTransaction> + Clone + 'static,
    Provider: StateProviderFactory + Send + Sync + Clone + 'static,
    Eth: FullEthApi + Send + Sync + Clone + 'static,
{
    async fn send_bundle(&self, bundle: Bundle) -> RpcResult<BundleResult> {
        let request_start_time = Instant::now();
        self.metrics.bundle_requests.increment(1);

        let bundle_result = self
            .send_bundle_inner(bundle)
            .await
            .inspect_err(|err| error!("eth_sendBundle request failed: {err:?}"));

        if bundle_result.is_ok() {
            self.metrics.valid_bundles.increment(1);
        } else {
            self.metrics.failed_bundles.increment(1);
        }

        self.metrics
            .bundle_receive_duration
            .record(request_start_time.elapsed());

        bundle_result
    }

    async fn transaction_receipt(
        &self,
        hash: B256,
    ) -> RpcResult<Option<RpcReceipt<Eth::NetworkTypes>>> {
        if let Some(receipt) = self.eth_api.transaction_receipt(hash).await.unwrap() {
            Ok(Some(receipt))
        } else if self.reverted_cache.get(&hash).await.is_some() {
            // Found the transaction in the reverted cache
            Err(
                EthApiError::InvalidParams("the transaction was dropped from the pool".into())
                    .into(),
            )
        } else {
            Ok(None)
        }
    }
}

impl<Pool, Provider, Eth> RevertProtectionExt<Pool, Provider, Eth>
where
    Pool: TransactionPool<Transaction = FBPooledTransaction> + Clone + 'static,
    Provider: StateProviderFactory + Send + Sync + Clone + 'static,
    Eth: FullEthApi + Send + Sync + Clone + 'static,
{
    async fn send_bundle_inner(&self, bundle: Bundle) -> RpcResult<BundleResult> {
        let last_block_number = self
            .provider
            .best_block_number()
            .map_err(|_e| EthApiError::InternalEthError)?;

        // Only one transaction in the bundle is expected
        let bundle_transaction = match bundle.transactions.len() {
            0 => {
                return Err(EthApiError::InvalidParams(
                    "bundle must contain at least one transaction".into(),
                )
                .into());
            }
            1 => bundle.transactions[0].clone(),
            _ => {
                return Err(EthApiError::InvalidParams(
                    "bundle must contain exactly one transaction".into(),
                )
                .into());
            }
        };

        let conditional = bundle
            .conditional(last_block_number)
            .map_err(EthApiError::from)?;

        let recovered = recover_raw_transaction(&bundle_transaction)?;
        let pool_transaction =
            FBPooledTransaction::from(OpPooledTransaction::from_pooled(recovered))
                .with_reverted_hashes(bundle.reverting_hashes.clone().unwrap_or_default())
                .with_flashblock_number_min(conditional.flashblock_number_min)
                .with_flashblock_number_max(conditional.flashblock_number_max)
                .with_conditional(conditional.transaction_conditional);

        let outcome = self
            .pool
            .add_transaction(TransactionOrigin::Local, pool_transaction)
            .await
            .map_err(EthApiError::from)?;

        let result = BundleResult {
            bundle_hash: outcome.hash,
        };
        Ok(result)
    }
}
