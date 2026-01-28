//! Deposit Transaction type.

use super::OpTxType;
use alloc::vec::Vec;
use alloy_consensus::{Sealable, Transaction, Typed2718};
use alloy_eips::{
    eip2718::{Decodable2718, Eip2718Error, Eip2718Result, Encodable2718, IsTyped2718},
    eip2930::AccessList,
};
use alloy_primitives::{Address, B256, Bytes, ChainId, Signature, TxHash, TxKind, U256, keccak256};
use alloy_rlp::{BufMut, Decodable, Encodable, Header};
use core::mem;

/// Deposit transactions, also known as deposits are initiated on L1, and executed on L2.
#[derive(Debug, Clone, PartialEq, Eq, Hash, Default)]
#[cfg_attr(feature = "arbitrary", derive(arbitrary::Arbitrary))]
#[cfg_attr(feature = "serde", derive(serde::Serialize, serde::Deserialize))]
#[cfg_attr(feature = "serde", serde(rename_all = "camelCase"))]
pub struct TxDeposit {
    /// Hash that uniquely identifies the source of the deposit.
    pub source_hash: B256,
    /// The address of the sender account.
    pub from: Address,
    /// The address of the recipient account, or the null (zero-length) address if the deposited
    /// transaction is a contract creation.
    #[cfg_attr(feature = "serde", serde(default, skip_serializing_if = "TxKind::is_create"))]
    pub to: TxKind,
    /// The ETH value to mint on L2.
    #[cfg_attr(feature = "serde", serde(default, with = "alloy_serde::quantity"))]
    pub mint: u128,
    ///  The ETH value to send to the recipient account.
    pub value: U256,
    /// The gas limit for the L2 transaction.
    #[cfg_attr(feature = "serde", serde(with = "alloy_serde::quantity", rename = "gas"))]
    pub gas_limit: u64,
    /// Field indicating if this transaction is exempt from the L2 gas limit.
    #[cfg_attr(
        feature = "serde",
        serde(
            default,
            with = "alloy_serde::quantity",
            rename = "isSystemTx",
            skip_serializing_if = "core::ops::Not::not"
        )
    )]
    pub is_system_transaction: bool,
    /// Input has two uses depending if transaction is Create or Call (if `to` field is None or
    /// Some).
    pub input: Bytes,
}

impl TxDeposit {
    /// Decodes the inner [TxDeposit] fields from RLP bytes.
    ///
    /// NOTE: This assumes a RLP header has already been decoded, and _just_ decodes the following
    /// RLP fields in the following order:
    ///
    /// - `source_hash`
    /// - `from`
    /// - `to`
    /// - `mint`
    /// - `value`
    /// - `gas_limit`
    /// - `is_system_transaction`
    /// - `input`
    pub fn rlp_decode_fields(buf: &mut &[u8]) -> alloy_rlp::Result<Self> {
        Ok(Self {
            source_hash: Decodable::decode(buf)?,
            from: Decodable::decode(buf)?,
            to: Decodable::decode(buf)?,
            mint: Decodable::decode(buf)?,
            value: Decodable::decode(buf)?,
            gas_limit: Decodable::decode(buf)?,
            is_system_transaction: Decodable::decode(buf)?,
            input: Decodable::decode(buf)?,
        })
    }

    /// Decodes the transaction from RLP bytes.
    pub fn rlp_decode(buf: &mut &[u8]) -> alloy_rlp::Result<Self> {
        let header = Header::decode(buf)?;
        if !header.list {
            return Err(alloy_rlp::Error::UnexpectedString);
        }
        let remaining = buf.len();

        if header.payload_length > remaining {
            return Err(alloy_rlp::Error::InputTooShort);
        }

        let this = Self::rlp_decode_fields(buf)?;

        if buf.len() + header.payload_length != remaining {
            return Err(alloy_rlp::Error::UnexpectedLength);
        }

        Ok(this)
    }

    /// Outputs the length of the transaction's fields, without a RLP header or length of the
    /// eip155 fields.
    pub(crate) fn rlp_encoded_fields_length(&self) -> usize {
        self.source_hash.length()
            + self.from.length()
            + self.to.length()
            + self.mint.length()
            + self.value.length()
            + self.gas_limit.length()
            + self.is_system_transaction.length()
            + self.input.0.length()
    }

    /// Encodes only the transaction's fields into the desired buffer, without a RLP header.
    /// <https://github.com/ethereum-optimism/specs/blob/main/specs/protocol/deposits.md#the-deposited-transaction-type>
    pub(crate) fn rlp_encode_fields(&self, out: &mut dyn alloy_rlp::BufMut) {
        self.source_hash.encode(out);
        self.from.encode(out);
        self.to.encode(out);
        self.mint.encode(out);
        self.value.encode(out);
        self.gas_limit.encode(out);
        self.is_system_transaction.encode(out);
        self.input.encode(out);
    }

    /// Calculates a heuristic for the in-memory size of the [TxDeposit] transaction.
    #[inline]
    pub fn size(&self) -> usize {
        mem::size_of::<B256>() + // source_hash
        mem::size_of::<Address>() + // from
        self.to.size() + // to
        mem::size_of::<u128>() + // mint
        mem::size_of::<U256>() + // value
        mem::size_of::<u128>() + // gas_limit
        mem::size_of::<bool>() + // is_system_transaction
        self.input.len() // input
    }

    /// Get the transaction type
    pub(crate) const fn tx_type(&self) -> OpTxType {
        OpTxType::Deposit
    }

    /// Create an rlp header for the transaction.
    fn rlp_header(&self) -> Header {
        Header { list: true, payload_length: self.rlp_encoded_fields_length() }
    }

    /// RLP encodes the transaction.
    pub fn rlp_encode(&self, out: &mut dyn BufMut) {
        self.rlp_header().encode(out);
        self.rlp_encode_fields(out);
    }

    /// Get the length of the transaction when RLP encoded.
    pub fn rlp_encoded_length(&self) -> usize {
        self.rlp_header().length_with_payload()
    }

    /// Get the length of the transaction when EIP-2718 encoded. This is the
    /// 1 byte type flag + the length of the RLP encoded transaction.
    pub fn eip2718_encoded_length(&self) -> usize {
        self.rlp_encoded_length() + 1
    }

    fn network_header(&self) -> Header {
        Header { list: false, payload_length: self.eip2718_encoded_length() }
    }

    /// Get the length of the transaction when network encoded. This is the
    /// EIP-2718 encoded length with an outer RLP header.
    pub fn network_encoded_length(&self) -> usize {
        self.network_header().length_with_payload()
    }

    /// Network encode the transaction with the given signature.
    pub fn network_encode(&self, out: &mut dyn BufMut) {
        self.network_header().encode(out);
        self.encode_2718(out);
    }

    /// Calculate the transaction hash.
    pub fn tx_hash(&self) -> TxHash {
        let mut buf = Vec::with_capacity(self.eip2718_encoded_length());
        self.encode_2718(&mut buf);
        keccak256(&buf)
    }

    /// Returns the signature for the optimism deposit transactions, which don't include a
    /// signature.
    pub const fn signature() -> Signature {
        Signature::new(U256::ZERO, U256::ZERO, false)
    }
}

impl Typed2718 for TxDeposit {
    fn ty(&self) -> u8 {
        OpTxType::Deposit as u8
    }
}

impl IsTyped2718 for TxDeposit {
    fn is_type(ty: u8) -> bool {
        OpTxType::Deposit as u8 == ty
    }
}

impl Transaction for TxDeposit {
    fn chain_id(&self) -> Option<ChainId> {
        None
    }

    fn nonce(&self) -> u64 {
        0u64
    }

    fn gas_limit(&self) -> u64 {
        self.gas_limit
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
        self.to
    }

    fn is_create(&self) -> bool {
        self.to.is_create()
    }

    fn value(&self) -> U256 {
        self.value
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

impl Encodable2718 for TxDeposit {
    fn type_flag(&self) -> Option<u8> {
        Some(OpTxType::Deposit as u8)
    }

    fn encode_2718_len(&self) -> usize {
        self.eip2718_encoded_length()
    }

    fn encode_2718(&self, out: &mut dyn alloy_rlp::BufMut) {
        out.put_u8(self.tx_type() as u8);
        self.rlp_encode(out);
    }
}

impl Decodable2718 for TxDeposit {
    fn typed_decode(ty: u8, data: &mut &[u8]) -> Eip2718Result<Self> {
        let ty: OpTxType = ty.try_into().map_err(|_| Eip2718Error::UnexpectedType(ty))?;
        if ty != OpTxType::Deposit as u8 {
            return Err(Eip2718Error::UnexpectedType(ty as u8));
        }
        let tx = Self::decode(data)?;
        Ok(tx)
    }

    fn fallback_decode(data: &mut &[u8]) -> Eip2718Result<Self> {
        let tx = Self::decode(data)?;
        Ok(tx)
    }
}

impl Encodable for TxDeposit {
    fn encode(&self, out: &mut dyn BufMut) {
        Header { list: true, payload_length: self.rlp_encoded_fields_length() }.encode(out);
        self.rlp_encode_fields(out);
    }

    fn length(&self) -> usize {
        let payload_length = self.rlp_encoded_fields_length();
        Header { list: true, payload_length }.length() + payload_length
    }
}

impl Decodable for TxDeposit {
    fn decode(data: &mut &[u8]) -> alloy_rlp::Result<Self> {
        Self::rlp_decode(data)
    }
}

impl Sealable for TxDeposit {
    fn hash_slow(&self) -> B256 {
        self.tx_hash()
    }
}

#[cfg(feature = "alloy-compat")]
impl From<TxDeposit> for alloy_rpc_types_eth::TransactionRequest {
    fn from(tx: TxDeposit) -> Self {
        let TxDeposit {
            source_hash: _,
            from,
            to,
            mint: _,
            value,
            gas_limit,
            is_system_transaction: _,
            input,
        } = tx;

        Self {
            from: Some(from),
            to: Some(to),
            value: Some(value),
            gas: Some(gas_limit),
            input: input.into(),
            ..Default::default()
        }
    }
}

/// A trait representing a deposit transaction with specific attributes.
pub trait DepositTransaction: Transaction {
    /// Returns the hash that uniquely identifies the source of the deposit.
    ///
    /// # Returns
    /// An `Option<B256>` containing the source hash if available.
    fn source_hash(&self) -> Option<B256>;

    /// Returns the optional mint value of the deposit transaction.
    ///
    /// # Returns
    /// An `u128` representing the ETH value to mint on L2, if any.
    fn mint(&self) -> u128;

    /// Indicates whether the transaction is exempt from the L2 gas limit.
    ///
    /// # Returns
    /// A `bool` indicating if the transaction is a system transaction.
    fn is_system_transaction(&self) -> bool;
}

impl DepositTransaction for TxDeposit {
    #[inline]
    fn source_hash(&self) -> Option<B256> {
        Some(self.source_hash)
    }

    #[inline]
    fn mint(&self) -> u128 {
        self.mint
    }

    #[inline]
    fn is_system_transaction(&self) -> bool {
        self.is_system_transaction
    }
}

/// Deposit transactions don't have a signature, however, we include an empty signature in the
/// response for better compatibility.
///
/// This function can be used as `serialize_with` serde attribute for the [`TxDeposit`] and will
/// flatten [`TxDeposit::signature`] into response.
#[cfg(feature = "serde")]
pub fn serde_deposit_tx_rpc<T: serde::Serialize, S: serde::Serializer>(
    value: &T,
    serializer: S,
) -> Result<S::Ok, S::Error> {
    use serde::Serialize;

    #[derive(Serialize)]
    struct SerdeHelper<'a, T> {
        #[serde(flatten)]
        value: &'a T,
        #[serde(flatten)]
        signature: Signature,
    }

    SerdeHelper { value, signature: TxDeposit::signature() }.serialize(serializer)
}

#[cfg(test)]
mod tests {
    use super::*;
    use alloy_primitives::hex;
    use alloy_rlp::BytesMut;

    #[test]
    fn test_deposit_transaction_trait() {
        let tx = TxDeposit {
            source_hash: B256::with_last_byte(42),
            from: Address::default(),
            to: TxKind::default(),
            mint: 100,
            value: U256::from(1000),
            gas_limit: 50000,
            is_system_transaction: true,
            input: Bytes::default(),
        };

        assert_eq!(tx.source_hash(), Some(B256::with_last_byte(42)));
        assert_eq!(tx.mint(), 100);
        assert!(tx.is_system_transaction());
    }

    #[test]
    fn test_deposit_transaction_without_mint() {
        let tx = TxDeposit {
            source_hash: B256::default(),
            from: Address::default(),
            to: TxKind::default(),
            mint: 0,
            value: U256::default(),
            gas_limit: 50000,
            is_system_transaction: false,
            input: Bytes::default(),
        };

        assert_eq!(tx.source_hash(), Some(B256::default()));
        assert_eq!(tx.mint(), 0);
        assert!(!tx.is_system_transaction());
    }

    #[test]
    fn test_deposit_transaction_to_contract() {
        let contract_address = Address::with_last_byte(0xFF);
        let tx = TxDeposit {
            source_hash: B256::default(),
            from: Address::default(),
            to: TxKind::Call(contract_address),
            mint: 200,
            value: U256::from(500),
            gas_limit: 100000,
            is_system_transaction: false,
            input: Bytes::from_static(&[1, 2, 3]),
        };

        assert_eq!(tx.source_hash(), Some(B256::default()));
        assert_eq!(tx.mint(), 200);
        assert!(!tx.is_system_transaction());
        assert_eq!(tx.kind(), TxKind::Call(contract_address));
    }

    #[test]
    fn test_rlp_roundtrip() {
        let bytes = Bytes::from_static(&hex!(
            "7ef9015aa044bae9d41b8380d781187b426c6fe43df5fb2fb57bd4466ef6a701e1f01e015694deaddeaddeaddeaddeaddeaddeaddeaddead000194420000000000000000000000000000000000001580808408f0d18001b90104015d8eb900000000000000000000000000000000000000000000000000000000008057650000000000000000000000000000000000000000000000000000000063d96d10000000000000000000000000000000000000000000000000000000000009f35273d89754a1e0387b89520d989d3be9c37c1f32495a88faf1ea05c61121ab0d1900000000000000000000000000000000000000000000000000000000000000010000000000000000000000002d679b567db6187c0c8323fa982cfb88b74dbcc7000000000000000000000000000000000000000000000000000000000000083400000000000000000000000000000000000000000000000000000000000f4240"
        ));
        let tx_a = TxDeposit::decode(&mut bytes[1..].as_ref()).unwrap();
        let mut buf_a = BytesMut::default();
        tx_a.encode(&mut buf_a);
        assert_eq!(&buf_a[..], &bytes[1..]);
    }

    #[test]
    fn test_encode_decode_fields() {
        let original = TxDeposit {
            source_hash: B256::default(),
            from: Address::default(),
            to: TxKind::default(),
            mint: 100,
            value: U256::default(),
            gas_limit: 50000,
            is_system_transaction: true,
            input: Bytes::default(),
        };

        let mut buffer = BytesMut::new();
        original.rlp_encode_fields(&mut buffer);
        let decoded = TxDeposit::rlp_decode_fields(&mut &buffer[..]).expect("Failed to decode");

        assert_eq!(original, decoded);
    }

    #[test]
    fn test_encode_with_and_without_header() {
        let tx_deposit = TxDeposit {
            source_hash: B256::default(),
            from: Address::default(),
            to: TxKind::default(),
            mint: 100,
            value: U256::default(),
            gas_limit: 50000,
            is_system_transaction: true,
            input: Bytes::default(),
        };

        let mut buffer_with_header = BytesMut::new();
        tx_deposit.encode(&mut buffer_with_header);

        let mut buffer_without_header = BytesMut::new();
        tx_deposit.rlp_encode_fields(&mut buffer_without_header);

        assert!(buffer_with_header.len() > buffer_without_header.len());
    }

    #[test]
    fn test_payload_length() {
        let tx_deposit = TxDeposit {
            source_hash: B256::default(),
            from: Address::default(),
            to: TxKind::default(),
            mint: 100,
            value: U256::default(),
            gas_limit: 50000,
            is_system_transaction: true,
            input: Bytes::default(),
        };

        assert!(tx_deposit.size() > tx_deposit.rlp_encoded_fields_length());
    }

    #[test]
    fn test_encode_inner_with_and_without_header() {
        let tx_deposit = TxDeposit {
            source_hash: B256::default(),
            from: Address::default(),
            to: TxKind::default(),
            mint: 100,
            value: U256::default(),
            gas_limit: 50000,
            is_system_transaction: true,
            input: Bytes::default(),
        };

        let mut buffer_with_header = BytesMut::new();
        tx_deposit.network_encode(&mut buffer_with_header);

        let mut buffer_without_header = BytesMut::new();
        tx_deposit.encode_2718(&mut buffer_without_header);

        assert!(buffer_with_header.len() > buffer_without_header.len());
    }

    #[test]
    fn test_payload_length_header() {
        let tx_deposit = TxDeposit {
            source_hash: B256::default(),
            from: Address::default(),
            to: TxKind::default(),
            mint: 100,
            value: U256::default(),
            gas_limit: 50000,
            is_system_transaction: true,
            input: Bytes::default(),
        };

        let total_len = tx_deposit.network_encoded_length();
        let len_without_header = tx_deposit.eip2718_encoded_length();

        assert!(total_len > len_without_header);
    }
    #[test]
    fn test_deposit_tx_roundtrip() {
        let raw_txs = [
            "7ef8f8a0871ec5fb6afe7e5ae950bbb4cfd7d7cb277b413e67da806d50834a814b14c9f494deaddeaddeaddeaddeaddeaddeaddeaddead00019442000000000000000000000000000000000000158080830f424080b8a4440a5e20000008dd00101c12000000000000000400000000681c941f0000000001566261000000000000000000000000000000000000000000000000000000005f629c020000000000000000000000000000000000000000000000000000000000000001937badfbcce566e0ba932a3f7659644aa0c6ef019541d3134a1d8cb9f84d45c70000000000000000000000005050f69a9786f081509234f1a7f4684b5e5b76c9",
        ];

        for raw_tx_hex in raw_txs {
            let raw_tx = hex::decode(raw_tx_hex).unwrap();

            let tx = TxDeposit::decode_2718(&mut raw_tx.as_ref()).unwrap();
            let mut encoded = BytesMut::new();
            tx.encode_2718(&mut encoded);
            assert_eq!(&encoded[..], &raw_tx[..], "Encoded bytes don't match original");

            let tx_from_fields = TxDeposit::rlp_decode(&mut &raw_tx[1..]).unwrap();
            let mut encoded_fields = BytesMut::new();
            tx_from_fields.rlp_encode(&mut encoded_fields);
            assert_eq!(
                &encoded_fields[..],
                &raw_tx[1..],
                "RLP encoded fields don't match original"
            );
        }
    }
}

/// Bincode-compatible [`TxDeposit`] serde implementation.
#[cfg(all(feature = "serde", feature = "serde-bincode-compat"))]
pub(super) mod serde_bincode_compat {
    use alloc::borrow::Cow;
    use alloy_primitives::{Address, B256, Bytes, TxKind, U256};
    use serde::{Deserialize, Deserializer, Serialize, Serializer};
    use serde_with::{DeserializeAs, SerializeAs};

    /// Bincode-compatible [`super::TxDeposit`] serde implementation.
    ///
    /// Intended to use with the [`serde_with::serde_as`] macro in the following way:
    /// ```rust
    /// use op_alloy_consensus::{TxDeposit, serde_bincode_compat};
    /// use serde::{Deserialize, Serialize};
    /// use serde_with::serde_as;
    ///
    /// #[serde_as]
    /// #[derive(Serialize, Deserialize)]
    /// struct Data {
    ///     #[serde_as(as = "serde_bincode_compat::TxDeposit")]
    ///     transaction: TxDeposit,
    /// }
    /// ```
    #[derive(Debug, Serialize, Deserialize)]
    pub struct TxDeposit<'a> {
        source_hash: B256,
        from: Address,
        #[serde(default)]
        to: TxKind,
        #[serde(default)]
        mint: u128,
        value: U256,
        gas_limit: u64,
        is_system_transaction: bool,
        input: Cow<'a, Bytes>,
    }

    impl<'a> From<&'a super::TxDeposit> for TxDeposit<'a> {
        fn from(value: &'a super::TxDeposit) -> Self {
            Self {
                source_hash: value.source_hash,
                from: value.from,
                to: value.to,
                mint: value.mint,
                value: value.value,
                gas_limit: value.gas_limit,
                is_system_transaction: value.is_system_transaction,
                input: Cow::Borrowed(&value.input),
            }
        }
    }

    impl<'a> From<TxDeposit<'a>> for super::TxDeposit {
        fn from(value: TxDeposit<'a>) -> Self {
            Self {
                source_hash: value.source_hash,
                from: value.from,
                to: value.to,
                mint: value.mint,
                value: value.value,
                gas_limit: value.gas_limit,
                is_system_transaction: value.is_system_transaction,
                input: value.input.into_owned(),
            }
        }
    }

    impl SerializeAs<super::TxDeposit> for TxDeposit<'_> {
        fn serialize_as<S>(source: &super::TxDeposit, serializer: S) -> Result<S::Ok, S::Error>
        where
            S: Serializer,
        {
            TxDeposit::from(source).serialize(serializer)
        }
    }

    impl<'de> DeserializeAs<'de, super::TxDeposit> for TxDeposit<'de> {
        fn deserialize_as<D>(deserializer: D) -> Result<super::TxDeposit, D::Error>
        where
            D: Deserializer<'de>,
        {
            TxDeposit::deserialize(deserializer).map(Into::into)
        }
    }

    #[cfg(test)]
    mod tests {
        use arbitrary::Arbitrary;
        use rand::Rng;
        use serde::{Deserialize, Serialize};
        use serde_with::serde_as;

        use super::super::{TxDeposit, serde_bincode_compat};

        #[test]
        fn test_tx_deposit_bincode_roundtrip() {
            #[serde_as]
            #[derive(Debug, PartialEq, Eq, Serialize, Deserialize)]
            struct Data {
                #[serde_as(as = "serde_bincode_compat::TxDeposit")]
                transaction: TxDeposit,
            }

            let mut bytes = [0u8; 1024];
            rand::rng().fill(bytes.as_mut_slice());
            let data = Data {
                transaction: TxDeposit::arbitrary(&mut arbitrary::Unstructured::new(&bytes))
                    .unwrap(),
            };

            let encoded = bincode::serde::encode_to_vec(&data, bincode::config::legacy()).unwrap();
            let (decoded, _) =
                bincode::serde::decode_from_slice::<Data, _>(&encoded, bincode::config::legacy())
                    .unwrap();
            assert_eq!(decoded, data);
        }
    }
}
