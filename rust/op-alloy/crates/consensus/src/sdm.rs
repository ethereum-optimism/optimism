//! Post-execution transaction types.

use alloc::vec::Vec;
#[cfg(feature = "serde")]
use alloc::{format, string::String};
use alloy_consensus::{Sealable, Transaction, Typed2718, transaction::RlpEcdsaEncodableTx};
use alloy_eips::{
    eip2718::{Decodable2718, Eip2718Error, Eip2718Result, Encodable2718, IsTyped2718},
    eip2930::AccessList,
};
use alloy_primitives::{B256, Bytes, ChainId, TxHash, TxKind, U256, keccak256};
use alloy_rlp::{BufMut, Decodable, Encodable, Header};

/// Type byte for the post-execution transaction.
pub const POST_EXEC_TX_TYPE_ID: u8 = 0x7D;

/// Per-transaction gas refund entry within a [`PostExecPayload`].
#[derive(Debug, Clone, PartialEq, Eq, Hash, Default)]
#[cfg_attr(feature = "arbitrary", derive(arbitrary::Arbitrary))]
#[cfg_attr(feature = "serde", derive(serde::Serialize, serde::Deserialize))]
#[cfg_attr(feature = "serde", serde(rename_all = "camelCase"))]
pub struct SDMGasEntry {
    /// Transaction index within the block.
    pub index: u64,
    /// Gas refund from post-execution warming settlement.
    pub gas_refund: u64,
}

impl Encodable for SDMGasEntry {
    fn encode(&self, out: &mut dyn BufMut) {
        let list_len = self.index.length() + self.gas_refund.length();
        Header { list: true, payload_length: list_len }.encode(out);
        self.index.encode(out);
        self.gas_refund.encode(out);
    }

    fn length(&self) -> usize {
        let list_len = self.index.length() + self.gas_refund.length();
        Header { list: true, payload_length: list_len }.length() + list_len
    }
}

impl Decodable for SDMGasEntry {
    fn decode(buf: &mut &[u8]) -> alloy_rlp::Result<Self> {
        let header = Header::decode(buf)?;
        if !header.list {
            return Err(alloy_rlp::Error::UnexpectedString);
        }
        let remaining = buf.len();
        let this = Self { index: Decodable::decode(buf)?, gas_refund: Decodable::decode(buf)? };
        if buf.len() + header.payload_length != remaining {
            return Err(alloy_rlp::Error::UnexpectedLength);
        }
        Ok(this)
    }
}

/// Payload for the post-execution transaction.
///
/// Today this only carries the SDM gas refund data, but additional post-exec fields may
/// be added in the future.
#[derive(Debug, Clone, PartialEq, Eq, Hash, Default)]
#[cfg_attr(feature = "arbitrary", derive(arbitrary::Arbitrary))]
#[cfg_attr(feature = "serde", derive(serde::Serialize, serde::Deserialize))]
#[cfg_attr(feature = "serde", serde(rename_all = "camelCase"))]
pub struct PostExecPayload {
    /// Format version.
    pub version: u64,
    /// L2 block number this synthetic payload is anchored to.
    pub block_number: u64,
    /// Initial SDM gas refund entries keyed by transaction index.
    pub gas_refund_entries: Vec<SDMGasEntry>,
}

impl PostExecPayload {
    /// Look up refund for a given tx index.
    pub fn gas_refund_for_idx(&self, index: u64) -> Option<u64> {
        self.gas_refund_entries.iter().find(|e| e.index == index).map(|e| e.gas_refund)
    }

    /// RLP-encode the payload into bytes.
    pub fn to_rlp_bytes(&self) -> Bytes {
        let mut buf = Vec::new();
        self.encode(&mut buf);
        buf.into()
    }

    /// Decode a payload from RLP bytes.
    pub fn from_rlp_bytes(data: &[u8]) -> alloy_rlp::Result<Self> {
        let mut buf = data;
        let payload = Self::decode(&mut buf)?;
        if !buf.is_empty() {
            return Err(alloy_rlp::Error::UnexpectedLength);
        }
        Ok(payload)
    }
}

impl Encodable for PostExecPayload {
    fn encode(&self, out: &mut dyn BufMut) {
        let list_len =
            self.version.length() + self.block_number.length() + self.gas_refund_entries.length();
        Header { list: true, payload_length: list_len }.encode(out);
        self.version.encode(out);
        self.block_number.encode(out);
        self.gas_refund_entries.encode(out);
    }

    fn length(&self) -> usize {
        let list_len =
            self.version.length() + self.block_number.length() + self.gas_refund_entries.length();
        Header { list: true, payload_length: list_len }.length() + list_len
    }
}

impl Decodable for PostExecPayload {
    fn decode(buf: &mut &[u8]) -> alloy_rlp::Result<Self> {
        let header = Header::decode(buf)?;
        if !header.list {
            return Err(alloy_rlp::Error::UnexpectedString);
        }
        let remaining = buf.len();
        let this = Self {
            version: Decodable::decode(buf)?,
            block_number: Decodable::decode(buf)?,
            gas_refund_entries: Decodable::decode(buf)?,
        };
        if buf.len() + header.payload_length != remaining {
            return Err(alloy_rlp::Error::UnexpectedLength);
        }
        Ok(this)
    }
}

/// Post-execution transaction carrying a [`PostExecPayload`].
#[derive(Debug, Clone, PartialEq, Eq, Hash, Default)]
#[cfg_attr(feature = "serde", derive(serde::Serialize, serde::Deserialize))]
#[cfg_attr(
    feature = "serde",
    serde(into = "TxPostExecSerdeHelper", try_from = "TxPostExecSerdeHelper")
)]
pub struct TxPostExec {
    /// Decoded payload.
    pub payload: PostExecPayload,
    /// RLP-encoded payload bytes used as the transaction input.
    pub input: Bytes,
}

#[cfg(feature = "arbitrary")]
impl<'a> arbitrary::Arbitrary<'a> for TxPostExec {
    fn arbitrary(u: &mut arbitrary::Unstructured<'a>) -> arbitrary::Result<Self> {
        Ok(Self::new(PostExecPayload::arbitrary(u)?))
    }
}

impl TxPostExec {
    /// Construct a post-exec transaction from its decoded payload.
    pub fn new(payload: PostExecPayload) -> Self {
        let input = payload.to_rlp_bytes();
        Self { payload, input }
    }

    /// Returns the canonical signature for post-exec transactions, which don't include a
    /// signature.
    pub const fn signature() -> alloy_primitives::Signature {
        alloy_primitives::Signature::new(U256::ZERO, U256::ZERO, false)
    }

    /// Encoded length of the transaction body.
    pub fn rlp_encoded_length(&self) -> usize {
        self.input.len()
    }

    /// Encoded length including the type byte.
    pub fn eip2718_encoded_length(&self) -> usize {
        self.rlp_encoded_length() + 1
    }

    fn network_header(&self) -> Header {
        Header { list: false, payload_length: self.eip2718_encoded_length() }
    }

    /// Encoded length including the outer network RLP header.
    pub fn network_encoded_length(&self) -> usize {
        self.network_header().length_with_payload()
    }

    /// Network encode the transaction.
    pub fn network_encode(&self, out: &mut dyn BufMut) {
        self.network_header().encode(out);
        self.encode_2718(out);
    }

    /// Calculates a heuristic for the in-memory size of the transaction.
    pub fn size(&self) -> usize {
        core::mem::size_of::<PostExecPayload>() +
            self.input.len() +
            self.payload.gas_refund_entries.len() * core::mem::size_of::<SDMGasEntry>()
    }

    /// Calculate the transaction hash.
    pub fn tx_hash(&self) -> TxHash {
        let mut buf = Vec::with_capacity(self.eip2718_encoded_length());
        self.encode_2718(&mut buf);
        keccak256(&buf)
    }
}

impl Typed2718 for TxPostExec {
    fn ty(&self) -> u8 {
        POST_EXEC_TX_TYPE_ID
    }
}

impl IsTyped2718 for TxPostExec {
    fn is_type(ty: u8) -> bool {
        ty == POST_EXEC_TX_TYPE_ID
    }
}

impl RlpEcdsaEncodableTx for TxPostExec {
    fn rlp_encoded_fields_length(&self) -> usize {
        self.input.len()
    }

    fn rlp_encode_fields(&self, out: &mut dyn alloy_rlp::BufMut) {
        // `input` already stores the canonical RLP-encoded payload, so the transaction fields are
        // written as-is instead of being wrapped in an additional RLP list.
        out.put_slice(self.input.as_ref());
    }
}

impl Transaction for TxPostExec {
    fn chain_id(&self) -> Option<ChainId> {
        None
    }
    fn nonce(&self) -> u64 {
        0
    }
    fn gas_limit(&self) -> u64 {
        0
    }
    fn gas_price(&self) -> Option<u128> {
        None
    }
    fn max_fee_per_gas(&self) -> u128 {
        0
    }
    fn max_priority_fee_per_gas(&self) -> Option<u128> {
        None
    }
    fn max_fee_per_blob_gas(&self) -> Option<u128> {
        None
    }
    fn priority_fee_or_price(&self) -> u128 {
        0
    }
    fn effective_gas_price(&self, _: Option<u64>) -> u128 {
        0
    }
    fn is_dynamic_fee(&self) -> bool {
        false
    }
    fn kind(&self) -> TxKind {
        // Post-exec transactions do not carry a destination like deposits do, so expose them as a
        // synthetic zero-address call placeholder.
        TxKind::Call(Default::default())
    }
    fn is_create(&self) -> bool {
        false
    }
    fn value(&self) -> U256 {
        U256::ZERO
    }
    fn input(&self) -> &Bytes {
        &self.input
    }
    fn access_list(&self) -> Option<&AccessList> {
        None
    }
    fn blob_versioned_hashes(&self) -> Option<&[B256]> {
        None
    }
    fn authorization_list(&self) -> Option<&[alloy_eips::eip7702::SignedAuthorization]> {
        None
    }
}

impl Encodable2718 for TxPostExec {
    fn type_flag(&self) -> Option<u8> {
        Some(POST_EXEC_TX_TYPE_ID)
    }
    fn encode_2718_len(&self) -> usize {
        self.eip2718_encoded_length()
    }
    fn encode_2718(&self, out: &mut dyn alloy_rlp::BufMut) {
        out.put_u8(POST_EXEC_TX_TYPE_ID);
        out.put_slice(self.input.as_ref());
    }
}

impl Decodable2718 for TxPostExec {
    fn typed_decode(ty: u8, data: &mut &[u8]) -> Eip2718Result<Self> {
        if ty != POST_EXEC_TX_TYPE_ID {
            return Err(Eip2718Error::UnexpectedType(ty));
        }
        Ok(Self::new(PostExecPayload::decode(data)?))
    }

    fn fallback_decode(data: &mut &[u8]) -> Eip2718Result<Self> {
        Ok(Self::new(PostExecPayload::decode(data)?))
    }
}

impl Encodable for TxPostExec {
    fn encode(&self, out: &mut dyn BufMut) {
        out.put_slice(self.input.as_ref());
    }

    fn length(&self) -> usize {
        self.rlp_encoded_length()
    }
}

impl Decodable for TxPostExec {
    fn decode(data: &mut &[u8]) -> alloy_rlp::Result<Self> {
        Ok(Self::new(PostExecPayload::decode(data)?))
    }
}

impl Sealable for TxPostExec {
    fn hash_slow(&self) -> B256 {
        self.tx_hash()
    }
}

#[cfg(feature = "serde")]
#[derive(Debug, Clone, serde::Serialize, serde::Deserialize)]
#[serde(rename_all = "camelCase")]
struct TxPostExecSerdeHelper {
    #[serde(default, skip_serializing_if = "Option::is_none", with = "alloy_serde::quantity::opt")]
    gas: Option<u64>,
    #[serde(default, skip_serializing_if = "Option::is_none", with = "alloy_serde::quantity::opt")]
    value: Option<u128>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    input: Option<Bytes>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    payload: Option<PostExecPayload>,
}

#[cfg(feature = "serde")]
impl From<TxPostExec> for TxPostExecSerdeHelper {
    fn from(value: TxPostExec) -> Self {
        // Emit both the canonical wire-format bytes and the decoded payload so JSON is both
        // round-trippable and human-readable.
        Self {
            gas: Some(0),
            value: Some(0),
            input: Some(value.input),
            payload: Some(value.payload),
        }
    }
}

#[cfg(feature = "serde")]
impl TryFrom<TxPostExecSerdeHelper> for TxPostExec {
    type Error = String;

    fn try_from(value: TxPostExecSerdeHelper) -> Result<Self, Self::Error> {
        if value.gas.is_some_and(|gas| gas != 0) {
            return Err("post-exec transaction gas must be 0".into());
        }
        if value.value.is_some_and(|value| value != 0) {
            return Err("post-exec transaction value must be 0".into());
        }

        // Accept either the raw input bytes or a decoded payload. If both are supplied, they must
        // describe the same payload.
        let input_payload = value
            .input
            .as_ref()
            .map(|input| {
                PostExecPayload::from_rlp_bytes(input.as_ref())
                    .map_err(|err| format!("invalid post-exec transaction input: {err}"))
            })
            .transpose()?;

        let payload = match (input_payload, value.payload) {
            (Some(input_payload), Some(payload)) => {
                if input_payload != payload {
                    return Err("post-exec transaction input and payload mismatch".into());
                }
                input_payload
            }
            (Some(payload), None) | (None, Some(payload)) => payload,
            (None, None) => return Err("missing post-exec transaction input or payload".into()),
        };

        Ok(Self::new(payload))
    }
}

#[cfg(feature = "alloy-compat")]
impl From<TxPostExec> for alloy_rpc_types_eth::TransactionRequest {
    fn from(tx: TxPostExec) -> Self {
        Self {
            from: Some(alloy_primitives::Address::ZERO),
            transaction_type: Some(POST_EXEC_TX_TYPE_ID),
            gas: Some(0),
            nonce: Some(0),
            value: Some(U256::ZERO),
            input: tx.input.into(),
            ..Default::default()
        }
    }
}

/// Build a post-execution transaction from a block number and refund entries.
pub fn build_post_exec_tx(block_number: u64, gas_refund_entries: Vec<SDMGasEntry>) -> TxPostExec {
    TxPostExec::new(PostExecPayload { version: 1, block_number, gas_refund_entries })
}

/// Check if a transaction type byte identifies a post-exec transaction.
pub const fn is_post_exec_tx(ty: u8) -> bool {
    ty == POST_EXEC_TX_TYPE_ID
}

#[cfg(test)]
mod tests {
    use super::*;

    #[cfg(feature = "serde")]
    use serde_json::json;

    #[test]
    fn post_exec_payload_rlp_roundtrip_preserves_block_number() {
        let payload = PostExecPayload {
            version: 1,
            block_number: 42,
            gas_refund_entries: vec![SDMGasEntry { index: 3, gas_refund: 7 }],
        };

        let encoded = payload.to_rlp_bytes();
        let decoded = PostExecPayload::from_rlp_bytes(encoded.as_ref()).expect("decode payload");

        assert_eq!(decoded, payload);
    }

    #[test]
    fn post_exec_payload_rlp_decode_rejects_trailing_bytes() {
        let payload = PostExecPayload {
            version: 1,
            block_number: 42,
            gas_refund_entries: vec![SDMGasEntry { index: 3, gas_refund: 7 }],
        };

        let mut encoded = payload.to_rlp_bytes().to_vec();
        encoded.push(0);

        let err = PostExecPayload::from_rlp_bytes(&encoded).expect_err("reject trailing bytes");
        assert_eq!(err, alloy_rlp::Error::UnexpectedLength);
    }

    #[test]
    fn post_exec_tx_hash_depends_on_block_number() {
        let entries = vec![SDMGasEntry { index: 3, gas_refund: 7 }];
        let tx_a = build_post_exec_tx(42, entries.clone());
        let tx_b = build_post_exec_tx(43, entries);

        assert_ne!(tx_a.tx_hash(), tx_b.tx_hash());
    }

    #[cfg(feature = "serde")]
    #[test]
    fn post_exec_tx_serde_emits_input_and_payload() {
        let tx = build_post_exec_tx(42, vec![SDMGasEntry { index: 3, gas_refund: 7 }]);
        let value = serde_json::to_value(&tx).expect("serialize tx");

        assert!(value.get("input").is_some());
        assert!(value.get("payload").is_some());
    }

    #[cfg(feature = "serde")]
    #[test]
    fn post_exec_tx_serde_accepts_matching_input_and_payload() {
        let tx = build_post_exec_tx(42, vec![SDMGasEntry { index: 3, gas_refund: 7 }]);
        let value = serde_json::to_value(&tx).expect("serialize tx");

        let decoded: TxPostExec = serde_json::from_value(value).expect("deserialize tx");
        assert_eq!(decoded, tx);
    }

    #[cfg(feature = "serde")]
    #[test]
    fn post_exec_tx_serde_accepts_input_only() {
        let tx = build_post_exec_tx(42, vec![SDMGasEntry { index: 3, gas_refund: 7 }]);
        let mut value = serde_json::to_value(&tx).expect("serialize tx");
        value.as_object_mut().expect("object").remove("payload");

        let decoded: TxPostExec = serde_json::from_value(value).expect("deserialize tx");
        assert_eq!(decoded, tx);
    }

    #[cfg(feature = "serde")]
    #[test]
    fn post_exec_tx_serde_accepts_payload_only() {
        let tx = build_post_exec_tx(42, vec![SDMGasEntry { index: 3, gas_refund: 7 }]);
        let mut value = serde_json::to_value(&tx).expect("serialize tx");
        value.as_object_mut().expect("object").remove("input");

        let decoded: TxPostExec = serde_json::from_value(value).expect("deserialize tx");
        assert_eq!(decoded, tx);
    }

    #[cfg(feature = "serde")]
    #[test]
    fn post_exec_tx_serde_rejects_conflicting_input_and_payload() {
        let tx = build_post_exec_tx(42, vec![SDMGasEntry { index: 3, gas_refund: 7 }]);
        let value = json!({
            "gas": "0x0",
            "value": "0x0",
            "input": tx.input,
            "payload": {
                "version": 1,
                "blockNumber": 43,
                "gasRefundEntries": [{ "index": 3, "gasRefund": 7 }]
            }
        });

        let err = serde_json::from_value::<TxPostExec>(value).expect_err("reject mismatch");
        assert!(err.to_string().contains("input and payload mismatch"));
    }

    #[cfg(feature = "serde")]
    #[test]
    fn post_exec_tx_serde_surfaces_input_decode_error() {
        let value = json!({
            "gas": "0x0",
            "value": "0x0",
            "input": "0xc0"
        });

        let err = serde_json::from_value::<TxPostExec>(value).expect_err("reject invalid input");
        assert!(err.to_string().contains("invalid post-exec transaction input"));
    }
}
