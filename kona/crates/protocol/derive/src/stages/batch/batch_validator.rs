//! Contains the [BatchValidator] stage.

use super::NextBatchProvider;
use crate::{
    errors::{PipelineError, PipelineErrorKind, ResetError},
    traits::{AttributesProvider, OriginAdvancer, OriginProvider, SignalReceiver},
    types::{PipelineResult, ResetSignal, Signal},
};
use alloc::{boxed::Box, sync::Arc, vec::Vec};
use async_trait::async_trait;
use core::fmt::Debug;
use kona_genesis::RollupConfig;
use kona_protocol::{Batch, BatchValidity, BlockInfo, L2BlockInfo, SingleBatch};

/// The [`BatchValidator`] stage is responsible for validating the [`SingleBatch`]es from
/// the [`BatchStream`] [`AttributesQueue`]'s consumption.
///
/// [`BatchStream`]: crate::stages::BatchStream
/// [`AttributesQueue`]: crate::stages::attributes_queue::AttributesQueue
#[derive(Debug)]
pub struct BatchValidator<P>
where
    P: NextBatchProvider + OriginAdvancer + OriginProvider + SignalReceiver + Debug,
{
    /// The rollup configuration.
    pub cfg: Arc<RollupConfig>,
    /// The previous stage of the derivation pipeline.
    pub prev: P,
    /// The L1 origin of the batch sequencer.
    pub origin: Option<BlockInfo>,
    /// A consecutive, time-centric window of L1 Blocks.
    /// Every L1 origin of unsafe L2 Blocks must be included in this list.
    /// If every L2 Block corresponding to a single L1 Block becomes safe,
    /// the block is popped from this list.
    /// If new L2 Block's L1 origin is not included in this list, fetch and
    /// push it to the list.
    pub l1_blocks: Vec<BlockInfo>,
}

impl<P> BatchValidator<P>
where
    P: NextBatchProvider + OriginAdvancer + OriginProvider + SignalReceiver + Debug,
{
    /// Create a new [`BatchValidator`] stage.
    pub const fn new(cfg: Arc<RollupConfig>, prev: P) -> Self {
        Self { cfg, prev, origin: None, l1_blocks: Vec::new() }
    }

    /// Returns `true` if the pipeline origin is behind the parent origin.
    ///
    /// ## Takes
    /// - `parent`: The parent block of the current batch.
    ///
    /// ## Returns
    /// - `true` if the origin is behind the parent origin.
    fn origin_behind(&self, parent: &L2BlockInfo) -> bool {
        self.prev.origin().is_none_or(|origin| origin.number < parent.l1_origin.number)
    }

    /// Updates the [`BatchValidator`]'s view of the L1 origin blocks.
    ///
    /// ## Takes
    /// - `parent`: The parent block of the current batch.
    ///
    /// ## Returns
    /// - `Ok(())` if the update was successful.
    /// - `Err(PipelineError)` if the update failed.
    pub(crate) fn update_origins(&mut self, parent: &L2BlockInfo) -> PipelineResult<()> {
        // NOTE: The origin is used to determine if it's behind.
        // It is the future origin that gets saved into the l1 blocks array.
        // We always update the origin of this stage if it's not the same so
        // after the update code runs, this is consistent.
        let origin_behind = self.origin_behind(parent);

        // Advance the origin if needed.
        // The entire pipeline has the same origin.
        // Batches prior to the l1 origin of the l2 safe head are not accepted.
        if self.origin != self.prev.origin() {
            self.origin = self.prev.origin();
            if !origin_behind {
                let origin = self.origin.as_ref().ok_or(PipelineError::MissingOrigin.crit())?;
                self.l1_blocks.push(*origin);
            } else {
                // This is to handle the special case of startup.
                // At startup, the batch validator is reset and includes the
                // l1 origin. That is the only time when immediately after
                // reset is called, the origin behind is false.
                self.l1_blocks.clear();
            }
            debug!(
                target: "batch_validator",
                "Advancing batch validator origin to L1 block #{}.{}",
                self.origin.map(|b| b.number).unwrap_or_default(),
                if origin_behind { " (origin behind)" } else { Default::default() }
            );
        }

        // If the epoch is advanced, update the l1 blocks.
        // Advancing epoch must be done after the pipeline successfully applies the entire span
        // batch to the chain.
        // Because the span batch can be reverted during processing the batch, then we must
        // preserve existing l1 blocks to verify the epochs of the next candidate batch.
        if !self.l1_blocks.is_empty() && parent.l1_origin.number > self.l1_blocks[0].number {
            for (i, block) in self.l1_blocks.iter().enumerate() {
                if parent.l1_origin.number == block.number {
                    self.l1_blocks.drain(0..i);
                    debug!(target: "batch_validator", "Advancing internal L1 epoch");
                    break;
                }
            }
            // If the origin of the parent block is not included, we must advance the origin.
        }

        #[cfg(feature = "metrics")]
        {
            if let Some(origin) = self.l1_blocks.first() {
                kona_macros::set!(
                    gauge,
                    crate::metrics::Metrics::PIPELINE_L1_BLOCKS_START,
                    origin.number as f64
                );
                let last = self.l1_blocks.last().unwrap_or(origin);
                kona_macros::set!(
                    gauge,
                    crate::metrics::Metrics::PIPELINE_L1_BLOCKS_END,
                    last.number as f64
                );
            }
        }

        Ok(())
    }

    /// Attempts to derive an empty batch, if the sequencing window is expired.
    ///
    /// ## Takes
    /// - `parent`: The parent block of the current batch.
    ///
    /// ## Returns
    /// - `Ok(SingleBatch)` if an empty batch was derived.
    /// - `Err(PipelineError)` if an empty batch could not be derived.
    pub(crate) fn try_derive_empty_batch(
        &mut self,
        parent: &L2BlockInfo,
    ) -> PipelineResult<SingleBatch> {
        let epoch = self.l1_blocks[0];

        // If the current epoch is too old compared to the L1 block we are at,
        // i.e. if the sequence window expired, we create empty batches for the current epoch
        let stage_origin = self.origin.ok_or(PipelineError::MissingOrigin.crit())?;
        let expiry_epoch = epoch.number + self.cfg.seq_window_size;
        let force_empty_batches = expiry_epoch <= stage_origin.number;
        let first_of_epoch = epoch.number == parent.l1_origin.number + 1;
        let next_timestamp = parent.block_info.timestamp + self.cfg.block_time;

        // If the sequencer window did not expire,
        // there is still room to receive batches for the current epoch.
        // No need to force-create empty batch(es) towards the next epoch yet.
        if !force_empty_batches {
            return Err(PipelineError::Eof.temp());
        }

        // The next L1 block is needed to proceed towards the next epoch.
        if self.l1_blocks.len() < 2 {
            return Err(PipelineError::Eof.temp());
        }

        let next_epoch = self.l1_blocks[1];

        // Fill with empty L2 blocks of the same epoch until we meet the time of the next L1 origin,
        // to preserve that L2 time >= L1 time. If this is the first block of the epoch, always
        // generate a batch to ensure that we at least have one batch per epoch.
        if next_timestamp < next_epoch.timestamp || first_of_epoch {
            info!(target: "batch_validator", "Generating empty batch for epoch #{}", epoch.number);
            return Ok(SingleBatch {
                parent_hash: parent.block_info.hash,
                epoch_num: epoch.number,
                epoch_hash: epoch.hash,
                timestamp: next_timestamp,
                transactions: Vec::new(),
            });
        }

        // At this point we have auto generated every batch for the current epoch
        // that we can, so we can advance to the next epoch.
        debug!(
            target: "batch_validator",
            "Advancing batch validator epoch: {}, timestamp: {}, epoch timestamp: {}",
            next_epoch.number, next_timestamp, next_epoch.timestamp
        );
        self.l1_blocks.remove(0);
        Err(PipelineError::Eof.temp())
    }
}

#[async_trait]
impl<P> AttributesProvider for BatchValidator<P>
where
    P: NextBatchProvider + OriginAdvancer + OriginProvider + SignalReceiver + Send + Debug,
{
    async fn next_batch(&mut self, parent: L2BlockInfo) -> PipelineResult<SingleBatch> {
        // Update the L1 origin blocks within the stage.
        self.update_origins(&parent)?;

        // If the origin is behind, we must drain previous stages to catch up.
        let stage_origin = self.origin.ok_or(PipelineError::MissingOrigin.crit())?;
        if self.origin_behind(&parent) || parent.l1_origin.number == stage_origin.number {
            self.prev.next_batch(parent, self.l1_blocks.as_ref()).await?;
            return Err(PipelineError::NotEnoughData.temp());
        }

        // At least the L1 origin of the safe block and the L1 origin of the following block must
        // be included in the l1 blocks.
        if self.l1_blocks.len() < 2 {
            return Err(PipelineError::MissingOrigin.crit());
        }

        // Note: epoch origin can now be one block ahead of the L2 Safe Head
        // This is in the case where we auto generate all batches in an epoch & advance the epoch
        // but don't advance the L2 Safe Head's epoch
        let epoch = self.l1_blocks[0];
        if parent.l1_origin != epoch.id() && parent.l1_origin.number != epoch.number - 1 {
            return Err(PipelineErrorKind::Reset(ResetError::L1OriginMismatch(
                parent.l1_origin.number,
                epoch.number - 1,
            )));
        }

        // Pull the next batch from the previous stage.
        let next_batch = match self.prev.next_batch(parent, self.l1_blocks.as_ref()).await {
            Ok(batch) => batch,
            Err(PipelineErrorKind::Temporary(PipelineError::Eof)) => {
                return self.try_derive_empty_batch(&parent);
            }
            Err(e) => {
                return Err(e);
            }
        };

        // The batch must be a single batch - this stage does not support span batches.
        let Batch::Single(mut next_batch) = next_batch else {
            error!(
                target: "batch_validator",
                "BatchValidator received a batch that is not a SingleBatch"
            );
            return Err(PipelineError::InvalidBatchType.crit());
        };
        next_batch.parent_hash = parent.block_info.hash;

        // Check the validity of the single batch before forwarding it.
        match next_batch.check_batch(
            self.cfg.as_ref(),
            self.l1_blocks.as_ref(),
            parent,
            &stage_origin,
        ) {
            BatchValidity::Accept => {
                info!(target: "batch_validator", "Found next batch (epoch #{})", next_batch.epoch_num);
                Ok(next_batch)
            }
            BatchValidity::Past => {
                warn!(target: "batch_validator", "Dropping old batch");
                Err(PipelineError::NotEnoughData.temp())
            }
            BatchValidity::Drop => {
                warn!(target: "batch_validator", "Invalid singular batch, flushing current channel.");
                self.prev.flush();
                Err(PipelineError::NotEnoughData.temp())
            }
            BatchValidity::Undecided => Err(PipelineError::NotEnoughData.temp()),
            BatchValidity::Future => {
                error!(target: "batch_validator", "Future batch detected in BatchValidator.");
                Err(PipelineError::InvalidBatchValidity.crit())
            }
        }
    }

    fn is_last_in_span(&self) -> bool {
        self.prev.span_buffer_size() == 0
    }
}

impl<P> OriginProvider for BatchValidator<P>
where
    P: NextBatchProvider + OriginAdvancer + OriginProvider + SignalReceiver + Debug,
{
    fn origin(&self) -> Option<BlockInfo> {
        self.prev.origin()
    }
}

#[async_trait]
impl<P> OriginAdvancer for BatchValidator<P>
where
    P: NextBatchProvider + OriginAdvancer + OriginProvider + SignalReceiver + Send + Debug,
{
    async fn advance_origin(&mut self) -> PipelineResult<()> {
        self.prev.advance_origin().await
    }
}

#[async_trait]
impl<P> SignalReceiver for BatchValidator<P>
where
    P: NextBatchProvider + OriginAdvancer + OriginProvider + SignalReceiver + Send + Debug,
{
    async fn signal(&mut self, signal: Signal) -> PipelineResult<()> {
        match signal {
            s @ Signal::Reset(ResetSignal { l1_origin, .. }) => {
                self.prev.signal(s).await?;
                self.origin = Some(l1_origin);
                // Include the new origin as an origin to build on.
                // This is only for the initialization case.
                // During normal resets we will later throw out this block.
                self.l1_blocks.clear();
                self.l1_blocks.push(l1_origin);
            }
            s @ Signal::Activation(_) | s @ Signal::FlushChannel | s @ Signal::ProvideBlock(_) => {
                self.prev.signal(s).await?;
            }
        }
        Ok(())
    }
}

#[cfg(test)]
mod test {
    use crate::{
        AttributesProvider, BatchValidator, NextBatchProvider, OriginAdvancer, PipelineError,
        PipelineErrorKind, PipelineResult, ResetError, ResetSignal, Signal, SignalReceiver,
        test_utils::{CollectingLayer, TestNextBatchProvider, TraceStorage},
    };
    use alloc::{sync::Arc, vec, vec::Vec};
    use alloy_eips::{BlockNumHash, NumHash};
    use alloy_primitives::B256;
    use kona_genesis::{HardForkConfig, RollupConfig};
    use kona_protocol::{Batch, BlockInfo, L2BlockInfo, SingleBatch, SpanBatch};
    use tracing::Level;
    use tracing_subscriber::layer::SubscriberExt;

    #[tokio::test]
    async fn test_batch_validator_origin_behind_eof() {
        let cfg = Arc::new(RollupConfig::default());
        let mut mock = TestNextBatchProvider::new(vec![]);
        mock.origin = Some(BlockInfo::default());
        let mut bv = BatchValidator::new(cfg, mock);
        bv.origin = Some(BlockInfo { number: 1, ..Default::default() });

        let mock_parent = L2BlockInfo {
            l1_origin: BlockNumHash { number: 5, ..Default::default() },
            ..Default::default()
        };
        assert_eq!(bv.next_batch(mock_parent).await.unwrap_err(), PipelineError::Eof.temp());
    }

    #[tokio::test]
    async fn test_batch_validator_origin_behind_startup() {
        let cfg = Arc::new(RollupConfig::default());
        let mut mock = TestNextBatchProvider::new(vec![]);
        mock.origin = Some(BlockInfo::default());
        let mut bv = BatchValidator::new(cfg, mock);

        // Reset the pipeline to add the L1 origin to the stage.
        bv.signal(Signal::Reset(ResetSignal {
            l1_origin: BlockInfo { number: 1, ..Default::default() },
            l2_safe_head: L2BlockInfo::new(
                BlockInfo::default(),
                NumHash::new(1, Default::default()),
                0,
            ),
            system_config: None,
        }))
        .await
        .unwrap();

        let mock_parent = L2BlockInfo {
            l1_origin: BlockNumHash { number: 2, ..Default::default() },
            ..Default::default()
        };
        assert_eq!(bv.l1_blocks.len(), 1);
        bv.update_origins(&mock_parent).unwrap();
        assert_eq!(bv.l1_blocks.len(), 0);
    }

    #[tokio::test]
    async fn test_batch_validator_origin_behind_advance() {
        let cfg = Arc::new(RollupConfig::default());
        let mut mock = TestNextBatchProvider::new(vec![]);
        mock.origin = Some(BlockInfo { number: 2, ..Default::default() });
        let mut bv = BatchValidator::new(cfg, mock);

        // Reset the pipeline to add the L1 origin to the stage.
        bv.signal(Signal::Reset(ResetSignal {
            l1_origin: BlockInfo { number: 1, ..Default::default() },
            l2_safe_head: L2BlockInfo::new(
                BlockInfo::default(),
                NumHash::new(1, Default::default()),
                0,
            ),
            system_config: None,
        }))
        .await
        .unwrap();

        let mock_parent = L2BlockInfo {
            l1_origin: BlockNumHash { number: 1, ..Default::default() },
            ..Default::default()
        };
        assert_eq!(bv.l1_blocks.len(), 1);
        bv.update_origins(&mock_parent).unwrap();
        assert_eq!(bv.l1_blocks.len(), 2);
    }

    #[tokio::test]
    async fn test_batch_validator_advance_epoch() {
        let cfg = Arc::new(RollupConfig::default());
        let mut mock = TestNextBatchProvider::new(vec![]);
        mock.origin = Some(BlockInfo { number: 2, ..Default::default() });
        let mut bv = BatchValidator::new(cfg, mock);

        // Reset the pipeline to add the L1 origin to the stage.
        bv.signal(Signal::Reset(ResetSignal {
            l1_origin: BlockInfo { number: 1, ..Default::default() },
            l2_safe_head: L2BlockInfo::new(
                BlockInfo::default(),
                NumHash::new(1, Default::default()),
                0,
            ),
            system_config: None,
        }))
        .await
        .unwrap();

        let mock_parent = L2BlockInfo {
            l1_origin: BlockNumHash { number: 2, ..Default::default() },
            ..Default::default()
        };
        assert_eq!(bv.l1_blocks.len(), 1);
        assert_eq!(bv.l1_blocks[0].number, 1);
        assert_eq!(bv.next_batch(mock_parent).await.unwrap_err(), PipelineError::Eof.temp());
        assert_eq!(bv.l1_blocks.len(), 1);
        assert_eq!(bv.l1_blocks[0].number, 2);
    }

    #[tokio::test]
    async fn test_batch_validator_origin_behind_drain_prev() {
        let cfg = Arc::new(RollupConfig::default());
        let mut mock = TestNextBatchProvider::new(
            (0..5).map(|_| Ok(Batch::Single(SingleBatch::default()))).collect(),
        );
        mock.origin = Some(BlockInfo::default());
        let mut bv = BatchValidator::new(cfg, mock);
        bv.origin = Some(BlockInfo::default());

        let mock_parent = L2BlockInfo {
            l1_origin: BlockNumHash { number: 5, ..Default::default() },
            ..Default::default()
        };
        assert_eq!(bv.prev.span_buffer_size(), 5);
        for i in 0..5 {
            assert_eq!(
                bv.next_batch(mock_parent).await.unwrap_err(),
                PipelineError::NotEnoughData.temp()
            );
            assert_eq!(bv.prev.span_buffer_size(), 4 - i);
        }
        assert_eq!(bv.next_batch(mock_parent).await.unwrap_err(), PipelineError::Eof.temp());
    }

    #[tokio::test]
    async fn test_batch_validator_l1_origin_mismatch() {
        let cfg = Arc::new(RollupConfig::default());
        let mut mock = TestNextBatchProvider::new(vec![Ok(Batch::Single(SingleBatch::default()))]);
        mock.origin = Some(BlockInfo { number: 1, ..Default::default() });
        let mut bv = BatchValidator::new(cfg, mock);
        bv.origin = Some(BlockInfo::default());
        bv.l1_blocks.push(BlockInfo::default());

        let mock_parent = L2BlockInfo {
            l1_origin: BlockNumHash { number: 0, hash: [0xFF; 32].into() },
            ..Default::default()
        };

        assert!(matches!(
            bv.next_batch(mock_parent).await.unwrap_err(),
            PipelineErrorKind::Reset(ResetError::L1OriginMismatch(_, _))
        ));
    }

    #[tokio::test]
    async fn test_batch_validator_received_span_batch() {
        let cfg = Arc::new(RollupConfig::default());
        let mut mock = TestNextBatchProvider::new(vec![Ok(Batch::Span(SpanBatch::default()))]);
        mock.origin = Some(BlockInfo { number: 1, ..Default::default() });
        let mut bv = BatchValidator::new(cfg, mock);
        bv.origin = Some(BlockInfo::default());
        bv.l1_blocks.push(BlockInfo::default());

        let mock_parent = L2BlockInfo {
            l1_origin: BlockNumHash { number: 0, ..Default::default() },
            ..Default::default()
        };

        assert_eq!(
            bv.next_batch(mock_parent).await.unwrap_err(),
            PipelineError::InvalidBatchType.crit()
        );
        assert_eq!(bv.next_batch(mock_parent).await.unwrap_err(), PipelineError::Eof.temp());
    }

    #[tokio::test]
    async fn test_batch_validator_next_batch_valid() {
        let cfg = Arc::new(RollupConfig {
            hardforks: HardForkConfig { holocene_time: Some(0), ..Default::default() },
            block_time: 2,
            max_sequencer_drift: 700,
            ..Default::default()
        });
        assert!(cfg.is_holocene_active(0));
        let batch = SingleBatch {
            parent_hash: B256::default(),
            epoch_num: 2,
            epoch_hash: B256::default(),
            timestamp: 4,
            transactions: Vec::new(),
        };
        let parent = L2BlockInfo {
            l1_origin: BlockNumHash { number: 0, ..Default::default() },
            block_info: BlockInfo { timestamp: 2, ..Default::default() },
            ..Default::default()
        };

        // Setup batch validator deps
        let batch_vec = vec![PipelineResult::Ok(Batch::Single(batch.clone()))];
        let mut mock = TestNextBatchProvider::new(batch_vec);
        mock.origin = Some(BlockInfo { number: 1, ..Default::default() });

        // Configure batch validator
        let mut bv = BatchValidator::new(cfg, mock);

        // Reset the pipeline to add the L1 origin to the stage.
        bv.signal(Signal::Reset(ResetSignal {
            l1_origin: BlockInfo { number: 1, ..Default::default() },
            ..Default::default()
        }))
        .await
        .unwrap();
        bv.l1_blocks.push(BlockInfo { number: 1, ..Default::default() });

        // Grab the next batch.
        let produced_batch = bv.next_batch(parent).await.unwrap();
        assert_eq!(batch, produced_batch);
    }

    #[tokio::test]
    async fn test_batch_validator_next_batch_sequence_window_expired() {
        let trace_store: TraceStorage = Default::default();
        let layer = CollectingLayer::new(trace_store.clone());
        let subscriber = tracing_subscriber::Registry::default().with(layer);
        let _guard = tracing::subscriber::set_default(subscriber);

        let cfg = Arc::new(RollupConfig { seq_window_size: 5, ..Default::default() });
        let mut mock = TestNextBatchProvider::new(vec![]);
        mock.origin = Some(BlockInfo { number: 1, ..Default::default() });
        let mut bv = BatchValidator::new(cfg, mock);

        // Reset the pipeline to add the L1 origin to the stage.
        bv.signal(Signal::Reset(ResetSignal {
            l1_origin: BlockInfo { number: 1, ..Default::default() },
            ..Default::default()
        }))
        .await
        .unwrap();

        // Advance the origin of the previous stage to block #6.
        for _ in 0..6 {
            bv.advance_origin().await.unwrap();
        }

        // The sequence window is expired, so we should generate an empty batch.
        let mock_parent = L2BlockInfo {
            l1_origin: BlockNumHash { number: 0, ..Default::default() },
            ..Default::default()
        };
        assert!(bv.next_batch(mock_parent).await.unwrap().transactions.is_empty());

        let trace_lock = trace_store.lock();
        assert_eq!(trace_lock.iter().filter(|(l, _)| matches!(l, &Level::DEBUG)).count(), 1);
        assert_eq!(trace_lock.iter().filter(|(l, _)| matches!(l, &Level::INFO)).count(), 1);
        assert!(trace_lock[0].1.contains("Advancing batch validator origin"));
        assert!(trace_lock[1].1.contains("Generating empty batch for epoch"));
    }

    #[tokio::test]
    async fn test_batch_validator_next_batch_sequence_window_expired_advance_epoch() {
        let trace_store: TraceStorage = Default::default();
        let layer = CollectingLayer::new(trace_store.clone());
        let subscriber = tracing_subscriber::Registry::default().with(layer);
        let _guard = tracing::subscriber::set_default(subscriber);

        let cfg = Arc::new(RollupConfig { seq_window_size: 5, ..Default::default() });
        let mut mock = TestNextBatchProvider::new(vec![]);
        mock.origin = Some(BlockInfo { number: 1, ..Default::default() });
        let mut bv = BatchValidator::new(cfg, mock);

        // Reset the pipeline to add the L1 origin to the stage.
        bv.signal(Signal::Reset(ResetSignal {
            l1_origin: BlockInfo { number: 1, ..Default::default() },
            ..Default::default()
        }))
        .await
        .unwrap();

        // Advance the origin of the previous stage to block #6.
        for _ in 0..6 {
            bv.advance_origin().await.unwrap();
        }

        // The sequence window is expired, so we should generate an empty batch.
        let mock_parent = L2BlockInfo {
            l1_origin: BlockNumHash { number: 1, ..Default::default() },
            ..Default::default()
        };
        assert_eq!(bv.next_batch(mock_parent).await.unwrap_err(), PipelineError::Eof.temp());

        let trace_lock = trace_store.lock();
        assert_eq!(trace_lock.iter().filter(|(l, _)| matches!(l, &Level::DEBUG)).count(), 2);
        assert!(trace_lock[0].1.contains("Advancing batch validator origin"));
        assert!(trace_lock[1].1.contains("Advancing batch validator epoch"));
    }
}
