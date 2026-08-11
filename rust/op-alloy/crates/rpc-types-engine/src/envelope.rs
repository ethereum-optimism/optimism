//! Optimism execution payload envelopes and Engine API data types.

#[cfg(any(feature = "serde", test))]
use crate::OpPayloadError;
use crate::{OpExecutionPayload, OpExecutionPayloadSidecar, OpExecutionPayloadV4};
use alloc::vec::Vec;
use alloy_eips::eip7685::Requests;
use alloy_primitives::{B256, keccak256};
use alloy_rpc_types_engine::{
    CancunPayloadFields, ExecutionPayloadInputV2, ExecutionPayloadV1, ExecutionPayloadV2,
    ExecutionPayloadV3, PraguePayloadFields,
};

/// A structurally complete OP execution payload envelope.
///
/// V3 and V4 envelopes always contain a parent beacon block root, while V1 and V2 envelopes
/// cannot contain one. Engine API sidecar fields unsupported by OP are validated when converting
/// from [`OpExecutionData`].
///
/// The transparent SSZ encoding does not include a version discriminator and is therefore not
/// self-describing. Decoding requires an externally supplied payload version.
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum OpExecutionPayloadEnvelope {
    /// A V1 execution payload.
    V1(ExecutionPayloadV1),
    /// A V2 execution payload.
    V2(ExecutionPayloadV2),
    /// A V3 execution payload and its required parent beacon block root.
    V3 {
        /// The execution payload.
        payload: ExecutionPayloadV3,
        /// The parent beacon block root.
        parent_beacon_block_root: B256,
    },
    /// A V4 execution payload and its required parent beacon block root.
    V4 {
        /// The execution payload.
        payload: OpExecutionPayloadV4,
        /// The parent beacon block root.
        parent_beacon_block_root: B256,
    },
}

impl OpExecutionPayloadEnvelope {
    /// Constructs an envelope from its wire-level payload and optional parent beacon block root.
    #[cfg(any(feature = "serde", test))]
    pub(crate) fn try_from_payload(
        payload: OpExecutionPayload,
        parent_beacon_block_root: Option<B256>,
    ) -> Result<Self, OpPayloadError> {
        match (payload, parent_beacon_block_root) {
            (OpExecutionPayload::V1(payload), None) => Ok(Self::V1(payload)),
            (OpExecutionPayload::V2(payload), None) => Ok(Self::V2(payload)),
            (OpExecutionPayload::V3(payload), Some(parent_beacon_block_root)) => {
                Ok(Self::V3 { payload, parent_beacon_block_root })
            }
            (OpExecutionPayload::V4(payload), Some(parent_beacon_block_root)) => {
                Ok(Self::V4 { payload, parent_beacon_block_root })
            }
            (OpExecutionPayload::V3(_) | OpExecutionPayload::V4(_), None) => {
                Err(OpPayloadError::MissingParentBeaconBlockRoot)
            }
            (OpExecutionPayload::V1(_) | OpExecutionPayload::V2(_), Some(_)) => {
                Err(OpPayloadError::UnexpectedParentBeaconBlockRoot)
            }
        }
    }

    /// Consumes the envelope and returns its wire-level payload and parent beacon block root.
    pub fn into_parts(self) -> (OpExecutionPayload, Option<B256>) {
        match self {
            Self::V1(payload) => (OpExecutionPayload::V1(payload), None),
            Self::V2(payload) => (OpExecutionPayload::V2(payload), None),
            Self::V3 { payload, parent_beacon_block_root } => {
                (OpExecutionPayload::V3(payload), Some(parent_beacon_block_root))
            }
            Self::V4 { payload, parent_beacon_block_root } => {
                (OpExecutionPayload::V4(payload), Some(parent_beacon_block_root))
            }
        }
    }

    /// Returns the payload hash over the SSZ-encoded payload envelope data.
    ///
    /// <https://specs.optimism.io/protocol/rollup-node-p2p.html#block-signatures>
    #[cfg(feature = "std")]
    pub fn payload_hash(&self) -> crate::PayloadHash {
        use ssz::Encode;
        crate::PayloadHash::from(self.as_ssz_bytes().as_slice())
    }
}

#[cfg(feature = "serde")]
impl serde::Serialize for OpExecutionPayloadEnvelope {
    fn serialize<S>(&self, serializer: S) -> Result<S::Ok, S::Error>
    where
        S: serde::Serializer,
    {
        use serde::ser::SerializeStruct;

        let mut envelope = serializer.serialize_struct("OpExecutionPayloadEnvelope", 2)?;
        envelope.serialize_field("parentBeaconBlockRoot", &self.parent_beacon_block_root())?;
        match self {
            Self::V1(payload) => envelope.serialize_field("executionPayload", payload)?,
            Self::V2(payload) => envelope.serialize_field("executionPayload", payload)?,
            Self::V3 { payload, .. } => envelope.serialize_field("executionPayload", payload)?,
            Self::V4 { payload, .. } => envelope.serialize_field("executionPayload", payload)?,
        }
        envelope.end()
    }
}

#[cfg(feature = "serde")]
impl<'de> serde::Deserialize<'de> for OpExecutionPayloadEnvelope {
    fn deserialize<D>(deserializer: D) -> Result<Self, D::Error>
    where
        D: serde::Deserializer<'de>,
    {
        #[derive(serde::Deserialize)]
        #[serde(rename_all = "camelCase")]
        struct WireEnvelope {
            parent_beacon_block_root: Option<B256>,
            execution_payload: OpExecutionPayload,
        }

        let envelope = WireEnvelope::deserialize(deserializer)?;
        Self::try_from_payload(envelope.execution_payload, envelope.parent_beacon_block_root)
            .map_err(serde::de::Error::custom)
    }
}

#[cfg(feature = "std")]
impl ssz::Encode for OpExecutionPayloadEnvelope {
    fn is_ssz_fixed_len() -> bool {
        false
    }

    fn ssz_append(&self, buf: &mut Vec<u8>) {
        match self {
            Self::V1(payload) => payload.ssz_append(buf),
            Self::V2(payload) => payload.ssz_append(buf),
            Self::V3 { payload, parent_beacon_block_root } => {
                buf.extend_from_slice(parent_beacon_block_root.as_slice());
                payload.ssz_append(buf);
            }
            Self::V4 { payload, parent_beacon_block_root } => {
                buf.extend_from_slice(parent_beacon_block_root.as_slice());
                payload.ssz_append(buf);
            }
        }
    }

    fn ssz_bytes_len(&self) -> usize {
        match self {
            Self::V1(payload) => payload.ssz_bytes_len(),
            Self::V2(payload) => payload.ssz_bytes_len(),
            Self::V3 { payload, .. } => B256::len_bytes() + payload.ssz_bytes_len(),
            Self::V4 { payload, .. } => B256::len_bytes() + payload.ssz_bytes_len(),
        }
    }
}

/// Struct aggregating [`OpExecutionPayload`] and [`OpExecutionPayloadSidecar`] and encapsulating
/// complete payload supplied for execution.
#[derive(Debug, Clone)]
#[cfg_attr(feature = "serde", derive(serde::Serialize, serde::Deserialize))]
pub struct OpExecutionData {
    /// Execution payload.
    pub payload: OpExecutionPayload,
    /// Additional fork-specific fields.
    pub sidecar: OpExecutionPayloadSidecar,
}

impl OpExecutionData {
    /// Creates new instance of [`OpExecutionData`].
    pub const fn new(payload: OpExecutionPayload, sidecar: OpExecutionPayloadSidecar) -> Self {
        Self { payload, sidecar }
    }

    /// Creates a new instance from args to engine API method `newPayloadV2`.
    ///
    /// Spec: <https://specs.optimism.io/protocol/exec-engine.html#engine_newpayloadv2>
    pub fn v2(payload: ExecutionPayloadInputV2) -> Self {
        Self::new(OpExecutionPayload::v2(payload), OpExecutionPayloadSidecar::default())
    }

    /// Creates a new instance from args to engine API method `newPayloadV3`.
    ///
    /// Spec: <https://specs.optimism.io/protocol/exec-engine.html#engine_newpayloadv3>
    pub fn v3(
        payload: ExecutionPayloadV3,
        versioned_hashes: Vec<B256>,
        parent_beacon_block_root: B256,
    ) -> Self {
        Self::new(
            OpExecutionPayload::v3(payload),
            OpExecutionPayloadSidecar::v3(CancunPayloadFields::new(
                parent_beacon_block_root,
                versioned_hashes,
            )),
        )
    }

    /// Creates a new instance from args to engine API method `newPayloadV4`.
    ///
    /// Spec: <https://specs.optimism.io/protocol/exec-engine.html#engine_newpayloadv4>
    pub fn v4(
        payload: OpExecutionPayloadV4,
        versioned_hashes: Vec<B256>,
        parent_beacon_block_root: B256,
        execution_requests: Requests,
    ) -> Self {
        Self::new(
            OpExecutionPayload::v4(payload),
            OpExecutionPayloadSidecar::v4(
                CancunPayloadFields::new(parent_beacon_block_root, versioned_hashes),
                PraguePayloadFields::new(execution_requests),
            ),
        )
    }
}

/// Represents the Keccak-256 hash of an encoded execution payload envelope.
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
#[cfg_attr(feature = "arbitrary", derive(arbitrary::Arbitrary))]
pub struct PayloadHash(pub B256);

impl From<&[u8]> for PayloadHash {
    /// Returns the Keccak256 hash of a sequence of bytes
    fn from(value: &[u8]) -> Self {
        Self(keccak256(value))
    }
}

impl PayloadHash {
    /// The expected message that should be signed by the unsafe block signer.
    pub fn signature_message(&self, chain_id: u64) -> B256 {
        let domain = B256::ZERO.as_slice();
        let chain_id = B256::left_padding_from(&chain_id.to_be_bytes()[..]);
        let payload_hash = self.0.as_slice();
        keccak256([domain, chain_id.as_slice(), payload_hash].concat())
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::{OpFlashblockError, OpFlashblockPayload, OpPayloadError};
    use alloy_consensus::Block;
    use alloy_primitives::{Address, Bloom, Bytes, U256, b256};
    use alloy_rpc_types_engine::{ExecutionPayloadV1, ExecutionPayloadV2, PayloadError};

    #[test]
    #[cfg(feature = "serde")]
    fn test_serde_roundtrip_op_execution_payload_envelope() {
        let envelope_str = r#"{
            "executionPayload": {"parentHash":"0xe927a1448525fb5d32cb50ee1408461a945ba6c39bd5cf5621407d500ecc8de9","feeRecipient":"0x0000000000000000000000000000000000000000","stateRoot":"0x10f8a0830000e8edef6d00cc727ff833f064b1950afd591ae41357f97e543119","receiptsRoot":"0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421","logsBloom":"0x00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000","prevRandao":"0xe0d8b4521a7da1582a713244ffb6a86aa1726932087386e2dc7973f43fc6cb24","blockNumber":"0x1","gasLimit":"0x2ffbd2","gasUsed":"0x0","timestamp":"0x1235","extraData":"0xd883010d00846765746888676f312e32312e30856c696e7578","baseFeePerGas":"0x342770c0","blockHash":"0x44d0fa5f2f73a938ebb96a2a21679eb8dea3e7b7dd8fd9f35aa756dda8bf0a8a","transactions":[],"withdrawals":[],"blobGasUsed":"0x0","excessBlobGas":"0x0","withdrawalsRoot":"0x10f8a0830000e8edef6d00cc727ff833f064b1950afd591ae41357f97e543119"},
            "parentBeaconBlockRoot": "0x9999999999999999999999999999999999999999999999999999999999999999"
        }"#;

        let envelope: OpExecutionPayloadEnvelope = serde_json::from_str(envelope_str).unwrap();
        let expected = b256!("9999999999999999999999999999999999999999999999999999999999999999");
        assert_eq!(envelope.parent_beacon_block_root().unwrap(), expected);
        assert_eq!(
            serde_json::to_value(&envelope).unwrap(),
            serde_json::from_str::<serde_json::Value>(envelope_str).unwrap()
        );

        let mut missing_root = serde_json::from_str::<serde_json::Value>(envelope_str).unwrap();
        missing_root.as_object_mut().unwrap().remove("parentBeaconBlockRoot");
        let err = serde_json::from_value::<OpExecutionPayloadEnvelope>(missing_root).unwrap_err();
        assert!(err.to_string().contains("missing parent beacon block root"));

        let mut unexpected_root = serde_json::from_str::<serde_json::Value>(envelope_str).unwrap();
        let payload = unexpected_root["executionPayload"].as_object_mut().unwrap();
        payload.remove("blobGasUsed");
        payload.remove("excessBlobGas");
        payload.remove("withdrawalsRoot");
        let err =
            serde_json::from_value::<OpExecutionPayloadEnvelope>(unexpected_root).unwrap_err();
        assert!(err.to_string().contains("unexpected parent beacon block root"));
    }

    fn execution_payload_v1(block_hash: B256) -> ExecutionPayloadV1 {
        ExecutionPayloadV1 {
            parent_hash: b256!("1111111111111111111111111111111111111111111111111111111111111111"),
            fee_recipient: Address::repeat_byte(0x22),
            state_root: b256!("3333333333333333333333333333333333333333333333333333333333333333"),
            receipts_root: b256!(
                "4444444444444444444444444444444444444444444444444444444444444444"
            ),
            logs_bloom: Bloom::ZERO,
            prev_randao: b256!("5555555555555555555555555555555555555555555555555555555555555555"),
            block_number: 1,
            gas_limit: 30_000_000,
            gas_used: 21_000,
            timestamp: 123_456_789,
            extra_data: Bytes::from_static(&[0x12, 0x34]),
            base_fee_per_gas: U256::from(1_000_000_000),
            block_hash,
            transactions: Vec::new(),
        }
    }

    #[test]
    #[cfg(feature = "serde")]
    fn test_serde_versioned_root_invariants() {
        let v1 = OpExecutionPayloadEnvelope::V1(execution_payload_v1(B256::ZERO));
        let v1_json = serde_json::to_value(&v1).unwrap();
        assert!(v1_json["parentBeaconBlockRoot"].is_null());
        assert_eq!(serde_json::from_value::<OpExecutionPayloadEnvelope>(v1_json).unwrap(), v1);

        let v2 = OpExecutionPayloadEnvelope::V2(ExecutionPayloadV2 {
            payload_inner: execution_payload_v1(B256::ZERO),
            withdrawals: Vec::new(),
        });
        let v2_json = serde_json::to_value(&v2).unwrap();
        assert!(v2_json["parentBeaconBlockRoot"].is_null());
        assert_eq!(serde_json::from_value::<OpExecutionPayloadEnvelope>(v2_json).unwrap(), v2);

        let v3 = OpExecutionPayloadEnvelope::V3 {
            payload: ExecutionPayloadV3 {
                payload_inner: ExecutionPayloadV2 {
                    payload_inner: execution_payload_v1(B256::ZERO),
                    withdrawals: Vec::new(),
                },
                blob_gas_used: 0,
                excess_blob_gas: 0,
            },
            parent_beacon_block_root: B256::ZERO,
        };
        let v3_json = serde_json::to_value(&v3).unwrap();
        assert_eq!(
            v3_json["parentBeaconBlockRoot"],
            serde_json::Value::String(B256::ZERO.to_string())
        );
        assert_eq!(serde_json::from_value::<OpExecutionPayloadEnvelope>(v3_json).unwrap(), v3);
    }

    #[test]
    fn test_try_into_checked_block() {
        // Pin hashes produced by op-service's independent ExecutionPayloadEnvelope.CheckBlockHash.
        // The V3 case uses a present zero beacon root to verify that presence is preserved.
        let v1_hash = b256!("55e9427d0f33b1e09e76c229aa4d685336ebc151a26820914652439a69e46791");
        let v2_hash = b256!("de05d55fdc91f15fc85cbf222ab05a586a3b7b6afd1a7abb7f353d3522dfff15");
        let v3_hash = b256!("f27a826220863770f4c8f5599c19a8dc82b8b2bbbc1bc689cb1b0dab6418886e");
        let v4_hash = b256!("4034384737b7667a4015c0083ba2262ca3d196cc6fa6c3e763ee0a69ff95e780");

        let cases = [
            OpExecutionPayloadEnvelope::V1(execution_payload_v1(v1_hash)),
            OpExecutionPayloadEnvelope::V2(ExecutionPayloadV2 {
                payload_inner: execution_payload_v1(v2_hash),
                withdrawals: Vec::new(),
            }),
            OpExecutionPayloadEnvelope::V3 {
                // Presence is required for V3; an explicitly zero root remains distinct from None.
                parent_beacon_block_root: B256::ZERO,
                payload: ExecutionPayloadV3 {
                    payload_inner: ExecutionPayloadV2 {
                        payload_inner: execution_payload_v1(v3_hash),
                        withdrawals: Vec::new(),
                    },
                    blob_gas_used: 0,
                    excess_blob_gas: 0,
                },
            },
            OpExecutionPayloadEnvelope::V4 {
                parent_beacon_block_root: b256!(
                    "7777777777777777777777777777777777777777777777777777777777777777"
                ),
                payload: OpExecutionPayloadV4 {
                    payload_inner: ExecutionPayloadV3 {
                        payload_inner: ExecutionPayloadV2 {
                            payload_inner: execution_payload_v1(v4_hash),
                            withdrawals: Vec::new(),
                        },
                        blob_gas_used: 0,
                        excess_blob_gas: 0,
                    },
                    withdrawals_root: b256!(
                        "6666666666666666666666666666666666666666666666666666666666666666"
                    ),
                },
            },
        ];

        for mut envelope in cases {
            let claimed = envelope.block_hash();
            assert!(envelope.check_block_hash().is_ok());
            let block: Block<op_alloy_consensus::OpTxEnvelope> =
                envelope.clone().try_into_checked_block().unwrap();
            assert_eq!(block.header.hash_slow(), claimed);

            envelope.as_v1_mut().state_root = B256::ZERO;
            assert!(matches!(
                envelope.check_block_hash(),
                Err(OpPayloadError::Eth(PayloadError::BlockHash {
                    execution,
                    consensus,
                })) if execution != claimed && consensus == claimed
            ));
            assert!(matches!(
                envelope.try_into_checked_block::<op_alloy_consensus::OpTxEnvelope>(),
                Err(OpPayloadError::Eth(PayloadError::BlockHash {
                    execution,
                    consensus,
                })) if execution != claimed && consensus == claimed
            ));
        }
    }

    #[test]
    fn test_signature_message() {
        let inner = b256!("9999999999999999999999999999999999999999999999999999999999999999");
        let hash = PayloadHash::from(inner.as_slice());
        let chain_id = 10;
        let expected = b256!("44a0e2b1aba1aae1771eddae1dcd2ad18a8cdac8891517153f03253e49d3f206");
        assert_eq!(hash.signature_message(chain_id), expected);
    }

    #[test]
    fn test_inner_payload_hash() {
        arbtest::arbtest(|u| {
            let inner = B256::from(u.arbitrary::<[u8; 32]>()?);
            let hash = PayloadHash::from(inner.as_slice());
            assert_eq!(hash.0, keccak256(inner.as_slice()));
            Ok(())
        });
    }

    #[test]
    #[cfg(feature = "std")]
    fn test_ssz_envelope_encoding_requires_external_version() {
        use ssz::{Decode, Encode};

        let mut payload_v1 = execution_payload_v1(B256::ZERO);
        payload_v1.extra_data = Bytes::new();

        // With the 32-byte root prefix, the V1 decoder reads these shifted V3 fields as its
        // extra-data and transactions offsets.
        payload_v1.block_number = payload_v1.ssz_bytes_len() as u64;
        let mut payload_v3 = ExecutionPayloadV3 {
            payload_inner: ExecutionPayloadV2 {
                payload_inner: payload_v1,
                withdrawals: Vec::new(),
            },
            blob_gas_used: 0,
            excess_blob_gas: 0,
        };
        let encoded_len = B256::len_bytes() + payload_v3.ssz_bytes_len();
        let mut block_hash = [0; B256::len_bytes()];
        block_hash[..4].copy_from_slice(&(encoded_len as u32).to_le_bytes());
        payload_v3.payload_inner.payload_inner.block_hash = B256::from(block_hash);
        let envelope = OpExecutionPayloadEnvelope::V3 {
            payload: payload_v3.clone(),
            parent_beacon_block_root: B256::ZERO,
        };

        let encoded = envelope.as_ssz_bytes();
        assert_eq!(encoded.len(), encoded_len);

        // The complete root-prefixed V3 encoding is also structurally valid as a V1 payload.
        assert!(ExecutionPayloadV1::from_ssz_bytes(&encoded).is_ok());
        assert_eq!(
            ExecutionPayloadV3::from_ssz_bytes(&encoded[B256::len_bytes()..]).unwrap(),
            payload_v3
        );
    }

    #[cfg(test)]
    fn create_test_flashblock(
        index: u64,
        with_base: bool,
        transactions: Vec<Bytes>,
        post_exec_tx: Option<Bytes>,
    ) -> OpFlashblockPayload {
        use crate::flashblock::{
            OpFlashblockPayloadBase, OpFlashblockPayloadDelta, OpFlashblockPayloadMetadata,
        };
        use alloc::collections::BTreeMap;
        use alloy_primitives::{Address, Bloom, U256};
        use alloy_rpc_types_engine::PayloadId;

        let base = with_base.then(|| OpFlashblockPayloadBase {
            parent_beacon_block_root: B256::ZERO,
            parent_hash: B256::ZERO,
            fee_recipient: Address::ZERO,
            prev_randao: B256::ZERO,
            block_number: 100,
            gas_limit: 30_000_000,
            timestamp: 1234567890,
            extra_data: Bytes::default(),
            base_fee_per_gas: U256::from(1000000000u64),
        });

        let diff = OpFlashblockPayloadDelta {
            state_root: B256::ZERO,
            receipts_root: B256::ZERO,
            logs_bloom: Bloom::ZERO,
            gas_used: 21000,
            block_hash: B256::ZERO,
            transactions,
            withdrawals: Vec::new(),
            withdrawals_root: B256::from([1u8; 32]), // Non-zero for Isthmus
            blob_gas_used: Some(0),
            post_exec_tx,
        };

        let metadata = OpFlashblockPayloadMetadata {
            block_number: 100,
            new_account_balances: BTreeMap::new(),
            receipts: BTreeMap::new(),
        };

        OpFlashblockPayload { payload_id: PayloadId::new([1u8; 8]), index, base, diff, metadata }
    }

    #[test]
    fn test_from_flashblocks_empty_vec() {
        let result = OpExecutionPayloadEnvelope::from_flashblocks(&[]);
        assert!(matches!(result, Err(OpFlashblockError::MissingPayload)));
    }

    #[test]
    fn test_from_flashblocks_non_sequential_indices() {
        let fb1 = create_test_flashblock(0, true, Vec::new(), None);
        let fb2 = create_test_flashblock(2, false, Vec::new(), None); // Skip index 1

        let result = OpExecutionPayloadEnvelope::from_flashblocks(&[fb1, fb2]);
        assert!(matches!(result, Err(OpFlashblockError::InvalidIndex)));
    }

    #[test]
    fn test_from_flashblocks_missing_base_in_first() {
        let fb1 = create_test_flashblock(0, false, Vec::new(), None); // First should have base

        let result = OpExecutionPayloadEnvelope::from_flashblocks(&[fb1]);
        assert!(matches!(result, Err(OpFlashblockError::MissingBasePayload)));
    }

    #[test]
    fn test_from_flashblocks_unexpected_base_in_second() {
        let fb1 = create_test_flashblock(0, true, Vec::new(), None);
        let fb2 = create_test_flashblock(1, true, Vec::new(), None); // Should not have base

        let result = OpExecutionPayloadEnvelope::from_flashblocks(&[fb1, fb2]);
        assert!(matches!(result, Err(OpFlashblockError::UnexpectedBasePayload)));
    }

    #[test]
    fn test_from_flashblocks_single_valid_flashblock() {
        let fb1 = create_test_flashblock(0, true, Vec::new(), None);

        let result = OpExecutionPayloadEnvelope::from_flashblocks(&[fb1]);
        assert!(result.is_ok(), "Single valid flashblock should succeed");
    }

    #[test]
    fn test_from_flashblocks_multiple_valid_flashblocks() {
        let fb1 = create_test_flashblock(0, true, Vec::new(), None);
        let fb2 = create_test_flashblock(1, false, Vec::new(), None);
        let fb3 = create_test_flashblock(2, false, Vec::new(), None);

        let result = OpExecutionPayloadEnvelope::from_flashblocks(&[fb1, fb2, fb3]);
        assert!(result.is_ok(), "Multiple valid flashblocks should succeed");
    }

    #[test]
    fn test_from_flashblocks_wrong_first_index() {
        let fb1 = create_test_flashblock(1, true, Vec::new(), None); // Should be index 0
        let result = OpExecutionPayloadEnvelope::from_flashblocks(&[fb1]);
        assert!(matches!(result, Err(OpFlashblockError::InvalidIndex)));
    }

    #[test]
    fn test_from_subblocks_appends_latest_post_exec_tx() {
        let older = Bytes::from(vec![0x7d, 0x01]);
        let latest = Bytes::from(vec![0x7d, 0x02]);

        let none = OpExecutionPayloadEnvelope::from_flashblocks(&[
            create_test_flashblock(0, true, vec![Bytes::from(vec![0x01])], None),
            create_test_flashblock(1, false, vec![Bytes::from(vec![0x02])], None),
        ])
        .unwrap();
        assert_eq!(none.transactions().len(), 2, "no 0x7D should be appended");

        let materialized = OpExecutionPayloadEnvelope::from_flashblocks(&[
            create_test_flashblock(0, true, vec![Bytes::from(vec![0x01])], Some(older.clone())),
            create_test_flashblock(1, false, vec![Bytes::from(vec![0x02])], Some(latest.clone())),
            create_test_flashblock(2, false, vec![Bytes::from(vec![0x03])], Some(latest.clone())),
        ])
        .unwrap();
        let txs = materialized.transactions();
        assert_eq!(txs.len(), 4, "3 user txs + exactly one trailing 0x7D");
        assert_eq!(txs.last(), Some(&latest), "the latest post_exec_tx must be the trailing tx");
        assert!(!txs.iter().any(|t| t == &older), "the superseded post_exec_tx must not appear");
    }

    #[test]
    fn test_from_subblocks_preserves_empty_post_exec_tx() {
        let empty = Bytes::new();
        let materialized = OpExecutionPayloadEnvelope::from_flashblocks(&[create_test_flashblock(
            0,
            true,
            Vec::new(),
            Some(empty.clone()),
        )])
        .unwrap();

        assert_eq!(materialized.transactions().last(), Some(&empty));
    }

    // Real-world test case from Unichain Sepolia
    // <https://unichain-sepolia.blockscout.com/block/35535698>
    #[test]
    #[cfg(feature = "serde")]
    fn test_from_flashblocks_unichain_sepolia_block() {
        use alloy_primitives::{address, b256};

        let raw_sequence = r#"[{"payload_id":"0x03c446f063e3735a","index":0,"base":{"parent_beacon_block_root":"0xf6d335a6b2b4fd8fb539cd51a49769df4d53c31a90c54dd270e54542638ff101","parent_hash":"0x06ff95a9cd23b0328da74a984aa986b2e01d377dab1825f1029e39ece6c4a3ea","fee_recipient":"0x4200000000000000000000000000000000000011","prev_randao":"0x8beee738d20a9d77c5f27e9cb799ebe5b536f0985efad5f7d77ebff47f092c4a","block_number":"0x21e3b52","gas_limit":"0x3938700","timestamp":"0x690be89e","extra_data":"0x00000000320000000c","base_fee_per_gas":"0x33"},"diff":{"state_root":"0xb29a9bcae8cf3ae6d68985fcd70db80b3818cd629c9d5da0bb116451739b2078","receipts_root":"0x91d8ad10740ccfc1bd848fba0e02668d95769c08eeea30f10698692ba86c6159","logs_bloom":"0x00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000","gas_used":"0x10994","block_hash":"0xa66f8562a861f906a2438d7d6ba79495640d98d9c6922b9605c54b57f97a345c","transactions":["0x7ef90104a035dd2ec802504a143048c7830f8f570e0d6cf5147217af869939c6b4ba710a3694deaddeaddeaddeaddeaddeaddeaddeaddead00019442000000000000000000000000000000000000158080830f424080b8b0098999be000007d0000dbba0000000000000000800000000690be848000000000092042e000000000000000000000000000000000000000000000000000000000000000900000000000000000000000000000000000000000000000000000000000000010ffd7e2fb2c36e5f27c015872ce733a7b4f3fc0f4ee668d7469c557c48f8250f0000000000000000000000004ab3387810ef500bfe05a49dc53a44c222cbab3e000000000000000000000000","0x02f87e8205158401c8ea9180338255789400000000000000000000000000000000000000008096426c6f636b204e756d6265723a203335353335363938c080a091f83058c881d9ad71c179ce680326501702eb68150d20b2bf7786e388f954a2a0180185d83e503f11bf3c265c1f9296ed8d3d7c04031cd8bb30509ad188ce7bbc"],"withdrawals":[],"withdrawals_root":"0x62ed62e0391b081bf172f287fbbe75e87d8a6c22f1d3b1f1aef4788c134633d2"},"metadata":{"block_number":35535698,"new_account_balances":{},"receipts":{}}},{"payload_id":"0x03c446f063e3735a","index":1,"base":null,"diff":{"state_root":"0xfb1794f74d405b345672c57a5053c6105cc55c8e63f96fb0db5b0260df42413a","receipts_root":"0x1eaaaeb9d43bead7d32b90f1b320589174c63d2fa8f5fd366f841a205b1eb2e0","logs_bloom":"0x00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000002000000001000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000040000000000000000000000000000000000000000000000000004040000000000000000000000000000000000000000000000000000000000000000000000","gas_used":"0x18f7d","block_hash":"0x67b0521ebfcb03d6ce2b6e1bad9c9c66795365f63ad8dc51e1e8f582a5ab7821","transactions":["0x02f86c8205158401c8ea92803382880994f878f0340bf132c28f3211e8b46c569edf81749580843fd553e8c001a0d73ce313aafea312e0b7244767e45f8b05d50305e0f4e4c3c564ddc751666815a02ee015ce2363311823c0b2e96bfb0e8090fd53c6cdd99be8cf343af123036dfc"],"withdrawals":[],"withdrawals_root":"0x62ed62e0391b081bf172f287fbbe75e87d8a6c22f1d3b1f1aef4788c134633d2"},"metadata":{"block_number":35535698,"new_account_balances":{},"receipts":{}}},{"payload_id":"0x03c446f063e3735a","index":2,"base":null,"diff":{"state_root":"0x90dd105c4a2a0dd9ffe994204bfa3e2b4f70f7ea760d5cb9a4263f26a89f91b4","receipts_root":"0x0fff0488aa3732c34018b938839ab2f0caa96018221e4ffaeca011fb06ba288f","logs_bloom":"0x00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000002000000001000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000040000000000000000000000000000000000000000000000000004040000000000000000000000000000000000000000000000000000000000000000000000","gas_used":"0x21566","block_hash":"0x720feb7457110a565b479fafbaa89cc984f5d673846a27d44bbb8cf5200b32fe","transactions":["0x02f86c8205158401c8ea93803382880994f878f0340bf132c28f3211e8b46c569edf81749580843fd553e8c001a0f8cd94080642e116bc772f36a02d002505227aa542e1c13e5129ab40b8b037fba00608318d3895388e39b218bcb275380cebc566e68f26d3d434e32b8b58366cdf"],"withdrawals":[],"withdrawals_root":"0x62ed62e0391b081bf172f287fbbe75e87d8a6c22f1d3b1f1aef4788c134633d2"},"metadata":{"block_number":35535698,"new_account_balances":{},"receipts":{}}},{"payload_id":"0x03c446f063e3735a","index":3,"base":null,"diff":{"state_root":"0x71f8c60fdfdd84cffda3b0b6af7c8ff92195918f4fc2abae750a7306521ac0dc","receipts_root":"0xa62d1d98f56ffb1464a2beb185484253df68208004306e155c0bd1519137afe6","logs_bloom":"0x00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000002000000001000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000040000000000000000000000000000000000000000000000000004040000000000000000000000000000000000000000000000000000000000000000000000","gas_used":"0x29b4f","block_hash":"0x670844e30f7325d4f290ea375e01f7e819afca317fc7db9723e6867a184984fa","transactions":["0x02f86c8205158401c8ea94803382880994f878f0340bf132c28f3211e8b46c569edf81749580843fd553e8c080a04368492ec1d087703aaf6f5fefe4427b3bf382e5cd07133f638bb6701f15fe61a05e28757fbdc7e744118be36d5a1548eb7c009eefcb5dc5c5040e09c2fc6de9d8"],"withdrawals":[],"withdrawals_root":"0x62ed62e0391b081bf172f287fbbe75e87d8a6c22f1d3b1f1aef4788c134633d2"},"metadata":{"block_number":35535698,"new_account_balances":{},"receipts":{}}},{"payload_id":"0x03c446f063e3735a","index":4,"base":null,"diff":{"state_root":"0x5615e4342d231c352438f0ba6a8f0f641459f67961961764b781a909969b28ad","receipts_root":"0x588e1d47b0618d7e935b20c3945cba3b7b8c00141904f79ceed20312ea502e63","logs_bloom":"0x00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000002000000001000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000040000000000000000000000000000000000000000000000000004040000000000000000000000000000000000000000000000000000000000000000000000","gas_used":"0x32138","block_hash":"0xc463a3120c35268f610d969f5608b479332ef10953af77c7a6be806195831196","transactions":["0x02f86c8205158401c8ea95803382880994f878f0340bf132c28f3211e8b46c569edf81749580843fd553e8c080a0802ba6d4f37e3b8de96095bd0b216144f276171d16dc62a004f1a89009af5deea00f0c6250cfd1a062a1bc2bc353a5c227a980cac0f233b7be8932f2192342ec4f"],"withdrawals":[],"withdrawals_root":"0x62ed62e0391b081bf172f287fbbe75e87d8a6c22f1d3b1f1aef4788c134633d2"},"metadata":{"block_number":35535698,"new_account_balances":{},"receipts":{}}}]"#;

        let flashblocks: Vec<OpFlashblockPayload> = serde_json::from_str(raw_sequence).unwrap();
        let execution_data = OpExecutionPayloadEnvelope::from_flashblocks(&flashblocks).unwrap();

        // Validate against expected final block state
        assert_eq!(
            execution_data.parent_hash(),
            b256!("06ff95a9cd23b0328da74a984aa986b2e01d377dab1825f1029e39ece6c4a3ea")
        );
        assert_eq!(
            execution_data.block_hash(),
            b256!("c463a3120c35268f610d969f5608b479332ef10953af77c7a6be806195831196")
        );
        assert_eq!(execution_data.block_number(), 0x21E3B52);
        assert_eq!(execution_data.timestamp(), 0x690be89e);
        assert_eq!(
            execution_data.fee_recipient(),
            address!("4200000000000000000000000000000000000011")
        );
        assert_eq!(execution_data.gas_limit(), 0x3938700);
        assert_eq!(execution_data.as_v1().gas_used, 0x32138);
        assert_eq!(
            execution_data.as_v1().state_root,
            b256!("5615e4342d231c352438f0ba6a8f0f641459f67961961764b781a909969b28ad")
        );
        assert_eq!(
            execution_data.as_v1().receipts_root,
            b256!("588e1d47b0618d7e935b20c3945cba3b7b8c00141904f79ceed20312ea502e63")
        );
        assert_eq!(execution_data.transactions().len(), 6);
        assert_eq!(
            execution_data.as_v4().unwrap().withdrawals_root,
            b256!("62ed62e0391b081bf172f287fbbe75e87d8a6c22f1d3b1f1aef4788c134633d2")
        );

        // Verify parent beacon block root
        assert_eq!(
            execution_data.parent_beacon_block_root(),
            Some(b256!("f6d335a6b2b4fd8fb539cd51a49769df4d53c31a90c54dd270e54542638ff101"))
        );
    }

    // Real-world test case from Base Sepolia
    // Block #33439826 with 11 flashblocks (indices 0-10)
    #[test]
    #[cfg(feature = "serde")]
    fn test_from_flashblocks_base_sepolia_block() {
        use alloy_primitives::{address, b256};

        let raw_sequence = r#"[{"payload_id":"0x03c33cc62b81edb6","index":0,"base":{"parent_beacon_block_root":"0xf058b1e43890ed5f838bd07e77db06d075d894343d1b31f6099a345b0d8f7d1b","parent_hash":"0x6ffd2714d5af6c412c57db3f664a5a127516573bbd987fd242d06f71ea662741","fee_recipient":"0x4200000000000000000000000000000000000011","prev_randao":"0x9985c1f8ec25b468cbf2b727a8371b4554b7e7adb059c08abf7a7d51d86ceee5","block_number":"0x1fe4052","gas_limit":"0x3938700","timestamp":"0x690fdf84","extra_data":"0x000000003200000004","base_fee_per_gas":"0x34"},"diff":{"state_root":"0x0000000000000000000000000000000000000000000000000000000000000000","receipts_root":"0x1b2fa5e4cbbc1f8c01a7c7204571ebe339dbdfadc666451d8e70d5c10c99830f","logs_bloom":"0x00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000","gas_used":"0xb41c","block_hash":"0x87c6775cc427caf4c0ffe0d4b6d76627536f38d77d23f105f9f104ef3e5541c7","transactions":["0x7ef90104a01c055ffd19ea027da4a8aae0a2734c6bf17c3f487d4cc22931d7dbe261409cda94deaddeaddeaddeaddeaddeaddeaddeaddead00019442000000000000000000000000000000000000158080830f424080b8b0098999be0000044d000a118b000000000000000400000000690fde3c00000000009252e3000000000000000000000000000000000000000000000000000000000000000a00000000000000000000000000000000000000000000000000000000000000014f1595c3798e3082aa093e433bd5cbd102a11f9619d20e6e821c1a30fb56b12b000000000000000000000000fc56e7272eebbba5bc6c544e159483c4a38f8ba3000000000000000000000000"],"withdrawals":[],"withdrawals_root":"0x77b0fb1616a212bd7cf33d7c28651f19bf6093b2c5f1967e674ec861aeaf9d44"},"metadata":{"block_number":33439826,"new_account_balances":{},"receipts":{}}},{"payload_id":"0x03c33cc62b81edb6","index":1,"diff":{"state_root":"0x0000000000000000000000000000000000000000000000000000000000000000","receipts_root":"0xe38b2090ddfa6ee25b15a8ebcdd7ecc0f1ee9128ec98cb24f47909e29e11832e","logs_bloom":"0x00000000000000000000000020000000040080000000000000020005000000004000000040040000000080000000000000000000000000000002000000000000008000000000000000000000000000014000000000800000000000000000000000000000000000040100000000000000000000000100000000000380008a02000000100000400200000100800000000000000000000004001000200000000000000000000800020000000000400000000000000000008000400801080000000000005000000400000000000000000000000110000000000000000000000000100200021004400010000000010000000400000008002000004080000000000000","gas_used":"0x9d2f2","block_hash":"0x4548d5014de4883cec380838f1b225996fa3c08c176f2f63d98d8c23169fab44","transactions":["0x02f89283014a348202ea830f4275830f427583045dd594a449bc031fa0b815ca14fafd0c5edb75ccd9c80f80a4de0e9a3e000000000000000000000000000000000000000000000001236efcbcbb340000c001a0742ff606597cda39751dd369e66e9978946ce8f4eb578a8d73314535a2df4388a06a6f83c3606c32e1677f62408b8ec69b09a82f499395b26eaefea567deb83843","0x02f9101583014a34830597bd830f4240830f42aa8306aecc9442826e92e6418877459f0920cb058e462ac6a0a480b90fa4dbaa1e6400000000000000000000000000a739e4479c97289801654ec1a52a67077613c000000000000000000000000000000000000000000000000000000000000000a000000000000000000000000000000000000000000000000000000000691d0e7f4f6ae70adc2708ec4857d3d5ca54a11710c9ac11989b1cb3d3d8d3298a78f6a50000000000000000000000000000000000000000000000000000000000000f200000000000000000000000000000000000000000000000000000000000000e44b653f0c300000000000000000000000000000000000000000000000000000000000000600000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000033bea00000000000000000000000000000000000000000000000000000000000000380000000000000000000000000000000000000000000000000000000000000002000000000000000000000000000000000000000000000000000000000000000030000000000000000000000000000000000000000000000000000000000000060000000000000000000000000000000000000000000000000000000000000014000000000000000000000000000000000000000000000000000000000000002200000000000000000000000000000000000000000000000000000000000000002000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000000000000000000000000000000004747970650000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000026f6b00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000040000000000000000000000000000000000000000000000000000000000000008000000000000000000000000000000000000000000000000000000000000000086f6b2e746f6b656e0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000003657468000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000040000000000000000000000000000000000000000000000000000000000000008000000000000000000000000000000000000000000000000000000000000000086f6b2e74785f69640000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000046626173653a3078343865643835396232636630633962366261633864373134653162363436313264313232346436643a38343533323a33333433393832323a3333393131333600000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000a20000000000000000000000000000000000000000000000000000000000000002000000000000000000000000000000000000000000000000000000000000000090000000000000000000000000000000000000000000000000000000000000120000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000002e000000000000000000000000000000000000000000000000000000000000003e000000000000000000000000000000000000000000000000000000000000004c000000000000000000000000000000000000000000000000000000000000005a000000000000000000000000000000000000000000000000000000000000006a000000000000000000000000000000000000000000000000000000000000007c000000000000000000000000000000000000000000000000000000000000008c00000000000000000000000000000000000000000000000000000000000000002000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000000000000000000000000000000004747970650000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000087769746864726177000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000001977697468647261772e73656e6465722e636861696e5f7569640000000000000000000000000000000000000000000000000000000000000000000000000000046261736500000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000001777697468647261772e73656e6465722e61646472657373000000000000000000000000000000000000000000000000000000000000000000000000000000002a30783438656438353962326366306339623662616338643731346531623634363132643132323464366400000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000e77697468647261772e746f6b656e00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000036574680000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000f77697468647261772e616d6f756e740000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001431303030303030303030303030303030303030300000000000000000000000000000000000000000000000000000000000000000000000000000000000000002000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000a0000000000000000000000000000000000000000000000000000000000000002f77697468647261772e63726f73735f636861696e5f6164647265737365732e302e757365722e636861696e5f756964000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000077365706f6c6961000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000002000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000a0000000000000000000000000000000000000000000000000000000000000002d77697468647261772e63726f73735f636861696e5f6164647265737365732e302e757365722e6164647265737300000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000002a307834386564383539623263663063396236626163386437313465316236343631326431323234643664000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000002000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000a0000000000000000000000000000000000000000000000000000000000000003977697468647261772e63726f73735f636861696e5f6164647265737365732e302e6c696d69742e6c6573735f7468616e5f6f725f657175616c0000000000000000000000000000000000000000000000000000000000000000000000000000143130303030303030303030303030303030303030000000000000000000000000000000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000e77697468647261772e74785f69640000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000046626173653a3078343865643835396232636630633962366261633864373134653162363436313264313232346436643a38343533323a33333433393832323a333339313133360000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000041ffb578b6e9ab1699e4d9cd0078d9f28e7f0ef2136a11596aa7b6d7fe7f896dd353b7b786bf155c924f35d5099f0df90650e74a5858b75673835d24ac6dc8f1e41b00000000000000000000000000000000000000000000000000000000000000c080a09c4f42d262ed1f1bee31461fd10d8d8fbac6e340d9bc2b8035df5faa30f88d4da06d832693c1e28d4f647a6ff08f5d037d08ad2599964a9f3600396efdaec07e4a"],"withdrawals":[],"withdrawals_root":"0x77b0fb1616a212bd7cf33d7c28651f19bf6093b2c5f1967e674ec861aeaf9d44"},"metadata":{"block_number":33439826,"new_account_balances":{},"receipts":{}}},{"payload_id":"0x03c33cc62b81edb6","index":2,"diff":{"state_root":"0x0000000000000000000000000000000000000000000000000000000000000000","receipts_root":"0xda7caba0b5682eda3aed5f47132da84aa2c2757499c23d609aa73dd3a449be1d","logs_bloom":"0x00000000000000000000000020000000040080000000000000020005000000004000000040040000000080000000000000000000000000000002000000000040008000000000000000000000000000014000000800800000000004000000000000000000000400040100000000000000000002800100000000000b80008a02000000100000400200000100800000000000000000000004001000200000000000000000000800020000000000400000000000000000008000440801080200000000005000000400000000000000000000000110000000000008000000000000100200021004400010000000110000000400000008002000004080010000000000","gas_used":"0xd6a91","block_hash":"0x17e106bfeebb2ff0123cf2e1f555e0441ed308773224513dc4ac6257d943e52c","transactions":["0x02f89283014a3482015f830f4275830f427583045dc694a449bc031fa0b815ca14fafd0c5edb75ccd9c80f80a4de0e9a3e000000000000000000000000000000000000000000000000c249fdd327780000c001a098b7dd6d4454a8d31170b5b2d1461bc8a74eed745eddc982232b2c1483cba322a07d3acfe989366b2729aa728ebca7009c15dc908954a9fb5459b75cff1bfd103f"],"withdrawals":[],"withdrawals_root":"0x77b0fb1616a212bd7cf33d7c28651f19bf6093b2c5f1967e674ec861aeaf9d44"},"metadata":{"block_number":33439826,"new_account_balances":{},"receipts":{}}},{"payload_id":"0x03c33cc62b81edb6","index":3,"diff":{"state_root":"0x0000000000000000000000000000000000000000000000000000000000000000","receipts_root":"0xda7caba0b5682eda3aed5f47132da84aa2c2757499c23d609aa73dd3a449be1d","logs_bloom":"0x00000000000000000000000020000000040080000000000000020005000000004000000040040000000080000000000000000000000000000002000000000040008000000000000000000000000000014000000800800000000004000000000000000000000400040100000000000000000002800100000000000b80008a02000000100000400200000100800000000000000000000004001000200000000000000000000800020000000000400000000000000000008000440801080200000000005000000400000000000000000000000110000000000008000000000000100200021004400010000000110000000400000008002000004080010000000000","gas_used":"0xd6a91","block_hash":"0x17e106bfeebb2ff0123cf2e1f555e0441ed308773224513dc4ac6257d943e52c","transactions":[],"withdrawals":[],"withdrawals_root":"0x77b0fb1616a212bd7cf33d7c28651f19bf6093b2c5f1967e674ec861aeaf9d44"},"metadata":{"block_number":33439826,"new_account_balances":{},"receipts":{}}},{"payload_id":"0x03c33cc62b81edb6","index":4,"diff":{"state_root":"0x0000000000000000000000000000000000000000000000000000000000000000","receipts_root":"0xaff50907a173fc423a499319437afffb8abc2071ce36b6f040dc487579a5d4c3","logs_bloom":"0x0002800000000000002000012000040004008000000010000012000500000000480000004004000000918000000000000000000000000000000200821000806000800010000000000000000800000001c000000800800000000004202000000800000000000400040100020100000000000002800100000000000b90008a02000000100000480200000100800010080400000000000004001000224080000000000000008c0002040080000840000000000000000100c000c4080108020000000001500a000400000000000000000000100110000020000008000000000000100a00221004400010000000110100000400100008002100004280010000000000","gas_used":"0x1498a3","block_hash":"0x4764a20ee262986e45d29251db593320bd4bf6de1133de553b6363a5691e7644","transactions":["0x02f89283014a348203af830f4275830f427583045dd594a449bc031fa0b815ca14fafd0c5edb75ccd9c80f80a4de0e9a3e000000000000000000000000000000000000000000000002017a67f731740000c001a04ce59ff67dc25a76f3027441513f916b809f55b29d5de4fecd4aa0136a3a1a4fa02c1b32b3a1600f6bb2365130797238162cbc797843169a4cfb1ebb41465877c7","0x02f8d483014a348309087a830f4240830f42a883030d4094d89f830d7795c10613e4d4769c24c05bf60932c680b864b781777000000000000000000000000022e40d0a0c0bb77b570445fb59d39bcf14790b660000000000000000000000000000000000000000000000004a61b425a5ee98000000000000000000000000000000000000000000000000000006431e74449860c001a002c2402941acdc25bcaae67c62d58f1a942b32723827f77972c74b159b2c174ea04772118ec71bc7fbe0c9f1c9ef90f58927126480ca769d73704365bfbac65db3","0x02f8d483014a348308c06b830f4240830f42a883030d4094d89f830d7795c10613e4d4769c24c05bf60932c680b864b78177700000000000000000000000005643a7772017c8544d3841894c1f7c264cd05ffe0000000000000000000000000000000000000000000000000b035a61b2e8be000000000000000000000000000000000000000000000000000006431e7446c578c001a0ac31a5ad06a3897a0c1a909770badf8cec728abd2daf4d125a551778fa597124a013b1de6f741139d957f299bf22de0a91c1d8a4f2ade6743ddcec89bcc9e8b07d","0x02f8d483014a34830922b1830f4240830f42a883030d4094d89f830d7795c10613e4d4769c24c05bf60932c680b864b7817770000000000000000000000000576831e77af4b5425b39efb23528441b79ee71e20000000000000000000000000000000000000000000000002bed26c4505ca4000000000000000000000000000000000000000000000000000006431e7446f712c080a0c105ef2c930e95694d112028a642399e5a56ce6416f9b8df9ad27baa26244483a064f6e5881fa728b7afaa2e2ddd62c3182789cb247f90b6276c14f8bfc1b4f2cf"],"withdrawals":[],"withdrawals_root":"0x77b0fb1616a212bd7cf33d7c28651f19bf6093b2c5f1967e674ec861aeaf9d44"},"metadata":{"block_number":33439826,"new_account_balances":{},"receipts":{}}},{"payload_id":"0x03c33cc62b81edb6","index":5,"diff":{"state_root":"0x0000000000000000000000000000000000000000000000000000000000000000","receipts_root":"0x6d12b13dcae85ef97ec3756b317ac9d33752bcd231a9323046ecd5a65e8ca8a2","logs_bloom":"0x0002800000000000002000012000040004008000000010000012000500000000480000004004000000918000000000000000000000000100000200821000806000800010000000000000000800000001c000000800800000000004202000000800000000000400040100020100000000000002800100000000004b90008a02000000100000480200000100800010080400000000000004001000224080000000000000008c0002040080000840000000000000000100c000c4080108020000000001500a000400000000000000000000100110004020000008000000000000100a00221004400010000000110100000400100008002100004280010000000000","gas_used":"0x153998","block_hash":"0x810679ccd05f90093eb0e88549d52ad196214f3a4a555cf0b06201f30aa61a2d","transactions":["0x02f8d483014a34830966ae830f4240830f42a883030d4094d89f830d7795c10613e4d4769c24c05bf60932c680b864b781777000000000000000000000000046195a8573f2610bba630bb0bd5c21c064594f3a0000000000000000000000000000000000000000000000002c94bc176f7cb4000000000000000000000000000000000000000000000000000006431e743d37eac080a053f1881c67ad8fa9838d83943afe83b6498dae96a13a019704f25e0df515dbdba05eef8e08269eaafd63ba7e14e13d73e03ec5e7fad5bcdbaaabc124da41e8e32c"],"withdrawals":[],"withdrawals_root":"0x77b0fb1616a212bd7cf33d7c28651f19bf6093b2c5f1967e674ec861aeaf9d44"},"metadata":{"block_number":33439826,"new_account_balances":{},"receipts":{}}},{"payload_id":"0x03c33cc62b81edb6","index":6,"diff":{"state_root":"0x0000000000000000000000000000000000000000000000000000000000000000","receipts_root":"0x1b76e086c31a8a08d1c4a93b868b00238faabd4d52d9e75e55a4abf3a75e65d8","logs_bloom":"0x0002800000000000002000012000048004008000000010000012000500000000480000004004000000918000000000000000000000000100000200821000806000800010000000000000000800000001c008000800800000000004202000000800000000000400040100020100000000000002800100000000004b90008a02000000100000480200000100800010080400000000000004001000224080000000000000008c0002040080000840000000000000000100c000c4080108020000010001500a000400000000000001000000100110004021000008000000000000100a00221004400010000000110100000400100008002100004280010000000001","gas_used":"0x189dc4","block_hash":"0xfdf2cbb452a36c9c4033d1c0bc2b3dd9cee7ba91d0ca5488aa3d9a23b127b79f","transactions":["0x02f89383014a348304e447830f4240830f42a8830226b494cd997aef0b9a1d8c02a16204ccce354844edeeff80a4f7a308060000000000000000000000000000000000000000000000000000000000016636c001a07dc2c0285cd2c53657c87826a698de9ae5bb38e2580657fe1772fc08ab53a9f2a05a183dac1ed51f6aac2eff4add4510fd76d71f9dce59a3536fc00bfbb2ac750c","0x02f8d483014a3483096a27830f4240830f42a883030d4094d89f830d7795c10613e4d4769c24c05bf60932c680b864b7817770000000000000000000000000fde9b0be445930f929705125fe24049093e628e4000000000000000000000000000000000000000000000001517fd24c7f6670000000000000000000000000000000000000000000000000000006431e74408803c080a036f0e0df96ee863041cc41fad376f2f88364225ff6c10c2e492da014d71ab530a03cca82dd065d09a150f75103ea2e1f2867210c604fd82592ec49fae02cadc20a","0x02f8d483014a348309a03d830f4240830f42a883030d4094d89f830d7795c10613e4d4769c24c05bf60932c680b864b781777000000000000000000000000088c7e4701045571734e2147bad80e3d8c56500d300000000000000000000000000000000000000000000000023e284d65ede20000000000000000000000000000000000000000000000000000006431e7441d02ac080a03ee196fff4a614411f9d41431f0b174141ae6f62246df4e54117205bb19c4f64a022f123e006139ae334de3bf7b62c06b72045ba7dc0a508d137bcd056d950da33"],"withdrawals":[],"withdrawals_root":"0x77b0fb1616a212bd7cf33d7c28651f19bf6093b2c5f1967e674ec861aeaf9d44"},"metadata":{"block_number":33439826,"new_account_balances":{},"receipts":{}}},{"payload_id":"0x03c33cc62b81edb6","index":7,"diff":{"state_root":"0x0000000000000000000000000000000000000000000000000000000000000000","receipts_root":"0x065878c1c4d88295544c04fec2e74c9dd8b5d656e196a1b7b09ce8cadbb8f979","logs_bloom":"0x0002800000000010002000052000048004008000000010000012000500000000490000004004000000918000010000000000008000000100040200821000806800800010000000000000000800000001c008000800800000000004202000000804000020000400040100020100000000020002800100000000004b90008a02000000180000480200000100800010080400000000000004011000224080000000000000008c0002040080000840000200000000000100c000c4080108020000012001500a000400000000000001000000100110004021000008000000000000100a00221104400010000000110100000400100008002100004280010000000001","gas_used":"0x1bc281","block_hash":"0xcc9c18ed55c91e97f32353e253c69766cd0d2e0acb0e7f92098d01e1d7761ce3","transactions":["0x02f8d483014a3483091e1f830f4240830f42a883030d4094d89f830d7795c10613e4d4769c24c05bf60932c680b864b7817770000000000000000000000000b501c0a0f800e68d980f5253650d0cf3a69d16c00000000000000000000000000000000000000000000000000b87d57d89ffe7800000000000000000000000000000000000000000000000000006431e7442365fc001a0b276c68f59bcfb78fe7905a720e9418130d5c87d60da4b6d55faf07e1b1724aba03425daae2e51a061a26bedcd89cf6ead44146ac97f831371ec36a0192728d204","0x02f8d483014a3483094dde830f4240830f42a883030d4094d89f830d7795c10613e4d4769c24c05bf60932c680b864b78177700000000000000000000000000097cc7164250c464fea5f9f91d1abec7718814a0000000000000000000000000000000000000000000000004c40d37c20f440000000000000000000000000000000000000000000000000000006431e744372abc001a01f3e58f3baa5e472c08097dafe1e756163c61e7200dc90751f167e796d542f20a02c10596de8b29462c0953a023a8b6c06f74fe77ea66a24598df920d542edab3b","0x02f8d483014a3483094ece830f4240830f42a883030d4094d89f830d7795c10613e4d4769c24c05bf60932c680b864b781777000000000000000000000000083fe74125ec8ffaeee4b2371d7ea17f6ad6f9ba2000000000000000000000000000000000000000000000000f9e4840a6e4938000000000000000000000000000000000000000000000000000006431e744362dec001a066724129c4de96e835cd1377b55541b4582bf4ebcd7c2a3faa4231ade86b14d8a03736bce9203cc0c92878fcc28ee8710961eaddde92bf6a2158c602b4d1bbdbd7","0x02f8d483014a348303750a830f4240830f42a883030d4094d89f830d7795c10613e4d4769c24c05bf60932c680b864b7817770000000000000000000000000414d9179c5d2207a6e0efeb0319b6c556265974600000000000000000000000000000000000000000000000033979a45ffefac000000000000000000000000000000000000000000000000000006431e74442677c001a0682d2489ba1d9666324060a006f0abe06830cecdeed4398169dc9fbf7199eb59a02e971034255d087d02b25f45a7962b31360bbed70e3aa30e69ee8f64dd6afdb4","0x02f8d483014a348308acf2830f4240830f42a883030d4094d89f830d7795c10613e4d4769c24c05bf60932c680b864b781777000000000000000000000000097c152d0fa30c49603e0e3e013e36c4e29bf7fea0000000000000000000000000000000000000000000000001d58bdca2addf5000000000000000000000000000000000000000000000000000006431e744447a9c001a030e423ab3697fe4ccc5ce92232d7a642a8295f489f2e52b3c3ba2f110c828e7ca057fd4d3d0e700734568b0be067deda7927188f6a67f9600bae3d6c75d201fe57"],"withdrawals":[],"withdrawals_root":"0x77b0fb1616a212bd7cf33d7c28651f19bf6093b2c5f1967e674ec861aeaf9d44"},"metadata":{"block_number":33439826,"new_account_balances":{},"receipts":{}}},{"payload_id":"0x03c33cc62b81edb6","index":8,"diff":{"state_root":"0x0000000000000000000000000000000000000000000000000000000000000000","receipts_root":"0x7bf525f832aecc6bf7f7b7e329779640bb4477cb47bf1bde512934c5ed45519b","logs_bloom":"0x0003800000000210002000052000048004008000000010000012000500000000490000004004000000918000010000000000008000000100040200821000806800800010000000000000000800000001c00c000800802000000004202000000804000020000400040100020108000000020002800100000000044b98008a02200000180000480200000100800010080400000000002004011000224080000000000000108c0002040080000844000200000040000100c000c4080108020000012001500a000400000000000001000000104110004021000108000000000000100a00221104400010010000110100000400100008002140004280010000000001","gas_used":"0x213d0b","block_hash":"0x5f9c957cde671b50c5661b328b7f3f8a0e56e194a954d8d7cc4274eb1e014a1e","transactions":["0x02f89283014a34820392830f4275830f427583045dd594a449bc031fa0b815ca14fafd0c5edb75ccd9c80f80a4de0e9a3e000000000000000000000000000000000000000000000002017a67f731740000c001a05c4f86d9218cfab447e6ead7abb27444f7e8d3a185a1fbfb6860a36513c89d93a01d4b9b74f049bfc10feeabcb101a18a14e774e89de35ac246e6452c05e94bc98","0x02f8d483014a348309087a830f4240830f42a883030d4094d89f830d7795c10613e4d4769c24c05bf60932c680b864b781777000000000000000000000000022e40d0a0c0bb77b570445fb59d39bcf14790b660000000000000000000000000000000000000000000000004a61b425a5ee98000000000000000000000000000000000000000000000000000006431e74449860c001a002c2402941acdc25bcaae67c62d58f1a942b32723827f77972c74b159b2c174ea04772118ec71bc7fbe0c9f1c9ef90f58927126480ca769d73704365bfbac65db3","0x02f8d483014a348308c06b830f4240830f42a883030d4094d89f830d7795c10613e4d4769c24c05bf60932c680b864b78177700000000000000000000000005643a7772017c8544d3841894c1f7c264cd05ffe0000000000000000000000000000000000000000000000000b035a61b2e8be000000000000000000000000000000000000000000000000000006431e7446c578c001a0ac31a5ad06a3897a0c1a909770badf8cec728abd2daf4d125a551778fa597124a013b1de6f741139d957f299bf22de0a91c1d8a4f2ade6743ddcec89bcc9e8b07d","0x02f8d483014a34830922b1830f4240830f42a883030d4094d89f830d7795c10613e4d4769c24c05bf60932c680b864b7817770000000000000000000000000576831e77af4b5425b39efb23528441b79ee71e20000000000000000000000000000000000000000000000002bed26c4505ca4000000000000000000000000000000000000000000000000000006431e7446f712c080a0c105ef2c930e95694d112028a642399e5a56ce6416f9b8df9ad27baa26244483a064f6e5881fa728b7afaa2e2ddd62c3182789cb247f90b6276c14f8bfc1b4f2cf"],"withdrawals":[],"withdrawals_root":"0x77b0fb1616a212bd7cf33d7c28651f19bf6093b2c5f1967e674ec861aeaf9d44"},"metadata":{"block_number":33439826,"new_account_balances":{},"receipts":{}}},{"payload_id":"0x03c33cc62b81edb6","index":9,"diff":{"state_root":"0x0000000000000000000000000000000000000000000000000000000000000000","receipts_root":"0xeb419bf069b8bf9738adcb7fad118724a1d4d6a83821bc532983a2949aa0910d","logs_bloom":"0x000380000000021000200005200004800400800000001000001a000500001000490000004004000000918000010000000000008000000100040200821000806800800010000000000000000800000001c00c000800802000000004202000000804000020000400040100020108000000020002800100000000044b98008a02200000180000480200000100800010080400000000002004011000224080000000000000108c0002040080000844000200000040000100c000c4080108020000012001500a000400000000000001000000104110004021000108000000000000100a00221104400010010004110100000400100008002140004280010000000001","gas_used":"0x21de0c","block_hash":"0xb802c08c65bdefdd507fe07634ea29eeaad1859b33ffac2c426dc7b620d22b19","transactions":["0x02f8d483014a3483095beb830f4240830f42a883030d4094d89f830d7795c10613e4d4769c24c05bf60932c680b864b7817770000000000000000000000000f73c129529caa024337c39e467c720cfc45874220000000000000000000000000000000000000000000000000de4f04092790e800000000000000000000000000000000000000000000000000006431e74489081c080a0a100818c4c3ec3b0bced80f81f09fc878b23274266b45e2043956562b6714dcfa023dbcbc4df92ed5817fcc9bcd238a038aad806c69585dc8cf582e6012d012d28"],"withdrawals":[],"withdrawals_root":"0x77b0fb1616a212bd7cf33d7c28651f19bf6093b2c5f1967e674ec861aeaf9d44"},"metadata":{"block_number":33439826,"new_account_balances":{},"receipts":{}}},{"payload_id":"0x03c33cc62b81edb6","index":10,"diff":{"state_root":"0x0000000000000000000000000000000000000000000000000000000000000000","receipts_root":"0xaa280e93aa4a7d3f616ad391404411abbeebe8bc8fb1ed9b3ef4d0a42bf64ccd","logs_bloom":"0x000380000000021000200005200004800400800000001000001a000500001000490000204004000000918000010000000000008000000100040200821020886800800010000000000000000800000001c00c000800802000000004202000000804000020000400040100020108000000020002800100000000044b98008a02200000180000480200000100800010080400000000002004011000224080000000020000108c0002040080000844000200000040000100c000c4080108020000012001500a000400000000000001000000104110004021000108000000000200100a10221104400010010004110100000400100008002140004280010000000001","gas_used":"0x49f43c","block_hash":"0x2b440a266840a96993d85d45d1de1e81f7a859aaac4654dcd5a990ffa2ef947b","transactions":["0x02f90fb583014a34831d4797830f4240830f42a88327fdba94ebaff6d578733e4603b99cbdbb221482f29a78e180b90f4484779f44000000000000000000000000000000000000000000000000000000000000002000000000000000000000000000000000000000000000000000000000000000140000000000000000000000000000000000000000000000000000000000000280000000000000000000000000000000000000000000000000000000000000032000000000000000000000000000000000000000000000000000000000000003c00000000000000000000000000000000000000000000000000000000000000460000000000000000000000000000000000000000000000000000000000000050000000000000000000000000000000000000000000000000000000000000005a0000000000000000000000000000000000000000000000000000000000000064000000000000000000000000000000000000000000000000000000000000006e00000000000000000000000000000000000000000000000000000000000000780000000000000000000000000000000000000000000000000000000000000082000000000000000000000000000000000000000000000000000000000000008c000000000000000000000000000000000000000000000000000000000000009600000000000000000000000000000000000000000000000000000000000000a000000000000000000000000000000000000000000000000000000000000000aa00000000000000000000000000000000000000000000000000000000000000b400000000000000000000000000000000000000000000000000000000000000be00000000000000000000000000000000000000000000000000000000000000c800000000000000000000000000000000000000000000000000000000000000d200000000000000000000000000000000000000000000000000000000000000dc00000000000000000000000000000000000000000000000000000000000000e600000000000000000000000000000000000000000000000000000000000000001000000000000000000000000000000000000000000000000000000000053ce75532be4cf5bacb01e018950b5be900eafa59f2431fed6b869799529ab39fe0000000000000000000000000000000000000000000000000000000000000000008000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001000000000000000000000000000000000000000000000000000000000053ce76343a51197104ee22e37cf9c48a9eb5c99031a25196c2f1264deb5d4d3ff80000000000000000000000000000000000000000000000000000000000000000008000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001000000000000000000000000000000000000000000000000000000000053ce770a821c08f4e200bf42a148754153d78e977260a213094b521b5625618ec70000000000000000000000000000000000000000000000000000000000000000008000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001000000000000000000000000000000000000000000000000000000000053ce78bb3bcd3592df48dcd3a6383c8f61d8434b6058f61a587dfb0c37134294420000000000000000000000000000000000000000000000000000000000000000008000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001000000000000000000000000000000000000000000000000000000000053ce79b35d157e36939c03df12e39599530f615a90e624610d8d023eaf2f8329030000000000000000000000000000000000000000000000000000000000000000008000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001000000000000000000000000000000000000000000000000000000000053ce7a6370bb580180c882bf7214d1f701529ea455f8567b2be79496c9437a2ce30000000000000000000000000000000000000000000000000000000000000000008000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001000000000000000000000000000000000000000000000000000000000053ce7bf53208371925c87cacbb0bbfbf330fc8a02818e1d73c56760a9fded7f8c80000000000000000000000000000000000000000000000000000000000000000008000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001000000000000000000000000000000000000000000000000000000000053ce7cac670fbf544ec6d7360aacecd6e3fb35ea8a6ebef6161c9563a6d16a4a200000000000000000000000000000000000000000000000000000000000000000008000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001000000000000000000000000000000000000000000000000000000000053ce7d91406552fdfe569345c8561328604a63912a36d21cafa1efed0275ce6b190000000000000000000000000000000000000000000000000000000000000000008000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001000000000000000000000000000000000000000000000000000000000053ce7e6e0b5ccd73c9cea553a19e7ab6e533bc253f552e6b9145dd5470d2612f8d0000000000000000000000000000000000000000000000000000000000000000008000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001000000000000000000000000000000000000000000000000000000000053ce7fc72e52aaff88c842a2092b7ce047cf47a8f56da1035142a41b6a59b856420000000000000000000000000000000000000000000000000000000000000000008000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001000000000000000000000000000000000000000000000000000000000053ce80fcaa166cc2fd1353b40f3071a491cd7ca2746c8943caaa6c024c8df0131f0000000000000000000000000000000000000000000000000000000000000000008000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001000000000000000000000000000000000000000000000000000000000053ce812aabb780f12ed0c0c5dc6932220d8c5f730c54ee63384fbfe1e7fa90a5090000000000000000000000000000000000000000000000000000000000000000008000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001000000000000000000000000000000000000000000000000000000000053ce82c1dea3b99a38cf0743f31402eba0d22c4da43e715d37533da9bc5f8ca4ae0000000000000000000000000000000000000000000000000000000000000000008000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001000000000000000000000000000000000000000000000000000000000053ce835d696b1a6f5089cf9bc4c2c529e181678fa2f2feb745223e7520d885a2260000000000000000000000000000000000000000000000000000000000000000008000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001000000000000000000000000000000000000000000000000000000000053ce8400f527b7b931ddfe77007be944f58173dfc1c5928eb433ae71e96f61a8420000000000000000000000000000000000000000000000000000000000000000008000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001000000000000000000000000000000000000000000000000000000000053ce85b6dcd2b462f2d1c72e4b46ea316f9183fb9ea40866724b7eef10211a83390000000000000000000000000000000000000000000000000000000000000000008000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001000000000000000000000000000000000000000000000000000000000053ce860986c742f73c595e7cf75d5014bdccde828c0fa3891f8a7e77cbaf974e7d0000000000000000000000000000000000000000000000000000000000000000008000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001000000000000000000000000000000000000000000000000000000000053ce879d5711ffb11c2d9fe9737837f55726ba0609c21d62e2783cc38db59edafa0000000000000000000000000000000000000000000000000000000000000000008000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001000000000000000000000000000000000000000000000000000000000053ce882976c03e7cf30e96a5a578eff196e4062258f3d859abdf161bcb5fd18356000000000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000000000000000000000000000000000c080a0c7ccb6ec845a35639b2905d243be7a6cf2ee1412331d348a4bf65f53ae89cde8a06ecc40e8297c75e86332c2924b96c6bf2334a6d1b1ef803e27c9de692906b138","0x02f8b183014a3481ad830ecd10830ecdaf82b6a994af33add7918f685b2a82c1077bd8c07d220ffa0480b844095ea7b3000000000000000000000000a449bc031fa0b815ca14fafd0c5edb75ccd9c80f00000000000000000000000000000000000000000000000c6a036eb4bc740000c001a0d1877e98821074c02cf20dc84d31d70fbc00027d404fe99f3e887a33082bb6cda016f8a55aea1573b3834180e43d90eb6c4b1ffb321d2a0be8b3aa71eeaed5104a"],"withdrawals":[],"withdrawals_root":"0x77b0fb1616a212bd7cf33d7c28651f19bf6093b2c5f1967e674ec861aeaf9d44"},"metadata":{"block_number":33439826,"new_account_balances":{},"receipts":{}}}]"#;

        let flashblocks: Vec<OpFlashblockPayload> = serde_json::from_str(raw_sequence).unwrap();
        let execution_data = OpExecutionPayloadEnvelope::from_flashblocks(&flashblocks).unwrap();

        // Validate against expected final block state from base payload (index 0)
        assert_eq!(
            execution_data.parent_hash(),
            b256!("6ffd2714d5af6c412c57db3f664a5a127516573bbd987fd242d06f71ea662741")
        );
        assert_eq!(execution_data.block_number(), 0x1fe4052);
        assert_eq!(execution_data.timestamp(), 0x690fdf84);
        assert_eq!(
            execution_data.fee_recipient(),
            address!("4200000000000000000000000000000000000011")
        );
        assert_eq!(execution_data.gas_limit(), 0x3938700);
        assert_eq!(execution_data.as_v1().gas_used, 0x49f43c);

        // Base skipped state root calculation thus state root is expected to be zeros.
        // And subsequently the last flashblocks' block hash is not the final block's block hash.
        // Real block hash: 0x0c3c3ff081d8a5ea1239bfb8a0593f641154a06b783fa142809880e011cd6a3f
        assert_eq!(
            execution_data.as_v1().state_root,
            b256!("0000000000000000000000000000000000000000000000000000000000000000")
        );
        assert_eq!(
            execution_data.block_hash(),
            // last flashblock block hash
            b256!("2b440a266840a96993d85d45d1de1e81f7a859aaac4654dcd5a990ffa2ef947b")
        );

        // Verify receipts root from last flashblock (index 10)
        assert_eq!(
            execution_data.as_v1().receipts_root,
            b256!("aa280e93aa4a7d3f616ad391404411abbeebe8bc8fb1ed9b3ef4d0a42bf64ccd")
        );

        // Verify total transaction count across all 11 flashblocks
        // Index 0: 1, Index 1: 2, Index 2: 1, Index 3: 0, Index 4: 4, Index 5: 1
        // Index 6: 3, Index 7: 5, Index 8: 4, Index 9: 1, Index 10: 2
        // Total: 24 transactions
        assert_eq!(execution_data.transactions().len(), 24);

        // Verify withdrawals root from last flashblock
        assert_eq!(
            execution_data.as_v4().unwrap().withdrawals_root,
            b256!("77b0fb1616a212bd7cf33d7c28651f19bf6093b2c5f1967e674ec861aeaf9d44")
        );

        // Verify parent beacon block root from base payload
        assert_eq!(
            execution_data.parent_beacon_block_root(),
            Some(b256!("f058b1e43890ed5f838bd07e77db06d075d894343d1b31f6099a345b0d8f7d1b"))
        );
    }
}
