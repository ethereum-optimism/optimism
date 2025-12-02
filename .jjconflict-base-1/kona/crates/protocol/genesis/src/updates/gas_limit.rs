//! The gas limit update type.

use alloy_primitives::{LogData, U64, U256};
use alloy_sol_types::{SolType, sol};

use crate::{GasLimitUpdateError, SystemConfig, SystemConfigLog};

/// The gas limit update type.
#[derive(Debug, Default, Clone, Hash, PartialEq, Eq)]
#[cfg_attr(feature = "serde", derive(serde::Serialize, serde::Deserialize))]
pub struct GasLimitUpdate {
    /// The gas limit.
    pub gas_limit: u64,
}

impl GasLimitUpdate {
    /// Applies the update to the [`SystemConfig`].
    pub const fn apply(&self, config: &mut SystemConfig) {
        config.gas_limit = self.gas_limit;
    }
}

impl TryFrom<&SystemConfigLog> for GasLimitUpdate {
    type Error = GasLimitUpdateError;

    fn try_from(log: &SystemConfigLog) -> Result<Self, Self::Error> {
        let LogData { data, .. } = &log.log.data;
        if data.len() != 96 {
            return Err(GasLimitUpdateError::InvalidDataLen(data.len()));
        }

        let Ok(pointer) = <sol!(uint64)>::abi_decode_validate(&data[0..32]) else {
            return Err(GasLimitUpdateError::PointerDecodingError);
        };
        if pointer != 32 {
            return Err(GasLimitUpdateError::InvalidDataPointer(pointer));
        }

        let Ok(length) = <sol!(uint64)>::abi_decode_validate(&data[32..64]) else {
            return Err(GasLimitUpdateError::LengthDecodingError);
        };
        if length != 32 {
            return Err(GasLimitUpdateError::InvalidDataLength(length));
        }

        let Ok(gas_limit) = <sol!(uint256)>::abi_decode_validate(&data[64..]) else {
            return Err(GasLimitUpdateError::GasLimitDecodingError);
        };

        // Prevent overflows here.
        let max = U256::from(u64::MAX as u128);
        if gas_limit > max {
            return Err(GasLimitUpdateError::GasLimitDecodingError);
        }

        Ok(Self { gas_limit: U64::from(gas_limit).saturating_to::<u64>() })
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::{CONFIG_UPDATE_EVENT_VERSION_0, CONFIG_UPDATE_TOPIC};
    use alloc::vec;
    use alloy_primitives::{Address, B256, Bytes, Log, LogData, hex};

    #[test]
    fn test_gas_limit_update_try_from() {
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
        let update = GasLimitUpdate::try_from(&system_log).unwrap();

        assert_eq!(update.gas_limit, 0xbeef_u64);
    }

    #[test]
    fn test_gas_limit_update_invalid_data_len() {
        let log =
            Log { address: Address::ZERO, data: LogData::new_unchecked(vec![], Bytes::default()) };
        let system_log = SystemConfigLog::new(log, false);
        let err = GasLimitUpdate::try_from(&system_log).unwrap_err();
        assert_eq!(err, GasLimitUpdateError::InvalidDataLen(0));
    }

    #[test]
    fn test_gas_limit_update_pointer_decoding_error() {
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
        let err = GasLimitUpdate::try_from(&system_log).unwrap_err();
        assert_eq!(err, GasLimitUpdateError::PointerDecodingError);
    }

    #[test]
    fn test_gas_limit_update_invalid_pointer_length() {
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
        let err = GasLimitUpdate::try_from(&system_log).unwrap_err();
        assert_eq!(err, GasLimitUpdateError::InvalidDataPointer(33));
    }

    #[test]
    fn test_gas_limit_update_length_decoding_error() {
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
        let err = GasLimitUpdate::try_from(&system_log).unwrap_err();
        assert_eq!(err, GasLimitUpdateError::LengthDecodingError);
    }

    #[test]
    fn test_gas_limit_update_invalid_data_length() {
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
        let err = GasLimitUpdate::try_from(&system_log).unwrap_err();
        assert_eq!(err, GasLimitUpdateError::InvalidDataLength(33));
    }

    #[test]
    fn test_gas_limit_update_gas_limit_decoding_error() {
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
        let err = GasLimitUpdate::try_from(&system_log).unwrap_err();
        assert_eq!(err, GasLimitUpdateError::GasLimitDecodingError);
    }
}
