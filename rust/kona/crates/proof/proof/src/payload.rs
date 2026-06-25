//! Payload attribute construction helpers.

use alloc::vec::Vec;

use alloy_consensus::Header;
use alloy_primitives::{B64, Bytes};
use kona_genesis::RollupConfig;
use op_alloy_rpc_types_engine::OpPayloadAttributes;
use thiserror::Error;

/// Error returned when block-header fields are insufficient to reconstruct payload attributes.
#[derive(Debug, Error, Clone, Copy, PartialEq, Eq)]
pub enum PayloadAttributesError {
    /// Holocene payload attributes require the encoded EIP-1559 params in header extra data.
    #[error("Holocene block header extra data is too short: {len} bytes")]
    HoloceneExtraDataTooShort {
        /// The observed extra-data length.
        len: usize,
    },
    /// Jovian payload attributes require the encoded min-base-fee field in header extra data.
    #[error("Jovian block header extra data is too short: {len} bytes")]
    JovianExtraDataTooShort {
        /// The observed extra-data length.
        len: usize,
    },
}

/// Reconstructs the payload attributes that produced `header`.
pub fn payload_attributes_from_block_header(
    header: &Header,
    transactions: Vec<Bytes>,
    rollup_config: &RollupConfig,
) -> Result<OpPayloadAttributes, PayloadAttributesError> {
    let mut payload_attributes = OpPayloadAttributes::default();
    payload_attributes.payload_attributes.timestamp = header.timestamp;
    payload_attributes.payload_attributes.prev_randao = header.mix_hash;
    payload_attributes.payload_attributes.suggested_fee_recipient = header.beneficiary;
    payload_attributes.payload_attributes.withdrawals =
        rollup_config.is_canyon_active(header.timestamp).then(Vec::new);
    payload_attributes.payload_attributes.parent_beacon_block_root =
        header.parent_beacon_block_root;
    payload_attributes.payload_attributes.slot_number = header.slot_number;
    payload_attributes.transactions = Some(transactions);
    payload_attributes.no_tx_pool = Some(true);
    payload_attributes.gas_limit = Some(header.gas_limit);

    if rollup_config.is_holocene_active(header.timestamp) {
        if header.extra_data.len() < 9 {
            return Err(PayloadAttributesError::HoloceneExtraDataTooShort {
                len: header.extra_data.len(),
            });
        }
        payload_attributes.eip_1559_params = Some(B64::from_slice(&header.extra_data[1..9]));
    }
    if rollup_config.is_jovian_active(header.timestamp) {
        if header.extra_data.len() < 17 {
            return Err(PayloadAttributesError::JovianExtraDataTooShort {
                len: header.extra_data.len(),
            });
        }

        let mut min_base_fee = [0u8; 8];
        min_base_fee.copy_from_slice(&header.extra_data[9..17]);
        payload_attributes.min_base_fee = Some(u64::from_be_bytes(min_base_fee));
    }

    Ok(payload_attributes)
}
