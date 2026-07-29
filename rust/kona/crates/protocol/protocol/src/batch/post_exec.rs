//! Structural validation of `PostExec` (`0x7D`) transactions in sequencer-provided batch data.
//!
//! Derivation and the execution layer must reach the same verdict on block validity. A block that
//! derivation accepts but the EL rejects does not stall a follower post-Holocene — it falls back to
//! deposits-only attributes — but that is worse than stalling for the height: the safe chain
//! commits a deposits-only block, discarding every user transaction the sequencer included, and
//! because the batch was accepted rather than dropped the batch queue never tries a subsequent
//! valid batch for the height. Dropping is strictly better, so every rejection here is a
//! [`Drop`](BatchValidity::Drop).
//!
//! # Enforced
//!
//! Everything the EL decides from the batch transactions of a single block, across its two
//! validation sites — op-alloy's `parse_post_exec_payload_from_transactions` (activation,
//! at-most-one, payload well-formedness, block-number anchor, last-in-block) and
//! `PostExecState::new` in `alloy-op-evm`'s `src/block/mod.rs`, whose two payload-only
//! preconditions are that no entry has a zero refund and no transaction index repeats.
//!
//! Rule *precedence* mirrors op-alloy arm for arm, so a payload violating several rules at once
//! yields the same reason there and here.
//!
//! # Not enforced
//!
//! An entry index that is out of range or that targets a deposit or the `0x7D` itself, and
//! `refund <= evm_gas_used`. Entry indices are **block-global** — they count the deposits the
//! derivation pipeline prepends — while a batch holds only the block's non-deposit transactions, so
//! those rules need the block's full transaction list; the refund ceiling needs execution. Only the
//! executor can decide them. The position and count rules need no such adjustment: deposits are
//! always prepended, so the last batch transaction is the last transaction of the block.

use crate::{BatchDropReason, BatchValidity};
use alloc::collections::BTreeSet;
use alloy_primitives::Bytes;
use op_alloy_consensus::{POST_EXEC_TX_TYPE_ID, PostExecPayload};
use tracing::warn;

/// Applies the `PostExec` rules decidable from one block's batch transactions. See the module docs
/// for which rules those are and why the rest belong to the executor.
///
/// `block_number` is the number of the L2 block `transactions` builds — the number a payload must
/// be anchored to. `sdm_active` is whether SDM is active at that block's timestamp.
pub(crate) fn check_post_exec_txs(
    transactions: &[Bytes],
    block_number: u64,
    sdm_active: bool,
) -> BatchValidity {
    let mut post_exec_index: Option<usize> = None;

    for (index, tx) in transactions.iter().enumerate() {
        // `0x7D` is below the `0xc0` legacy-RLP list-header range, so a leading byte match cannot
        // collide with an untyped transaction.
        if tx.first() != Some(&POST_EXEC_TX_TYPE_ID) {
            continue;
        }

        // Activation first, matching op-alloy: a pre-activation `0x7D` is rejected before its
        // payload is looked at, so a bare type byte reports the activation failure rather than a
        // decode failure.
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

        // The canonical encoding is `0x7D || payload`, so the payload is everything after the type
        // byte. `from_rlp_bytes` applies the same checks the EL does — the supported `version` and
        // no trailing bytes — and is not stricter: the derivation-side executor frames the
        // transaction with `decode_2718_exact`, which rejects leftover bytes too. Version
        // and trailing-byte rejection are covered by op-alloy's own `from_rlp_bytes` tests.
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

    // Deferred out of the loop on purpose: op-alloy decides `MultiplePostExecTxs` inside its loop
    // and `PostExecTxNotLast` after it, so folding this check into the loop would report
    // `NotLast` where op-alloy reports `Multiple` for a batch that is both.
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
