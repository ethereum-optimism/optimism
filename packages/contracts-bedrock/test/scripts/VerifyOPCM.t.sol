// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Foundry
import { VmSafe } from "forge-std/Vm.sol";

// Tests
import { OPContractsManager_TestInit } from "test/L1/OPContractsManager.t.sol";

// Scripts
import { VerifyOPCM } from "scripts/deploy/VerifyOPCM.s.sol";

// Interfaces
import { IOPContractsManager } from "interfaces/L1/IOPContractsManager.sol";

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
}

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

contract VerifyOPCM_run_Test is VerifyOPCM_TestInit {
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
}
