//! Raw Span Batch Prefix

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
        let (rel_timestamp, remaining) = unsigned_varint::decode::u64(r)
            .map_err(|_| SpanBatchError::Decoding(SpanDecodingError::RelativeTimestamp))?;
        *r = remaining;
        self.rel_timestamp = rel_timestamp;
        Ok(())
    }

    /// Decodes the L1 origin number from a reader.
    pub fn decode_l1_origin_num(&mut self, r: &mut &[u8]) -> Result<(), SpanBatchError> {
        let (l1_origin_num, remaining) = unsigned_varint::decode::u64(r)
            .map_err(|_| SpanBatchError::Decoding(SpanDecodingError::L1OriginNumber))?;
        *r = remaining;
        self.l1_origin_num = l1_origin_num;
        Ok(())
    }

    /// Decodes the parent check from a reader.
    pub fn decode_parent_check(&mut self, r: &mut &[u8]) -> Result<(), SpanBatchError> {
        let (parent_check, remaining) = r.split_at(20);
        let parent_check = FixedBytes::<20>::from_slice(parent_check);
        *r = remaining;
        self.parent_check = parent_check;
        Ok(())
    }

    /// Decodes the L1 origin check from a reader.
    pub fn decode_l1_origin_check(&mut self, r: &mut &[u8]) -> Result<(), SpanBatchError> {
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
