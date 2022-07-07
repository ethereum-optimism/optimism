//SPDX-License-Identifier: MIT
pragma solidity 0.8.10;

import { CommonTest } from "./CommonTest.t.sol";
import { CrossDomainHashing } from "../libraries/CrossDomainHashing.sol";

contract CrossDomainHashing_Test is CommonTest {
    function test_nonceVersioning(uint240 _nonce, uint16 _version) external {
        uint256 nonce = CrossDomainHashing.addVersionToNonce(uint256(_nonce), _version);
        uint16 version = CrossDomainHashing.getVersionFromNonce(nonce);
        assertEq(version, _version);
    }

    // TODO(tynes): turn this into differential fuzzing
    // it is very easy to do so with the typescript
    function test_l2TransactionHash() external {
        bytes32 l1BlockHash = 0xd25df7858efc1778118fb133ac561b138845361626dfb976699c5287ed0f4959;
        uint256 logIndex = 0x1;
        address from = 0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266;
        address to = 0xB79f76EF2c5F0286176833E7B2eEe103b1CC3244;
        bool isCreate = false;
        uint256 mint = 0xe043da617250000;
        uint256 value = 0xde0b6b3a7640000;
        uint256 gas = 0x2dc6c0;
        bytes memory data = hex"";

        bytes32 sourceHash = CrossDomainHashing.sourceHash(
            l1BlockHash,
            logIndex
        );

        assertEq(
            sourceHash,
            0xf923fb07134d7d287cb52c770cc619e17e82606c21a875c92f4c63b65280a5cc
        );

        bytes memory raw = CrossDomainHashing.L2Transaction(
            l1BlockHash,
            logIndex,
            from,
            to,
            isCreate,
            mint,
            value,
            gas,
            data
        );

        assertEq(
            raw,
            hex"7e00f862a0f923fb07134d7d287cb52c770cc619e17e82606c21a875c92f4c63b65280a5cc94f39fd6e51aad88f6f4ce6ab8827279cfffb9226694b79f76ef2c5f0286176833e7b2eee103b1cc3244880e043da617250000880de0b6b3a7640000832dc6c080"
        );

        bytes32 digest = CrossDomainHashing.L2TransactionHash(
           l1BlockHash,
           logIndex,
           from,
           to,
           isCreate,
           mint,
           value,
           gas,
           data
        );

        assertEq(
            digest,
            0xf58e30138cb01330f6450b9a5e717a63840ad2e21f17340105b388ad3c668749
        );
    }
}
