//! This module contains the `BatchStream` stage.

use crate::{
    L2ChainProvider, NextBatchProvider, OriginAdvancer, OriginProvider, PipelineError,
    PipelineResult, Signal, SignalReceiver,
};
use alloc::{boxed::Box, collections::VecDeque, sync::Arc};
use async_trait::async_trait;
use core::fmt::Debug;
use kona_genesis::RollupConfig;
use kona_protocol::{
    Batch, BatchValidity, BatchWithInclusionBlock, BlockInfo, L2BlockInfo, SingleBatch, SpanBatch,
    SpanBatchError,
};

/// Provides [`Batch`]es for the [`BatchStream`] stage.
#[async_trait]
pub trait BatchStreamProvider {
    /// Returns the next [`Batch`] in the [`BatchStream`] stage.
    async fn next_batch(&mut self) -> PipelineResult<Batch>;

    /// Drains the recent `Channel` if an invalid span batch is found post-holocene.
    fn flush(&mut self);
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
    P: BatchStreamProvider + OriginAdvancer + OriginProvider + SignalReceiver + Debug,
    BF: L2ChainProvider + Debug,
{
    /// The previous stage in the derivation pipeline.
    pub prev: P,
    /// There can only be a single staged span batch.
    pub span: Option<SpanBatch>,
    /// A buffer of single batches derived from the [`SpanBatch`].
    pub buffer: VecDeque<SingleBatch>,
    /// A reference to the rollup config, used to check
    /// if the [`BatchStream`] stage should be activated.
    pub config: Arc<RollupConfig>,
    /// Used to validate the batches.
    pub fetcher: BF,
}

impl<P, BF> BatchStream<P, BF>
where
    P: BatchStreamProvider + OriginAdvancer + OriginProvider + SignalReceiver + Debug,
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

    /// Gets a [`SingleBatch`] from the in-memory buffer.
    pub fn get_single_batch(
        &mut self,
        parent: L2BlockInfo,
        l1_origins: &[BlockInfo],
    ) -> Result<Option<SingleBatch>, SpanBatchError> {
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
        if let Some(span) = self.span.take() {
            self.buffer.extend(span.get_singular_batches(l1_origins, parent)?);
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
    P: BatchStreamProvider + OriginAdvancer + OriginProvider + SignalReceiver + Send + Debug,
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
    ) -> PipelineResult<Batch> {
        // If the stage is not active, "pass" the next batch
        // through this stage to the BatchQueue stage.
        if !self.is_active()? {
            trace!(target: "batch_span", "BatchStream stage is inactive, pass-through.");
            return self.prev.next_batch().await;
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
                Batch::Single(b) => return Ok(Batch::Single(b)),
                Batch::Span(b) => {
                    #[cfg(feature = "metrics")]
                    let start = std::time::Instant::now();
                    let (validity, _) = b
                        .check_batch_prefix(
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
                        "validity" => validity.to_string(),
                    );

                    match validity {
                        BatchValidity::Accept => self.span = Some(b),
                        BatchValidity::Drop => {
                            // Flush the stage.
                            self.flush();

                            return Err(PipelineError::NotEnoughData.temp());
                        }
                        BatchValidity::Past => {
                            if !self.is_active()? {
                                error!(target: "batch_stream", "BatchValidity::Past is not allowed pre-holocene");
                                return Err(PipelineError::InvalidBatchValidity.crit());
                            }

                            return Err(PipelineError::NotEnoughData.temp());
                        }
                        BatchValidity::Undecided | BatchValidity::Future => {
                            return Err(PipelineError::NotEnoughData.temp());
                        }
                    }
                }
            }
        }

        // Attempt to pull a SingleBatch out of the SpanBatch.
        match self.get_single_batch(parent, l1_origins) {
            Ok(Some(single_batch)) => Ok(Batch::Single(single_batch)),
            Ok(None) => Err(PipelineError::NotEnoughData.temp()),
            Err(e) => {
                warn!(target: "batch_span", "Extracting singular batches from span batch failed: {}", e);
                // If singular batch extraction fails, it should be handled the same as a
                // dropped batch during span batch prefix checks.
                self.flush();
                Err(PipelineError::NotEnoughData.temp())
            }
        }
    }
}

#[async_trait]
impl<P, BF> OriginAdvancer for BatchStream<P, BF>
where
    P: BatchStreamProvider + OriginAdvancer + OriginProvider + SignalReceiver + Send + Debug,
    BF: L2ChainProvider + Send + Debug,
{
    async fn advance_origin(&mut self) -> PipelineResult<()> {
        self.prev.advance_origin().await
    }
}

impl<P, BF> OriginProvider for BatchStream<P, BF>
where
    P: BatchStreamProvider + OriginAdvancer + OriginProvider + SignalReceiver + Debug,
    BF: L2ChainProvider + Debug,
{
    fn origin(&self) -> Option<BlockInfo> {
        self.prev.origin()
    }
}

#[async_trait]
impl<P, BF> SignalReceiver for BatchStream<P, BF>
where
    P: BatchStreamProvider + OriginAdvancer + OriginProvider + SignalReceiver + Debug + Send,
    BF: L2ChainProvider + Send + Debug,
{
    async fn signal(&mut self, signal: Signal) -> PipelineResult<()> {
        self.prev.signal(signal).await?;
        self.buffer.clear();
        self.span.take();
        Ok(())
    }
}

#[cfg(test)]
mod test {
    use super::*;
    use crate::{
        test_utils::{CollectingLayer, TestBatchStreamProvider, TestL2ChainProvider, TraceStorage},
        types::ResetSignal,
    };
    use alloc::vec;
    use alloy_consensus::{BlockBody, Header};
    use alloy_eips::{BlockNumHash, NumHash};
    use alloy_primitives::{FixedBytes, b256};
    use kona_genesis::{ChainGenesis, HardForkConfig};
    use kona_protocol::{SingleBatch, SpanBatchElement};
    use op_alloy_consensus::OpBlock;
    use tracing_subscriber::{layer::SubscriberExt, util::SubscriberInitExt};

    #[tokio::test]
    async fn test_batch_stream_flush() {
        let config = Arc::new(RollupConfig {
            hardforks: HardForkConfig { holocene_time: Some(0), ..Default::default() },
            ..Default::default()
        });
        let prev = TestBatchStreamProvider::new(vec![]);
        let mut stream = BatchStream::new(prev, config, TestL2ChainProvider::default());
        stream.buffer.push_back(SingleBatch::default());
        stream.span = Some(SpanBatch::default());
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
        stream.buffer.push_back(SingleBatch::default());
        stream.span = Some(SpanBatch::default());
        assert!(!stream.prev.reset);
        stream.signal(ResetSignal::default().signal()).await.unwrap();
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
        stream.buffer.push_back(SingleBatch::default());
        stream.span = Some(SpanBatch::default());
        assert!(!stream.prev.flushed);
        stream.signal(Signal::FlushChannel).await.unwrap();
        assert!(stream.prev.flushed);
        assert!(stream.buffer.is_empty());
        assert!(stream.span.is_none());
    }

    #[tokio::test]
    async fn test_batch_stream_inactive() {
        let trace_store: TraceStorage = Default::default();
        let layer = CollectingLayer::new(trace_store.clone());
        tracing_subscriber::Registry::default().with(layer).init();

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
        assert_eq!(batch, Batch::Single(SingleBatch::default()));

        let logs = trace_store.get_by_level(tracing::Level::TRACE);
        assert_eq!(logs.len(), 1);
        assert!(logs[0].contains("BatchStream stage is inactive, pass-through."));
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
        if let Batch::Single(single) = batch {
            assert_eq!(single.epoch_num, 1);
            assert_eq!(single.timestamp, 2);
        } else {
            panic!("Wrong batch type");
        }

        let batch = stream.next_batch(Default::default(), &mock_origins).await.unwrap();
        if let Batch::Single(single) = batch {
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
        if let Batch::Single(single) = batch {
            assert_eq!(single.epoch_num, 1);
            assert_eq!(single.timestamp, 2);
        } else {
            panic!("Wrong batch type");
        }

        let batch = stream.next_batch(Default::default(), &mock_origins).await.unwrap();
        if let Batch::Single(single) = batch {
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
        tracing_subscriber::Registry::default().with(layer).init();

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
        let op_block = OpBlock {
            header: Header { number: 41, ..Default::default() },
            body: BlockBody { transactions: vec![], ommers: vec![], withdrawals: None },
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
        assert!(matches!(batch, Batch::Single(_)));
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
