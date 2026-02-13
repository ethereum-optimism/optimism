//! The unsafe block signer update.

use alloy_primitives::{Address, LogData};
use alloy_sol_types::{SolType, sol};

use crate::{
    SystemConfigLog, UnsafeBlockSignerUpdateError,
    updates::common::{ValidationError, validate_update_data},
};

/// The unsafe block signer update type.
#[derive(Debug, Clone, Hash, PartialEq, Eq)]
#[cfg_attr(feature = "serde", derive(serde::Serialize, serde::Deserialize))]
pub struct UnsafeBlockSignerUpdate {
    /// The new unsafe block signer address.
    pub unsafe_block_signer: Address,
}

impl TryFrom<&SystemConfigLog> for UnsafeBlockSignerUpdate {
    type Error = UnsafeBlockSignerUpdateError;

    fn try_from(log: &SystemConfigLog) -> Result<Self, Self::Error> {
        let LogData { data, .. } = &log.log.data;

        let validated = validate_update_data(data).map_err(|e| match e {
            ValidationError::InvalidDataLen(_expected, actual) => {
                UnsafeBlockSignerUpdateError::InvalidDataLen(actual)
            }
            ValidationError::PointerDecodingError => {
                UnsafeBlockSignerUpdateError::PointerDecodingError
            }
            ValidationError::InvalidDataPointer(pointer) => {
                UnsafeBlockSignerUpdateError::InvalidDataPointer(pointer)
            }
            ValidationError::LengthDecodingError => {
                UnsafeBlockSignerUpdateError::LengthDecodingError
            }
            ValidationError::InvalidDataLength(length) => {
                UnsafeBlockSignerUpdateError::InvalidDataLength(length)
            }
        })?;

        let Ok(unsafe_block_signer) = <sol!(address)>::abi_decode_validate(validated.payload())
        else {
            return Err(UnsafeBlockSignerUpdateError::UnsafeBlockSignerAddressDecodingError);
        };

        Ok(Self { unsafe_block_signer })
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::{CONFIG_UPDATE_EVENT_VERSION_0, CONFIG_UPDATE_TOPIC};
    use alloc::vec;
    use alloy_primitives::{B256, Bytes, Log, LogData, address, hex};

    #[test]
    fn test_signer_update_try_from() {
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
        let update = UnsafeBlockSignerUpdate::try_from(&system_log).unwrap();
        assert_eq!(
            update.unsafe_block_signer,
            address!("000000000000000000000000000000000000bEEF"),
        );
    }

    #[test]
    fn test_signer_update_invalid_data_len() {
        let log =
            Log { address: Address::ZERO, data: LogData::new_unchecked(vec![], Bytes::default()) };
        let system_log = SystemConfigLog::new(log, false);
        let err = UnsafeBlockSignerUpdate::try_from(&system_log).unwrap_err();
        assert_eq!(err, UnsafeBlockSignerUpdateError::InvalidDataLen(0));
    }

    #[test]
    fn test_signer_update_pointer_decoding_error() {
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
        let err = UnsafeBlockSignerUpdate::try_from(&system_log).unwrap_err();
        assert_eq!(err, UnsafeBlockSignerUpdateError::PointerDecodingError);
    }

    #[test]
    fn test_signer_update_invalid_pointer_length() {
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
        let err = UnsafeBlockSignerUpdate::try_from(&system_log).unwrap_err();
        assert_eq!(err, UnsafeBlockSignerUpdateError::InvalidDataPointer(33));
    }

    #[test]
    fn test_signer_update_length_decoding_error() {
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
        let err = UnsafeBlockSignerUpdate::try_from(&system_log).unwrap_err();
        assert_eq!(err, UnsafeBlockSignerUpdateError::LengthDecodingError);
    }

    #[test]
    fn test_signer_update_invalid_data_length() {
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
        let err = UnsafeBlockSignerUpdate::try_from(&system_log).unwrap_err();
        assert_eq!(err, UnsafeBlockSignerUpdateError::InvalidDataLength(33));
    }

    #[test]
    fn test_signer_update_address_decoding_error() {
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
        let err = UnsafeBlockSignerUpdate::try_from(&system_log).unwrap_err();
        assert_eq!(err, UnsafeBlockSignerUpdateError::UnsafeBlockSignerAddressDecodingError);
    }
}
