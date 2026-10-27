//! Replays of a block's own transactions for the `debug_` and `trace_` namespaces.
//!
//! SDM settlement (canonical gas, the sender's refund and the fee vaults' debits) is applied by the
//! block executor, keyed by each transaction's position in the block, so a replay on the bare EVM
//! reports pre-refund gas and unsettled balances. These overrides replay through the executor.
//!
//! op-reth never caches a block access list (`ensure_no_computed_bals` rejects the options that
//! would), so the BAL argument upstream uses to skip the replay is always `None` here.

use crate::{OpEthApi, OpEthApiError};
use alloy_consensus::transaction::TxHashRef;
use alloy_eip7928::bal::DecodedBal;
use alloy_primitives::B256;
use alloy_rpc_types_eth::TransactionInfo;
use alloy_rpc_types_trace::geth::{GethDebugTracingOptions, TraceResult};
use reth_errors::RethError;
use reth_evm::{
    ConfigureEvm, Evm, EvmEnvFor, EvmFor, InspectorFor, IntoTxEnv, TxEnvFor,
    block::{BlockExecutor, BlockExecutorFor, TxResult},
};
use reth_primitives_traits::{BlockBody, BlockTy, Recovered, RecoveredBlock};
use reth_rpc_eth_api::{
    FromEthApiError, FromEvmError, RpcConvert, RpcNodeCore,
    helpers::{
        LoadState, Trace,
        trace::{BlockReplayMode, TracingCtx, TransactionReplayResult},
    },
};
use reth_rpc_eth_types::{EthApiError, StateCacheDb};
use reth_storage_api::{HashedPostStateProvider, ProviderBlock, ProviderTx, StateRootProvider};
use revm::{
    context::Block, context_interface::result::ResultAndState,
    database::states::bundle_state::BundleRetention, state::bal::Bal as RevmBal,
};
use revm_inspectors::tracing::{DebugInspector, TransactionContext};
use std::sync::Arc;

/// UPSTREAM-MIRROR(override): reth@rev:1f20b35 `reth_rpc_eth_api::helpers::Trace`
///
/// Canonical replays include executor settlement; diagnostic block replays use the bare EVM.
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

    fn inspect_canonical_transaction_in_block<'a>(
        &self,
        block: &RecoveredBlock<BlockTy<Self::Primitives>>,
        db: &'a mut StateCacheDb,
        inspector: impl InspectorFor<Self::Evm, &'a mut StateCacheDb>,
        target_tx_index: usize,
        _bal: Option<&DecodedBal<Arc<RevmBal>>>,
    ) -> TransactionReplayResult<Self::Evm, Self::Error> {
        let evm_env = self.evm_env_for_header(block.sealed_block().sealed_header())?;
        let mut executor = inspected_block_executor(self, block, db, evm_env.clone(), inspector)?;

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

    fn inspect_transaction_in_block<'a>(
        &self,
        block: &RecoveredBlock<BlockTy<Self::Primitives>>,
        db: &'a mut StateCacheDb,
        inspector: impl InspectorFor<Self::Evm, &'a mut StateCacheDb>,
        target_tx_index: usize,
        target_tx_env: impl IntoTxEnv<TxEnvFor<Self::Evm>>,
        _bal: Option<&DecodedBal<Arc<RevmBal>>>,
    ) -> TransactionReplayResult<Self::Evm, Self::Error> {
        let evm_env = self.evm_env_for_header(block.sealed_block().sealed_header())?;
        let mut executor = inspected_block_executor(self, block, db, evm_env.clone(), inspector)?;
        for transaction in block.transactions_recovered().take(target_tx_index) {
            executor.execute_transaction(transaction).map_err(Self::Error::from_eth_err)?;
        }
        executor.evm_mut().enable_inspector();
        let result =
            executor.evm_mut().transact(target_tx_env).map_err(Self::Error::from_evm_err)?;
        Ok((result, evm_env))
    }

    /// Unlike upstream, a transaction that fails to execute fails the call instead of ending the
    /// iteration early.
    fn trace_block_transactions<Insp, F, R>(
        &self,
        db: &mut StateCacheDb,
        block: &RecoveredBlock<ProviderBlock<Self::Provider>>,
        evm_env: EvmEnvFor<Self::Evm>,
        max_transactions: usize,
        inspector: Insp,
        f: F,
    ) -> Result<Vec<R>, Self::Error>
    where
        F: Fn(
            TransactionInfo,
            TracingCtx<
                '_,
                Recovered<&ProviderTx<Self::Provider>>,
                EvmFor<Self::Evm, &mut StateCacheDb, Insp>,
            >,
        ) -> Result<R, Self::Error>,
        Insp: Clone + for<'a> InspectorFor<Self::Evm, &'a mut StateCacheDb>,
    {
        let block_hash = block.hash();
        let block_number = evm_env.block_env.number().saturating_to();
        let block_timestamp = evm_env.block_env.timestamp().saturating_to();
        let base_fee = evm_env.block_env.basefee();

        let fused_inspector = inspector.clone();
        let mut executor = inspected_block_executor(self, block, db, evm_env, inspector)?;
        executor.evm_mut().enable_inspector();

        let mut results =
            Vec::with_capacity(max_transactions.min(block.body().transaction_count()));
        let mut transactions =
            block.transactions_recovered().take(max_transactions).enumerate().peekable();
        while let Some((idx, tx)) = transactions.next() {
            let output = executor
                .execute_transaction_without_commit(tx)
                .map_err(Self::Error::from_eth_err)?;
            let ResultAndState { result, state } = output.result();

            let tx_info = TransactionInfo {
                hash: Some(*tx.tx_hash()),
                index: Some(idx as u64),
                block_hash: Some(block_hash),
                block_number: Some(block_number),
                block_timestamp: Some(block_timestamp),
                base_fee: Some(base_fee),
            };
            let mut was_fused = false;
            let (db, inspector, _) = executor.evm_mut().components_mut();
            let traced = f(
                tx_info,
                TracingCtx::new(
                    tx,
                    result.clone(),
                    state,
                    inspector,
                    db,
                    &fused_inspector,
                    &mut was_fused,
                ),
            );
            // The inspector and state of the last traced transaction are not needed.
            if transactions.peek().is_some() {
                if !was_fused {
                    *executor.evm_mut().inspector_mut() = fused_inspector.clone();
                }
                executor.commit_transaction(output);
            }
            results.push(traced?);
        }
        Ok(results)
    }

    fn debug_trace_block_transactions(
        &self,
        db: &mut StateCacheDb,
        block: &RecoveredBlock<ProviderBlock<Self::Provider>>,
        evm_env: EvmEnvFor<Self::Evm>,
        mode: BlockReplayMode,
        opts: GethDebugTracingOptions,
    ) -> Result<Vec<TraceResult>, Self::Error> {
        if mode == BlockReplayMode::Unvalidated {
            return self.debug_trace_unvalidated_block_transactions(db, block, evm_env, opts);
        }
        let mut results = Vec::with_capacity(block.body().transactions().len());
        let block_env = evm_env.block_env.clone();

        let inspector = DebugInspector::new(opts).map_err(Self::Error::from_eth_err)?;
        let mut executor = inspected_block_executor(self, block, db, evm_env, inspector)?;
        executor.evm_mut().enable_inspector();

        let mut transactions = block.transactions_recovered().enumerate().peekable();
        while let Some((index, tx)) = transactions.next() {
            let tx_env = self.evm_config().tx_env(tx);
            let output = executor
                .execute_transaction_without_commit(tx)
                .map_err(Self::Error::from_eth_err)?;

            let (db, inspector, _) = executor.evm_mut().components_mut();
            let result = inspector
                .get_result(
                    Some(TransactionContext {
                        block_hash: Some(block.hash()),
                        tx_hash: Some(*tx.tx_hash()),
                        tx_index: Some(index),
                    }),
                    &tx_env,
                    &block_env,
                    output.result(),
                    db,
                )
                .map_err(Self::Error::from_eth_err)?;

            results.push(TraceResult::Success { result, tx_hash: Some(*tx.tx_hash()) });
            if transactions.peek().is_some() {
                inspector.fuse().map_err(Self::Error::from_eth_err)?;
                executor.commit_transaction(output);
            }
        }
        Ok(results)
    }

    fn debug_trace_chain_block_transactions(
        &self,
        db: &mut StateCacheDb,
        block: &RecoveredBlock<ProviderBlock<Self::Provider>>,
        evm_env: EvmEnvFor<Self::Evm>,
        opts: GethDebugTracingOptions,
    ) -> Result<Vec<Option<TraceResult>>, Self::Error> {
        let tx_count = block.body().transactions().len();
        let mut results = Vec::with_capacity(tx_count);
        let block_env = evm_env.block_env.clone();

        let mut transactions = block.transactions_recovered().enumerate().peekable();
        let inspector = match DebugInspector::new(opts) {
            Ok(inspector) => inspector,
            Err(err) => {
                if let Some((_, tx)) = transactions.peek() {
                    results.push(Some(TraceResult::Error {
                        error: err.to_string(),
                        tx_hash: Some(*tx.tx_hash()),
                    }));
                }
                results.resize(tx_count, None);
                return Ok(results);
            }
        };
        let mut executor = inspected_block_executor(self, block, db, evm_env, inspector)?;
        executor.evm_mut().enable_inspector();

        while let Some((index, tx)) = transactions.next() {
            let tx_hash = *tx.tx_hash();
            let tx_env = self.evm_config().tx_env(tx);
            let output = match executor.execute_transaction_without_commit(tx) {
                Ok(output) => output,
                Err(err) => {
                    results.push(Some(TraceResult::Error {
                        error: err.to_string(),
                        tx_hash: Some(tx_hash),
                    }));
                    break;
                }
            };

            let (db, inspector, _) = executor.evm_mut().components_mut();
            let result = match inspector.get_result(
                Some(TransactionContext {
                    block_hash: Some(block.hash()),
                    tx_hash: Some(tx_hash),
                    tx_index: Some(index),
                }),
                &tx_env,
                &block_env,
                output.result(),
                db,
            ) {
                Ok(result) => result,
                Err(err) => {
                    results.push(Some(TraceResult::Error {
                        error: err.to_string(),
                        tx_hash: Some(tx_hash),
                    }));
                    break;
                }
            };

            results.push(Some(TraceResult::Success { result, tx_hash: Some(tx_hash) }));
            if let Some((_, next_tx)) = transactions.peek() {
                if let Err(err) = inspector.fuse() {
                    results.push(Some(TraceResult::Error {
                        error: err.to_string(),
                        tx_hash: Some(*next_tx.tx_hash()),
                    }));
                    break;
                }
                executor.commit_transaction(output);
            }
        }

        results.resize(tx_count, None);
        Ok(results)
    }

    fn block_intermediate_roots(
        &self,
        db: &mut StateCacheDb,
        block: &RecoveredBlock<ProviderBlock<Self::Provider>>,
        _evm_env: EvmEnvFor<Self::Evm>,
    ) -> Result<Vec<B256>, Self::Error> {
        // Enable transition tracking so that merge_transitions works
        db.transition_state = Some(Default::default());

        let mut executor = self
            .evm_config()
            .executor_for_block(db, block.sealed_block())
            .map_err(RethError::other)
            .map_err(Self::Error::from_eth_err)?;
        executor.apply_pre_execution_changes().map_err(Self::Error::from_eth_err)?;

        let mut roots = Vec::with_capacity(block.body().transactions().len());
        for tx in block.transactions_recovered() {
            executor.execute_transaction(tx).map_err(Self::Error::from_eth_err)?;

            let state = executor.evm_mut().db_mut();
            // Merge transitions into cumulative bundle_state
            state.merge_transitions(BundleRetention::PlainState);
            // Compute state root from the accumulated state changes
            let hashed_state = state
                .database
                .hashed_post_state(&state.bundle_state)
                .map_err(Self::Error::from_eth_err)?;
            let root =
                state.database.state_root(hashed_state).map_err(Self::Error::from_eth_err)?;
            roots.push(root);
        }
        Ok(roots)
    }
}

/// Creates a block executor for `block` whose EVM runs the given inspector, and applies the
/// block's pre-execution changes. The inspector is left disabled.
#[expect(clippy::type_complexity, reason = "the executor type is spelled out by its factory")]
fn inspected_block_executor<'a, 'db, Api, I>(
    api: &'a Api,
    block: &'a RecoveredBlock<BlockTy<Api::Primitives>>,
    db: &'db mut StateCacheDb,
    evm_env: EvmEnvFor<Api::Evm>,
    inspector: I,
) -> Result<
    BlockExecutorFor<
        'a,
        <Api::Evm as ConfigureEvm>::BlockExecutorFactory,
        &'db mut StateCacheDb,
        I,
    >,
    Api::Error,
>
where
    Api: Trace,
    I: InspectorFor<Api::Evm, &'db mut StateCacheDb>,
{
    let evm_config = api.evm_config();
    let ctx = evm_config
        .context_for_block(block.sealed_block())
        .map_err(RethError::other)
        .map_err(Api::Error::from_eth_err)?;
    let mut evm = evm_config.evm_with_env_and_inspector(db, evm_env, inspector);
    evm.disable_inspector();

    let mut executor = evm_config.create_executor_with_state(evm, ctx);
    executor.apply_pre_execution_changes().map_err(Api::Error::from_eth_err)?;
    Ok(executor)
}
