// SPDX-License-Identifier: MIT
pragma solidity ^0.8.9;

/* Library Imports */
import { Lib_Strings } from "../../libraries/utils/Lib_Strings.sol";

/**
 * @title TestLib_Strings
 */
contract TestLib_Strings {
    function addressToString(address _address) public pure returns (string memory) {
        return Lib_Strings.addressToString(_address);
    }

    function hexCharToAscii(bytes1 _byte) public pure returns (bytes1) {
        return Lib_Strings.hexCharToAscii(_byte);
    }
}
