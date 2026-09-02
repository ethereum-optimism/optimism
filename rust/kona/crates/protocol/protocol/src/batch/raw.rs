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
    /// The wire version of the span batch
    pub version: BatchType,
    /// The span batch prefix
    pub prefix: SpanBatchPrefix,
    /// The span batch payload
    pub payload: SpanBatchPayload,
}

impl RawSpanBatch {
    /// Returns the batch type
    pub const fn get_batch_type(&self) -> BatchType {
        self.version
    }

    /// Encodes the [`RawSpanBatch`] into a writer.
    pub fn encode(&self, w: &mut dyn bytes::BufMut) -> Result<(), SpanBatchError> {
        self.prefix.encode_prefix(w);
        self.payload.encode_payload(w, self.version)
    }

    /// Decodes a [`RawSpanBatch`] of the given `version` from a reader.
    pub fn decode(r: &mut &[u8], version: BatchType) -> Result<Self, SpanBatchError> {
        if !version.is_span() {
            return Err(SpanBatchError::NotASpanVersion);
        }
        let prefix = SpanBatchPrefix::decode_prefix(r)?;
        let payload = SpanBatchPayload::decode_payload(r, version)?;
        Ok(Self { version, prefix, payload })
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
        let mut timestamp = genesis_time + self.prefix.rel_timestamp;
        let mut batches = Vec::with_capacity(self.payload.block_count as usize);
        for i in 0..self.payload.block_count {
            // Bit 0 relates the first element to the span's parent block, so it never shifts the
            // span's own starting timestamp.
            if i > 0 && !self.shares_predecessor_timestamp(i as usize)? {
                timestamp += block_time;
            }
            let transactions =
                (0..self.payload.block_tx_counts[i as usize]).fold(Vec::new(), |mut acc, _| {
                    acc.push(enveloped_txs[tx_idx].clone());
                    tx_idx += 1;
                    acc
                });
            batches.push(SpanBatchElement {
                epoch_num: block_origin_nums[i as usize],
                timestamp,
                transactions: transactions.into_iter().map(|v| v.into()).collect(),
            });
        }

        Ok(SpanBatch {
            version: self.version,
            parent_check: self.prefix.parent_check,
            l1_origin_check: self.prefix.l1_origin_check,
            same_ts_bits: self.payload.same_ts_bits.clone(),
            batches,
            ..Default::default()
        })
    }

    /// Returns true if element `index` shares the timestamp of its predecessor. Always false for
    /// versions without the `same_ts_bits` bitlist.
    fn shares_predecessor_timestamp(&self, index: usize) -> Result<bool, SpanBatchError> {
        let Some(same_ts_bits) = self.payload.same_ts_bits.as_ref() else {
            return Ok(false);
        };
        let bit = same_ts_bits
            .get_bit(index)
            .ok_or(SpanBatchError::Decoding(SpanDecodingError::SameTimestampBits))?;
        Ok(bit == 1)
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
        let raw_span_batch =
            RawSpanBatch::decode(&mut raw_span_batch_hex.as_slice(), BatchType::Span).unwrap();

        let mut encoding_buf = Vec::new();
        raw_span_batch.encode(&mut encoding_buf).unwrap();
        assert_eq!(encoding_buf, raw_span_batch_hex);
    }

    /// The shared span batch v2 conformance vectors, produced by op-node's
    /// `TestSpanBatchV2GoldenVectors` and byte-identical to
    /// `op-node/rollup/derive/testdata/`. Each file holds the ASCII hex of a complete typed
    /// batch: the `0x02` type byte followed by the span batch prefix and payload.
    #[test]
    fn test_decode_encode_raw_span_batch_v2() {
        // (file, same_ts_bits, derived timestamps)
        let vectors: [(&str, &[u8], &[u64]); 2] = [
            (
                include_str!("./testdata/raw_batch_v2.hex"),
                &[0, 1, 1, 0, 1],
                &[1010, 1010, 1010, 1012, 1012],
            ),
            (include_str!("./testdata/raw_batch_v2_bit0_set.hex"), &[1, 1, 0], &[1010, 1010, 1012]),
        ];

        for (encoded, same_ts_bits, timestamps) in vectors {
            let encoded = alloy_primitives::hex::decode(encoded.trim()).unwrap();
            let (batch_type, payload) = encoded.split_first().unwrap();
            assert_eq!(BatchType::try_from(*batch_type), Ok(BatchType::SpanV2));

            let mut raw_span_batch =
                RawSpanBatch::decode(&mut &payload[..], BatchType::SpanV2).unwrap();
            let decoded_bits = raw_span_batch.payload.same_ts_bits.clone().unwrap();
            for (i, bit) in same_ts_bits.iter().enumerate() {
                assert_eq!(decoded_bits.get_bit(i), Some(*bit), "same_ts_bit {i}");
            }

            let span_batch = raw_span_batch.derive(2, 1000, 1234).unwrap();
            assert_eq!(
                span_batch.batches.iter().map(|b| b.timestamp).collect::<Vec<_>>(),
                timestamps
            );

            let mut re_encoded = Vec::new();
            raw_span_batch.encode(&mut re_encoded).unwrap();
            assert_eq!(re_encoded, payload);
        }
    }

    /// The golden v2 vector, minus its type byte.
    fn golden_v2_batch() -> RawSpanBatch {
        let encoded =
            alloy_primitives::hex::decode(include_str!("./testdata/raw_batch_v2.hex").trim())
                .unwrap();
        RawSpanBatch::decode(&mut &encoded[1..], BatchType::SpanV2).unwrap()
    }

    /// The golden vector's bytes up to and including `origin_bits`, i.e. everything that precedes
    /// the `same_ts_bits` bitlist.
    fn golden_v2_bytes_before_same_ts_bits() -> Vec<u8> {
        let raw = golden_v2_batch();
        let mut buf = Vec::new();
        raw.prefix.encode_prefix(&mut buf);
        raw.payload.encode_block_count(&mut buf);
        raw.payload.encode_origin_bits(&mut buf).unwrap();
        buf
    }

    /// Re-encodes the golden vector with the `same_ts_bits` bitlist replaced by `bits`, or omitted
    /// entirely when `bits` is `None`.
    fn golden_v2_bytes_with_same_ts_bits(bits: Option<&[u8]>) -> Vec<u8> {
        let raw = golden_v2_batch();
        let mut buf = golden_v2_bytes_before_same_ts_bits();
        if let Some(bits) = bits {
            buf.extend_from_slice(bits);
        }
        raw.payload.encode_block_tx_counts(&mut buf);
        raw.payload.encode_txs(&mut buf).unwrap();
        buf
    }

    /// A `same_ts_bits` bitlist that is not there at all must not be read out of the fields that
    /// follow it: the first `block_tx_counts` byte lands where the bitlist is expected, and its
    /// value is out of range for the block count. op-node's twin is
    /// `TestSpanBatchV2RejectsTruncatedSameTsBits`.
    #[test]
    fn test_decode_raw_span_batch_v2_missing_same_ts_bits() {
        let encoded = golden_v2_bytes_with_same_ts_bits(None);
        let err = RawSpanBatch::decode(&mut &encoded[..], BatchType::SpanV2).unwrap_err();
        assert_eq!(err, SpanBatchError::BitfieldTooLong);
    }

    /// Truncated input is rejected rather than zero-padded, matching the `origin_bits` rule in
    /// [`crate::SpanBatchBits::decode`].
    #[test]
    fn test_decode_raw_span_batch_v2_short_same_ts_bits() {
        let truncated = golden_v2_bytes_before_same_ts_bits();
        let err = RawSpanBatch::decode(&mut &truncated[..], BatchType::SpanV2).unwrap_err();
        assert_eq!(err, SpanBatchError::BitfieldTooShort);
    }

    /// A bitlist with a bit set past `block_count` is over-long, exactly as for `origin_bits`.
    #[test]
    fn test_decode_raw_span_batch_v2_long_same_ts_bits() {
        // The golden vector has five elements, so a set bit 7 is out of range.
        let encoded = golden_v2_bytes_with_same_ts_bits(Some(&[0b1000_0000]));
        let err = RawSpanBatch::decode(&mut &encoded[..], BatchType::SpanV2).unwrap_err();
        assert_eq!(err, SpanBatchError::BitfieldTooLong);
    }

    /// A span batch can only be decoded as one of the span wire versions; the type byte and the
    /// version cannot get out of step.
    #[test]
    fn test_decode_raw_span_batch_rejects_non_span_version() {
        let encoded = golden_v2_bytes_with_same_ts_bits(Some(&[0b0001_0110]));
        let err = RawSpanBatch::decode(&mut &encoded[..], BatchType::Single).unwrap_err();
        assert_eq!(err, SpanBatchError::NotASpanVersion);
    }

    /// Re-encodes a real op-node span batch with a non-minimally encoded `block_count`, leaving
    /// every other field untouched — the batch a byzantine batcher would publish to split
    /// derivation. op-node decodes it to the same batch as the minimal encoding, so kona must too.
    /// `TestSpanBatchNonMinimalBlockCount` runs the same scenario against op-node, on its own
    /// batch — unlike the field-level vectors, the bytes here are not shared between the suites.
    #[test]
    fn test_decode_raw_span_batch_non_minimal_block_count() {
        let raw_span_batch_hex = include_bytes!("./testdata/raw_batch.hex");
        let expected =
            RawSpanBatch::decode(&mut raw_span_batch_hex.as_slice(), BatchType::Span).unwrap();

        let mut varint_buf = [0u8; 10];
        let minimal =
            unsigned_varint::encode::u64(expected.payload.block_count, &mut varint_buf).to_vec();
        let (last, head) = minimal.split_last().unwrap();

        let mut buf = Vec::new();
        expected.prefix.encode_prefix(&mut buf);
        buf.extend_from_slice(head);
        buf.push(last | 0x80); // mark the final byte as continuing…
        buf.push(0x00); // …into a redundant zero terminator
        expected.payload.encode_origin_bits(&mut buf).unwrap();
        expected.payload.encode_block_tx_counts(&mut buf);
        expected.payload.encode_txs(&mut buf).unwrap();

        assert_ne!(buf, raw_span_batch_hex, "block_count must be re-encoded");
        assert_eq!(RawSpanBatch::decode(&mut buf.as_slice(), BatchType::Span).unwrap(), expected);
    }
}
