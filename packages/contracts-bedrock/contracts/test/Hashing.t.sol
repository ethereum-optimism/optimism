//SPDX-License-Identifier: MIT
pragma solidity 0.8.10;

import { CommonTest } from "./CommonTest.t.sol";
import { Hashing } from "../libraries/Hashing.sol";
import { Encoding } from "../libraries/Encoding.sol";

contract Hashing_Test is CommonTest {
    // TODO(tynes): turn this into differential fuzzing
    // it is very easy to do so with the typescript
    function test_hashDepositSource() external {
        bytes32 sourceHash = Hashing.hashDepositSource(
            0xd25df7858efc1778118fb133ac561b138845361626dfb976699c5287ed0f4959,
            0x1
        );

        assertEq(
            sourceHash,
            0xf923fb07134d7d287cb52c770cc619e17e82606c21a875c92f4c63b65280a5cc
        );
    }

    function test_hashDepositTransaction() external {
        bytes32 digest = Hashing.hashDepositTransaction(
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
            digest,
            0xf58e30138cb01330f6450b9a5e717a63840ad2e21f17340105b388ad3c668749
        );
    }
}
