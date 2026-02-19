use alloc::vec::Vec;
use alloy_consensus::{
    Eip658Value, Eip2718EncodableReceipt, Receipt, ReceiptWithBloom, RlpDecodableReceipt,
    RlpEncodableReceipt, TxReceipt, Typed2718,
};
use alloy_eips::{
    Decodable2718, Encodable2718,
    eip2718::{Eip2718Result, IsTyped2718},
};
use alloy_primitives::{Bloom, Log};
use alloy_rlp::{BufMut, Decodable, Encodable, Header};
use op_alloy_consensus::{OpDepositReceipt, OpReceipt, OpTxType};
use reth_primitives_traits::InMemorySize;

/// Trait for deposit receipt.
pub trait DepositReceipt: reth_primitives_traits::Receipt {
    /// Converts a `Receipt` into a mutable Optimism deposit receipt.
    fn as_deposit_receipt_mut(&mut self) -> Option<&mut OpDepositReceipt>;

    /// Extracts an Optimism deposit receipt from `Receipt`.
    fn as_deposit_receipt(&self) -> Option<&OpDepositReceipt>;
}

impl DepositReceipt for OpReceipt {
    fn as_deposit_receipt_mut(&mut self) -> Option<&mut OpDepositReceipt> {
        match self {
            Self::Deposit(receipt) => Some(receipt),
            _ => None,
        }
    }

    fn as_deposit_receipt(&self) -> Option<&OpDepositReceipt> {
        match self {
            Self::Deposit(receipt) => Some(receipt),
            _ => None,
        }
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
    /// use reth_optimism_primitives::OpReceipt;
    /// use reth_primitives_traits::serde_bincode_compat::SerdeBincodeCompat;
    /// use serde::{Deserialize, Serialize, de::DeserializeOwned};
    /// use serde_with::serde_as;
    ///
    /// #[serde_as]
    /// #[derive(Serialize, Deserialize)]
    /// struct Data {
    ///     #[serde_as(
    ///         as = "reth_primitives_traits::serde_bincode_compat::BincodeReprFor<'_, OpReceipt>"
    ///     )]
    ///     receipt: OpReceipt,
    /// }
    /// ```
    #[allow(rustdoc::private_doc_tests)]
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

    #[cfg(test)]
    mod tests {
        use crate::{OpReceipt, receipt::serde_bincode_compat};
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
                receipt: OpReceipt,
            }

            let mut bytes = [0u8; 1024];
            rand::rng().fill(bytes.as_mut_slice());
            let mut data = Data {
                receipt: OpReceipt::arbitrary(&mut arbitrary::Unstructured::new(&bytes)).unwrap(),
            };
            let success = data.receipt.as_receipt_mut().status.coerce_status();
            // // ensure we don't have an invalid poststate variant
            data.receipt.as_receipt_mut().status = success.into();

            let encoded = bincode::serde::encode_to_vec(&data, bincode::config::legacy()).unwrap();
            let (decoded, _) =
                bincode::serde::decode_from_slice::<Data, _>(&encoded, bincode::config::legacy())
                    .unwrap();
            assert_eq!(decoded, data);
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use alloy_eips::eip2718::Encodable2718;
    use alloy_primitives::{Bytes, address, b256, bytes, hex_literal::hex};
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
            receipt: OpReceipt::Legacy(Receipt::<Log> {
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
            receipt: OpReceipt::Legacy(Receipt::<Log> {
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
                inner: Receipt::<Log> {
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

        // Deposit Receipt (post-canyon)
        let expected = ReceiptWithBloom {
            receipt: OpReceipt::Deposit(OpDepositReceipt {
                inner: Receipt::<Log> {
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
        let receipt = OpReceipt::Legacy(Receipt::<Log> {
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
            receipt: OpReceipt::Eip1559(Receipt::<Log> {
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
            receipt: OpReceipt::Legacy(Receipt::<Log> {
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
