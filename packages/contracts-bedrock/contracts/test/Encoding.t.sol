//SPDX-License-Identifier: MIT
pragma solidity 0.8.10;

import { CommonTest } from "./CommonTest.t.sol";
import { Encoding } from "../libraries/Encoding.sol";

contract Encoding_Test is CommonTest {
    function test_nonceVersioning(uint240 _nonce, uint16 _version) external {
        (uint240 nonce, uint16 version) = Encoding.decodeVersionedNonce(
            Encoding.encodeVersionedNonce(_nonce, _version)
        );
        assertEq(version, _version);
        assertEq(nonce, _nonce);
    }
}
