//! Per-transaction sequencer rules shared by the batch validity checks.

use crate::{BatchDropReason, BatchValidity};
use alloy_primitives::Bytes;
use kona_genesis::RollupConfig;
use op_alloy_consensus::OpTxType;
use tracing::warn;

/// Applies the per-transaction sequencer rules to the batched transactions of a single L2 block.
///
/// [`SingleBatch`](crate::SingleBatch) and [`SpanBatch`](crate::SpanBatch) share this so the rules
/// cannot drift apart: both formats carry sequencer-authored transaction lists subject to the same
/// restrictions, and a rule that only one path enforces is a consensus split between batch formats.
///
/// The activation-gated types are resolved from `cfg` and `block_timestamp` here rather than taken
/// as pre-computed fork flags, so a caller cannot pass a flag read at the wrong timestamp. For a
/// span batch that timestamp is the one of the block the transactions belong to, not the span's
/// first block.
pub(crate) fn check_sequencer_txs(
    cfg: &RollupConfig,
    txs: &[Bytes],
    block_timestamp: u64,
) -> BatchValidity {
    let is_isthmus = cfg.is_isthmus_active(block_timestamp);
    let is_sdm = cfg.is_sdm_active(block_timestamp);

    for (i, tx) in txs.iter().enumerate() {
        let Some(first_byte) = tx.as_ref().first().copied() else {
            warn!(
                target: "batch_txs",
                "transaction data must not be empty, but found empty tx, tx_index: {i}"
            );
            return BatchValidity::Drop(BatchDropReason::EmptyTransaction);
        };
        // A leading byte that doesn't decode to a typed transaction (e.g. a legacy RLP
        // list header) isn't one of the restricted types, so it falls through to `Accept`.
        match OpTxType::try_from(first_byte) {
            Ok(OpTxType::Deposit) => {
                warn!(
                    target: "batch_txs",
                    "sequencers may not embed any deposits into batch data, but found tx that has one, tx_index: {i}"
                );
                return BatchValidity::Drop(BatchDropReason::DepositTransaction);
            }
            Ok(OpTxType::Eip7702) if !is_isthmus => {
                warn!(target: "batch_txs", "EIP-7702 transactions are not supported pre-isthmus. tx_index: {i}");
                return BatchValidity::Drop(BatchDropReason::Eip7702PreIsthmus);
            }
            Ok(OpTxType::PostExec) if !is_sdm => {
                warn!(target: "batch_txs", "PostExec transactions are not supported pre-Lagoon. tx_index: {i}");
                return BatchValidity::Drop(BatchDropReason::PostExecPreLagoon);
            }
            _ => {}
        }
    }

    BatchValidity::Accept
}

#[cfg(test)]
mod tests;
