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
    /// @notice Tests that the contract creation reverts when init version is zero.
    function test_constructor_zeroVersion_reverts() public {
        vm.expectRevert(ReinitializableBase.ReinitializableBase_ZeroInitVersion.selector);
        new ReinitializableBase_Harness(0);
    }

    /// @notice Tests that constructor succeeds with valid non-zero init versions.
    /// @param _initVersion Init version to use when creating the contract.
    function testFuzz_constructor_validVersion_succeeds(uint8 _initVersion) public {
        _initVersion = uint8(bound(_initVersion, 1, type(uint8).max));
        ReinitializableBase_Harness harness = new ReinitializableBase_Harness(_initVersion);
        assertEq(harness.initVersion(), _initVersion);
    }
}
