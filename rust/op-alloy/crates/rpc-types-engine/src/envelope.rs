//! Optimism execution payload envelope in network format and related types.
//!
//! This module uses the `snappy` compression algorithm to decompress the payload.
//! The license for snappy can be found in the `SNAPPY-LICENSE` at the root of the repository.

use crate::{
    OpExecutionPayload, OpExecutionPayloadSidecar, OpExecutionPayloadV4, OpFlashblockError,
    OpFlashblockPayload,
};
use alloc::vec::Vec;
use alloy_consensus::{Block, BlockHeader, Sealable, Transaction};
use alloy_eips::{Encodable2718, eip4895::Withdrawal, eip7685::Requests};
use alloy_primitives::{B256, Signature, keccak256};
use alloy_rpc_types_engine::{
    CancunPayloadFields, ExecutionPayloadInputV2, ExecutionPayloadV1, ExecutionPayloadV2,
    ExecutionPayloadV3, PraguePayloadFields,
};

/// A thin wrapper around [`OpExecutionPayload`] that includes the parent beacon block root.
#[derive(Debug, Clone, PartialEq, Eq)]
#[cfg_attr(feature = "serde", derive(serde::Serialize, serde::Deserialize))]
#[cfg_attr(feature = "serde", serde(rename_all = "camelCase"))]
pub struct OpExecutionPayloadEnvelope {
    /// The parent beacon block root, if any.
    pub parent_beacon_block_root: Option<B256>,
    /// The execution payload.
    pub execution_payload: OpExecutionPayload,
}

impl OpExecutionPayloadEnvelope {
    /// Returns the payload hash over the ssz encoded payload envelope data.
    ///
    /// <https://specs.optimism.io/protocol/rollup-node-p2p.html#block-signatures>
    #[cfg(feature = "std")]
    pub fn payload_hash(&self) -> crate::PayloadHash {
        use ssz::Encode;
        let ssz_bytes = self.as_ssz_bytes();
        crate::PayloadHash::from(ssz_bytes.as_slice())
    }
}

#[cfg(feature = "std")]
impl ssz::Encode for OpExecutionPayloadEnvelope {
    fn is_ssz_fixed_len() -> bool {
        false
    }

    fn ssz_append(&self, buf: &mut Vec<u8>) {
        // Write parent beacon block root only if the payload is not a v1 or v2 payload.
        // <https://specs.optimism.io/protocol/rollup-node-p2p.html#block-encoding>
        if !matches!(self.execution_payload, OpExecutionPayload::V1(_) | OpExecutionPayload::V2(_))
        {
            buf.extend_from_slice(self.parent_beacon_block_root.unwrap_or_default().as_slice());
        }

        // Write payload
        self.execution_payload.ssz_append(buf);
    }

    fn ssz_bytes_len(&self) -> usize {
        let mut len = 0;
        len += B256::ssz_fixed_len(); // parent_beacon_block_root is always 32 bytes
        len += self.execution_payload.ssz_bytes_len();
        len
    }
}

#[cfg(feature = "std")]
impl ssz::Decode for OpExecutionPayloadEnvelope {
    fn is_ssz_fixed_len() -> bool {
        false
    }

    fn from_ssz_bytes(bytes: &[u8]) -> Result<Self, ssz::DecodeError> {
        if bytes.len() < B256::ssz_fixed_len() {
            return Err(ssz::DecodeError::InvalidByteLength {
                len: bytes.len(),
                expected: B256::ssz_fixed_len(),
            });
        }

        // Decode parent_beacon_block_root
        let parent_beacon_block_root = {
            let root_bytes = &bytes[..B256::ssz_fixed_len()];
            if root_bytes.iter().all(|&b| b == 0) {
                None
            } else {
                Some(B256::from_slice(root_bytes))
            }
        };

        // Decode payload
        let execution_payload =
            OpExecutionPayload::from_ssz_bytes(&bytes[B256::ssz_fixed_len()..])?;

        Ok(Self { parent_beacon_block_root, execution_payload })
    }
}

impl From<OpNetworkPayloadEnvelope> for OpExecutionPayloadEnvelope {
    fn from(envelope: OpNetworkPayloadEnvelope) -> Self {
        Self {
            execution_payload: envelope.payload,
            parent_beacon_block_root: envelope.parent_beacon_block_root,
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

    /// Conversion from [`alloy_consensus::Block`]. Also returns the [`OpExecutionPayloadSidecar`]
    /// extracted from the block.
    ///
    /// See also [`from_block_unchecked`](OpExecutionPayload::from_block_slow).
    ///
    /// Note: This re-calculates the block hash.
    pub fn from_block_slow<T, H>(block: &Block<T, H>) -> Self
    where
        T: Encodable2718 + Transaction,
        H: BlockHeader + Sealable,
    {
        let (payload, sidecar) = OpExecutionPayload::from_block_slow(block);

        Self::new(payload, sidecar)
    }

    /// Conversion from [`alloy_consensus::Block`]. Also returns the [`OpExecutionPayloadSidecar`]
    /// extracted from the block.
    ///
    /// See also [`OpExecutionPayload::from_block_unchecked`].
    pub fn from_block_unchecked<T, H>(block_hash: B256, block: &Block<T, H>) -> Self
    where
        T: Encodable2718 + Transaction,
        H: BlockHeader,
    {
        let (payload, sidecar) = OpExecutionPayload::from_block_unchecked(block_hash, block);

        Self::new(payload, sidecar)
    }

    /// Conversion from a vec of [`OpFlashblockPayload`]. Also returns the
    /// [`OpExecutionPayloadSidecar`] extracted from the payloads.
    ///
    /// # Validation
    ///
    /// This method performs the following validations:
    /// - At least one flashblock must be present
    /// - Indices must be sequential starting from 0
    /// - First flashblock (index 0) must have a base payload
    /// - Only the first flashblock may have a base payload
    ///
    /// # Errors
    ///
    /// Returns an error if any validation fails.
    pub fn from_flashblocks(
        flashblocks: &[OpFlashblockPayload],
    ) -> Result<Self, OpFlashblockError> {
        // Validate we have at least one flashblock
        if flashblocks.is_empty() {
            return Err(OpFlashblockError::MissingPayload);
        }

        // Validate indices are sequential starting from 0
        for (i, fb) in flashblocks.iter().enumerate() {
            if fb.index as usize != i {
                return Err(OpFlashblockError::InvalidIndex);
            }
        }

        // Validate first flashblock has base and extract it
        let first = flashblocks.first().unwrap(); // Safe: checked empty above
        if first.base.is_none() {
            return Err(OpFlashblockError::MissingBasePayload);
        }

        // Validate no other flashblocks have base (only first should have it)
        for fb in flashblocks.iter().skip(1) {
            if fb.base.is_some() {
                return Err(OpFlashblockError::UnexpectedBasePayload);
            }
        }

        Ok(Self::from_flashblocks_unchecked(flashblocks))
    }

    /// Conversion from a vec of [`OpFlashblockPayload`] without validation.
    ///
    /// This is a faster alternative to [`Self::from_flashblocks`] that skips all validation
    /// checks. Use this method only when you are certain the input data is valid.
    ///
    /// # Safety Requirements
    ///
    /// The caller must ensure:
    /// - At least one flashblock is present
    /// - Indices are sequential starting from 0
    /// - First flashblock (index 0) has a base payload
    /// - Only the first flashblock has a base payload
    ///
    /// # Panics
    ///
    /// Panics if any of the safety requirements are violated.
    pub fn from_flashblocks_unchecked(flashblocks: &[OpFlashblockPayload]) -> Self {
        // Extract base from first flashblock
        // SAFETY: Caller guarantees at least one flashblock exists with base payload
        let first = flashblocks.first().expect("flashblocks must not be empty");
        let base = first.base.as_ref().expect("first flashblock must have base payload");

        // Get the final state from the last flashblock
        // SAFETY: Caller guarantees at least one flashblock exists
        let diff = &flashblocks.last().expect("flashblocks must not be empty").diff;

        // Collect all transactions and withdrawals from all flashblocks
        let (transactions, withdrawals) =
            flashblocks.iter().fold((Vec::new(), Vec::new()), |(mut txs, mut withdrawals), p| {
                txs.extend(p.diff.transactions.iter().cloned());
                withdrawals.extend(p.diff.withdrawals.iter().cloned());
                (txs, withdrawals)
            });

        let v3 = ExecutionPayloadV3 {
            blob_gas_used: diff.blob_gas_used.unwrap_or(0),
            excess_blob_gas: 0,
            payload_inner: ExecutionPayloadV2 {
                withdrawals,
                payload_inner: ExecutionPayloadV1 {
                    parent_hash: base.parent_hash,
                    fee_recipient: base.fee_recipient,
                    state_root: diff.state_root,
                    receipts_root: diff.receipts_root,
                    logs_bloom: diff.logs_bloom,
                    prev_randao: base.prev_randao,
                    block_number: base.block_number,
                    gas_limit: base.gas_limit,
                    gas_used: diff.gas_used,
                    timestamp: base.timestamp,
                    extra_data: base.extra_data.clone(),
                    base_fee_per_gas: base.base_fee_per_gas,
                    block_hash: diff.block_hash,
                    transactions,
                },
            },
        };

        // Before Isthmus hardfork, withdrawals_root was not included.
        // A zero withdrawals_root indicates a pre-Isthmus flashblock.
        if diff.withdrawals_root == B256::ZERO {
            return Self::v3(v3, Vec::new(), base.parent_beacon_block_root);
        }

        let v4 =
            OpExecutionPayloadV4 { withdrawals_root: diff.withdrawals_root, payload_inner: v3 };

        Self::v4(v4, Vec::new(), base.parent_beacon_block_root, Default::default())
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

    /// Returns the parent beacon block root, if any.
    pub fn parent_beacon_block_root(&self) -> Option<B256> {
        self.sidecar.parent_beacon_block_root()
    }

    /// Return the withdrawals for the payload or attributes.
    pub const fn withdrawals(&self) -> Option<&Vec<Withdrawal>> {
        match &self.payload {
            OpExecutionPayload::V1(_) => None,
            OpExecutionPayload::V2(execution_payload_v2) => Some(&execution_payload_v2.withdrawals),
            OpExecutionPayload::V3(execution_payload_v3) => {
                Some(execution_payload_v3.withdrawals())
            }
            OpExecutionPayload::V4(op_execution_payload_v4) => {
                Some(op_execution_payload_v4.payload_inner.withdrawals())
            }
        }
    }

    /// Returns the parent hash of the block.
    pub const fn parent_hash(&self) -> B256 {
        self.payload.parent_hash()
    }

    /// Returns the hash of the block.
    pub const fn block_hash(&self) -> B256 {
        self.payload.block_hash()
    }

    /// Returns the number of the block.
    pub const fn block_number(&self) -> u64 {
        self.payload.block_number()
    }
}

/// Optimism execution payload envelope in network format.
///
/// This struct is used to represent payloads that are sent over the Optimism
/// CL p2p network in a snappy-compressed format.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct OpNetworkPayloadEnvelope {
    /// The execution payload.
    pub payload: OpExecutionPayload,
    /// A signature for the payload.
    pub signature: Signature,
    /// The hash of the payload.
    pub payload_hash: PayloadHash,
    /// The parent beacon block root.
    pub parent_beacon_block_root: Option<B256>,
}

impl OpNetworkPayloadEnvelope {
    /// Decode a payload envelope from a snappy-compressed byte array.
    /// The payload version decoded is `ExecutionPayloadV1` from SSZ bytes.
    #[cfg(feature = "std")]
    pub fn decode_v1(data: &[u8]) -> Result<Self, PayloadEnvelopeError> {
        use ssz::Decode;
        let mut decoder = snap::raw::Decoder::new();
        let decompressed = decoder.decompress_vec(data)?;

        if decompressed.len() < 66 {
            return Err(PayloadEnvelopeError::InvalidLength);
        }

        let sig_data = &decompressed[..65];
        let block_data = &decompressed[65..];

        let signature = Signature::try_from(sig_data)?;
        let hash = PayloadHash::from(block_data);

        let payload = OpExecutionPayload::V1(
            alloy_rpc_types_engine::ExecutionPayloadV1::from_ssz_bytes(block_data)?,
        );

        Ok(Self { payload, signature, payload_hash: hash, parent_beacon_block_root: None })
    }

    /// Encodes a payload envelope as a snappy-compressed byte array.
    #[cfg(feature = "std")]
    pub fn encode_v1(&self) -> Result<Vec<u8>, PayloadEnvelopeEncodeError> {
        use ssz::Encode;
        let execution_payload_v1 = match &self.payload {
            OpExecutionPayload::V1(execution_payload_v1) => execution_payload_v1,
            _ => return Err(PayloadEnvelopeEncodeError::WrongVersion),
        };

        let mut data = Vec::new();
        let mut sig = self.signature.as_bytes();
        sig[64] = self.signature.v() as u8;
        data.extend_from_slice(&sig[..]);
        let block_data = execution_payload_v1.as_ssz_bytes();
        data.extend_from_slice(block_data.as_slice());

        Ok(snap::raw::Encoder::new().compress_vec(&data)?)
    }

    /// Decode a payload envelope from a snappy-compressed byte array.
    /// The payload version decoded is `ExecutionPayloadV2` from SSZ bytes.
    #[cfg(feature = "std")]
    pub fn decode_v2(data: &[u8]) -> Result<Self, PayloadEnvelopeError> {
        use ssz::Decode;
        let mut decoder = snap::raw::Decoder::new();
        let decompressed = decoder.decompress_vec(data)?;

        if decompressed.len() < 66 {
            return Err(PayloadEnvelopeError::InvalidLength);
        }

        let sig_data = &decompressed[..65];
        let block_data = &decompressed[65..];

        let signature = Signature::try_from(sig_data)?;
        let hash = PayloadHash::from(block_data);

        let payload = OpExecutionPayload::V2(
            alloy_rpc_types_engine::ExecutionPayloadV2::from_ssz_bytes(block_data)?,
        );

        Ok(Self { payload, signature, payload_hash: hash, parent_beacon_block_root: None })
    }

    /// Encodes a payload envelope as a snappy-compressed byte array.
    #[cfg(feature = "std")]
    pub fn encode_v2(&self) -> Result<Vec<u8>, PayloadEnvelopeEncodeError> {
        use ssz::Encode;
        let execution_payload_v2 = match &self.payload {
            OpExecutionPayload::V2(execution_payload_v2) => execution_payload_v2,
            _ => return Err(PayloadEnvelopeEncodeError::WrongVersion),
        };

        let mut data = Vec::new();
        let mut sig = self.signature.as_bytes();
        sig[64] = self.signature.v() as u8;
        data.extend_from_slice(&sig[..]);
        let block_data = execution_payload_v2.as_ssz_bytes();
        data.extend_from_slice(block_data.as_slice());

        Ok(snap::raw::Encoder::new().compress_vec(&data)?)
    }

    /// Decode a payload envelope from a snappy-compressed byte array.
    /// The payload version decoded is `ExecutionPayloadV3` from SSZ bytes.
    #[cfg(feature = "std")]
    pub fn decode_v3(data: &[u8]) -> Result<Self, PayloadEnvelopeError> {
        use ssz::Decode;
        let mut decoder = snap::raw::Decoder::new();
        let decompressed = decoder.decompress_vec(data)?;

        if decompressed.len() < 98 {
            return Err(PayloadEnvelopeError::InvalidLength);
        }

        let sig_data = &decompressed[..65];
        let parent_beacon_block_root = &decompressed[65..97];
        let block_data = &decompressed[97..];

        let signature = Signature::try_from(sig_data)?;
        let parent_beacon_block_root = B256::from_slice(parent_beacon_block_root);
        let hash = PayloadHash::from(
            [parent_beacon_block_root.as_slice(), block_data].concat().as_slice(),
        );

        let payload = OpExecutionPayload::V3(
            alloy_rpc_types_engine::ExecutionPayloadV3::from_ssz_bytes(block_data)?,
        );

        Ok(Self {
            payload,
            signature,
            payload_hash: hash,
            parent_beacon_block_root: Some(parent_beacon_block_root),
        })
    }

    /// Encodes a payload envelope as a snappy-compressed byte array.
    #[cfg(feature = "std")]
    pub fn encode_v3(&self) -> Result<Vec<u8>, PayloadEnvelopeEncodeError> {
        use ssz::Encode;
        let execution_payload_v3 = match &self.payload {
            OpExecutionPayload::V3(execution_payload_v3) => execution_payload_v3,
            _ => return Err(PayloadEnvelopeEncodeError::WrongVersion),
        };

        let mut data = Vec::new();
        let mut sig = self.signature.as_bytes();
        sig[64] = self.signature.v() as u8;
        data.extend_from_slice(&sig[..]);
        data.extend_from_slice(self.parent_beacon_block_root.as_ref().unwrap().as_slice());
        let block_data = execution_payload_v3.as_ssz_bytes();
        data.extend_from_slice(block_data.as_slice());

        Ok(snap::raw::Encoder::new().compress_vec(&data)?)
    }

    /// Decode a payload envelope from a snappy-compressed byte array.
    /// The payload version decoded is `ExecutionPayloadV4` from SSZ bytes.
    #[cfg(feature = "std")]
    pub fn decode_v4(data: &[u8]) -> Result<Self, PayloadEnvelopeError> {
        use ssz::Decode;
        let mut decoder = snap::raw::Decoder::new();
        let decompressed = decoder.decompress_vec(data)?;

        if decompressed.len() < 98 {
            return Err(PayloadEnvelopeError::InvalidLength);
        }

        let sig_data = &decompressed[..65];
        let parent_beacon_block_root = &decompressed[65..97];
        let block_data = &decompressed[97..];

        let signature = Signature::try_from(sig_data)?;
        let parent_beacon_block_root = B256::from_slice(parent_beacon_block_root);
        let hash = PayloadHash::from(
            [parent_beacon_block_root.as_slice(), block_data].concat().as_slice(),
        );

        let payload = OpExecutionPayload::V4(OpExecutionPayloadV4::from_ssz_bytes(block_data)?);

        Ok(Self {
            payload,
            signature,
            payload_hash: hash,
            parent_beacon_block_root: Some(parent_beacon_block_root),
        })
    }

    /// Encodes a payload envelope as a snappy-compressed byte array.
    #[cfg(feature = "std")]
    pub fn encode_v4(&self) -> Result<Vec<u8>, PayloadEnvelopeEncodeError> {
        use ssz::Encode;
        let execution_payload_v4 = match &self.payload {
            OpExecutionPayload::V4(execution_payload_v4) => execution_payload_v4,
            _ => return Err(PayloadEnvelopeEncodeError::WrongVersion),
        };

        let mut data = Vec::new();
        let mut sig = self.signature.as_bytes();
        sig[64] = self.signature.v() as u8;
        data.extend_from_slice(&sig[..]);
        data.extend_from_slice(self.parent_beacon_block_root.as_ref().unwrap().as_slice());
        let block_data = execution_payload_v4.as_ssz_bytes();
        data.extend_from_slice(block_data.as_slice());

        Ok(snap::raw::Encoder::new().compress_vec(&data)?)
    }
}

/// Errors that can occur when encoding a payload envelope.
#[derive(Debug, Clone, PartialEq, Eq, thiserror::Error)]
pub enum PayloadEnvelopeEncodeError {
    /// Wrong versions of the payload.
    #[error("Wrong version of the payload")]
    WrongVersion,
    /// An error occurred during snap encoding.
    #[error(transparent)]
    #[cfg(feature = "std")]
    SnapEncoding(#[from] snap::Error),
}

/// Errors that can occur when decoding a payload envelope.
#[derive(Debug, Clone, PartialEq, Eq, thiserror::Error)]
pub enum PayloadEnvelopeError {
    /// The snappy encoding is broken.
    #[error("Broken snappy encoding")]
    BrokenSnappyEncoding,
    /// The signature is invalid.
    #[error("Invalid signature")]
    InvalidSignature,
    /// The SSZ encoding is broken.
    #[error("Broken SSZ encoding")]
    BrokenSszEncoding,
    /// The payload envelope is of invalid length.
    #[error("Invalid length")]
    InvalidLength,
}

impl From<alloy_primitives::SignatureError> for PayloadEnvelopeError {
    fn from(_: alloy_primitives::SignatureError) -> Self {
        Self::InvalidSignature
    }
}

#[cfg(feature = "std")]
impl From<snap::Error> for PayloadEnvelopeError {
    fn from(_: snap::Error) -> Self {
        Self::BrokenSnappyEncoding
    }
}

#[cfg(feature = "std")]
impl From<ssz::DecodeError> for PayloadEnvelopeError {
    fn from(_: ssz::DecodeError) -> Self {
        Self::BrokenSszEncoding
    }
}

/// Represents the Keccak256 hash of the block
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
    use alloy_primitives::b256;

    #[test]
    #[cfg(feature = "std")]
    fn test_roundtrip_encode_rpc_execution_payload_envelope() {
        use alloy_primitives::hex;
        use ssz::{Decode, Encode};
        let data = hex!(
            "00000000000000000000000000000000000000000000000000000000000001230000000000000000000000000000000000000000000000000000000000000123000000000000000000000000000000000000045600000000000000000000000000000000000000000000000000000000000007890000000000000000000000000000000000000000000000000000000000000abc0d0e0f000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000111de000000000000004d01000000000000bc010000000000002b02000000000000300200000903000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000088832020000380200000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001236666040000009999"
        );

        let payload = OpExecutionPayloadEnvelope::from_ssz_bytes(&data).unwrap();
        let serialized = payload.as_ssz_bytes();
        assert_eq!(data, &serialized[..]);
    }

    #[test]
    #[cfg(feature = "serde")]
    fn test_serde_roundtrip_op_execution_payload_envelope() {
        let envelope_str = r#"{
            "executionPayload": {"parentHash":"0xe927a1448525fb5d32cb50ee1408461a945ba6c39bd5cf5621407d500ecc8de9","feeRecipient":"0x0000000000000000000000000000000000000000","stateRoot":"0x10f8a0830000e8edef6d00cc727ff833f064b1950afd591ae41357f97e543119","receiptsRoot":"0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421","logsBloom":"0x00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000","prevRandao":"0xe0d8b4521a7da1582a713244ffb6a86aa1726932087386e2dc7973f43fc6cb24","blockNumber":"0x1","gasLimit":"0x2ffbd2","gasUsed":"0x0","timestamp":"0x1235","extraData":"0xd883010d00846765746888676f312e32312e30856c696e7578","baseFeePerGas":"0x342770c0","blockHash":"0x44d0fa5f2f73a938ebb96a2a21679eb8dea3e7b7dd8fd9f35aa756dda8bf0a8a","transactions":[],"withdrawals":[],"blobGasUsed":"0x0","excessBlobGas":"0x0","withdrawalsRoot":"0x10f8a0830000e8edef6d00cc727ff833f064b1950afd591ae41357f97e543119"},
            "parentBeaconBlockRoot": "0x9999999999999999999999999999999999999999999999999999999999999999"
        }"#;

        let envelope: OpExecutionPayloadEnvelope = serde_json::from_str(envelope_str).unwrap();
        let expected = b256!("9999999999999999999999999999999999999999999999999999999999999999");
        assert_eq!(envelope.parent_beacon_block_root.unwrap(), expected);
        let _ = serde_json::to_string(&envelope).unwrap();
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
    fn test_roundtrip_encode_envelope_v1() {
        use alloy_primitives::hex;
        let data = hex::decode("0xbd04f043128457c6ccf35128497167442bcc0f8cce78cda8b366e6a12e526d938d1e4c1046acffffbfc542a7e212bb7d80d3a4b2f84f7b196d935398a24eb84c519789b401000000fe0300fe0300fe0300fe0300fe0300fe0300a203000c4a8fd56621ad04fc0101067601008ce60be0005b220117c32c0f3b394b346c2aa42cfa8157cd41f891aa0bec485a62fc010000").unwrap();
        let payload_envelop = OpNetworkPayloadEnvelope::decode_v1(&data).unwrap();
        assert_eq!(1725271882, payload_envelop.payload.timestamp());
        let encoded = payload_envelop.encode_v1().unwrap();
        assert_eq!(data, encoded);
    }

    #[test]
    #[cfg(feature = "std")]
    fn test_roundtrip_encode_envelope_v2() {
        use alloy_primitives::hex;
        let data = hex::decode("0xc104f0433805080eb36c0b130a7cc1dc74c3f721af4e249aa6f61bb89d1557143e971bb738a3f3b98df7c457e74048e9d2d7e5cd82bb45e3760467e2270e9db86d1271a700000000fe0300fe0300fe0300fe0300fe0300fe0300a203000c6b89d46525ad000205067201009cda69cb5b9b73fc4eb2458b37d37f04ff507fe6c9cd2ab704a05ea9dae3cd61760002000000020000").unwrap();
        let payload_envelop = OpNetworkPayloadEnvelope::decode_v2(&data).unwrap();
        assert_eq!(1708427627, payload_envelop.payload.timestamp());
        let encoded = payload_envelop.encode_v2().unwrap();
        assert_eq!(data, encoded);
    }

    #[test]
    #[cfg(feature = "std")]
    fn test_roundtrip_encode_envelope_v3() {
        use alloy_primitives::hex;
        let data = hex::decode("0xf104f0434442b9eb38b259f5b23826e6b623e829d2fb878dac70187a1aecf42a3f9bedfd29793d1fcb5822324be0d3e12340a95855553a65d64b83e5579dffb31470df5d010000006a03000412346a1d00fe0100fe0100fe0100fe0100fe0100fe01004201000cc588d465219504100201067601007cfece77b89685f60e3663b6e0faf2de0734674eb91339700c4858c773a8ff921e014401043e0100").unwrap();
        let payload_envelop = OpNetworkPayloadEnvelope::decode_v3(&data).unwrap();
        assert_eq!(1708427461, payload_envelop.payload.timestamp());
        let encoded = payload_envelop.encode_v3().unwrap();
        assert_eq!(data, encoded);
    }

    #[test]
    #[cfg(feature = "std")]
    fn test_roundtrip_encode_envelope_v4() {
        use alloy_primitives::hex;
        let data = hex::decode("0x9105f043cee25401b6853202950d1d8a082f31a80c4fef5782c049a731f5d104b1b9b9aa7618605b420438ae98b44c8aaaebd482854473c2ae57c079286bb634bece5210000000006a03000412346a1d00fe0100fe0100fe0100fe0100fe0100fe01004201000c5766d26721950430020106f6010001440104b60100049876").unwrap();
        let payload_envelop = OpNetworkPayloadEnvelope::decode_v4(&data).unwrap();
        assert_eq!(1741842007, payload_envelop.payload.timestamp());
        let encoded = payload_envelop.encode_v4().unwrap();
        assert_eq!(data, encoded);
    }

    // Helper function to create a test flashblock
    #[cfg(test)]
    fn create_test_flashblock(index: u64, with_base: bool) -> OpFlashblockPayload {
        use crate::flashblock::{
            OpFlashblockPayloadBase, OpFlashblockPayloadDelta, OpFlashblockPayloadMetadata,
        };
        use alloc::collections::BTreeMap;
        use alloy_primitives::{Address, Bloom, Bytes, U256};
        use alloy_rpc_types_engine::PayloadId;

        let base = if with_base {
            Some(OpFlashblockPayloadBase {
                parent_beacon_block_root: B256::ZERO,
                parent_hash: B256::ZERO,
                fee_recipient: Address::ZERO,
                prev_randao: B256::ZERO,
                block_number: 100,
                gas_limit: 30_000_000,
                timestamp: 1234567890,
                extra_data: Bytes::default(),
                base_fee_per_gas: U256::from(1000000000u64),
            })
        } else {
            None
        };

        let diff = OpFlashblockPayloadDelta {
            state_root: B256::ZERO,
            receipts_root: B256::ZERO,
            logs_bloom: Bloom::ZERO,
            gas_used: 21000,
            block_hash: B256::ZERO,
            transactions: Vec::new(),
            withdrawals: Vec::new(),
            withdrawals_root: B256::from([1u8; 32]), // Non-zero for Isthmus
            blob_gas_used: Some(0),
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
        let result = OpExecutionData::from_flashblocks(&[]);
        assert!(matches!(result, Err(OpFlashblockError::MissingPayload)));
    }

    #[test]
    fn test_from_flashblocks_non_sequential_indices() {
        let fb1 = create_test_flashblock(0, true);
        let fb2 = create_test_flashblock(2, false); // Skip index 1

        let result = OpExecutionData::from_flashblocks(&[fb1, fb2]);
        assert!(matches!(result, Err(OpFlashblockError::InvalidIndex)));
    }

    #[test]
    fn test_from_flashblocks_missing_base_in_first() {
        let fb1 = create_test_flashblock(0, false); // First should have base

        let result = OpExecutionData::from_flashblocks(&[fb1]);
        assert!(matches!(result, Err(OpFlashblockError::MissingBasePayload)));
    }

    #[test]
    fn test_from_flashblocks_unexpected_base_in_second() {
        let fb1 = create_test_flashblock(0, true);
        let fb2 = create_test_flashblock(1, true); // Should not have base

        let result = OpExecutionData::from_flashblocks(&[fb1, fb2]);
        assert!(matches!(result, Err(OpFlashblockError::UnexpectedBasePayload)));
    }

    #[test]
    fn test_from_flashblocks_single_valid_flashblock() {
        let fb1 = create_test_flashblock(0, true);

        let result = OpExecutionData::from_flashblocks(&[fb1]);
        assert!(result.is_ok(), "Single valid flashblock should succeed");
    }

    #[test]
    fn test_from_flashblocks_multiple_valid_flashblocks() {
        let fb1 = create_test_flashblock(0, true);
        let fb2 = create_test_flashblock(1, false);
        let fb3 = create_test_flashblock(2, false);

        let result = OpExecutionData::from_flashblocks(&[fb1, fb2, fb3]);
        assert!(result.is_ok(), "Multiple valid flashblocks should succeed");
    }

    #[test]
    fn test_from_flashblocks_wrong_first_index() {
        let fb1 = create_test_flashblock(1, true); // Should be index 0
        let result = OpExecutionData::from_flashblocks(&[fb1]);
        assert!(matches!(result, Err(OpFlashblockError::InvalidIndex)));
    }

    // Real-world test case from Unichain Sepolia
    // <https://unichain-sepolia.blockscout.com/block/35535698>
    #[test]
    #[cfg(feature = "serde")]
    fn test_from_flashblocks_unichain_sepolia_block() {
        use alloy_primitives::{address, b256};

        let raw_sequence = r#"[{"payload_id":"0x03c446f063e3735a","index":0,"base":{"parent_beacon_block_root":"0xf6d335a6b2b4fd8fb539cd51a49769df4d53c31a90c54dd270e54542638ff101","parent_hash":"0x06ff95a9cd23b0328da74a984aa986b2e01d377dab1825f1029e39ece6c4a3ea","fee_recipient":"0x4200000000000000000000000000000000000011","prev_randao":"0x8beee738d20a9d77c5f27e9cb799ebe5b536f0985efad5f7d77ebff47f092c4a","block_number":"0x21e3b52","gas_limit":"0x3938700","timestamp":"0x690be89e","extra_data":"0x00000000320000000c","base_fee_per_gas":"0x33"},"diff":{"state_root":"0xb29a9bcae8cf3ae6d68985fcd70db80b3818cd629c9d5da0bb116451739b2078","receipts_root":"0x91d8ad10740ccfc1bd848fba0e02668d95769c08eeea30f10698692ba86c6159","logs_bloom":"0x00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000","gas_used":"0x10994","block_hash":"0xa66f8562a861f906a2438d7d6ba79495640d98d9c6922b9605c54b57f97a345c","transactions":["0x7ef90104a035dd2ec802504a143048c7830f8f570e0d6cf5147217af869939c6b4ba710a3694deaddeaddeaddeaddeaddeaddeaddeaddead00019442000000000000000000000000000000000000158080830f424080b8b0098999be000007d0000dbba0000000000000000800000000690be848000000000092042e000000000000000000000000000000000000000000000000000000000000000900000000000000000000000000000000000000000000000000000000000000010ffd7e2fb2c36e5f27c015872ce733a7b4f3fc0f4ee668d7469c557c48f8250f0000000000000000000000004ab3387810ef500bfe05a49dc53a44c222cbab3e000000000000000000000000","0x02f87e8205158401c8ea9180338255789400000000000000000000000000000000000000008096426c6f636b204e756d6265723a203335353335363938c080a091f83058c881d9ad71c179ce680326501702eb68150d20b2bf7786e388f954a2a0180185d83e503f11bf3c265c1f9296ed8d3d7c04031cd8bb30509ad188ce7bbc"],"withdrawals":[],"withdrawals_root":"0x62ed62e0391b081bf172f287fbbe75e87d8a6c22f1d3b1f1aef4788c134633d2"},"metadata":{"block_number":35535698,"new_account_balances":{},"receipts":{}}},{"payload_id":"0x03c446f063e3735a","index":1,"base":null,"diff":{"state_root":"0xfb1794f74d405b345672c57a5053c6105cc55c8e63f96fb0db5b0260df42413a","receipts_root":"0x1eaaaeb9d43bead7d32b90f1b320589174c63d2fa8f5fd366f841a205b1eb2e0","logs_bloom":"0x00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000002000000001000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000040000000000000000000000000000000000000000000000000004040000000000000000000000000000000000000000000000000000000000000000000000","gas_used":"0x18f7d","block_hash":"0x67b0521ebfcb03d6ce2b6e1bad9c9c66795365f63ad8dc51e1e8f582a5ab7821","transactions":["0x02f86c8205158401c8ea92803382880994f878f0340bf132c28f3211e8b46c569edf81749580843fd553e8c001a0d73ce313aafea312e0b7244767e45f8b05d50305e0f4e4c3c564ddc751666815a02ee015ce2363311823c0b2e96bfb0e8090fd53c6cdd99be8cf343af123036dfc"],"withdrawals":[],"withdrawals_root":"0x62ed62e0391b081bf172f287fbbe75e87d8a6c22f1d3b1f1aef4788c134633d2"},"metadata":{"block_number":35535698,"new_account_balances":{},"receipts":{}}},{"payload_id":"0x03c446f063e3735a","index":2,"base":null,"diff":{"state_root":"0x90dd105c4a2a0dd9ffe994204bfa3e2b4f70f7ea760d5cb9a4263f26a89f91b4","receipts_root":"0x0fff0488aa3732c34018b938839ab2f0caa96018221e4ffaeca011fb06ba288f","logs_bloom":"0x00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000002000000001000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000040000000000000000000000000000000000000000000000000004040000000000000000000000000000000000000000000000000000000000000000000000","gas_used":"0x21566","block_hash":"0x720feb7457110a565b479fafbaa89cc984f5d673846a27d44bbb8cf5200b32fe","transactions":["0x02f86c8205158401c8ea93803382880994f878f0340bf132c28f3211e8b46c569edf81749580843fd553e8c001a0f8cd94080642e116bc772f36a02d002505227aa542e1c13e5129ab40b8b037fba00608318d3895388e39b218bcb275380cebc566e68f26d3d434e32b8b58366cdf"],"withdrawals":[],"withdrawals_root":"0x62ed62e0391b081bf172f287fbbe75e87d8a6c22f1d3b1f1aef4788c134633d2"},"metadata":{"block_number":35535698,"new_account_balances":{},"receipts":{}}},{"payload_id":"0x03c446f063e3735a","index":3,"base":null,"diff":{"state_root":"0x71f8c60fdfdd84cffda3b0b6af7c8ff92195918f4fc2abae750a7306521ac0dc","receipts_root":"0xa62d1d98f56ffb1464a2beb185484253df68208004306e155c0bd1519137afe6","logs_bloom":"0x00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000002000000001000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000040000000000000000000000000000000000000000000000000004040000000000000000000000000000000000000000000000000000000000000000000000","gas_used":"0x29b4f","block_hash":"0x670844e30f7325d4f290ea375e01f7e819afca317fc7db9723e6867a184984fa","transactions":["0x02f86c8205158401c8ea94803382880994f878f0340bf132c28f3211e8b46c569edf81749580843fd553e8c080a04368492ec1d087703aaf6f5fefe4427b3bf382e5cd07133f638bb6701f15fe61a05e28757fbdc7e744118be36d5a1548eb7c009eefcb5dc5c5040e09c2fc6de9d8"],"withdrawals":[],"withdrawals_root":"0x62ed62e0391b081bf172f287fbbe75e87d8a6c22f1d3b1f1aef4788c134633d2"},"metadata":{"block_number":35535698,"new_account_balances":{},"receipts":{}}},{"payload_id":"0x03c446f063e3735a","index":4,"base":null,"diff":{"state_root":"0x5615e4342d231c352438f0ba6a8f0f641459f67961961764b781a909969b28ad","receipts_root":"0x588e1d47b0618d7e935b20c3945cba3b7b8c00141904f79ceed20312ea502e63","logs_bloom":"0x00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000002000000001000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000040000000000000000000000000000000000000000000000000004040000000000000000000000000000000000000000000000000000000000000000000000","gas_used":"0x32138","block_hash":"0xc463a3120c35268f610d969f5608b479332ef10953af77c7a6be806195831196","transactions":["0x02f86c8205158401c8ea95803382880994f878f0340bf132c28f3211e8b46c569edf81749580843fd553e8c080a0802ba6d4f37e3b8de96095bd0b216144f276171d16dc62a004f1a89009af5deea00f0c6250cfd1a062a1bc2bc353a5c227a980cac0f233b7be8932f2192342ec4f"],"withdrawals":[],"withdrawals_root":"0x62ed62e0391b081bf172f287fbbe75e87d8a6c22f1d3b1f1aef4788c134633d2"},"metadata":{"block_number":35535698,"new_account_balances":{},"receipts":{}}}]"#;

        let flashblocks: Vec<OpFlashblockPayload> = serde_json::from_str(raw_sequence).unwrap();
        let execution_data = OpExecutionData::from_flashblocks(&flashblocks).unwrap();

        // Validate against expected final block state
        assert_eq!(
            execution_data.payload.parent_hash(),
            b256!("06ff95a9cd23b0328da74a984aa986b2e01d377dab1825f1029e39ece6c4a3ea")
        );
        assert_eq!(
            execution_data.payload.block_hash(),
            b256!("c463a3120c35268f610d969f5608b479332ef10953af77c7a6be806195831196")
        );
        assert_eq!(execution_data.payload.block_number(), 0x21E3B52);
        assert_eq!(execution_data.payload.timestamp(), 0x690be89e);
        assert_eq!(
            execution_data.payload.fee_recipient(),
            address!("4200000000000000000000000000000000000011")
        );
        assert_eq!(execution_data.payload.gas_limit(), 0x3938700);
        assert_eq!(execution_data.payload.as_v1().gas_used, 0x32138);
        assert_eq!(
            execution_data.payload.as_v1().state_root,
            b256!("5615e4342d231c352438f0ba6a8f0f641459f67961961764b781a909969b28ad")
        );
        assert_eq!(
            execution_data.payload.as_v1().receipts_root,
            b256!("588e1d47b0618d7e935b20c3945cba3b7b8c00141904f79ceed20312ea502e63")
        );
        assert_eq!(execution_data.payload.transactions().len(), 6);
        assert_eq!(
            execution_data.payload.as_v4().unwrap().withdrawals_root,
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
        let execution_data = OpExecutionData::from_flashblocks(&flashblocks).unwrap();

        // Validate against expected final block state from base payload (index 0)
        assert_eq!(
            execution_data.payload.parent_hash(),
            b256!("6ffd2714d5af6c412c57db3f664a5a127516573bbd987fd242d06f71ea662741")
        );
        assert_eq!(execution_data.payload.block_number(), 0x1fe4052);
        assert_eq!(execution_data.payload.timestamp(), 0x690fdf84);
        assert_eq!(
            execution_data.payload.fee_recipient(),
            address!("4200000000000000000000000000000000000011")
        );
        assert_eq!(execution_data.payload.gas_limit(), 0x3938700);
        assert_eq!(execution_data.payload.as_v1().gas_used, 0x49f43c);

        // Base skipped state root calculation thus state root is expected to be zeros.
        // And subsequently the last flashblocks' block hash is not the final block's block hash.
        // Real block hash: 0x0c3c3ff081d8a5ea1239bfb8a0593f641154a06b783fa142809880e011cd6a3f
        assert_eq!(
            execution_data.payload.as_v1().state_root,
            b256!("0000000000000000000000000000000000000000000000000000000000000000")
        );
        assert_eq!(
            execution_data.payload.block_hash(),
            // last flashblock block hash
            b256!("2b440a266840a96993d85d45d1de1e81f7a859aaac4654dcd5a990ffa2ef947b")
        );

        // Verify receipts root from last flashblock (index 10)
        assert_eq!(
            execution_data.payload.as_v1().receipts_root,
            b256!("aa280e93aa4a7d3f616ad391404411abbeebe8bc8fb1ed9b3ef4d0a42bf64ccd")
        );

        // Verify total transaction count across all 11 flashblocks
        // Index 0: 1, Index 1: 2, Index 2: 1, Index 3: 0, Index 4: 4, Index 5: 1
        // Index 6: 3, Index 7: 5, Index 8: 4, Index 9: 1, Index 10: 2
        // Total: 24 transactions
        assert_eq!(execution_data.payload.transactions().len(), 24);

        // Verify withdrawals root from last flashblock
        assert_eq!(
            execution_data.payload.as_v4().unwrap().withdrawals_root,
            b256!("77b0fb1616a212bd7cf33d7c28651f19bf6093b2c5f1967e674ec861aeaf9d44")
        );

        // Verify parent beacon block root from base payload
        assert_eq!(
            execution_data.parent_beacon_block_root(),
            Some(b256!("f058b1e43890ed5f838bd07e77db06d075d894343d1b31f6099a345b0d8f7d1b"))
        );
    }
}
