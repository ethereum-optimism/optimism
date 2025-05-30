// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { Test } from "forge-std/Test.sol";

// Contracts
import { ReinitializableBase } from "src/universal/ReinitializableBase.sol";

/// @title ReinitializableBase_Harness
/// @notice Harness contract to allow direct instantiation and testing of `ReinitializableBase`
///         logic.
contract ReinitializableBase_Harness is ReinitializableBase {
    constructor(uint8 _initVersion) ReinitializableBase(_initVersion) { }
}

/// @title ReinitializableBase_InitVersion_Test
/// @notice Tests the `initVersion` function of the `ReinitializableBase` contract.
contract ReinitializableBase_InitVersion_Test is Test {
    /// @notice Tests that the contract is created correctly and initVersion returns the right
    ///         value when the provided init version is non-zero.
    /// @param _initVersion The init version to use when creating the contract.
    function testFuzz_initVersion_validVersion_succeeds(uint8 _initVersion) public {
        // Zero version not allowed.
        _initVersion = uint8(bound(_initVersion, 1, type(uint8).max));

        // Deploy the reinitializable contract.
        ReinitializableBase_Harness harness = new ReinitializableBase_Harness(_initVersion);

        // Check the init version.
        assertEq(harness.initVersion(), _initVersion);
    }

    /// @notice Tests that the contract creation reverts when the init version is zero.
    function test_initVersion_zeroVersion_reverts() public {
        vm.expectRevert(ReinitializableBase.ReinitializableBase_ZeroInitVersion.selector);
        new ReinitializableBase_Harness(0);
    }
}
