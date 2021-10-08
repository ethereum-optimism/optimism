// SPDX-License-Identifier: MIT
pragma solidity ^0.8.9;

/* Library Imports */
import { Lib_RLPWriter } from "../../libraries/rlp/Lib_RLPWriter.sol";
import { TestERC20 } from "../../test-helpers/TestERC20.sol";

/**
 * @title TestLib_RLPWriter
 */
contract TestLib_RLPWriter {
    function writeBytes(bytes memory _in) public pure returns (bytes memory _out) {
        return Lib_RLPWriter.writeBytes(_in);
    }

    function writeList(bytes[] memory _in) public pure returns (bytes memory _out) {
        return Lib_RLPWriter.writeList(_in);
    }

    function writeString(string memory _in) public pure returns (bytes memory _out) {
        return Lib_RLPWriter.writeString(_in);
    }

    function writeAddress(address _in) public pure returns (bytes memory _out) {
        return Lib_RLPWriter.writeAddress(_in);
    }

    function writeUint(uint256 _in) public pure returns (bytes memory _out) {
        return Lib_RLPWriter.writeUint(_in);
    }

    function writeBool(bool _in) public pure returns (bytes memory _out) {
        return Lib_RLPWriter.writeBool(_in);
    }

    function writeAddressWithTaintedMemory(address _in) public returns (bytes memory _out) {
        new TestERC20();
        return Lib_RLPWriter.writeAddress(_in);
    }
}
