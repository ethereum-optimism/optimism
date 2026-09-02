//! This module contains the `BatchStream` stage.

use crate::{
    L2ChainProvider, NextBatchProvider, OriginAdvancer, OriginProvider, PipelineError,
    PipelineResult, Stage, StagedBatch,
};
use alloc::{boxed::Box, collections::VecDeque, sync::Arc};
use alloy_eips::BlockNumHash;
use async_trait::async_trait;
use core::fmt::Debug;
use kona_genesis::{RollupConfig, SystemConfig};
use kona_protocol::{
    Batch, BatchValidity, BatchWithInclusionBlock, BlockInfo, L2BlockInfo, SpanBatch,
    SpanBatchError, SpanBatchOutcome, SpanSingleBatch,
};

/// Provides [`Batch`]es for the [`BatchStream`] stage.
#[async_trait]
pub trait BatchStreamProvider {
    /// Returns the next [`Batch`] in the [`BatchStream`] stage.
    async fn next_batch(&mut self) -> PipelineResult<Batch>;

    /// Drains the recent `Channel` if an invalid span batch is found post-holocene.
    fn flush(&mut self);
}

/// A [`SpanBatch`] the prefix checks accepted, with the block its first element builds on.
///
/// The parent is what turns element indices into block numbers, which is the only way to tell a
/// sibling apart from the block it shares a timestamp with.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct StagedSpan {
    /// The staged span batch.
    pub batch: SpanBatch,
    /// The block the span's first element builds on.
    pub parent: L2BlockInfo,
}

/// [`BatchStream`] stage in the derivation pipeline.
///
/// This stage is introduced in the [`Holocene`] hardfork.
/// It slots in between the [`ChannelReader`] and [`BatchQueue`]
/// stages, buffering span batches until they are validated.
///
/// [`Holocene`]: https://specs.optimism.io/protocol/holocene/overview.html
/// [`ChannelReader`]: crate::stages::ChannelReader
/// [`BatchQueue`]: crate::stages::BatchQueue
#[derive(Debug)]
pub struct BatchStream<P, BF>
where
    P: BatchStreamProvider + OriginAdvancer + OriginProvider + Stage + Debug,
    BF: L2ChainProvider + Debug,
{
    /// The previous stage in the derivation pipeline.
    pub prev: P,
    /// There can only be a single staged span batch.
    pub span: Option<StagedSpan>,
    /// A buffer of single batches derived from the [`SpanBatch`].
    pub buffer: VecDeque<SpanSingleBatch>,
    /// A reference to the rollup config, used to check
    /// if the [`BatchStream`] stage should be activated.
    pub config: Arc<RollupConfig>,
    /// Used to validate the batches.
    pub fetcher: BF,
}

impl<P, BF> BatchStream<P, BF>
where
    P: BatchStreamProvider + OriginAdvancer + OriginProvider + Stage + Debug,
    BF: L2ChainProvider + Debug,
{
    /// Create a new [`BatchStream`] stage.
    pub const fn new(prev: P, config: Arc<RollupConfig>, fetcher: BF) -> Self {
        Self { prev, span: None, buffer: VecDeque::new(), config, fetcher }
    }

    /// Returns if the [`BatchStream`] stage is active based on the
    /// origin timestamp and holocene activation timestamp.
    pub fn is_active(&self) -> PipelineResult<bool> {
        let origin = self.prev.origin().ok_or(PipelineError::MissingOrigin.crit())?;
        Ok(self.config.is_holocene_active(origin.timestamp))
    }

    /// Gets a [`SpanSingleBatch`] from the in-memory buffer.
    pub fn get_single_batch(
        &mut self,
        parent: L2BlockInfo,
        l1_origins: &[BlockInfo],
    ) -> Result<Option<SpanSingleBatch>, SpanBatchError> {
        trace!(target: "batch_span", "Attempting to get a SingleBatch from buffer len: {}", self.buffer.len());

        self.try_hydrate_buffer(parent, l1_origins)?;
        Ok(self.buffer.pop_front())
    }

    /// Hydrates the buffer with single batches derived from the span batch, if there is one
    /// queued up.
    pub fn try_hydrate_buffer(
        &mut self,
        parent: L2BlockInfo,
        l1_origins: &[BlockInfo],
    ) -> Result<(), SpanBatchError> {
        if let Some(StagedSpan { batch, parent: span_parent }) = self.span.take() {
            self.buffer.extend(batch.get_singular_batches(
                l1_origins,
                parent,
                span_parent.block_info.number,
            )?);
        }
        #[cfg(feature = "metrics")]
        {
            let batch_count = self.buffer.len() as f64;
            kona_macros::set!(gauge, crate::metrics::Metrics::PIPELINE_BATCH_BUFFER, batch_count);
            let batch_size = std::mem::size_of_val(&self.buffer) as f64;
            kona_macros::set!(gauge, crate::metrics::Metrics::PIPELINE_BATCH_MEM, batch_size);
        }
        Ok(())
    }
}

#[async_trait]
impl<P, BF> NextBatchProvider for BatchStream<P, BF>
where
    P: BatchStreamProvider + OriginAdvancer + OriginProvider + Stage + Send + Debug,
    BF: L2ChainProvider + Send + Debug,
{
    fn flush(&mut self) {
        if self.is_active().unwrap_or(false) {
            self.prev.flush();
            self.span = None;
            self.buffer.clear();
        }
    }

    fn span_buffer_size(&self) -> usize {
        self.buffer.len()
    }

    async fn next_batch(
        &mut self,
        parent: L2BlockInfo,
        l1_origins: &[BlockInfo],
    ) -> PipelineResult<StagedBatch<Batch>> {
        // If the stage is not active, "pass" the next batch
        // through this stage to the BatchQueue stage.
        if !self.is_active()? {
            trace!(target: "batch_span", "BatchStream stage is inactive, pass-through.");
            return self.prev.next_batch().await.map(StagedBatch::new);
        }

        // If the buffer is empty, attempt to pull a batch from the previous stage.
        if self.buffer.is_empty() {
            // Safety: bubble up any errors from the batch reader.
            let batch_with_inclusion = BatchWithInclusionBlock::new(
                self.origin().ok_or(PipelineError::MissingOrigin.crit())?,
                self.prev.next_batch().await?,
            );

            // If the next batch is a singular batch, it is immediately
            // forwarded to the `BatchQueue` stage. Otherwise, we buffer
            // the span batch in this stage if it passes the validity checks.
            match batch_with_inclusion.batch {
                Batch::Single(b) => return Ok(StagedBatch::new(Batch::Single(b))),
                Batch::Span(b) => {
                    #[cfg(feature = "metrics")]
                    let start = std::time::Instant::now();
                    let outcome = b
                        .check_batch_holocene(
                            self.config.as_ref(),
                            l1_origins,
                            parent,
                            &batch_with_inclusion.inclusion_block,
                            &mut self.fetcher,
                        )
                        .await;
                    kona_macros::record!(
                        histogram,
                        crate::metrics::Metrics::PIPELINE_CHECK_BATCH_PREFIX,
                        start.elapsed().as_secs_f64()
                    );

                    kona_macros::inc!(
                        gauge,
                        crate::metrics::Metrics::PIPELINE_BATCH_VALIDITY,
                        "validity" => outcome.validity().to_string(),
                    );

                    match outcome {
                        SpanBatchOutcome::Accepted(span_parent) => {
                            self.span = Some(StagedSpan { batch: b, parent: span_parent });
                        }
                        SpanBatchOutcome::Rejected(BatchValidity::Drop(_)) => {
                            // Flush the stage.
                            self.flush();

                            return Err(PipelineError::NotEnoughData.temp());
                        }
                        SpanBatchOutcome::Rejected(BatchValidity::Past) => {
                            if !self.is_active()? {
                                error!(target: "batch_stream", "BatchValidity::Past is not allowed pre-holocene");
                                return Err(PipelineError::InvalidBatchValidity.crit());
                            }

                            return Err(PipelineError::NotEnoughData.temp());
                        }
                        SpanBatchOutcome::Rejected(_) => {
                            // Undecided: the span was already consumed and is skipped, not
                            // retried.
                            return Err(PipelineError::NotEnoughData.temp());
                        }
                    }
                }
            }
        }

        // Attempt to pull a SingleBatch out of the SpanBatch.
        match self.get_single_batch(parent, l1_origins) {
            Ok(Some(SpanSingleBatch { batch, is_sibling })) => {
                Ok(StagedBatch { batch: Batch::Single(batch), is_sibling })
            }
            Ok(None) => Err(PipelineError::NotEnoughData.temp()),
            Err(e) => {
                warn!(target: "batch_span", "Extracting singular batches from span batch failed: {}", e);
                // If singular batch extraction fails, handle it like a batch dropped during the
                // Holocene span batch checks.
                self.flush();
                Err(PipelineError::NotEnoughData.temp())
            }
        }
    }
}

#[async_trait]
impl<P, BF> OriginAdvancer for BatchStream<P, BF>
where
    P: BatchStreamProvider + OriginAdvancer + OriginProvider + Stage + Send + Debug,
    BF: L2ChainProvider + Send + Debug,
{
    async fn advance_origin(&mut self) -> PipelineResult<()> {
        self.prev.advance_origin().await
    }
}

impl<P, BF> OriginProvider for BatchStream<P, BF>
where
    P: BatchStreamProvider + OriginAdvancer + OriginProvider + Stage + Debug,
    BF: L2ChainProvider + Debug,
{
    fn origin(&self) -> Option<BlockInfo> {
        self.prev.origin()
    }
}

#[async_trait]
impl<P, BF> Stage for BatchStream<P, BF>
where
    P: BatchStreamProvider + OriginAdvancer + OriginProvider + Stage + Debug + Send,
    BF: L2ChainProvider + Send + Debug,
{
    async fn reset(
        &mut self,
        l1_origin: BlockNumHash,
        system_config: SystemConfig,
    ) -> PipelineResult<()> {
        self.prev.reset(l1_origin, system_config).await?;
        self.buffer.clear();
        self.span.take();
        Ok(())
    }

    async fn activate(&mut self) -> PipelineResult<()> {
        self.prev.activate().await?;
        self.buffer.clear();
        self.span.take();
        Ok(())
    }

    async fn flush_channel(&mut self) -> PipelineResult<()> {
        self.prev.flush_channel().await?;
        self.buffer.clear();
        self.span.take();
        Ok(())
    }
}

#[cfg(test)]
mod test {
    use super::*;
    use crate::test_utils::{
        CollectingLayer, TestBatchStreamProvider, TestL2ChainProvider, TraceStorage,
    };
    use alloc::vec;
    use alloy_consensus::{BlockBody, Header};
    use alloy_eips::{BlockNumHash, NumHash};
    use alloy_primitives::{FixedBytes, b256};
    use kona_genesis::{ChainGenesis, HardForkConfig};
    use kona_protocol::{SingleBatch, SpanBatchElement};
    use op_alloy_consensus::OpBlock;
    use tracing_subscriber::layer::SubscriberExt;

    #[tokio::test]
    async fn test_batch_stream_flush() {
        let config = Arc::new(RollupConfig {
            hardforks: HardForkConfig { holocene_time: Some(0), ..Default::default() },
            ..Default::default()
        });
        let prev = TestBatchStreamProvider::new(vec![]);
        let mut stream = BatchStream::new(prev, config, TestL2ChainProvider::default());
        stream
            .buffer
            .push_back(SpanSingleBatch { batch: SingleBatch::default(), is_sibling: false });
        stream.span =
            Some(StagedSpan { batch: SpanBatch::default(), parent: L2BlockInfo::default() });
        assert!(!stream.buffer.is_empty());
        assert!(stream.span.is_some());
        stream.flush();
        assert!(stream.buffer.is_empty());
        assert!(stream.span.is_none());
    }

    #[tokio::test]
    async fn test_batch_stream_reset() {
        let config = Arc::new(RollupConfig {
            hardforks: HardForkConfig { holocene_time: Some(0), ..Default::default() },
            ..Default::default()
        });
        let prev = TestBatchStreamProvider::new(vec![]);
        let mut stream = BatchStream::new(prev, config.clone(), TestL2ChainProvider::default());
        stream
            .buffer
            .push_back(SpanSingleBatch { batch: SingleBatch::default(), is_sibling: false });
        stream.span =
            Some(StagedSpan { batch: SpanBatch::default(), parent: L2BlockInfo::default() });
        assert!(!stream.prev.reset);
        stream.reset(BlockNumHash::default(), SystemConfig::default()).await.unwrap();
        assert!(stream.prev.reset);
        assert!(stream.buffer.is_empty());
        assert!(stream.span.is_none());
    }

    #[tokio::test]
    async fn test_batch_stream_flush_channel() {
        let config = Arc::new(RollupConfig {
            hardforks: HardForkConfig { holocene_time: Some(0), ..Default::default() },
            ..Default::default()
        });
        let prev = TestBatchStreamProvider::new(vec![]);
        let mut stream = BatchStream::new(prev, config.clone(), TestL2ChainProvider::default());
        stream
            .buffer
            .push_back(SpanSingleBatch { batch: SingleBatch::default(), is_sibling: false });
        stream.span =
            Some(StagedSpan { batch: SpanBatch::default(), parent: L2BlockInfo::default() });
        assert!(!stream.prev.flushed);
        stream.flush_channel().await.unwrap();
        assert!(stream.prev.flushed);
        assert!(stream.buffer.is_empty());
        assert!(stream.span.is_none());
    }

    #[tokio::test]
    async fn test_batch_stream_inactive() {
        let trace_store: TraceStorage = Default::default();
        let layer = CollectingLayer::new(trace_store.clone());
        let subscriber = tracing_subscriber::Registry::default().with(layer);
        let _guard = tracing::subscriber::set_default(subscriber);

        let data = vec![Ok(Batch::Single(SingleBatch::default()))];
        let config = Arc::new(RollupConfig {
            hardforks: HardForkConfig { holocene_time: Some(100), ..Default::default() },
            ..Default::default()
        });
        let prev = TestBatchStreamProvider::new(data);
        let mut stream = BatchStream::new(prev, config.clone(), TestL2ChainProvider::default());

        // The stage should not be active.
        assert!(!stream.is_active().unwrap());

        // The next batch should be passed through to the [BatchQueue] stage.
        let batch = stream.next_batch(Default::default(), &[]).await.unwrap();
        assert_eq!(batch, StagedBatch::new(Batch::Single(SingleBatch::default())));

        let logs = trace_store.get_by_level(tracing::Level::TRACE);
        assert_eq!(logs.len(), 1);
        assert!(logs[0].contains("BatchStream stage is inactive, pass-through."));
    }

    /// A chain with `block_time` 2 that allows up to three blocks per timestamp from 1000 on.
    fn multi_block_config() -> Arc<RollupConfig> {
        Arc::new(RollupConfig {
            block_time: 2,
            seq_window_size: 100,
            max_sequencer_drift: 1000,
            hardforks: HardForkConfig {
                delta_time: Some(0),
                holocene_time: Some(0),
                karst_time: Some(0),
                ..Default::default()
            },
            multi_block_time: Some(1000),
            max_multi_blocks: Some(3),
            ..Default::default()
        })
    }

    fn multi_block_l2_block(number: u64, timestamp: u64) -> L2BlockInfo {
        L2BlockInfo {
            block_info: BlockInfo {
                number,
                timestamp,
                hash: FixedBytes::<32>::repeat_byte(number as u8),
                ..Default::default()
            },
            l1_origin: NumHash { number: 100, hash: FixedBytes::<32>::repeat_byte(100) },
            seq_num: 0,
        }
    }

    /// A span batch continuing the group the safe head is in.
    fn multi_block_span(parent: L2BlockInfo) -> SpanBatch {
        let mut same_ts_bits = kona_protocol::SpanBatchBits::default();
        same_ts_bits.set_bit(0, true);
        same_ts_bits.set_bit(1, false);
        SpanBatch {
            version: kona_protocol::BatchType::SpanV2,
            parent_check: FixedBytes::<20>::from_slice(&parent.block_info.hash[..20]),
            l1_origin_check: FixedBytes::<20>::from_slice(
                &FixedBytes::<32>::repeat_byte(100)[..20],
            ),
            same_ts_bits: Some(same_ts_bits),
            batches: vec![
                SpanBatchElement { epoch_num: 100, timestamp: 1010, ..Default::default() },
                SpanBatchElement { epoch_num: 100, timestamp: 1012, ..Default::default() },
            ],
            ..Default::default()
        }
    }

    /// Nothing about a group is carried across a flush: the accounting is redone from the safe
    /// chain, so the same span is accepted again exactly as before.
    #[tokio::test]
    async fn test_span_batch_v2_group_accounting_recomputed_after_flush() {
        let safe_head = multi_block_l2_block(11, 1010);
        let span = multi_block_span(safe_head);
        let l1_origins = [BlockInfo {
            number: 100,
            timestamp: 900,
            hash: FixedBytes::<32>::repeat_byte(100),
            ..Default::default()
        }];

        let mut prev = TestBatchStreamProvider::new(vec![Ok(Batch::Span(span.clone()))]);
        prev.origin = Some(BlockInfo { number: 110, timestamp: 1100, ..Default::default() });
        let provider = TestL2ChainProvider {
            blocks: vec![multi_block_l2_block(9, 1008), multi_block_l2_block(10, 1010), safe_head],
            ..Default::default()
        };
        let mut stream = BatchStream::new(prev, multi_block_config(), provider);

        let first = stream.next_batch(safe_head, &l1_origins).await.unwrap();
        assert!(first.is_sibling);
        assert_eq!(stream.span_buffer_size(), 1);

        stream.flush();
        assert_eq!(stream.span_buffer_size(), 0);

        stream.prev.batches.push(Ok(Batch::Span(span)));
        let again = stream.next_batch(safe_head, &l1_origins).await.unwrap();
        assert_eq!(again, first);
    }

    #[tokio::test]
    async fn test_span_buffer() {
        let mock_batch = SpanBatch {
            batches: vec![
                SpanBatchElement { epoch_num: 1, timestamp: 2, ..Default::default() },
                SpanBatchElement { epoch_num: 1, timestamp: 4, ..Default::default() },
            ],
            ..Default::default()
        };
        let mock_origins = [BlockInfo { number: 1, timestamp: 12, ..Default::default() }];

        let data = vec![Ok(Batch::Span(mock_batch.clone()))];
        let config = Arc::new(RollupConfig {
            block_time: 2,
            hardforks: HardForkConfig {
                delta_time: Some(0),
                holocene_time: Some(0),
                ..Default::default()
            },
            ..Default::default()
        });
        let prev = TestBatchStreamProvider::new(data);
        let provider = TestL2ChainProvider::default();
        let mut stream = BatchStream::new(prev, config.clone(), provider);

        // The stage should be active.
        assert!(stream.is_active().unwrap());

        // The next batches should be single batches derived from the span batch.
        let batch = stream.next_batch(Default::default(), &mock_origins).await.unwrap();
        if let Batch::Single(single) = batch.batch {
            assert_eq!(single.epoch_num, 1);
            assert_eq!(single.timestamp, 2);
        } else {
            panic!("Wrong batch type");
        }

        let batch = stream.next_batch(Default::default(), &mock_origins).await.unwrap();
        if let Batch::Single(single) = batch.batch {
            assert_eq!(single.epoch_num, 1);
            assert_eq!(single.timestamp, 4);
        } else {
            panic!("Wrong batch type");
        }

        let err = stream.next_batch(Default::default(), &mock_origins).await.unwrap_err();
        assert_eq!(err, PipelineError::Eof.temp());
        assert_eq!(stream.span_buffer_size(), 0);
        assert!(stream.span.is_none());

        // Add more data into the provider, see if the buffer is re-hydrated.
        stream.prev.batches.push(Ok(Batch::Span(mock_batch.clone())));

        // The next batches should be single batches derived from the span batch.
        let batch = stream.next_batch(Default::default(), &mock_origins).await.unwrap();
        if let Batch::Single(single) = batch.batch {
            assert_eq!(single.epoch_num, 1);
            assert_eq!(single.timestamp, 2);
        } else {
            panic!("Wrong batch type");
        }

        let batch = stream.next_batch(Default::default(), &mock_origins).await.unwrap();
        if let Batch::Single(single) = batch.batch {
            assert_eq!(single.epoch_num, 1);
            assert_eq!(single.timestamp, 4);
        } else {
            panic!("Wrong batch type");
        }

        let err = stream.next_batch(Default::default(), &mock_origins).await.unwrap_err();
        assert_eq!(err, PipelineError::Eof.temp());
        assert_eq!(stream.span_buffer_size(), 0);
        assert!(stream.span.is_none());
    }

    #[tokio::test]
    async fn test_span_batch_extraction_error_flushes_stage() {
        let trace_store: TraceStorage = Default::default();
        let layer = CollectingLayer::new(trace_store.clone());
        let subscriber = tracing_subscriber::Registry::default().with(layer);
        let _guard = tracing::subscriber::set_default(subscriber);

        let parent_hash = b256!("1111111111111111111111111111111111111111000000000000000000000000");
        let l1_block_hash =
            b256!("3333333333333333333333333333333333333333000000000000000000000000");
        let config = Arc::new(RollupConfig {
            seq_window_size: 100,
            block_time: 10,
            hardforks: HardForkConfig {
                delta_time: Some(0),
                holocene_time: Some(0),
                ..Default::default()
            },
            genesis: ChainGenesis {
                l2: BlockNumHash { number: 40, hash: parent_hash },
                ..Default::default()
            },
            ..Default::default()
        });

        let l1_block =
            BlockInfo { number: 10, timestamp: 5, hash: l1_block_hash, ..Default::default() };
        let l1_blocks = vec![l1_block];
        let l2_safe_head = L2BlockInfo {
            block_info: BlockInfo { number: 41, timestamp: 10, parent_hash, ..Default::default() },
            l1_origin: l1_block.id(),
            ..Default::default()
        };
        let l2_parent = L2BlockInfo {
            block_info: BlockInfo {
                number: 40,
                hash: parent_hash,
                timestamp: 0,
                ..Default::default()
            },
            l1_origin: BlockNumHash { number: 9, ..Default::default() },
            ..Default::default()
        };
        // A valid overlapped canonical block (L1 info deposit only, origin 9), so the overlap
        // content checks pass and singular batch extraction is reached.
        let l1_info = kona_protocol::L1BlockInfoBedrock::new(
            9,
            0,
            0,
            alloy_primitives::B256::ZERO,
            0,
            alloy_primitives::Address::ZERO,
            alloy_primitives::U256::ZERO,
            alloy_primitives::U256::ZERO,
        );
        let info_tx = op_alloy_consensus::OpTxEnvelope::Deposit(alloy_primitives::Sealed::new(
            op_alloy_consensus::TxDeposit {
                input: l1_info.encode_calldata(),
                ..Default::default()
            },
        ));
        let op_block = OpBlock {
            header: Header { number: 41, ..Default::default() },
            body: BlockBody { transactions: vec![info_tx], ommers: vec![], withdrawals: None },
        };

        let span_batch = SpanBatch {
            batches: vec![
                SpanBatchElement { epoch_num: 9, timestamp: 10, ..Default::default() },
                SpanBatchElement { epoch_num: 9, timestamp: 20, ..Default::default() },
                SpanBatchElement { epoch_num: 10, timestamp: 30, ..Default::default() },
            ],
            parent_check: FixedBytes::<20>::from_slice(&parent_hash[..20]),
            l1_origin_check: FixedBytes::<20>::from_slice(&l1_block_hash[..20]),
            ..Default::default()
        };

        let mut prev = TestBatchStreamProvider::new(vec![Ok(Batch::Span(span_batch))]);
        prev.origin = Some(l1_block);

        let mut provider = TestL2ChainProvider::default();
        provider.blocks.push(l2_parent);
        provider.op_blocks.push(op_block);

        let mut stream = BatchStream::new(prev, config, provider);
        let err = stream.next_batch(l2_safe_head, &l1_blocks).await.unwrap_err();

        assert_eq!(err, PipelineError::NotEnoughData.temp());
        assert!(stream.span.is_none());
        assert_eq!(stream.span_buffer_size(), 0);

        let logs = trace_store.get_by_level(tracing::Level::WARN);
        assert_eq!(logs.len(), 1);
        assert!(logs[0].contains("Extracting singular batches from span batch failed: Future batch L1 origin before safe head"));
    }

    #[tokio::test]
    async fn test_overlap_mismatch_drops_span_and_flushes_unread_batches() {
        let trace_store: TraceStorage = Default::default();
        let layer = CollectingLayer::new(trace_store.clone());
        let subscriber = tracing_subscriber::Registry::default().with(layer);
        let _guard = tracing::subscriber::set_default(subscriber);

        let parent_hash = b256!("1111111111111111111111111111111111111111000000000000000000000000");
        let l1_block_hash =
            b256!("3333333333333333333333333333333333333333000000000000000000000000");
        let config = Arc::new(RollupConfig {
            seq_window_size: 100,
            block_time: 10,
            hardforks: HardForkConfig {
                delta_time: Some(0),
                holocene_time: Some(0),
                ..Default::default()
            },
            genesis: ChainGenesis {
                l2: BlockNumHash { number: 40, hash: parent_hash },
                ..Default::default()
            },
            ..Default::default()
        });

        let l1_block =
            BlockInfo { number: 10, timestamp: 5, hash: l1_block_hash, ..Default::default() };
        let l1_blocks = vec![l1_block];
        // A two-block overlap: blocks 41 and 42 are already safe.
        let l2_safe_head = L2BlockInfo {
            block_info: BlockInfo { number: 42, timestamp: 20, ..Default::default() },
            l1_origin: l1_block.id(),
            ..Default::default()
        };
        let l2_parent = L2BlockInfo {
            block_info: BlockInfo {
                number: 40,
                hash: parent_hash,
                timestamp: 0,
                ..Default::default()
            },
            l1_origin: BlockNumHash { number: 9, ..Default::default() },
            ..Default::default()
        };
        // The canonical overlapped blocks carry only their L1 info deposit (origin 9).
        let l1_info = kona_protocol::L1BlockInfoBedrock::new(
            9,
            0,
            0,
            alloy_primitives::B256::ZERO,
            0,
            alloy_primitives::Address::ZERO,
            alloy_primitives::U256::ZERO,
            alloy_primitives::U256::ZERO,
        );
        let info_tx = op_alloy_consensus::OpTxEnvelope::Deposit(alloy_primitives::Sealed::new(
            op_alloy_consensus::TxDeposit {
                input: l1_info.encode_calldata(),
                ..Default::default()
            },
        ));
        let op_block_41 = OpBlock {
            header: Header { number: 41, ..Default::default() },
            body: BlockBody {
                transactions: vec![info_tx.clone()],
                ommers: vec![],
                withdrawals: None,
            },
        };
        let op_block_42 = OpBlock {
            header: Header { number: 42, ..Default::default() },
            body: BlockBody { transactions: vec![info_tx], ommers: vec![], withdrawals: None },
        };

        // The span's first overlapped element matches the safe chain; only the second
        // diverges (it carries a transaction the canonical block does not), exercising the
        // overlap comparison loop beyond its first iteration.
        let span_batch = SpanBatch {
            batches: vec![
                SpanBatchElement { epoch_num: 9, timestamp: 10, ..Default::default() },
                SpanBatchElement {
                    epoch_num: 9,
                    timestamp: 20,
                    transactions: vec![alloy_primitives::Bytes::from_static(&[0x02, 0x01])],
                },
                SpanBatchElement { epoch_num: 10, timestamp: 30, ..Default::default() },
            ],
            parent_check: FixedBytes::<20>::from_slice(&parent_hash[..20]),
            l1_origin_check: FixedBytes::<20>::from_slice(&l1_block_hash[..20]),
            ..Default::default()
        };

        // An unread sentinel sits behind the conflicting span in the channel (batches are
        // served back-to-front): it only surviving the drop would prove the flush was
        // signaled but not performed.
        let sentinel = Batch::Single(SingleBatch::default());
        let mut prev =
            TestBatchStreamProvider::new(vec![Ok(sentinel), Ok(Batch::Span(span_batch))]);
        prev.origin = Some(l1_block);

        let mut provider = TestL2ChainProvider::default();
        provider.blocks.push(l2_parent);
        provider.op_blocks.push(op_block_41);
        provider.op_blocks.push(op_block_42);

        let mut stream = BatchStream::new(prev, config, provider);
        let err = stream.next_batch(l2_safe_head, &l1_blocks).await.unwrap_err();

        assert_eq!(err, PipelineError::NotEnoughData.temp());
        assert!(stream.span.is_none());
        assert_eq!(stream.span_buffer_size(), 0);
        // The flush must have discarded the unread sentinel along with the rest of the channel.
        assert!(stream.prev.batches.is_empty());

        let logs = trace_store.get_by_level(tracing::Level::WARN);
        assert_eq!(logs.len(), 1);
        assert!(logs[0].contains("overlapped block's tx count does not match"));
    }

    #[tokio::test]
    async fn test_single_batch_pass_through() {
        let data = vec![Ok(Batch::Single(SingleBatch::default()))];
        let config = Arc::new(RollupConfig {
            hardforks: HardForkConfig { holocene_time: Some(0), ..Default::default() },
            ..Default::default()
        });
        let prev = TestBatchStreamProvider::new(data);
        let mut stream = BatchStream::new(prev, config.clone(), TestL2ChainProvider::default());

        // The stage should be active.
        assert!(stream.is_active().unwrap());

        // The next batch should be passed through to the [BatchQueue] stage.
        let batch = stream.next_batch(Default::default(), &[]).await.unwrap();
        assert!(matches!(batch.batch, Batch::Single(_)));
        assert_eq!(stream.span_buffer_size(), 0);
        assert!(stream.span.is_none());
    }

    #[tokio::test]
    async fn test_past_span_batch() {
        let mock_batch = SpanBatch {
            batches: vec![
                SpanBatchElement { epoch_num: 1, timestamp: 2, ..Default::default() },
                SpanBatchElement { epoch_num: 1, timestamp: 4, ..Default::default() },
            ],
            ..Default::default()
        };
        let mock_origins = [BlockInfo { number: 1, timestamp: 12, ..Default::default() }];
        let data = vec![Ok(Batch::Span(mock_batch))];

        let config = Arc::new(RollupConfig {
            hardforks: HardForkConfig { holocene_time: Some(0), ..Default::default() },
            ..Default::default()
        });
        let prev = TestBatchStreamProvider::new(data);
        let mut stream = BatchStream::new(prev, config.clone(), TestL2ChainProvider::default());

        // The stage should be active.
        assert!(stream.is_active().unwrap());

        let parent = L2BlockInfo {
            block_info: BlockInfo { number: 10, timestamp: 100, ..Default::default() },
            l1_origin: NumHash::default(),
            seq_num: 0,
        };

        // `next_batch` should return an error if the span batch is in the past.
        let err = stream.next_batch(parent, &mock_origins).await.unwrap_err();
        assert_eq!(err, PipelineError::NotEnoughData.temp());
    }
}
