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

use alloy_rlp::{Decodable, Encodable};

/// The single batch type identifier.
pub const SINGLE_BATCH_TYPE: u8 = 0x00;

/// The span batch type identifier.
pub const SPAN_BATCH_TYPE: u8 = 0x01;

/// The Batch Type.
#[derive(Debug, Clone, PartialEq, Eq)]
#[repr(u8)]
pub enum BatchType {
    /// Single Batch.
    Single = SINGLE_BATCH_TYPE,
    /// Span Batch.
    Span = SPAN_BATCH_TYPE,
}

impl TryFrom<u8> for BatchType {
    type Error = u8;

    fn try_from(val: u8) -> Result<Self, Self::Error> {
        match val {
            SINGLE_BATCH_TYPE => Ok(Self::Single),
            SPAN_BATCH_TYPE => Ok(Self::Span),
            _ => Err(val),
        }
    }
}

impl Encodable for BatchType {
    fn encode(&self, out: &mut dyn alloy_rlp::BufMut) {
        let val = match self {
            Self::Single => SINGLE_BATCH_TYPE,
            Self::Span => SPAN_BATCH_TYPE,
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
        let batch_type = BatchType::Single;
        let mut buf = Vec::new();
        batch_type.encode(&mut buf);
        let decoded = BatchType::decode(&mut buf.as_slice()).unwrap();
        assert_eq!(batch_type, decoded);
    }

    #[test]
    fn test_try_from_valid_types() {
        assert_eq!(BatchType::try_from(SINGLE_BATCH_TYPE), Ok(BatchType::Single));
        assert_eq!(BatchType::try_from(SPAN_BATCH_TYPE), Ok(BatchType::Span));
    }

    #[test]
    fn test_try_from_unknown_type_returns_error() {
        assert_eq!(BatchType::try_from(0xFF), Err(0xFF));
        assert_eq!(BatchType::try_from(0x02), Err(0x02));
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
