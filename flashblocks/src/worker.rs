use crate::{ExecutionPayloadBaseV1, PendingFlashBlock};
use alloy_eips::{eip2718::WithEncoded, BlockNumberOrTag};
use alloy_primitives::B256;
use reth_chain_state::{CanonStateSubscriptions, ExecutedBlock};
use reth_errors::RethError;
use reth_evm::{
    execute::{BlockBuilder, BlockBuilderOutcome},
    ConfigureEvm,
};
use reth_execution_types::ExecutionOutcome;
use reth_primitives_traits::{
    AlloyBlockHeader, BlockTy, HeaderTy, NodePrimitives, ReceiptTy, Recovered,
};
use reth_revm::{cached::CachedReads, database::StateProviderDatabase, db::State};
use reth_rpc_eth_types::{EthApiError, PendingBlock};
use reth_storage_api::{noop::NoopProvider, BlockReaderIdExt, StateProviderFactory};
use std::{
    sync::Arc,
    time::{Duration, Instant},
};
use tracing::trace;

/// The `FlashBlockBuilder` builds [`PendingBlock`] out of a sequence of transactions.
#[derive(Debug)]
pub(crate) struct FlashBlockBuilder<EvmConfig, Provider> {
    evm_config: EvmConfig,
    provider: Provider,
}

impl<EvmConfig, Provider> FlashBlockBuilder<EvmConfig, Provider> {
    pub(crate) const fn new(evm_config: EvmConfig, provider: Provider) -> Self {
        Self { evm_config, provider }
    }

    pub(crate) const fn provider(&self) -> &Provider {
        &self.provider
    }
}

pub(crate) struct BuildArgs<I> {
    pub(crate) base: ExecutionPayloadBaseV1,
    pub(crate) transactions: I,
    pub(crate) cached_state: Option<(B256, CachedReads)>,
    pub(crate) last_flashblock_index: u64,
    pub(crate) last_flashblock_hash: B256,
    pub(crate) compute_state_root: bool,
}

impl<N, EvmConfig, Provider> FlashBlockBuilder<EvmConfig, Provider>
where
    N: NodePrimitives,
    EvmConfig: ConfigureEvm<Primitives = N, NextBlockEnvCtx: From<ExecutionPayloadBaseV1> + Unpin>,
    Provider: StateProviderFactory
        + CanonStateSubscriptions<Primitives = N>
        + BlockReaderIdExt<
            Header = HeaderTy<N>,
            Block = BlockTy<N>,
            Transaction = N::SignedTx,
            Receipt = ReceiptTy<N>,
        > + Unpin,
{
    /// Returns the [`PendingFlashBlock`] made purely out of transactions and
    /// [`ExecutionPayloadBaseV1`] in `args`.
    ///
    /// Returns `None` if the flashblock doesn't attach to the latest header.
    pub(crate) fn execute<I: IntoIterator<Item = WithEncoded<Recovered<N::SignedTx>>>>(
        &self,
        mut args: BuildArgs<I>,
    ) -> eyre::Result<Option<(PendingFlashBlock<N>, CachedReads)>> {
        trace!("Attempting new pending block from flashblocks");

        let latest = self
            .provider
            .latest_header()?
            .ok_or(EthApiError::HeaderNotFound(BlockNumberOrTag::Latest.into()))?;
        let latest_hash = latest.hash();

        if args.base.parent_hash != latest_hash {
            trace!(flashblock_parent = ?args.base.parent_hash, local_latest=?latest.num_hash(),"Skipping non consecutive flashblock");
            // doesn't attach to the latest block
            return Ok(None)
        }

        let state_provider = self.provider.history_by_block_hash(latest.hash())?;

        let mut request_cache = args
            .cached_state
            .take()
            .filter(|(hash, _)| hash == &latest_hash)
            .map(|(_, state)| state)
            .unwrap_or_default();
        let cached_db = request_cache.as_db_mut(StateProviderDatabase::new(&state_provider));
        let mut state = State::builder().with_database(cached_db).with_bundle_update().build();

        let mut builder = self
            .evm_config
            .builder_for_next_block(&mut state, &latest, args.base.into())
            .map_err(RethError::other)?;

        builder.apply_pre_execution_changes()?;

        for tx in args.transactions {
            let _gas_used = builder.execute_transaction(tx)?;
        }

        // if the real state root should be computed
        let BlockBuilderOutcome { execution_result, block, hashed_state, .. } =
            if args.compute_state_root {
                builder.finish(&state_provider)?
            } else {
                builder.finish(NoopProvider::default())?
            };

        let execution_outcome = ExecutionOutcome::new(
            state.take_bundle(),
            vec![execution_result.receipts],
            block.number(),
            vec![execution_result.requests],
        );

        let pending_block = PendingBlock::with_executed_block(
            Instant::now() + Duration::from_secs(1),
            ExecutedBlock {
                recovered_block: block.into(),
                execution_output: Arc::new(execution_outcome),
                hashed_state: Arc::new(hashed_state),
            },
        );
        let pending_flashblock = PendingFlashBlock::new(
            pending_block,
            args.last_flashblock_index,
            args.last_flashblock_hash,
            args.compute_state_root,
        );

        Ok(Some((pending_flashblock, request_cache)))
    }
}

impl<EvmConfig: Clone, Provider: Clone> Clone for FlashBlockBuilder<EvmConfig, Provider> {
    fn clone(&self) -> Self {
        Self { evm_config: self.evm_config.clone(), provider: self.provider.clone() }
    }
}
