// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing utilities
import { Bridge_Initializer } from "test/setup/Bridge_Initializer.sol";

// Libraries
import { Predeploys } from "src/libraries/Predeploys.sol";
import { IBeacon } from "@openzeppelin/contracts/proxy/beacon/IBeacon.sol";

/// @title OptimismSuperchainERC20BeaconTest
/// @notice Contract for testing the OptimismSuperchainERC20Beacon contract.
contract OptimismSuperchainERC20BeaconTest is Bridge_Initializer {
    /// @notice Sets up the test suite.
    function setUp() public override {
        super.enableInterop();
        super.setUp();
    }

    /// @notice Test that calling the implementation function returns the correct implementation address.
    function test_implementation_is_correct() public view {
        IBeacon beacon = IBeacon(Predeploys.OPTIMISM_SUPERCHAIN_ERC20_BEACON);
        assertEq(beacon.implementation(), Predeploys.OPTIMISM_SUPERCHAIN_ERC20);
    }
}
