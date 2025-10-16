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

/// @title ReinitializableBase_Constructor_Test
/// @notice Tests the constructor of the `ReinitializableBase` contract.
contract ReinitializableBase_Constructor_Test is Test {
    /// @notice Tests that the contract is created correctly with a valid init version.
    /// @param _initVersion The init version to use when creating the contract.
    function testFuzz_constructor_validVersion_succeeds(uint8 _initVersion) public {
        _initVersion = uint8(bound(_initVersion, 1, type(uint8).max));

        ReinitializableBase_Harness harness = new ReinitializableBase_Harness(_initVersion);

        assertEq(harness.initVersion(), _initVersion);
    }

    /// @notice Tests that the contract creation reverts when the init version is zero.
    function test_constructor_zeroVersion_reverts() public {
        vm.expectRevert(ReinitializableBase.ReinitializableBase_ZeroInitVersion.selector);
        new ReinitializableBase_Harness(0);
    }
}

/// @title ReinitializableBase_InitVersion_Test
/// @notice Tests the `initVersion` getter function of the `ReinitializableBase` contract.
contract ReinitializableBase_InitVersion_Test is Test {
    /// @notice Tests that initVersion getter function works correctly.
    /// @param _initVersion The init version to test.
    function testFuzz_initVersion_succeeds(uint8 _initVersion) public {
        _initVersion = uint8(bound(_initVersion, 1, type(uint8).max));

        ReinitializableBase_Harness harness = new ReinitializableBase_Harness(_initVersion);

        assertEq(harness.initVersion(), _initVersion);
    }
}
