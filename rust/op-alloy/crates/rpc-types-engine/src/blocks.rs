//! Versioned wire encoding for the canonical blocks stream.
//!
//! Each frame contains a one-byte OP P2P block version followed by the exact uncompressed,
//! unsigned payload bytes used by OP Stack P2P gossip. V1/V2 frames contain the SSZ execution
//! payload. V3/V4 frames contain the 32-byte parent beacon block root followed by the SSZ
//! execution payload.

use crate::{OpExecutionPayload, OpExecutionPayloadEnvelope, OpExecutionPayloadV4};
use alloc::{format, string::String, vec::Vec};
use alloy_primitives::B256;
use core::fmt;
use ssz::{Decode, Encode};

/// P2P block version for an [`alloy_rpc_types_engine::ExecutionPayloadV1`].
pub const BLOCK_VERSION_V1: u8 = 0;
/// P2P block version for an [`alloy_rpc_types_engine::ExecutionPayloadV2`].
pub const BLOCK_VERSION_V2: u8 = 1;
/// P2P block version for an [`alloy_rpc_types_engine::ExecutionPayloadV3`].
pub const BLOCK_VERSION_V3: u8 = 2;
/// P2P block version for an [`OpExecutionPayloadV4`].
pub const BLOCK_VERSION_V4: u8 = 3;

/// Error encoding or decoding a blocks stream message.
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum BlocksWireError {
    /// The message does not contain a block-version byte.
    Empty,
    /// A V3/V4 payload is missing its required parent beacon block root.
    MissingParentBeaconBlockRoot(u8),
    /// The P2P block version is unsupported.
    UnsupportedPayloadVersion(u8),
    /// The message is too short for the indicated P2P block version.
    Truncated,
    /// A payload could not be decoded from SSZ.
    InvalidPayload(String),
}

impl fmt::Display for BlocksWireError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Self::Empty => f.write_str("empty blocks stream message"),
            Self::MissingParentBeaconBlockRoot(version) => {
                write!(f, "block version {version} requires a parent beacon block root")
            }
            Self::UnsupportedPayloadVersion(version) => {
                write!(f, "unsupported P2P block version {version}")
            }
            Self::Truncated => f.write_str("truncated blocks stream message"),
            Self::InvalidPayload(error) => write!(f, "invalid execution payload: {error}"),
        }
    }
}

impl std::error::Error for BlocksWireError {}

/// Encodes an envelope using the P2P payload encoding, prefixed by its P2P block version.
pub fn encode_block_frame(
    envelope: &OpExecutionPayloadEnvelope,
) -> Result<Vec<u8>, BlocksWireError> {
    let (block_version, payload, has_parent_beacon_block_root) = match &envelope.execution_payload {
        OpExecutionPayload::V1(payload) => (BLOCK_VERSION_V1, payload.as_ssz_bytes(), false),
        OpExecutionPayload::V2(payload) => (BLOCK_VERSION_V2, payload.as_ssz_bytes(), false),
        OpExecutionPayload::V3(payload) => (BLOCK_VERSION_V3, payload.as_ssz_bytes(), true),
        OpExecutionPayload::V4(payload) => (BLOCK_VERSION_V4, payload.as_ssz_bytes(), true),
    };

    let mut frame =
        Vec::with_capacity(1 + payload.len() + usize::from(has_parent_beacon_block_root) * 32);
    frame.push(block_version);
    if has_parent_beacon_block_root {
        let root = envelope
            .parent_beacon_block_root
            .ok_or(BlocksWireError::MissingParentBeaconBlockRoot(block_version))?;
        frame.extend_from_slice(root.as_slice());
    }
    frame.extend_from_slice(&payload);
    Ok(frame)
}

/// Decodes a P2P-version-prefixed payload produced by [`encode_block_frame`].
pub fn decode_block_frame(frame: &[u8]) -> Result<OpExecutionPayloadEnvelope, BlocksWireError> {
    let (&block_version, body) = frame.split_first().ok_or(BlocksWireError::Empty)?;
    if block_version > BLOCK_VERSION_V4 {
        return Err(BlocksWireError::UnsupportedPayloadVersion(block_version));
    }

    let (parent_beacon_block_root, payload_bytes) = if block_version >= BLOCK_VERSION_V3 {
        if body.len() < 32 {
            return Err(BlocksWireError::Truncated);
        }
        (Some(B256::from_slice(&body[..32])), &body[32..])
    } else {
        (None, body)
    };
    let decode_error =
        |error: ssz::DecodeError| BlocksWireError::InvalidPayload(format!("{error:?}"));
    let execution_payload = match block_version {
        BLOCK_VERSION_V1 => OpExecutionPayload::V1(
            alloy_rpc_types_engine::ExecutionPayloadV1::from_ssz_bytes(payload_bytes)
                .map_err(decode_error)?,
        ),
        BLOCK_VERSION_V2 => OpExecutionPayload::V2(
            alloy_rpc_types_engine::ExecutionPayloadV2::from_ssz_bytes(payload_bytes)
                .map_err(decode_error)?,
        ),
        BLOCK_VERSION_V3 => OpExecutionPayload::V3(
            alloy_rpc_types_engine::ExecutionPayloadV3::from_ssz_bytes(payload_bytes)
                .map_err(decode_error)?,
        ),
        BLOCK_VERSION_V4 => OpExecutionPayload::V4(
            OpExecutionPayloadV4::from_ssz_bytes(payload_bytes).map_err(decode_error)?,
        ),
        _ => unreachable!("unsupported versions returned above"),
    };
    Ok(OpExecutionPayloadEnvelope { parent_beacon_block_root, execution_payload })
}

#[cfg(test)]
mod tests {
    use super::*;
    use alloy_primitives::{Address, Bloom, Bytes, U256, b256};
    use alloy_rpc_types_engine::{ExecutionPayloadV1, ExecutionPayloadV2, ExecutionPayloadV3};

    fn payload_v1(number: u64) -> ExecutionPayloadV1 {
        ExecutionPayloadV1 {
            parent_hash: b256!("0101010101010101010101010101010101010101010101010101010101010101"),
            fee_recipient: Address::ZERO,
            state_root: B256::ZERO,
            receipts_root: B256::ZERO,
            logs_bloom: Bloom::ZERO,
            prev_randao: B256::ZERO,
            block_number: number,
            gas_limit: 30_000_000,
            gas_used: 21_000,
            timestamp: 1,
            extra_data: Bytes::new(),
            base_fee_per_gas: U256::from(7),
            block_hash: b256!("0202020202020202020202020202020202020202020202020202020202020202"),
            transactions: vec![Bytes::from_static(&[0x01, 0x02, 0x03])],
        }
    }

    fn payload_envelopes() -> [OpExecutionPayloadEnvelope; 4] {
        let v1 = payload_v1(1);
        let v2 = ExecutionPayloadV2 { payload_inner: v1.clone(), withdrawals: Vec::new() };
        let v3 =
            ExecutionPayloadV3 { payload_inner: v2.clone(), blob_gas_used: 0, excess_blob_gas: 0 };
        let root = b256!("0303030303030303030303030303030303030303030303030303030303030303");

        [
            OpExecutionPayloadEnvelope {
                parent_beacon_block_root: None,
                execution_payload: OpExecutionPayload::V1(v1),
            },
            OpExecutionPayloadEnvelope {
                parent_beacon_block_root: None,
                execution_payload: OpExecutionPayload::V2(v2),
            },
            OpExecutionPayloadEnvelope {
                parent_beacon_block_root: Some(root),
                execution_payload: OpExecutionPayload::V3(v3.clone()),
            },
            OpExecutionPayloadEnvelope {
                parent_beacon_block_root: Some(root),
                execution_payload: OpExecutionPayload::V4(OpExecutionPayloadV4 {
                    payload_inner: v3,
                    withdrawals_root: B256::ZERO,
                }),
            },
        ]
    }

    #[test]
    fn payload_versions_roundtrip() {
        for (expected_version, envelope) in payload_envelopes().into_iter().enumerate() {
            let frame = encode_block_frame(&envelope).unwrap();
            assert_eq!(frame[0], expected_version as u8);
            assert_eq!(&frame[1..], envelope.as_ssz_bytes());
            assert_eq!(decode_block_frame(&frame).unwrap(), envelope);
        }
    }

    #[test]
    fn encoder_requires_parent_beacon_block_root() {
        let mut envelope = payload_envelopes()[2].clone();
        envelope.parent_beacon_block_root = None;

        assert_eq!(
            encode_block_frame(&envelope),
            Err(BlocksWireError::MissingParentBeaconBlockRoot(BLOCK_VERSION_V3))
        );
    }

    #[test]
    fn decoder_rejects_invalid_messages() {
        assert_eq!(decode_block_frame(&[]), Err(BlocksWireError::Empty));
        assert_eq!(
            decode_block_frame(&[BLOCK_VERSION_V4 + 1]),
            Err(BlocksWireError::UnsupportedPayloadVersion(BLOCK_VERSION_V4 + 1))
        );
        assert_eq!(decode_block_frame(&[BLOCK_VERSION_V3]), Err(BlocksWireError::Truncated));
        assert!(matches!(
            decode_block_frame(&[BLOCK_VERSION_V1, 0]),
            Err(BlocksWireError::InvalidPayload(_))
        ));
    }
}
