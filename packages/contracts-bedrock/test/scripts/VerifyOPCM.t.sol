// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Foundry
import { VmSafe } from "forge-std/Vm.sol";

// Libraries
import { LibString } from "@solady/utils/LibString.sol";

// Tests
import { OPContractsManager_TestInit } from "test/L1/OPContractsManager.t.sol";

// Scripts
import { VerifyOPCM } from "scripts/deploy/VerifyOPCM.s.sol";

// Interfaces
import { IOPContractsManager, IOPContractsManagerUpgrader } from "interfaces/L1/IOPContractsManager.sol";

contract VerifyOPCM_Harness is VerifyOPCM {
    function loadArtifactInfo(string memory _artifactPath) public view returns (ArtifactInfo memory) {
        return _loadArtifactInfo(_artifactPath);
    }

    function getOpcmPropertyRefs(IOPContractsManager _opcm) public returns (OpcmContractRef[] memory) {
        return _getOpcmPropertyRefs(_opcm);
    }

    function getOpcmContractRefs(
        IOPContractsManager _opcm,
        string memory _property,
        bool _blueprint
    )
        public
        returns (OpcmContractRef[] memory)
    {
        return _getOpcmContractRefs(_opcm, _property, _blueprint);
    }

    function buildArtifactPath(string memory _contractName) public view returns (string memory) {
        return _buildArtifactPath(_contractName);
    }

    function verifyContractsContainerConsistency(OpcmContractRef[] memory _propRefs) public view {
        return _verifyContractsContainerConsistency(_propRefs);
    }
}

/// @title VerifyOPCM_TestInit
/// @notice Reusable test initialization for `VerifyOPCM` tests.
contract VerifyOPCM_TestInit is OPContractsManager_TestInit {
    VerifyOPCM_Harness internal harness;

    function setUp() public override {
        super.setUp();
        harness = new VerifyOPCM_Harness();
        harness.setUp();
    }

    /// @notice Skips if running in coverage mode.
    function skipIfCoverage() public {
        if (vm.isContext(VmSafe.ForgeContext.Coverage)) {
            vm.skip(true);
        }
    }
}

/// @title VerifyOPCM_Run_Test
/// @notice Tests the `run` function of the `VerifyOPCM` script.
contract VerifyOPCM_Run_Test is VerifyOPCM_TestInit {
    /// @notice Tests that the script succeeds when no changes are introduced.
    function test_run_succeeds() public {
        // Coverage changes bytecode and causes failures, skip.
        skipIfCoverage();

        // Run the script.
        harness.run(address(opcm), true);
    }

    /// @notice Tests that the script succeeds when differences are introduced into the immutable
    ///         variables of implementation contracts. Fuzzing is too slow here, randomness is good
    ///         enough.
    function test_run_implementationDifferentInsideImmutable_succeeds() public {
        // Coverage changes bytecode and causes failures, skip.
        skipIfCoverage();

        // Grab the list of implementations.
        VerifyOPCM.OpcmContractRef[] memory refs = harness.getOpcmContractRefs(opcm, "implementations", false);

        // Change 256 bytes at random.
        for (uint8 i = 0; i < 255; i++) {
            // Pick a random implementation to change.
            uint256 randomImplIndex = vm.randomUint(0, refs.length - 1);
            VerifyOPCM.OpcmContractRef memory ref = refs[randomImplIndex];

            // Get the code for the implementation.
            bytes memory implCode = ref.addr.code;

            // Grab the artifact info for the implementation.
            VerifyOPCM.ArtifactInfo memory artifact = harness.loadArtifactInfo(harness.buildArtifactPath(ref.name));

            // Skip, no immutable references. Will make some fuzz runs useless but it's not worth
            // the extra complexity to handle this properly.
            if (artifact.immutableRefs.length == 0) {
                continue;
            }

            // Find a random byte that's inside an immutable reference.
            bool inImmutable = false;
            uint256 randomDiffPosition;
            while (!inImmutable) {
                randomDiffPosition = vm.randomUint(0, implCode.length - 1);
                inImmutable = false;
                for (uint256 j = 0; j < artifact.immutableRefs.length; j++) {
                    VerifyOPCM.ImmutableRef memory immRef = artifact.immutableRefs[j];
                    if (randomDiffPosition >= immRef.offset && randomDiffPosition < immRef.offset + immRef.length) {
                        inImmutable = true;
                        break;
                    }
                }
            }

            // Change the byte to something new.
            bytes1 existingByte = implCode[randomDiffPosition];
            bytes1 newByte = bytes1(uint8(vm.randomUint(0, 255)));
            while (newByte == existingByte) {
                newByte = bytes1(uint8(vm.randomUint(0, 255)));
            }

            // Write the new byte to the code.
            implCode[randomDiffPosition] = newByte;
            vm.etch(ref.addr, implCode);
        }

        // Run the script.
        // No revert expected.
        harness.run(address(opcm), true);
    }

    /// @notice Tests that the script reverts when differences are introduced into the code of
    ///         implementation contracts that are not inside immutable references. Fuzzing is too
    ///         slow here, randomness is good enough.
    function test_run_implementationDifferentOutsideImmutable_reverts() public {
        // Coverage changes bytecode and causes failures, skip.
        skipIfCoverage();

        // Grab the list of implementations.
        VerifyOPCM.OpcmContractRef[] memory refs = harness.getOpcmContractRefs(opcm, "implementations", false);

        // Change 256 bytes at random.
        for (uint8 i = 0; i < 255; i++) {
            // Pick a random implementation to change.
            uint256 randomImplIndex = vm.randomUint(0, refs.length - 1);
            VerifyOPCM.OpcmContractRef memory ref = refs[randomImplIndex];

            // Get the code for the implementation.
            bytes memory implCode = ref.addr.code;

            // Grab the artifact info for the implementation.
            VerifyOPCM.ArtifactInfo memory artifact = harness.loadArtifactInfo(harness.buildArtifactPath(ref.name));

            // Find a random byte that isn't in an immutable reference.
            bool inImmutable = true;
            uint256 randomDiffPosition;
            while (inImmutable) {
                randomDiffPosition = vm.randomUint(0, implCode.length - 1);
                inImmutable = false;
                for (uint256 j = 0; j < artifact.immutableRefs.length; j++) {
                    VerifyOPCM.ImmutableRef memory immRef = artifact.immutableRefs[j];
                    if (randomDiffPosition >= immRef.offset && randomDiffPosition < immRef.offset + immRef.length) {
                        inImmutable = true;
                        break;
                    }
                }
            }

            // Change the byte to something new.
            bytes1 existingByte = implCode[randomDiffPosition];
            bytes1 newByte = bytes1(uint8(vm.randomUint(0, 255)));
            while (newByte == existingByte) {
                newByte = bytes1(uint8(vm.randomUint(0, 255)));
            }

            // Write the new byte to the code.
            implCode[randomDiffPosition] = newByte;
            vm.etch(ref.addr, implCode);
        }

        // Run the script.
        vm.expectRevert(VerifyOPCM.VerifyOPCM_Failed.selector);
        harness.run(address(opcm), true);
    }

    /// @notice Tests that the script reverts when differences are introduced into the code of
    ///         blueprints. Unlike immutables, any difference anywhere in the blueprint should
    ///         cause the script to revert. Fuzzing is too slow here, randomness is good enough.
    function test_run_blueprintAnyDifference_reverts() public {
        // Coverage changes bytecode and causes failures, skip.
        skipIfCoverage();

        // Grab the list of blueprints.
        VerifyOPCM.OpcmContractRef[] memory refs = harness.getOpcmContractRefs(opcm, "blueprints", true);

        // Change 256 bytes at random.
        for (uint8 i = 0; i < 255; i++) {
            // Pick a random blueprint to change.
            uint256 randomBlueprintIndex = vm.randomUint(0, refs.length - 1);
            VerifyOPCM.OpcmContractRef memory ref = refs[randomBlueprintIndex];

            // Get the code for the blueprint.
            address blueprint = ref.addr;
            bytes memory blueprintCode = blueprint.code;

            // We don't care about immutable references for blueprints.
            // Pick a random position.
            uint256 randomDiffPosition = vm.randomUint(0, blueprintCode.length - 1);

            // Change the byte to something new.
            bytes1 existingByte = blueprintCode[randomDiffPosition];
            bytes1 newByte = bytes1(uint8(vm.randomUint(0, 255)));
            while (newByte == existingByte) {
                newByte = bytes1(uint8(vm.randomUint(0, 255)));
            }

            // Write the new byte to the code.
            blueprintCode[randomDiffPosition] = newByte;
            vm.etch(blueprint, blueprintCode);
        }

        // Run the script.
        vm.expectRevert(VerifyOPCM.VerifyOPCM_Failed.selector);
        harness.run(address(opcm), true);
    }

    /// @notice Tests that the script verifies all component contracts have the same contractsContainer address.
    function test_verifyContractsContainerConsistency_succeeds() public {
        // Coverage changes bytecode and causes failures, skip.
        skipIfCoverage();

        // Get the property references (which include the component addresses)
        VerifyOPCM.OpcmContractRef[] memory propRefs = harness.getOpcmPropertyRefs(opcm);

        // This should succeed with the current setup where all contracts have the same containerAddress.
        harness.verifyContractsContainerConsistency(propRefs);
    }

    /// @notice Tests that the script reverts when contracts have different contractsContainer addresses.
    function test_verifyContractsContainerConsistency_mismatch_reverts() public {
        // Coverage changes bytecode and causes failures, skip.
        skipIfCoverage();

        // Get the property references (which include the component addresses)
        VerifyOPCM.OpcmContractRef[] memory propRefs = harness.getOpcmPropertyRefs(opcm);

        // Create a different address to simulate a mismatch.
        address differentContainer = address(0x9999999999999999999999999999999999999999);

        // Mock the first OPCM component found to return a different contractsContainer address
        _mockFirstOpcmComponent(propRefs, differentContainer);

        // Now the consistency check should fail.
        vm.expectRevert(VerifyOPCM.VerifyOPCM_ContractsContainerMismatch.selector);
        harness.verifyContractsContainerConsistency(propRefs);
    }

    /// @notice Tests that each OPCM component can be individually tested for container mismatch.
    function test_verifyContractsContainerConsistency_eachComponent_reverts() public {
        // Coverage changes bytecode and causes failures, skip.
        skipIfCoverage();

        // Get the property references (which include the component addresses)
        VerifyOPCM.OpcmContractRef[] memory propRefs = harness.getOpcmPropertyRefs(opcm);

        // Test each OPCM component individually (only those that actually have contractsContainer())
        address differentContainer = address(0x9999999999999999999999999999999999999999);

        uint256 componentsWithContainerTested = 0;
        for (uint256 i = 0; i < propRefs.length; i++) {
            string memory field = propRefs[i].field;
            if (_hasContractsContainer(field)) {
                // Mock this specific component to return a different address
                vm.mockCall(
                    propRefs[i].addr,
                    abi.encodeCall(IOPContractsManagerUpgrader.contractsContainer, ()),
                    abi.encode(differentContainer)
                );

                // The consistency check should fail
                vm.expectRevert(VerifyOPCM.VerifyOPCM_ContractsContainerMismatch.selector);
                harness.verifyContractsContainerConsistency(propRefs);

                // Clear the mock for next iteration
                vm.clearMockedCalls();
                componentsWithContainerTested++;
            }
        }

        // Ensure we actually tested some components (currently: deployer, gameTypeAdder, upgrader, interopMigrator)
        assertGt(componentsWithContainerTested, 0, "Should have tested at least one component");
    }

    /// @notice Utility function to mock the first OPCM component's contractsContainer address.
    /// @param _propRefs Array of property references to search through.
    /// @param _mockAddress The address to mock the contractsContainer call to return.
    function _mockFirstOpcmComponent(VerifyOPCM.OpcmContractRef[] memory _propRefs, address _mockAddress) internal {
        for (uint256 i = 0; i < _propRefs.length; i++) {
            string memory field = _propRefs[i].field;
            // Check if this is an OPCM component that has contractsContainer()
            if (_hasContractsContainer(field)) {
                vm.mockCall(
                    _propRefs[i].addr,
                    abi.encodeCall(IOPContractsManagerUpgrader.contractsContainer, ()),
                    abi.encode(_mockAddress)
                );
                return;
            }
        }
    }

    /// @notice Helper function to check if a field represents an OPCM component.
    /// @param _field The field name to check.
    /// @return True if the field represents an OPCM component (starts with "opcm"), false otherwise.
    function _isOpcmComponent(string memory _field) internal pure returns (bool) {
        return LibString.startsWith(_field, "opcm");
    }

    /// @notice Helper function to check if a field represents an OPCM component that has contractsContainer().
    /// @param _field The field name to check.
    /// @return True if the field represents an OPCM component with contractsContainer(), false otherwise.
    function _hasContractsContainer(string memory _field) internal pure returns (bool) {
        // Check if it starts with "opcm"
        if (!LibString.startsWith(_field, "opcm")) {
            return false;
        }

        // Components that start with "opcm" but don't extend OPContractsManagerBase (and thus don't have
        // contractsContainer())
        string[] memory exclusions = new string[](1);
        exclusions[0] = "opcmStandardValidator";

        // Check if the field is in the exclusion list
        for (uint256 i = 0; i < exclusions.length; i++) {
            if (LibString.eq(_field, exclusions[i])) {
                return false;
            }
        }

        return true;
    }
}
