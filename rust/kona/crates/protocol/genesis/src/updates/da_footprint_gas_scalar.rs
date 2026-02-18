//! The da footprint gas scalar update type.

use alloy_primitives::LogData;
use alloy_sol_types::{SolType, sol};

use crate::{SystemConfig, SystemConfigLog, system::DaFootprintGasScalarUpdateError};

/// The da footprint gas scalar update type.
#[derive(Debug, Default, Clone, Hash, PartialEq, Eq)]
#[cfg_attr(feature = "serde", derive(serde::Serialize, serde::Deserialize))]
pub struct DaFootprintGasScalarUpdate {
    /// The da footprint gas scalar.
    pub da_footprint_gas_scalar: u16,
}

impl DaFootprintGasScalarUpdate {
    /// The default DA footprint gas scalar
    /// <https://github.com/ethereum-optimism/specs/blob/664cba65ab9686b0e70ad19fdf2ad054d6295986/specs/protocol/jovian/l1-attributes.md#overview>
    pub const DEFAULT_DA_FOOTPRINT_GAS_SCALAR: u16 = 400;

    /// Applies the update to the [`SystemConfig`].
    pub const fn apply(&self, config: &mut SystemConfig) {
        let mut da_footprint_gas_scalar = self.da_footprint_gas_scalar;

        // If the da footprint gas scalar is 0, use the default value
        // <https://github.com/ethereum-optimism/specs/blob/664cba65ab9686b0e70ad19fdf2ad054d6295986/specs/protocol/jovian/l1-attributes.md#overview>
        if da_footprint_gas_scalar == 0 {
            da_footprint_gas_scalar = Self::DEFAULT_DA_FOOTPRINT_GAS_SCALAR;
        };

        config.da_footprint_gas_scalar = Some(da_footprint_gas_scalar);
    }
}

impl TryFrom<&SystemConfigLog> for DaFootprintGasScalarUpdate {
    type Error = DaFootprintGasScalarUpdateError;

    fn try_from(log: &SystemConfigLog) -> Result<Self, Self::Error> {
        let LogData { data, .. } = &log.log.data;
        if data.len() != 96 {
            return Err(DaFootprintGasScalarUpdateError::InvalidDataLen(data.len()));
        }

        let Ok(pointer) = <sol!(uint64)>::abi_decode_validate(&data[0..32]) else {
            return Err(DaFootprintGasScalarUpdateError::PointerDecodingError);
        };
        if pointer != 32 {
            return Err(DaFootprintGasScalarUpdateError::InvalidDataPointer(pointer));
        }

        let Ok(length) = <sol!(uint64)>::abi_decode_validate(&data[32..64]) else {
            return Err(DaFootprintGasScalarUpdateError::LengthDecodingError);
        };
        if length != 32 {
            return Err(DaFootprintGasScalarUpdateError::InvalidDataLength(length));
        }

        let Ok(da_footprint_gas_scalar) = <sol!(uint16)>::abi_decode_validate(&data[64..96]) else {
            return Err(DaFootprintGasScalarUpdateError::DaFootprintGasScalarDecodingError);
        };

        Ok(Self { da_footprint_gas_scalar })
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::{CONFIG_UPDATE_EVENT_VERSION_0, CONFIG_UPDATE_TOPIC};
    use alloc::vec;
    use alloy_primitives::{Address, B256, Bytes, Log, LogData, hex};

    #[test]
    fn test_da_footprint_update_try_from() {
        let update_type = B256::ZERO;

        let log = Log {
            address: Address::ZERO,
            data: LogData::new_unchecked(
                vec![
                    CONFIG_UPDATE_TOPIC,
                    CONFIG_UPDATE_EVENT_VERSION_0,
                    update_type,
                ],
                hex!("00000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000beef").into()
            )
        };

        let system_log = SystemConfigLog::new(log, false);
        let update = DaFootprintGasScalarUpdate::try_from(&system_log).unwrap();

        assert_eq!(update.da_footprint_gas_scalar, 0xbeef_u16);
    }

    #[test]
    fn test_da_footprint_update_invalid_data_len() {
        let log =
            Log { address: Address::ZERO, data: LogData::new_unchecked(vec![], Bytes::default()) };
        let system_log = SystemConfigLog::new(log, false);
        let err = DaFootprintGasScalarUpdate::try_from(&system_log).unwrap_err();
        assert_eq!(err, DaFootprintGasScalarUpdateError::InvalidDataLen(0));
    }

    #[test]
    fn test_da_footprint_update_pointer_decoding_error() {
        let log = Log {
            address: Address::ZERO,
            data: LogData::new_unchecked(
                vec![
                    CONFIG_UPDATE_TOPIC,
                    CONFIG_UPDATE_EVENT_VERSION_0,
                    B256::ZERO,
                ],
                hex!("FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF00000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000babe0000beef").into()
            )
        };

        let system_log = SystemConfigLog::new(log, false);
        let err = DaFootprintGasScalarUpdate::try_from(&system_log).unwrap_err();
        assert_eq!(err, DaFootprintGasScalarUpdateError::PointerDecodingError);
    }

    #[test]
    fn test_da_footprint_update_invalid_pointer_length() {
        let log = Log {
            address: Address::ZERO,
            data: LogData::new_unchecked(
                vec![
                    CONFIG_UPDATE_TOPIC,
                    CONFIG_UPDATE_EVENT_VERSION_0,
                    B256::ZERO,
                ],
                hex!("000000000000000000000000000000000000000000000000000000000000002100000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000babe0000beef").into()
            )
        };

        let system_log = SystemConfigLog::new(log, false);
        let err = DaFootprintGasScalarUpdate::try_from(&system_log).unwrap_err();
        assert_eq!(err, DaFootprintGasScalarUpdateError::InvalidDataPointer(33));
    }

    #[test]
    fn test_da_footprint_update_length_decoding_error() {
        let log = Log {
            address: Address::ZERO,
            data: LogData::new_unchecked(
                vec![
                    CONFIG_UPDATE_TOPIC,
                    CONFIG_UPDATE_EVENT_VERSION_0,
                    B256::ZERO,
                ],
                hex!("0000000000000000000000000000000000000000000000000000000000000020FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF0000000000000000000000000000000000000000000000000000babe0000beef").into()
            )
        };

        let system_log = SystemConfigLog::new(log, false);
        let err = DaFootprintGasScalarUpdate::try_from(&system_log).unwrap_err();
        assert_eq!(err, DaFootprintGasScalarUpdateError::LengthDecodingError);
    }

    #[test]
    fn test_da_footprint_update_invalid_data_length() {
        let log = Log {
            address: Address::ZERO,
            data: LogData::new_unchecked(
                vec![
                    CONFIG_UPDATE_TOPIC,
                    CONFIG_UPDATE_EVENT_VERSION_0,
                    B256::ZERO,
                ],
                hex!("000000000000000000000000000000000000000000000000000000000000002000000000000000000000000000000000000000000000000000000000000000210000000000000000000000000000000000000000000000000000babe0000beef").into()
            )
        };

        let system_log = SystemConfigLog::new(log, false);
        let err = DaFootprintGasScalarUpdate::try_from(&system_log).unwrap_err();
        assert_eq!(err, DaFootprintGasScalarUpdateError::InvalidDataLength(33));
    }

    #[test]
    fn test_da_footprint_update_da_footprint_decoding_error() {
        let log = Log {
            address: Address::ZERO,
            data: LogData::new_unchecked(
                vec![
                    CONFIG_UPDATE_TOPIC,
                    CONFIG_UPDATE_EVENT_VERSION_0,
                    B256::ZERO,
                ],
                hex!("00000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000020FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF").into()
            )
        };

        let system_log = SystemConfigLog::new(log, false);
        let err = DaFootprintGasScalarUpdate::try_from(&system_log).unwrap_err();
        assert_eq!(err, DaFootprintGasScalarUpdateError::DaFootprintGasScalarDecodingError);
    }
}
