use alloy_primitives::{Address, B64, B256, Bloom, Bytes, U256};
use alloy_rlp::{Decodable, Encodable, Header, RlpDecodable, RlpEncodable};
use alloy_rpc_types_engine::PayloadId;
use alloy_rpc_types_eth::Withdrawal;
use serde::{Deserialize, Serialize};
use serde_json::Value;

/// Represents the modified portions of an execution payload within a flashblock.
/// This structure contains only the fields that can be updated during block construction,
/// such as state root, receipts, logs, and new transactions. Other immutable block fields
/// like parent hash and block number are excluded since they remain constant throughout
/// the block's construction.
#[derive(
    Clone, Debug, PartialEq, Default, Deserialize, Serialize, Eq, RlpEncodable, RlpDecodable,
)]
#[rlp(trailing)]
pub struct ExecutionPayloadFlashblockDeltaV1 {
    /// The state root of the block.
    pub state_root: B256,
    /// The receipts root of the block.
    pub receipts_root: B256,
    /// The logs bloom of the block.
    pub logs_bloom: Bloom,
    /// The gas used of the block.
    #[serde(with = "alloy_serde::quantity")]
    pub gas_used: u64,
    /// The block hash of the block.
    pub block_hash: B256,
    /// The transactions of the block.
    pub transactions: Vec<Bytes>,
    /// Array of [`Withdrawal`] enabled with V2
    pub withdrawals: Vec<Withdrawal>,
    /// The withdrawals root of the block.
    pub withdrawals_root: B256,
    /// The blob gas used
    #[serde(
        default,
        skip_serializing_if = "Option::is_none",
        with = "alloy_serde::quantity::opt"
    )]
    pub blob_gas_used: Option<u64>,
}

/// Represents the base configuration of an execution payload that remains constant
/// throughout block construction. This includes fundamental block properties like
/// parent hash, block number, and other header fields that are determined at
/// block creation and cannot be modified.
#[derive(
    Clone, Debug, PartialEq, Default, Deserialize, Serialize, Eq, RlpEncodable, RlpDecodable,
)]
pub struct ExecutionPayloadBaseV1 {
    /// Ecotone parent beacon block root
    pub parent_beacon_block_root: B256,
    /// The parent hash of the block.
    pub parent_hash: B256,
    /// The fee recipient of the block.
    pub fee_recipient: Address,
    /// The previous randao of the block.
    pub prev_randao: B256,
    /// The block number.
    #[serde(with = "alloy_serde::quantity")]
    pub block_number: u64,
    /// The gas limit of the block.
    #[serde(with = "alloy_serde::quantity")]
    pub gas_limit: u64,
    /// The timestamp of the block.
    #[serde(with = "alloy_serde::quantity")]
    pub timestamp: u64,
    /// The extra data of the block.
    pub extra_data: Bytes,
    /// The base fee per gas of the block.
    pub base_fee_per_gas: U256,
}

#[derive(Clone, Debug, PartialEq, Default, Deserialize, Serialize, Eq)]
pub struct FlashblocksPayloadV1 {
    /// The payload id of the flashblock
    pub payload_id: PayloadId,
    /// The index of the flashblock in the block
    pub index: u64,
    /// The delta/diff containing modified portions of the execution payload
    pub diff: ExecutionPayloadFlashblockDeltaV1,
    /// Additional metadata associated with the flashblock
    pub metadata: Value,
    /// The base execution payload configuration
    #[serde(skip_serializing_if = "Option::is_none")]
    pub base: Option<ExecutionPayloadBaseV1>,
}

/// Manual RLP implementation because `PayloadId` and `serde_json::Value` are
/// outside of alloy-rlp’s blanket impls.
impl Encodable for FlashblocksPayloadV1 {
    fn encode(&self, out: &mut dyn alloy_rlp::BufMut) {
        // ---- compute payload length -------------------------------------------------
        let json_bytes = Bytes::from(
            serde_json::to_vec(&self.metadata).expect("serialising `metadata` to JSON never fails"),
        );

        // encoded-len helper — empty string is one byte (`0x80`)
        let empty_len = 1usize;

        let base_len = self.base.as_ref().map(|b| b.length()).unwrap_or(empty_len);

        let payload_len = self.payload_id.0.length()
            + self.index.length()
            + self.diff.length()
            + json_bytes.length()
            + base_len;

        Header {
            list: true,
            payload_length: payload_len,
        }
        .encode(out);

        // 1. `payload_id` – the inner `B64` already impls `Encodable`
        self.payload_id.0.encode(out);

        // 2. `index`
        self.index.encode(out);

        // 3. `diff`
        self.diff.encode(out);

        // 4. `metadata` (as raw JSON bytes)
        json_bytes.encode(out);

        // 5. `base` (`Option` as “value | empty string”)
        if let Some(base) = &self.base {
            base.encode(out);
        } else {
            // RLP encoding for empty value
            out.put_u8(0x80);
        }
    }

    fn length(&self) -> usize {
        let json_bytes = Bytes::from(
            serde_json::to_vec(&self.metadata).expect("serialising `metadata` to JSON never fails"),
        );

        let empty_len = 1usize;

        let base_len = self.base.as_ref().map(|b| b.length()).unwrap_or(empty_len);

        // list header length + payload length
        let payload_length = self.payload_id.0.length()
            + self.index.length()
            + self.diff.length()
            + json_bytes.length()
            + base_len;

        Header {
            list: true,
            payload_length,
        }
        .length()
            + payload_length
    }
}

impl Decodable for FlashblocksPayloadV1 {
    fn decode(buf: &mut &[u8]) -> Result<Self, alloy_rlp::Error> {
        let header = Header::decode(buf)?;
        if !header.list {
            return Err(alloy_rlp::Error::UnexpectedString);
        }

        // Limit the decoding window to the list payload only.
        let mut body = &buf[..header.payload_length];

        let payload_id = B64::decode(&mut body)?.into();
        let index = u64::decode(&mut body)?;
        let diff = ExecutionPayloadFlashblockDeltaV1::decode(&mut body)?;

        // metadata – stored as raw JSON bytes
        let meta_bytes = Bytes::decode(&mut body)?;
        let metadata: Value = serde_json::from_slice(&meta_bytes)
            .map_err(|_| alloy_rlp::Error::Custom("bad JSON"))?;

        // base (`Option`)
        let base = if body.first() == Some(&0x80) {
            None
        } else {
            Some(ExecutionPayloadBaseV1::decode(&mut body)?)
        };

        // advance the original buffer cursor
        *buf = &buf[header.payload_length..];

        Ok(Self {
            payload_id,
            index,
            diff,
            metadata,
            base,
        })
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use alloy_rlp::{Decodable, encode};

    fn sample_diff() -> ExecutionPayloadFlashblockDeltaV1 {
        ExecutionPayloadFlashblockDeltaV1 {
            state_root: B256::from([1u8; 32]),
            receipts_root: B256::from([2u8; 32]),
            logs_bloom: Bloom::default(),
            gas_used: 21_000,
            block_hash: B256::from([3u8; 32]),
            transactions: vec![Bytes::from(vec![0xde, 0xad, 0xbe, 0xef])],
            withdrawals: vec![Withdrawal::default()],
            withdrawals_root: B256::from([4u8; 32]),
            blob_gas_used: None,
        }
    }

    fn sample_base() -> ExecutionPayloadBaseV1 {
        ExecutionPayloadBaseV1 {
            parent_beacon_block_root: B256::from([5u8; 32]),
            parent_hash: B256::from([6u8; 32]),
            fee_recipient: Address::from([0u8; 20]),
            prev_randao: B256::from([7u8; 32]),
            block_number: 123,
            gas_limit: 30_000_000,
            timestamp: 1_700_000_000,
            extra_data: Bytes::from(b"hello".to_vec()),
            base_fee_per_gas: U256::from(1_000_000_000u64),
        }
    }

    #[test]
    fn roundtrip_without_base() {
        let original = FlashblocksPayloadV1 {
            payload_id: PayloadId::default(),
            index: 0,
            diff: sample_diff(),
            metadata: serde_json::json!({ "key": "value" }),
            base: None,
        };

        let encoded = encode(&original);
        assert_eq!(
            encoded.len(),
            original.length(),
            "length() must match actually-encoded size"
        );

        let mut slice = encoded.as_ref();
        let decoded = FlashblocksPayloadV1::decode(&mut slice).expect("decode succeeds");
        assert_eq!(original, decoded, "round-trip must be loss-less");
        assert!(
            slice.is_empty(),
            "decoder should consume the entire input buffer"
        );
    }

    #[test]
    fn roundtrip_with_base() {
        let original = FlashblocksPayloadV1 {
            payload_id: PayloadId::default(),
            index: 42,
            diff: sample_diff(),
            metadata: serde_json::json!({ "foo": 1, "bar": [1, 2, 3] }),
            base: Some(sample_base()),
        };

        let encoded = encode(&original);
        assert_eq!(encoded.len(), original.length());

        let mut slice = encoded.as_ref();
        let decoded = FlashblocksPayloadV1::decode(&mut slice).expect("decode succeeds");
        assert_eq!(original, decoded);
        assert!(slice.is_empty());
    }

    #[test]
    fn invalid_rlp_is_rejected() {
        let valid = FlashblocksPayloadV1 {
            payload_id: PayloadId::default(),
            index: 1,
            diff: sample_diff(),
            metadata: serde_json::json!({}),
            base: None,
        };

        // Encode, then truncate the last byte to corrupt the payload.
        let mut corrupted = encode(&valid);
        corrupted.pop();

        let mut slice = corrupted.as_ref();
        let result = FlashblocksPayloadV1::decode(&mut slice);
        assert!(
            result.is_err(),
            "decoder must flag malformed / truncated input"
        );
    }
}
