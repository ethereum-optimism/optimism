use alloy_consensus::{
    Eip2718EncodableReceipt, Eip658Value, Receipt, ReceiptWithBloom, RlpDecodableReceipt,
    RlpEncodableReceipt, TxReceipt, Typed2718,
};
use alloy_eips::{eip2718::Eip2718Result, Decodable2718, Encodable2718};
use alloy_primitives::{Bloom, Log};
use alloy_rlp::{BufMut, Decodable, Encodable, Header};
use op_alloy_consensus::{OpDepositReceipt, OpTxType};
use reth_primitives_traits::InMemorySize;

/// Typed ethereum transaction receipt.
/// Receipt containing result of transaction execution.
#[derive(Clone, Debug, PartialEq, Eq)]
#[cfg_attr(feature = "serde", derive(serde::Serialize, serde::Deserialize))]
#[cfg_attr(feature = "arbitrary", derive(arbitrary::Arbitrary))]
#[cfg_attr(feature = "reth-codec", reth_codecs::add_arbitrary_tests(rlp))]
pub enum OpReceipt {
    /// Legacy receipt
    Legacy(Receipt),
    /// EIP-2930 receipt
    Eip2930(Receipt),
    /// EIP-1559 receipt
    Eip1559(Receipt),
    /// EIP-7702 receipt
    Eip7702(Receipt),
    /// Deposit receipt
    Deposit(OpDepositReceipt),
}

impl OpReceipt {
    /// Returns [`OpTxType`] of the receipt.
    pub const fn tx_type(&self) -> OpTxType {
        match self {
            Self::Legacy(_) => OpTxType::Legacy,
            Self::Eip2930(_) => OpTxType::Eip2930,
            Self::Eip1559(_) => OpTxType::Eip1559,
            Self::Eip7702(_) => OpTxType::Eip7702,
            Self::Deposit(_) => OpTxType::Deposit,
        }
    }

    /// Returns inner [`Receipt`],
    pub const fn as_receipt(&self) -> &Receipt {
        match self {
            Self::Legacy(receipt) |
            Self::Eip2930(receipt) |
            Self::Eip1559(receipt) |
            Self::Eip7702(receipt) => receipt,
            Self::Deposit(receipt) => &receipt.inner,
        }
    }

    /// Returns a mutable reference to the inner [`Receipt`],
    pub const fn as_receipt_mut(&mut self) -> &mut Receipt {
        match self {
            Self::Legacy(receipt) |
            Self::Eip2930(receipt) |
            Self::Eip1559(receipt) |
            Self::Eip7702(receipt) => receipt,
            Self::Deposit(receipt) => &mut receipt.inner,
        }
    }

    /// Returns length of RLP-encoded receipt fields with the given [`Bloom`] without an RLP header.
    pub fn rlp_encoded_fields_length(&self, bloom: &Bloom) -> usize {
        match self {
            Self::Legacy(receipt) |
            Self::Eip2930(receipt) |
            Self::Eip1559(receipt) |
            Self::Eip7702(receipt) => receipt.rlp_encoded_fields_length_with_bloom(bloom),
            Self::Deposit(receipt) => receipt.rlp_encoded_fields_length_with_bloom(bloom),
        }
    }

    /// RLP-encodes receipt fields with the given [`Bloom`] without an RLP header.
    pub fn rlp_encode_fields(&self, bloom: &Bloom, out: &mut dyn BufMut) {
        match self {
            Self::Legacy(receipt) |
            Self::Eip2930(receipt) |
            Self::Eip1559(receipt) |
            Self::Eip7702(receipt) => receipt.rlp_encode_fields_with_bloom(bloom, out),
            Self::Deposit(receipt) => receipt.rlp_encode_fields_with_bloom(bloom, out),
        }
    }

    /// Returns RLP header for inner encoding.
    pub fn rlp_header_inner(&self, bloom: &Bloom) -> Header {
        Header { list: true, payload_length: self.rlp_encoded_fields_length(bloom) }
    }

    /// Returns RLP header for inner encoding without bloom.
    pub fn rlp_header_inner_without_bloom(&self) -> Header {
        Header { list: true, payload_length: self.rlp_encoded_fields_length_without_bloom() }
    }

    /// RLP-decodes the receipt from the provided buffer. This does not expect a type byte or
    /// network header.
    pub fn rlp_decode_inner(
        buf: &mut &[u8],
        tx_type: OpTxType,
    ) -> alloy_rlp::Result<ReceiptWithBloom<Self>> {
        match tx_type {
            OpTxType::Legacy => {
                let ReceiptWithBloom { receipt, logs_bloom } =
                    RlpDecodableReceipt::rlp_decode_with_bloom(buf)?;
                Ok(ReceiptWithBloom { receipt: Self::Legacy(receipt), logs_bloom })
            }
            OpTxType::Eip2930 => {
                let ReceiptWithBloom { receipt, logs_bloom } =
                    RlpDecodableReceipt::rlp_decode_with_bloom(buf)?;
                Ok(ReceiptWithBloom { receipt: Self::Eip2930(receipt), logs_bloom })
            }
            OpTxType::Eip1559 => {
                let ReceiptWithBloom { receipt, logs_bloom } =
                    RlpDecodableReceipt::rlp_decode_with_bloom(buf)?;
                Ok(ReceiptWithBloom { receipt: Self::Eip1559(receipt), logs_bloom })
            }
            OpTxType::Eip7702 => {
                let ReceiptWithBloom { receipt, logs_bloom } =
                    RlpDecodableReceipt::rlp_decode_with_bloom(buf)?;
                Ok(ReceiptWithBloom { receipt: Self::Eip7702(receipt), logs_bloom })
            }
            OpTxType::Deposit => {
                let ReceiptWithBloom { receipt, logs_bloom } =
                    RlpDecodableReceipt::rlp_decode_with_bloom(buf)?;
                Ok(ReceiptWithBloom { receipt: Self::Deposit(receipt), logs_bloom })
            }
        }
    }

    /// RLP-encodes receipt fields without an RLP header.
    pub fn rlp_encode_fields_without_bloom(&self, out: &mut dyn BufMut) {
        match self {
            Self::Legacy(receipt) |
            Self::Eip2930(receipt) |
            Self::Eip1559(receipt) |
            Self::Eip7702(receipt) => {
                receipt.status.encode(out);
                receipt.cumulative_gas_used.encode(out);
                receipt.logs.encode(out);
            }
            Self::Deposit(receipt) => {
                receipt.inner.status.encode(out);
                receipt.inner.cumulative_gas_used.encode(out);
                receipt.inner.logs.encode(out);
                if let Some(nonce) = receipt.deposit_nonce {
                    nonce.encode(out);
                }
                if let Some(version) = receipt.deposit_receipt_version {
                    version.encode(out);
                }
            }
        }
    }

    /// Returns length of RLP-encoded receipt fields without an RLP header.
    pub fn rlp_encoded_fields_length_without_bloom(&self) -> usize {
        match self {
            Self::Legacy(receipt) |
            Self::Eip2930(receipt) |
            Self::Eip1559(receipt) |
            Self::Eip7702(receipt) => {
                receipt.status.length() +
                    receipt.cumulative_gas_used.length() +
                    receipt.logs.length()
            }
            Self::Deposit(receipt) => {
                receipt.inner.status.length() +
                    receipt.inner.cumulative_gas_used.length() +
                    receipt.inner.logs.length() +
                    receipt.deposit_nonce.map_or(0, |nonce| nonce.length()) +
                    receipt.deposit_receipt_version.map_or(0, |version| version.length())
            }
        }
    }

    /// RLP-decodes the receipt from the provided buffer without bloom.
    pub fn rlp_decode_inner_without_bloom(
        buf: &mut &[u8],
        tx_type: OpTxType,
    ) -> alloy_rlp::Result<Self> {
        let header = Header::decode(buf)?;
        if !header.list {
            return Err(alloy_rlp::Error::UnexpectedString);
        }

        let remaining = buf.len();
        let status = Decodable::decode(buf)?;
        let cumulative_gas_used = Decodable::decode(buf)?;
        let logs = Decodable::decode(buf)?;

        let mut deposit_nonce = None;
        let mut deposit_receipt_version = None;

        // For deposit receipts, try to decode nonce and version if they exist
        if tx_type == OpTxType::Deposit && buf.len() + header.payload_length > remaining {
            deposit_nonce = Some(Decodable::decode(buf)?);
            if buf.len() + header.payload_length > remaining {
                deposit_receipt_version = Some(Decodable::decode(buf)?);
            }
        }

        if buf.len() + header.payload_length != remaining {
            return Err(alloy_rlp::Error::UnexpectedLength);
        }

        match tx_type {
            OpTxType::Legacy => Ok(Self::Legacy(Receipt { status, cumulative_gas_used, logs })),
            OpTxType::Eip2930 => Ok(Self::Eip2930(Receipt { status, cumulative_gas_used, logs })),
            OpTxType::Eip1559 => Ok(Self::Eip1559(Receipt { status, cumulative_gas_used, logs })),
            OpTxType::Eip7702 => Ok(Self::Eip7702(Receipt { status, cumulative_gas_used, logs })),
            OpTxType::Deposit => Ok(Self::Deposit(OpDepositReceipt {
                inner: Receipt { status, cumulative_gas_used, logs },
                deposit_nonce,
                deposit_receipt_version,
            })),
        }
    }
}

impl Eip2718EncodableReceipt for OpReceipt {
    fn eip2718_encoded_length_with_bloom(&self, bloom: &Bloom) -> usize {
        !self.tx_type().is_legacy() as usize + self.rlp_header_inner(bloom).length_with_payload()
    }

    fn eip2718_encode_with_bloom(&self, bloom: &Bloom, out: &mut dyn BufMut) {
        if !self.tx_type().is_legacy() {
            out.put_u8(self.tx_type() as u8);
        }
        self.rlp_header_inner(bloom).encode(out);
        self.rlp_encode_fields(bloom, out);
    }
}

impl RlpEncodableReceipt for OpReceipt {
    fn rlp_encoded_length_with_bloom(&self, bloom: &Bloom) -> usize {
        let mut len = self.eip2718_encoded_length_with_bloom(bloom);
        if !self.tx_type().is_legacy() {
            len += Header {
                list: false,
                payload_length: self.eip2718_encoded_length_with_bloom(bloom),
            }
            .length();
        }

        len
    }

    fn rlp_encode_with_bloom(&self, bloom: &Bloom, out: &mut dyn BufMut) {
        if !self.tx_type().is_legacy() {
            Header { list: false, payload_length: self.eip2718_encoded_length_with_bloom(bloom) }
                .encode(out);
        }
        self.eip2718_encode_with_bloom(bloom, out);
    }
}

impl RlpDecodableReceipt for OpReceipt {
    fn rlp_decode_with_bloom(buf: &mut &[u8]) -> alloy_rlp::Result<ReceiptWithBloom<Self>> {
        let header_buf = &mut &**buf;
        let header = Header::decode(header_buf)?;

        // Legacy receipt, reuse initial buffer without advancing
        if header.list {
            return Self::rlp_decode_inner(buf, OpTxType::Legacy)
        }

        // Otherwise, advance the buffer and try decoding type flag followed by receipt
        *buf = *header_buf;

        let remaining = buf.len();
        let tx_type = OpTxType::decode(buf)?;
        let this = Self::rlp_decode_inner(buf, tx_type)?;

        if buf.len() + header.payload_length != remaining {
            return Err(alloy_rlp::Error::UnexpectedLength);
        }

        Ok(this)
    }
}

impl Encodable2718 for OpReceipt {
    fn encode_2718_len(&self) -> usize {
        !self.tx_type().is_legacy() as usize +
            self.rlp_header_inner_without_bloom().length_with_payload()
    }

    fn encode_2718(&self, out: &mut dyn BufMut) {
        if !self.tx_type().is_legacy() {
            out.put_u8(self.tx_type() as u8);
        }
        self.rlp_header_inner_without_bloom().encode(out);
        self.rlp_encode_fields_without_bloom(out);
    }
}

impl Decodable2718 for OpReceipt {
    fn typed_decode(ty: u8, buf: &mut &[u8]) -> Eip2718Result<Self> {
        Ok(Self::rlp_decode_inner_without_bloom(buf, OpTxType::try_from(ty)?)?)
    }

    fn fallback_decode(buf: &mut &[u8]) -> Eip2718Result<Self> {
        Ok(Self::rlp_decode_inner_without_bloom(buf, OpTxType::Legacy)?)
    }
}

impl Encodable for OpReceipt {
    fn encode(&self, out: &mut dyn BufMut) {
        self.network_encode(out);
    }

    fn length(&self) -> usize {
        self.network_len()
    }
}

impl Decodable for OpReceipt {
    fn decode(buf: &mut &[u8]) -> alloy_rlp::Result<Self> {
        Ok(Self::network_decode(buf)?)
    }
}

impl TxReceipt for OpReceipt {
    type Log = Log;

    fn status_or_post_state(&self) -> Eip658Value {
        self.as_receipt().status_or_post_state()
    }

    fn status(&self) -> bool {
        self.as_receipt().status()
    }

    fn bloom(&self) -> Bloom {
        self.as_receipt().bloom()
    }

    fn cumulative_gas_used(&self) -> u64 {
        self.as_receipt().cumulative_gas_used()
    }

    fn logs(&self) -> &[Log] {
        self.as_receipt().logs()
    }
}

impl Typed2718 for OpReceipt {
    fn ty(&self) -> u8 {
        self.tx_type().into()
    }
}

impl InMemorySize for OpReceipt {
    fn size(&self) -> usize {
        self.as_receipt().size()
    }
}

impl reth_primitives_traits::Receipt for OpReceipt {}

/// Trait for deposit receipt.
pub trait DepositReceipt: reth_primitives_traits::Receipt {
    /// Returns deposit receipt if it is a deposit transaction.
    fn as_deposit_receipt_mut(&mut self) -> Option<&mut OpDepositReceipt>;
}

impl DepositReceipt for OpReceipt {
    fn as_deposit_receipt_mut(&mut self) -> Option<&mut OpDepositReceipt> {
        match self {
            Self::Deposit(receipt) => Some(receipt),
            _ => None,
        }
    }
}

#[cfg(feature = "reth-codec")]
mod compact {
    use super::*;
    use alloc::borrow::Cow;
    use reth_codecs::Compact;

    #[derive(reth_codecs::CompactZstd)]
    #[reth_zstd(
        compressor = reth_zstd_compressors::RECEIPT_COMPRESSOR,
        decompressor = reth_zstd_compressors::RECEIPT_DECOMPRESSOR
    )]
    struct CompactOpReceipt<'a> {
        tx_type: OpTxType,
        success: bool,
        cumulative_gas_used: u64,
        #[expect(clippy::owned_cow)]
        logs: Cow<'a, Vec<Log>>,
        deposit_nonce: Option<u64>,
        deposit_receipt_version: Option<u64>,
    }

    impl<'a> From<&'a OpReceipt> for CompactOpReceipt<'a> {
        fn from(receipt: &'a OpReceipt) -> Self {
            Self {
                tx_type: receipt.tx_type(),
                success: receipt.status(),
                cumulative_gas_used: receipt.cumulative_gas_used(),
                logs: Cow::Borrowed(&receipt.as_receipt().logs),
                deposit_nonce: if let OpReceipt::Deposit(receipt) = receipt {
                    receipt.deposit_nonce
                } else {
                    None
                },
                deposit_receipt_version: if let OpReceipt::Deposit(receipt) = receipt {
                    receipt.deposit_receipt_version
                } else {
                    None
                },
            }
        }
    }

    impl From<CompactOpReceipt<'_>> for OpReceipt {
        fn from(receipt: CompactOpReceipt<'_>) -> Self {
            let CompactOpReceipt {
                tx_type,
                success,
                cumulative_gas_used,
                logs,
                deposit_nonce,
                deposit_receipt_version,
            } = receipt;

            let inner =
                Receipt { status: success.into(), cumulative_gas_used, logs: logs.into_owned() };

            match tx_type {
                OpTxType::Legacy => Self::Legacy(inner),
                OpTxType::Eip2930 => Self::Eip2930(inner),
                OpTxType::Eip1559 => Self::Eip1559(inner),
                OpTxType::Eip7702 => Self::Eip7702(inner),
                OpTxType::Deposit => Self::Deposit(OpDepositReceipt {
                    inner,
                    deposit_nonce,
                    deposit_receipt_version,
                }),
            }
        }
    }

    impl Compact for OpReceipt {
        fn to_compact<B>(&self, buf: &mut B) -> usize
        where
            B: bytes::BufMut + AsMut<[u8]>,
        {
            CompactOpReceipt::from(self).to_compact(buf)
        }

        fn from_compact(buf: &[u8], len: usize) -> (Self, &[u8]) {
            let (receipt, buf) = CompactOpReceipt::from_compact(buf, len);
            (receipt.into(), buf)
        }
    }

    #[cfg(test)]
    #[test]
    fn test_ensure_backwards_compatibility() {
        use reth_codecs::{test_utils::UnusedBits, validate_bitflag_backwards_compat};

        assert_eq!(CompactOpReceipt::bitflag_encoded_bytes(), 2);
        validate_bitflag_backwards_compat!(CompactOpReceipt<'_>, UnusedBits::NotZero);
    }
}

#[cfg(all(feature = "serde", feature = "serde-bincode-compat"))]
pub(super) mod serde_bincode_compat {
    use serde::{Deserialize, Deserializer, Serialize, Serializer};
    use serde_with::{DeserializeAs, SerializeAs};

    /// Bincode-compatible [`super::OpReceipt`] serde implementation.
    ///
    /// Intended to use with the [`serde_with::serde_as`] macro in the following way:
    /// ```rust
    /// use reth_optimism_primitives::{serde_bincode_compat, OpReceipt};
    /// use serde::{de::DeserializeOwned, Deserialize, Serialize};
    /// use serde_with::serde_as;
    ///
    /// #[serde_as]
    /// #[derive(Serialize, Deserialize)]
    /// struct Data {
    ///     #[serde_as(as = "serde_bincode_compat::OpReceipt<'_>")]
    ///     receipt: OpReceipt,
    /// }
    /// ```
    #[derive(Debug, Serialize, Deserialize)]
    pub enum OpReceipt<'a> {
        /// Legacy receipt
        Legacy(alloy_consensus::serde_bincode_compat::Receipt<'a, alloy_primitives::Log>),
        /// EIP-2930 receipt
        Eip2930(alloy_consensus::serde_bincode_compat::Receipt<'a, alloy_primitives::Log>),
        /// EIP-1559 receipt
        Eip1559(alloy_consensus::serde_bincode_compat::Receipt<'a, alloy_primitives::Log>),
        /// EIP-7702 receipt
        Eip7702(alloy_consensus::serde_bincode_compat::Receipt<'a, alloy_primitives::Log>),
        /// Deposit receipt
        Deposit(
            op_alloy_consensus::serde_bincode_compat::OpDepositReceipt<'a, alloy_primitives::Log>,
        ),
    }

    impl<'a> From<&'a super::OpReceipt> for OpReceipt<'a> {
        fn from(value: &'a super::OpReceipt) -> Self {
            match value {
                super::OpReceipt::Legacy(receipt) => Self::Legacy(receipt.into()),
                super::OpReceipt::Eip2930(receipt) => Self::Eip2930(receipt.into()),
                super::OpReceipt::Eip1559(receipt) => Self::Eip1559(receipt.into()),
                super::OpReceipt::Eip7702(receipt) => Self::Eip7702(receipt.into()),
                super::OpReceipt::Deposit(receipt) => Self::Deposit(receipt.into()),
            }
        }
    }

    impl<'a> From<OpReceipt<'a>> for super::OpReceipt {
        fn from(value: OpReceipt<'a>) -> Self {
            match value {
                OpReceipt::Legacy(receipt) => Self::Legacy(receipt.into()),
                OpReceipt::Eip2930(receipt) => Self::Eip2930(receipt.into()),
                OpReceipt::Eip1559(receipt) => Self::Eip1559(receipt.into()),
                OpReceipt::Eip7702(receipt) => Self::Eip7702(receipt.into()),
                OpReceipt::Deposit(receipt) => Self::Deposit(receipt.into()),
            }
        }
    }

    impl SerializeAs<super::OpReceipt> for OpReceipt<'_> {
        fn serialize_as<S>(source: &super::OpReceipt, serializer: S) -> Result<S::Ok, S::Error>
        where
            S: Serializer,
        {
            OpReceipt::<'_>::from(source).serialize(serializer)
        }
    }

    impl<'de> DeserializeAs<'de, super::OpReceipt> for OpReceipt<'de> {
        fn deserialize_as<D>(deserializer: D) -> Result<super::OpReceipt, D::Error>
        where
            D: Deserializer<'de>,
        {
            OpReceipt::<'_>::deserialize(deserializer).map(Into::into)
        }
    }

    impl reth_primitives_traits::serde_bincode_compat::SerdeBincodeCompat for super::OpReceipt {
        type BincodeRepr<'a> = OpReceipt<'a>;

        fn as_repr(&self) -> Self::BincodeRepr<'_> {
            self.into()
        }

        fn from_repr(repr: Self::BincodeRepr<'_>) -> Self {
            repr.into()
        }
    }

    #[cfg(test)]
    mod tests {
        use crate::{receipt::serde_bincode_compat, OpReceipt};
        use arbitrary::Arbitrary;
        use rand::Rng;
        use serde::{Deserialize, Serialize};
        use serde_with::serde_as;

        #[test]
        fn test_tx_bincode_roundtrip() {
            #[serde_as]
            #[derive(Debug, PartialEq, Eq, Serialize, Deserialize)]
            struct Data {
                #[serde_as(as = "serde_bincode_compat::OpReceipt<'_>")]
                reseipt: OpReceipt,
            }

            let mut bytes = [0u8; 1024];
            rand::rng().fill(bytes.as_mut_slice());
            let mut data = Data {
                reseipt: OpReceipt::arbitrary(&mut arbitrary::Unstructured::new(&bytes)).unwrap(),
            };
            let success = data.reseipt.as_receipt_mut().status.coerce_status();
            // // ensure we don't have an invalid poststate variant
            data.reseipt.as_receipt_mut().status = success.into();

            let encoded = bincode::serialize(&data).unwrap();
            let decoded: Data = bincode::deserialize(&encoded).unwrap();
            assert_eq!(decoded, data);
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use alloy_eips::eip2718::Encodable2718;
    use alloy_primitives::{address, b256, bytes, hex_literal::hex, Bytes};
    use alloy_rlp::Encodable;
    use reth_codecs::Compact;

    #[test]
    fn test_decode_receipt() {
        reth_codecs::test_utils::test_decode::<OpReceipt>(&hex!(
            "c30328b52ffd23fc426961a00105007eb0042307705a97e503562eacf2b95060cce9de6de68386b6c155b73a9650021a49e2f8baad17f30faff5899d785c4c0873e45bc268bcf07560106424570d11f9a59e8f3db1efa4ceec680123712275f10d92c3411e1caaa11c7c5d591bc11487168e09934a9986848136da1b583babf3a7188e3aed007a1520f1cf4c1ca7d3482c6c28d37c298613c70a76940008816c4c95644579fd08471dc34732fd0f24"
        ));
    }

    // Test vector from: https://eips.ethereum.org/EIPS/eip-2481
    #[test]
    fn encode_legacy_receipt() {
        let expected = hex!(
            "f901668001b9010000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000f85ff85d940000000000000000000000000000000000000011f842a0000000000000000000000000000000000000000000000000000000000000deada0000000000000000000000000000000000000000000000000000000000000beef830100ff"
        );

        let mut data = Vec::with_capacity(expected.length());
        let receipt = ReceiptWithBloom {
            receipt: OpReceipt::Legacy(Receipt {
                status: Eip658Value::Eip658(false),
                cumulative_gas_used: 0x1,
                logs: vec![Log::new_unchecked(
                    address!("0x0000000000000000000000000000000000000011"),
                    vec![
                        b256!("0x000000000000000000000000000000000000000000000000000000000000dead"),
                        b256!("0x000000000000000000000000000000000000000000000000000000000000beef"),
                    ],
                    bytes!("0100ff"),
                )],
            }),
            logs_bloom: [0; 256].into(),
        };

        receipt.encode(&mut data);

        // check that the rlp length equals the length of the expected rlp
        assert_eq!(receipt.length(), expected.len());
        assert_eq!(data, expected);
    }

    // Test vector from: https://eips.ethereum.org/EIPS/eip-2481
    #[test]
    fn decode_legacy_receipt() {
        let data = hex!(
            "f901668001b9010000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000f85ff85d940000000000000000000000000000000000000011f842a0000000000000000000000000000000000000000000000000000000000000deada0000000000000000000000000000000000000000000000000000000000000beef830100ff"
        );

        // EIP658Receipt
        let expected = ReceiptWithBloom {
            receipt: OpReceipt::Legacy(Receipt {
                status: Eip658Value::Eip658(false),
                cumulative_gas_used: 0x1,
                logs: vec![Log::new_unchecked(
                    address!("0x0000000000000000000000000000000000000011"),
                    vec![
                        b256!("0x000000000000000000000000000000000000000000000000000000000000dead"),
                        b256!("0x000000000000000000000000000000000000000000000000000000000000beef"),
                    ],
                    bytes!("0100ff"),
                )],
            }),
            logs_bloom: [0; 256].into(),
        };

        let receipt = ReceiptWithBloom::decode(&mut &data[..]).unwrap();
        assert_eq!(receipt, expected);
    }

    #[test]
    fn decode_deposit_receipt_regolith_roundtrip() {
        let data = hex!(
            "b901107ef9010c0182b741b9010000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000c0833d3bbf"
        );

        // Deposit Receipt (post-regolith)
        let expected = ReceiptWithBloom {
            receipt: OpReceipt::Deposit(OpDepositReceipt {
                inner: Receipt {
                    status: Eip658Value::Eip658(true),
                    cumulative_gas_used: 46913,
                    logs: vec![],
                },
                deposit_nonce: Some(4012991),
                deposit_receipt_version: None,
            }),
            logs_bloom: [0; 256].into(),
        };

        let receipt = ReceiptWithBloom::decode(&mut &data[..]).unwrap();
        assert_eq!(receipt, expected);

        let mut buf = Vec::with_capacity(data.len());
        receipt.encode(&mut buf);
        assert_eq!(buf, &data[..]);
    }

    #[test]
    fn decode_deposit_receipt_canyon_roundtrip() {
        let data = hex!(
            "b901117ef9010d0182b741b9010000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000c0833d3bbf01"
        );

        // Deposit Receipt (post-regolith)
        let expected = ReceiptWithBloom {
            receipt: OpReceipt::Deposit(OpDepositReceipt {
                inner: Receipt {
                    status: Eip658Value::Eip658(true),
                    cumulative_gas_used: 46913,
                    logs: vec![],
                },
                deposit_nonce: Some(4012991),
                deposit_receipt_version: Some(1),
            }),
            logs_bloom: [0; 256].into(),
        };

        let receipt = ReceiptWithBloom::decode(&mut &data[..]).unwrap();
        assert_eq!(receipt, expected);

        let mut buf = Vec::with_capacity(data.len());
        expected.encode(&mut buf);
        assert_eq!(buf, &data[..]);
    }

    #[test]
    fn gigantic_receipt() {
        let receipt = OpReceipt::Legacy(Receipt {
            status: Eip658Value::Eip658(true),
            cumulative_gas_used: 16747627,
            logs: vec![
                Log::new_unchecked(
                    address!("0x4bf56695415f725e43c3e04354b604bcfb6dfb6e"),
                    vec![b256!(
                        "0xc69dc3d7ebff79e41f525be431d5cd3cc08f80eaf0f7819054a726eeb7086eb9"
                    )],
                    Bytes::from(vec![1; 0xffffff]),
                ),
                Log::new_unchecked(
                    address!("0xfaca325c86bf9c2d5b413cd7b90b209be92229c2"),
                    vec![b256!(
                        "0x8cca58667b1e9ffa004720ac99a3d61a138181963b294d270d91c53d36402ae2"
                    )],
                    Bytes::from(vec![1; 0xffffff]),
                ),
            ],
        });

        let mut data = vec![];
        receipt.to_compact(&mut data);
        let (decoded, _) = OpReceipt::from_compact(&data[..], data.len());
        assert_eq!(decoded, receipt);
    }

    #[test]
    fn test_encode_2718_length() {
        let receipt = ReceiptWithBloom {
            receipt: OpReceipt::Eip1559(Receipt {
                status: Eip658Value::Eip658(true),
                cumulative_gas_used: 21000,
                logs: vec![],
            }),
            logs_bloom: Bloom::default(),
        };

        let encoded = receipt.encoded_2718();
        assert_eq!(
            encoded.len(),
            receipt.encode_2718_len(),
            "Encoded length should match the actual encoded data length"
        );

        // Test for legacy receipt as well
        let legacy_receipt = ReceiptWithBloom {
            receipt: OpReceipt::Legacy(Receipt {
                status: Eip658Value::Eip658(true),
                cumulative_gas_used: 21000,
                logs: vec![],
            }),
            logs_bloom: Bloom::default(),
        };

        let legacy_encoded = legacy_receipt.encoded_2718();
        assert_eq!(
            legacy_encoded.len(),
            legacy_receipt.encode_2718_len(),
            "Encoded length for legacy receipt should match the actual encoded data length"
        );
    }
}
