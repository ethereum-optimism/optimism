// SPDX-License-Identifier: MIT
pragma solidity ^0.8.9;

/**
 * @title Lib_Strings
 * @dev This library implements a function to convert an address to an ASCII string.
 * It uses the implementation written by tkeber:
 * https://ethereum.stackexchange.com/questions/8346/convert-address-to-string/8447#8447
 */
library Lib_Strings {
    /**********************
     * Internal Functions *
     **********************/

    /**
     * Converts an address to its ASCII string representation. The returned string will be
     * lowercase and the 0x prefix will be removed.
     * @param _address Address to convert to an ASCII string.
     * @return String representation of the address.
     */
    function addressToString(address _address) internal pure returns (string memory) {
        bytes memory s = new bytes(40);
        for (uint256 i = 0; i < 20; i++) {
            bytes1 b = bytes1(uint8(uint256(uint160(_address)) / (2**(8 * (19 - i)))));
            bytes1 hi = bytes1(uint8(b) / 16);
            bytes1 lo = bytes1(uint8(b) - 16 * uint8(hi));
            s[2 * i] = hexCharToAscii(hi);
            s[2 * i + 1] = hexCharToAscii(lo);
        }
        return string(s);
    }

    /**
     * Converts a hexadecimal character into its ASCII representation.
     * @param _byte A single hexadecimal character
     * @return ASCII representation of the hexadecimal character.
     */
    function hexCharToAscii(bytes1 _byte) internal pure returns (bytes1) {
        if (uint8(_byte) < 10) return bytes1(uint8(_byte) + 0x30);
        else return bytes1(uint8(_byte) + 0x57);
    }
}
