//! Loads and formats OP transaction RPC response.

use crate::{OpEthApi, OpEthApiError, SequencerClient};
use alloy_primitives::{B256, Bytes};
use alloy_rpc_types_eth::TransactionInfo;
use futures::StreamExt;
use op_alloy_consensus::{
    OpTransaction,
    transaction::{OpDepositInfo, OpTransactionInfo},
};
use reth_chain_state::CanonStateSubscriptions;
use reth_optimism_primitives::DepositReceipt;
use reth_primitives_traits::{SignedTransaction, WithEncoded};
use reth_rpc_eth_api::{
    EthApiTypes as _, FromEthApiError, FromEvmError, RpcConvert, RpcNodeCore, RpcReceipt,
    TxInfoMapper,
    helpers::{EthTransactions, LoadReceipt, LoadTransaction, spec::SignersForRpc},
};
use reth_rpc_eth_types::{EthApiError, TransactionSource, block::convert_transaction_receipt};
use reth_storage_api::{ProviderTx, ReceiptProvider, errors::ProviderError};
use reth_transaction_pool::{
    AddedTransactionOutcome, PoolTransaction, PoolTx, TransactionOrigin, TransactionPool,
};
use std::{
    fmt::{Debug, Formatter},
    future::Future,
    time::Duration,
};
use tokio_stream::wrappers::WatchStream;

impl<N, Rpc> EthTransactions for OpEthApi<N, Rpc>
where
    N: RpcNodeCore,
    OpEthApiError: FromEvmError<N::Evm>,
    Rpc: RpcConvert<Primitives = N::Primitives, Error = OpEthApiError>,
{
    fn signers(&self) -> &SignersForRpc<Self::Provider, Self::NetworkTypes> {
        self.inner.eth_api.signers()
    }

    fn send_raw_transaction_sync_timeout(&self) -> Duration {
        self.inner.eth_api.send_raw_transaction_sync_timeout()
    }

    async fn send_pool_transaction(
        &self,
        origin: TransactionOrigin,
        tx: WithEncoded<PoolTx<Self::Pool>>,
    ) -> Result<B256, Self::Error> {
        let (tx, pool_transaction) = tx.split();

        // broadcast raw transaction to subscribers if there is any.
        self.eth_api().broadcast_raw_transaction(tx.clone());

        // On optimism, transactions are forwarded directly to the sequencer to be included in
        // blocks that it builds.
        if let Some(client) = self.raw_tx_forwarder().as_ref() {
            tracing::debug!(target: "rpc::eth", hash = %pool_transaction.hash(), "forwarding raw transaction to sequencer");
            let hash = client.forward_raw_transaction(&tx).await.inspect_err(|err| {
                    tracing::debug!(target: "rpc::eth", %err, hash=% *pool_transaction.hash(), "failed to forward raw transaction");
                })?;

            if self.inner.retain_forwarded_txs() {
                let _ = self.inner.eth_api.add_pool_transaction(origin, pool_transaction).await.inspect_err(|err| {
                    tracing::warn!(target: "rpc::eth", %err, %hash, "successfully sent tx to sequencer, but failed to persist in local tx pool");
                });
            }

            return Ok(hash);
        }

        // submit the transaction to the pool with the given origin
        let AddedTransactionOutcome { hash, .. } = self
            .pool()
            .add_transaction(origin, pool_transaction)
            .await
            .map_err(Self::Error::from_eth_err)?;

        Ok(hash)
    }

    /// UPSTREAM-MIRROR(override): reth@rev:8d359d0
    /// `reth_rpc_eth_api::helpers::EthTransactions::send_raw_transaction_sync`
    ///
    /// Derived from the upstream default: the timeout computation is taken verbatim and the
    /// wait loop is extended to also resolve the receipt from a flashblock. Re-derive on any
    /// change to the default body.
    ///
    /// Decodes and recovers the transaction and submits it to the pool.
    ///
    /// And awaits the receipt, checking both canonical blocks and flashblocks for faster
    /// confirmation.
    fn send_raw_transaction_sync(
        &self,
        tx: Bytes,
        timeout_ms: Option<u64>,
    ) -> impl Future<Output = Result<RpcReceipt<Self::NetworkTypes>, Self::Error>> + Send {
        let this = self.clone();
        let configured_timeout = self.send_raw_transaction_sync_timeout();
        let timeout_duration = timeout_ms
            .filter(|timeout_ms| *timeout_ms > 0)
            .map(Duration::from_millis)
            .map(|timeout| timeout.min(configured_timeout))
            .unwrap_or(configured_timeout);
        async move {
            let mut canonical_stream = this.provider().canonical_state_stream();
            let hash = EthTransactions::send_raw_transaction(&this, tx).await?;
            let mut flashblock_stream = this.pending_block_rx().map(WatchStream::new);

            tokio::time::timeout(timeout_duration, async {
                loop {
                    tokio::select! {
                        biased;
                        // check if the tx was preconfirmed in a new flashblock
                        flashblock = async {
                            if let Some(stream) = &mut flashblock_stream {
                                stream.next().await
                            } else {
                                futures::future::pending().await
                            }
                        } => {
                            if let Some(flashblock) = flashblock.flatten() {
                                // if flashblocks are supported, attempt to find id from the pending block
                                if let Some(receipt) = flashblock
                                .find_and_convert_transaction_receipt(hash, this.converter())
                                {
                                    return receipt;
                                }
                            }
                        }
                        // Listen for regular canonical block updates for inclusion
                        canonical_notification = canonical_stream.next() => {
                            if let Some(notification) = canonical_notification {
                                let chain = notification.committed();
                                if let Some((block, tx, receipt, all_receipts)) =
                                    chain.find_transaction_and_receipt_by_hash(hash) &&
                                    let Some(receipt) = convert_transaction_receipt(
                                        block,
                                        all_receipts,
                                        tx,
                                        receipt,
                                        this.converter(),
                                    )
                                    .transpose()?
                                {
                                    return Ok(receipt);
                                }
                            } else {
                                // Canonical stream ended
                                break;
                            }
                        }
                    }
                }
                Err(Self::Error::from_eth_err(EthApiError::TransactionConfirmationTimeout {
                    hash,
                    duration: timeout_duration,
                }))
            })
            .await
            .unwrap_or_else(|_elapsed| {
                Err(Self::Error::from_eth_err(EthApiError::TransactionConfirmationTimeout {
                    hash,
                    duration: timeout_duration,
                }))
            })
        }
    }

    /// Returns the transaction receipt for the given hash.
    ///
    /// With flashblocks, we should also lookup the pending block for the transaction
    /// because this is considered confirmed/mined.
    /// UPSTREAM-MIRROR(override): reth@rev:8d359d0
    /// `reth_rpc_eth_api::helpers::EthTransactions::transaction_receipt`
    ///
    /// Extends the upstream default with a flashblock receipt lookup.
    fn transaction_receipt(
        &self,
        hash: B256,
    ) -> impl Future<Output = Result<Option<RpcReceipt<Self::NetworkTypes>>, Self::Error>> + Send
    {
        let this = self.clone();
        async move {
            // first attempt to fetch the mined transaction receipt data
            let tx_receipt = this.load_transaction_and_receipt(hash).await?;

            if tx_receipt.is_none() {
                // if flashblocks are supported, attempt to find id from the pending block
                if let Ok(Some(pending_block)) = this.pending_flashblock().await &&
                    let Some(Ok(receipt)) = pending_block
                        .find_and_convert_transaction_receipt(hash, this.converter())
                {
                    return Ok(Some(receipt));
                }
            }
            let Some((tx, meta, receipt, all_receipts, block)) = tx_receipt else {
                return Ok(None);
            };
            self.build_transaction_receipt(tx, meta, receipt, all_receipts, block).await.map(Some)
        }
    }
}

impl<N, Rpc> LoadTransaction for OpEthApi<N, Rpc>
where
    N: RpcNodeCore,
    OpEthApiError: FromEvmError<N::Evm>,
    Rpc: RpcConvert<Primitives = N::Primitives, Error = OpEthApiError>,
{
    // Keep the cache, historical lookup, recovery, and pool fallback in reth's shared pipeline.
    // Flashblocks take precedence over the pool, while mined transactions take precedence over
    // Flashblocks. Ignore Flashblocks lookup failures as for the other pending-block RPC paths.
    async fn transaction_by_hash_from_pending(
        &self,
        hash: B256,
    ) -> Result<Option<TransactionSource<ProviderTx<Self::Provider>>>, Self::Error> {
        if let Ok(Some(pending_block)) = self.pending_flashblock().await &&
            let Some(indexed_tx) = pending_block.block().find_indexed(hash)
        {
            let meta = indexed_tx.meta();
            return Ok(Some(TransactionSource::Block {
                transaction: indexed_tx.recovered_tx().cloned(),
                index: meta.index,
                block_hash: meta.block_hash,
                block_number: meta.block_number,
                block_timestamp: meta.timestamp,
                base_fee: meta.base_fee,
            }));
        }

        Ok(None)
    }
}

impl<N, Rpc> OpEthApi<N, Rpc>
where
    N: RpcNodeCore,
    Rpc: RpcConvert<Primitives = N::Primitives>,
{
    /// Returns the [`SequencerClient`] if one is set.
    pub fn raw_tx_forwarder(&self) -> Option<SequencerClient> {
        self.inner.sequencer_client.clone()
    }
}

/// Optimism implementation of [`TxInfoMapper`].
///
/// For deposits, receipt is fetched to extract `deposit_nonce` and `deposit_receipt_version`.
/// Otherwise, it works like regular Ethereum implementation, i.e. uses [`TransactionInfo`].
pub struct OpTxInfoMapper<Provider> {
    provider: Provider,
}

impl<Provider: Clone> Clone for OpTxInfoMapper<Provider> {
    fn clone(&self) -> Self {
        Self { provider: self.provider.clone() }
    }
}

impl<Provider> Debug for OpTxInfoMapper<Provider> {
    fn fmt(&self, f: &mut Formatter<'_>) -> std::fmt::Result {
        f.debug_struct("OpTxInfoMapper").finish()
    }
}

impl<Provider> OpTxInfoMapper<Provider> {
    /// Creates [`OpTxInfoMapper`] that uses [`ReceiptProvider`] borrowed from given `eth_api`.
    pub const fn new(provider: Provider) -> Self {
        Self { provider }
    }
}

impl<T, Provider> TxInfoMapper<T> for OpTxInfoMapper<Provider>
where
    T: OpTransaction + SignedTransaction,
    Provider: ReceiptProvider<Receipt: DepositReceipt>,
{
    type Out = OpTransactionInfo;
    type Err = ProviderError;

    fn try_map(&self, tx: &T, tx_info: TransactionInfo) -> Result<Self::Out, ProviderError> {
        let deposit_meta = if tx.is_deposit() {
            self.provider.receipt_by_hash(*tx.tx_hash())?.and_then(|receipt| {
                receipt.as_deposit_receipt().map(|receipt| OpDepositInfo {
                    deposit_receipt_version: receipt.deposit_receipt_version,
                    deposit_nonce: receipt.deposit_nonce,
                })
            })
        } else {
            None
        }
        .unwrap_or_default();

        Ok(OpTransactionInfo::new(tx_info, deposit_meta))
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::eth::receipt::OpReceiptConverter;
    use alloy_consensus::{Block, BlockBody, Header};
    use alloy_primitives::{Address, Signature, U256};
    use op_alloy_consensus::{OpTypedTransaction, TxDeposit};
    use op_alloy_network::Optimism;
    use reth_chain_state::{CanonStateNotification, ExecutedBlock};
    use reth_network_api::noop::NoopNetwork;
    use reth_optimism_chainspec::OP_MAINNET;
    use reth_optimism_evm::{OpEvmConfig, tx::OpTxEnvConverter};
    use reth_optimism_flashblocks::{FlashblocksListeners, PendingFlashBlock};
    use reth_optimism_primitives::{OpPrimitives, OpTransactionSigned};
    use reth_optimism_txpool::OpPooledTransaction;
    use reth_primitives_traits::RecoveredBlock;
    use reth_provider::{Chain, ExecutionOutcome, test_utils::MockEthProvider};
    use reth_rpc::eth::EthApiBuilder;
    use reth_rpc_eth_api::{RpcConverter, RpcNodeCoreExt};
    use reth_rpc_eth_types::{PendingBlock, cache::cache_new_blocks_task};
    use reth_transaction_pool::noop::NoopTransactionPool;
    use std::{
        sync::Arc,
        time::{Duration, Instant},
    };
    use tokio::sync::{broadcast, watch};

    #[tokio::test]
    async fn flashblocks_transaction_lookup_preserves_metadata_and_handles_missing_blocks() {
        let provider =
            MockEthProvider::<OpPrimitives>::new().with_chain_spec(OP_MAINNET.as_ref().clone());
        let converter = RpcConverter::<Optimism, OpEvmConfig, _>::new(OpReceiptConverter::new(
            provider.clone(),
        ))
        .with_mapper(OpTxInfoMapper::new(provider.clone()))
        .with_tx_env_converter(OpTxEnvConverter);
        let backend = EthApiBuilder::new(
            provider,
            NoopTransactionPool::<OpPooledTransaction>::new(),
            NoopNetwork::default(),
            OpEvmConfig::optimism(OP_MAINNET.clone()),
        )
        .with_rpc_converter(converter)
        .build_inner();
        let transaction = OpTransactionSigned::new_unhashed(
            OpTypedTransaction::Deposit(TxDeposit {
                from: Address::repeat_byte(1),
                gas_limit: 21_000,
                ..Default::default()
            }),
            Signature::new(U256::ZERO, U256::ZERO, false),
        );
        let hash = transaction.tx_hash();
        let block = RecoveredBlock::new_unhashed(
            Block::new(
                Header {
                    number: 42,
                    timestamp: 43,
                    base_fee_per_gas: Some(7),
                    ..Default::default()
                },
                BlockBody { transactions: vec![transaction.clone()], ..Default::default() },
            ),
            vec![Address::repeat_byte(1)],
        );
        let block_hash = block.hash();
        let mut canonical_block = block.clone().into_block();
        canonical_block.header.number = 41;
        let canonical = Arc::new(Chain::<OpPrimitives>::new(
            [RecoveredBlock::new_unhashed(canonical_block, vec![Address::repeat_byte(1)])],
            ExecutionOutcome::new(Default::default(), vec![vec![]], 41, vec![]),
            Default::default(),
        ));
        let pending = PendingFlashBlock::new(
            PendingBlock::with_executed_block(
                Instant::now() + Duration::from_secs(1),
                ExecutedBlock::new(Arc::new(block), Default::default(), Default::default()),
            ),
            B256::ZERO,
            0,
            block_hash,
            false,
        );
        let (pending_tx, pending_rx) = watch::channel(Some(pending));
        let (_in_progress_tx, in_progress_rx) = watch::channel(None);
        let listeners = FlashblocksListeners::new(
            pending_rx,
            broadcast::channel(1).0,
            in_progress_rx,
            broadcast::channel(1).0,
        );
        let api = OpEthApi::new(backend, None, U256::ZERO, Some(listeners), false);
        // Exercise the inherited pipeline, including the custom Flashblocks hook.
        let result = LoadTransaction::transaction_by_hash(&api, hash).await.unwrap().unwrap();
        assert!(matches!(&result, TransactionSource::Block {
            block_hash: actual, block_number: 42, block_timestamp: 43, base_fee: Some(7), index: 0, ..
        } if *actual == block_hash));
        assert_eq!(result.into_recovered().into_inner(), transaction);
        cache_new_blocks_task(
            api.cache().clone(),
            futures::stream::iter([CanonStateNotification::Commit { new: canonical.clone() }]),
        )
        .await;
        assert!(matches!(
            LoadTransaction::transaction_by_hash(&api, hash).await.unwrap(),
            Some(TransactionSource::Block { block_number: 41, .. })
        ));
        let replacement = Arc::new(Chain::<OpPrimitives>::new(
            [RecoveredBlock::new_unhashed(
                Block::new(
                    Header { number: 41, timestamp: 44, ..Default::default() },
                    Default::default(),
                ),
                vec![],
            )],
            ExecutionOutcome::new(Default::default(), vec![vec![]], 41, vec![]),
            Default::default(),
        ));
        cache_new_blocks_task(
            api.cache().clone(),
            futures::stream::iter([CanonStateNotification::Reorg {
                old: canonical,
                new: replacement,
            }]),
        )
        .await;
        assert!(matches!(
            LoadTransaction::transaction_by_hash(&api, hash).await.unwrap(),
            Some(TransactionSource::Block { block_number: 42, .. })
        ));
        assert!(LoadTransaction::transaction_by_hash(&api, B256::ZERO).await.unwrap().is_none());
        pending_tx.send_replace(None);
        assert!(LoadTransaction::transaction_by_hash(&api, hash).await.unwrap().is_none());
        drop(pending_tx);
        assert!(LoadTransaction::transaction_by_hash(&api, hash).await.unwrap().is_none());
    }
}
