//! The batcher update type.

use alloy_primitives::{Address, LogData};
use alloy_sol_types::{SolType, sol};

use crate::{
    BatcherUpdateError, SystemConfig, SystemConfigLog,
    updates::common::{ValidationError, validate_update_data},
};

/// The batcher update type.
#[derive(Debug, Clone, Hash, PartialEq, Eq)]
#[cfg_attr(feature = "serde", derive(serde::Serialize, serde::Deserialize))]
pub struct BatcherUpdate {
    /// The batcher address.
    pub batcher_address: Address,
}

impl BatcherUpdate {
    /// Applies the update to the [`SystemConfig`].
    pub const fn apply(&self, config: &mut SystemConfig) {
        config.batcher_address = self.batcher_address;
    }
}

impl TryFrom<&SystemConfigLog> for BatcherUpdate {
    type Error = BatcherUpdateError;

    fn try_from(log: &SystemConfigLog) -> Result<Self, Self::Error> {
        let LogData { data, .. } = &log.log.data;

        let validated = validate_update_data(data).map_err(|e| match e {
            ValidationError::InvalidDataLen(_expected, actual) => {
                BatcherUpdateError::InvalidDataLen(actual)
            }
            ValidationError::PointerDecodingError => BatcherUpdateError::PointerDecodingError,
            ValidationError::InvalidDataPointer(pointer) => {
                BatcherUpdateError::InvalidDataPointer(pointer)
            }
            ValidationError::LengthDecodingError => BatcherUpdateError::LengthDecodingError,
            ValidationError::InvalidDataLength(length) => {
                BatcherUpdateError::InvalidDataLength(length)
            }
        })?;

        let Ok(batcher_address) = <sol!(address)>::abi_decode_validate(validated.payload()) else {
            return Err(BatcherUpdateError::BatcherAddressDecodingError);
        };

        Ok(Self { batcher_address })
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::{CONFIG_UPDATE_EVENT_VERSION_0, CONFIG_UPDATE_TOPIC};
    use alloc::vec;
    use alloy_primitives::{B256, Bytes, Log, LogData, address, hex};

    #[test]
    fn test_batcher_update_try_from() {
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
        let update = BatcherUpdate::try_from(&system_log).unwrap();
        assert_eq!(update.batcher_address, address!("000000000000000000000000000000000000bEEF"),);
    }

    #[test]
    fn test_batcher_update_invalid_data_len() {
        let log =
            Log { address: Address::ZERO, data: LogData::new_unchecked(vec![], Bytes::default()) };
        let system_log = SystemConfigLog::new(log, false);
        let err = BatcherUpdate::try_from(&system_log).unwrap_err();
        assert_eq!(err, BatcherUpdateError::InvalidDataLen(0));
    }

    #[test]
    fn test_batcher_update_pointer_decoding_error() {
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
        let err = BatcherUpdate::try_from(&system_log).unwrap_err();
        assert_eq!(err, BatcherUpdateError::PointerDecodingError);
    }

    #[test]
    fn test_batcher_update_invalid_pointer_length() {
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
        let err = BatcherUpdate::try_from(&system_log).unwrap_err();
        assert_eq!(err, BatcherUpdateError::InvalidDataPointer(33));
    }

    #[test]
    fn test_batcher_update_length_decoding_error() {
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
        let err = BatcherUpdate::try_from(&system_log).unwrap_err();
        assert_eq!(err, BatcherUpdateError::LengthDecodingError);
    }

    #[test]
    fn test_batcher_update_invalid_data_length() {
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
        let err = BatcherUpdate::try_from(&system_log).unwrap_err();
        assert_eq!(err, BatcherUpdateError::InvalidDataLength(33));
    }

    #[test]
    fn test_batcher_update_batcher_decoding_error() {
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
        let err = BatcherUpdate::try_from(&system_log).unwrap_err();
        assert_eq!(err, BatcherUpdateError::BatcherAddressDecodingError);
    }
}
