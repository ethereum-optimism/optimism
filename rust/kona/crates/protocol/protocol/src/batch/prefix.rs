//! Raw Span Batch Prefix

use super::varint::read_uvarint;
use crate::{SpanBatchError, SpanDecodingError};
use alloy_primitives::{FixedBytes, bytes};

/// Span Batch Prefix
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct SpanBatchPrefix {
    /// Relative timestamp of the first block
    pub rel_timestamp: u64,
    /// L1 origin number
    pub l1_origin_num: u64,
    /// First 20 bytes of the first block's parent hash
    pub parent_check: FixedBytes<20>,
    /// First 20 bytes of the last block's L1 origin hash
    pub l1_origin_check: FixedBytes<20>,
}

impl SpanBatchPrefix {
    /// Decodes a [`SpanBatchPrefix`] from a reader.
    pub fn decode_prefix(r: &mut &[u8]) -> Result<Self, SpanBatchError> {
        let mut prefix = Self::default();
        prefix.decode_rel_timestamp(r)?;
        prefix.decode_l1_origin_num(r)?;
        prefix.decode_parent_check(r)?;
        prefix.decode_l1_origin_check(r)?;
        Ok(prefix)
    }

    /// Decodes the relative timestamp from a reader.
    pub fn decode_rel_timestamp(&mut self, r: &mut &[u8]) -> Result<(), SpanBatchError> {
        let (rel_timestamp, remaining) = read_uvarint(r)
            .ok_or(SpanBatchError::Decoding(SpanDecodingError::RelativeTimestamp))?;
        *r = remaining;
        self.rel_timestamp = rel_timestamp;
        Ok(())
    }

    /// Decodes the L1 origin number from a reader.
    pub fn decode_l1_origin_num(&mut self, r: &mut &[u8]) -> Result<(), SpanBatchError> {
        let (l1_origin_num, remaining) =
            read_uvarint(r).ok_or(SpanBatchError::Decoding(SpanDecodingError::L1OriginNumber))?;
        *r = remaining;
        self.l1_origin_num = l1_origin_num;
        Ok(())
    }

    /// Decodes the parent check from a reader.
    pub fn decode_parent_check(&mut self, r: &mut &[u8]) -> Result<(), SpanBatchError> {
        if r.len() < 20 {
            return Err(SpanBatchError::Decoding(SpanDecodingError::ParentCheck));
        }
        let (parent_check, remaining) = r.split_at(20);
        let parent_check = FixedBytes::<20>::from_slice(parent_check);
        *r = remaining;
        self.parent_check = parent_check;
        Ok(())
    }

    /// Decodes the L1 origin check from a reader.
    pub fn decode_l1_origin_check(&mut self, r: &mut &[u8]) -> Result<(), SpanBatchError> {
        if r.len() < 20 {
            return Err(SpanBatchError::Decoding(SpanDecodingError::L1OriginCheck));
        }
        let (l1_origin_check, remaining) = r.split_at(20);
        let l1_origin_check = FixedBytes::<20>::from_slice(l1_origin_check);
        *r = remaining;
        self.l1_origin_check = l1_origin_check;
        Ok(())
    }

    /// Encodes the [`SpanBatchPrefix`] into a writer.
    pub fn encode_prefix(&self, w: &mut dyn bytes::BufMut) {
        let mut u64_buf = [0u8; 10];
        w.put_slice(unsigned_varint::encode::u64(self.rel_timestamp, &mut u64_buf));
        w.put_slice(unsigned_varint::encode::u64(self.l1_origin_num, &mut u64_buf));
        w.put_slice(self.parent_check.as_slice());
        w.put_slice(self.l1_origin_check.as_slice());
    }
}

#[cfg(test)]
mod test {
    use super::*;
    use alloc::vec::Vec;
    use alloy_primitives::address;

    #[test]
    fn test_decode_parent_check_truncated() {
        let mut prefix = SpanBatchPrefix::default();
        let buf = [0u8; 19]; // one byte short
        let result = prefix.decode_parent_check(&mut buf.as_slice());
        assert_eq!(result, Err(SpanBatchError::Decoding(SpanDecodingError::ParentCheck)));
    }

    #[test]
    fn test_decode_l1_origin_check_truncated() {
        let mut prefix = SpanBatchPrefix::default();
        let buf = [0u8; 19]; // one byte short
        let result = prefix.decode_l1_origin_check(&mut buf.as_slice());
        assert_eq!(result, Err(SpanBatchError::Decoding(SpanDecodingError::L1OriginCheck)));
    }

    #[test]
    fn test_decode_parent_check_empty() {
        let mut prefix = SpanBatchPrefix::default();
        let result = prefix.decode_parent_check(&mut [].as_slice());
        assert_eq!(result, Err(SpanBatchError::Decoding(SpanDecodingError::ParentCheck)));
    }

    // Span-batch `uvarint` fields are protobuf Base128 varints, which op-node decodes with Go's
    // `binary.ReadUvarint`. Kona must accept exactly the encodings op-node accepts: any
    // disagreement on batcher-controlled bytes forks the two derivations on identical L1 data.
    // `rel_timestamp` is unbounded, so it exercises the full accepted range. The vectors are
    // shared with `op-node/rollup/derive/span_batch_test.go`.

    #[test]
    fn test_decode_rel_timestamp_non_minimal() {
        let mut prefix = SpanBatchPrefix::default();
        // `1` with a redundant trailing zero byte; the minimal form is `[0x01]`.
        let buf = [0x81, 0x00];
        let mut r = buf.as_slice();
        prefix.decode_rel_timestamp(&mut r).unwrap();
        assert_eq!(prefix.rel_timestamp, 1);
        assert!(r.is_empty(), "both bytes must be consumed");
    }

    #[test]
    fn test_decode_rel_timestamp_non_minimal_multi_byte() {
        let mut prefix = SpanBatchPrefix::default();
        // `1` padded with three redundant continuation bytes.
        let buf = [0x81, 0x80, 0x80, 0x00];
        let mut r = buf.as_slice();
        prefix.decode_rel_timestamp(&mut r).unwrap();
        assert_eq!(prefix.rel_timestamp, 1);
        assert!(r.is_empty(), "all four bytes must be consumed");
    }

    #[test]
    fn test_decode_rel_timestamp_ten_byte_zero_terminator() {
        let mut prefix = SpanBatchPrefix::default();
        // `1` padded out to the ten-byte maximum. The terminator contributes no bits.
        let buf = [0x81, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x00];
        let mut r = buf.as_slice();
        prefix.decode_rel_timestamp(&mut r).unwrap();
        assert_eq!(prefix.rel_timestamp, 1);
        assert!(r.is_empty(), "all ten bytes must be consumed");
    }

    #[test]
    fn test_decode_rel_timestamp_ten_byte_one_terminator() {
        let mut prefix = SpanBatchPrefix::default();
        // A ten-byte varint whose terminator sets bit 63 exactly — the largest accepted width.
        let buf = [0x81, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x01];
        let mut r = buf.as_slice();
        prefix.decode_rel_timestamp(&mut r).unwrap();
        assert_eq!(prefix.rel_timestamp, 1 | 1 << 63);
        assert!(r.is_empty(), "all ten bytes must be consumed");
    }

    #[test]
    fn test_decode_rel_timestamp_max_u64() {
        let mut prefix = SpanBatchPrefix::default();
        let buf = [0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0x01];
        let mut r = buf.as_slice();
        prefix.decode_rel_timestamp(&mut r).unwrap();
        assert_eq!(prefix.rel_timestamp, u64::MAX);
        assert!(r.is_empty(), "all ten bytes must be consumed");
    }

    #[test]
    fn test_decode_rel_timestamp_overflow_ten_byte_terminator() {
        let mut prefix = SpanBatchPrefix::default();
        // The tenth byte terminates the varint but sets bits above 63, so the value does not
        // fit in a `u64`.
        let buf = [0x81, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x02];
        let result = prefix.decode_rel_timestamp(&mut buf.as_slice());
        assert_eq!(result, Err(SpanBatchError::Decoding(SpanDecodingError::RelativeTimestamp)));
    }

    #[test]
    fn test_decode_rel_timestamp_overflow_past_max_width() {
        let mut prefix = SpanBatchPrefix::default();
        // The varint would only terminate on the eleventh byte, past the maximum width read.
        let buf = [0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x01];
        let result = prefix.decode_rel_timestamp(&mut buf.as_slice());
        assert_eq!(result, Err(SpanBatchError::Decoding(SpanDecodingError::RelativeTimestamp)));
    }

    #[test]
    fn test_decode_rel_timestamp_minimal() {
        let mut prefix = SpanBatchPrefix::default();
        let buf = [0x01];
        let mut r = buf.as_slice();
        prefix.decode_rel_timestamp(&mut r).unwrap();
        assert_eq!(prefix.rel_timestamp, 1);
        assert!(r.is_empty(), "the single byte must be consumed");
    }

    #[test]
    fn test_decode_rel_timestamp_overflow_all_continuation() {
        let mut prefix = SpanBatchPrefix::default();
        // Ten continuation bytes: the varint never terminates within the maximum width.
        let buf = [0x80; 10];
        let result = prefix.decode_rel_timestamp(&mut buf.as_slice());
        assert_eq!(result, Err(SpanBatchError::Decoding(SpanDecodingError::RelativeTimestamp)));
    }

    #[test]
    fn test_decode_rel_timestamp_truncated() {
        let mut prefix = SpanBatchPrefix::default();
        // A continuation byte with no terminator following it.
        let buf = [0x81];
        let result = prefix.decode_rel_timestamp(&mut buf.as_slice());
        assert_eq!(result, Err(SpanBatchError::Decoding(SpanDecodingError::RelativeTimestamp)));
    }

    #[test]
    fn test_decode_rel_timestamp_empty() {
        let mut prefix = SpanBatchPrefix::default();
        let result = prefix.decode_rel_timestamp(&mut [].as_slice());
        assert_eq!(result, Err(SpanBatchError::Decoding(SpanDecodingError::RelativeTimestamp)));
    }

    #[test]
    fn test_decode_l1_origin_num_non_minimal() {
        let mut prefix = SpanBatchPrefix::default();
        let buf = [0x81, 0x00];
        let mut r = buf.as_slice();
        prefix.decode_l1_origin_num(&mut r).unwrap();
        assert_eq!(prefix.l1_origin_num, 1);
        assert!(r.is_empty(), "both bytes must be consumed");
    }

    #[test]
    fn test_decode_l1_origin_num_overflow_ten_byte_terminator() {
        let mut prefix = SpanBatchPrefix::default();
        let buf = [0x81, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x02];
        let result = prefix.decode_l1_origin_num(&mut buf.as_slice());
        assert_eq!(result, Err(SpanBatchError::Decoding(SpanDecodingError::L1OriginNumber)));
    }

    #[test]
    fn test_decode_prefix_non_minimal_varints() {
        let expected = SpanBatchPrefix {
            rel_timestamp: 1,
            l1_origin_num: 2,
            parent_check: address!("beef00000000000000000000000000000000beef").into(),
            l1_origin_check: address!("babe00000000000000000000000000000000babe").into(),
        };

        let mut buf = Vec::new();
        buf.extend_from_slice(&[0x81, 0x00]); // rel_timestamp = 1, non-minimal
        buf.extend_from_slice(&[0x82, 0x00]); // l1_origin_num = 2, non-minimal
        buf.extend_from_slice(expected.parent_check.as_slice());
        buf.extend_from_slice(expected.l1_origin_check.as_slice());

        assert_eq!(SpanBatchPrefix::decode_prefix(&mut buf.as_slice()).unwrap(), expected);
    }

    #[test]
    fn test_span_batch_prefix_encoding_roundtrip() {
        let expected = SpanBatchPrefix {
            rel_timestamp: 0xFF,
            l1_origin_num: 0xEE,
            parent_check: address!("beef00000000000000000000000000000000beef").into(),
            l1_origin_check: address!("babe00000000000000000000000000000000babe").into(),
        };

        let mut buf = Vec::new();
        expected.encode_prefix(&mut buf);

        assert_eq!(SpanBatchPrefix::decode_prefix(&mut buf.as_slice()).unwrap(), expected);
    }
}
