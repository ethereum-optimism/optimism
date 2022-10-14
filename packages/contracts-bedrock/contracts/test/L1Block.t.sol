// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { CommonTest } from "./CommonTest.t.sol";
import { L1Block } from "../L2/L1Block.sol";

contract L1Block_TestInit is CommonTest {
    L1Block lb;
    address depositor;
    bytes32 immutable NON_ZERO_HASH = keccak256(abi.encode(1));

    function setUp() external {
        lb = new L1Block();
        depositor = lb.DEPOSITOR_ACCOUNT();
        vm.prank(depositor);
        lb.setL1BlockValues(uint64(1), uint64(2), 3, NON_ZERO_HASH, uint64(4));
    }
}

contract L1Block_Getters_TestFail is L1Block_TestInit {
    // none
}

contract L1Block_Getters_Test is L1Block_TestInit {
    function test_number() external {
        assertEq(lb.number(), uint64(1));
    }

    function test_timestamp() external {
        assertEq(lb.timestamp(), uint64(2));
    }

    function test_basefee() external {
        assertEq(lb.basefee(), 3);
    }

    function test_hash() external {
        assertEq(lb.hash(), NON_ZERO_HASH);
    }

    function test_sequenceNumber() external {
        assertEq(lb.sequenceNumber(), uint64(4));
    }
}

contract L1Block_Setters_TestFail is L1Block_TestInit {
    // none
}

contract L1Block_Setters_Test is L1Block_TestInit {
    function test_setL1BlockValues_succeeds() external {
        vm.prank(depositor);
        lb.setL1BlockValues(
            type(uint64).max,
            type(uint64).max,
            type(uint256).max,
            keccak256(abi.encode(1)),
            type(uint64).max
        );
    }

    function testFuzz_setL1BlockValues_succeeds(uint64 n, uint64 t, uint256 b, bytes32 h, uint64 s) external {
        vm.prank(depositor);
        lb.setL1BlockValues(n, t, b, h, s);
        assertEq(lb.number(), n);
        assertEq(lb.timestamp(), t);
        assertEq(lb.basefee(), b);
        assertEq(lb.hash(), h);
        assertEq(lb.sequenceNumber(), s);
    }
}
