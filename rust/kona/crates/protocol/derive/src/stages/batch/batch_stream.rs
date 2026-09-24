//! This module contains the `BatchStream` stage.

use crate::{
    L2ChainProvider, NextBatchProvider, OriginAdvancer, OriginProvider, PipelineError,
    PipelineResult, Stage,
};
use alloc::{boxed::Box, collections::VecDeque, sync::Arc};
use alloy_eips::BlockNumHash;
use async_trait::async_trait;
use core::fmt::Debug;
use kona_genesis::{RollupConfig, SystemConfig};
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
    P: BatchStreamProvider + OriginAdvancer + OriginProvider + Stage + Debug,
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
    /// Candidate retained while canonical projection context is unavailable.
    pub pending: Option<BatchWithInclusionBlock>,
    /// Original inclusion of the accepted span, retained through its last emitted block.
    pub inclusion: Option<BlockInfo>,
    /// Bounded, incremental context collection outside the pure validator.
    pub collector: kona_protocol::projection::ContextCollector,
}

impl<P, BF> BatchStream<P, BF>
where
    P: BatchStreamProvider + OriginAdvancer + OriginProvider + Stage + Debug,
    BF: L2ChainProvider + Debug,
{
    /// Create a new [`BatchStream`] stage.
    pub const fn new(prev: P, config: Arc<RollupConfig>, fetcher: BF) -> Self {
        Self {
            prev,
            span: None,
            buffer: VecDeque::new(),
            config,
            fetcher,
            pending: None,
            inclusion: None,
            collector: kona_protocol::projection::ContextCollector::new(),
        }
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
    P: BatchStreamProvider + OriginAdvancer + OriginProvider + Stage + Send + Debug,
    BF: L2ChainProvider + Send + Debug,
{
    fn flush(&mut self) {
        if self.is_active().unwrap_or(false) {
            self.prev.flush();
            self.span = None;
            self.pending = None;
            self.inclusion = None;
            self.collector.reset();
            self.buffer.clear();
        }
    }

    fn batch_inclusion_block(&self) -> Option<BlockInfo> {
        self.inclusion
    }

    fn has_pending_batch(&self) -> bool {
        self.pending.is_some()
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
            if self.config.private_projection.is_some() {
                return Err(PipelineError::InvalidBatchValidity.crit());
            }
            trace!(target: "batch_span", "BatchStream stage is inactive, pass-through.");
            return self.prev.next_batch().await;
        }

        // If the buffer is empty, attempt to pull a batch from the previous stage.
        if self.buffer.is_empty() {
            self.inclusion = None;
            // Safety: bubble up any errors from the batch reader.
            let batch_with_inclusion = match self.pending.take() {
                Some(batch) => batch,
                None => BatchWithInclusionBlock::new(
                    self.origin().ok_or(PipelineError::MissingOrigin.crit())?,
                    self.prev.next_batch().await?,
                ),
            };

            // If the next batch is a singular batch, it is immediately
            // forwarded to the `BatchQueue` stage. Otherwise, we buffer
            // the span batch in this stage if it passes the validity checks.
            match batch_with_inclusion.batch {
                Batch::Single(b) => {
                    if self.config.private_projection.is_some() {
                        self.flush();
                        return Err(PipelineError::NotEnoughData.temp());
                    }
                    return Ok(Batch::Single(b));
                }
                Batch::Span(b) => {
                    #[cfg(feature = "metrics")]
                    let start = std::time::Instant::now();
                    let validity = b
                        .check_batch_holocene_with_context(
                            self.config.as_ref(),
                            l1_origins,
                            parent,
                            &batch_with_inclusion.inclusion_block,
                            &mut self.fetcher,
                            &mut self.collector,
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
                        BatchValidity::Accept => {
                            self.inclusion = Some(batch_with_inclusion.inclusion_block);
                            self.span = Some(b);
                        }
                        BatchValidity::Drop(_) => {
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
                        BatchValidity::Retry => {
                            self.pending = Some(BatchWithInclusionBlock::new(
                                batch_with_inclusion.inclusion_block,
                                Batch::Span(b),
                            ));
                            return Err(PipelineError::ProjectionContextUnavailable.temp());
                        }
                        BatchValidity::Undecided | BatchValidity::Future => {
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
            Ok(Some(single_batch)) => Ok(Batch::Single(single_batch)),
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
        self.pending = None;
        self.inclusion = None;
        self.collector.reset();
        Ok(())
    }

    async fn activate(&mut self) -> PipelineResult<()> {
        self.prev.activate().await?;
        self.buffer.clear();
        self.span.take();
        self.pending = None;
        self.inclusion = None;
        self.collector.reset();
        Ok(())
    }

    async fn flush_channel(&mut self) -> PipelineResult<()> {
        self.prev.flush_channel().await?;
        self.buffer.clear();
        self.span.take();
        self.pending = None;
        self.inclusion = None;
        self.collector.reset();
        Ok(())
    }
}

#[cfg(test)]
mod test {
    use super::*;
    use crate::test_utils::{
        CollectingLayer, TestBatchStreamProvider, TestL2ChainProvider, TraceStorage,
    };
    use alloc::{vec, vec::Vec};
    use alloy_consensus::{BlockBody, Header};
    use alloy_eips::{BlockNumHash, NumHash};
    use alloy_primitives::{FixedBytes, b256};
    use kona_genesis::{ChainGenesis, HardForkConfig};
    use kona_protocol::{SingleBatch, SpanBatchElement};
    use op_alloy_consensus::OpBlock;
    use tracing_subscriber::layer::SubscriberExt;

    #[tokio::test]
    async fn projection_retry_preserves_inclusion_through_last_single() {
        use crate::{AttributesProvider, BatchValidator};
        use alloy_primitives::{B256, Bytes, TxKind};
        use kona_genesis::PrivateProjectionConfig;
        let vectors: serde_json::Value = serde_json::from_str(include_str!(
            "../../../../../../../../op-private-interop/projection/testdata/ranges.json"
        ))
        .unwrap();
        let v = &vectors[0];
        let h = |x: &serde_json::Value| -> B256 {
            if x.is_null() { B256::ZERO } else { serde_json::from_value(x.clone()).unwrap() }
        };
        let (c, x) = (&v["config"], &v["context"]);
        let root = h(&x["continuation"]["output_root"]);
        let l1_head = h(&x["l1_head"]);
        let cfg = Arc::new(RollupConfig {
            l2_chain_id: x["chain_id"].as_u64().unwrap().into(),
            block_time: x["block_time"].as_u64().unwrap(),
            seq_window_size: 2,
            max_sequencer_drift: 600,
            genesis: ChainGenesis {
                l2_time: x["genesis_time"].as_u64().unwrap(),
                l2: BlockNumHash {
                    number: x["genesis_number"].as_u64().unwrap(),
                    hash: h(&x["genesis_hash"]),
                },
                ..Default::default()
            },
            hardforks: HardForkConfig {
                delta_time: Some(0),
                holocene_time: Some(0),
                ..Default::default()
            },
            private_projection: Some(PrivateProjectionConfig {
                verifier: c["verifier"].as_str().unwrap().into(),
                allow_events: c["allow_events"].as_bool().unwrap_or(false),
                genesis_output_root: h(&c["genesis_output_root"]),
                program_vkey: h(&c["program_vkey"]),
                private_config_hash: h(&c["private_config_hash"]),
                dependency_set_hash: h(&c["dependency_set_hash"]),
                mock_proofs: c["mock_proofs"].as_bool().unwrap_or(false),
            }),
            ..Default::default()
        });
        let checkpoint = OpBlock {
            header: Header { number: 9, ..Default::default() },
            body: BlockBody {
                transactions: vec![op_alloy_consensus::OpTxEnvelope::Eip1559(
                    alloy_consensus::Signed::new_unchecked(
                        alloy_consensus::TxEip1559 {
                            to: TxKind::Call(alloy_primitives::address!(
                                "420000000000000000000000000000000000002e"
                            )),
                            input: [vec![0x61, 0x84, 0xd0, 0x8e], root.to_vec()].concat().into(),
                            ..Default::default()
                        },
                        alloy_primitives::Signature::test_signature(),
                        B256::ZERO,
                    ),
                )],
                ..Default::default()
            },
        };
        let origins = vec![
            BlockInfo {
                number: 5,
                hash: B256::with_last_byte(5),
                timestamp: 1000,
                ..Default::default()
            },
            // The span's last epoch: its hash is the claim's `l1Head`.
            BlockInfo { number: 6, hash: l1_head, timestamp: 1012, ..Default::default() },
        ];
        let mut parent = L2BlockInfo {
            block_info: BlockInfo {
                number: 9,
                hash: checkpoint.header.hash_slow(),
                timestamp: 1018,
                ..Default::default()
            },
            l1_origin: origins[0].id(),
            ..Default::default()
        };
        let mut span = SpanBatch {
            parent_check: parent.block_info.hash[..20].try_into().unwrap(),
            l1_origin_check: origins[1].hash[..20].try_into().unwrap(),
            batches: v["blocks"]
                .as_array()
                .unwrap()
                .iter()
                .map(|b| SpanBatchElement {
                    timestamp: b["timestamp"].as_u64().unwrap(),
                    epoch_num: b["epoch"].as_u64().unwrap(),
                    transactions: serde_json::from_value::<Vec<Bytes>>(b["transactions"].clone())
                        .unwrap(),
                })
                .collect(),
            ..Default::default()
        };
        // The vector's execution-mock proof binds its own parent; re-prove the span for
        // this derivation context (parent and anchor = the checkpoint) and re-sign the
        // claim with the vector key, as the batcher would.
        {
            use kona_protocol::projection;
            let ctx = projection::ProjectionContext {
                parent_hash: parent.block_info.hash,
                l1_head,
                continuation: projection::Continuation {
                    anchor: parent.block_info.id(),
                    output_root: root,
                    recovery_hash: B256::ZERO,
                },
            };
            let statement =
                projection::validate_projection_range(&cfg, ctx, &span, &projection::StubVerifier)
                    .unwrap();
            let proof = projection::execution_mock_proof(&statement);
            let raw = &mut span.batches[0].transactions[0];
            *raw = kona_protocol::test_utils::resign_projection_claim(
                raw,
                kona_protocol::test_utils::PROJECTION_VECTOR_KEY,
                |claim| claim.proof = proof.into(),
            );
        }
        let expected = span.batches.clone();
        let prev = TestBatchStreamProvider {
            origin: Some(origins[1]),
            batches: vec![Ok(Batch::Span(span))],
            ..Default::default()
        };
        let stream = BatchStream::new(prev, cfg.clone(), TestL2ChainProvider::default());
        let mut validator = BatchValidator::new(cfg, stream);
        validator.origin = Some(origins[1]);
        validator.l1_blocks = origins.clone();
        assert_eq!(
            validator.next_batch(parent).await.unwrap_err(),
            PipelineError::ProjectionContextUnavailable.temp()
        );
        assert!(validator.prev.has_pending_batch());
        assert!(validator.prev.prev.batches.is_empty());
        validator.prev.fetcher.op_blocks.push(checkpoint);
        validator.prev.prev.origin =
            Some(BlockInfo { number: 20, timestamp: 1180, ..Default::default() });
        for block in expected {
            let next = validator.next_batch(parent).await.unwrap();
            assert_eq!(next.transactions, block.transactions);
            assert_eq!(validator.prev.batch_inclusion_block(), Some(origins[1]));
            parent.block_info.number += 1;
            parent.block_info.timestamp = next.timestamp;
            parent.block_info.hash = B256::with_last_byte(parent.block_info.number as u8);
            parent.l1_origin = origins[(next.epoch_num - 5) as usize].id();
        }
        validator.prev.flush();
        assert!(!validator.prev.has_pending_batch());
        assert!(validator.prev.batch_inclusion_block().is_none());
    }

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
        stream.buffer.push_back(SingleBatch::default());
        stream.span = Some(SpanBatch::default());
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
