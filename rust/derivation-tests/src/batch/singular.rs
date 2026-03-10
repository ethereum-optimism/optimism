//! Singular batch construction from L2 blocks.

use alloy_eips::Encodable2718;
use alloy_primitives::Bytes;
use kona_genesis::RollupConfig;
use kona_protocol::SingleBatch;

use crate::l2::L2Block;

/// Extract a `SingleBatch` from an L2 block.
///
/// Includes all non-deposit transactions. Mirrors Go's `BlockToSingularBatch`.
pub fn block_to_singular_batch(block: &L2Block, _rollup_config: &RollupConfig) -> SingleBatch {
    let mut transactions = Vec::new();

    for tx in &block.transactions {
        // Skip deposit transactions — they are not included in batches
        if matches!(tx, op_alloy_consensus::OpTxEnvelope::Deposit(_)) {
            continue;
        }
        let mut buf = Vec::new();
        tx.encode_2718(&mut buf);
        transactions.push(Bytes::from(buf));
    }

    SingleBatch {
        parent_hash: block.header.inner().parent_hash,
        epoch_num: 0,                   // Set by caller based on L1 origin
        epoch_hash: Default::default(), // Set by caller
        timestamp: block.header.inner().timestamp,
        transactions,
    }
}
