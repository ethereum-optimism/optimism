//! Loads and formats OP transaction RPC response.

use crate::{OpEthApi, OpEthApiError, SequencerClient};
use alloy_primitives::{Bytes, B256};
use alloy_rpc_types_eth::TransactionInfo;
use futures::StreamExt;
use op_alloy_consensus::{transaction::OpTransactionInfo, OpTransaction};
use reth_chain_state::CanonStateSubscriptions;
use reth_optimism_primitives::DepositReceipt;
use reth_primitives_traits::{
    BlockBody, Recovered, SignedTransaction, SignerRecoverable, WithEncoded,
};
use reth_rpc_eth_api::{
    helpers::{spec::SignersForRpc, EthTransactions, LoadReceipt, LoadTransaction, SpawnBlocking},
    try_into_op_tx_info, EthApiTypes as _, FromEthApiError, FromEvmError, RpcConvert, RpcNodeCore,
    RpcReceipt, TxInfoMapper,
};
use reth_rpc_eth_types::{EthApiError, TransactionSource};
use reth_storage_api::{errors::ProviderError, ProviderTx, ReceiptProvider, TransactionsProvider};
use reth_transaction_pool::{
    AddedTransactionOutcome, PoolPooledTx, PoolTransaction, TransactionOrigin, TransactionPool,
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

    async fn send_transaction(
        &self,
        tx: WithEncoded<Recovered<PoolPooledTx<Self::Pool>>>,
    ) -> Result<B256, Self::Error> {
        let (tx, recovered) = tx.split();

        // broadcast raw transaction to subscribers if there is any.
        self.eth_api().broadcast_raw_transaction(tx.clone());

        let pool_transaction = <Self::Pool as TransactionPool>::Transaction::from_pooled(recovered);

        // On optimism, transactions are forwarded directly to the sequencer to be included in
        // blocks that it builds.
        if let Some(client) = self.raw_tx_forwarder().as_ref() {
            tracing::debug!(target: "rpc::eth", hash = %pool_transaction.hash(), "forwarding raw transaction to sequencer");
            let hash = client.forward_raw_transaction(&tx).await.inspect_err(|err| {
                    tracing::debug!(target: "rpc::eth", %err, hash=% *pool_transaction.hash(), "failed to forward raw transaction");
                })?;

            // Retain tx in local tx pool after forwarding, for local RPC usage.
            let _ = self.inner.eth_api.add_pool_transaction(pool_transaction).await.inspect_err(|err| {
                tracing::warn!(target: "rpc::eth", %err, %hash, "successfully sent tx to sequencer, but failed to persist in local tx pool");
            });

            return Ok(hash)
        }

        // submit the transaction to the pool with a `Local` origin
        let AddedTransactionOutcome { hash, .. } = self
            .pool()
            .add_transaction(TransactionOrigin::Local, pool_transaction)
            .await
            .map_err(Self::Error::from_eth_err)?;

        Ok(hash)
    }

    /// Decodes and recovers the transaction and submits it to the pool.
    ///
    /// And awaits the receipt, checking both canonical blocks and flashblocks for faster
    /// confirmation.
    fn send_raw_transaction_sync(
        &self,
        tx: Bytes,
    ) -> impl Future<Output = Result<RpcReceipt<Self::NetworkTypes>, Self::Error>> + Send {
        let this = self.clone();
        let timeout_duration = self.send_raw_transaction_sync_timeout();
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
                                for block in chain.blocks_iter() {
                                    if block.body().contains_transaction(&hash)
                                        && let Some(receipt) = this.transaction_receipt(hash).await? {
                                            return Ok(receipt);
                                        }
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
            let Some((tx, meta, receipt)) = tx_receipt else { return Ok(None) };
            self.build_transaction_receipt(tx, meta, receipt).await.map(Some)
        }
    }
}

impl<N, Rpc> LoadTransaction for OpEthApi<N, Rpc>
where
    N: RpcNodeCore,
    OpEthApiError: FromEvmError<N::Evm>,
    Rpc: RpcConvert<Primitives = N::Primitives, Error = OpEthApiError>,
{
    async fn transaction_by_hash(
        &self,
        hash: B256,
    ) -> Result<Option<TransactionSource<ProviderTx<Self::Provider>>>, Self::Error> {
        // 1. Try to find the transaction on disk (historical blocks)
        if let Some((tx, meta)) = self
            .spawn_blocking_io(move |this| {
                this.provider()
                    .transaction_by_hash_with_meta(hash)
                    .map_err(Self::Error::from_eth_err)
            })
            .await?
        {
            let transaction = tx
                .try_into_recovered_unchecked()
                .map_err(|_| EthApiError::InvalidTransactionSignature)?;

            return Ok(Some(TransactionSource::Block {
                transaction,
                index: meta.index,
                block_hash: meta.block_hash,
                block_number: meta.block_number,
                base_fee: meta.base_fee,
            }));
        }

        // 2. check flashblocks (sequencer preconfirmations)
        if let Ok(Some(pending_block)) = self.pending_flashblock().await &&
            let Some(indexed_tx) = pending_block.block().find_indexed(hash)
        {
            let meta = indexed_tx.meta();
            return Ok(Some(TransactionSource::Block {
                transaction: indexed_tx.recovered_tx().cloned(),
                index: meta.index,
                block_hash: meta.block_hash,
                block_number: meta.block_number,
                base_fee: meta.base_fee,
            }));
        }

        // 3. check local pool
        if let Some(tx) = self.pool().get(&hash).map(|tx| tx.transaction.clone_into_consensus()) {
            return Ok(Some(TransactionSource::Pool(tx)));
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
        try_into_op_tx_info(&self.provider, tx, tx_info)
    }
}
