//SPDX-License-Identifier: MIT
pragma solidity 0.8.10;

import { Test } from "forge-std/Test.sol";
import { L1Block } from "../L2/L1Block.sol";
import { L1BlockNumber } from "../L2/L1BlockNumber.sol";
import { Lib_PredeployAddresses } from "../libraries/Lib_PredeployAddresses.sol";

contract L1BlockNumberTest is Test {
    L1Block lb;
    L1BlockNumber bn;

    function setUp() external {
        vm.etch(Lib_PredeployAddresses.L1_BLOCK_ATTRIBUTES, address(new L1Block()).code);
        lb = L1Block(Lib_PredeployAddresses.L1_BLOCK_ATTRIBUTES);
        bn = new L1BlockNumber();
        vm.prank(lb.DEPOSITOR_ACCOUNT());
        lb.setL1BlockValues(uint64(999), uint64(2), 3, keccak256(abi.encode(1)), uint64(4));
    }

    function test_getL1BlockNumber() external {
        assertEq(bn.getL1BlockNumber(), 999);
    }

    function test_fallback() external {
        (bool success, bytes memory ret) = address(bn).call(hex"");
        assertEq(success, true);
        assertEq(ret, abi.encode(999));
    }

    function test_receive() external {
        (bool success, bytes memory ret) = address(bn).call{ value: 1 }(hex"");
        assertEq(success, true);
        assertEq(ret, abi.encode(999));
    }
}
