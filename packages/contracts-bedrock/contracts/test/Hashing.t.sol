//SPDX-License-Identifier: MIT
pragma solidity 0.8.10;

import { CommonTest } from "./CommonTest.t.sol";
import { Hashing } from "../libraries/Hashing.sol";
import { Encoding } from "../libraries/Encoding.sol";

contract Hashing_Test is CommonTest {
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
}
