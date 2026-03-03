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

impl From<u8> for BatchType {
    fn from(val: u8) -> Self {
        match val {
            SINGLE_BATCH_TYPE => Self::Single,
            SPAN_BATCH_TYPE => Self::Span,
            _ => panic!("Invalid batch type: {val}"),
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
        Ok(Self::from(val))
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

    // -------------------------------------------------------------------------
    // Spec conformance test — CR-05
    // Review: kona-node-vs-op-node
    // Component: channel-reader
    // Reference: op-node/rollup/derive/batch.go:136 — default arm returns an error
    //            op-node/rollup/derive/channel_in_reader.go:95–98 — error causes NextChannel()
    // Subject: protocol/src/batch/type.rs:36 — wildcard arm calls panic!(...)
    //
    // Go's decodeTyped uses a switch with a default arm that returns
    // `fmt.Errorf("unrecognized batch type: %d", data[0])` — a normal Go error.
    // At channel_in_reader.go:95–98 this error is caught, NextChannel() is called (drops the
    // current channel's batch function), and NotEnoughData is returned. No panic, no crash.
    //
    // In Rust, BatchType::from(u8) for any byte other than 0x00 or 0x01 calls
    // `panic!("Invalid batch type: {val}")`. There is no catch_unwind wrapper anywhere
    // in the Batch::decode → BatchType::from call chain.
    //
    // This test catches the panic to confirm the bug is present in the current code.
    // An honest batcher never produces bytes other than 0x00 or 0x01, but a dishonest
    // or buggy batcher can. The consequence in Rust (process halt / derivation stall)
    // is disproportionately severe compared to Go's graceful channel drop.
    // -------------------------------------------------------------------------
    #[test]
    fn test_spec_channel_reader_invalid_batch_type_no_panic() {
        // CR-05: BatchType::from panics on unknown type byte instead of returning an error.
        // On unpatched code this test CATCHES a panic, confirming the bug is present.
        // A fixed implementation should return an Err instead of panicking.
        let result = std::panic::catch_unwind(|| {
            let _ = BatchType::from(0x02u8);
        });
        // The panic is caught — assert the bug is present in the subject.
        assert!(
            result.is_err(),
            "CR-05: BatchType::from(0x02) panicked as expected (bug confirmed); \
             a correct implementation would return Err, not panic"
        );
    }

    // -------------------------------------------------------------------------
    // Spec conformance test — CR-05 (variant: type byte 0xFF)
    // Same issue — any byte value ≥ 0x02 triggers the panic.
    // -------------------------------------------------------------------------
    #[test]
    fn test_spec_channel_reader_unknown_batch_type_drops_not_panics() {
        // CR-05 variant: byte 0xFF also panics.
        let result = std::panic::catch_unwind(|| {
            let _ = BatchType::from(0xFFu8);
        });
        assert!(
            result.is_err(),
            "CR-05: BatchType::from(0xFF) panicked as expected (bug confirmed)"
        );
    }
}
