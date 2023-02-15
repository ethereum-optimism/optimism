//SPDX-License-Identifier: MIT
pragma solidity 0.8.16;

import { Test } from "forge-std/Test.sol";
import { CheckBalanceHigh } from "../universal/drippie/dripchecks/CheckBalanceHigh.sol";

/**
 * @title  CheckBalanceHighTest
 * @notice Tests the CheckBalanceHigh contract via fuzzing both the success case
 *         and the failure case.
 */
contract CheckBalanceHighTest is Test {
    /**
     * @notice An instance of the CheckBalanceHigh contract.
     */
    CheckBalanceHigh c;

    /**
     * @notice Deploy the `CheckTrue` contract.
     */
    function setUp() external {
        c = new CheckBalanceHigh();
    }

    /**
     * @notice Fuzz the `check` function and assert that it always returns true
     *         when the target's balance is larger than the threshold.
     */
    function testFuzz_check_succeeds(address _target, uint256 _threshold) external {
        CheckBalanceHigh.Params memory p = CheckBalanceHigh.Params({
            target: _target,
            threshold: _threshold
        });

        // prevent overflows
        vm.assume(_threshold != type(uint256).max);
        vm.deal(_target, _threshold + 1);

        assertEq(c.check(abi.encode(p)), true);
    }

    /**
     * @notice Fuzz the `check` function and assert that it always returns false
     *         when the target's balance is smaller than the threshold.
     */
    function testFuzz_check_fails(address _target, uint256 _threshold) external {
        CheckBalanceHigh.Params memory p = CheckBalanceHigh.Params({
            target: _target,
            threshold: _threshold
        });

        vm.assume(_target.balance < _threshold);

        assertEq(c.check(abi.encode(p)), false);
    }
}
