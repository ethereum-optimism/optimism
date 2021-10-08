// SPDX-License-Identifier: MIT
pragma solidity ^0.8.9;

/* Library Imports */
import { Lib_RLPReader } from "../../libraries/rlp/Lib_RLPReader.sol";

/**
 * @title TestLib_RLPReader
 */
contract TestLib_RLPReader {
    function readList(bytes memory _in) public pure returns (bytes[] memory) {
        Lib_RLPReader.RLPItem[] memory decoded = Lib_RLPReader.readList(_in);
        bytes[] memory out = new bytes[](decoded.length);
        for (uint256 i = 0; i < out.length; i++) {
            out[i] = Lib_RLPReader.readRawBytes(decoded[i]);
        }
        return out;
    }

    function readString(bytes memory _in) public pure returns (string memory) {
        return Lib_RLPReader.readString(_in);
    }

    function readBytes(bytes memory _in) public pure returns (bytes memory) {
        return Lib_RLPReader.readBytes(_in);
    }

    function readBytes32(bytes memory _in) public pure returns (bytes32) {
        return Lib_RLPReader.readBytes32(_in);
    }

    function readUint256(bytes memory _in) public pure returns (uint256) {
        return Lib_RLPReader.readUint256(_in);
    }

    function readBool(bytes memory _in) public pure returns (bool) {
        return Lib_RLPReader.readBool(_in);
    }

    function readAddress(bytes memory _in) public pure returns (address) {
        return Lib_RLPReader.readAddress(_in);
    }
}
