// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Test } from "forge-std/Test.sol";
import { L1Block } from "../L2/L1Block.sol";
import { L1BlockNumber } from "../legacy/L1BlockNumber.sol";
import { Predeploys } from "../libraries/Predeploys.sol";

contract L1BlockNumber_TestInit is Test {
    L1Block lb;
    L1BlockNumber bn;

    function setUp() external {
        vm.etch(Predeploys.L1_BLOCK_ATTRIBUTES, address(new L1Block()).code);
        lb = L1Block(Predeploys.L1_BLOCK_ATTRIBUTES);
        bn = new L1BlockNumber();
        vm.prank(lb.DEPOSITOR_ACCOUNT());
        lb.setL1BlockValues(uint64(999), uint64(2), 3, keccak256(abi.encode(1)), uint64(4));
    }
}

contract L1BlockNumber_Getters_Test is L1BlockNumber_TestInit {
    function test_getL1BlockNumber() external {
        assertEq(bn.getL1BlockNumber(), 999);
    }
}

contract L1BlockNumber_Fallback_TestFail is L1BlockNumber_TestInit {
    // none
}

contract L1BlockNumber_Fallback_Test is L1BlockNumber_TestInit {
    function test_fallback() external {
        (bool success, bytes memory ret) = address(bn).call(hex"");
        assertEq(success, true);
        assertEq(ret, abi.encode(999));
    }
}

contract L1BlockNumber_Receive_TestFail is L1BlockNumber_TestInit {
    // none
}

contract L1BlockNumber_Receive_Test is L1BlockNumber_TestInit {
    function test_receive() external {
        (bool success, bytes memory ret) = address(bn).call{ value: 1 }(hex"");
        assertEq(success, true);
        assertEq(ret, abi.encode(999));
    }
}
