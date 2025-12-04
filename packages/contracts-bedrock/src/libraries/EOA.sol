// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

/// @title EOA
/// @notice A library for detecting if an address is an EOA.
library EOA {
    /// @notice Returns true if sender address is an EOA.
    /// @return isEOA_ True if the sender address is an EOA.
    function isSenderEOA() internal view returns (bool isEOA_) {
        if (msg.sender == tx.origin) {
            isEOA_ = true;
        } else if (address(msg.sender).code.length == 23) {
            // If the sender is not the origin, check for 7702 delegated EOAs.
            assembly {
                let ptr := mload(0x40)
                mstore(0x40, add(ptr, 0x20))
                extcodecopy(caller(), ptr, 0, 0x20)
                isEOA_ := eq(shr(232, mload(ptr)), 0xEF0100)
            }
        } else {
            // If more or less than 23 bytes of code, not a 7702 delegated EOA.
            isEOA_ = false;
        }
    }
}
