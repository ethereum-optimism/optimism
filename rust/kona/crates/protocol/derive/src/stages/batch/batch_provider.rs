//! This module contains the [`BatchProvider`] stage.

use super::NextBatchProvider;
use crate::{
    AttributesProvider, BatchQueue, BatchValidator, L2ChainProvider, OriginAdvancer,
    OriginProvider, PipelineError, PipelineResult, Stage,
};
use alloc::{boxed::Box, sync::Arc};
use alloy_eips::BlockNumHash;
use async_trait::async_trait;
use core::fmt::Debug;
use kona_genesis::{RollupConfig, SystemConfig};
use kona_protocol::{BlockInfo, L2BlockInfo, SingleBatch};

/// The [`BatchProvider`] stage is a mux between the [`BatchQueue`] and [`BatchValidator`] stages.
///
/// Rules:
/// When Holocene is not active, the [`BatchQueue`] is used.
/// When Holocene is active, the [`BatchValidator`] is used.
///
/// When transitioning between the two stages, the mux will reset the active stage, but
/// retain `l1_blocks` and `origin`.
#[derive(Debug)]
pub struct BatchProvider<P, F>
where
    P: NextBatchProvider + OriginAdvancer + OriginProvider + Stage + Debug,
    F: L2ChainProvider + Clone + Debug,
{
    /// The rollup configuration.
    pub cfg: Arc<RollupConfig>,
    /// The L2 chain provider.
    pub provider: F,
    /// The previous stage of the derivation pipeline.
    ///
    /// If this is set to [`None`], the multiplexer has been activated and the active stage
    /// owns the previous stage.
    ///
    /// Must be [`None`] if `batch_queue` or `batch_validator` is [`Some`].
    pub prev: Option<P>,
    /// The batch queue stage of the provider.
    ///
    /// Must be [`None`] if `prev` or `batch_validator` is [`Some`].
    pub batch_queue: Option<BatchQueue<P, F>>,
    /// The batch validator stage of the provider.
    ///
    /// Must be [`None`] if `prev` or `batch_queue` is [`Some`].
    pub batch_validator: Option<BatchValidator<P>>,
}

impl<P, F> BatchProvider<P, F>
where
    P: NextBatchProvider + OriginAdvancer + OriginProvider + Stage + Debug,
    F: L2ChainProvider + Clone + Debug,
{
    /// Creates a new [`BatchProvider`] with the given configuration and previous stage.
    pub const fn new(cfg: Arc<RollupConfig>, prev: P, provider: F) -> Self {
        Self { cfg, provider, prev: Some(prev), batch_queue: None, batch_validator: None }
    }

    /// Attempts to update the active stage of the mux.
    pub(crate) fn attempt_update(&mut self) -> PipelineResult<()> {
        let origin = self.origin().ok_or(PipelineError::MissingOrigin.crit())?;
        if let Some(prev) = self.prev.take() {
            // On the first call to `attempt_update`, we need to determine the active stage to
            // initialize the mux with.
            if self.cfg.is_holocene_active(origin.timestamp) {
                self.batch_validator = Some(BatchValidator::new(self.cfg.clone(), prev));
            } else {
                self.batch_queue =
                    Some(BatchQueue::new(self.cfg.clone(), prev, self.provider.clone()));
            }
        } else if self.batch_queue.is_some() && self.cfg.is_holocene_active(origin.timestamp) {
            // If the batch queue is active and Holocene is also active, transition to the batch
            // validator.
            let batch_queue = self.batch_queue.take().expect("Must have batch queue");
            let mut bv = BatchValidator::new(self.cfg.clone(), batch_queue.prev);
            bv.l1_blocks = batch_queue.l1_blocks;
            bv.origin = batch_queue.origin;
            self.batch_validator = Some(bv);
        } else if self.batch_validator.is_some() && !self.cfg.is_holocene_active(origin.timestamp) {
            // If the batch validator is active, and Holocene is not active, it indicates an L1
            // reorg around Holocene activation. Transition back to the batch queue
            // until Holocene re-activates.
            let batch_validator = self.batch_validator.take().expect("Must have batch validator");
            let mut bq =
                BatchQueue::new(self.cfg.clone(), batch_validator.prev, self.provider.clone());
            bq.l1_blocks = batch_validator.l1_blocks;
            self.batch_queue = Some(bq);
        }
        Ok(())
    }
}

#[async_trait]
impl<P, F> OriginAdvancer for BatchProvider<P, F>
where
    P: NextBatchProvider + OriginAdvancer + OriginProvider + Stage + Send + Debug,
    F: L2ChainProvider + Clone + Send + Debug,
{
    async fn advance_origin(&mut self) -> PipelineResult<()> {
        self.attempt_update()?;

        if let Some(batch_validator) = self.batch_validator.as_mut() {
            batch_validator.advance_origin().await
        } else if let Some(batch_queue) = self.batch_queue.as_mut() {
            batch_queue.advance_origin().await
        } else {
            Err(PipelineError::NotEnoughData.temp())
        }
    }
}

impl<P, F> OriginProvider for BatchProvider<P, F>
where
    P: NextBatchProvider + OriginAdvancer + OriginProvider + Stage + Debug,
    F: L2ChainProvider + Clone + Debug,
{
    fn origin(&self) -> Option<BlockInfo> {
        self.batch_validator.as_ref().map_or_else(
            || {
                self.batch_queue.as_ref().map_or_else(
                    || self.prev.as_ref().and_then(|prev| prev.origin()),
                    |batch_queue| batch_queue.origin(),
                )
            },
            |batch_validator| batch_validator.origin(),
        )
    }
}

/// Dispatches a method call to the active inner batch stage.
macro_rules! dispatch_inner {
    ($self:ident, $method:ident $(, $arg:expr)*) => {{
        $self.attempt_update()?;
        if let Some(inner) = $self.batch_validator.as_mut() {
            inner.$method($($arg),*).await
        } else if let Some(inner) = $self.batch_queue.as_mut() {
            inner.$method($($arg),*).await
        } else {
            Err(PipelineError::NotEnoughData.temp())
        }
    }};
}

#[async_trait]
impl<P, F> Stage for BatchProvider<P, F>
where
    P: NextBatchProvider + OriginAdvancer + OriginProvider + Stage + Send + Debug,
    F: L2ChainProvider + Clone + Send + Debug,
{
    async fn reset(
        &mut self,
        l1_origin: BlockNumHash,
        system_config: SystemConfig,
    ) -> PipelineResult<()> {
        dispatch_inner!(self, reset, l1_origin, system_config)
    }

    async fn activate(&mut self) -> PipelineResult<()> {
        dispatch_inner!(self, activate)
    }

    async fn flush_channel(&mut self) -> PipelineResult<()> {
        dispatch_inner!(self, flush_channel)
    }

    async fn provide_block(&mut self, block: BlockInfo) -> PipelineResult<()> {
        dispatch_inner!(self, provide_block, block)
    }
}

#[async_trait]
impl<P, F> AttributesProvider for BatchProvider<P, F>
where
    P: NextBatchProvider + OriginAdvancer + OriginProvider + Stage + Debug + Send,
    F: L2ChainProvider + Clone + Send + Debug,
{
    fn is_last_in_span(&self) -> bool {
        self.batch_validator.as_ref().map_or_else(
            || self.batch_queue.as_ref().is_some_and(|batch_queue| batch_queue.is_last_in_span()),
            |batch_validator| batch_validator.is_last_in_span(),
        )
    }

    async fn next_batch(&mut self, parent: L2BlockInfo) -> PipelineResult<SingleBatch> {
        self.attempt_update()?;

        if let Some(batch_validator) = self.batch_validator.as_mut() {
            batch_validator.next_batch(parent).await
        } else if let Some(batch_queue) = self.batch_queue.as_mut() {
            batch_queue.next_batch(parent).await
        } else {
            Err(PipelineError::NotEnoughData.temp())
        }
    }
}

#[cfg(test)]
mod test {
    use super::BatchProvider;
    use crate::{
        test_utils::{TestL2ChainProvider, TestNextBatchProvider},
        traits::{OriginProvider, Stage},
    };
    use alloc::{sync::Arc, vec};
    use alloy_eips::BlockNumHash;
    use kona_genesis::{HardForkConfig, RollupConfig, SystemConfig};
    use kona_protocol::BlockInfo;

    #[test]
    fn test_batch_provider_validator_active() {
        let provider = TestNextBatchProvider::new(vec![]);
        let l2_provider = TestL2ChainProvider::default();
        let cfg = Arc::new(RollupConfig {
            hardforks: HardForkConfig { holocene_time: Some(0), ..Default::default() },
            ..Default::default()
        });
        let mut batch_provider = BatchProvider::new(cfg, provider, l2_provider);

        assert!(batch_provider.attempt_update().is_ok());
        assert!(batch_provider.prev.is_none());
        assert!(batch_provider.batch_queue.is_none());
        assert!(batch_provider.batch_validator.is_some());
    }

    #[test]
    fn test_batch_provider_batch_queue_active() {
        let provider = TestNextBatchProvider::new(vec![]);
        let l2_provider = TestL2ChainProvider::default();
        let cfg = Arc::new(RollupConfig::default());
        let mut batch_provider = BatchProvider::new(cfg, provider, l2_provider);

        assert!(batch_provider.attempt_update().is_ok());
        assert!(batch_provider.prev.is_none());
        assert!(batch_provider.batch_queue.is_some());
        assert!(batch_provider.batch_validator.is_none());
    }

    #[test]
    fn test_batch_provider_transition_stage() {
        let provider = TestNextBatchProvider::new(vec![]);
        let l2_provider = TestL2ChainProvider::default();
        let cfg = Arc::new(RollupConfig {
            hardforks: HardForkConfig { holocene_time: Some(2), ..Default::default() },
            ..Default::default()
        });
        let mut batch_provider = BatchProvider::new(cfg, provider, l2_provider);

        batch_provider.attempt_update().unwrap();

        // Update the L1 origin to Holocene activation.
        let Some(ref mut stage) = batch_provider.batch_queue else {
            panic!("Expected BatchQueue");
        };
        stage.prev.origin = Some(BlockInfo { number: 1, timestamp: 2, ..Default::default() });

        // Transition to the BatchValidator stage.
        batch_provider.attempt_update().unwrap();
        assert!(batch_provider.batch_queue.is_none());
        assert!(batch_provider.batch_validator.is_some());

        assert_eq!(batch_provider.origin().unwrap().number, 1);
    }

    #[test]
    fn test_batch_provider_transition_stage_backwards() {
        let provider = TestNextBatchProvider::new(vec![]);
        let l2_provider = TestL2ChainProvider::default();
        let cfg = Arc::new(RollupConfig {
            hardforks: HardForkConfig { holocene_time: Some(2), ..Default::default() },
            ..Default::default()
        });
        let mut batch_provider = BatchProvider::new(cfg, provider, l2_provider);

        batch_provider.attempt_update().unwrap();

        // Update the L1 origin to Holocene activation.
        let Some(ref mut stage) = batch_provider.batch_queue else {
            panic!("Expected BatchQueue");
        };
        stage.prev.origin = Some(BlockInfo { number: 1, timestamp: 2, ..Default::default() });

        // Transition to the BatchValidator stage.
        batch_provider.attempt_update().unwrap();
        assert!(batch_provider.batch_queue.is_none());
        assert!(batch_provider.batch_validator.is_some());

        // Update the L1 origin to before Holocene activation, to simulate a re-org.
        let Some(ref mut stage) = batch_provider.batch_validator else {
            panic!("Expected BatchValidator");
        };
        stage.prev.origin = Some(BlockInfo::default());

        batch_provider.attempt_update().unwrap();
        assert!(batch_provider.batch_queue.is_some());
        assert!(batch_provider.batch_validator.is_none());
    }

    #[tokio::test]
    async fn test_batch_provider_reset_bq() {
        let provider = TestNextBatchProvider::new(vec![]);
        let l2_provider = TestL2ChainProvider::default();
        let cfg = Arc::new(RollupConfig::default());
        let mut batch_provider = BatchProvider::new(cfg, provider, l2_provider);

        // Reset the batch provider.
        batch_provider.reset(BlockNumHash::default(), SystemConfig::default()).await.unwrap();

        let Some(bq) = batch_provider.batch_queue else {
            panic!("Expected BatchQueue");
        };
        assert_eq!(bq.l1_blocks.len(), 1);
    }

    #[tokio::test]
    async fn test_batch_provider_reset_validator() {
        let provider = TestNextBatchProvider::new(vec![]);
        let l2_provider = TestL2ChainProvider::default();
        let cfg = Arc::new(RollupConfig {
            hardforks: HardForkConfig { holocene_time: Some(0), ..Default::default() },
            ..Default::default()
        });
        let mut batch_provider = BatchProvider::new(cfg, provider, l2_provider);

        // Reset the batch provider.
        batch_provider.reset(BlockNumHash::default(), SystemConfig::default()).await.unwrap();

        let Some(bv) = batch_provider.batch_validator else {
            panic!("Expected BatchValidator");
        };
        assert_eq!(bv.l1_blocks.len(), 1);
    }

    // On Holocene activation, BatchProvider.attempt_update() must copy BOTH l1_blocks
    // AND origin from the old BatchQueue to the new BatchValidator.
    //
    // Without copying origin, BatchValidator.origin starts as None. The first
    // update_origins() call always enters the `self.origin != self.prev.origin()` branch
    // (None != Some(...)), causing either duplicate l1_block insertion (normal case) or
    // l1_blocks.clear() followed by MissingOrigin.crit() halt (lagging case).
    #[test]
    fn test_spec_batch_provider_holocene_transition_origin_transferred() {
        let provider = TestNextBatchProvider::new(vec![]);
        let l2_provider = TestL2ChainProvider::default();
        // Holocene activates at timestamp 2.
        let cfg = Arc::new(RollupConfig {
            hardforks: HardForkConfig { holocene_time: Some(2), ..Default::default() },
            ..Default::default()
        });
        let mut batch_provider = BatchProvider::new(cfg, provider, l2_provider);

        // Initialize with BatchQueue (Holocene not yet active at timestamp 0).
        batch_provider.attempt_update().unwrap();
        assert!(batch_provider.batch_queue.is_some(), "Expected BatchQueue pre-Holocene");

        // Seed BatchQueue.l1_blocks with two blocks (as would happen during normal derivation).
        let block_a = BlockInfo { number: 1, timestamp: 0, ..Default::default() };
        let block_b = BlockInfo { number: 2, timestamp: 2, ..Default::default() };
        {
            let bq = batch_provider.batch_queue.as_mut().unwrap();
            bq.l1_blocks.push(block_a);
            bq.l1_blocks.push(block_b);
            bq.origin = Some(block_b); // BatchQueue.origin set to the current L1 head
            // Advance the mock prev origin to Holocene activation timestamp.
            bq.prev.origin = Some(block_b);
        }

        // Trigger Holocene transition via attempt_update().
        batch_provider.attempt_update().unwrap();
        assert!(batch_provider.batch_queue.is_none(), "BatchQueue should be gone post-Holocene");
        assert!(batch_provider.batch_validator.is_some(), "Expected BatchValidator post-Holocene");

        let bv = batch_provider.batch_validator.as_ref().unwrap();

        // Verify l1_blocks were transferred.
        assert_eq!(bv.l1_blocks.len(), 2, "l1_blocks must be transferred from BatchQueue");
        assert_eq!(bv.l1_blocks[0], block_a);
        assert_eq!(bv.l1_blocks[1], block_b);

        // Verify origin was transferred: BatchValidator.origin must equal the
        // origin from the old BatchQueue, matching Go's TransformHolocene (batch_mux.go:68).
        assert_eq!(
            bv.origin,
            Some(block_b),
            "BatchValidator.origin must be copied from BatchQueue on Holocene transition"
        );
    }
}
