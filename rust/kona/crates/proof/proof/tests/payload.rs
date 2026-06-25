//! Tests for shared payload attribute construction helpers.

use alloy_consensus::Header;
use alloy_primitives::{Address, B64, B256, Bytes};
use kona_genesis::{HardForkConfig, RollupConfig};
use kona_proof::{PayloadAttributesError, payload_attributes_from_block_header};

#[test]
fn payload_attributes_from_block_header_sets_post_canyon_fields() {
    let header = Header {
        timestamp: 10,
        mix_hash: B256::repeat_byte(0x11),
        beneficiary: Address::repeat_byte(0x22),
        gas_limit: 30_000_000,
        parent_beacon_block_root: Some(B256::repeat_byte(0x33)),
        extra_data: Bytes::from_static(&[1, 0, 0, 0, 8, 0, 0, 0, 8, 0, 0, 0, 0, 0, 0, 0, 123]),
        ..Default::default()
    };
    let transactions = vec![Bytes::from_static(&[0x7e])];
    let rollup_config = RollupConfig {
        hardforks: HardForkConfig {
            canyon_time: Some(0),
            holocene_time: Some(0),
            jovian_time: Some(0),
            ..Default::default()
        },
        ..Default::default()
    };

    let attrs = payload_attributes_from_block_header(&header, transactions.clone(), &rollup_config)
        .unwrap();

    assert_eq!(attrs.payload_attributes.timestamp, header.timestamp);
    assert_eq!(attrs.payload_attributes.prev_randao, header.mix_hash);
    assert_eq!(attrs.payload_attributes.suggested_fee_recipient, header.beneficiary);
    assert_eq!(attrs.payload_attributes.withdrawals, Some(Vec::new()));
    assert_eq!(attrs.payload_attributes.parent_beacon_block_root, header.parent_beacon_block_root);
    assert_eq!(attrs.transactions, Some(transactions));
    assert_eq!(attrs.no_tx_pool, Some(true));
    assert_eq!(attrs.gas_limit, Some(header.gas_limit));
    assert_eq!(attrs.eip_1559_params, Some(B64::from_slice(&header.extra_data[1..9])));
    assert_eq!(attrs.min_base_fee, Some(123));
}

#[test]
fn payload_attributes_from_block_header_leaves_pre_holocene_fields_empty() {
    let header = Header { timestamp: 10, ..Default::default() };
    let rollup_config = RollupConfig::default();

    let attrs = payload_attributes_from_block_header(&header, Vec::new(), &rollup_config).unwrap();

    assert_eq!(attrs.payload_attributes.withdrawals, None);
    assert_eq!(attrs.eip_1559_params, None);
    assert_eq!(attrs.min_base_fee, None);
}

#[test]
fn payload_attributes_from_block_header_rejects_short_holocene_extra_data() {
    let header =
        Header { timestamp: 10, extra_data: Bytes::from_static(&[0]), ..Default::default() };
    let rollup_config = RollupConfig {
        hardforks: HardForkConfig { holocene_time: Some(0), ..Default::default() },
        ..Default::default()
    };

    let err =
        payload_attributes_from_block_header(&header, Vec::new(), &rollup_config).unwrap_err();

    assert_eq!(err, PayloadAttributesError::HoloceneExtraDataTooShort { len: 1 });
}
