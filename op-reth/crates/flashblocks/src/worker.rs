use crate::{pending_state::PendingBlockState, tx_cache::TransactionCache, PendingFlashBlock};
use alloy_eips::{eip2718::WithEncoded, BlockNumberOrTag};
use alloy_primitives::B256;
use op_alloy_rpc_types_engine::OpFlashblockPayloadBase;
use reth_chain_state::{ComputedTrieData, ExecutedBlock};
use reth_errors::RethError;
use reth_evm::{
    execute::{BlockBuilder, BlockBuilderOutcome},
    ConfigureEvm,
};
use reth_execution_types::{BlockExecutionOutput, BlockExecutionResult};
use reth_primitives_traits::{
    transaction::TxHashRef, AlloyBlockHeader, BlockTy, HeaderTy, NodePrimitives, ReceiptTy,
    Recovered,
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

pub(crate) struct BuildArgs<I, N: NodePrimitives> {
    pub(crate) base: OpFlashblockPayloadBase,
    pub(crate) transactions: I,
    pub(crate) cached_state: Option<(B256, CachedReads)>,
    pub(crate) last_flashblock_index: u64,
    pub(crate) last_flashblock_hash: B256,
    pub(crate) compute_state_root: bool,
    /// Optional pending parent state for speculative building.
    /// When set, allows building on top of a pending block that hasn't been
    /// canonicalized yet.
    pub(crate) pending_parent: Option<PendingBlockState<N>>,
}

/// Result of a flashblock build operation.
#[derive(Debug)]
pub(crate) struct BuildResult<N: NodePrimitives> {
    /// The built pending flashblock.
    pub(crate) pending_flashblock: PendingFlashBlock<N>,
    /// Cached reads from this build.
    pub(crate) cached_reads: CachedReads,
    /// Pending state that can be used for building subsequent blocks.
    pub(crate) pending_state: PendingBlockState<N>,
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
    /// This method supports two building modes:
    /// 1. **Canonical mode**: Parent matches local tip - uses state from storage
    /// 2. **Speculative mode**: Parent is a pending block - uses pending state
    ///
    /// When a `tx_cache` is provided and we're in canonical mode, the builder will
    /// attempt to resume from cached state if the transaction list is a continuation
    /// of what was previously executed.
    ///
    /// Returns `None` if:
    /// - In canonical mode: flashblock doesn't attach to the latest header
    /// - In speculative mode: no pending parent state provided
    pub(crate) fn execute<I: IntoIterator<Item = WithEncoded<Recovered<N::SignedTx>>>>(
        &self,
        mut args: BuildArgs<I, N>,
        tx_cache: Option<&mut TransactionCache<N>>,
    ) -> eyre::Result<Option<BuildResult<N>>> {
        trace!(target: "flashblocks", "Attempting new pending block from flashblocks");

        let latest = self
            .provider
            .latest_header()?
            .ok_or(EthApiError::HeaderNotFound(BlockNumberOrTag::Latest.into()))?;
        let latest_hash = latest.hash();

        // Determine build mode: canonical (parent is local tip) or speculative (parent is pending)
        let is_canonical = args.base.parent_hash == latest_hash;
        let has_pending_parent = args.pending_parent.is_some();

        if !is_canonical && !has_pending_parent {
            trace!(
                target: "flashblocks",
                flashblock_parent = ?args.base.parent_hash,
                local_latest = ?latest.num_hash(),
                "Skipping non-consecutive flashblock (no pending parent available)"
            );
            return Ok(None);
        }

        // Collect transactions and extract hashes for cache lookup
        let transactions: Vec<_> = args.transactions.into_iter().collect();
        let tx_hashes: Vec<B256> = transactions.iter().map(|tx| *tx.tx_hash()).collect();

        // Get state provider - either from storage or pending state
        let state_provider = if is_canonical {
            self.provider.history_by_block_hash(latest.hash())?
        } else {
            // For speculative building, we need to use the latest available canonical state
            // and apply the pending state's bundle on top of it
            let pending = args.pending_parent.as_ref().unwrap();
            trace!(
                target: "flashblocks",
                pending_block_number = pending.block_number,
                pending_block_hash = ?pending.block_hash,
                "Building speculatively on pending state"
            );
            self.provider.history_by_block_hash(pending.parent_hash)?
        };

        // Set up cached reads
        let cache_key = if is_canonical { latest_hash } else { args.base.parent_hash };
        let mut request_cache = args
            .cached_state
            .take()
            .filter(|(hash, _)| hash == &cache_key)
            .map(|(_, state)| state)
            .unwrap_or_else(|| {
                // For speculative builds, use cached reads from pending parent
                args.pending_parent.as_ref().map(|p| p.cached_reads.clone()).unwrap_or_default()
            });

        let cached_db = request_cache.as_db_mut(StateProviderDatabase::new(&state_provider));

        // Check transaction cache for resumable state (only in canonical mode)
        // In speculative mode, the pending parent's bundle takes precedence
        let (cached_bundle, cached_receipts, skip_count) = if is_canonical {
            tx_cache
                .as_ref()
                .and_then(|cache| cache.get_resumable_state(args.base.block_number, &tx_hashes))
                .map(|(bundle, receipts, skip)| {
                    trace!(
                        target: "flashblocks",
                        skip_count = skip,
                        total_txs = tx_hashes.len(),
                        "Resuming from cached transaction state"
                    );
                    (Some(bundle.clone()), receipts.to_vec(), skip)
                })
                .unwrap_or_default()
        } else {
            Default::default()
        };

        // Build state with appropriate prestate
        let mut state = if let Some(ref pending) = args.pending_parent {
            // Speculative mode - pending parent's bundle as prestate
            State::builder()
                .with_database(cached_db)
                .with_bundle_prestate(pending.execution_outcome.state.clone())
                .with_bundle_update()
                .build()
        } else if let Some(bundle) = cached_bundle {
            // Canonical mode with cached state - use cached bundle as prestate
            State::builder()
                .with_database(cached_db)
                .with_bundle_prestate(bundle)
                .with_bundle_update()
                .build()
        } else {
            // Fresh build from scratch
            State::builder().with_database(cached_db).with_bundle_update().build()
        };

        let mut builder = self
            .evm_config
            .builder_for_next_block(&mut state, &latest, args.base.clone().into())
            .map_err(RethError::other)?;

        builder.apply_pre_execution_changes()?;

        // Execute transactions, skipping those already in cache
        for tx in transactions.into_iter().skip(skip_count) {
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

        // Combine cached receipts with newly executed receipts
        let all_receipts = if skip_count > 0 {
            let mut receipts = cached_receipts;
            receipts.extend(execution_result.receipts);
            receipts
        } else {
            execution_result.receipts
        };

        // Take the bundle before creating execution_outcome (for cache update)
        let bundle = state.take_bundle();

        // Update transaction cache if provided (only in canonical mode)
        if let Some(cache) = tx_cache &&
            is_canonical
        {
            cache.update(args.base.block_number, tx_hashes, bundle.clone(), all_receipts.clone());
        }

        // Update execution_result with combined receipts
        let execution_result = BlockExecutionResult {
            receipts: all_receipts,
            requests: execution_result.requests,
            gas_used: execution_result.gas_used,
            blob_gas_used: execution_result.blob_gas_used,
        };

        let execution_outcome = BlockExecutionOutput { state: bundle, result: execution_result };
        let execution_outcome = Arc::new(execution_outcome);

        // Create pending state for subsequent builds
        let pending_state = PendingBlockState::new(
            block.hash(),
            block.number(),
            args.base.parent_hash,
            execution_outcome.clone(),
            request_cache.clone(),
        );

        let pending_block = PendingBlock::with_executed_block(
            Instant::now() + Duration::from_secs(1),
            ExecutedBlock::new(
                block.into(),
                execution_outcome,
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

        Ok(Some(BuildResult { pending_flashblock, cached_reads: request_cache, pending_state }))
    }
}

impl<EvmConfig: Clone, Provider: Clone> Clone for FlashBlockBuilder<EvmConfig, Provider> {
    fn clone(&self) -> Self {
        Self { evm_config: self.evm_config.clone(), provider: self.provider.clone() }
    }
}
