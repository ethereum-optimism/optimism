// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { OPContractsManager_TestInit } from "test/L1/OPContractsManager.t.sol";

// Contracts
import { OPContractsManagerContractsContainer } from "src/L1/opcm/OPContractsManagerContractsContainer.sol";

/// @title OPContractsManagerContractsContainer_Constructor_Test
/// @notice Tests the constructor of the `OPContractsManagerContractsContainer` contract.
contract OPContractsManagerContractsContainer_Constructor_Test is OPContractsManager_TestInit {
    /// @notice Tests that the constructor succeeds when the devFeatureBitmap is in dev.
    /// @param _devFeatureBitmap The devFeatureBitmap to use.
    function testFuzz_constructor_devBitmapInDev_succeeds(bytes32 _devFeatureBitmap) public {
        // Etch into the magic testing address.
        vm.etch(address(0xbeefcafe), hex"01");

        // Create empty blueprints and implementations.
        OPContractsManagerContractsContainer.Blueprints memory blueprints;
        OPContractsManagerContractsContainer.Implementations memory implementations;

        // Should not revert.
        OPContractsManagerContractsContainer container = new OPContractsManagerContractsContainer({
            _blueprints: blueprints,
            _implementations: implementations,
            _devFeatureBitmap: _devFeatureBitmap
        });

        // Should have the correct devFeatureBitmap.
        assertEq(container.devFeatureBitmap(), _devFeatureBitmap);
    }

    /// @notice Tests that the constructor reverts when the devFeatureBitmap is in prod.
    /// @param _devFeatureBitmap The devFeatureBitmap to use.
    function testFuzz_constructor_devBitmapInProd_reverts(bytes32 _devFeatureBitmap) public {
        // Anything but zero!
        _devFeatureBitmap = bytes32(bound(uint256(_devFeatureBitmap), 1, type(uint256).max));

        // Make sure magic address has no code.
        vm.etch(address(0xbeefcafe), bytes(""));

        // Set the chain ID to 1.
        vm.chainId(1);

        // Create empty blueprints and implementations.
        OPContractsManagerContractsContainer.Blueprints memory blueprints;
        OPContractsManagerContractsContainer.Implementations memory implementations;

        // Should revert.
        vm.expectRevert(
            OPContractsManagerContractsContainer.OPContractsManagerContractsContainer_DevFeatureInProd.selector
        );
        OPContractsManagerContractsContainer container = new OPContractsManagerContractsContainer({
            _blueprints: blueprints,
            _implementations: implementations,
            _devFeatureBitmap: _devFeatureBitmap
        });

        // Constructor shouldn't have worked, foundry makes this return address(1).
        assertEq(address(container), address(1));
    }

    /// @notice Tests that the constructor succeeds when the devFeatureBitmap is used on the
    ///         mainnet chain ID but this is actually a test environment as shown by the magic
    ///         address having code.
    /// @param _devFeatureBitmap The devFeatureBitmap to use.
    function test_constructor_devBitmapMainnetButTestEnv_succeeds(bytes32 _devFeatureBitmap) public {
        // Make sure magic address has code.
        vm.etch(address(0xbeefcafe), hex"01");

        // Create empty blueprints and implementations.
        OPContractsManagerContractsContainer.Blueprints memory blueprints;
        OPContractsManagerContractsContainer.Implementations memory implementations;

        // Set the chain ID to 1.
        vm.chainId(1);

        // Should not revert.
        OPContractsManagerContractsContainer container = new OPContractsManagerContractsContainer({
            _blueprints: blueprints,
            _implementations: implementations,
            _devFeatureBitmap: _devFeatureBitmap
        });

        // Should have the correct devFeatureBitmap.
        assertEq(container.devFeatureBitmap(), _devFeatureBitmap);
    }
}
