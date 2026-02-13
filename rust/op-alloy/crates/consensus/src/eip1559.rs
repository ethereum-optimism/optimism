//! Support for EIP-1559 parameters after holocene.

use alloy_eips::eip1559::BaseFeeParams;
use alloy_primitives::{B64, Bytes};

const HOLOCENE_EXTRA_DATA_VERSION_BYTE: u8 = 0;
const JOVIAN_EXTRA_DATA_VERSION_BYTE: u8 = 1;

/// Encodes the `eip1559` parameters for the payload.
fn encode_eip_1559_params(
    eip_1559_params: B64,
    default_base_fee_params: BaseFeeParams,
    extra_data: &mut [u8],
) -> Result<(), EIP1559ParamError> {
    if extra_data.len() < 9 {
        return Err(EIP1559ParamError::InvalidExtraDataLength);
    }
    if eip_1559_params.is_zero() {
        let max_change_denominator: u32 = (default_base_fee_params.max_change_denominator)
            .try_into()
            .map_err(|_| EIP1559ParamError::DenominatorOverflow)?;

        let elasticity_multiplier: u32 = (default_base_fee_params.elasticity_multiplier)
            .try_into()
            .map_err(|_| EIP1559ParamError::ElasticityOverflow)?;

        extra_data[1..5].copy_from_slice(&max_change_denominator.to_be_bytes());
        extra_data[5..9].copy_from_slice(&elasticity_multiplier.to_be_bytes());
    } else {
        let (elasticity, denominator) = decode_eip_1559_params(eip_1559_params);
        extra_data[1..5].copy_from_slice(&denominator.to_be_bytes());
        extra_data[5..9].copy_from_slice(&elasticity.to_be_bytes());
    }
    Ok(())
}

/// Extracts the Holocene 1599 parameters from the encoded form:
/// <https://github.com/ethereum-optimism/specs/blob/main/specs/protocol/holocene/exec-engine.md#eip1559params-encoding>
///
/// Returns (`elasticity`, `denominator`)
pub fn decode_eip_1559_params(eip_1559_params: B64) -> (u32, u32) {
    let denominator: [u8; 4] = eip_1559_params.0[..4].try_into().expect("sufficient length");
    let elasticity: [u8; 4] = eip_1559_params.0[4..8].try_into().expect("sufficient length");

    (u32::from_be_bytes(elasticity), u32::from_be_bytes(denominator))
}

/// Decodes the `eip1559` parameters from the `extradata` bytes.
///
/// Returns (`elasticity`, `denominator`)
pub fn decode_holocene_extra_data(extra_data: &[u8]) -> Result<(u32, u32), EIP1559ParamError> {
    // Holocene extra data is always 9 _exactly_ bytes
    if extra_data.len() != 9 {
        return Err(EIP1559ParamError::InvalidExtraDataLength);
    }

    if extra_data[0] != HOLOCENE_EXTRA_DATA_VERSION_BYTE {
        // version must be 0: https://github.com/ethereum-optimism/specs/blob/main/specs/protocol/holocene/exec-engine.md#eip-1559-parameters-in-block-header
        return Err(EIP1559ParamError::InvalidVersion(extra_data[0]));
    }
    // skip the first version byte
    Ok(decode_eip_1559_params(B64::from_slice(&extra_data[1..9])))
}

/// Encodes the `eip1559` parameters for the payload.
pub fn encode_holocene_extra_data(
    eip_1559_params: B64,
    default_base_fee_params: BaseFeeParams,
) -> Result<Bytes, EIP1559ParamError> {
    // 9 bytes: 1 byte for version (0) and 8 bytes for eip1559 params
    let mut extra_data = [0u8; 9];
    encode_eip_1559_params(eip_1559_params, default_base_fee_params, &mut extra_data)?;
    Ok(Bytes::copy_from_slice(&extra_data))
}

/// Decodes the EIP-1559 parameters from `extra_data`,
/// as well as the minimum base fee.
///
/// Returns (`elasticity`, `denominator`, `min_base_fee`)
pub fn decode_jovian_extra_data(extra_data: &[u8]) -> Result<(u32, u32, u64), EIP1559ParamError> {
    if extra_data.len() != 17 {
        return Err(EIP1559ParamError::InvalidExtraDataLength);
    }

    if extra_data[0] != JOVIAN_EXTRA_DATA_VERSION_BYTE {
        // version must be 1: https://specs.optimism.io/protocol/jovian/exec-engine.html
        return Err(EIP1559ParamError::InvalidVersion(extra_data[0]));
    }
    // skip the first version byte
    let denominator: [u8; 4] = extra_data[1..5].try_into().expect("sufficient length");
    let elasticity: [u8; 4] = extra_data[5..9].try_into().expect("sufficient length");
    let min_base_fee: [u8; 8] = extra_data[9..17].try_into().expect("sufficient length");

    Ok((
        u32::from_be_bytes(elasticity),
        u32::from_be_bytes(denominator),
        u64::from_be_bytes(min_base_fee),
    ))
}

/// Encodes the EIP-1559 parameters for the payload,
/// as well as the minimum base fee.
pub fn encode_jovian_extra_data(
    eip_1559_params: B64,
    default_base_fee_params: BaseFeeParams,
    min_base_fee: u64,
) -> Result<Bytes, EIP1559ParamError> {
    // 17 bytes: 1 byte for version (1), 8 bytes for eip1559 params, and 8 byte for the minimum base
    // fee
    let mut extra_data = [0u8; 17];
    extra_data[0] = JOVIAN_EXTRA_DATA_VERSION_BYTE;
    encode_eip_1559_params(eip_1559_params, default_base_fee_params, &mut extra_data)?;
    extra_data[9..17].copy_from_slice(&min_base_fee.to_be_bytes());
    Ok(Bytes::copy_from_slice(&extra_data))
}

/// Error type for EIP-1559 parameters
#[derive(Debug, thiserror::Error, Clone, Copy, PartialEq, Eq)]
pub enum EIP1559ParamError {
    /// Thrown if the extra data begins with the wrong version byte.
    #[error("Invalid EIP1559 version byte: {0}")]
    InvalidVersion(u8),
    /// No EIP-1559 parameters provided.
    #[error("No EIP1559 parameters provided")]
    NoEIP1559Params,
    /// Denominator overflow.
    #[error("Denominator overflow")]
    DenominatorOverflow,
    /// Elasticity overflow.
    #[error("Elasticity overflow")]
    ElasticityOverflow,
    /// Extra data is not the correct length.
    #[error("Extra data is not the correct length")]
    InvalidExtraDataLength,
    /// Minimum base fee must be None before Jovian.
    #[error("Minimum base fee must be None before Jovian")]
    MinBaseFeeMustBeNone,
    /// Minimum base fee cannot be None after Jovian.
    #[error("Minimum base fee cannot be None after Jovian")]
    MinBaseFeeNotSet,
}

#[cfg(test)]
mod tests {
    use super::*;
    use core::str::FromStr;

    #[test]
    fn test_get_extra_data_post_holocene() {
        let eip_1559_params = B64::from_str("0x0000000800000008").unwrap();
        let extra_data = encode_holocene_extra_data(eip_1559_params, BaseFeeParams::new(80, 60));
        assert_eq!(extra_data.unwrap(), Bytes::copy_from_slice(&[0, 0, 0, 0, 8, 0, 0, 0, 8]));
    }

    #[test]
    fn test_get_extra_data_post_holocene_default() {
        let eip_1559_params = B64::ZERO;
        let extra_data = encode_holocene_extra_data(eip_1559_params, BaseFeeParams::new(80, 60));
        assert_eq!(extra_data.unwrap(), Bytes::copy_from_slice(&[0, 0, 0, 0, 80, 0, 0, 0, 60]));
    }

    #[test]
    fn test_encode_holocene_invalid_length() {
        let eip_1559_params = B64::ZERO;
        let extra_data = encode_holocene_extra_data(eip_1559_params, BaseFeeParams::new(80, 60));
        let res = decode_holocene_extra_data(&extra_data.unwrap()[..8]).unwrap_err();
        assert_eq!(res, EIP1559ParamError::InvalidExtraDataLength);
    }

    /// It shouldn't be possible to decode jovian extra data with holocene decoding
    #[test]
    fn test_decode_holocene_invalid_length() {
        let eip_1559_params = B64::ZERO;
        let extra_data = encode_jovian_extra_data(eip_1559_params, BaseFeeParams::new(80, 60), 0);
        let res = decode_holocene_extra_data(&extra_data.unwrap()).unwrap_err();

        assert_eq!(res, EIP1559ParamError::InvalidExtraDataLength);
    }

    #[test]
    fn test_get_extra_data_post_jovian() {
        let eip_1559_params = B64::from_str("0x0000000800000008").unwrap();
        let extra_data = encode_jovian_extra_data(eip_1559_params, BaseFeeParams::new(80, 60), 257);
        assert_eq!(
            extra_data.unwrap(),
            Bytes::copy_from_slice(&[1, 0, 0, 0, 8, 0, 0, 0, 8, 0, 0, 0, 0, 0, 0, 1, 1])
        );
    }

    #[test]
    fn test_get_extra_data_post_jovian_default() {
        let eip_1559_params = B64::ZERO;
        let extra_data = encode_jovian_extra_data(eip_1559_params, BaseFeeParams::new(80, 60), 0);
        // check the version byte is 1 and the min_base_fee is 0
        assert_eq!(
            extra_data.unwrap(),
            Bytes::copy_from_slice(&[1, 0, 0, 0, 80, 0, 0, 0, 60, 0, 0, 0, 0, 0, 0, 0, 0])
        );
    }

    #[test]
    fn test_encode_jovian_invalid_length() {
        let eip_1559_params = B64::ZERO;

        // Extra data is less than 9 bytes, which would error when encoding the eip1559 params
        let mut extra_data = [0u8; 8];
        let result =
            encode_eip_1559_params(eip_1559_params, BaseFeeParams::new(80, 60), &mut extra_data);
        assert_eq!(result.unwrap_err(), EIP1559ParamError::InvalidExtraDataLength);
    }

    #[test]
    fn test_decode_jovian_invalid_length() {
        let extra_data = [0u8; 8];
        let res = decode_jovian_extra_data(&extra_data);
        assert_eq!(res.unwrap_err(), EIP1559ParamError::InvalidExtraDataLength);
    }
}
