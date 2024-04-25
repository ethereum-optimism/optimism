// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing utilities
import { Test } from "forge-std/Test.sol";

// Target contract dependencies
import { L1Block } from "src/L2/L1Block.sol";
import { Predeploys } from "src/libraries/Predeploys.sol";

// Target contract
import { L1BlockNumber } from "src/legacy/L1BlockNumber.sol";

contract L1BlockNumberTest is Test {
    L1Block lb;
    L1BlockNumber bn;

    uint64 constant number = 99;

    /// @dev Sets up the test suite.
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

    /// @dev Tests that `getL1BlockNumber` returns the set block number.
    function test_getL1BlockNumber_succeeds() external view {
        assertEq(bn.getL1BlockNumber(), number);
    }

    /// @dev Tests that `fallback` is correctly dispatched.
    function test_fallback_succeeds() external {
        (bool success, bytes memory ret) = address(bn).call(hex"");
        assertEq(success, true);
        assertEq(ret, abi.encode(number));
    }

    /// @dev Tests that `receive` is correctly dispatched.
    function test_receive_succeeds() external {
        (bool success, bytes memory ret) = address(bn).call{ value: 1 }(hex"");
        assertEq(success, true);
        assertEq(ret, abi.encode(number));
    }
}
