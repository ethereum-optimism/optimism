//! `PostExec` (`0x7D`) validation for sequencer batches.
//!
//! Entry bounds and targets require a full block; refund limits require execution.

use crate::{BatchDropReason, BatchValidity};
use alloc::collections::BTreeSet;
use alloy_primitives::Bytes;
use op_alloy_consensus::{POST_EXEC_TX_TYPE_ID, PostExecPayload};
use tracing::warn;

/// Validates one block's `PostExec` transactions and payload anchor.
pub(crate) fn check_post_exec_txs(
    transactions: &[Bytes],
    block_number: u64,
    sdm_active: bool,
) -> BatchValidity {
    let mut post_exec_index: Option<usize> = None;

    for (index, tx) in transactions.iter().enumerate() {
        // Legacy transactions cannot start with `0x7D`.
        if tx.first() != Some(&POST_EXEC_TX_TYPE_ID) {
            continue;
        }

        // Match EL validation order.
        if !sdm_active {
            warn!(
                target: "batch_post_exec",
                "PostExec transactions are not supported pre-Lagoon. tx_index: {}",
                index
            );
            return BatchValidity::Drop(BatchDropReason::PostExecPreLagoon);
        }

        if let Some(first_index) = post_exec_index {
            warn!(
                target: "batch_post_exec",
                "a block may contain at most one PostExec transaction, first_index: {}, tx_index: {}",
                first_index,
                index
            );
            return BatchValidity::Drop(BatchDropReason::MultiplePostExecTxs);
        }
        post_exec_index = Some(index);

        // Decode bytes after the type byte.
        let payload = match PostExecPayload::from_rlp_bytes(&tx[1..]) {
            Ok(payload) => payload,
            Err(err) => {
                warn!(
                    target: "batch_post_exec",
                    "PostExec transaction payload is invalid, tx_index: {}, err: {}",
                    index,
                    err
                );
                return BatchValidity::Drop(BatchDropReason::PostExecPayloadInvalid);
            }
        };

        if payload.block_number != block_number {
            warn!(
                target: "batch_post_exec",
                "PostExec payload is anchored to the wrong block, tx_index: {}, payload_block_number: {}, block_number: {}",
                index,
                payload.block_number,
                block_number
            );
            return BatchValidity::Drop(BatchDropReason::PostExecPayloadBlockNumberMismatch);
        }

        let mut seen = BTreeSet::new();
        for entry in &payload.gas_refund_entries {
            if entry.gas_refund == 0 {
                warn!(
                    target: "batch_post_exec",
                    "PostExec payload has a zero-refund entry, entry_index: {}",
                    entry.index
                );
                return BatchValidity::Drop(BatchDropReason::PostExecPayloadZeroRefund);
            }
            if !seen.insert(entry.index) {
                warn!(
                    target: "batch_post_exec",
                    "PostExec payload has a duplicate entry, entry_index: {}",
                    entry.index
                );
                return BatchValidity::Drop(BatchDropReason::PostExecPayloadDuplicateEntry);
            }
        }
    }

    // Check position after count to match the EL.
    if let Some(index) = post_exec_index &&
        index != transactions.len() - 1
    {
        warn!(
            target: "batch_post_exec",
            "PostExec transaction must be the last transaction in the block, tx_index: {}, last_index: {}",
            index,
            transactions.len() - 1
        );
        return BatchValidity::Drop(BatchDropReason::PostExecTxNotLast);
    }

    BatchValidity::Accept
}
