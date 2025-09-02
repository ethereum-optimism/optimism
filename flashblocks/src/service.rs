use crate::{ExecutionPayloadBaseV1, FlashBlock};
use alloy_eips::{eip2718::WithEncoded, BlockNumberOrTag};
use alloy_primitives::B256;
use futures_util::{FutureExt, Stream, StreamExt};
use reth_chain_state::{CanonStateNotifications, CanonStateSubscriptions, ExecutedBlock};
use reth_errors::RethError;
use reth_evm::{
    execute::{BlockBuilder, BlockBuilderOutcome},
    ConfigureEvm,
};
use reth_execution_types::ExecutionOutcome;
use reth_primitives_traits::{
    AlloyBlockHeader, BlockTy, HeaderTy, NodePrimitives, ReceiptTy, Recovered, SignedTransaction,
};
use reth_revm::{cached::CachedReads, database::StateProviderDatabase, db::State};
use reth_rpc_eth_types::{EthApiError, PendingBlock};
use reth_storage_api::{noop::NoopProvider, BlockReaderIdExt, StateProviderFactory};
use std::{
    collections::BTreeMap,
    pin::Pin,
    sync::Arc,
    task::{Context, Poll},
    time::{Duration, Instant},
};
use tokio::pin;
use tracing::{debug, trace};

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

        // advance new canonical message, if any to reset flashblock
        {
            let fut = this.canon_receiver.recv();
            pin!(fut);
            if fut.poll_unpin(cx).is_ready() {
                // if we have a new canonical message, we know the currently tracked flashblock is
                // invalidated
                if this.current.take().is_some() {
                    trace!("Clearing current flashblock on new canonical block");
                    return Poll::Ready(Some(Ok(None)))
                }
            }
        }

        if !this.rebuild && this.current.is_some() {
            return Poll::Pending;
        }

        // try to build a block on top of latest
        match this.execute() {
            Ok(Some(new_pending)) => {
                // built a new pending block
                this.current = Some(new_pending.clone());
                this.rebuild = false;
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

/// Simple wrapper around an ordered B-tree to keep track of a sequence of flashblocks by index.
#[derive(Debug)]
struct FlashBlockSequence<T> {
    /// tracks the individual flashblocks in order
    ///
    /// With a blocktime of 2s and flashblock tickrate of ~200ms, we expect 10 or 11 flashblocks
    /// per slot.
    inner: BTreeMap<u64, PreparedFlashBlock<T>>,
}

impl<T> FlashBlockSequence<T>
where
    T: SignedTransaction,
{
    const fn new() -> Self {
        Self { inner: BTreeMap::new() }
    }

    /// Inserts a new block into the sequence.
    ///
    /// A [`FlashBlock`] with index 0 resets the set.
    fn insert(&mut self, flashblock: FlashBlock) -> eyre::Result<()> {
        if flashblock.index == 0 {
            trace!(number=%flashblock.block_number(), "Tracking new flashblock sequence");
            // Flash block at index zero resets the whole state
            self.clear();
            self.inner.insert(flashblock.index, PreparedFlashBlock::new(flashblock)?);
            return Ok(())
        }

        // only insert if we we previously received the same block, assume we received index 0
        if self.block_number() == Some(flashblock.metadata.block_number) {
            trace!(number=%flashblock.block_number(), index = %flashblock.index, block_count = self.inner.len()  ,"Received followup flashblock");
            self.inner.insert(flashblock.index, PreparedFlashBlock::new(flashblock)?);
        } else {
            trace!(number=%flashblock.block_number(), index = %flashblock.index, current=?self.block_number()  ,"Ignoring untracked flashblock following");
        }

        Ok(())
    }

    /// Returns the number of tracked flashblocks.
    fn count(&self) -> usize {
        self.inner.len()
    }

    /// Returns the first block number
    fn block_number(&self) -> Option<u64> {
        Some(self.inner.values().next()?.block().metadata.block_number)
    }

    /// Returns the payload base of the first tracked flashblock.
    fn payload_base(&self) -> Option<ExecutionPayloadBaseV1> {
        self.inner.values().next()?.block().base.clone()
    }

    fn clear(&mut self) {
        self.inner.clear();
    }

    /// Iterator over sequence of executable transactions.
    ///
    /// A flashblocks is not ready if there's missing previous flashblocks, i.e. there's a gap in
    /// the sequence
    ///
    /// Note: flashblocks start at `index 0`.
    fn ready_transactions(&self) -> impl Iterator<Item = WithEncoded<Recovered<T>>> + '_ {
        self.inner
            .values()
            .enumerate()
            .take_while(|(idx, block)| {
                // flashblock index 0 is the first flashblock
                block.block().index == *idx as u64
            })
            .flat_map(|(_, block)| block.txs.clone())
    }
}

#[derive(Debug)]
struct PreparedFlashBlock<T> {
    /// The prepared transactions, ready for execution
    txs: Vec<WithEncoded<Recovered<T>>>,
    /// The tracked flashblock
    block: FlashBlock,
}

impl<T> PreparedFlashBlock<T> {
    const fn block(&self) -> &FlashBlock {
        &self.block
    }
}

impl<T> PreparedFlashBlock<T>
where
    T: SignedTransaction,
{
    /// Creates a flashblock that is ready for execution by preparing all transactions
    ///
    /// Returns an error if decoding or signer recovery fails.
    fn new(block: FlashBlock) -> eyre::Result<Self> {
        let mut txs = Vec::with_capacity(block.diff.transactions.len());
        for encoded in block.diff.transactions.iter().cloned() {
            let tx = T::decode_2718_exact(encoded.as_ref())?;
            let signer = tx.try_recover()?;
            let tx = WithEncoded::new(encoded, tx.with_signer(signer));
            txs.push(tx);
        }

        Ok(Self { txs, block })
    }
}
