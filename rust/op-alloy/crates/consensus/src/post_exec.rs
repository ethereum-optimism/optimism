//! Post-execution transaction types.

use alloc::{
    string::{String, ToString},
    vec::Vec,
};
#[cfg(feature = "serde")]
use alloy_consensus::Sealed;
use alloy_consensus::{Sealable, Transaction, Typed2718, transaction::RlpEcdsaEncodableTx};
use alloy_eips::{
    eip2718::{Decodable2718, Eip2718Error, Eip2718Result, Encodable2718, IsTyped2718},
    eip2930::AccessList,
};
use alloy_primitives::{Address, B256, Bytes, ChainId, TxHash, TxKind, U256, keccak256};
use alloy_rlp::{BufMut, Decodable, Encodable, Header, RlpDecodable, RlpEncodable};

use crate::OpTransaction;

/// Type byte for the post-execution transaction.
pub const POST_EXEC_TX_TYPE_ID: u8 = 0x7D;

/// Current format version for [`PostExecPayload`].
pub const POST_EXEC_PAYLOAD_VERSION: u8 = 1;

/// Per-transaction gas refund entry within a [`PostExecPayload`].
#[derive(Debug, Clone, PartialEq, Eq, Hash, Default, RlpEncodable, RlpDecodable)]
#[cfg_attr(feature = "arbitrary", derive(arbitrary::Arbitrary))]
#[cfg_attr(feature = "serde", derive(serde::Serialize, serde::Deserialize))]
#[cfg_attr(feature = "serde", serde(rename_all = "camelCase"))]
pub struct SDMGasEntry {
    /// Transaction index within the block.
    pub index: u64,
    /// Gas refund from post-execution warming settlement.
    pub gas_refund: u64,
}

/// Payload for the post-execution transaction.
///
/// Today this only carries the SDM gas refund data, but additional post-exec fields may
/// be added in the future.
#[derive(Debug, Clone, PartialEq, Eq, Hash, Default, RlpEncodable, RlpDecodable)]
#[cfg_attr(feature = "serde", derive(serde::Serialize, serde::Deserialize))]
#[cfg_attr(feature = "serde", serde(rename_all = "camelCase"))]
pub struct PostExecPayload {
    /// Format version.
    pub version: u8,
    /// L2 block number this post-execution payload is anchored to.
    ///
    /// This is included in the encoded post-exec transaction so otherwise identical
    /// payloads in different blocks produce distinct transaction hashes.
    pub block_number: u64,
    /// Initial SDM gas refund entries keyed by transaction index.
    pub gas_refund_entries: Vec<SDMGasEntry>,
}

// `version` is pinned rather than left arbitrary because `decode_checked` rejects any
// non-`POST_EXEC_PAYLOAD_VERSION` value, which would break encode/decode roundtrip
// property tests (and any downstream fuzzer using arbitrary-generated payloads).
#[cfg(feature = "arbitrary")]
impl<'a> arbitrary::Arbitrary<'a> for PostExecPayload {
    fn arbitrary(u: &mut arbitrary::Unstructured<'a>) -> arbitrary::Result<Self> {
        Ok(Self {
            version: POST_EXEC_PAYLOAD_VERSION,
            block_number: u64::arbitrary(u)?,
            gas_refund_entries: <Vec<SDMGasEntry>>::arbitrary(u)?,
        })
    }
}

/// Parsed post-exec transaction metadata for a block.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct ParsedPostExecPayload {
    /// Transaction index of the post-exec transaction within the block.
    pub tx_index: u64,
    /// Decoded post-exec payload.
    pub payload: PostExecPayload,
}

/// Errors returned while validating a block's post-exec transaction structure.
#[derive(Debug, Clone, Copy, PartialEq, Eq, thiserror::Error)]
pub enum PostExecPayloadValidationError {
    /// The block contains a post-exec transaction before SDM is active.
    #[error("unexpected post-exec transaction at index {tx_index}: SDM not active for this block")]
    UnexpectedPostExecTx {
        /// Transaction index.
        tx_index: u64,
    },
    /// The block contains more than one post-exec transaction.
    #[error(
        "multiple post-exec transactions: first at index {first_index}, duplicate at index {duplicate_index}"
    )]
    MultiplePostExecTxs {
        /// First post-exec tx index.
        first_index: u64,
        /// Duplicate post-exec tx index.
        duplicate_index: u64,
    },
    /// The post-exec transaction is not the final transaction in the block.
    #[error(
        "post-exec transaction at index {tx_index} must be the final transaction; final index is {last_index}"
    )]
    PostExecTxNotLast {
        /// Post-exec tx index.
        tx_index: u64,
        /// Final transaction index.
        last_index: u64,
    },
    /// The payload block number does not match the containing block.
    #[error(
        "payload block number {payload_block_number} does not match block number {block_number}"
    )]
    BlockNumberMismatch {
        /// Block number encoded in the payload.
        payload_block_number: u64,
        /// Containing block number.
        block_number: u64,
    },
}

impl PostExecPayloadValidationError {
    /// Returns this error as an owned string.
    #[must_use]
    pub fn into_string(self) -> String {
        self.to_string()
    }
}

/// Parse and validate the block-level post-exec transaction, if present.
///
/// This enforces the shared consensus structure rules: post-exec transactions are only valid after
/// activation, at most one may be present, and when present it must be the final transaction and be
/// anchored to the containing block number.
///
/// # Errors
///
/// Returns [`PostExecPayloadValidationError`] if the post-exec transaction is not valid for the
/// block or SDM activation state.
pub fn parse_post_exec_payload_from_transactions<'a, I, T>(
    transactions: I,
    block_number: u64,
    sdm_active: bool,
) -> Result<Option<ParsedPostExecPayload>, PostExecPayloadValidationError>
where
    I: IntoIterator<Item = &'a T>,
    T: OpTransaction + 'a,
{
    let mut parsed = None::<ParsedPostExecPayload>;
    let mut last_index = None::<u64>;

    for (idx, tx) in transactions.into_iter().enumerate() {
        let tx_index = idx as u64;
        last_index = Some(tx_index);

        let Some(post_exec) = tx.as_post_exec() else {
            continue;
        };

        if !sdm_active {
            return Err(PostExecPayloadValidationError::UnexpectedPostExecTx { tx_index });
        }

        if let Some(first) = &parsed {
            return Err(PostExecPayloadValidationError::MultiplePostExecTxs {
                first_index: first.tx_index,
                duplicate_index: tx_index,
            });
        }

        let payload = post_exec.inner().payload.clone();
        if payload.block_number != block_number {
            return Err(PostExecPayloadValidationError::BlockNumberMismatch {
                payload_block_number: payload.block_number,
                block_number,
            });
        }

        parsed = Some(ParsedPostExecPayload { tx_index, payload });
    }

    if let (Some(parsed), Some(last_index)) = (&parsed, last_index) &&
        parsed.tx_index != last_index
    {
        return Err(PostExecPayloadValidationError::PostExecTxNotLast {
            tx_index: parsed.tx_index,
            last_index,
        });
    }

    Ok(parsed)
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

    /// Decode a payload from an RLP stream, validating the payload version.
    ///
    /// Advances `buf` past the consumed bytes. Unlike [`Self::from_rlp_bytes`], trailing bytes
    /// are left in place for the caller to consume; this is the decoder to use on the EIP-2718
    /// path where the envelope already framed the packet exactly.
    pub fn decode_checked(buf: &mut &[u8]) -> alloy_rlp::Result<Self> {
        let payload = Self::decode(buf)?;
        if payload.version != POST_EXEC_PAYLOAD_VERSION {
            return Err(alloy_rlp::Error::Custom("unsupported post-exec payload version"));
        }
        Ok(payload)
    }

    /// Decode a payload from RLP bytes.
    ///
    /// Rejects payloads whose `version` is not [`POST_EXEC_PAYLOAD_VERSION`] and rejects any
    /// trailing bytes after the RLP structure.
    pub fn from_rlp_bytes(data: &[u8]) -> alloy_rlp::Result<Self> {
        let mut buf = data;
        let payload = Self::decode_checked(&mut buf)?;
        if !buf.is_empty() {
            return Err(alloy_rlp::Error::UnexpectedLength);
        }
        Ok(payload)
    }
}

/// Post-execution transaction carrying a [`PostExecPayload`].
#[derive(Debug, Clone, PartialEq, Eq, Hash, Default)]
#[cfg_attr(feature = "serde", derive(serde::Serialize, serde::Deserialize))]
#[cfg_attr(feature = "serde", serde(into = "PostExecPayload", from = "PostExecPayload"))]
pub struct TxPostExec {
    /// Decoded payload.
    pub payload: PostExecPayload,
    /// RLP-encoded payload bytes used as the transaction input.
    pub input: Bytes,
}

#[cfg(feature = "arbitrary")]
impl<'a> arbitrary::Arbitrary<'a> for TxPostExec {
    fn arbitrary(u: &mut arbitrary::Unstructured<'a>) -> arbitrary::Result<Self> {
        // Keep `payload` and the cached `input` bytes in sync for fuzzed values.
        Ok(Self::new(PostExecPayload::arbitrary(u)?))
    }
}

impl From<PostExecPayload> for TxPostExec {
    fn from(payload: PostExecPayload) -> Self {
        Self::new(payload)
    }
}

impl From<TxPostExec> for PostExecPayload {
    fn from(tx: TxPostExec) -> Self {
        tx.payload
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

    /// Returns the canonical signer address for post-exec transactions.
    pub const fn signer_address(&self) -> Address {
        Address::ZERO
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
        // zero-address call placeholder.
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
        Ok(Self::new(PostExecPayload::decode_checked(data)?))
    }

    fn fallback_decode(data: &mut &[u8]) -> Eip2718Result<Self> {
        Ok(Self::new(PostExecPayload::decode_checked(data)?))
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
        Ok(Self::new(PostExecPayload::decode_checked(data)?))
    }
}

impl Sealable for TxPostExec {
    fn hash_slow(&self) -> B256 {
        self.tx_hash()
    }
}

#[cfg(feature = "alloy-compat")]
impl From<TxPostExec> for alloy_rpc_types_eth::TransactionRequest {
    fn from(tx: TxPostExec) -> Self {
        Self {
            from: Some(tx.signer_address()),
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
    TxPostExec::new(PostExecPayload {
        version: POST_EXEC_PAYLOAD_VERSION,
        block_number,
        gas_refund_entries,
    })
}

/// Post-exec transactions serialize as full RPC transaction objects when embedded in a
/// [`crate::OpTxEnvelope`] response.
///
/// Unlike the standalone [`TxPostExec`] serde form, RPC consumers such as op-node expect the
/// canonical `input` field to be present so the transaction can be decoded by go-ethereum types.
#[cfg(feature = "serde")]
pub fn serde_post_exec_tx_rpc<S>(
    value: &Sealed<TxPostExec>,
    serializer: S,
) -> Result<S::Ok, S::Error>
where
    S: serde::Serializer,
{
    use serde::Serialize;

    #[derive(Serialize)]
    struct SerdeHelper<'a> {
        hash: B256,
        #[serde(rename = "type", with = "alloy_serde::quantity")]
        tx_type: u8,
        #[serde(with = "alloy_serde::quantity")]
        gas: u64,
        value: U256,
        input: &'a Bytes,
    }

    SerdeHelper {
        hash: value.hash(),
        tx_type: POST_EXEC_TX_TYPE_ID,
        gas: 0,
        value: U256::ZERO,
        input: &value.inner().input,
    }
    .serialize(serializer)
}

#[cfg(test)]
mod tests {
    use super::*;

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
    fn post_exec_payload_rlp_decode_rejects_unknown_version() {
        let payload = PostExecPayload {
            version: POST_EXEC_PAYLOAD_VERSION + 1,
            block_number: 42,
            gas_refund_entries: vec![SDMGasEntry { index: 3, gas_refund: 7 }],
        };

        let encoded = payload.to_rlp_bytes();
        let err =
            PostExecPayload::from_rlp_bytes(encoded.as_ref()).expect_err("reject unknown version");
        assert_eq!(err, alloy_rlp::Error::Custom("unsupported post-exec payload version"));
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

    #[test]
    fn post_exec_tx_eip2718_roundtrip() {
        let tx = build_post_exec_tx(
            99,
            vec![
                SDMGasEntry { index: 0, gas_refund: 100 },
                SDMGasEntry { index: 5, gas_refund: 200 },
            ],
        );

        let mut buf = Vec::new();
        tx.encode_2718(&mut buf);

        let decoded = TxPostExec::decode_2718(&mut buf.as_slice()).expect("decode 2718");
        assert_eq!(decoded, tx);
        assert_eq!(decoded.tx_hash(), tx.tx_hash());
    }

    #[test]
    fn post_exec_tx_eip2718_decode_rejects_unknown_version() {
        let payload = PostExecPayload {
            version: POST_EXEC_PAYLOAD_VERSION + 1,
            block_number: 42,
            gas_refund_entries: vec![SDMGasEntry { index: 3, gas_refund: 7 }],
        };

        let mut buf = Vec::new();
        buf.put_u8(POST_EXEC_TX_TYPE_ID);
        payload.encode(&mut buf);

        let err = TxPostExec::decode_2718(&mut buf.as_slice())
            .expect_err("2718 decode must reject unknown version");
        assert!(
            matches!(
                err,
                Eip2718Error::RlpError(alloy_rlp::Error::Custom(
                    "unsupported post-exec payload version"
                ))
            ),
            "unexpected error: {err:?}"
        );
    }

    #[test]
    fn post_exec_tx_rlp_decode_rejects_unknown_version() {
        let payload = PostExecPayload {
            version: POST_EXEC_PAYLOAD_VERSION + 1,
            block_number: 42,
            gas_refund_entries: vec![SDMGasEntry { index: 3, gas_refund: 7 }],
        };
        let mut buf = Vec::new();
        payload.encode(&mut buf);

        let err = TxPostExec::decode(&mut buf.as_slice())
            .expect_err("rlp decode must reject unknown version");
        assert_eq!(err, alloy_rlp::Error::Custom("unsupported post-exec payload version"));
    }

    #[test]
    fn post_exec_tx_eip2718_roundtrip_empty_refunds() {
        let tx = build_post_exec_tx(1, vec![]);

        let mut buf = Vec::new();
        tx.encode_2718(&mut buf);

        let decoded = TxPostExec::decode_2718(&mut buf.as_slice()).expect("decode 2718");
        assert_eq!(decoded, tx);
    }

    #[cfg(feature = "serde")]
    #[test]
    fn post_exec_tx_serde_serializes_as_payload() {
        let tx = build_post_exec_tx(42, vec![SDMGasEntry { index: 3, gas_refund: 7 }]);
        let value = serde_json::to_value(&tx).expect("serialize tx");

        assert_eq!(value, serde_json::to_value(&tx.payload).expect("serialize payload"));
    }

    #[cfg(feature = "serde")]
    #[test]
    fn post_exec_tx_serde_roundtrip_preserves_cached_input() {
        let tx = build_post_exec_tx(42, vec![SDMGasEntry { index: 3, gas_refund: 7 }]);
        let value = serde_json::to_value(&tx).expect("serialize tx");

        let decoded: TxPostExec = serde_json::from_value(value).expect("deserialize tx");
        assert_eq!(decoded, tx);
        assert_eq!(decoded.input, decoded.payload.to_rlp_bytes());
    }
}
