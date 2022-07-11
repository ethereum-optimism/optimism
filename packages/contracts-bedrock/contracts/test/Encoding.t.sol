//SPDX-License-Identifier: MIT
pragma solidity 0.8.10;

import { CommonTest } from "./CommonTest.t.sol";
import { Encoding } from "../libraries/Encoding.sol";

contract Encoding_Test is CommonTest {
    function test_nonceVersioning(uint240 _nonce, uint16 _version) external {
        uint256 nonce = Encoding.addVersionToNonce(uint256(_nonce), _version);
        uint16 version = Encoding.getVersionFromNonce(nonce);
        assertEq(version, _version);
    }
}
