//! The EIP-1559 update type.

use alloy_primitives::LogData;
use alloy_sol_types::{SolType, sol};

use crate::{
    EIP1559UpdateError, SystemConfig, SystemConfigLog,
    updates::common::{ValidationError, validate_update_data},
};

/// The EIP-1559 update type.
#[derive(Debug, Default, Clone, Hash, PartialEq, Eq)]
#[cfg_attr(feature = "serde", derive(serde::Serialize, serde::Deserialize))]
pub struct Eip1559Update {
    /// The EIP-1559 denominator.
    pub eip1559_denominator: u32,
    /// The EIP-1559 elasticity multiplier.
    pub eip1559_elasticity: u32,
}

impl Eip1559Update {
    /// Applies the update to the [`SystemConfig`].
    pub const fn apply(&self, config: &mut SystemConfig) {
        config.eip1559_denominator = Some(self.eip1559_denominator);
        config.eip1559_elasticity = Some(self.eip1559_elasticity);
    }
}

impl TryFrom<&SystemConfigLog> for Eip1559Update {
    type Error = EIP1559UpdateError;

    fn try_from(log: &SystemConfigLog) -> Result<Self, Self::Error> {
        let LogData { data, .. } = &log.log.data;

        let validated = validate_update_data(data).map_err(|e| match e {
            ValidationError::InvalidDataLen(_expected, actual) => {
                EIP1559UpdateError::InvalidDataLen(actual)
            }
            ValidationError::PointerDecodingError => EIP1559UpdateError::PointerDecodingError,
            ValidationError::InvalidDataPointer(pointer) => {
                EIP1559UpdateError::InvalidDataPointer(pointer)
            }
            ValidationError::LengthDecodingError => EIP1559UpdateError::LengthDecodingError,
            ValidationError::InvalidDataLength(length) => {
                EIP1559UpdateError::InvalidDataLength(length)
            }
        })?;

        let Ok(eip1559_params) = <sol!(uint64)>::abi_decode_validate(validated.payload()) else {
            return Err(EIP1559UpdateError::EIP1559DecodingError);
        };

        Ok(Self {
            eip1559_denominator: (eip1559_params >> 32) as u32,
            eip1559_elasticity: eip1559_params as u32,
        })
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::{CONFIG_UPDATE_EVENT_VERSION_0, CONFIG_UPDATE_TOPIC};
    use alloc::vec;
    use alloy_primitives::{Address, B256, Bytes, Log, LogData, hex};

    #[test]
    fn test_eip1559_update_try_from() {
        let update_type = B256::ZERO;

        let log = Log {
            address: Address::ZERO,
            data: LogData::new_unchecked(
                vec![
                    CONFIG_UPDATE_TOPIC,
                    CONFIG_UPDATE_EVENT_VERSION_0,
                    update_type,
                ],
                hex!("000000000000000000000000000000000000000000000000000000000000002000000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000babe0000beef").into()
            )
        };

        let system_log = SystemConfigLog::new(log, false);
        let update = Eip1559Update::try_from(&system_log).unwrap();

        assert_eq!(update.eip1559_denominator, 0xbabe_u32);
        assert_eq!(update.eip1559_elasticity, 0xbeef_u32);
    }

    #[test]
    fn test_eip1559_update_invalid_data_len() {
        let log =
            Log { address: Address::ZERO, data: LogData::new_unchecked(vec![], Bytes::default()) };
        let system_log = SystemConfigLog::new(log, false);
        let err = Eip1559Update::try_from(&system_log).unwrap_err();
        assert_eq!(err, EIP1559UpdateError::InvalidDataLen(0));
    }

    #[test]
    fn test_eip1559_update_pointer_decoding_error() {
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
        let err = Eip1559Update::try_from(&system_log).unwrap_err();
        assert_eq!(err, EIP1559UpdateError::PointerDecodingError);
    }

    #[test]
    fn test_eip1559_update_invalid_pointer_length() {
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
        let err = Eip1559Update::try_from(&system_log).unwrap_err();
        assert_eq!(err, EIP1559UpdateError::InvalidDataPointer(33));
    }

    #[test]
    fn test_eip1559_update_length_decoding_error() {
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
        let err = Eip1559Update::try_from(&system_log).unwrap_err();
        assert_eq!(err, EIP1559UpdateError::LengthDecodingError);
    }

    #[test]
    fn test_eip1559_update_invalid_data_length() {
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
        let err = Eip1559Update::try_from(&system_log).unwrap_err();
        assert_eq!(err, EIP1559UpdateError::InvalidDataLength(33));
    }

    #[test]
    fn test_eip1559_update_eip1559_decoding_error() {
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
        let err = Eip1559Update::try_from(&system_log).unwrap_err();
        assert_eq!(err, EIP1559UpdateError::EIP1559DecodingError);
    }
}
