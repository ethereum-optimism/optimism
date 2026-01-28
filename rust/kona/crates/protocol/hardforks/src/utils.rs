//! Utilities for creating hardforks.

use alloy_primitives::{Address, Bytes, hex};

/// UpgradeTo Function 4Byte Signature
pub(crate) const UPGRADE_TO_FUNC_BYTES_4: [u8; 4] = hex!("3659cfe6");

/// Turns the given address into calldata for the `upgradeTo` function.
pub(crate) fn upgrade_to_calldata(addr: Address) -> Bytes {
    let mut v = UPGRADE_TO_FUNC_BYTES_4.to_vec();
    v.extend_from_slice(addr.into_word().as_slice());
    v.into()
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::{Ecotone, Fjord, Isthmus};
    use alloy_primitives::keccak256;

    #[test]
    fn test_upgrade_to_selector_is_valid() {
        let expected_selector = &keccak256("upgradeTo(address)")[..4];
        assert_eq!(UPGRADE_TO_FUNC_BYTES_4, expected_selector);
    }

    #[test]
    fn test_upgrade_to_calldata_format() {
        let test_addr = Address::from([0x42; 20]);
        let calldata = upgrade_to_calldata(test_addr);

        assert_eq!(calldata.len(), 36);
        assert_eq!(&calldata[..4], UPGRADE_TO_FUNC_BYTES_4);
        assert_eq!(&calldata[4..36], test_addr.into_word().as_slice());
    }

    #[test]
    fn test_ecotone_selector_is_valid() {
        let expected_selector = &keccak256("setEcotone()")[..4];
        assert_eq!(Ecotone::ENABLE_ECOTONE_INPUT, expected_selector);
    }

    #[test]
    fn test_fjord_selector_is_valid() {
        let expected_selector = &keccak256("setFjord()")[..4];
        assert_eq!(Fjord::SET_FJORD_METHOD_SIGNATURE, expected_selector);
    }

    #[test]
    fn test_isthmus_selector_is_valid() {
        let expected_selector = &keccak256("setIsthmus()")[..4];
        assert_eq!(Isthmus::ENABLE_ISTHMUS_INPUT, expected_selector);
    }
}
