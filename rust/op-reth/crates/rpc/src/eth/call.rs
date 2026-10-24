use crate::{OpEthApi, OpEthApiError, eth::RpcNodeCore};
use alloy_consensus::{BlockHeader, transaction::TxHashRef};
use alloy_primitives::B256;
use alloy_rpc_types_eth::TransactionInfo;
use reth_errors::RethError;
use reth_evm::{
    ConfigureEvm, HaltReasonFor,
    block::{BlockExecutor, TxResult},
};
use reth_revm::{State, database::StateProviderDatabase};
use reth_rpc_eth_api::{
    FromEthApiError, FromEvmError, RpcConvert,
    helpers::{
        Call, EthCall, LoadBlock, LoadState, LoadTransaction, SpawnBlocking, estimate::EstimateCall,
    },
};
use reth_rpc_eth_types::StateCacheDb;
use revm::context_interface::result::ResultAndState;

impl<N, Rpc> EthCall for OpEthApi<N, Rpc>
where
    N: RpcNodeCore,
    OpEthApiError: FromEvmError<N::Evm>,
    Rpc: RpcConvert<Primitives = N::Primitives, Error = OpEthApiError, Evm = N::Evm>,
{
}

impl<N, Rpc> EstimateCall for OpEthApi<N, Rpc>
where
    N: RpcNodeCore,
    OpEthApiError: FromEvmError<N::Evm>,
    Rpc: RpcConvert<Primitives = N::Primitives, Error = OpEthApiError, Evm = N::Evm>,
{
}

/// UPSTREAM-MIRROR(override): reth@rev:8d359d0 `reth_rpc_eth_api::helpers::Call`
///
/// `spawn_replay_transaction` executes the target through the block executor, see
/// [`crate::eth::trace`].
impl<N, Rpc> Call for OpEthApi<N, Rpc>
where
    N: RpcNodeCore,
    OpEthApiError: FromEvmError<N::Evm>,
    Rpc: RpcConvert<Primitives = N::Primitives, Error = OpEthApiError, Evm = N::Evm>,
{
    #[inline]
    fn call_gas_limit(&self) -> u64 {
        self.inner.eth_api.gas_cap()
    }

    #[inline]
    fn max_simulate_blocks(&self) -> u64 {
        self.inner.eth_api.max_simulate_blocks()
    }

    #[inline]
    fn compute_state_root_for_eth_simulate(&self) -> bool {
        self.inner.eth_api.compute_state_root_for_eth_simulate()
    }

    #[inline]
    fn evm_memory_limit(&self) -> u64 {
        self.inner.eth_api.evm_memory_limit()
    }

    async fn spawn_replay_transaction<F, R>(
        &self,
        hash: B256,
        f: F,
    ) -> Result<Option<R>, Self::Error>
    where
        Self: LoadBlock + LoadTransaction,
        F: FnOnce(
                TransactionInfo,
                ResultAndState<HaltReasonFor<Self::Evm>>,
                StateCacheDb,
            ) -> Result<R, Self::Error>
            + Send
            + 'static,
        R: Send + 'static,
    {
        let Some((transaction, block)) = self.transaction_and_block(hash).await? else {
            return Ok(None);
        };
        let (tx, tx_info) = transaction.split();

        // `Call::spawn_with_state_at_block`, inlined: calling it here would need this impl's
        // own `Call` bounds, which the compiler can't prove inside it.
        let parent_hash = block.parent_hash();
        self.spawn_blocking_io_fut(async move |this| {
            let state = this.state_at_block_id(parent_hash.into()).await?;
            let mut db = State::builder().with_database(StateProviderDatabase::new(state)).build();
            let mut executor = RpcNodeCore::evm_config(&this)
                .executor_for_block(&mut db, block.sealed_block())
                .map_err(RethError::other)
                .map_err(Self::Error::from_eth_err)?;
            executor.apply_pre_execution_changes().map_err(Self::Error::from_eth_err)?;

            for block_tx in block.transactions_recovered() {
                if block_tx.tx_hash() == tx.tx_hash() {
                    break;
                }
                executor.execute_transaction(block_tx).map_err(Self::Error::from_eth_err)?;
            }

            let res = executor
                .execute_transaction_without_commit(tx)
                .map_err(Self::Error::from_eth_err)?
                .into_result();
            drop(executor);
            f(tx_info, res, db)
        })
        .await
        .map(Some)
    }
}
