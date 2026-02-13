//! Module containing the [`RawSpanBatch`] struct.

use alloc::{vec, vec::Vec};
use alloy_primitives::bytes;

use crate::{
    BatchType, SpanBatch, SpanBatchElement, SpanBatchError, SpanBatchPayload, SpanBatchPrefix,
    SpanDecodingError,
};

/// Raw Span Batch
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct RawSpanBatch {
    /// The span batch prefix
    pub prefix: SpanBatchPrefix,
    /// The span batch payload
    pub payload: SpanBatchPayload,
}

impl RawSpanBatch {
    /// Returns the batch type
    pub const fn get_batch_type(&self) -> BatchType {
        BatchType::Span
    }

    /// Encodes the [`RawSpanBatch`] into a writer.
    pub fn encode(&self, w: &mut dyn bytes::BufMut) -> Result<(), SpanBatchError> {
        self.prefix.encode_prefix(w);
        self.payload.encode_payload(w)
    }

    /// Decodes the [`RawSpanBatch`] from a reader.]
    pub fn decode(r: &mut &[u8]) -> Result<Self, SpanBatchError> {
        let prefix = SpanBatchPrefix::decode_prefix(r)?;
        let payload = SpanBatchPayload::decode_payload(r)?;
        Ok(Self { prefix, payload })
    }

    /// Converts a [`RawSpanBatch`] into a [`SpanBatch`], which has a list of [`SpanBatchElement`]s.
    /// This function does not populate the [`SpanBatch`] with chain configuration data, which
    /// is required for making payload attributes.
    pub fn derive(
        &mut self,
        block_time: u64,
        genesis_time: u64,
        chain_id: u64,
    ) -> Result<SpanBatch, SpanBatchError> {
        if self.payload.block_count == 0 {
            return Err(SpanBatchError::EmptySpanBatch);
        }

        let mut block_origin_nums = vec![0u64; self.payload.block_count as usize];
        let mut l1_origin_number = self.prefix.l1_origin_num;
        for i in (0..self.payload.block_count).rev() {
            block_origin_nums[i as usize] = l1_origin_number;
            if self
                .payload
                .origin_bits
                .get_bit(i as usize)
                .ok_or(SpanBatchError::Decoding(SpanDecodingError::L1OriginCheck))? ==
                1 &&
                i > 0
            {
                l1_origin_number -= 1;
            }
        }

        // Get all transactions in the batch.
        let enveloped_txs = self.payload.txs.full_txs(chain_id)?;

        let mut tx_idx = 0;
        let batches = (0..self.payload.block_count).fold(Vec::new(), |mut acc, i| {
            let transactions =
                (0..self.payload.block_tx_counts[i as usize]).fold(Vec::new(), |mut acc, _| {
                    acc.push(enveloped_txs[tx_idx].clone());
                    tx_idx += 1;
                    acc
                });
            acc.push(SpanBatchElement {
                epoch_num: block_origin_nums[i as usize],
                timestamp: genesis_time + self.prefix.rel_timestamp + block_time * i,
                transactions: transactions.into_iter().map(|v| v.into()).collect(),
            });
            acc
        });

        Ok(SpanBatch {
            parent_check: self.prefix.parent_check,
            l1_origin_check: self.prefix.l1_origin_check,
            batches,
            ..Default::default()
        })
    }
}

#[cfg(test)]
mod test {
    use super::*;
    use alloy_primitives::FixedBytes;

    #[test]
    fn test_try_from_span_batch_empty_batches_errors() {
        let span_batch = SpanBatch::default();
        let raw_span_batch = span_batch.to_raw_span_batch().unwrap_err();
        assert_eq!(raw_span_batch, SpanBatchError::EmptySpanBatch);
    }

    #[test]
    fn test_try_from_span_batch_succeeds() {
        let parent_check = FixedBytes::from([2u8; 20]);
        let l1_origin_check = FixedBytes::from([3u8; 20]);
        let first = SpanBatchElement { epoch_num: 100, timestamp: 400, transactions: Vec::new() };
        let last = SpanBatchElement { epoch_num: 200, timestamp: 500, transactions: Vec::new() };
        let span_batch = SpanBatch {
            batches: vec![first, last],
            genesis_timestamp: 300,
            parent_check,
            l1_origin_check,
            ..Default::default()
        };
        let expected_prefix = SpanBatchPrefix {
            rel_timestamp: 100,
            l1_origin_num: 200,
            parent_check,
            l1_origin_check,
        };
        let expected_payload = SpanBatchPayload { block_count: 2, ..Default::default() };
        let raw_span_batch = span_batch.to_raw_span_batch().unwrap();
        assert_eq!(raw_span_batch.prefix, expected_prefix);
        assert_eq!(raw_span_batch.payload, expected_payload);
    }

    #[test]
    fn test_decode_encode_raw_span_batch() {
        // Load in the raw span batch from the `op-node` derivation pipeline implementation.
        let raw_span_batch_hex = include_bytes!("./testdata/raw_batch.hex");
        let raw_span_batch = RawSpanBatch::decode(&mut raw_span_batch_hex.as_slice()).unwrap();

        let mut encoding_buf = Vec::new();
        raw_span_batch.encode(&mut encoding_buf).unwrap();
        assert_eq!(encoding_buf, raw_span_batch_hex);
    }
}
