//! Private wire representation of signed OP Stack gossip payloads.

use crate::HandlerEncodeError;
use alloy_primitives::{B256, Bytes, Signature};
use alloy_rpc_types_engine::{ExecutionPayloadV1, ExecutionPayloadV2, ExecutionPayloadV3};
use op_alloy_rpc_types_engine::{OpExecutionPayloadEnvelope, OpExecutionPayloadV4, PayloadHash};
use ssz::{Decode, Encode};

/// The execution payload version selected by the gossip topic.
///
/// The version is transport metadata and is intentionally kept separate from
/// [`SignedGossipPayload`], which represents only the decompressed bytes carried by the message.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub(crate) enum GossipPayloadVersion {
    /// Pre-Canyon payload.
    V1,
    /// Canyon payload.
    V2,
    /// Ecotone payload.
    V3,
    /// Isthmus payload.
    V4,
}

/// The exact encoded execution payload envelope bytes authenticated by the block signature.
///
/// For V1 and V2 these are the SSZ execution payload bytes. For V3 and V4 these are the parent
/// beacon block root followed by the SSZ execution payload bytes.
#[derive(Clone, Debug, PartialEq, Eq)]
pub(crate) struct EncodedExecutionPayloadEnvelope(Bytes);

impl EncodedExecutionPayloadEnvelope {
    /// Encodes an execution payload envelope for the version selected by its gossip topic.
    fn encode(
        envelope: &OpExecutionPayloadEnvelope,
        version: GossipPayloadVersion,
    ) -> Result<Self, HandlerEncodeError> {
        let version_matches = matches!(
            (envelope, version),
            (OpExecutionPayloadEnvelope::V1(_), GossipPayloadVersion::V1) |
                (OpExecutionPayloadEnvelope::V2(_), GossipPayloadVersion::V2) |
                (OpExecutionPayloadEnvelope::V3 { .. }, GossipPayloadVersion::V3) |
                (OpExecutionPayloadEnvelope::V4 { .. }, GossipPayloadVersion::V4)
        );
        if !version_matches {
            return Err(HandlerEncodeError::WrongPayloadVersion);
        }

        Ok(Self(envelope.as_ssz_bytes().into()))
    }

    /// Decodes these bytes according to the version selected by the gossip topic.
    pub(crate) fn decode(
        self,
        version: GossipPayloadVersion,
    ) -> Result<OpExecutionPayloadEnvelope, GossipPayloadDecodeError> {
        let bytes = self.0.as_ref();
        match version {
            GossipPayloadVersion::V1 => {
                Ok(OpExecutionPayloadEnvelope::V1(ExecutionPayloadV1::from_ssz_bytes(bytes)?))
            }
            GossipPayloadVersion::V2 => {
                Ok(OpExecutionPayloadEnvelope::V2(ExecutionPayloadV2::from_ssz_bytes(bytes)?))
            }
            GossipPayloadVersion::V3 => {
                let (parent_beacon_block_root, payload) = decode_parent_beacon_block_root(bytes)?;
                Ok(OpExecutionPayloadEnvelope::V3 {
                    payload: ExecutionPayloadV3::from_ssz_bytes(payload)?,
                    parent_beacon_block_root,
                })
            }
            GossipPayloadVersion::V4 => {
                let (parent_beacon_block_root, payload) = decode_parent_beacon_block_root(bytes)?;
                Ok(OpExecutionPayloadEnvelope::V4 {
                    payload: OpExecutionPayloadV4::from_ssz_bytes(payload)?,
                    parent_beacon_block_root,
                })
            }
        }
    }

    /// Returns the hash authenticated by the unsafe block signer.
    pub(crate) fn payload_hash(&self) -> PayloadHash {
        PayloadHash::from(self.0.as_ref())
    }
}

/// The decompressed bytes carried by an OP Stack block gossip message.
///
/// The gossip topic determines how to interpret [`Self::envelope`] and is deliberately not part of
/// this type because it is not included in the signed payload bytes.
#[derive(Clone, Debug, PartialEq, Eq)]
pub(crate) struct SignedGossipPayload {
    signature: Signature,
    envelope: EncodedExecutionPayloadEnvelope,
}

impl SignedGossipPayload {
    /// Decodes the Snappy framing and separates the signature from the exact signed envelope bytes.
    pub(crate) fn decode_snappy(data: &[u8]) -> Result<Self, GossipPayloadDecodeError> {
        let decompressed: Bytes = snap::raw::Decoder::new().decompress_vec(data)?.into();
        if decompressed.len() < 66 {
            return Err(GossipPayloadDecodeError::InvalidLength);
        }

        let signature = Signature::try_from(&decompressed[..65])?;
        Ok(Self { signature, envelope: EncodedExecutionPayloadEnvelope(decompressed.slice(65..)) })
    }

    /// Constructs a signed payload from a typed envelope for outbound gossip.
    pub(crate) fn from_envelope(
        envelope: &OpExecutionPayloadEnvelope,
        signature: Signature,
        version: GossipPayloadVersion,
    ) -> Result<Self, HandlerEncodeError> {
        Ok(Self {
            signature,
            envelope: EncodedExecutionPayloadEnvelope::encode(envelope, version)?,
        })
    }

    /// Encodes this signed payload using the Snappy wire framing.
    pub(crate) fn encode_snappy(&self) -> Result<Vec<u8>, HandlerEncodeError> {
        let mut data = Vec::with_capacity(65 + self.envelope.0.len());
        let mut signature = self.signature.as_bytes();
        signature[64] = self.signature.v() as u8;
        data.extend_from_slice(&signature);
        data.extend_from_slice(self.envelope.0.as_ref());
        Ok(snap::raw::Encoder::new().compress_vec(&data)?)
    }

    /// Returns the parsed block signature.
    pub(crate) const fn signature(&self) -> &Signature {
        &self.signature
    }

    /// Returns the hash authenticated by the block signature.
    pub(crate) fn payload_hash(&self) -> PayloadHash {
        self.envelope.payload_hash()
    }

    /// Decodes the signed envelope bytes according to the gossip topic version.
    pub(crate) fn into_envelope(
        self,
        version: GossipPayloadVersion,
    ) -> Result<OpExecutionPayloadEnvelope, GossipPayloadDecodeError> {
        self.envelope.decode(version)
    }
}

fn decode_parent_beacon_block_root(
    bytes: &[u8],
) -> Result<(B256, &[u8]), GossipPayloadDecodeError> {
    if bytes.len() < B256::len_bytes() {
        return Err(GossipPayloadDecodeError::InvalidLength);
    }

    let (root, payload) = bytes.split_at(B256::len_bytes());
    Ok((B256::from_slice(root), payload))
}

/// An error encountered while decoding a signed gossip payload.
#[derive(Clone, Debug, PartialEq, Eq, thiserror::Error)]
pub(crate) enum GossipPayloadDecodeError {
    /// The Snappy framing is invalid.
    #[error("Broken Snappy encoding")]
    BrokenSnappyEncoding,
    /// The signature bytes are invalid.
    #[error("Invalid signature")]
    InvalidSignature,
    /// The execution payload SSZ is invalid.
    #[error("Broken SSZ encoding")]
    BrokenSszEncoding,
    /// The signed payload is too short.
    #[error("Invalid payload length")]
    InvalidLength,
}

impl From<snap::Error> for GossipPayloadDecodeError {
    fn from(_: snap::Error) -> Self {
        Self::BrokenSnappyEncoding
    }
}

impl From<alloy_primitives::SignatureError> for GossipPayloadDecodeError {
    fn from(_: alloy_primitives::SignatureError) -> Self {
        Self::InvalidSignature
    }
}

impl From<ssz::DecodeError> for GossipPayloadDecodeError {
    fn from(_: ssz::DecodeError) -> Self {
        Self::BrokenSszEncoding
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use alloy_primitives::hex;

    fn assert_roundtrip(data: &[u8], version: GossipPayloadVersion, timestamp: u64) {
        let signed = SignedGossipPayload::decode_snappy(data).unwrap();
        let payload_hash = signed.payload_hash();
        let signature = *signed.signature();
        let envelope = signed.into_envelope(version).unwrap();

        assert_eq!(envelope.timestamp(), timestamp);
        assert_eq!(envelope.payload_hash(), payload_hash);

        let encoded = SignedGossipPayload::from_envelope(&envelope, signature, version)
            .unwrap()
            .encode_snappy()
            .unwrap();
        assert_eq!(encoded, data);
    }

    #[test]
    fn roundtrip_v1_golden() {
        let data = hex::decode("0xbd04f043128457c6ccf35128497167442bcc0f8cce78cda8b366e6a12e526d938d1e4c1046acffffbfc542a7e212bb7d80d3a4b2f84f7b196d935398a24eb84c519789b401000000fe0300fe0300fe0300fe0300fe0300fe0300a203000c4a8fd56621ad04fc0101067601008ce60be0005b220117c32c0f3b394b346c2aa42cfa8157cd41f891aa0bec485a62fc010000").unwrap();
        assert_roundtrip(&data, GossipPayloadVersion::V1, 1_725_271_882);
    }

    #[test]
    fn roundtrip_v2_golden() {
        let data = hex::decode("0xc104f0433805080eb36c0b130a7cc1dc74c3f721af4e249aa6f61bb89d1557143e971bb738a3f3b98df7c457e74048e9d2d7e5cd82bb45e3760467e2270e9db86d1271a700000000fe0300fe0300fe0300fe0300fe0300fe0300a203000c6b89d46525ad000205067201009cda69cb5b9b73fc4eb2458b37d37f04ff507fe6c9cd2ab704a05ea9dae3cd61760002000000020000").unwrap();
        assert_roundtrip(&data, GossipPayloadVersion::V2, 1_708_427_627);
    }

    #[test]
    fn roundtrip_v3_golden() {
        let data = hex::decode("0xf104f0434442b9eb38b259f5b23826e6b623e829d2fb878dac70187a1aecf42a3f9bedfd29793d1fcb5822324be0d3e12340a95855553a65d64b83e5579dffb31470df5d010000006a03000412346a1d00fe0100fe0100fe0100fe0100fe0100fe01004201000cc588d465219504100201067601007cfece77b89685f60e3663b6e0faf2de0734674eb91339700c4858c773a8ff921e014401043e0100").unwrap();
        assert_roundtrip(&data, GossipPayloadVersion::V3, 1_708_427_461);
    }

    #[test]
    fn roundtrip_v4_golden() {
        let data = hex::decode("0x9105f043cee25401b6853202950d1d8a082f31a80c4fef5782c049a731f5d104b1b9b9aa7618605b420438ae98b44c8aaaebd482854473c2ae57c079286bb634bece5210000000006a03000412346a1d00fe0100fe0100fe0100fe0100fe0100fe01004201000c5766d26721950430020106f6010001440104b60100049876").unwrap();
        assert_roundtrip(&data, GossipPayloadVersion::V4, 1_741_842_007);
    }

    #[test]
    fn rejects_topic_version_mismatch_when_encoding() {
        let data = hex::decode("0xbd04f043128457c6ccf35128497167442bcc0f8cce78cda8b366e6a12e526d938d1e4c1046acffffbfc542a7e212bb7d80d3a4b2f84f7b196d935398a24eb84c519789b401000000fe0300fe0300fe0300fe0300fe0300fe0300a203000c4a8fd56621ad04fc0101067601008ce60be0005b220117c32c0f3b394b346c2aa42cfa8157cd41f891aa0bec485a62fc010000").unwrap();
        let signed = SignedGossipPayload::decode_snappy(&data).unwrap();
        let signature = *signed.signature();
        let envelope = signed.into_envelope(GossipPayloadVersion::V1).unwrap();

        assert!(matches!(
            SignedGossipPayload::from_envelope(&envelope, signature, GossipPayloadVersion::V2),
            Err(HandlerEncodeError::WrongPayloadVersion)
        ));
    }
}
