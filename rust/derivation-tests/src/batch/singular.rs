//! Singular batch construction from L2 blocks.

use alloy_eips::Encodable2718;
use alloy_primitives::{B256, Bytes};
use kona_genesis::RollupConfig;
use kona_protocol::SingleBatch;

use crate::l2::L2Block;

/// L1 origin reference for a batch (block number and hash).
#[derive(Debug, Clone, Copy)]
pub struct L1Origin {
    /// L1 block number.
    pub number: u64,
    /// L1 block hash.
    pub hash: B256,
}

/// Extract a `SingleBatch` from an L2 block.
///
/// Includes all non-deposit transactions. Mirrors Go's `BlockToSingularBatch`.
pub fn block_to_singular_batch(
    block: &L2Block,
    _rollup_config: &RollupConfig,
    l1_origin: L1Origin,
) -> SingleBatch {
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
        epoch_num: l1_origin.number,
        epoch_hash: l1_origin.hash,
        timestamp: block.header.inner().timestamp,
        transactions,
    }
}
