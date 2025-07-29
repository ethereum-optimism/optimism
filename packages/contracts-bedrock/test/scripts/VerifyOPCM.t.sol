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

    function verifyOpcmImmutableVariables(IOPContractsManager _opcm) public returns (bool) {
        return _verifyOpcmImmutableVariables(_opcm);
    }

    function validateAllGettersAccounted() public {
        return _validateAllGettersAccounted();
    }

    function setExpectedGetter(string memory _getter, string memory _verificationMethod) public {
        expectedGetters[_getter] = _verificationMethod;
    }

    function removeExpectedGetter(string memory _getter) public {
        expectedGetters[_getter] = "";
    }
}

/// @title VerifyOPCM_TestInit
/// @notice Reusable test initialization for `VerifyOPCM` tests.
contract VerifyOPCM_TestInit is OPContractsManager_TestInit {
    VerifyOPCM_Harness internal harness;

    function setUp() public virtual override {
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
    function setUp() public override {
        super.setUp();
        setupEnvVars();
    }

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

    /// @notice Tests that immutable variables are correctly verified in the OPCM contract.
    function test_verifyOpcmImmutableVariables_succeeds() public {
        // Coverage changes bytecode and causes failures, skip.
        skipIfCoverage();

        // Ensure environment variables are set correctly (in case other tests modified them)
        setupEnvVars();

        // Test that the immutable variables are correctly verified.
        // Environment variables are set in setUp() to match the actual OPCM addresses.
        bool result = harness.verifyOpcmImmutableVariables(opcm);
        assertTrue(result, "OPCM immutable variables should be valid");
    }

    /// @notice Mocks a call to the OPCM contract and verifies validation fails.
    /// @param _selector The function selector for the OPCM contract method to mock.
    function _assertOnOpcmGetter(bytes4 _selector) internal {
        bytes memory callData = abi.encodePacked(_selector);
        vm.mockCall(address(opcm), callData, abi.encode(address(0x8888)));

        // Verify that immutable variables fail validation
        bool result = harness.verifyOpcmImmutableVariables(opcm);
        assertFalse(result, "OPCM with invalid immutable variables should fail verification");

        // Clear mock calls and restore original environment variables to avoid test isolation issues
        vm.clearMockedCalls();
    }

    /// @notice Tests that the script fails when OPCM immutable variables are invalid.
    /// We test this by setting expected addresses and mocking OPCM methods to return different addresses.
    function test_verifyOpcmImmutableVariables_mismatch_fails() public {
        // Coverage changes bytecode and causes failures, skip.
        skipIfCoverage();

        // Set expected addresses via environment variables
        address expectedSuperchainConfig = address(0x1111);
        address expectedProtocolVersions = address(0x2222);
        address expectedSuperchainProxyAdmin = address(0x3333);
        address expectedUpgradeController = address(0x4444);

        vm.setEnv("EXPECTED_SUPERCHAIN_CONFIG", vm.toString(expectedSuperchainConfig));
        vm.setEnv("EXPECTED_PROTOCOL_VERSIONS", vm.toString(expectedProtocolVersions));
        vm.setEnv("EXPECTED_SUPERCHAIN_PROXY_ADMIN", vm.toString(expectedSuperchainProxyAdmin));
        vm.setEnv("EXPECTED_UPGRADE_CONTROLLER", vm.toString(expectedUpgradeController));

        // Test that mocking each individual getter causes verification to fail
        _assertOnOpcmGetter(IOPContractsManager.superchainConfig.selector);
        _assertOnOpcmGetter(IOPContractsManager.protocolVersions.selector);
        _assertOnOpcmGetter(IOPContractsManager.superchainProxyAdmin.selector);
        _assertOnOpcmGetter(IOPContractsManager.upgradeController.selector);

        // Reset environment variables to correct values (as set in setUp())
        setupEnvVars();
    }

    /// @notice Tests that the ABI getter validation succeeds when all getters are accounted for.
    function test_validateAllGettersAccounted_succeeds() public {
        // This should succeed as setUp() configures all expected getters
        harness.validateAllGettersAccounted();
    }

    /// @notice Tests that the ABI getter validation fails when there are unaccounted getters.
    /// We test this by removing an expected getter from the mapping.
    function test_validateAllGettersAccounted_unaccountedGetters_reverts() public {
        // Remove one of the expected getters to simulate an unaccounted getter
        harness.removeExpectedGetter("blueprints");

        // This should revert with VerifyOPCM_UnaccountedGetters error
        // The error includes the array of unaccounted getters as a parameter
        string[] memory expectedUnaccounted = new string[](1);
        expectedUnaccounted[0] = "blueprints";
        vm.expectRevert(abi.encodeWithSelector(VerifyOPCM.VerifyOPCM_UnaccountedGetters.selector, expectedUnaccounted));
        harness.validateAllGettersAccounted();
    }
}
