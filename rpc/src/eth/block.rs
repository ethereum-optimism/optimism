//! Loads and formats OP block RPC response.

use alloy_consensus::{transaction::TransactionMeta, BlockHeader};
use alloy_rpc_types_eth::BlockId;
use op_alloy_rpc_types::OpTransactionReceipt;
use reth_chainspec::ChainSpecProvider;
use reth_node_api::BlockBody;
use reth_optimism_chainspec::OpChainSpec;
use reth_optimism_primitives::{OpReceipt, OpTransactionSigned};
use reth_rpc_eth_api::{
    helpers::{EthBlocks, LoadBlock, LoadPendingBlock, LoadReceipt, SpawnBlocking},
    types::RpcTypes,
    RpcReceipt,
};
use reth_storage_api::{BlockReader, HeaderProvider, ProviderTx};
use reth_transaction_pool::{PoolTransaction, TransactionPool};

use crate::{eth::OpNodeCore, OpEthApi, OpEthApiError, OpReceiptBuilder};

impl<N> EthBlocks for OpEthApi<N>
where
    Self: LoadBlock<
        Error = OpEthApiError,
        NetworkTypes: RpcTypes<Receipt = OpTransactionReceipt>,
        Provider: BlockReader<Receipt = OpReceipt, Transaction = OpTransactionSigned>,
    >,
    N: OpNodeCore<Provider: ChainSpecProvider<ChainSpec = OpChainSpec> + HeaderProvider>,
{
    async fn block_receipts(
        &self,
        block_id: BlockId,
    ) -> Result<Option<Vec<RpcReceipt<Self::NetworkTypes>>>, Self::Error>
    where
        Self: LoadReceipt,
    {
        if let Some((block, receipts)) = self.load_block_and_receipts(block_id).await? {
            let block_number = block.number();
            let base_fee = block.base_fee_per_gas();
            let block_hash = block.hash();
            let excess_blob_gas = block.excess_blob_gas();
            let timestamp = block.timestamp();

            let mut l1_block_info = reth_optimism_evm::extract_l1_info(block.body())?;

            return block
                .body()
                .transactions()
                .iter()
                .zip(receipts.iter())
                .enumerate()
                .map(|(idx, (tx, receipt))| -> Result<_, _> {
                    let meta = TransactionMeta {
                        tx_hash: tx.tx_hash(),
                        index: idx as u64,
                        block_hash,
                        block_number,
                        base_fee,
                        excess_blob_gas,
                        timestamp,
                    };

                    // We must clear this cache as different L2 transactions can have different
                    // L1 costs. A potential improvement here is to only clear the cache if the
                    // new transaction input has changed, since otherwise the L1 cost wouldn't.
                    l1_block_info.clear_tx_l1_cost();

                    Ok(OpReceiptBuilder::new(
                        &self.inner.eth_api.provider().chain_spec(),
                        tx,
                        meta,
                        receipt,
                        &receipts,
                        &mut l1_block_info,
                    )?
                    .build())
                })
                .collect::<Result<Vec<_>, Self::Error>>()
                .map(Some)
        }

        Ok(None)
    }
}

impl<N> LoadBlock for OpEthApi<N>
where
    Self: LoadPendingBlock<
            Pool: TransactionPool<
                Transaction: PoolTransaction<Consensus = ProviderTx<Self::Provider>>,
            >,
        > + SpawnBlocking,
    N: OpNodeCore,
{
}
