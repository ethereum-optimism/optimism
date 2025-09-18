use crate::{
    sequence::FlashBlockPendingSequence,
    worker::{BuildArgs, FlashBlockBuilder},
    ExecutionPayloadBaseV1, FlashBlock, FlashBlockCompleteSequenceRx,
};
use alloy_eips::eip2718::WithEncoded;
use alloy_primitives::B256;
use futures_util::{FutureExt, Stream, StreamExt};
use reth_chain_state::{CanonStateNotification, CanonStateNotifications, CanonStateSubscriptions};
use reth_evm::ConfigureEvm;
use reth_primitives_traits::{
    AlloyBlockHeader, BlockTy, HeaderTy, NodePrimitives, ReceiptTy, Recovered,
};
use reth_revm::cached::CachedReads;
use reth_rpc_eth_types::PendingBlock;
use reth_storage_api::{BlockReaderIdExt, StateProviderFactory};
use reth_tasks::TaskExecutor;
use std::{
    pin::Pin,
    task::{ready, Context, Poll},
    time::Instant,
};
use tokio::{pin, sync::oneshot};
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
    blocks: FlashBlockPendingSequence<N::SignedTx>,
    rebuild: bool,
    builder: FlashBlockBuilder<EvmConfig, Provider>,
    canon_receiver: CanonStateNotifications<N>,
    spawner: TaskExecutor,
    job: Option<BuildJob<N>>,
    /// Cached state reads for the current block.
    /// Current `PendingBlock` is built out of a sequence of `FlashBlocks`, and executed again when
    /// fb received on top of the same block. Avoid redundant I/O across multiple executions
    /// within the same block.
    cached_state: Option<(B256, CachedReads)>,
}

impl<N, S, EvmConfig, Provider> FlashBlockService<N, S, EvmConfig, Provider>
where
    N: NodePrimitives,
    S: Stream<Item = eyre::Result<FlashBlock>> + Unpin + 'static,
    EvmConfig: ConfigureEvm<Primitives = N, NextBlockEnvCtx: From<ExecutionPayloadBaseV1> + Unpin>
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
        Self {
            rx,
            current: None,
            blocks: FlashBlockPendingSequence::new(),
            canon_receiver: provider.subscribe_to_canonical_state(),
            builder: FlashBlockBuilder::new(evm_config, provider),
            rebuild: false,
            spawner,
            job: None,
            cached_state: None,
        }
    }

    /// Returns a subscriber to the flashblock sequence.
    pub fn subscribe_block_sequence(&self) -> FlashBlockCompleteSequenceRx {
        self.blocks.subscribe_block_sequence()
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

    /// Returns the [`BuildArgs`] made purely out of [`FlashBlock`]s that were received earlier.
    ///
    /// Returns `None` if the flashblock have no `base` or the base is not a child block of latest.
    fn build_args(
        &mut self,
    ) -> Option<BuildArgs<impl IntoIterator<Item = WithEncoded<Recovered<N::SignedTx>>>>> {
        let Some(base) = self.blocks.payload_base() else {
            trace!(
                flashblock_number = ?self.blocks.block_number(),
                count = %self.blocks.count(),
                "Missing flashblock payload base"
            );

            return None
        };

        // attempt an initial consecutive check
        if let Some(latest) = self.builder.provider().latest_header().ok().flatten() {
            if latest.hash() != base.parent_hash {
                trace!(flashblock_parent=?base.parent_hash, flashblock_number=base.block_number, local_latest=?latest.num_hash(), "Skipping non consecutive build attempt");
                return None;
            }
        }

        Some(BuildArgs {
            base,
            transactions: self.blocks.ready_transactions().collect::<Vec<_>>(),
            cached_state: self.cached_state.take(),
        })
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
    S: Stream<Item = eyre::Result<FlashBlock>> + Unpin + 'static,
    EvmConfig: ConfigureEvm<Primitives = N, NextBlockEnvCtx: From<ExecutionPayloadBaseV1> + Unpin>
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
    type Item = eyre::Result<Option<PendingBlock<N>>>;

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

            if let Some((now, result)) = result {
                match result {
                    Ok(Some((new_pending, cached_reads))) => {
                        // built a new pending block
                        this.current = Some(new_pending.clone());
                        // cache reads
                        this.cached_state = Some((new_pending.parent_hash(), cached_reads));
                        this.rebuild = false;

                        trace!(
                            parent_hash = %new_pending.block().parent_hash(),
                            block_number = new_pending.block().number(),
                            flash_blocks = this.blocks.count(),
                            elapsed = ?now.elapsed(),
                            "Built new block with flashblocks"
                        );

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
            }

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

            // update on new head block
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

            // try to build a block on top of latest
            if let Some(args) = this.build_args() {
                let now = Instant::now();

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

type BuildJob<N> =
    (Instant, oneshot::Receiver<eyre::Result<Option<(PendingBlock<N>, CachedReads)>>>);
