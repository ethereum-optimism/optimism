use crate::{
    FlashBlock, FlashBlockCompleteSequence, FlashBlockCompleteSequenceRx, InProgressFlashBlockRx,
    PendingFlashBlock, cache::SequenceManager, worker::FlashBlockBuilder,
};
use alloy_primitives::B256;
use futures_util::{FutureExt, Stream, StreamExt};
use metrics::{Gauge, Histogram};
use op_alloy_rpc_types_engine::OpFlashblockPayloadBase;
use reth_evm::ConfigureEvm;
use reth_metrics::Metrics;
use reth_primitives_traits::{AlloyBlockHeader, BlockTy, HeaderTy, NodePrimitives, ReceiptTy};
use reth_revm::cached::CachedReads;
use reth_storage_api::{BlockReaderIdExt, StateProviderFactory};
use reth_tasks::TaskExecutor;
use std::{
    sync::Arc,
    time::{Duration, Instant},
};
use tokio::{
    sync::{oneshot, watch},
    time::sleep,
};
use tracing::*;

const CONNECTION_BACKOUT_PERIOD: Duration = Duration::from_secs(5);

/// The `FlashBlockService` maintains an in-memory [`PendingFlashBlock`] built out of a sequence of
/// [`FlashBlock`]s.
#[derive(Debug)]
pub struct FlashBlockService<
    N: NodePrimitives,
    S,
    EvmConfig: ConfigureEvm<Primitives = N, NextBlockEnvCtx: From<OpFlashblockPayloadBase> + Unpin>,
    Provider,
> {
    /// Incoming flashblock stream.
    incoming_flashblock_rx: S,
    /// Signals when a block build is in progress.
    in_progress_tx: watch::Sender<Option<FlashBlockBuildInfo>>,
    /// Broadcast channel to forward received flashblocks from the subscription.
    received_flashblocks_tx: tokio::sync::broadcast::Sender<Arc<FlashBlock>>,

    /// Executes flashblock sequences to build pending blocks.
    builder: FlashBlockBuilder<EvmConfig, Provider>,
    /// Task executor for spawning block build jobs.
    spawner: TaskExecutor,
    /// Currently running block build job with start time and result receiver.
    job: Option<BuildJob<N>>,
    /// Manages flashblock sequences with caching and intelligent build selection.
    sequences: SequenceManager<N::SignedTx>,

    /// `FlashBlock` service's metrics
    metrics: FlashBlockServiceMetrics,
}

impl<N, S, EvmConfig, Provider> FlashBlockService<N, S, EvmConfig, Provider>
where
    N: NodePrimitives,
    S: Stream<Item = eyre::Result<FlashBlock>> + Unpin + 'static,
    EvmConfig: ConfigureEvm<Primitives = N, NextBlockEnvCtx: From<OpFlashblockPayloadBase> + Unpin>
        + Clone
        + 'static,
    Provider: StateProviderFactory
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
    pub fn new(
        incoming_flashblock_rx: S,
        evm_config: EvmConfig,
        provider: Provider,
        spawner: TaskExecutor,
        compute_state_root: bool,
    ) -> Self {
        let (in_progress_tx, _) = watch::channel(None);
        let (received_flashblocks_tx, _) = tokio::sync::broadcast::channel(128);
        Self {
            incoming_flashblock_rx,
            in_progress_tx,
            received_flashblocks_tx,
            builder: FlashBlockBuilder::new(evm_config, provider),
            spawner,
            job: None,
            sequences: SequenceManager::new(compute_state_root),
            metrics: FlashBlockServiceMetrics::default(),
        }
    }

    /// Returns the sender half for the received flashblocks broadcast channel.
    pub const fn flashblocks_broadcaster(
        &self,
    ) -> &tokio::sync::broadcast::Sender<Arc<FlashBlock>> {
        &self.received_flashblocks_tx
    }

    /// Returns the sender half for the flashblock sequence broadcast channel.
    pub const fn block_sequence_broadcaster(
        &self,
    ) -> &tokio::sync::broadcast::Sender<FlashBlockCompleteSequence> {
        self.sequences.block_sequence_broadcaster()
    }

    /// Returns a subscriber to the flashblock sequence.
    pub fn subscribe_block_sequence(&self) -> FlashBlockCompleteSequenceRx {
        self.sequences.subscribe_block_sequence()
    }

    /// Returns a receiver that signals when a flashblock is being built.
    pub fn subscribe_in_progress(&self) -> InProgressFlashBlockRx {
        self.in_progress_tx.subscribe()
    }

    /// Drives the service and sends new blocks to the receiver.
    ///
    /// This loop:
    /// 1. Checks if any build job has completed and processes results
    /// 2. Receives and batches all immediately available flashblocks
    /// 3. Attempts to build a block from the complete sequence
    ///
    /// Note: this should be spawned
    pub async fn run(mut self, tx: watch::Sender<Option<PendingFlashBlock<N>>>) {
        loop {
            tokio::select! {
                // Event 1: job exists, listen to job results
                Some(result) = async {
                    match self.job.as_mut() {
                        Some((_, rx)) => rx.await.ok(),
                        None => std::future::pending().await,
                    }
                } => {
                    let (start_time, _) = self.job.take().unwrap();
                    let _ = self.in_progress_tx.send(None);

                    match result {
                        Ok(Some((pending, cached_reads))) => {
                            let parent_hash = pending.parent_hash();
                            self.sequences
                                .on_build_complete(parent_hash, Some((pending.clone(), cached_reads)));

                            let elapsed = start_time.elapsed();
                            self.metrics.execution_duration.record(elapsed.as_secs_f64());

                            let _ = tx.send(Some(pending));
                        }
                        Ok(None) => {
                            trace!(target: "flashblocks", "Build job returned None");
                        }
                        Err(err) => {
                            warn!(target: "flashblocks", %err, "Build job failed");
                        }
                    }
                }

                // Event 2: New flashblock arrives (batch process all ready flashblocks)
                result = self.incoming_flashblock_rx.next() => {
                    match result {
                        Some(Ok(flashblock)) => {
                            // Process first flashblock
                            self.process_flashblock(flashblock);

                            // Batch process all other immediately available flashblocks
                            while let Some(result) = self.incoming_flashblock_rx.next().now_or_never().flatten() {
                                match result {
                                    Ok(fb) => self.process_flashblock(fb),
                                    Err(err) => warn!(target: "flashblocks", %err, "Error receiving flashblock"),
                                }
                            }

                            self.try_start_build_job();
                        }
                        Some(Err(err)) => {
                            warn!(
                                target: "flashblocks",
                                %err,
                                retry_period = CONNECTION_BACKOUT_PERIOD.as_secs(),
                                "Error receiving flashblock"
                            );
                            sleep(CONNECTION_BACKOUT_PERIOD).await;
                        }
                        None => {
                            warn!(target: "flashblocks", "Flashblock stream ended");
                            break;
                        }
                    }
                }
            }
        }
    }

    /// Processes a single flashblock: notifies subscribers, records metrics, and inserts into
    /// sequence.
    fn process_flashblock(&mut self, flashblock: FlashBlock) {
        self.notify_received_flashblock(&flashblock);

        if flashblock.index == 0 {
            self.metrics.last_flashblock_length.record(self.sequences.pending().count() as f64);
        }

        if let Err(err) = self.sequences.insert_flashblock(flashblock) {
            trace!(target: "flashblocks", %err, "Failed to insert flashblock");
        }
    }

    /// Notifies all subscribers about the received flashblock.
    fn notify_received_flashblock(&self, flashblock: &FlashBlock) {
        if self.received_flashblocks_tx.receiver_count() > 0 {
            let _ = self.received_flashblocks_tx.send(Arc::new(flashblock.clone()));
        }
    }

    /// Attempts to build a block if no job is currently running and a buildable sequence exists.
    fn try_start_build_job(&mut self) {
        if self.job.is_some() {
            return; // Already building
        }

        let Some(latest) = self.builder.provider().latest_header().ok().flatten() else {
            return;
        };

        let Some(args) = self.sequences.next_buildable_args(latest.hash(), latest.timestamp())
        else {
            return; // Nothing buildable
        };

        // Spawn build job
        let fb_info = FlashBlockBuildInfo {
            parent_hash: args.base.parent_hash,
            index: args.last_flashblock_index,
            block_number: args.base.block_number,
        };
        self.metrics.current_block_height.set(fb_info.block_number as f64);
        self.metrics.current_index.set(fb_info.index as f64);
        let _ = self.in_progress_tx.send(Some(fb_info));

        let (tx, rx) = oneshot::channel();
        let builder = self.builder.clone();
        self.spawner.spawn_blocking(move || {
            let _ = tx.send(builder.execute(args));
        });
        self.job = Some((Instant::now(), rx));
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
