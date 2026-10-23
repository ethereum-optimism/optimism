//! Replays of a block's own transactions for the `debug_` and `trace_` namespaces.
//!
//! SDM settlement (canonical gas, the sender's refund and the fee vaults' debits) is applied by the
//! block executor, keyed by each transaction's position in the block, so a replay on the bare EVM
//! reports pre-refund gas and unsettled balances. These overrides replay through the executor.
//!
//! op-reth never caches a block access list (`ensure_no_computed_bals` rejects the options that
//! would), so the BAL argument upstream uses to skip the replay is always `None` here.

use crate::{OpEthApi, OpEthApiError};
use alloy_eip7928::bal::DecodedBal;
use reth_errors::RethError;
use reth_evm::{
    ConfigureEvm, Evm, EvmEnvFor, HaltReasonFor, InspectorFor, IntoTxEnv, TxEnvFor,
    block::{BlockExecutor, TxResult},
};
use reth_primitives_traits::{BlockBody, BlockTy, RecoveredBlock};
use reth_rpc_eth_api::{
    FromEthApiError, FromEvmError, RpcConvert, RpcNodeCore,
    helpers::{LoadState, Trace},
};
use reth_rpc_eth_types::{EthApiError, StateCacheDb};
use revm::{context_interface::result::ResultAndState, state::bal::Bal as RevmBal};
use std::sync::Arc;

/// UPSTREAM-MIRROR(override): reth@rev:8d359d0 `reth_rpc_eth_api::helpers::Trace`
///
/// Replays through the block executor where upstream replays on the bare EVM.
impl<N, Rpc> Trace for OpEthApi<N, Rpc>
where
    N: RpcNodeCore,
    OpEthApiError: FromEvmError<N::Evm>,
    Rpc: RpcConvert<Primitives = N::Primitives, Error = OpEthApiError, Evm = N::Evm>,
{
    fn replay_block_until(
        &self,
        db: &mut StateCacheDb,
        block: &RecoveredBlock<BlockTy<Self::Primitives>>,
        target_tx_index: usize,
        _bal: Option<&DecodedBal<Arc<RevmBal>>>,
    ) -> Result<(), Self::Error> {
        let mut executor = self
            .evm_config()
            .executor_for_block(db, block.sealed_block())
            .map_err(RethError::other)
            .map_err(Self::Error::from_eth_err)?;
        executor.apply_pre_execution_changes().map_err(Self::Error::from_eth_err)?;

        for tx in block.transactions_recovered().take(target_tx_index) {
            executor.execute_transaction(tx).map_err(Self::Error::from_eth_err)?;
        }
        Ok(())
    }

    /// Executes the block's transaction at the target index, not `target_tx_env`: the executor
    /// derives the refund from the transaction it runs and its position.
    fn inspect_transaction_in_block<'a>(
        &self,
        block: &RecoveredBlock<BlockTy<Self::Primitives>>,
        db: &'a mut StateCacheDb,
        inspector: impl InspectorFor<Self::Evm, &'a mut StateCacheDb>,
        target_tx_index: usize,
        _target_tx_env: impl IntoTxEnv<TxEnvFor<Self::Evm>>,
        _bal: Option<&DecodedBal<Arc<RevmBal>>>,
    ) -> Result<(ResultAndState<HaltReasonFor<Self::Evm>>, EvmEnvFor<Self::Evm>), Self::Error> {
        let evm_env = self.evm_env_for_header(block.sealed_block().sealed_header())?;
        let evm_config = self.evm_config();
        let ctx = evm_config
            .context_for_block(block.sealed_block())
            .map_err(RethError::other)
            .map_err(Self::Error::from_eth_err)?;
        let mut evm = evm_config.evm_with_env_and_inspector(db, evm_env.clone(), inspector);
        // The transactions before the target replay uninspected.
        evm.disable_inspector();
        let mut executor = evm_config.create_executor_with_state(evm, ctx);
        executor.apply_pre_execution_changes().map_err(Self::Error::from_eth_err)?;

        let mut block_txs = block.transactions_recovered();
        for tx in block_txs.by_ref().take(target_tx_index) {
            executor.execute_transaction(tx).map_err(Self::Error::from_eth_err)?;
        }
        let target_tx = block_txs.next().ok_or_else(|| {
            Self::Error::from_eth_err(EthApiError::InvalidParams(format!(
                "tx_index {target_tx_index} out of bounds for block with {} transactions",
                block.body().transaction_count()
            )))
        })?;

        executor.evm_mut().enable_inspector();
        let res = executor
            .execute_transaction_without_commit(target_tx)
            .map_err(Self::Error::from_eth_err)?
            .into_result();
        Ok((res, evm_env))
    }
}
