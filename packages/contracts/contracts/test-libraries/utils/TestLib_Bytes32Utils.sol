// SPDX-License-Identifier: MIT
pragma solidity ^0.8.9;

/* Library Imports */
import { Lib_Bytes32Utils } from "../../libraries/utils/Lib_Bytes32Utils.sol";

/**
 * @title TestLib_Byte32Utils
 */
contract TestLib_Bytes32Utils {
    function toBool(bytes32 _in) public pure returns (bool _out) {
        return Lib_Bytes32Utils.toBool(_in);
    }

    function fromBool(bool _in) public pure returns (bytes32 _out) {
        return Lib_Bytes32Utils.fromBool(_in);
    }

    function toAddress(bytes32 _in) public pure returns (address _out) {
        return Lib_Bytes32Utils.toAddress(_in);
    }

    function fromAddress(address _in) public pure returns (bytes32 _out) {
        return Lib_Bytes32Utils.fromAddress(_in);
    }
}
