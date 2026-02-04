//! Optimism-specific payload attributes.

use alloc::vec::Vec;
use alloy_eips::{
    Decodable2718,
    eip1559::BaseFeeParams,
    eip2718::{Eip2718Result, WithEncoded},
};
use alloy_primitives::{B64, B256, Bytes, keccak256};
use alloy_rlp::{Encodable, Result};
use alloy_rpc_types_engine::{PayloadAttributes, PayloadId};
use op_alloy_consensus::{
    EIP1559ParamError, OpTxEnvelope, decode_eip_1559_params, encode_holocene_extra_data,
    encode_jovian_extra_data,
};
use sha2::Digest;

/// Optimism Payload Attributes
#[derive(Clone, Debug, Default, PartialEq, Eq)]
#[cfg_attr(feature = "serde", derive(serde::Serialize, serde::Deserialize))]
#[cfg_attr(feature = "serde", serde(rename_all = "camelCase"))]
pub struct OpPayloadAttributes {
    /// The payload attributes
    #[cfg_attr(feature = "serde", serde(flatten))]
    pub payload_attributes: PayloadAttributes,
    /// Transactions is a field for rollups: the transactions list is forced into the block
    #[cfg_attr(feature = "serde", serde(default, skip_serializing_if = "Option::is_none"))]
    pub transactions: Option<Vec<Bytes>>,
    /// If true, the no transactions are taken out of the tx-pool, only transactions from the above
    /// Transactions list will be included.
    #[cfg_attr(feature = "serde", serde(default, skip_serializing_if = "Option::is_none"))]
    pub no_tx_pool: Option<bool>,
    /// If set, this sets the exact gas limit the block produced with.
    #[cfg_attr(
        feature = "serde",
        serde(
            default,
            skip_serializing_if = "Option::is_none",
            with = "alloy_serde::quantity::opt"
        )
    )]
    pub gas_limit: Option<u64>,
    /// If set, this sets the EIP-1559 parameters for the block.
    ///
    /// Prior to Holocene activation, this field should always be [None].
    #[cfg_attr(feature = "serde", serde(default, skip_serializing_if = "Option::is_none"))]
    pub eip_1559_params: Option<B64>,
    /// If set, this sets the minimum base fee for the block.
    ///
    /// Prior to Jovian activation, this field should always be [None].
    #[cfg_attr(feature = "serde", serde(default, skip_serializing_if = "Option::is_none"))]
    pub min_base_fee: Option<u64>,
}

impl OpPayloadAttributes {
    /// Generates the payload id for the configured payload from the [`OpPayloadAttributes`].
    ///
    /// Returns an 8-byte identifier by hashing the payload components with sha256 hash.
    ///
    /// Note: This must be updated whenever the [`OpPayloadAttributes`] changes for a hardfork.
    /// See also <https://github.com/ethereum-optimism/op-geth/blob/d401af16f2dd94b010a72eaef10e07ac10b31931/miner/payload_building.go#L59-L59>
    pub fn payload_id(&self, parent: &B256, payload_version: u8) -> PayloadId {
        let mut hasher = sha2::Sha256::new();
        hasher.update(parent.as_slice());
        hasher.update(&self.payload_attributes.timestamp.to_be_bytes()[..]);
        hasher.update(self.payload_attributes.prev_randao.as_slice());
        hasher.update(self.payload_attributes.suggested_fee_recipient.as_slice());
        if let Some(withdrawals) = &self.payload_attributes.withdrawals {
            let mut buf = Vec::new();
            withdrawals.encode(&mut buf);
            hasher.update(buf);
        }

        if let Some(parent_beacon_block) = self.payload_attributes.parent_beacon_block_root {
            hasher.update(parent_beacon_block);
        }

        let no_tx_pool = self.no_tx_pool.unwrap_or_default();
        if no_tx_pool || self.transactions.as_ref().is_some_and(|txs| !txs.is_empty()) {
            hasher.update([no_tx_pool as u8]);
            let txs_len = self.transactions.as_ref().map(|txs| txs.len()).unwrap_or_default();
            hasher.update(&txs_len.to_be_bytes()[..]);
            if let Some(txs) = &self.transactions {
                for tx in txs {
                    // we have to just hash the bytes here because otherwise we would need to decode
                    // the transactions here which really isn't ideal
                    let tx_hash = keccak256(tx);
                    // maybe we can try just taking the hash and not decoding
                    hasher.update(tx_hash)
                }
            }
        }

        if let Some(gas_limit) = self.gas_limit {
            hasher.update(gas_limit.to_be_bytes());
        }

        if let Some(eip_1559_params) = self.eip_1559_params {
            hasher.update(eip_1559_params.as_slice());
        }

        if let Some(min_base_fee) = self.min_base_fee {
            hasher.update(min_base_fee.to_be_bytes());
        }

        let mut out = hasher.finalize();
        out[0] = payload_version;
        PayloadId::new(out[..8].try_into().expect("sufficient length"))
    }

    /// Encodes the `eip1559` parameters for the payload.
    pub fn get_holocene_extra_data(
        &self,
        default_base_fee_params: BaseFeeParams,
    ) -> Result<Bytes, EIP1559ParamError> {
        if self.min_base_fee.is_some() {
            return Err(EIP1559ParamError::MinBaseFeeMustBeNone);
        }
        self.eip_1559_params
            .map(|params| encode_holocene_extra_data(params, default_base_fee_params))
            .ok_or(EIP1559ParamError::NoEIP1559Params)?
    }

    /// Extracts the Holocene 1599 parameters from the encoded form:
    /// <https://github.com/ethereum-optimism/specs/blob/main/specs/protocol/holocene/exec-engine.md#eip1559params-encoding>
    ///
    /// Returns (`elasticity`, `denominator`)
    pub fn decode_eip_1559_params(&self) -> Option<(u32, u32)> {
        self.eip_1559_params.map(decode_eip_1559_params)
    }

    /// Encodes the `eip1559` parameters for the payload along with the minimum base fee.
    pub fn get_jovian_extra_data(
        &self,
        default_base_fee_params: BaseFeeParams,
    ) -> Result<Bytes, EIP1559ParamError> {
        if self.min_base_fee.is_none() {
            return Err(EIP1559ParamError::MinBaseFeeNotSet);
        }
        self.eip_1559_params
            .map(|params| {
                encode_jovian_extra_data(
                    params,
                    default_base_fee_params,
                    self.min_base_fee.unwrap(),
                )
            })
            .ok_or(EIP1559ParamError::NoEIP1559Params)?
    }

    /// Returns an iterator over the decoded [`OpTxEnvelope`] in this attributes.
    ///
    /// This iterator will be empty if there are no transactions in the attributes.
    pub fn decoded_transactions(&self) -> impl Iterator<Item = Eip2718Result<OpTxEnvelope>> + '_ {
        self.transactions.iter().flatten().map(|tx_bytes| {
            let mut buf = tx_bytes.as_ref();
            let tx = OpTxEnvelope::decode_2718(&mut buf).map_err(alloy_rlp::Error::from)?;
            if !buf.is_empty() {
                return Err(alloy_rlp::Error::UnexpectedLength.into());
            }
            Ok(tx)
        })
    }

    /// Returns iterator over decoded transactions with their original encoded bytes.
    ///
    /// This iterator will be empty if there are no transactions in the attributes.
    pub fn decoded_transactions_with_encoded(
        &self,
    ) -> impl Iterator<Item = Eip2718Result<WithEncoded<OpTxEnvelope>>> + '_ {
        self.transactions
            .iter()
            .flatten()
            .cloned()
            .zip(self.decoded_transactions())
            .map(|(tx_bytes, result)| result.map(|op_tx| WithEncoded::new(tx_bytes, op_tx)))
    }

    /// Returns an iterator over the recovered [`OpTxEnvelope`] in this attributes.
    ///
    /// This iterator will be empty if there are no transactions in the attributes.
    #[cfg(feature = "k256")]
    pub fn recovered_transactions(
        &self,
    ) -> impl Iterator<
        Item = Result<
            alloy_consensus::transaction::Recovered<OpTxEnvelope>,
            alloy_consensus::crypto::RecoveryError,
        >,
    > + '_ {
        use alloy_consensus::transaction::SignerRecoverable;

        self.decoded_transactions().map(|res| {
            res.map_err(alloy_consensus::crypto::RecoveryError::from_source)
                .and_then(|tx| tx.try_into_recovered())
        })
    }

    /// Returns an iterator over the recovered [`OpTxEnvelope`] in this attributes with their
    /// original encoded bytes.
    ///
    /// This iterator will be empty if there are no transactions in the attributes.
    #[cfg(feature = "k256")]
    pub fn recovered_transactions_with_encoded(
        &self,
    ) -> impl Iterator<
        Item = Result<
            WithEncoded<alloy_consensus::transaction::Recovered<OpTxEnvelope>>,
            alloy_consensus::crypto::RecoveryError,
        >,
    > + '_ {
        self.transactions
            .iter()
            .flatten()
            .cloned()
            .zip(self.recovered_transactions())
            .map(|(tx_bytes, result)| result.map(|op_tx| WithEncoded::new(tx_bytes, op_tx)))
    }
}

#[cfg(all(test, feature = "serde"))]
mod test {
    use super::*;
    use alloc::vec;
    use alloy_primitives::{Address, B256, FixedBytes, address, b64, b256, bytes};
    use alloy_rpc_types_engine::PayloadAttributes;
    use core::str::FromStr;

    #[test]
    fn test_payload_id_parity_op_geth() {
        const PAYLOAD_VERSION: u8 = 3;

        // INFO rollup_boost::server:received fork_choice_updated_v3 from builder and l2_client
        // payload_id_builder="0x6ef26ca02318dcf9" payload_id_l2="0x03d2dae446d2a86a"
        let expected =
            PayloadId::new(FixedBytes::<8>::from_str("0x03d2dae446d2a86a").unwrap().into());
        let attrs = OpPayloadAttributes {
            payload_attributes: PayloadAttributes {
                timestamp: 1728933301,
                prev_randao: b256!("0x9158595abbdab2c90635087619aa7042bbebe47642dfab3c9bfb934f6b082765"),
                suggested_fee_recipient: address!("0x4200000000000000000000000000000000000011"),
                withdrawals: Some([].into()),
                parent_beacon_block_root: b256!("0x8fe0193b9bf83cb7e5a08538e494fecc23046aab9a497af3704f4afdae3250ff").into(),
            },
            transactions: Some([bytes!("7ef8f8a0dc19cfa777d90980e4875d0a548a881baaa3f83f14d1bc0d3038bc329350e54194deaddeaddeaddeaddeaddeaddeaddeaddead00019442000000000000000000000000000000000000158080830f424080b8a4440a5e20000f424000000000000000000000000300000000670d6d890000000000000125000000000000000000000000000000000000000000000000000000000000000700000000000000000000000000000000000000000000000000000000000000014bf9181db6e381d4384bbf69c48b0ee0eed23c6ca26143c6d2544f9d39997a590000000000000000000000007f83d659683caf2767fd3c720981d51f5bc365bc")].into()),
            no_tx_pool: None,
            gas_limit: Some(30000000),
            eip_1559_params: None,
            min_base_fee: None,
        };

        // Reth's `PayloadId` should match op-geth's `PayloadId`. This fails
        assert_eq!(
            expected,
            attrs.payload_id(
                &b256!("0x3533bf30edaf9505d0810bf475cbe4e5f4b9889904b9845e83efdeab4e92eb1e"),
                // Payload version
                PAYLOAD_VERSION
            )
        );
    }

    #[test]
    fn test_payload_id_parity_op_geth_jovian() {
        // <https://github.com/ethereum-optimism/op-geth/compare/optimism...mattsse:op-geth:matt/check-payload-id-equality>
        const PAYLOAD_VERSION: u8 = 4;

        let expected =
            PayloadId::new(FixedBytes::<8>::from_str("0x046c65ffc4d659ec").unwrap().into());
        let attrs = OpPayloadAttributes {
            payload_attributes: PayloadAttributes {
                timestamp: 1728933301,
                prev_randao: b256!("0x9158595abbdab2c90635087619aa7042bbebe47642dfab3c9bfb934f6b082765"),
                suggested_fee_recipient: address!("0x4200000000000000000000000000000000000011"),
                withdrawals: Some([].into()),
                parent_beacon_block_root: b256!("0x8fe0193b9bf83cb7e5a08538e494fecc23046aab9a497af3704f4afdae3250ff").into(),
            },
            transactions: Some([bytes!("7ef8f8a0dc19cfa777d90980e4875d0a548a881baaa3f83f14d1bc0d3038bc329350e54194deaddeaddeaddeaddeaddeaddeaddeaddead00019442000000000000000000000000000000000000158080830f424080b8a4440a5e20000f424000000000000000000000000300000000670d6d890000000000000125000000000000000000000000000000000000000000000000000000000000000700000000000000000000000000000000000000000000000000000000000000014bf9181db6e381d4384bbf69c48b0ee0eed23c6ca26143c6d2544f9d39997a590000000000000000000000007f83d659683caf2767fd3c720981d51f5bc365bc")].into()),
            no_tx_pool: None,
            gas_limit: Some(30000000),
            eip_1559_params: None,
            min_base_fee: Some(100),
        };

        // Reth's `PayloadId` should match op-geth's `PayloadId`. This fails
        assert_eq!(
            expected,
            attrs.payload_id(
                &b256!("0x3533bf30edaf9505d0810bf475cbe4e5f4b9889904b9845e83efdeab4e92eb1e"),
                // Payload version
                PAYLOAD_VERSION
            )
        );
    }

    #[test]
    fn test_serde_roundtrip_attributes_pre_holocene() {
        let attributes = OpPayloadAttributes {
            payload_attributes: PayloadAttributes {
                timestamp: 0x1337,
                prev_randao: B256::ZERO,
                suggested_fee_recipient: Address::ZERO,
                withdrawals: Default::default(),
                parent_beacon_block_root: Some(B256::ZERO),
            },
            transactions: Some(vec![b"hello".to_vec().into()]),
            no_tx_pool: Some(true),
            gas_limit: Some(42),
            eip_1559_params: None,
            min_base_fee: None,
        };

        let ser = serde_json::to_string(&attributes).unwrap();
        let de: OpPayloadAttributes = serde_json::from_str(&ser).unwrap();

        assert_eq!(attributes, de);
    }

    #[test]
    fn test_serde_roundtrip_attributes_post_holocene() {
        let attributes = OpPayloadAttributes {
            payload_attributes: PayloadAttributes {
                timestamp: 0x1337,
                prev_randao: B256::ZERO,
                suggested_fee_recipient: Address::ZERO,
                withdrawals: Default::default(),
                parent_beacon_block_root: Some(B256::ZERO),
            },
            transactions: Some(vec![b"hello".to_vec().into()]),
            no_tx_pool: Some(true),
            gas_limit: Some(42),
            eip_1559_params: Some(b64!("0000dead0000beef")),
            min_base_fee: None,
        };

        let ser = serde_json::to_string(&attributes).unwrap();
        let de: OpPayloadAttributes = serde_json::from_str(&ser).unwrap();

        assert_eq!(attributes, de);
    }

    #[test]
    fn test_get_extra_data_post_holocene() {
        let attributes = OpPayloadAttributes {
            eip_1559_params: Some(B64::from_str("0x0000000800000008").unwrap()),
            ..Default::default()
        };
        let extra_data = attributes.get_holocene_extra_data(BaseFeeParams::new(80, 60));
        assert_eq!(extra_data.unwrap(), Bytes::copy_from_slice(&[0, 0, 0, 0, 8, 0, 0, 0, 8]));
    }

    #[test]
    fn test_get_extra_data_post_holocene_default() {
        let attributes =
            OpPayloadAttributes { eip_1559_params: Some(B64::ZERO), ..Default::default() };
        let extra_data = attributes.get_holocene_extra_data(BaseFeeParams::new(80, 60));
        assert_eq!(extra_data.unwrap(), Bytes::copy_from_slice(&[0, 0, 0, 0, 80, 0, 0, 0, 60]));
    }

    #[test]
    fn test_serde_roundtrip_attributes_pre_jovian() {
        let attributes = OpPayloadAttributes {
            payload_attributes: PayloadAttributes {
                timestamp: 0x1337,
                prev_randao: B256::ZERO,
                suggested_fee_recipient: Address::ZERO,
                withdrawals: Default::default(),
                parent_beacon_block_root: Some(B256::ZERO),
            },
            transactions: Some(vec![b"hello".to_vec().into()]),
            no_tx_pool: Some(true),
            gas_limit: Some(42),
            eip_1559_params: Some(b64!("0000dead0000beef")),
            min_base_fee: None,
        };

        let ser = serde_json::to_string(&attributes).unwrap();
        let de: OpPayloadAttributes = serde_json::from_str(&ser).unwrap();

        assert_eq!(attributes, de);
    }

    #[test]
    fn test_serde_roundtrip_attributes_post_jovian() {
        let attributes = OpPayloadAttributes {
            payload_attributes: PayloadAttributes {
                timestamp: 0x1337,
                prev_randao: B256::ZERO,
                suggested_fee_recipient: Address::ZERO,
                withdrawals: Default::default(),
                parent_beacon_block_root: Some(B256::ZERO),
            },
            transactions: Some(vec![b"hello".to_vec().into()]),
            no_tx_pool: Some(true),
            gas_limit: Some(42),
            eip_1559_params: None,
            min_base_fee: Some(1),
        };

        let ser = serde_json::to_string(&attributes).unwrap();
        let de: OpPayloadAttributes = serde_json::from_str(&ser).unwrap();

        assert_eq!(attributes, de);
    }

    #[test]
    fn test_get_extra_data_post_jovian() {
        let attributes = OpPayloadAttributes {
            eip_1559_params: Some(B64::from_str("0x0000000800000008").unwrap()),
            min_base_fee: Some(257),
            ..Default::default()
        };

        let extra_data = attributes.get_jovian_extra_data(BaseFeeParams::new(80, 60));
        assert_eq!(
            extra_data.unwrap(),
            Bytes::copy_from_slice(&[1, 0, 0, 0, 8, 0, 0, 0, 8, 0, 0, 0, 0, 0, 0, 1, 1])
        );
    }

    #[test]
    fn test_get_extra_data_post_jovian_default() {
        let attributes = OpPayloadAttributes {
            eip_1559_params: Some(B64::ZERO),
            min_base_fee: Some(0),
            ..Default::default()
        };

        let extra_data = attributes.get_jovian_extra_data(BaseFeeParams::new(80, 60));
        assert_eq!(
            extra_data.unwrap(),
            Bytes::copy_from_slice(&[1, 0, 0, 0, 80, 0, 0, 0, 60, 0, 0, 0, 0, 0, 0, 0, 0])
        );
    }

    #[test]
    fn test_get_jovian_extra_data_fails_without_min_base_fee() {
        let attributes = OpPayloadAttributes {
            eip_1559_params: Some(B64::from_str("0x0000000800000008").unwrap()),
            min_base_fee: None,
            ..Default::default()
        };

        let result = attributes.get_jovian_extra_data(BaseFeeParams::new(80, 60));
        assert_eq!(result.unwrap_err(), EIP1559ParamError::MinBaseFeeNotSet);
    }

    #[test]
    fn test_min_base_fee_must_be_none_before_jovian() {
        let attributes = OpPayloadAttributes {
            eip_1559_params: Some(B64::from_str("0x0000000800000008").unwrap()),
            min_base_fee: Some(100),
            ..Default::default()
        };

        // Use Holocene function for pre-Jovian decoding of extra data
        let result = attributes.get_holocene_extra_data(BaseFeeParams::new(80, 60));
        assert_eq!(result.unwrap_err(), EIP1559ParamError::MinBaseFeeMustBeNone);
    }

    // <https://github.com/alloy-rs/op-alloy/issues/601>
    #[test]
    fn test_serde_attributes() {
        let json = r#"{"timestamp":"0x68e8f68b","prevRandao":"0x0c00c066d51a9cd87d962de52da13e9dfd7f08d507601f916e116aacfe370de7","suggestedFeeRecipient":"0x4200000000000000000000000000000000000011","withdrawals":[],"parentBeaconBlockRoot":"0x6d9579b008332936037f0167d74af108db1fbe22d1bd6552f2d4453419afa4e2","transactions":["0x7ef8f8a058642a460a8c2fb85bae8237221cfa2f138777697c73052afe5adbd911ec9f5194deaddeaddeaddeaddeaddeaddeaddeaddead00019442000000000000000000000000000000000000158080830f424080b8a4440a5e2000000558000c5fc500000000000000010000000068e8f688000000000000000d000000000000000000000000000000000000000000000000000000000a9dd4be0000000000000000000000000000000000000000000000000000000000000001844ea1c8f674542957f8fd73f34545ed30d24ebfa80775b869ea8848a6f38259000000000000000000000000aff0ca253b97e54440965855cec0a8a2e2399896"],"eip1559Params":"0x000000fa00000006"}"#;

        let attributes: OpPayloadAttributes = serde_json::from_str(json).unwrap();
        let val = serde_json::to_value(&attributes).unwrap();
        let round_trip: OpPayloadAttributes = serde_json::from_value(val).unwrap();
        assert_eq!(attributes, round_trip);
    }
}
