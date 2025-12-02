//! Raw Span Batch Payload

use super::MAX_SPAN_BATCH_ELEMENTS;
use crate::{SpanBatchBits, SpanBatchError, SpanBatchTransactions, SpanDecodingError};
use alloc::vec::Vec;
use alloy_primitives::bytes;

/// Span Batch Payload
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct SpanBatchPayload {
    /// Number of L2 block in the span
    pub block_count: u64,
    /// Standard span-batch bitlist of blockCount bits. Each bit indicates if the L1 origin is
    /// changed at the L2 block.
    pub origin_bits: SpanBatchBits,
    /// List of transaction counts for each L2 block
    pub block_tx_counts: Vec<u64>,
    /// Transactions encoded in SpanBatch specs
    pub txs: SpanBatchTransactions,
}

impl SpanBatchPayload {
    /// Decodes a [`SpanBatchPayload`] from a reader.
    pub fn decode_payload(r: &mut &[u8]) -> Result<Self, SpanBatchError> {
        let mut payload = Self::default();
        payload.decode_block_count(r)?;
        payload.decode_origin_bits(r)?;
        payload.decode_block_tx_counts(r)?;
        payload.decode_txs(r)?;
        Ok(payload)
    }

    /// Encodes a [`SpanBatchPayload`] into a writer.
    pub fn encode_payload(&self, w: &mut dyn bytes::BufMut) -> Result<(), SpanBatchError> {
        self.encode_block_count(w);
        self.encode_origin_bits(w)?;
        self.encode_block_tx_counts(w);
        self.encode_txs(w)
    }

    /// Decodes the origin bits from a reader.
    pub fn decode_origin_bits(&mut self, r: &mut &[u8]) -> Result<(), SpanBatchError> {
        if self.block_count > MAX_SPAN_BATCH_ELEMENTS {
            return Err(SpanBatchError::TooBigSpanBatchSize);
        }

        self.origin_bits = SpanBatchBits::decode(r, self.block_count as usize)?;
        Ok(())
    }

    /// Decode a block count from a reader.
    pub fn decode_block_count(&mut self, r: &mut &[u8]) -> Result<(), SpanBatchError> {
        let (block_count, remaining) = unsigned_varint::decode::u64(r)
            .map_err(|_| SpanBatchError::Decoding(SpanDecodingError::BlockCount))?;
        // The number of transactions in a single L2 block cannot be greater than
        // [MAX_SPAN_BATCH_ELEMENTS].
        if block_count > MAX_SPAN_BATCH_ELEMENTS {
            return Err(SpanBatchError::TooBigSpanBatchSize);
        }
        if block_count == 0 {
            return Err(SpanBatchError::EmptySpanBatch);
        }
        self.block_count = block_count;
        *r = remaining;
        Ok(())
    }

    /// Decode block transaction counts from a reader.
    pub fn decode_block_tx_counts(&mut self, r: &mut &[u8]) -> Result<(), SpanBatchError> {
        // Initially allocate the vec with the block count, to reduce re-allocations in the first
        // few blocks.
        let mut block_tx_counts = Vec::with_capacity(self.block_count as usize);

        for _ in 0..self.block_count {
            let (block_tx_count, remaining) = unsigned_varint::decode::u64(r)
                .map_err(|_| SpanBatchError::Decoding(SpanDecodingError::BlockTxCounts))?;

            // The number of transactions in a single L2 block cannot be greater than
            // [MAX_SPAN_BATCH_ELEMENTS].
            if block_tx_count > MAX_SPAN_BATCH_ELEMENTS {
                return Err(SpanBatchError::TooBigSpanBatchSize);
            }
            block_tx_counts.push(block_tx_count);
            *r = remaining;
        }
        self.block_tx_counts = block_tx_counts;
        Ok(())
    }

    /// Decode transactions from a reader.
    pub fn decode_txs(&mut self, r: &mut &[u8]) -> Result<(), SpanBatchError> {
        if self.block_tx_counts.is_empty() {
            return Err(SpanBatchError::EmptySpanBatch);
        }

        let total_block_tx_count =
            self.block_tx_counts.iter().try_fold(0u64, |acc, block_tx_count| {
                acc.checked_add(*block_tx_count).ok_or(SpanBatchError::TooBigSpanBatchSize)
            })?;

        // The total number of transactions in a span batch cannot be greater than
        // [MAX_SPAN_BATCH_ELEMENTS].
        if total_block_tx_count > MAX_SPAN_BATCH_ELEMENTS {
            return Err(SpanBatchError::TooBigSpanBatchSize);
        }
        self.txs.total_block_tx_count = total_block_tx_count;
        self.txs.decode(r)?;
        Ok(())
    }

    /// Encode the origin bits into a writer.
    pub fn encode_origin_bits(&self, w: &mut dyn bytes::BufMut) -> Result<(), SpanBatchError> {
        SpanBatchBits::encode(w, self.block_count as usize, &self.origin_bits)
    }

    /// Encode the block count into a writer.
    pub fn encode_block_count(&self, w: &mut dyn bytes::BufMut) {
        let mut u64_varint_buf = [0u8; 10];
        w.put_slice(unsigned_varint::encode::u64(self.block_count, &mut u64_varint_buf));
    }

    /// Encode the block transaction counts into a writer.
    pub fn encode_block_tx_counts(&self, w: &mut dyn bytes::BufMut) {
        let mut u64_varint_buf = [0u8; 10];
        for block_tx_count in &self.block_tx_counts {
            u64_varint_buf.fill(0);
            w.put_slice(unsigned_varint::encode::u64(*block_tx_count, &mut u64_varint_buf));
        }
    }

    /// Encode the transactions into a writer.
    pub fn encode_txs(&self, w: &mut dyn bytes::BufMut) -> Result<(), SpanBatchError> {
        self.txs.encode(w)
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use alloc::vec;

    #[test]
    fn test_decode_origin_bits() {
        let block_count = 10;
        let encoded = vec![2; block_count / 8 + 1];
        let mut payload =
            SpanBatchPayload { block_count: block_count as u64, ..Default::default() };
        payload.decode_origin_bits(&mut encoded.as_slice()).unwrap();
        assert_eq!(payload.origin_bits, SpanBatchBits::new(vec![2; block_count / 8 + 1]));
    }

    #[test]
    fn test_zero_block_count() {
        let mut u64_varint_buf = [0; 10];
        let mut encoded = unsigned_varint::encode::u64(0, &mut u64_varint_buf);
        let mut payload = SpanBatchPayload::default();
        let err = payload.decode_block_count(&mut encoded).unwrap_err();
        assert_eq!(err, SpanBatchError::EmptySpanBatch);
    }

    #[test]
    fn test_decode_block_count() {
        let block_count = MAX_SPAN_BATCH_ELEMENTS;
        let mut u64_varint_buf = [0; 10];
        let mut encoded = unsigned_varint::encode::u64(block_count, &mut u64_varint_buf);
        let mut payload = SpanBatchPayload::default();
        payload.decode_block_count(&mut encoded).unwrap();
        assert_eq!(payload.block_count, block_count);
    }

    #[test]
    fn test_decode_block_count_errors() {
        let block_count = MAX_SPAN_BATCH_ELEMENTS + 1;
        let mut u64_varint_buf = [0; 10];
        let mut encoded = unsigned_varint::encode::u64(block_count, &mut u64_varint_buf);
        let mut payload = SpanBatchPayload::default();
        let err = payload.decode_block_count(&mut encoded).unwrap_err();
        assert_eq!(err, SpanBatchError::TooBigSpanBatchSize);
    }

    #[test]
    fn test_decode_block_tx_counts() {
        let block_count = 2;
        let mut u64_varint_buf = [0; 10];
        let mut encoded = unsigned_varint::encode::u64(block_count, &mut u64_varint_buf);
        let mut payload = SpanBatchPayload::default();
        payload.decode_block_count(&mut encoded).unwrap();
        let mut r: Vec<u8> = Vec::new();
        for _ in 0..2 {
            let mut buf = [0u8; 10];
            let encoded = unsigned_varint::encode::u64(2, &mut buf);
            r.append(&mut encoded.to_vec());
        }
        payload.decode_block_tx_counts(&mut r.as_slice()).unwrap();
        assert_eq!(payload.block_tx_counts, vec![2, 2]);
    }
}
