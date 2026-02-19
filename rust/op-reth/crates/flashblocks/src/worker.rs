use crate::PendingFlashBlock;
use alloy_eips::{BlockNumberOrTag, eip2718::WithEncoded};
use alloy_primitives::B256;
use op_alloy_rpc_types_engine::OpFlashblockPayloadBase;
use reth_chain_state::{ComputedTrieData, ExecutedBlock};
use reth_errors::RethError;
use reth_evm::{
    ConfigureEvm,
    execute::{BlockBuilder, BlockBuilderOutcome},
};
use reth_execution_types::BlockExecutionOutput;
use reth_primitives_traits::{BlockTy, HeaderTy, NodePrimitives, ReceiptTy, Recovered};
use reth_revm::{cached::CachedReads, database::StateProviderDatabase, db::State};
use reth_rpc_eth_types::{EthApiError, PendingBlock};
use reth_storage_api::{BlockReaderIdExt, StateProviderFactory, noop::NoopProvider};
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
    pub(crate) base: OpFlashblockPayloadBase,
    pub(crate) transactions: I,
    pub(crate) cached_state: Option<(B256, CachedReads)>,
    pub(crate) last_flashblock_index: u64,
    pub(crate) last_flashblock_hash: B256,
    pub(crate) compute_state_root: bool,
}

impl<N, EvmConfig, Provider> FlashBlockBuilder<EvmConfig, Provider>
where
    N: NodePrimitives,
    EvmConfig: ConfigureEvm<Primitives = N, NextBlockEnvCtx: From<OpFlashblockPayloadBase> + Unpin>,
    Provider: StateProviderFactory
        + BlockReaderIdExt<
            Header = HeaderTy<N>,
            Block = BlockTy<N>,
            Transaction = N::SignedTx,
            Receipt = ReceiptTy<N>,
        > + Unpin,
{
    /// Returns the [`PendingFlashBlock`] made purely out of transactions and
    /// [`OpFlashblockPayloadBase`] in `args`.
    ///
    /// Returns `None` if the flashblock doesn't attach to the latest header.
    pub(crate) fn execute<I: IntoIterator<Item = WithEncoded<Recovered<N::SignedTx>>>>(
        &self,
        mut args: BuildArgs<I>,
    ) -> eyre::Result<Option<(PendingFlashBlock<N>, CachedReads)>> {
        trace!(target: "flashblocks", "Attempting new pending block from flashblocks");

        let latest = self
            .provider
            .latest_header()?
            .ok_or(EthApiError::HeaderNotFound(BlockNumberOrTag::Latest.into()))?;
        let latest_hash = latest.hash();

        if args.base.parent_hash != latest_hash {
            trace!(target: "flashblocks", flashblock_parent = ?args.base.parent_hash, local_latest=?latest.num_hash(),"Skipping non consecutive flashblock");
            // doesn't attach to the latest block
            return Ok(None);
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
                trace!(target: "flashblocks", "Computing block state root");
                builder.finish(&state_provider)?
            } else {
                builder.finish(NoopProvider::default())?
            };

        let execution_outcome =
            BlockExecutionOutput { state: state.take_bundle(), result: execution_result };

        let pending_block = PendingBlock::with_executed_block(
            Instant::now() + Duration::from_secs(1),
            ExecutedBlock::new(
                block.into(),
                Arc::new(execution_outcome),
                ComputedTrieData::without_trie_input(
                    Arc::new(hashed_state.into_sorted()),
                    Arc::default(),
                ),
            ),
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
