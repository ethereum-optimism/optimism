// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing utilities
import { CommonTest } from "test/setup/CommonTest.sol";

// Libraries
import { Predeploys } from "src/libraries/Predeploys.sol";
import { IBeacon } from "@openzeppelin/contracts/proxy/beacon/IBeacon.sol";

/// @title OptimismSuperchainERC20Beacon_TestInit
/// @notice Reusable test initialization for `OptimismSuperchainERC20Beacon` tests.
abstract contract OptimismSuperchainERC20Beacon_TestInit is CommonTest {
    /// @notice Sets up the test suite.
    function setUp() public override {
        // Skip the test until OptimismSuperchainERC20Beacon is integrated again
        vm.skip(true);

        super.enableInterop();
        super.setUp();
    }
}

/// @title OptimismSuperchainERC20Beacon_Implementation_Test
/// @notice Contract for testing the `implementation` function of the
///         `OptimismSuperchainERC20Beacon` contract.
contract OptimismSuperchainERC20Beacon_Implementation_Test is OptimismSuperchainERC20Beacon_TestInit {
    /// @notice Test that calling the implementation function returns the correct implementation address.
    function test_implementation_isCorrect_works() public view {
        IBeacon beacon = IBeacon(Predeploys.OPTIMISM_SUPERCHAIN_ERC20_BEACON);
        assertEq(beacon.implementation(), Predeploys.OPTIMISM_SUPERCHAIN_ERC20);
    }
}
