//! Transaction receipt types for Optimism.

use super::OpTxReceipt;
use crate::transaction::OpDepositInfo;
use alloy_consensus::{
    Eip658Value, Receipt, ReceiptWithBloom, RlpDecodableReceipt, RlpEncodableReceipt, TxReceipt,
};
use alloy_primitives::{Bloom, Log};
use alloy_rlp::{Buf, BufMut, Decodable, Encodable, Header};

/// [`OpDepositReceipt`] with calculated bloom filter, modified for the OP Stack.
///
/// This convenience type allows us to lazily calculate the bloom filter for a
/// receipt, similar to [`Sealed`].
///
/// [`Sealed`]: alloy_consensus::Sealed
pub type OpDepositReceiptWithBloom<T = Log> = ReceiptWithBloom<OpDepositReceipt<T>>;

/// Receipt containing result of transaction execution.
#[derive(Clone, Debug, PartialEq, Eq, Default)]
#[cfg_attr(feature = "serde", derive(serde::Serialize, serde::Deserialize))]
#[cfg_attr(feature = "serde", serde(rename_all = "camelCase"))]
pub struct OpDepositReceipt<T = Log> {
    /// The inner receipt type.
    #[cfg_attr(feature = "serde", serde(flatten))]
    pub inner: Receipt<T>,
    /// Deposit nonce for Optimism deposit transactions
    #[cfg_attr(
        feature = "serde",
        serde(
            default,
            skip_serializing_if = "Option::is_none",
            with = "alloy_serde::quantity::opt"
        )
    )]
    pub deposit_nonce: Option<u64>,
    /// Deposit receipt version for Optimism deposit transactions
    ///
    /// The deposit receipt version was introduced in Canyon to indicate an update to how
    /// receipt hashes should be computed when set. The state transition process
    /// ensures this is only set for post-Canyon deposit transactions.
    #[cfg_attr(
        feature = "serde",
        serde(
            default,
            skip_serializing_if = "Option::is_none",
            with = "alloy_serde::quantity::opt"
        )
    )]
    pub deposit_receipt_version: Option<u64>,
}

impl OpDepositReceipt {
    /// Calculates [`Log`]'s bloom filter. this is slow operation and [OpDepositReceiptWithBloom]
    /// can be used to cache this value.
    pub fn bloom_slow(&self) -> Bloom {
        self.inner.logs.iter().collect()
    }

    /// Calculates the bloom filter for the receipt and returns the [OpDepositReceiptWithBloom]
    /// container type.
    pub fn with_bloom(self) -> OpDepositReceiptWithBloom {
        self.into()
    }
}

impl<T> OpDepositReceipt<T> {
    /// Maps the inner receipt value of this receipt.
    ///
    /// This is mainly useful for mapping the receipt log type to the rpc variant.
    pub fn map_inner<U, F>(self, f: F) -> OpDepositReceipt<U>
    where
        F: FnOnce(Receipt<T>) -> Receipt<U>,
    {
        OpDepositReceipt {
            inner: f(self.inner),
            deposit_nonce: self.deposit_nonce,
            deposit_receipt_version: self.deposit_receipt_version,
        }
    }

    /// Attaches the given bloom to the receipt returning [`ReceiptWithBloom`].
    pub const fn with_bloom_unchecked(self, bloom: Bloom) -> ReceiptWithBloom<Self> {
        ReceiptWithBloom::new(self, bloom)
    }

    /// Consumes the type and returns the inner [`Receipt`].
    pub fn into_inner(self) -> Receipt<T> {
        self.inner
    }

    /// Returns the deposit info for this receipt.
    pub const fn deposit_info(&self) -> OpDepositInfo {
        OpDepositInfo {
            deposit_nonce: self.deposit_nonce,
            deposit_receipt_version: self.deposit_receipt_version,
        }
    }

    /// Converts the receipt's log type by applying a function to each log.
    ///
    /// Returns the receipt with the new log type
    pub fn map_logs<U>(self, f: impl FnMut(T) -> U) -> OpDepositReceipt<U> {
        self.map_inner(|r| r.map_logs(f))
    }
}

impl<T: Encodable> OpDepositReceipt<T> {
    /// Returns length of RLP-encoded receipt fields with the given [`Bloom`] without an RLP header.
    pub fn rlp_encoded_fields_length_with_bloom(&self, bloom: &Bloom) -> usize {
        self.inner.rlp_encoded_fields_length_with_bloom(bloom)
            + self.deposit_nonce.map_or(0, |nonce| nonce.length())
            + self.deposit_receipt_version.map_or(0, |version| version.length())
    }

    /// RLP-encodes receipt fields with the given [`Bloom`] without an RLP header.
    pub fn rlp_encode_fields_with_bloom(&self, bloom: &Bloom, out: &mut dyn BufMut) {
        self.inner.rlp_encode_fields_with_bloom(bloom, out);

        if let Some(nonce) = self.deposit_nonce {
            nonce.encode(out);
        }
        if let Some(version) = self.deposit_receipt_version {
            version.encode(out);
        }
    }

    /// Returns RLP header for this receipt encoding with the given [`Bloom`].
    pub fn rlp_header_with_bloom(&self, bloom: &Bloom) -> Header {
        Header { list: true, payload_length: self.rlp_encoded_fields_length_with_bloom(bloom) }
    }
}

impl<T: Decodable> OpDepositReceipt<T> {
    /// RLP-decodes receipt's field with a [`Bloom`].
    ///
    /// Does not expect an RLP header.
    pub fn rlp_decode_fields_with_bloom(
        buf: &mut &[u8],
    ) -> alloy_rlp::Result<ReceiptWithBloom<Self>> {
        let ReceiptWithBloom { receipt: inner, logs_bloom } =
            Receipt::rlp_decode_fields_with_bloom(buf)?;

        let deposit_nonce = (!buf.is_empty()).then(|| Decodable::decode(buf)).transpose()?;
        let deposit_receipt_version =
            (!buf.is_empty()).then(|| Decodable::decode(buf)).transpose()?;

        Ok(ReceiptWithBloom {
            logs_bloom,
            receipt: Self { inner, deposit_nonce, deposit_receipt_version },
        })
    }
}

impl<T> AsRef<Receipt<T>> for OpDepositReceipt<T> {
    fn as_ref(&self) -> &Receipt<T> {
        &self.inner
    }
}

impl<T> From<OpDepositReceipt<T>> for Receipt<T> {
    fn from(value: OpDepositReceipt<T>) -> Self {
        value.into_inner()
    }
}

impl<T> TxReceipt for OpDepositReceipt<T>
where
    T: AsRef<Log> + Clone + core::fmt::Debug + PartialEq + Eq + Send + Sync,
{
    type Log = T;

    fn status_or_post_state(&self) -> Eip658Value {
        self.inner.status_or_post_state()
    }

    fn status(&self) -> bool {
        self.inner.status()
    }

    fn bloom(&self) -> Bloom {
        self.inner.bloom_slow()
    }

    fn cumulative_gas_used(&self) -> u64 {
        self.inner.cumulative_gas_used()
    }

    fn logs(&self) -> &[Self::Log] {
        self.inner.logs()
    }
}

impl<T: Encodable> RlpEncodableReceipt for OpDepositReceipt<T> {
    fn rlp_encoded_length_with_bloom(&self, bloom: &Bloom) -> usize {
        self.rlp_header_with_bloom(bloom).length_with_payload()
    }

    fn rlp_encode_with_bloom(&self, bloom: &Bloom, out: &mut dyn BufMut) {
        self.rlp_header_with_bloom(bloom).encode(out);
        self.rlp_encode_fields_with_bloom(bloom, out);
    }
}

impl<T: Decodable> RlpDecodableReceipt for OpDepositReceipt<T> {
    fn rlp_decode_with_bloom(buf: &mut &[u8]) -> alloy_rlp::Result<ReceiptWithBloom<Self>> {
        let header = Header::decode(buf)?;
        if !header.list {
            return Err(alloy_rlp::Error::UnexpectedString);
        }

        if buf.len() < header.payload_length {
            return Err(alloy_rlp::Error::InputTooShort);
        }

        // Note: we pass a separate buffer to `rlp_decode_fields_with_bloom` to allow it decode
        // optional fields based on the remaining length.
        let mut fields_buf = &buf[..header.payload_length];
        let this = Self::rlp_decode_fields_with_bloom(&mut fields_buf)?;

        if !fields_buf.is_empty() {
            return Err(alloy_rlp::Error::UnexpectedLength);
        }

        buf.advance(header.payload_length);

        Ok(this)
    }
}

impl OpTxReceipt for OpDepositReceipt {
    fn deposit_nonce(&self) -> Option<u64> {
        self.deposit_nonce
    }

    fn deposit_receipt_version(&self) -> Option<u64> {
        self.deposit_receipt_version
    }
}

impl<T> From<ReceiptWithBloom<Self>> for OpDepositReceipt<T> {
    fn from(value: ReceiptWithBloom<Self>) -> Self {
        value.receipt
    }
}

#[cfg(feature = "arbitrary")]
impl<'a, T> arbitrary::Arbitrary<'a> for OpDepositReceipt<T>
where
    T: arbitrary::Arbitrary<'a>,
{
    fn arbitrary(u: &mut arbitrary::Unstructured<'a>) -> arbitrary::Result<Self> {
        use alloc::vec::Vec;
        let deposit_nonce = Option::<u64>::arbitrary(u)?;
        let deposit_receipt_version =
            deposit_nonce.is_some().then(|| u64::arbitrary(u)).transpose()?;
        Ok(Self {
            inner: Receipt {
                status: Eip658Value::arbitrary(u)?,
                cumulative_gas_used: u64::arbitrary(u)?,
                logs: Vec::<T>::arbitrary(u)?,
            },
            deposit_nonce,
            deposit_receipt_version,
        })
    }
}

/// Bincode-compatible [`OpDepositReceipt`] serde implementation.
#[cfg(all(feature = "serde", feature = "serde-bincode-compat"))]
pub(crate) mod serde_bincode_compat {
    use alloc::borrow::Cow;
    use alloy_consensus::Receipt;
    use serde::{Deserialize, Deserializer, Serialize, Serializer};
    use serde_with::{DeserializeAs, SerializeAs};

    /// Bincode-compatible [`super::OpDepositReceipt`] serde implementation.
    ///
    /// Intended to use with the [`serde_with::serde_as`] macro in the following way:
    /// ```rust
    /// use op_alloy_consensus::{OpDepositReceipt, serde_bincode_compat};
    /// use serde::{Deserialize, Serialize, de::DeserializeOwned};
    /// use serde_with::serde_as;
    ///
    /// #[serde_as]
    /// #[derive(Serialize, Deserialize)]
    /// struct Data<T: Serialize + DeserializeOwned + Clone + 'static> {
    ///     #[serde_as(as = "serde_bincode_compat::OpDepositReceipt<'_, T>")]
    ///     receipt: OpDepositReceipt<T>,
    /// }
    /// ```
    #[derive(Debug, Serialize, Deserialize)]
    pub struct OpDepositReceipt<'a, T: Clone> {
        logs: Cow<'a, [T]>,
        status: bool,
        cumulative_gas_used: u64,
        deposit_nonce: Option<u64>,
        deposit_receipt_version: Option<u64>,
    }

    impl<'a, T: Clone> From<&'a super::OpDepositReceipt<T>> for OpDepositReceipt<'a, T> {
        fn from(value: &'a super::OpDepositReceipt<T>) -> Self {
            Self {
                logs: Cow::Borrowed(&value.inner.logs),
                // OP has no post state root variant
                status: value.inner.status.coerce_status(),
                cumulative_gas_used: value.inner.cumulative_gas_used,
                deposit_nonce: value.deposit_nonce,
                deposit_receipt_version: value.deposit_receipt_version,
            }
        }
    }

    impl<'a, T: Clone> From<OpDepositReceipt<'a, T>> for super::OpDepositReceipt<T> {
        fn from(value: OpDepositReceipt<'a, T>) -> Self {
            Self {
                inner: Receipt {
                    status: value.status.into(),
                    cumulative_gas_used: value.cumulative_gas_used,
                    logs: value.logs.into_owned(),
                },
                deposit_nonce: value.deposit_nonce,
                deposit_receipt_version: value.deposit_receipt_version,
            }
        }
    }

    impl<T: Serialize + Clone> SerializeAs<super::OpDepositReceipt<T>> for OpDepositReceipt<'_, T> {
        fn serialize_as<S>(
            source: &super::OpDepositReceipt<T>,
            serializer: S,
        ) -> Result<S::Ok, S::Error>
        where
            S: Serializer,
        {
            OpDepositReceipt::<'_, T>::from(source).serialize(serializer)
        }
    }

    impl<'de, T: Deserialize<'de> + Clone> DeserializeAs<'de, super::OpDepositReceipt<T>>
        for OpDepositReceipt<'de, T>
    {
        fn deserialize_as<D>(deserializer: D) -> Result<super::OpDepositReceipt<T>, D::Error>
        where
            D: Deserializer<'de>,
        {
            OpDepositReceipt::<'_, T>::deserialize(deserializer).map(Into::into)
        }
    }

    #[cfg(test)]
    mod tests {
        use super::super::{OpDepositReceipt, serde_bincode_compat};
        use alloy_primitives::Log;
        use arbitrary::Arbitrary;
        use rand::Rng;
        use serde::{Deserialize, Serialize, de::DeserializeOwned};
        use serde_with::serde_as;

        #[test]
        fn test_tx_deposit_bincode_roundtrip() {
            #[serde_as]
            #[derive(Debug, PartialEq, Eq, Serialize, Deserialize)]
            struct Data<T: Serialize + DeserializeOwned + Clone + 'static> {
                #[serde_as(as = "serde_bincode_compat::OpDepositReceipt<'_,T>")]
                transaction: OpDepositReceipt<T>,
            }

            let mut bytes = [0u8; 1024];
            rand::rng().fill(bytes.as_mut_slice());
            let mut data = Data {
                transaction: OpDepositReceipt::arbitrary(&mut arbitrary::Unstructured::new(&bytes))
                    .unwrap(),
            };
            // ensure we don't have an invalid poststate variant
            data.transaction.inner.status = data.transaction.inner.status.coerce_status().into();

            let encoded = bincode::serde::encode_to_vec(&data, bincode::config::legacy()).unwrap();
            let (decoded, _) = bincode::serde::decode_from_slice::<Data<Log>, _>(
                &encoded,
                bincode::config::legacy(),
            )
            .unwrap();
            assert_eq!(decoded, data);
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use alloy_consensus::Receipt;
    use alloy_primitives::{Bytes, Log, LogData, address, b256, bytes, hex};
    use alloy_rlp::{Decodable, Encodable};

    #[cfg(not(feature = "std"))]
    use alloc::{vec, vec::Vec};

    // Test vector from: https://eips.ethereum.org/EIPS/eip-2481
    #[test]
    fn decode_legacy_receipt() {
        let data = hex!(
            "f901668001b9010000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000f85ff85d940000000000000000000000000000000000000011f842a0000000000000000000000000000000000000000000000000000000000000deada0000000000000000000000000000000000000000000000000000000000000beef830100ff"
        );

        // EIP658Receipt
        let expected = OpDepositReceiptWithBloom {
            receipt: OpDepositReceipt {
                inner: Receipt {
                    status: false.into(),
                    cumulative_gas_used: 0x1,
                    logs: vec![Log {
                        address: address!("0000000000000000000000000000000000000011"),
                        data: LogData::new_unchecked(
                            vec![
                                b256!(
                                    "000000000000000000000000000000000000000000000000000000000000dead"
                                ),
                                b256!(
                                    "000000000000000000000000000000000000000000000000000000000000beef"
                                ),
                            ],
                            bytes!("0100ff"),
                        ),
                    }],
                },
                deposit_nonce: None,
                deposit_receipt_version: None,
            },
            logs_bloom: [0; 256].into(),
        };

        let receipt = OpDepositReceiptWithBloom::decode(&mut &data[..]).unwrap();
        assert_eq!(receipt, expected);
    }

    #[test]
    fn gigantic_receipt() {
        let receipt = OpDepositReceipt {
            inner: Receipt {
                cumulative_gas_used: 16747627,
                status: true.into(),
                logs: vec![
                    Log {
                        address: address!("4bf56695415f725e43c3e04354b604bcfb6dfb6e"),
                        data: LogData::new_unchecked(
                            vec![b256!(
                                "c69dc3d7ebff79e41f525be431d5cd3cc08f80eaf0f7819054a726eeb7086eb9"
                            )],
                            Bytes::from(vec![1; 0xffffff]),
                        ),
                    },
                    Log {
                        address: address!("faca325c86bf9c2d5b413cd7b90b209be92229c2"),
                        data: LogData::new_unchecked(
                            vec![b256!(
                                "8cca58667b1e9ffa004720ac99a3d61a138181963b294d270d91c53d36402ae2"
                            )],
                            Bytes::from(vec![1; 0xffffff]),
                        ),
                    },
                ],
            },
            deposit_nonce: None,
            deposit_receipt_version: None,
        }
        .with_bloom();

        let mut data = vec![];

        receipt.encode(&mut data);
        let decoded = OpDepositReceiptWithBloom::decode(&mut &data[..]).unwrap();

        // receipt.clone().to_compact(&mut data);
        // let (decoded, _) = Receipt::from_compact(&data[..], data.len());
        assert_eq!(decoded, receipt);
    }

    #[test]
    fn regolith_receipt_roundtrip() {
        let data = hex!(
            "f9010c0182b741b9010000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000c0833d3bbf"
        );

        // Deposit Receipt (post-regolith)
        let expected = OpDepositReceiptWithBloom {
            receipt: OpDepositReceipt {
                inner: Receipt::<Log> {
                    cumulative_gas_used: 46913,
                    logs: vec![],
                    status: true.into(),
                },
                deposit_nonce: Some(4012991),
                deposit_receipt_version: None,
            },
            logs_bloom: [0; 256].into(),
        };

        let receipt = OpDepositReceiptWithBloom::decode(&mut &data[..]).unwrap();
        assert_eq!(receipt, expected);

        let mut buf = Vec::new();
        receipt.encode(&mut buf);
        assert_eq!(buf, &data[..]);
    }

    #[test]
    fn post_canyon_receipt_roundtrip() {
        let data = hex!(
            "f9010d0182b741b9010000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000c0833d3bbf01"
        );

        // Deposit Receipt (post-regolith)
        let expected = OpDepositReceiptWithBloom {
            receipt: OpDepositReceipt {
                inner: Receipt::<Log> {
                    cumulative_gas_used: 46913,
                    logs: vec![],
                    status: true.into(),
                },
                deposit_nonce: Some(4012991),
                deposit_receipt_version: Some(1),
            },
            logs_bloom: [0; 256].into(),
        };

        let receipt = OpDepositReceiptWithBloom::decode(&mut &data[..]).unwrap();
        assert_eq!(receipt, expected);

        let mut buf = Vec::new();
        expected.encode(&mut buf);
        assert_eq!(buf, &data[..]);
    }
}
