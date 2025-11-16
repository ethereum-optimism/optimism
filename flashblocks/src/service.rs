use crate::{
    sequence::{FlashBlockPendingSequence, SequenceExecutionOutcome},
    worker::{BuildArgs, FlashBlockBuilder},
    FlashBlock, FlashBlockCompleteSequence, FlashBlockCompleteSequenceRx, InProgressFlashBlockRx,
    PendingFlashBlock,
};
use alloy_eips::eip2718::WithEncoded;
use alloy_primitives::B256;
use futures_util::{FutureExt, Stream, StreamExt};
use metrics::{Gauge, Histogram};
use op_alloy_rpc_types_engine::OpFlashblockPayloadBase;
use reth_chain_state::{CanonStateNotification, CanonStateNotifications, CanonStateSubscriptions};
use reth_evm::ConfigureEvm;
use reth_metrics::Metrics;
use reth_primitives_traits::{
    AlloyBlockHeader, BlockTy, HeaderTy, NodePrimitives, ReceiptTy, Recovered,
};
use reth_revm::cached::CachedReads;
use reth_storage_api::{BlockReaderIdExt, StateProviderFactory};
use reth_tasks::TaskExecutor;
use std::{
    pin::Pin,
    sync::Arc,
    task::{ready, Context, Poll},
    time::Instant,
};
use tokio::{
    pin,
    sync::{oneshot, watch},
};
use tracing::{debug, trace, warn};

/// 200 ms flashblock time.
pub(crate) const FLASHBLOCK_BLOCK_TIME: u64 = 200;

/// The `FlashBlockService` maintains an in-memory [`PendingFlashBlock`] built out of a sequence of
/// [`FlashBlock`]s.
#[derive(Debug)]
pub struct FlashBlockService<
    N: NodePrimitives,
    S,
    EvmConfig: ConfigureEvm<Primitives = N, NextBlockEnvCtx: From<OpFlashblockPayloadBase> + Unpin>,
    Provider,
> {
    rx: S,
    current: Option<PendingFlashBlock<N>>,
    blocks: FlashBlockPendingSequence<N::SignedTx>,
    /// Broadcast channel to forward received flashblocks from the subscription.
    received_flashblocks_tx: tokio::sync::broadcast::Sender<Arc<FlashBlock>>,
    rebuild: bool,
    builder: FlashBlockBuilder<EvmConfig, Provider>,
    canon_receiver: CanonStateNotifications<N>,
    spawner: TaskExecutor,
    job: Option<BuildJob<N>>,
    /// Cached state reads for the current block.
    /// Current `PendingFlashBlock` is built out of a sequence of `FlashBlocks`, and executed again
    /// when fb received on top of the same block. Avoid redundant I/O across multiple
    /// executions within the same block.
    cached_state: Option<(B256, CachedReads)>,
    /// Signals when a block build is in progress
    in_progress_tx: watch::Sender<Option<FlashBlockBuildInfo>>,
    /// `FlashBlock` service's metrics
    metrics: FlashBlockServiceMetrics,
    /// Enable state root calculation
    compute_state_root: bool,
}

impl<N, S, EvmConfig, Provider> FlashBlockService<N, S, EvmConfig, Provider>
where
    N: NodePrimitives,
    S: Stream<Item = eyre::Result<FlashBlock>> + Unpin + 'static,
    EvmConfig: ConfigureEvm<Primitives = N, NextBlockEnvCtx: From<OpFlashblockPayloadBase> + Unpin>
        + Clone
        + 'static,
    Provider: StateProviderFactory
        + CanonStateSubscriptions<Primitives = N>
        + BlockReaderIdExt<
            Header = HeaderTy<N>,
            Block = BlockTy<N>,
            Transaction = N::SignedTx,
            Receipt = ReceiptTy<N>,
        > + Unpin
        + Clone
        + 'static,
{
    /// Constructs a new `FlashBlockService` that receives [`FlashBlock`]s from `rx` stream.
    pub fn new(rx: S, evm_config: EvmConfig, provider: Provider, spawner: TaskExecutor) -> Self {
        let (in_progress_tx, _) = watch::channel(None);
        let (received_flashblocks_tx, _) = tokio::sync::broadcast::channel(128);
        Self {
            rx,
            current: None,
            blocks: FlashBlockPendingSequence::new(),
            received_flashblocks_tx,
            canon_receiver: provider.subscribe_to_canonical_state(),
            builder: FlashBlockBuilder::new(evm_config, provider),
            rebuild: false,
            spawner,
            job: None,
            cached_state: None,
            in_progress_tx,
            metrics: FlashBlockServiceMetrics::default(),
            compute_state_root: false,
        }
    }

    /// Enable state root calculation from flashblock
    pub const fn compute_state_root(mut self, enable_state_root: bool) -> Self {
        self.compute_state_root = enable_state_root;
        self
    }

    /// Returns the sender half to the received flashblocks.
    pub const fn flashblocks_broadcaster(
        &self,
    ) -> &tokio::sync::broadcast::Sender<Arc<FlashBlock>> {
        &self.received_flashblocks_tx
    }

    /// Returns the sender half to the flashblock sequence.
    pub const fn block_sequence_broadcaster(
        &self,
    ) -> &tokio::sync::broadcast::Sender<FlashBlockCompleteSequence> {
        self.blocks.block_sequence_broadcaster()
    }

    /// Returns a subscriber to the flashblock sequence.
    pub fn subscribe_block_sequence(&self) -> FlashBlockCompleteSequenceRx {
        self.blocks.subscribe_block_sequence()
    }

    /// Returns a receiver that signals when a flashblock is being built.
    pub fn subscribe_in_progress(&self) -> InProgressFlashBlockRx {
        self.in_progress_tx.subscribe()
    }

    /// Drives the services and sends new blocks to the receiver
    ///
    /// Note: this should be spawned
    pub async fn run(mut self, tx: tokio::sync::watch::Sender<Option<PendingFlashBlock<N>>>) {
        while let Some(block) = self.next().await {
            if let Ok(block) = block.inspect_err(|e| tracing::error!(target: "flashblocks", "{e}"))
            {
                let _ =
                    tx.send(block).inspect_err(|e| tracing::error!(target: "flashblocks", "{e}"));
            }
        }

        warn!(target: "flashblocks", "Flashblock service has stopped");
    }

    /// Notifies all subscribers about the received flashblock
    fn notify_received_flashblock(&self, flashblock: &FlashBlock) {
        if self.received_flashblocks_tx.receiver_count() > 0 {
            let _ = self.received_flashblocks_tx.send(Arc::new(flashblock.clone()));
        }
    }

    /// Returns the [`BuildArgs`] made purely out of [`FlashBlock`]s that were received earlier.
    ///
    /// Returns `None` if the flashblock have no `base` or the base is not a child block of latest.
    fn build_args(
        &mut self,
    ) -> Option<
        BuildArgs<
            impl IntoIterator<Item = WithEncoded<Recovered<N::SignedTx>>>
                + use<N, S, EvmConfig, Provider>,
        >,
    > {
        let Some(base) = self.blocks.payload_base() else {
            trace!(
                target: "flashblocks",
                flashblock_number = ?self.blocks.block_number(),
                count = %self.blocks.count(),
                "Missing flashblock payload base"
            );

            return None
        };

        let Some(latest) = self.builder.provider().latest_header().ok().flatten() else {
            trace!(target: "flashblocks", "No latest header found");
            return None
        };

        // attempt an initial consecutive check
        if latest.hash() != base.parent_hash {
            trace!(target: "flashblocks", flashblock_parent=?base.parent_hash, flashblock_number=base.block_number, local_latest=?latest.num_hash(), "Skipping non consecutive build attempt");
            return None
        }

        let Some(last_flashblock) = self.blocks.last_flashblock() else {
            trace!(target: "flashblocks", flashblock_number = ?self.blocks.block_number(), count = %self.blocks.count(), "Missing last flashblock");
            return None
        };

        // Auto-detect when to compute state root: only if the builder didn't provide it (sent
        // B256::ZERO) and we're near the expected final flashblock index.
        //
        // Background: Each block period receives multiple flashblocks at regular intervals.
        // The sequencer sends an initial "base" flashblock at index 0 when a new block starts,
        // then subsequent flashblocks are produced every FLASHBLOCK_BLOCK_TIME intervals (200ms).
        //
        // Examples with different block times:
        // - Base (2s blocks):    expect 2000ms / 200ms = 10 intervals → Flashblocks: index 0 (base)
        //   + indices 1-10 = potentially 11 total
        //
        // - Unichain (1s blocks): expect 1000ms / 200ms = 5 intervals → Flashblocks: index 0 (base)
        //   + indices 1-5 = potentially 6 total
        //
        // Why compute at N-1 instead of N:
        // 1. Timing variance in flashblock producing time may mean only N flashblocks were produced
        //    instead of N+1 (missing the final one). Computing at N-1 ensures we get the state root
        //    for most common cases.
        //
        // 2. The +1 case (index 0 base + N intervals): If all N+1 flashblocks do arrive, we'll
        //    still calculate state root for flashblock N, which sacrifices a little performance but
        //    still ensures correctness for common cases.
        //
        // Note: Pathological cases may result in fewer flashblocks than expected (e.g., builder
        // downtime, flashblock execution exceeding timing budget). When this occurs, we won't
        // compute the state root, causing FlashblockConsensusClient to lack precomputed state for
        // engine_newPayload. This is safe: we still have op-node as backstop to maintain
        // chain progression.
        let block_time_ms = (base.timestamp - latest.timestamp()) * 1000;
        let expected_final_flashblock = block_time_ms / FLASHBLOCK_BLOCK_TIME;
        let compute_state_root = self.compute_state_root &&
            last_flashblock.diff.state_root.is_zero() &&
            self.blocks.index() >= Some(expected_final_flashblock.saturating_sub(1));

        Some(BuildArgs {
            base,
            transactions: self.blocks.ready_transactions().collect::<Vec<_>>(),
            cached_state: self.cached_state.take(),
            last_flashblock_index: last_flashblock.index,
            last_flashblock_hash: last_flashblock.diff.block_hash,
            compute_state_root,
        })
    }

    /// Takes out `current` [`PendingFlashBlock`] if `state` is not preceding it.
    fn on_new_tip(&mut self, state: CanonStateNotification<N>) -> Option<PendingFlashBlock<N>> {
        let tip = state.tip_checked()?;
        let tip_hash = tip.hash();
        let current = self.current.take_if(|current| current.parent_hash() != tip_hash);

        // Prefill the cache with state from the new canonical tip, similar to payload/basic
        let mut cached = CachedReads::default();
        let committed = state.committed();
        let new_execution_outcome = committed.execution_outcome();
        for (addr, acc) in new_execution_outcome.bundle_accounts_iter() {
            if let Some(info) = acc.info.clone() {
                // Pre-cache existing accounts and their storage (only changed accounts/storage)
                let storage =
                    acc.storage.iter().map(|(key, slot)| (*key, slot.present_value)).collect();
                cached.insert_account(addr, info, storage);
            }
        }
        self.cached_state = Some((tip_hash, cached));

        current
    }
}

impl<N, S, EvmConfig, Provider> Stream for FlashBlockService<N, S, EvmConfig, Provider>
where
    N: NodePrimitives,
    S: Stream<Item = eyre::Result<FlashBlock>> + Unpin + 'static,
    EvmConfig: ConfigureEvm<Primitives = N, NextBlockEnvCtx: From<OpFlashblockPayloadBase> + Unpin>
        + Clone
        + 'static,
    Provider: StateProviderFactory
        + CanonStateSubscriptions<Primitives = N>
        + BlockReaderIdExt<
            Header = HeaderTy<N>,
            Block = BlockTy<N>,
            Transaction = N::SignedTx,
            Receipt = ReceiptTy<N>,
        > + Unpin
        + Clone
        + 'static,
{
    type Item = eyre::Result<Option<PendingFlashBlock<N>>>;

    fn poll_next(self: Pin<&mut Self>, cx: &mut Context<'_>) -> Poll<Option<Self::Item>> {
        let this = self.get_mut();

        loop {
            // drive pending build job to completion
            let result = match this.job.as_mut() {
                Some((now, rx)) => {
                    let result = ready!(rx.poll_unpin(cx));
                    result.ok().map(|res| (*now, res))
                }
                None => None,
            };
            // reset job
            this.job.take();
            // No build in progress
            let _ = this.in_progress_tx.send(None);

            if let Some((now, result)) = result {
                match result {
                    Ok(Some((new_pending, cached_reads))) => {
                        // update execution outcome of the current sequence
                        let execution_outcome =
                            new_pending.computed_state_root().map(|state_root| {
                                SequenceExecutionOutcome {
                                    block_hash: new_pending.block().hash(),
                                    state_root,
                                }
                            });
                        this.blocks.set_execution_outcome(execution_outcome);

                        // built a new pending block
                        this.current = Some(new_pending.clone());
                        // cache reads
                        this.cached_state = Some((new_pending.parent_hash(), cached_reads));
                        this.rebuild = false;

                        let elapsed = now.elapsed();
                        this.metrics.execution_duration.record(elapsed.as_secs_f64());

                        trace!(
                            target: "flashblocks",
                            parent_hash = %new_pending.block().parent_hash(),
                            block_number = new_pending.block().number(),
                            flash_blocks = this.blocks.count(),
                            ?elapsed,
                            "Built new block with flashblocks"
                        );

                        return Poll::Ready(Some(Ok(Some(new_pending))));
                    }
                    Ok(None) => {
                        // nothing to do because tracked flashblock doesn't attach to latest
                    }
                    Err(err) => {
                        // we can ignore this error
                        debug!(target: "flashblocks", %err, "failed to execute flashblock");
                    }
                }
            }

            // consume new flashblocks while they're ready
            while let Poll::Ready(Some(result)) = this.rx.poll_next_unpin(cx) {
                match result {
                    Ok(flashblock) => {
                        this.notify_received_flashblock(&flashblock);
                        if flashblock.index == 0 {
                            this.metrics.last_flashblock_length.record(this.blocks.count() as f64);
                        }
                        match this.blocks.insert(flashblock) {
                            Ok(_) => this.rebuild = true,
                            Err(err) => {
                                debug!(target: "flashblocks", %err, "Failed to prepare flashblock")
                            }
                        }
                    }
                    Err(err) => return Poll::Ready(Some(Err(err))),
                }
            }

            // update on new head block
            if let Poll::Ready(Ok(state)) = {
                let fut = this.canon_receiver.recv();
                pin!(fut);
                fut.poll_unpin(cx)
            } && let Some(current) = this.on_new_tip(state)
            {
                trace!(
                    target: "flashblocks",
                    parent_hash = %current.block().parent_hash(),
                    block_number = current.block().number(),
                    "Clearing current flashblock on new canonical block"
                );

                return Poll::Ready(Some(Ok(None)))
            }

            if !this.rebuild && this.current.is_some() {
                return Poll::Pending
            }

            // try to build a block on top of latest
            if let Some(args) = this.build_args() {
                let now = Instant::now();

                let fb_info = FlashBlockBuildInfo {
                    parent_hash: args.base.parent_hash,
                    index: args.last_flashblock_index,
                    block_number: args.base.block_number,
                };
                // Record current block and index metrics
                this.metrics.current_block_height.set(fb_info.block_number as f64);
                this.metrics.current_index.set(fb_info.index as f64);
                // Signal that a flashblock build has started with build metadata
                let _ = this.in_progress_tx.send(Some(fb_info));
                let (tx, rx) = oneshot::channel();
                let builder = this.builder.clone();

                this.spawner.spawn_blocking(async move {
                    let _ = tx.send(builder.execute(args));
                });
                this.job.replace((now, rx));

                // continue and poll the spawned job
                continue
            }

            return Poll::Pending
        }
    }
}

/// Information for a flashblock currently built
#[derive(Debug, Clone, Copy)]
pub struct FlashBlockBuildInfo {
    /// Parent block hash
    pub parent_hash: B256,
    /// Flashblock index within the current block's sequence
    pub index: u64,
    /// Block number of the flashblock being built.
    pub block_number: u64,
}

type BuildJob<N> =
    (Instant, oneshot::Receiver<eyre::Result<Option<(PendingFlashBlock<N>, CachedReads)>>>);

#[derive(Metrics)]
#[metrics(scope = "flashblock_service")]
struct FlashBlockServiceMetrics {
    /// The last complete length of flashblocks per block.
    last_flashblock_length: Histogram,
    /// The duration applying flashblock state changes in seconds.
    execution_duration: Histogram,
    /// Current block height.
    current_block_height: Gauge,
    /// Current flashblock index.
    current_index: Gauge,
}
