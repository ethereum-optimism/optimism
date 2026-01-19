//! Module containing the core [`Batch`] enum.

use crate::{
    BatchDecodingError, BatchEncodingError, BatchType, RawSpanBatch, SingleBatch, SpanBatch,
};
use alloy_primitives::bytes;
use alloy_rlp::{Buf, Decodable, Encodable};
use kona_genesis::RollupConfig;

/// A Batch.
#[derive(Debug, Clone, PartialEq, Eq)]
#[allow(clippy::large_enum_variant)]
pub enum Batch {
    /// A single batch
    Single(SingleBatch),
    /// Span Batches
    Span(SpanBatch),
}

impl core::fmt::Display for Batch {
    fn fmt(&self, f: &mut core::fmt::Formatter<'_>) -> core::fmt::Result {
        match self {
            Self::Single(_) => write!(f, "single"),
            Self::Span(_) => write!(f, "span"),
        }
    }
}

impl Batch {
    /// Returns the timestamp for the batch.
    pub fn timestamp(&self) -> u64 {
        match self {
            Self::Single(sb) => sb.timestamp,
            Self::Span(sb) => sb.starting_timestamp(),
        }
    }

    /// Attempts to decode a batch from a reader.
    pub fn decode(r: &mut &[u8], cfg: &RollupConfig) -> Result<Self, BatchDecodingError> {
        if r.is_empty() {
            return Err(BatchDecodingError::EmptyBuffer);
        }

        // Read the batch type
        let batch_type = BatchType::from(r[0]);
        r.advance(1);

        match batch_type {
            BatchType::Single => {
                let single_batch =
                    SingleBatch::decode(r).map_err(BatchDecodingError::AlloyRlpError)?;
                Ok(Self::Single(single_batch))
            }
            BatchType::Span => {
                let mut raw_span_batch = RawSpanBatch::decode(r)?;
                let span_batch = raw_span_batch
                    .derive(cfg.block_time, cfg.genesis.l2_time, cfg.l2_chain_id.id())
                    .map_err(BatchDecodingError::SpanBatchError)?;
                Ok(Self::Span(span_batch))
            }
        }
    }

    /// Attempts to encode the batch to a writer.
    pub fn encode(&self, out: &mut dyn bytes::BufMut) -> Result<(), BatchEncodingError> {
        match self {
            Self::Single(sb) => {
                out.put_u8(BatchType::Single as u8);
                sb.encode(out);
            }
            Self::Span(sb) => {
                out.put_u8(BatchType::Span as u8);
                let raw_span_batch =
                    sb.to_raw_span_batch().map_err(BatchEncodingError::SpanBatchError)?;
                raw_span_batch.encode(out).map_err(BatchEncodingError::SpanBatchError)?;
            }
        }
        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::{SpanBatchElement, SpanBatchError, SpanBatchTransactions};
    use alloc::{vec, vec::Vec};
    use alloy_consensus::{Signed, TxEip2930, TxEnvelope};
    use alloy_primitives::{Bytes, Signature, TxKind, address, hex};

    #[test]
    fn test_single_batch_encode_decode() {
        let mut out = Vec::new();
        let batch = Batch::Single(SingleBatch::default());
        batch.encode(&mut out).unwrap();
        let decoded = Batch::decode(&mut out.as_slice(), &RollupConfig::default()).unwrap();
        assert_eq!(batch, decoded);
    }

    #[test]
    fn test_span_batch_encode_decode() {
        let sig = Signature::test_signature();
        let to = address!("0123456789012345678901234567890123456789");
        let tx = TxEnvelope::Eip2930(Signed::new_unchecked(
            TxEip2930 { to: TxKind::Call(to), chain_id: 1, ..Default::default() },
            sig,
            Default::default(),
        ));
        let mut span_batch_txs = SpanBatchTransactions::default();
        let mut buf = vec![];
        tx.encode(&mut buf);
        let txs = vec![Bytes::from(buf)];
        let chain_id = 1;
        span_batch_txs.add_txs(txs, chain_id).unwrap();

        let mut out = Vec::new();
        let batch = Batch::Span(SpanBatch {
            block_tx_counts: vec![1],
            batches: vec![SpanBatchElement::default()],
            txs: span_batch_txs,
            ..Default::default()
        });
        batch.encode(&mut out).unwrap();
        let decoded = Batch::decode(&mut out.as_slice(), &RollupConfig::default()).unwrap();
        assert_eq!(Batch::Span(SpanBatch {
            batches: vec![SpanBatchElement {
                transactions: vec![hex!("01f85f808080809401234567890123456789012345678901234567898080c080a0840cfc572845f5786e702984c2a582528cad4b49b2a10b9db1be7fca90058565a025e7109ceb98168d95b09b18bbf6b685130e0562f233877d492b94eee0c5b6d1").into()],
                ..Default::default()
            }],
            txs: SpanBatchTransactions::default(),
            ..Default::default()
        }), decoded);
    }

    #[test]
    fn test_empty_span_batch() {
        let mut out = Vec::new();
        let batch = Batch::Span(SpanBatch::default());
        // Fails to even encode an empty span batch - decoding will do the same
        let err = batch.encode(&mut out).unwrap_err();
        assert_eq!(BatchEncodingError::SpanBatchError(SpanBatchError::EmptySpanBatch), err);
    }
}
