//! Loads and formats OP transaction RPC response.

use alloy_consensus::{transaction::Recovered, SignableTransaction, Transaction as _};
use alloy_primitives::{Bytes, Signature, B256};
use alloy_rpc_types_eth::TransactionInfo;
use op_alloy_consensus::OpTxEnvelope;
use op_alloy_rpc_types::{OpTransactionRequest, Transaction};
use reth_node_api::FullNodeComponents;
use reth_optimism_primitives::{OpReceipt, OpTransactionSigned};
use reth_rpc_eth_api::{
    helpers::{EthSigner, EthTransactions, LoadTransaction, SpawnBlocking},
    EthApiTypes, FromEthApiError, FullEthApiTypes, RpcNodeCore, RpcNodeCoreExt, TransactionCompat,
};
use reth_rpc_eth_types::{utils::recover_raw_transaction, EthApiError};
use reth_storage_api::{
    BlockReader, BlockReaderIdExt, ProviderTx, ReceiptProvider, TransactionsProvider,
};
use reth_transaction_pool::{PoolTransaction, TransactionOrigin, TransactionPool};

use crate::{eth::OpNodeCore, OpEthApi, OpEthApiError, SequencerClient};

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

impl<N> TransactionCompat<OpTransactionSigned> for OpEthApi<N>
where
    N: FullNodeComponents<Provider: ReceiptProvider<Receipt = OpReceipt>>,
{
    type Transaction = Transaction;
    type Error = OpEthApiError;

    fn fill(
        &self,
        tx: Recovered<OpTransactionSigned>,
        tx_info: TransactionInfo,
    ) -> Result<Self::Transaction, Self::Error> {
        let mut tx = tx.convert::<OpTxEnvelope>();
        let mut deposit_receipt_version = None;
        let mut deposit_nonce = None;

        if let OpTxEnvelope::Deposit(tx) = tx.inner_mut() {
            // for depost tx we need to fetch the receipt
            self.inner
                .eth_api
                .provider()
                .receipt_by_hash(tx.tx_hash())
                .map_err(Self::Error::from_eth_err)?
                .inspect(|receipt| {
                    if let OpReceipt::Deposit(receipt) = receipt {
                        deposit_receipt_version = receipt.deposit_receipt_version;
                        deposit_nonce = receipt.deposit_nonce;
                    }
                });

            // For consistency with op-geth, we always return `0x0` for mint if it is
            // missing This is because op-geth does not distinguish
            // between null and 0, because this value is decoded from RLP where null is
            // represented as 0
            tx.inner_mut().mint = Some(tx.mint.unwrap_or_default());
        }

        let TransactionInfo {
            block_hash, block_number, index: transaction_index, base_fee, ..
        } = tx_info;

        let effective_gas_price = if tx.is_deposit() {
            // For deposits, we must always set the `gasPrice` field to 0 in rpc
            // deposit tx don't have a gas price field, but serde of `Transaction` will take care of
            // it
            0
        } else {
            base_fee
                .map(|base_fee| {
                    tx.effective_tip_per_gas(base_fee).unwrap_or_default() + base_fee as u128
                })
                .unwrap_or_else(|| tx.max_fee_per_gas())
        };

        Ok(Transaction {
            inner: alloy_rpc_types_eth::Transaction {
                inner: tx,
                block_hash,
                block_number,
                transaction_index,
                effective_gas_price: Some(effective_gas_price),
            },
            deposit_nonce,
            deposit_receipt_version,
        })
    }

    fn build_simulate_v1_transaction(
        &self,
        request: alloy_rpc_types_eth::TransactionRequest,
    ) -> Result<OpTransactionSigned, Self::Error> {
        let request: OpTransactionRequest = request.into();
        let Ok(tx) = request.build_typed_tx() else {
            return Err(OpEthApiError::Eth(EthApiError::TransactionConversionError))
        };

        // Create an empty signature for the transaction.
        let signature = Signature::new(Default::default(), Default::default(), false);
        Ok(tx.into_signed(signature).into())
    }

    fn otterscan_api_truncate_input(tx: &mut Self::Transaction) {
        let input = tx.inner.inner.inner_mut().input_mut();
        *input = input.slice(..4);
    }
}
