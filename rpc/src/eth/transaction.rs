//! Loads and formats OP transaction RPC response.

use crate::{
    eth::{OpEthApiInner, OpNodeCore},
    OpEthApi, OpEthApiError, SequencerClient,
};
use alloy_primitives::{Bytes, B256};
use alloy_rpc_types_eth::TransactionInfo;
use op_alloy_consensus::{transaction::OpTransactionInfo, OpTxEnvelope};
use reth_node_api::FullNodeComponents;
use reth_optimism_primitives::DepositReceipt;
use reth_rpc_eth_api::{
    helpers::{EthSigner, EthTransactions, LoadTransaction, SpawnBlocking},
    try_into_op_tx_info, EthApiTypes, FromEthApiError, FullEthApiTypes, RpcNodeCore,
    RpcNodeCoreExt, TxInfoMapper,
};
use reth_rpc_eth_types::utils::recover_raw_transaction;
use reth_storage_api::{
    errors::ProviderError, BlockReader, BlockReaderIdExt, ProviderTx, ReceiptProvider,
    TransactionsProvider,
};
use reth_transaction_pool::{PoolTransaction, TransactionOrigin, TransactionPool};
use std::{
    fmt::{Debug, Formatter},
    sync::Arc,
};

impl<N> EthTransactions for OpEthApi<N>
where
    Self: LoadTransaction<Provider: BlockReaderIdExt> + EthApiTypes<Error = OpEthApiError>,
    N: OpNodeCore<Provider: BlockReader<Transaction = ProviderTx<Self::Provider>>>,
{
    fn signers(&self) -> &parking_lot::RwLock<Vec<Box<dyn EthSigner<ProviderTx<Self::Provider>>>>> {
        self.inner.eth_api.signers()
    }

    /// Decodes and recovers the transaction and submits it to the pool.
    ///
    /// Returns the hash of the transaction.
    async fn send_raw_transaction(&self, tx: Bytes) -> Result<B256, Self::Error> {
        let recovered = recover_raw_transaction(&tx)?;

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
            let _ = self
                .pool()
                .add_transaction(TransactionOrigin::Local, pool_transaction)
                .await.inspect_err(|err| {
                    tracing::warn!(target: "rpc::eth", %err, %hash, "successfully sent tx to sequencer, but failed to persist in local tx pool");
            });

            return Ok(hash)
        }

        // submit the transaction to the pool with a `Local` origin
        let hash = self
            .pool()
            .add_transaction(TransactionOrigin::Local, pool_transaction)
            .await
            .map_err(Self::Error::from_eth_err)?;

        Ok(hash)
    }
}

impl<N> LoadTransaction for OpEthApi<N>
where
    Self: SpawnBlocking + FullEthApiTypes + RpcNodeCoreExt,
    N: OpNodeCore<Provider: TransactionsProvider, Pool: TransactionPool>,
    Self::Pool: TransactionPool,
{
}

impl<N> OpEthApi<N>
where
    N: OpNodeCore,
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
#[derive(Clone)]
pub struct OpTxInfoMapper<N: OpNodeCore>(Arc<OpEthApiInner<N>>);

impl<N: OpNodeCore> Debug for OpTxInfoMapper<N> {
    fn fmt(&self, f: &mut Formatter<'_>) -> std::fmt::Result {
        f.debug_struct("OpTxInfoMapper").finish()
    }
}

impl<N: OpNodeCore> OpTxInfoMapper<N> {
    /// Creates [`OpTxInfoMapper`] that uses [`ReceiptProvider`] borrowed from given `eth_api`.
    pub const fn new(eth_api: Arc<OpEthApiInner<N>>) -> Self {
        Self(eth_api)
    }
}

impl<N> TxInfoMapper<&OpTxEnvelope> for OpTxInfoMapper<N>
where
    N: FullNodeComponents,
    N::Provider: ReceiptProvider<Receipt: DepositReceipt>,
{
    type Out = OpTransactionInfo;
    type Err = ProviderError;

    fn try_map(
        &self,
        tx: &OpTxEnvelope,
        tx_info: TransactionInfo,
    ) -> Result<Self::Out, ProviderError> {
        try_into_op_tx_info(self.0.eth_api.provider(), tx, tx_info)
    }
}
