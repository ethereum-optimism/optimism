// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Test } from "forge-std/Test.sol";
import { Decimals } from "src/libraries/Decimals.sol";

contract Decimals_Test is Test {
    function test_scaleUp_succeeds() external {
        // 1.0 scaled with 2 decimals is 100
        uint8 decimals = 2;
        uint256 amount = 1 * 10 ** decimals;
        assertEq(amount, 100);

        // 1.0 scaled up to 4 decimals is 1000
        uint8 target = 4;
        uint256 scaled = 1 * 10 ** target;
        assertEq(scaled, 10000);
        assertEq(Decimals.scale(amount, decimals, target), scaled);
    }

    function test_scaleDown_succeeds() external {
        // 1.0 scaled with 4 decimals is 1000
        uint8 decimals = 4;
        uint256 amount = 1 * 10 ** decimals;
        // 1.0 scaled down to 2 decimals
        uint8 target = 2;
        uint256 scaled = 100;
        assertEq(Decimals.scale(amount, decimals, target), scaled);
    }

    function test_scaleEqual_succeeds() external {
        uint256 amount = 0x20;

        uint256 scaled = Decimals.scale({ _amount: amount, _decimals: 2, _target: 2 });

        assertEq(scaled, amount);
    }

    function testFuzz_scale_succeeds(uint256 _amount, uint8 _decimals, uint8 _target) external pure {
        _amount = bound(_amount, 0, type(uint192).max);
        _decimals = uint8(bound(_decimals, 0, 18));
        _target = uint8(bound(_target, 0, 18));

        Decimals.scale(_amount, _decimals, _target);
    }
}
