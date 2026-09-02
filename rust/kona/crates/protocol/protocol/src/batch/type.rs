//! Batch Types
//!
//! This module contains the batch types for the OP Stack derivation pipeline.
//!
//! ## Batch
//!
//! A batch is either a `SpanBatch` or a `SingleBatch`.
//!
//! The batch type is encoded as a single byte:
//! - `0x00` for a `SingleBatch`
//! - `0x01` for a `SpanBatch`
//! - `0x02` for a `SpanBatch` carrying the `same_ts_bits` bitlist

use alloy_rlp::{Decodable, Encodable};

/// The single batch type identifier.
pub const SINGLE_BATCH_TYPE: u8 = 0x00;

/// The span batch type identifier.
pub const SPAN_BATCH_TYPE: u8 = 0x01;

/// The span batch v2 type identifier.
pub const SPAN_BATCH_V2_TYPE: u8 = 0x02;

/// The Batch Type.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
#[repr(u8)]
pub enum BatchType {
    /// Single Batch.
    Single = SINGLE_BATCH_TYPE,
    /// Span Batch.
    Span = SPAN_BATCH_TYPE,
    /// Span Batch with the `same_ts_bits` bitlist, which lets consecutive elements share a
    /// timestamp.
    SpanV2 = SPAN_BATCH_V2_TYPE,
}

impl BatchType {
    /// Returns true if this batch type carries the `same_ts_bits` bitlist.
    pub const fn has_same_ts_bits(&self) -> bool {
        matches!(self, Self::SpanV2)
    }

    /// Returns true if this batch type is one of the span batch wire versions.
    pub const fn is_span(&self) -> bool {
        matches!(self, Self::Span | Self::SpanV2)
    }
}

impl TryFrom<u8> for BatchType {
    type Error = u8;

    fn try_from(val: u8) -> Result<Self, Self::Error> {
        match val {
            SINGLE_BATCH_TYPE => Ok(Self::Single),
            SPAN_BATCH_TYPE => Ok(Self::Span),
            SPAN_BATCH_V2_TYPE => Ok(Self::SpanV2),
            _ => Err(val),
        }
    }
}

impl Encodable for BatchType {
    fn encode(&self, out: &mut dyn alloy_rlp::BufMut) {
        let val = match self {
            Self::Single => SINGLE_BATCH_TYPE,
            Self::Span => SPAN_BATCH_TYPE,
            Self::SpanV2 => SPAN_BATCH_V2_TYPE,
        };
        val.encode(out);
    }
}

impl Decodable for BatchType {
    fn decode(buf: &mut &[u8]) -> alloy_rlp::Result<Self> {
        let val = u8::decode(buf)?;
        Self::try_from(val).map_err(|_| alloy_rlp::Error::Custom("invalid batch type"))
    }
}

#[cfg(test)]
mod test {
    use super::*;
    use alloc::vec::Vec;

    #[test]
    fn test_batch_type_rlp_roundtrip() {
        for batch_type in [BatchType::Single, BatchType::Span, BatchType::SpanV2] {
            let mut buf = Vec::new();
            batch_type.encode(&mut buf);
            let decoded = BatchType::decode(&mut buf.as_slice()).unwrap();
            assert_eq!(batch_type, decoded);
        }
    }

    #[test]
    fn test_try_from_valid_types() {
        assert_eq!(BatchType::try_from(SINGLE_BATCH_TYPE), Ok(BatchType::Single));
        assert_eq!(BatchType::try_from(SPAN_BATCH_TYPE), Ok(BatchType::Span));
        assert_eq!(BatchType::try_from(SPAN_BATCH_V2_TYPE), Ok(BatchType::SpanV2));
    }

    #[test]
    fn test_try_from_unknown_type_returns_error() {
        assert_eq!(BatchType::try_from(0xFF), Err(0xFF));
        assert_eq!(BatchType::try_from(0x03), Err(0x03));
    }

    #[test]
    fn test_rlp_decode_unknown_type_returns_error() {
        let mut buf = Vec::new();
        // RLP-encode an invalid batch type byte
        0xFFu8.encode(&mut buf);
        let result = BatchType::decode(&mut buf.as_slice());
        assert!(result.is_err());
    }
}
