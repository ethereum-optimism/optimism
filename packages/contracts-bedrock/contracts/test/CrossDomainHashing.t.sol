//SPDX-License-Identifier: MIT
pragma solidity 0.8.10;

import { CommonTest } from "./CommonTest.t.sol";
import { CrossDomainHashing } from "../libraries/Lib_CrossDomainHashing.sol";

contract CrossDomainHashing_Test is CommonTest {
    function test_nonceVersioning(uint240 _nonce, uint16 _version) external {
        uint256 nonce = CrossDomainHashing.addVersionToNonce(uint256(_nonce), _version);
        uint16 version = CrossDomainHashing.getVersionFromNonce(nonce);
        assertEq(version, _version);
    }

    // TODO(tynes): turn this into differential fuzzing
    // it is very easy to do so with the typescript
    function test_l2TransactionHash() external {
        bytes32 l1BlockHash = 0xd1a498e053451fc90bd8a597051a1039010c8e55e2659b940d3070b326e4f4c5;
        uint256 logIndex = 0x0;
        address from =  address(0xDe3829A23DF1479438622a08a116E8Eb3f620BB5);
        address to = address(0xB7e390864a90b7b923C9f9310C6F98aafE43F707);
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
            0x77fc5994647d128a4d131d273a5e89e0306aac472494068a4f1fceab83dd0735
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
            hex"7ef862a077fc5994647d128a4d131d273a5e89e0306aac472494068a4f1fceab83dd073594de3829a23df1479438622a08a116e8eb3f620bb594b7e390864a90b7b923c9f9310c6f98aafe43f707880e043da617250000880de0b6b3a7640000832dc6c080"
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
            0xf5f97d03e8be48a4b20ed70c9d8b11f1c851bf949bf602b7580985705bb09077
        );
    }
}
