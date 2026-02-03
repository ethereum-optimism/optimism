//! Contains utilities for the L2 executor.

use crate::{Eip1559ValidationError, ExecutorError, ExecutorResult};
use alloy_consensus::{BlockHeader, Header};
use alloy_eips::eip1559::BaseFeeParams;
use alloy_primitives::Bytes;
use kona_genesis::RollupConfig;
use op_alloy_consensus::{
    EIP1559ParamError, decode_holocene_extra_data, decode_jovian_extra_data,
    encode_holocene_extra_data, encode_jovian_extra_data,
};
use op_alloy_rpc_types_engine::OpPayloadAttributes;

/// Parse Holocene [Header] extra data from the block header.
///
/// ## Takes
/// - `extra_data`: The extra data field of the [Header].
///
/// ## Returns
/// - `Ok(BaseFeeParams)`: The EIP-1559 parameters.
/// - `Err(ExecutorError::InvalidExtraData)`: If the extra data is invalid.
pub(crate) fn decode_holocene_eip_1559_params_block_header(
    header: &Header,
) -> ExecutorResult<BaseFeeParams> {
    let (elasticity, denominator) = decode_holocene_extra_data(header.extra_data())?;

    // Check for potential division by zero.
    // In the block header, the denominator is always non-zero.
    // <https://github.com/ethereum-optimism/specs/blob/main/specs/protocol/holocene/exec-engine.md#eip-1559-parameters-in-block-header>
    if denominator == 0 {
        return Err(ExecutorError::InvalidExtraData(Eip1559ValidationError::ZeroDenominator));
    }

    Ok(BaseFeeParams {
        elasticity_multiplier: elasticity.into(),
        max_change_denominator: denominator.into(),
    })
}

pub(crate) fn decode_jovian_eip_1559_params_block_header(
    header: &Header,
) -> ExecutorResult<(BaseFeeParams, u64)> {
    let (elasticity, denominator, min_base_fee) = decode_jovian_extra_data(header.extra_data())?;

    // Check for potential division by zero.
    // In the block header, the denominator is always non-zero.
    // <https://github.com/ethereum-optimism/specs/blob/main/specs/protocol/holocene/exec-engine.md#eip-1559-parameters-in-block-header>
    if denominator == 0 {
        return Err(ExecutorError::InvalidExtraData(Eip1559ValidationError::ZeroDenominator));
    }

    Ok((
        BaseFeeParams {
            elasticity_multiplier: elasticity.into(),
            max_change_denominator: denominator.into(),
        },
        min_base_fee,
    ))
}

/// Encode Holocene [Header] extra data.
///
/// ## Takes
/// - `config`: The [RollupConfig] for the chain.
/// - `attributes`: The [OpPayloadAttributes] for the block.
///
/// ## Returns
/// - `Ok(data)`: The encoded extra data.
/// - `Err(ExecutorError::MissingEIP1559Params)`: If the EIP-1559 parameters are missing.
pub(crate) fn encode_holocene_eip_1559_params(
    config: &RollupConfig,
    attributes: &OpPayloadAttributes,
) -> ExecutorResult<Bytes> {
    Ok(encode_holocene_extra_data(
        attributes.eip_1559_params.ok_or(ExecutorError::MissingEIP1559Params)?,
        config.chain_op_config.post_canyon_params(),
    )?)
}

/// Encode Jovian [Header] extra data.
///
/// ## Takes
/// - `config`: The [RollupConfig] for the chain.
/// - `attributes`: The [OpPayloadAttributes] for the block.
///
/// ## Returns
/// - `Ok(data)`: The encoded extra data.
/// - `Err(ExecutorError::MissingEIP1559Params)`: If the EIP-1559 parameters are missing.
pub(crate) fn encode_jovian_eip_1559_params(
    config: &RollupConfig,
    attributes: &OpPayloadAttributes,
) -> ExecutorResult<Bytes> {
    Ok(encode_jovian_extra_data(
        attributes.eip_1559_params.ok_or(ExecutorError::MissingEIP1559Params)?,
        config.chain_op_config.post_canyon_params(),
        attributes.min_base_fee.ok_or(ExecutorError::InvalidExtraData(
            Eip1559ValidationError::Decode(EIP1559ParamError::MinBaseFeeNotSet),
        ))?,
    )?)
}

#[cfg(test)]
mod test {
    use super::decode_holocene_eip_1559_params_block_header;
    use crate::util::{
        decode_jovian_eip_1559_params_block_header, encode_holocene_eip_1559_params,
    };
    use alloy_consensus::Header;
    use alloy_primitives::{B64, b64, bytes};
    use alloy_rpc_types_engine::PayloadAttributes;
    use kona_genesis::{BaseFeeConfig, RollupConfig};
    use op_alloy_rpc_types_engine::OpPayloadAttributes;

    fn mock_payload(eip_1559_params: Option<B64>) -> OpPayloadAttributes {
        OpPayloadAttributes {
            payload_attributes: PayloadAttributes {
                timestamp: 0,
                prev_randao: Default::default(),
                suggested_fee_recipient: Default::default(),
                withdrawals: Default::default(),
                parent_beacon_block_root: Default::default(),
            },
            transactions: None,
            no_tx_pool: None,
            gas_limit: None,
            eip_1559_params,
            min_base_fee: None,
        }
    }

    #[test]
    fn test_decode_holocene_eip_1559_params() {
        let params = bytes!("00BEEFBABE0BADC0DE");
        let mock_header = Header { extra_data: params, ..Default::default() };
        let params = decode_holocene_eip_1559_params_block_header(&mock_header).unwrap();

        assert_eq!(params.elasticity_multiplier, 0x0BAD_C0DE);
        assert_eq!(params.max_change_denominator, 0xBEEF_BABE);
    }

    #[test]
    fn test_decode_jovian_eip_1559_params() {
        let params = bytes!("01BEEFBABE0BADC0DE00000000DEADBEEF");
        let mock_header = Header { extra_data: params, ..Default::default() };
        let (params, base_fee) = decode_jovian_eip_1559_params_block_header(&mock_header).unwrap();

        assert_eq!(params.elasticity_multiplier, 0x0BAD_C0DE);
        assert_eq!(params.max_change_denominator, 0xBEEF_BABE);
        assert_eq!(base_fee, 0xDEAD_BEEF);
    }

    #[test]
    fn test_decode_holocene_eip_1559_params_invalid_version() {
        let params = bytes!("01BEEFBABE0BADC0DE");
        let mock_header = Header { extra_data: params, ..Default::default() };
        assert!(decode_holocene_eip_1559_params_block_header(&mock_header).is_err());
    }

    #[test]
    fn test_decode_holocene_eip_1559_params_invalid_denominator() {
        let params = bytes!("00000000000BADC0DE");
        let mock_header = Header { extra_data: params, ..Default::default() };
        assert!(decode_holocene_eip_1559_params_block_header(&mock_header).is_err());
    }

    #[test]
    fn test_decode_holocene_eip_1559_params_invalid_length() {
        let params = bytes!("00");
        let mock_header = Header { extra_data: params, ..Default::default() };
        assert!(decode_holocene_eip_1559_params_block_header(&mock_header).is_err());
    }

    #[test]
    fn test_encode_holocene_eip_1559_params_missing() {
        let cfg = RollupConfig {
            chain_op_config: BaseFeeConfig {
                eip1559_denominator: 50,
                eip1559_elasticity: 64,
                eip1559_denominator_canyon: 250,
            },
            ..Default::default()
        };
        let attrs = mock_payload(None);

        assert!(encode_holocene_eip_1559_params(&cfg, &attrs).is_err());
    }

    #[test]
    fn test_encode_holocene_eip_1559_params_default() {
        let cfg = RollupConfig {
            chain_op_config: BaseFeeConfig {
                eip1559_denominator: 50,
                eip1559_elasticity: 64,
                eip1559_denominator_canyon: 250,
            },
            ..Default::default()
        };
        let attrs = mock_payload(Some(B64::ZERO));

        assert_eq!(
            encode_holocene_eip_1559_params(&cfg, &attrs).unwrap(),
            bytes!("00000000fa00000040")
        );
    }

    #[test]
    fn test_encode_holocene_eip_1559_params() {
        let cfg = RollupConfig {
            chain_op_config: BaseFeeConfig {
                eip1559_denominator: 50,
                eip1559_elasticity: 64,
                eip1559_denominator_canyon: 250,
            },
            ..Default::default()
        };
        let attrs = mock_payload(Some(b64!("0000004000000060")));

        assert_eq!(
            encode_holocene_eip_1559_params(&cfg, &attrs).unwrap(),
            bytes!("000000004000000060")
        );
    }
}
