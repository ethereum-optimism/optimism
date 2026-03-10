//! Span batch construction from L2 blocks.

use alloy_primitives::FixedBytes;
use kona_genesis::RollupConfig;
use kona_protocol::{SpanBatch, SpanBatchElement};

use crate::l2::L2Block;

use super::singular::{L1Origin, block_to_singular_batch};

/// Build a [`SpanBatch`] from a slice of L2 blocks.
///
/// The resulting span batch contains one element per block, with epoch numbers
/// and timestamps derived from the singular batch conversion. The `parent_check`
/// and `l1_origin_check` fields are populated from the first block's parent hash
/// and the L1 origin hash respectively.
pub fn build_span_batch(
    blocks: &[&L2Block],
    l1_origin: L1Origin,
    rollup_config: &RollupConfig,
) -> SpanBatch {
    assert!(!blocks.is_empty(), "cannot build span batch from zero blocks");

    let elements: Vec<SpanBatchElement> = blocks
        .iter()
        .map(|block| {
            let singular = block_to_singular_batch(block, rollup_config, l1_origin);
            SpanBatchElement {
                epoch_num: l1_origin.number,
                timestamp: singular.timestamp,
                transactions: singular.transactions,
            }
        })
        .collect();

    let parent_hash = blocks[0].header.inner().parent_hash;

    SpanBatch {
        parent_check: FixedBytes::from_slice(&parent_hash[..20]),
        l1_origin_check: FixedBytes::from_slice(&l1_origin.hash[..20]),
        genesis_timestamp: rollup_config.genesis.l2_time,
        chain_id: rollup_config.l2_chain_id.id(),
        batches: elements,
        ..Default::default()
    }
}
