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

    function test_encodeDepositTransaction() external {
        bytes memory raw = Encoding.encodeDepositTransaction(
            0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266,
            0xB79f76EF2c5F0286176833E7B2eEe103b1CC3244,
            0xde0b6b3a7640000,
            0xe043da617250000,
            0x2dc6c0,
            false,
            hex"",
            0xd25df7858efc1778118fb133ac561b138845361626dfb976699c5287ed0f4959,
            0x1
        );

        assertEq(
            raw,
            hex"7e00f862a0f923fb07134d7d287cb52c770cc619e17e82606c21a875c92f4c63b65280a5cc94f39fd6e51aad88f6f4ce6ab8827279cfffb9226694b79f76ef2c5f0286176833e7b2eee103b1cc3244880e043da617250000880de0b6b3a7640000832dc6c080"
        );
    }
}
