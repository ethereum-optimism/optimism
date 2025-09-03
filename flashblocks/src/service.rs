use crate::{sequence::FlashBlockSequence, ExecutionPayloadBaseV1, FlashBlock};
use alloy_eips::BlockNumberOrTag;
use alloy_primitives::B256;
use futures_util::{FutureExt, Stream, StreamExt};
use reth_chain_state::{
    CanonStateNotification, CanonStateNotifications, CanonStateSubscriptions, ExecutedBlock,
};
use reth_errors::RethError;
use reth_evm::{
    execute::{BlockBuilder, BlockBuilderOutcome},
    ConfigureEvm,
};
use reth_execution_types::ExecutionOutcome;
use reth_primitives_traits::{AlloyBlockHeader, BlockTy, HeaderTy, NodePrimitives, ReceiptTy};
use reth_revm::{cached::CachedReads, database::StateProviderDatabase, db::State};
use reth_rpc_eth_types::{EthApiError, PendingBlock};
use reth_storage_api::{noop::NoopProvider, BlockReaderIdExt, StateProviderFactory};
use std::{
    pin::Pin,
    sync::Arc,
    task::{Context, Poll},
    time::{Duration, Instant},
};
use tokio::pin;
use tracing::{debug, trace, warn};

/// The `FlashBlockService` maintains an in-memory [`PendingBlock`] built out of a sequence of
/// [`FlashBlock`]s.
#[derive(Debug)]
pub struct FlashBlockService<
    N: NodePrimitives,
    S,
    EvmConfig: ConfigureEvm<Primitives = N, NextBlockEnvCtx: Unpin>,
    Provider,
> {
    rx: S,
    current: Option<PendingBlock<N>>,
    blocks: FlashBlockSequence<N::SignedTx>,
    rebuild: bool,
    evm_config: EvmConfig,
    provider: Provider,
    canon_receiver: CanonStateNotifications<N>,
    /// Cached state reads for the current block.
    /// Current `PendingBlock` is built out of a sequence of `FlashBlocks`, and executed again when
    /// fb received on top of the same block. Avoid redundant I/O across multiple executions
    /// within the same block.
    cached_state: Option<(B256, CachedReads)>,
}

impl<N, S, EvmConfig, Provider> FlashBlockService<N, S, EvmConfig, Provider>
where
    N: NodePrimitives,
    S: Stream<Item = eyre::Result<FlashBlock>> + Unpin,
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
    /// Constructs a new `FlashBlockService` that receives [`FlashBlock`]s from `rx` stream.
    pub fn new(rx: S, evm_config: EvmConfig, provider: Provider) -> Self {
        Self {
            rx,
            current: None,
            blocks: FlashBlockSequence::new(),
            evm_config,
            canon_receiver: provider.subscribe_to_canonical_state(),
            provider,
            cached_state: None,
            rebuild: false,
        }
    }

    /// Drives the services and sends new blocks to the receiver
    ///
    /// Note: this should be spawned
    pub async fn run(mut self, tx: tokio::sync::watch::Sender<Option<PendingBlock<N>>>) {
        while let Some(block) = self.next().await {
            if let Ok(block) = block.inspect_err(|e| tracing::error!("{e}")) {
                let _ = tx.send(block).inspect_err(|e| tracing::error!("{e}"));
            }
        }

        warn!("Flashblock service has stopped");
    }

    /// Returns the cached reads at the given head hash.
    ///
    /// Returns a new cache instance if this is new `head` hash.
    fn cached_reads(&mut self, head: B256) -> CachedReads {
        if let Some((tracked, cache)) = self.cached_state.take() {
            if tracked == head {
                return cache
            }
        }

        // instantiate a new cache instance
        CachedReads::default()
    }

    /// Updates the cached reads at the given head hash
    fn update_cached_reads(&mut self, head: B256, cached_reads: CachedReads) {
        self.cached_state = Some((head, cached_reads));
    }

    /// Returns the [`ExecutedBlock`] made purely out of [`FlashBlock`]s that were received earlier.
    ///
    /// Returns None if the flashblock doesn't attach to the latest header.
    fn execute(&mut self) -> eyre::Result<Option<PendingBlock<N>>> {
        trace!("Attempting new flashblock");

        let latest = self
            .provider
            .latest_header()?
            .ok_or(EthApiError::HeaderNotFound(BlockNumberOrTag::Latest.into()))?;
        let latest_hash = latest.hash();

        let Some(attrs) = self.blocks.payload_base() else {
            trace!(flashblock_number = ?self.blocks.block_number(), count = %self.blocks.count(), "Missing flashblock payload base");
            return Ok(None)
        };

        if attrs.parent_hash != latest_hash {
            trace!(flashblock_parent = ?attrs.parent_hash, local_latest=?latest.num_hash(),"Skipping non consecutive flashblock");
            // doesn't attach to the latest block
            return Ok(None)
        }

        let state_provider = self.provider.history_by_block_hash(latest.hash())?;

        let mut request_cache = self.cached_reads(latest_hash);
        let cached_db = request_cache.as_db_mut(StateProviderDatabase::new(&state_provider));
        let mut state = State::builder().with_database(cached_db).with_bundle_update().build();

        let mut builder = self
            .evm_config
            .builder_for_next_block(&mut state, &latest, attrs.into())
            .map_err(RethError::other)?;

        builder.apply_pre_execution_changes()?;

        for tx in self.blocks.ready_transactions() {
            let _gas_used = builder.execute_transaction(tx)?;
        }

        let BlockBuilderOutcome { execution_result, block, hashed_state, .. } =
            builder.finish(NoopProvider::default())?;

        let execution_outcome = ExecutionOutcome::new(
            state.take_bundle(),
            vec![execution_result.receipts],
            block.number(),
            vec![execution_result.requests],
        );

        // update cached reads
        self.update_cached_reads(latest_hash, request_cache);

        Ok(Some(PendingBlock::with_executed_block(
            Instant::now() + Duration::from_secs(1),
            ExecutedBlock {
                recovered_block: block.into(),
                execution_output: Arc::new(execution_outcome),
                hashed_state: Arc::new(hashed_state),
            },
        )))
    }

    /// Takes out `current` [`PendingBlock`] if `state` is not preceding it.
    fn on_new_tip(&mut self, state: CanonStateNotification<N>) -> Option<PendingBlock<N>> {
        let latest = state.tip_checked()?.hash();
        self.current.take_if(|current| current.parent_hash() != latest)
    }
}

impl<N, S, EvmConfig, Provider> Stream for FlashBlockService<N, S, EvmConfig, Provider>
where
    N: NodePrimitives,
    S: Stream<Item = eyre::Result<FlashBlock>> + Unpin,
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
    type Item = eyre::Result<Option<PendingBlock<N>>>;

    fn poll_next(self: Pin<&mut Self>, cx: &mut Context<'_>) -> Poll<Option<Self::Item>> {
        let this = self.get_mut();

        // consume new flashblocks while they're ready
        while let Poll::Ready(Some(result)) = this.rx.poll_next_unpin(cx) {
            match result {
                Ok(flashblock) => match this.blocks.insert(flashblock) {
                    Ok(_) => this.rebuild = true,
                    Err(err) => debug!(%err, "Failed to prepare flashblock"),
                },
                Err(err) => return Poll::Ready(Some(Err(err))),
            }
        }

        if let Poll::Ready(Ok(state)) = {
            let fut = this.canon_receiver.recv();
            pin!(fut);
            fut.poll_unpin(cx)
        } {
            if let Some(current) = this.on_new_tip(state) {
                trace!(
                    parent_hash = %current.block().parent_hash(),
                    block_number = current.block().number(),
                    "Clearing current flashblock on new canonical block"
                );

                return Poll::Ready(Some(Ok(None)))
            }
        }

        if !this.rebuild && this.current.is_some() {
            return Poll::Pending
        }

        let now = Instant::now();
        // try to build a block on top of latest
        match this.execute() {
            Ok(Some(new_pending)) => {
                // built a new pending block
                this.current = Some(new_pending.clone());
                this.rebuild = false;
                trace!(parent_hash=%new_pending.block().parent_hash(), block_number=new_pending.block().number(), flash_blocks=this.blocks.count(), elapsed=?now.elapsed(), "Built new block with flashblocks");
                return Poll::Ready(Some(Ok(Some(new_pending))));
            }
            Ok(None) => {
                // nothing to do because tracked flashblock doesn't attach to latest
            }
            Err(err) => {
                // we can ignore this error
                debug!(%err, "failed to execute flashblock");
            }
        }

        Poll::Pending
    }
}
