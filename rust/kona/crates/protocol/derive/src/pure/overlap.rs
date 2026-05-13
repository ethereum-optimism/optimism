//! Full byte-wise span-batch overlap content check.
//!
//! **Spec-gap closure.** Post-Holocene, both op-node
//! (`op-node/rollup/derive/batch_stage.go:139`) and kona's async pipeline
//! (`crates/protocol/protocol/src/batch/span.rs:591`) only run the
//! parent-hash prefix check on span batches. The full byte-wise overlap
//! content verification (the bottom half of
//! `kona_protocol::SpanBatch::check_batch`) is only invoked from the
//! pre-Holocene `BatchQueue` path. This module brings the full check into
//! the pure deriver.
//!
//! The check verifies, for every overlap block:
//!
//! 1. Transaction count matches the span batch's claim (excluding leading deposit transactions,
//!    which the span batch never includes).
//! 2. Each transaction's RLP bytes match byte-for-byte. Caller-supplied `SpanBatchOverlapBlock.txs`
//!    is raw RLP — no encode/decode round-trip.
//! 3. The block's L1 origin number, decoded from the leading L1 info deposit transaction, matches
//!    the span batch's per-block epoch.
//!
//! Failures produce typed `BatchDropReason` values, consumed by the deriver
//! which turns them into [`crate::TraceEntry::BatchVerdict`] entries.

use crate::pure::{SpanBatchOverlap, SpanBatchOverlapBlock};
use alloy_eips::eip2718::Decodable2718;
use kona_protocol::{BatchDropReason, L1BlockInfoTx, SpanBatch};
use op_alloy_consensus::OpTxEnvelope;

/// Result of running the overlap content check.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub(crate) enum OverlapResult {
    /// The span batch's prefix matches the overlap content.
    Accept,
    /// The span batch's claim disagreed with the overlap content. Caller
    /// emits a `BatchVerdict::Drop(reason)` and flushes the channel.
    Drop(BatchDropReason),
}

/// Verifies that `overlap.blocks` is consistent with the span batch's
/// per-block claims for the same range. The caller has already confirmed
/// (in the prefix check) that `overlap.parent.hash == span.parent_check[..20]`
/// and that the block range matches what the deriver requested.
pub(crate) fn verify(
    span: &SpanBatch,
    overlap: &SpanBatchOverlap,
    parent_num: u64,
) -> OverlapResult {
    // `overlap.blocks[i]` corresponds to `span.batches[i]` for the overlap
    // range. The deriver requested exactly the overlap range, so length
    // equality is a separate contract check at the API boundary; here we
    // iterate the shorter of the two and trust the contract.
    for (i, block) in overlap.blocks.iter().enumerate() {
        let Some(span_block) = span.batches.get(i) else {
            // Span batch is shorter than overlap claimed. The prefix check
            // guarantees `final_timestamp >= safe_head.timestamp`, so this
            // can only happen on a misshapen span batch. Drop with the
            // closest matching reason.
            return OverlapResult::Drop(BatchDropReason::OverlappedTxCountMismatch);
        };

        // 1. Decode the L1 info deposit (first tx of every L2 block per the L1InfoTx scheme) and
        //    check the L1 origin number matches the span batch's epoch number for this block.
        match decode_l1_origin_number(block) {
            Some(l1_origin_num) => {
                if l1_origin_num != span_block.epoch_num {
                    return OverlapResult::Drop(BatchDropReason::OverlappedL1OriginMismatch);
                }
            }
            None => {
                // Same drop reason — a missing/malformed L1 info deposit is
                // semantically "we couldn't extract the L1 origin to
                // compare", which is just a special case of mismatch.
                return OverlapResult::Drop(BatchDropReason::L2BlockInfoExtractionFailed);
            }
        }

        // 2. Count deposits in the overlap block (always at the front), then compare user-tx
        //    counts.
        let deposit_count = count_leading_deposits(&block.txs);
        let user_tx_count = block.txs.len().saturating_sub(deposit_count);
        if user_tx_count != span_block.transactions.len() {
            return OverlapResult::Drop(BatchDropReason::OverlappedTxCountMismatch);
        }

        // 3. Byte-equal compare of each user-tx RLP against the span batch's transaction at the
        //    same index. No encode/decode round-trip — both sides are raw bytes.
        for (j, span_tx) in span_block.transactions.iter().enumerate() {
            let overlap_tx = &block.txs[deposit_count + j];
            if overlap_tx.as_ref() != span_tx.0.as_ref() {
                return OverlapResult::Drop(BatchDropReason::OverlappedTxMismatch);
            }
        }

        // Sanity: parent_num is the first overlap block's `number - 1`.
        // Iteration order means we don't need to check it explicitly per
        // block, but a debug_assert keeps the invariant visible during
        // development.
        debug_assert_eq!(block.number, parent_num + 1 + i as u64);
    }

    OverlapResult::Accept
}

/// Decodes the L1 origin number from an L2 block's first transaction (the
/// L1 info deposit). Returns `None` if the first tx is missing, isn't a
/// deposit, or the calldata fails to decode as an `L1BlockInfoTx`.
fn decode_l1_origin_number(block: &SpanBatchOverlapBlock) -> Option<u64> {
    let first_tx = block.txs.first()?;
    // Decode 2718-encoded envelope.
    let envelope = OpTxEnvelope::decode_2718(&mut first_tx.as_ref()).ok()?;
    let deposit = envelope.as_deposit()?;
    let l1_info = L1BlockInfoTx::decode_calldata(deposit.input.as_ref()).ok()?;
    Some(l1_info.id().number)
}

/// Counts leading deposit transactions in `txs`. Span batches never carry
/// deposits, so all deposits in the executed block are at the front (the
/// L1 info tx plus any user deposits).
fn count_leading_deposits(txs: &[alloy_primitives::Bytes]) -> usize {
    let mut count = 0;
    for tx in txs {
        // EIP-2718 type byte for a deposit transaction.
        if tx.as_ref().first() == Some(&(op_alloy_consensus::OpTxType::Deposit as u8)) {
            count += 1;
        } else {
            break;
        }
    }
    count
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::pure::{SpanBatchOverlap, SpanBatchOverlapBlock};
    use alloc::{vec, vec::Vec};
    use alloy_consensus::SignableTransaction;
    use alloy_eips::eip2718::Encodable2718;
    use alloy_primitives::{Address, B256, Bytes, Signature, U256};
    use kona_protocol::{L1BlockInfoBedrock, L2BlockInfo, SpanBatchElement};
    use op_alloy_consensus::{OpTxEnvelope, TxDeposit};

    fn deposit_tx_with_origin(origin: u64) -> Bytes {
        let info = L1BlockInfoBedrock::new(
            origin,
            0,
            0,
            B256::ZERO,
            0,
            Address::ZERO,
            U256::ZERO,
            U256::ZERO,
        );
        let calldata = info.encode_calldata();
        let tx = TxDeposit { input: calldata, ..Default::default() };
        let envelope = OpTxEnvelope::Deposit(alloy_primitives::Sealed::new(tx));
        Bytes::from(envelope.encoded_2718())
    }

    fn user_tx(byte: u8) -> Bytes {
        let tx = alloy_consensus::TxEip1559 {
            chain_id: 10,
            nonce: byte as u64,
            gas_limit: 21000,
            max_fee_per_gas: 1,
            max_priority_fee_per_gas: 1,
            to: alloy_primitives::TxKind::Call(Address::with_last_byte(byte)),
            value: U256::ZERO,
            input: Default::default(),
            access_list: Default::default(),
        };
        let signed = tx.into_signed(Signature::test_signature());
        let envelope: alloy_consensus::TxEnvelope = signed.into();
        Bytes::from(envelope.encoded_2718())
    }

    fn make_span(epoch: u64, user_txs: Vec<Bytes>) -> SpanBatch {
        SpanBatch {
            batches: vec![SpanBatchElement {
                epoch_num: epoch,
                timestamp: 0,
                transactions: user_txs,
            }],
            ..Default::default()
        }
    }

    fn make_overlap(
        parent_num: u64,
        deposit_origin: u64,
        user_txs: Vec<Bytes>,
    ) -> SpanBatchOverlap {
        let mut txs = vec![deposit_tx_with_origin(deposit_origin)];
        txs.extend(user_txs);
        SpanBatchOverlap {
            parent: L2BlockInfo {
                block_info: kona_protocol::BlockInfo { number: parent_num, ..Default::default() },
                ..Default::default()
            },
            blocks: vec![SpanBatchOverlapBlock { number: parent_num + 1, txs }],
        }
    }

    #[test]
    fn accept_when_everything_matches() {
        let user = user_tx(1);
        let span = make_span(100, vec![user.clone()]);
        let overlap = make_overlap(0, 100, vec![user]);
        assert_eq!(verify(&span, &overlap, 0), OverlapResult::Accept);
    }

    #[test]
    fn drop_on_tx_count_mismatch() {
        let user1 = user_tx(1);
        let user2 = user_tx(2);
        // Span batch claims one user tx, overlap has two.
        let span = make_span(100, vec![user1.clone()]);
        let overlap = make_overlap(0, 100, vec![user1, user2]);
        assert_eq!(
            verify(&span, &overlap, 0),
            OverlapResult::Drop(BatchDropReason::OverlappedTxCountMismatch),
        );
    }

    #[test]
    fn drop_on_tx_mismatch() {
        let user1 = user_tx(1);
        let user2 = user_tx(2);
        let span = make_span(100, vec![user1]);
        let overlap = make_overlap(0, 100, vec![user2]);
        assert_eq!(
            verify(&span, &overlap, 0),
            OverlapResult::Drop(BatchDropReason::OverlappedTxMismatch),
        );
    }

    #[test]
    fn drop_on_l1_origin_mismatch() {
        let user = user_tx(1);
        // Span batch claims epoch 100, overlap's L1 info tx encodes 999.
        let span = make_span(100, vec![user.clone()]);
        let overlap = make_overlap(0, 999, vec![user]);
        assert_eq!(
            verify(&span, &overlap, 0),
            OverlapResult::Drop(BatchDropReason::OverlappedL1OriginMismatch),
        );
    }

    #[test]
    fn drop_on_missing_l1_info_tx() {
        // Overlap block has zero txs at all — no L1 info deposit.
        let span = make_span(100, vec![]);
        let overlap = SpanBatchOverlap {
            parent: L2BlockInfo::default(),
            blocks: vec![SpanBatchOverlapBlock { number: 1, txs: vec![] }],
        };
        assert_eq!(
            verify(&span, &overlap, 0),
            OverlapResult::Drop(BatchDropReason::L2BlockInfoExtractionFailed),
        );
    }
}
