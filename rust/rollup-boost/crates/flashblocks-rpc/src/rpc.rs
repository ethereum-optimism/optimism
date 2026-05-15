use alloy_eips::{BlockId, BlockNumberOrTag};
use alloy_primitives::{Address, TxHash, U256};
use jsonrpsee::{
    core::{RpcResult, async_trait},
    proc_macros::rpc,
};
use op_alloy_network::Optimism;
use reth_optimism_primitives::OpTransactionSigned;
use reth_provider::TransactionsProvider;
use reth_rpc_eth_api::{
    RpcBlock, RpcNodeCore, RpcReceipt,
    helpers::{EthBlocks, EthState, EthTransactions, FullEthApi},
};
use tracing::debug;

#[cfg_attr(not(test), rpc(server, namespace = "eth"))]
#[cfg_attr(test, rpc(server, client, namespace = "eth"))]
pub trait EthApiOverride {
    #[method(name = "getBlockByNumber")]
    async fn block_by_number(
        &self,
        number: BlockNumberOrTag,
        full: bool,
    ) -> RpcResult<Option<RpcBlock<op_alloy_network::Optimism>>>;

    #[method(name = "getTransactionReceipt")]
    async fn get_transaction_receipt(
        &self,
        tx_hash: TxHash,
    ) -> RpcResult<Option<RpcReceipt<Optimism>>>;

    #[method(name = "getBalance")]
    async fn get_balance(&self, address: Address, block_number: Option<BlockId>)
    -> RpcResult<U256>;

    #[method(name = "getTransactionCount")]
    async fn get_transaction_count(
        &self,
        address: Address,
        block_number: Option<BlockId>,
    ) -> RpcResult<U256>;
}

#[async_trait]
pub trait FlashblocksApi {
    async fn block_by_number(&self, full: bool) -> Option<RpcBlock<Optimism>>;

    async fn get_transaction_receipt(&self, tx_hash: TxHash) -> Option<RpcReceipt<Optimism>>;

    async fn get_balance(&self, address: Address) -> Option<U256>;

    async fn get_transaction_count(&self, address: Address) -> Option<u64>;
}

pub struct FlashblocksApiExt<Eth, FB> {
    eth_api: Eth,
    flashblocks_api: FB,
}

impl<Eth, FB> FlashblocksApiExt<Eth, FB> {
    pub fn new(eth_api: Eth, flashblocks_api: FB) -> Self {
        Self {
            eth_api,
            flashblocks_api,
        }
    }
}

#[async_trait]
impl<Eth, FB> EthApiOverrideServer for FlashblocksApiExt<Eth, FB>
where
    Eth: FullEthApi<NetworkTypes = Optimism> + Send + Sync + 'static,
    Eth: RpcNodeCore,
    <Eth as RpcNodeCore>::Provider: TransactionsProvider<Transaction = OpTransactionSigned>,
    FB: FlashblocksApi + Send + Sync + 'static,
{
    async fn block_by_number(
        &self,
        number: BlockNumberOrTag,
        full: bool,
    ) -> RpcResult<Option<RpcBlock<Optimism>>> {
        debug!("block_by_number: {:?}", number);

        if number.is_pending() {
            Ok(self.flashblocks_api.block_by_number(full).await)
        } else {
            EthBlocks::rpc_block(&self.eth_api, number.into(), full)
                .await
                .map_err(Into::into)
        }
    }

    async fn get_transaction_receipt(
        &self,
        tx_hash: TxHash,
    ) -> RpcResult<Option<RpcReceipt<Optimism>>> {
        debug!("get_transaction_receipt: {:?}", tx_hash);

        if let Some(fb_receipt) = self.flashblocks_api.get_transaction_receipt(tx_hash).await {
            return Ok(Some(fb_receipt));
        }

        EthTransactions::transaction_receipt(&self.eth_api, tx_hash)
            .await
            .map_err(Into::into)
    }

    async fn get_balance(
        &self,
        address: Address,
        block_number: Option<BlockId>,
    ) -> RpcResult<U256> {
        debug!("get_balance: {:?}, {:?}", address, block_number);

        let block_id = block_number.unwrap_or_default();
        if block_id.is_pending()
            && let Some(balance) = self.flashblocks_api.get_balance(address).await
        {
            return Ok(balance);
        }
        EthState::balance(&self.eth_api, address, block_number)
            .await
            .map_err(Into::into)
    }

    async fn get_transaction_count(
        &self,
        address: Address,
        block_number: Option<BlockId>,
    ) -> RpcResult<U256> {
        debug!("get_transaction_count: {:?}, {:?}", address, block_number);

        let block_id = block_number.unwrap_or_default();
        if block_id.is_pending() {
            let latest_count = EthState::transaction_count(
                &self.eth_api,
                address,
                Some(BlockId::Number(BlockNumberOrTag::Latest)),
            )
            .await
            .map_err(Into::into)?;

            if let Some(count) = self.flashblocks_api.get_transaction_count(address).await {
                return Ok(latest_count + U256::from(count));
            }
        }

        EthState::transaction_count(&self.eth_api, address, block_number)
            .await
            .map_err(Into::into)
    }
}
