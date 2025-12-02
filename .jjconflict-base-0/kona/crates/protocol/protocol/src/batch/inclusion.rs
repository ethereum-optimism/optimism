//! Module containing the [`BatchWithInclusionBlock`] struct.

use crate::{Batch, BatchValidationProvider, BatchValidity, BlockInfo, L2BlockInfo};
use kona_genesis::RollupConfig;

/// A batch with its inclusion block.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct BatchWithInclusionBlock {
    /// The inclusion block
    pub inclusion_block: BlockInfo,
    /// The batch
    pub batch: Batch,
}

impl BatchWithInclusionBlock {
    /// Creates a new batch with inclusion block.
    pub const fn new(inclusion_block: BlockInfo, batch: Batch) -> Self {
        Self { inclusion_block, batch }
    }

    /// Validates the batch can be applied on top of the specified L2 safe head.
    /// The first entry of the l1_blocks should match the origin of the l2_safe_head.
    /// One or more consecutive l1_blocks should be provided.
    /// In case of only a single L1 block, the decision whether a batch is valid may have to stay
    /// undecided.
    pub async fn check_batch<BF: BatchValidationProvider>(
        &self,
        cfg: &RollupConfig,
        l1_blocks: &[BlockInfo],
        l2_safe_head: L2BlockInfo,
        fetcher: &mut BF,
    ) -> BatchValidity {
        match &self.batch {
            Batch::Single(single_batch) => {
                single_batch.check_batch(cfg, l1_blocks, l2_safe_head, &self.inclusion_block)
            }
            Batch::Span(span_batch) => {
                span_batch
                    .check_batch(cfg, l1_blocks, l2_safe_head, &self.inclusion_block, fetcher)
                    .await
            }
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::test_utils::TestBatchValidator;
    use alloc::vec;

    #[tokio::test]
    async fn test_single_batch_with_inclusion_block() {
        let batch =
            BatchWithInclusionBlock::new(BlockInfo::default(), Batch::Single(Default::default()));
        let l1_blocks = vec![BlockInfo::default()];
        let l2_safe_head = L2BlockInfo::default();
        let cfg = RollupConfig::default();
        let mut validator = TestBatchValidator::default();
        let result = batch.check_batch(&cfg, &l1_blocks, l2_safe_head, &mut validator).await;
        assert_eq!(result, BatchValidity::Accept);
    }

    #[tokio::test]
    async fn test_span_batch_with_inclusion_block() {
        let batch =
            BatchWithInclusionBlock::new(BlockInfo::default(), Batch::Span(Default::default()));
        let l1_blocks = vec![BlockInfo::default()];
        let l2_safe_head = L2BlockInfo::default();
        let cfg = RollupConfig::default();
        let mut validator = TestBatchValidator::default();
        let result = batch.check_batch(&cfg, &l1_blocks, l2_safe_head, &mut validator).await;
        assert_eq!(result, BatchValidity::Undecided);
    }
}
