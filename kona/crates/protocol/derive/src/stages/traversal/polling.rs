//! Contains the [`PollingTraversal`] stage of the derivation pipeline.

use crate::{
    ActivationSignal, ChainProvider, L1RetrievalProvider, OriginAdvancer, OriginProvider,
    PipelineError, PipelineResult, ResetError, ResetSignal, Signal, SignalReceiver,
};
use alloc::{boxed::Box, sync::Arc};
use alloy_primitives::Address;
use async_trait::async_trait;
use kona_genesis::{RollupConfig, SystemConfig};
use kona_protocol::BlockInfo;

/// The [`PollingTraversal`] stage of the derivation pipeline.
///
/// This stage sits at the bottom of the pipeline, holding a handle to the data source
/// (a [`ChainProvider`] implementation) and the current L1 [`BlockInfo`] in the pipeline,
/// which are used to traverse the L1 chain. When the [`PollingTraversal`] stage is advanced,
/// it fetches the next L1 [`BlockInfo`] from the data source and updates the [`SystemConfig`]
/// with the receipts from the block.
#[derive(Debug, Clone)]
pub struct PollingTraversal<Provider: ChainProvider> {
    /// The current block in the traversal stage.
    pub block: Option<BlockInfo>,
    /// The data source for the traversal stage.
    pub data_source: Provider,
    /// Signals whether or not the traversal stage is complete.
    pub done: bool,
    /// The system config.
    pub system_config: SystemConfig,
    /// A reference to the rollup config.
    pub rollup_config: Arc<RollupConfig>,
}

#[async_trait]
impl<F: ChainProvider + Send> L1RetrievalProvider for PollingTraversal<F> {
    fn batcher_addr(&self) -> Address {
        self.system_config.batcher_address
    }

    async fn next_l1_block(&mut self) -> PipelineResult<Option<BlockInfo>> {
        if !self.done {
            self.done = true;
            Ok(self.block)
        } else {
            Err(PipelineError::Eof.temp())
        }
    }
}

impl<F: ChainProvider> PollingTraversal<F> {
    /// Creates a new [`PollingTraversal`] instance.
    pub fn new(data_source: F, cfg: Arc<RollupConfig>) -> Self {
        Self {
            block: Some(BlockInfo::default()),
            data_source,
            done: false,
            system_config: SystemConfig::default(),
            rollup_config: cfg,
        }
    }

    /// Update the origin block in the traversal stage.
    fn update_origin(&mut self, block: BlockInfo) {
        self.done = false;
        self.block = Some(block);
        kona_macros::set!(gauge, crate::metrics::Metrics::PIPELINE_ORIGIN, block.number as f64);
    }
}

#[async_trait]
impl<F: ChainProvider + Send> OriginAdvancer for PollingTraversal<F> {
    /// Advances the internal state of the [`PollingTraversal`] stage to the next L1 block.
    /// This function fetches the next L1 [`BlockInfo`] from the data source and updates the
    /// [`SystemConfig`] with the receipts from the block.
    async fn advance_origin(&mut self) -> PipelineResult<()> {
        // Advance start time for metrics.
        #[cfg(feature = "metrics")]
        let start_time = std::time::Instant::now();

        // Pull the next block or return EOF.
        // PipelineError::EOF has special handling further up the pipeline.
        let block = match self.block {
            Some(block) => block,
            None => {
                warn!(target: "l1_traversal",  "Missing current block, can't advance origin with no reference.");
                return Err(PipelineError::Eof.temp());
            }
        };
        let next_l1_origin =
            self.data_source.block_info_by_number(block.number + 1).await.map_err(Into::into)?;

        // Check block hashes for reorgs.
        if block.hash != next_l1_origin.parent_hash {
            return Err(ResetError::ReorgDetected(block.hash, next_l1_origin.parent_hash).into());
        }

        // Fetch receipts for the next l1 block and update the system config.
        let receipts =
            self.data_source.receipts_by_hash(next_l1_origin.hash).await.map_err(Into::into)?;

        let addr = self.rollup_config.l1_system_config_address;
        let active = self.rollup_config.is_ecotone_active(next_l1_origin.timestamp);
        match self.system_config.update_with_receipts(&receipts[..], addr, active) {
            Ok(true) => {
                let next = next_l1_origin.number as f64;
                kona_macros::set!(gauge, crate::Metrics::PIPELINE_LATEST_SYS_CONFIG_UPDATE, next);
                info!(target: "l1_traversal", "System config updated at block {next}.");
            }
            Ok(false) => { /* Ignore, no update applied */ }
            Err(err) => {
                error!(target: "l1_traversal", ?err, "Failed to update system config at block {}", next_l1_origin.number);
                kona_macros::set!(
                    gauge,
                    crate::Metrics::PIPELINE_SYS_CONFIG_UPDATE_ERROR,
                    next_l1_origin.number as f64
                );
                return Err(PipelineError::SystemConfigUpdate(err).crit());
            }
        }

        let prev_block_holocene = self.rollup_config.is_holocene_active(block.timestamp);
        let next_block_holocene = self.rollup_config.is_holocene_active(next_l1_origin.timestamp);

        // Update the block origin regardless of if a holocene activation is required.
        self.update_origin(next_l1_origin);

        // Record the origin as advanced.
        #[cfg(feature = "metrics")]
        {
            let duration = start_time.elapsed();
            kona_macros::record!(
                histogram,
                crate::metrics::Metrics::PIPELINE_ORIGIN_ADVANCE,
                duration.as_secs_f64()
            );
        }

        // If the prev block is not holocene, but the next is, we need to flag this
        // so the pipeline driver will reset the pipeline for holocene activation.
        if !prev_block_holocene && next_block_holocene {
            return Err(ResetError::HoloceneActivation.reset());
        }

        Ok(())
    }
}

impl<F: ChainProvider> OriginProvider for PollingTraversal<F> {
    fn origin(&self) -> Option<BlockInfo> {
        self.block
    }
}

#[async_trait]
impl<F: ChainProvider + Send> SignalReceiver for PollingTraversal<F> {
    async fn signal(&mut self, signal: Signal) -> PipelineResult<()> {
        match signal {
            Signal::Reset(ResetSignal { l1_origin, system_config, .. }) |
            Signal::Activation(ActivationSignal { l1_origin, system_config, .. }) => {
                self.update_origin(l1_origin);
                self.system_config = system_config.expect("System config must be provided.");
            }
            Signal::ProvideBlock(_) => {
                /* Not supported in this stage. */
                warn!(target: "traversal", "ProvideBlock signal not supported in PollingTraversal stage.");
                return Err(PipelineError::UnsupportedSignal.temp());
            }
            _ => {}
        }

        Ok(())
    }
}

#[cfg(test)]
pub(crate) mod tests {
    use super::*;
    use crate::{errors::PipelineErrorKind, test_utils::TraversalTestHelper};
    use alloc::vec;
    use alloy_primitives::{address, b256};

    #[test]
    fn test_l1_traversal_batcher_address() {
        let mut traversal = TraversalTestHelper::new_populated();
        traversal.system_config.batcher_address = TraversalTestHelper::L1_SYS_CONFIG_ADDR;
        assert_eq!(traversal.batcher_addr(), TraversalTestHelper::L1_SYS_CONFIG_ADDR);
    }

    #[tokio::test]
    async fn test_l1_traversal_flush_channel() {
        let blocks = vec![BlockInfo::default(), BlockInfo::default()];
        let receipts = TraversalTestHelper::new_receipts();
        let mut traversal = TraversalTestHelper::new_from_blocks(blocks, receipts);
        assert!(traversal.advance_origin().await.is_ok());
        traversal.done = true;
        assert!(traversal.signal(Signal::FlushChannel).await.is_ok());
        assert_eq!(traversal.origin(), Some(BlockInfo::default()));
        assert!(traversal.done);
    }

    #[tokio::test]
    async fn test_l1_traversal_activation_signal() {
        let blocks = vec![BlockInfo::default(), BlockInfo::default()];
        let receipts = TraversalTestHelper::new_receipts();
        let mut traversal = TraversalTestHelper::new_from_blocks(blocks, receipts);
        assert!(traversal.advance_origin().await.is_ok());
        let cfg = SystemConfig::default();
        traversal.done = true;
        assert!(
            traversal
                .signal(
                    ActivationSignal { system_config: Some(cfg), ..Default::default() }.signal()
                )
                .await
                .is_ok()
        );
        assert_eq!(traversal.origin(), Some(BlockInfo::default()));
        assert_eq!(traversal.system_config, cfg);
        assert!(!traversal.done);
    }

    #[tokio::test]
    async fn test_l1_traversal_reset_signal() {
        let blocks = vec![BlockInfo::default(), BlockInfo::default()];
        let receipts = TraversalTestHelper::new_receipts();
        let mut traversal = TraversalTestHelper::new_from_blocks(blocks, receipts);
        assert!(traversal.advance_origin().await.is_ok());
        let cfg = SystemConfig::default();
        traversal.done = true;
        assert!(
            traversal
                .signal(ResetSignal { system_config: Some(cfg), ..Default::default() }.signal())
                .await
                .is_ok()
        );
        assert_eq!(traversal.origin(), Some(BlockInfo::default()));
        assert_eq!(traversal.system_config, cfg);
        assert!(!traversal.done);
    }

    #[tokio::test]
    async fn test_l1_traversal() {
        let blocks = vec![BlockInfo::default(), BlockInfo::default()];
        let receipts = TraversalTestHelper::new_receipts();
        let mut traversal = TraversalTestHelper::new_from_blocks(blocks, receipts);
        assert_eq!(traversal.next_l1_block().await.unwrap(), Some(BlockInfo::default()));
        assert_eq!(traversal.next_l1_block().await.unwrap_err(), PipelineError::Eof.temp());
        assert!(traversal.advance_origin().await.is_ok());
    }

    #[tokio::test]
    async fn test_l1_traversal_missing_receipts() {
        let blocks = vec![BlockInfo::default(), BlockInfo::default()];
        let mut traversal = TraversalTestHelper::new_from_blocks(blocks, vec![]);
        assert_eq!(traversal.next_l1_block().await.unwrap(), Some(BlockInfo::default()));
        assert_eq!(traversal.next_l1_block().await.unwrap_err(), PipelineError::Eof.temp());
        matches!(
            traversal.advance_origin().await.unwrap_err(),
            PipelineErrorKind::Temporary(PipelineError::Provider(_))
        );
    }

    #[tokio::test]
    async fn test_l1_traversal_reorgs() {
        let hash = b256!("3333333333333333333333333333333333333333333333333333333333333333");
        let block = BlockInfo { hash, ..BlockInfo::default() };
        let blocks = vec![block, block];
        let receipts = TraversalTestHelper::new_receipts();
        let mut traversal = TraversalTestHelper::new_from_blocks(blocks, receipts);
        assert!(traversal.advance_origin().await.is_ok());
        let err = traversal.advance_origin().await.unwrap_err();
        assert_eq!(err, ResetError::ReorgDetected(block.hash, block.parent_hash).into());
    }

    #[tokio::test]
    async fn test_l1_traversal_missing_blocks() {
        let mut traversal = TraversalTestHelper::new_from_blocks(vec![], vec![]);
        assert_eq!(traversal.next_l1_block().await.unwrap(), Some(BlockInfo::default()));
        assert_eq!(traversal.next_l1_block().await.unwrap_err(), PipelineError::Eof.temp());
        matches!(
            traversal.advance_origin().await.unwrap_err(),
            PipelineErrorKind::Temporary(PipelineError::Provider(_))
        );
    }

    #[tokio::test]
    async fn test_l1_traversal_system_config_update_fails() {
        let first = b256!("3333333333333333333333333333333333333333333333333333333333333333");
        let second = b256!("4444444444444444444444444444444444444444444444444444444444444444");
        let block1 = BlockInfo { hash: first, ..BlockInfo::default() };
        let block2 = BlockInfo { hash: second, ..BlockInfo::default() };
        let blocks = vec![block1, block2];
        let receipts = TraversalTestHelper::new_receipts();
        let mut traversal = TraversalTestHelper::new_from_blocks(blocks, receipts);
        assert!(traversal.advance_origin().await.is_ok());
        // Only the second block should fail since the second receipt
        // contains invalid logs that will error for a system config update.
        let err = traversal.advance_origin().await.unwrap_err();
        matches!(err, PipelineErrorKind::Critical(PipelineError::SystemConfigUpdate(_)));
    }

    #[tokio::test]
    async fn test_l1_traversal_system_config_updated() {
        let blocks = vec![BlockInfo::default(), BlockInfo::default()];
        let receipts = TraversalTestHelper::new_receipts();
        let mut traversal = TraversalTestHelper::new_from_blocks(blocks, receipts);
        assert_eq!(traversal.next_l1_block().await.unwrap(), Some(BlockInfo::default()));
        assert_eq!(traversal.next_l1_block().await.unwrap_err(), PipelineError::Eof.temp());
        assert!(traversal.advance_origin().await.is_ok());
        let expected = address!("000000000000000000000000000000000000bEEF");
        assert_eq!(traversal.system_config.batcher_address, expected);
    }
}
