//! Contains the [`IndexedTraversal`] stage of the derivation pipeline.

use crate::{
    ActivationSignal, ChainProvider, L1RetrievalProvider, OriginAdvancer, OriginProvider,
    PipelineError, PipelineResult, ResetError, ResetSignal, Signal, SignalReceiver,
};
use alloc::{boxed::Box, sync::Arc};
use alloy_primitives::Address;
use async_trait::async_trait;
use kona_genesis::{RollupConfig, SystemConfig};
use kona_protocol::BlockInfo;

/// The [`IndexedTraversal`] stage of the derivation pipeline.
///
/// This stage sits at the bottom of the pipeline, holding a handle to the data source
/// (a [`ChainProvider`] implementation) and the current L1 [`BlockInfo`] in the pipeline,
/// which are used to traverse the L1 chain. When the [`IndexedTraversal`] stage is advanced,
/// it fetches the next L1 [`BlockInfo`] from the data source and updates the [`SystemConfig`]
/// with the receipts from the block.
#[derive(Debug, Clone)]
pub struct IndexedTraversal<Provider: ChainProvider> {
    /// The current block in the traversal stage.
    pub block: Option<BlockInfo>,
    /// The data source for the traversal stage.
    pub data_source: Provider,
    /// Indicates whether the block has been consumed by other stages.
    pub done: bool,
    /// The system config.
    pub system_config: SystemConfig,
    /// A reference to the rollup config.
    pub rollup_config: Arc<RollupConfig>,
}

#[async_trait]
impl<F: ChainProvider + Send> L1RetrievalProvider for IndexedTraversal<F> {
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

impl<F: ChainProvider> IndexedTraversal<F> {
    /// Creates a new [`IndexedTraversal`] instance.
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

    /// Provide the next block to the traversal stage.
    async fn provide_next_block(&mut self, block_info: BlockInfo) -> PipelineResult<()> {
        if !self.done {
            debug!(target: "traversal", "Not finished consuming block, ignoring provided block.");
            return Ok(());
        }
        let Some(block) = self.block else {
            return Err(PipelineError::MissingOrigin.temp());
        };
        if block.number + 1 != block_info.number {
            // Safe to ignore.
            // The next step will exhaust l1 and get the correct next l1 block.
            return Ok(());
        }
        if block.hash != block_info.parent_hash {
            return Err(
                ResetError::NextL1BlockHashMismatch(block.hash, block_info.parent_hash).reset()
            );
        }

        // Fetch receipts for the next l1 block and update the system config.
        let receipts =
            self.data_source.receipts_by_hash(block_info.hash).await.map_err(Into::into)?;

        let addr = self.rollup_config.l1_system_config_address;
        let active = self.rollup_config.is_ecotone_active(block_info.timestamp);
        match self.system_config.update_with_receipts(&receipts[..], addr, active) {
            Ok(true) => {
                let next = block_info.number as f64;
                kona_macros::set!(gauge, crate::Metrics::PIPELINE_LATEST_SYS_CONFIG_UPDATE, next);
                info!(target: "traversal", "System config updated at block {next}.");
            }
            Ok(false) => { /* Ignore, no update applied */ }
            Err(err) => {
                error!(target: "traversal", ?err, "Failed to update system config at block {}", block_info.number);
                kona_macros::set!(
                    gauge,
                    crate::Metrics::PIPELINE_SYS_CONFIG_UPDATE_ERROR,
                    block_info.number as f64
                );
                return Err(PipelineError::SystemConfigUpdate(err).crit());
            }
        }

        // Update the origin block.
        self.update_origin(block_info);

        Ok(())
    }
}

#[async_trait]
impl<F: ChainProvider + Send> OriginAdvancer for IndexedTraversal<F> {
    async fn advance_origin(&mut self) -> PipelineResult<()> {
        if !self.done {
            debug!(target: "traversal", "Not finished consuming block, ignoring advance call.");
            return Ok(());
        }
        return Err(PipelineError::Eof.temp());
    }
}

impl<F: ChainProvider> OriginProvider for IndexedTraversal<F> {
    fn origin(&self) -> Option<BlockInfo> {
        self.block
    }
}

#[async_trait]
impl<F: ChainProvider + Send> SignalReceiver for IndexedTraversal<F> {
    async fn signal(&mut self, signal: Signal) -> PipelineResult<()> {
        match signal {
            Signal::Reset(ResetSignal { l1_origin, system_config, .. }) |
            Signal::Activation(ActivationSignal { l1_origin, system_config, .. }) => {
                self.update_origin(l1_origin);
                self.system_config = system_config.expect("System config must be provided.");
            }
            Signal::ProvideBlock(block_info) => self.provide_next_block(block_info).await?,
            _ => {}
        }

        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::{errors::PipelineErrorKind, test_utils::TestChainProvider};
    use alloc::vec;
    use alloy_consensus::Receipt;
    use alloy_primitives::{B256, Bytes, Log, LogData, address, b256, hex};
    use kona_genesis::{CONFIG_UPDATE_EVENT_VERSION_0, CONFIG_UPDATE_TOPIC};

    const L1_SYS_CONFIG_ADDR: Address = address!("1337000000000000000000000000000000000000");

    fn new_update_batcher_log() -> Log {
        Log {
            address: L1_SYS_CONFIG_ADDR,
            data: LogData::new_unchecked(
                vec![
                    CONFIG_UPDATE_TOPIC,
                    CONFIG_UPDATE_EVENT_VERSION_0,
                    B256::ZERO, // Update type
                ],
                hex!("00000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000beef").into()
            )
        }
    }

    fn new_receipts() -> alloc::vec::Vec<Receipt> {
        let mut receipt =
            Receipt { status: alloy_consensus::Eip658Value::Eip658(true), ..Receipt::default() };
        let bad = Log::new(
            Address::from([2; 20]),
            vec![CONFIG_UPDATE_TOPIC, B256::default()],
            Bytes::default(),
        )
        .unwrap();
        receipt.logs = vec![new_update_batcher_log(), bad, new_update_batcher_log()];
        vec![receipt.clone(), Receipt::default(), receipt]
    }

    fn new_test_managed(
        blocks: alloc::vec::Vec<BlockInfo>,
        receipts: alloc::vec::Vec<Receipt>,
    ) -> IndexedTraversal<TestChainProvider> {
        let mut provider = TestChainProvider::default();
        let rollup_config = RollupConfig {
            l1_system_config_address: L1_SYS_CONFIG_ADDR,
            ..RollupConfig::default()
        };
        for (i, block) in blocks.iter().enumerate() {
            provider.insert_block(i as u64, *block);
        }
        for (i, receipt) in receipts.iter().enumerate() {
            let hash = blocks.get(i).map(|b| b.hash).unwrap_or_default();
            provider.insert_receipts(hash, vec![receipt.clone()]);
        }
        IndexedTraversal::new(provider, Arc::new(rollup_config))
    }

    fn new_populated_test_managed() -> IndexedTraversal<TestChainProvider> {
        let blocks = vec![BlockInfo::default(), BlockInfo::default()];
        let receipts = new_receipts();
        new_test_managed(blocks, receipts)
    }

    #[test]
    fn test_managed_traversal_batcher_address() {
        let mut traversal = new_populated_test_managed();
        traversal.system_config.batcher_address = L1_SYS_CONFIG_ADDR;
        assert_eq!(traversal.batcher_addr(), L1_SYS_CONFIG_ADDR);
    }

    #[tokio::test]
    async fn test_managed_traversal_activation_signal() {
        let blocks = vec![BlockInfo::default(), BlockInfo::default()];
        let receipts = new_receipts();
        let mut traversal = new_test_managed(blocks, receipts);
        let cfg = SystemConfig::default();
        traversal.done = true;
        assert!(
            traversal
                .signal(Signal::Activation(ActivationSignal {
                    system_config: Some(cfg),
                    ..Default::default()
                }))
                .await
                .is_ok()
        );
        assert_eq!(traversal.origin(), Some(BlockInfo::default()));
        assert_eq!(traversal.system_config, cfg);
        assert!(!traversal.done);
    }

    #[tokio::test]
    async fn test_managed_traversal_reset_signal() {
        let blocks = vec![BlockInfo::default(), BlockInfo::default()];
        let receipts = new_receipts();
        let mut traversal = new_test_managed(blocks, receipts);
        let cfg = SystemConfig::default();
        traversal.done = true;
        assert!(
            traversal
                .signal(Signal::Reset(ResetSignal {
                    system_config: Some(cfg),
                    ..Default::default()
                }))
                .await
                .is_ok()
        );
        assert_eq!(traversal.origin(), Some(BlockInfo::default()));
        assert_eq!(traversal.system_config, cfg);
        assert!(!traversal.done);
    }

    #[tokio::test]
    async fn test_managed_traversal_next_l1_block() {
        let blocks = vec![BlockInfo::default(), BlockInfo::default()];
        let receipts = new_receipts();
        let mut traversal = new_test_managed(blocks, receipts);
        assert_eq!(traversal.next_l1_block().await.unwrap(), Some(BlockInfo::default()));
        assert_eq!(traversal.next_l1_block().await.unwrap_err(), PipelineError::Eof.temp());
    }

    #[tokio::test]
    async fn test_managed_traversal_missing_receipts() {
        let blocks = vec![BlockInfo::default(), BlockInfo::default()];
        let mut traversal = new_test_managed(blocks, vec![]);
        assert_eq!(traversal.next_l1_block().await.unwrap(), Some(BlockInfo::default()));
        assert_eq!(traversal.next_l1_block().await.unwrap_err(), PipelineError::Eof.temp());
        // provide_next_block will fail due to missing receipts
        let next_block = BlockInfo { number: 1, ..BlockInfo::default() };
        let err = traversal.provide_next_block(next_block).await.unwrap_err();
        matches!(err, PipelineErrorKind::Temporary(PipelineError::Provider(_)));
    }

    #[tokio::test]
    async fn test_managed_traversal_reorgs() {
        let hash = b256!("3333333333333333333333333333333333333333333333333333333333333333");
        let block = BlockInfo { hash, number: 0, ..BlockInfo::default() };
        let blocks = vec![block];
        let receipts = new_receipts();
        let mut traversal = new_test_managed(blocks, receipts);
        traversal.block = Some(block);
        assert_eq!(traversal.next_l1_block().await.unwrap(), Some(block));
        // provide_next_block will fail due to hash mismatch (simulate reorg)
        let next_block = BlockInfo { number: 1, ..BlockInfo::default() };
        let err = traversal.provide_next_block(next_block).await.unwrap_err();
        assert_eq!(err, ResetError::NextL1BlockHashMismatch(hash, next_block.parent_hash).reset());
    }

    #[tokio::test]
    async fn test_managed_traversal_missing_blocks() {
        let mut traversal = new_test_managed(vec![], vec![]);
        assert_eq!(traversal.next_l1_block().await.unwrap(), Some(BlockInfo::default()));
        assert_eq!(traversal.next_l1_block().await.unwrap_err(), PipelineError::Eof.temp());
        // provide_next_block will fail due to missing origin
        let next_block = BlockInfo { number: 1, ..BlockInfo::default() };
        let err = traversal.provide_next_block(next_block).await.unwrap_err();
        matches!(err, PipelineErrorKind::Temporary(PipelineError::MissingOrigin));
    }

    #[tokio::test]
    async fn test_managed_traversal_system_config_update_fails() {
        let first = b256!("3333333333333333333333333333333333333333333333333333333333333333");
        let second = b256!("4444444444444444444444444444444444444444444444444444444444444444");
        let block1 = BlockInfo { hash: first, ..BlockInfo::default() };
        let block2 = BlockInfo { number: 1, hash: second, ..BlockInfo::default() };
        let blocks = vec![block1, block2];
        let receipts = new_receipts();
        let mut traversal = new_test_managed(blocks, receipts);
        traversal.block = Some(block1);
        assert_eq!(traversal.next_l1_block().await.unwrap(), Some(block1));
        // provide_next_block will fail due to system config update error
        let err = traversal.provide_next_block(block2).await.unwrap_err();
        matches!(err, PipelineErrorKind::Critical(PipelineError::SystemConfigUpdate(_)));
    }

    #[tokio::test]
    async fn test_managed_traversal_system_config_updated() {
        let blocks = vec![BlockInfo::default(), BlockInfo::default()];
        let receipts = new_receipts();
        let mut traversal = new_test_managed(blocks, receipts);
        assert_eq!(traversal.next_l1_block().await.unwrap(), Some(BlockInfo::default()));
        assert_eq!(traversal.next_l1_block().await.unwrap_err(), PipelineError::Eof.temp());
        // provide_next_block should update system config
        let next_block = BlockInfo { number: 1, ..BlockInfo::default() };
        assert!(traversal.provide_next_block(next_block).await.is_ok());
        let expected = address!("000000000000000000000000000000000000bEEF");
        assert_eq!(traversal.system_config.batcher_address, expected);
    }
}
