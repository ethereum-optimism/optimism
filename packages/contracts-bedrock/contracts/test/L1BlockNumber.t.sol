// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Test } from "forge-std/Test.sol";
import { L1Block } from "../L2/L1Block.sol";
import { L1BlockNumber } from "../legacy/L1BlockNumber.sol";
import { Predeploys } from "../libraries/Predeploys.sol";

contract L1BlockNumber_TestInit is Test {
    L1Block lb;
    L1BlockNumber bn;

    uint64 constant number = 99;

    function setUp() external {
        vm.etch(Predeploys.L1_BLOCK_ATTRIBUTES, address(new L1Block()).code);
        lb = L1Block(Predeploys.L1_BLOCK_ATTRIBUTES);
        bn = new L1BlockNumber();
        vm.prank(lb.DEPOSITOR_ACCOUNT());

        lb.setL1BlockValues({
            _number: number,
            _timestamp: uint64(2),
            _basefee: 3,
            _hash: bytes32(uint256(10)),
            _sequenceNumber: uint64(4),
            _batcherHash: bytes32(uint256(0)),
            _l1FeeOverhead: 2,
            _l1FeeScalar: 3
        });
    }
}

contract L1BlockNumber_Getters_Test is L1BlockNumber_TestInit {
    function test_getL1BlockNumber_succeeds() external {
        assertEq(bn.getL1BlockNumber(), number);
    }
}

contract L1BlockNumber_Fallback_TestFail is L1BlockNumber_TestInit {
    // none
}

contract L1BlockNumber_Fallback_Test is L1BlockNumber_TestInit {
    function test_fallback_succeeds() external {
        (bool success, bytes memory ret) = address(bn).call(hex"");
        assertEq(success, true);
        assertEq(ret, abi.encode(number));
    }
}

contract L1BlockNumber_Receive_TestFail is L1BlockNumber_TestInit {
    // none
}

contract L1BlockNumber_Receive_Test is L1BlockNumber_TestInit {
    function test_receive_succeeds() external {
        (bool success, bytes memory ret) = address(bn).call{ value: 1 }(hex"");
        assertEq(success, true);
        assertEq(ret, abi.encode(number));
    }
}
