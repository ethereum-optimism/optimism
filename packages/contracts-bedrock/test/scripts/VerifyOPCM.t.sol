// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Libraries
import { LibString } from "@solady/utils/LibString.sol";
import { DevFeatures } from "src/libraries/DevFeatures.sol";

// Tests
import { CommonTest } from "test/setup/CommonTest.sol";

// Scripts
import { VerifyOPCM } from "scripts/deploy/VerifyOPCM.s.sol";

// Interfaces
import { IOPContractsManager, IOPContractsManagerUpgrader } from "interfaces/L1/IOPContractsManager.sol";
import { IOPContractsManagerStandardValidator } from "interfaces/L1/IOPContractsManagerStandardValidator.sol";
import { IOPContractsManagerV2 } from "interfaces/L1/opcm/IOPContractsManagerV2.sol";
import { IOptimismPortal2 } from "interfaces/L1/IOptimismPortal2.sol";
import { IAnchorStateRegistry } from "interfaces/dispute/IAnchorStateRegistry.sol";
import { IMIPS64 } from "interfaces/cannon/IMIPS64.sol";

contract VerifyOPCM_Harness is VerifyOPCM {
    bool private _skipSecurityChecks;

    function setSkipSecurityValueChecks(bool _skip) public {
        _skipSecurityChecks = _skip;
    }

    function skipSecurityValueChecks() public view override returns (bool) {
        return _skipSecurityChecks;
    }

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

    function verifyOpcmUtilsConsistency(OpcmContractRef[] memory _propRefs) public view {
        return _verifyOpcmUtilsConsistency(_propRefs);
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

    function verifyPreimageOracle(IMIPS64 _mips) public view returns (bool) {
        return _verifyPreimageOracle(_mips);
    }

    function verifyPortalDelays(IOptimismPortal2 _portal) public view returns (bool) {
        return _verifyPortalDelays(_portal);
    }

    function verifyAnchorStateRegistryDelays(IAnchorStateRegistry _asr) public view returns (bool) {
        return _verifyAnchorStateRegistryDelays(_asr);
    }

    function verifyStandardValidatorArgs(IOPContractsManager _opcm, address _validator) public returns (bool) {
        return _verifyStandardValidatorArgs(_opcm, _validator);
    }

    function setValidatorGetterCheck(string memory _getter, string memory _check) public {
        validatorGetterChecks[_getter] = _check;
    }
}

/// @title VerifyOPCM_TestInit
/// @notice Reusable test initialization for `VerifyOPCM` tests.
abstract contract VerifyOPCM_TestInit is CommonTest {
    VerifyOPCM_Harness internal harness;

    function setUp() public virtual override {
        super.setUp();
        harness = new VerifyOPCM_Harness();
        harness.setUp();

        // If OPCM V2 is enabled, set up the test environment for OPCM V2.
        // nosemgrep: sol-style-vm-env-only-in-config-sol
        if (vm.envOr("DEV_FEATURE__OPCM_V2", false)) {
            opcm = IOPContractsManager(address(opcmV2));
        }

        // Always set up the environment variables for the test.
        setupEnvVars();

        // Set the OPCM address so that runSingle also runs for V2 OPCM if the dev feature is enabled.
        vm.setEnv("OPCM_ADDRESS", vm.toString(address(opcm)));
    }

    /// @notice Sets up the environment variables for the VerifyOPCM test.
    function setupEnvVars() public {
        // If OPCM V2 is not enabled, set the environment variables for the old OPCM.
        if (!isDevFeatureEnabled(DevFeatures.OPCM_V2)) {
            vm.setEnv("EXPECTED_SUPERCHAIN_CONFIG", vm.toString(address(opcm.superchainConfig())));
            vm.setEnv("EXPECTED_PROTOCOL_VERSIONS", vm.toString(address(opcm.protocolVersions())));
        }

        // Grab a reference to the validator.
        IOPContractsManagerStandardValidator validator =
            IOPContractsManagerStandardValidator(opcm.opcmStandardValidator());

        // Fetch all of the expected values from existing contracts, this just makes the tests pass
        // by default. We will override these with bad values during tests to demonstrate that the
        // script correctly rejects them.
        vm.setEnv("EXPECTED_L1_PAO_MULTISIG", vm.toString(validator.l1PAOMultisig()));
        vm.setEnv("EXPECTED_CHALLENGER", vm.toString(validator.challenger()));
        vm.setEnv("EXPECTED_WITHDRAWAL_DELAY_SECONDS", vm.toString(validator.withdrawalDelaySeconds()));
        vm.setEnv("EXPECTED_SUPERCHAIN_CONFIG", vm.toString(address(optimismPortal2.superchainConfig())));
        vm.setEnv("EXPECTED_PROOF_MATURITY_DELAY_SECONDS", vm.toString(optimismPortal2.proofMaturityDelaySeconds()));
        vm.setEnv(
            "EXPECTED_DISPUTE_GAME_FINALITY_DELAY_SECONDS",
            vm.toString(anchorStateRegistry.disputeGameFinalityDelaySeconds())
        );
    }
}

/// @title VerifyOPCM_Run_Test
/// @notice Tests the `run` function of the `VerifyOPCM` script.
contract VerifyOPCM_Run_Test is VerifyOPCM_TestInit {
    function setUp() public override {
        super.setUp();
    }

    /// @notice Tests that the script succeeds when no changes are introduced.
    function test_run_succeeds() public {
        // Coverage changes bytecode and causes failures, skip.
        skipIfCoverage();

        // Run the script.
        harness.run(address(opcm), true);
    }

    /// @notice Tests that the runSingle script succeeds when run against production contracts.
    function test_runSingle_succeeds() public {
        VerifyOPCM.OpcmContractRef[][2] memory refsByType;
        refsByType[0] = harness.getOpcmContractRefs(opcm, "implementations", false);
        refsByType[1] = harness.getOpcmContractRefs(opcm, "blueprints", true);

        for (uint8 i = 0; i < refsByType.length; i++) {
            for (uint256 j = 0; j < refsByType[i].length; j++) {
                VerifyOPCM.OpcmContractRef memory ref = refsByType[i][j];

                // TODO(#17262): Remove this skip once Super dispute games are no longer behind a feature flag
                if (_isSuperDisputeGameContractRef(ref)) {
                    continue;
                }

                harness.runSingle(ref.name, ref.addr, true);
            }
        }
    }

    function test_run_bitmapNotEmptyOnMainnet_reverts(bytes32 _devFeatureBitmap) public {
        // Coverage changes bytecode and causes failures, skip.
        skipIfCoverage();

        // Anything but zero!
        _devFeatureBitmap = bytes32(bound(uint256(_devFeatureBitmap), 1, type(uint256).max));

        // Mock opcm to return a non-zero dev feature bitmap.
        vm.mockCall(
            address(opcm), abi.encodeCall(IOPContractsManager.devFeatureBitmap, ()), abi.encode(_devFeatureBitmap)
        );

        // Set the chain ID to 1.
        vm.chainId(1);

        // Disable testing environment.
        vm.etch(address(0xbeefcafe), bytes(""));

        // Run the script.
        vm.expectRevert(VerifyOPCM.VerifyOPCM_DevFeatureBitmapNotEmpty.selector);
        harness.run(address(opcm), true);
    }

    /// @notice Tests that the script succeeds when differences are introduced into the immutable
    ///         variables of implementation contracts. Fuzzing is too slow here, randomness is good
    ///         enough.
    function test_run_implementationDifferentInsideImmutable_succeeds() public {
        // Coverage changes bytecode and causes failures, skip.
        skipIfCoverage();

        // Skip security value checks since this test deliberately corrupts immutable values.
        harness.setSkipSecurityValueChecks(true);

        // Grab the list of implementations.
        VerifyOPCM.OpcmContractRef[] memory refs = harness.getOpcmContractRefs(opcm, "implementations", false);

        // Check if V2 dispute games feature is enabled
        bytes32 bitmap = opcm.devFeatureBitmap();
        bool superGamesEnabled = DevFeatures.isDevFeatureEnabled(bitmap, DevFeatures.OPTIMISM_PORTAL_INTEROP);

        // Change 256 bytes at random.
        for (uint256 i = 0; i < 255; i++) {
            // Pick a random implementation to change.
            uint256 randomImplIndex = vm.randomUint(0, refs.length - 1);
            VerifyOPCM.OpcmContractRef memory ref = refs[randomImplIndex];

            // Skip super dispute games when feature disabled
            if (_isSuperDisputeGameContractRef(ref) && !superGamesEnabled) {
                continue;
            }

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

        // Skip security value checks since corrupted bytecode may break contract queries.
        harness.setSkipSecurityValueChecks(true);

        // Grab the list of implementations.
        VerifyOPCM.OpcmContractRef[] memory refs = harness.getOpcmContractRefs(opcm, "implementations", false);

        // Check if V2 dispute games feature is enabled
        bytes32 bitmap = opcm.devFeatureBitmap();
        bool superGamesEnabled = DevFeatures.isDevFeatureEnabled(bitmap, DevFeatures.OPTIMISM_PORTAL_INTEROP);

        // Change 256 bytes at random.
        for (uint8 i = 0; i < 255; i++) {
            // Pick a random implementation to change.
            uint256 randomImplIndex = vm.randomUint(0, refs.length - 1);
            VerifyOPCM.OpcmContractRef memory ref = refs[randomImplIndex];

            // Skip super dispute games when feature disabled
            if (_isSuperDisputeGameContractRef(ref) && !superGamesEnabled) {
                continue;
            }

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

            if (blueprintCode.length == 0) {
                continue;
            }

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
            // We want to do nothing if the field is opcmUtils because it all other components get their
            // contractsContainer from it
            // So mocking it to return a different value would make the other components have the same return value.
            if (_hasContractsContainer(field) && !LibString.eq(field, "opcmUtils")) {
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

    /// @notice Tests that the script verifies all component contracts with opcmUtils() have the same address.
    function test_verifyOpcmUtilsConsistency_succeeds() public {
        // Coverage changes bytecode and causes failures, skip.
        skipIfCoverage();

        // Only run for OPCM V2
        skipIfDevFeatureDisabled(DevFeatures.OPCM_V2);

        // Get the property references (which include the component addresses)
        VerifyOPCM.OpcmContractRef[] memory propRefs = harness.getOpcmPropertyRefs(opcm);

        // This should succeed with the current setup where all contracts have the same opcmUtils address.
        harness.verifyOpcmUtilsConsistency(propRefs);
    }

    /// @notice Tests that the script reverts when contracts have different opcmUtils addresses.
    function test_verifyOpcmUtilsConsistency_mismatch_reverts() public {
        // Coverage changes bytecode and causes failures, skip.
        skipIfCoverage();

        // Only run for OPCM V2
        skipIfDevFeatureDisabled(DevFeatures.OPCM_V2);

        // Get the property references (which include the component addresses)
        VerifyOPCM.OpcmContractRef[] memory propRefs = harness.getOpcmPropertyRefs(opcm);

        // Create a different address to simulate a mismatch.
        address differentUtils = address(0x9999999999999999999999999999999999999999);

        // Mock the first component with opcmUtils() to return a different address
        _mockFirstOpcmUtilsComponent(propRefs, differentUtils);

        // Now the consistency check should fail.
        vm.expectRevert(VerifyOPCM.VerifyOPCM_OpcmUtilsMismatch.selector);
        harness.verifyOpcmUtilsConsistency(propRefs);
    }

    /// @notice Tests that each OPCM component with opcmUtils() can be individually tested for mismatch.
    function test_verifyOpcmUtilsConsistency_eachComponent_reverts() public {
        // Coverage changes bytecode and causes failures, skip.
        skipIfCoverage();

        // Only run for OPCM V2
        skipIfDevFeatureDisabled(DevFeatures.OPCM_V2);

        // Get the property references (which include the component addresses)
        VerifyOPCM.OpcmContractRef[] memory propRefs = harness.getOpcmPropertyRefs(opcm);

        // Test each OPCM component individually (only those that actually have opcmUtils())
        address differentUtils = address(0x9999999999999999999999999999999999999999);

        uint256 componentsWithUtilsTested = 0;
        for (uint256 i = 0; i < propRefs.length; i++) {
            string memory field = propRefs[i].field;
            if (_hasOpcmUtils(field)) {
                // Mock this specific component to return a different address
                vm.mockCall(
                    propRefs[i].addr, abi.encodeCall(IOPContractsManagerV2.opcmUtils, ()), abi.encode(differentUtils)
                );

                // The consistency check should fail
                vm.expectRevert(VerifyOPCM.VerifyOPCM_OpcmUtilsMismatch.selector);
                harness.verifyOpcmUtilsConsistency(propRefs);

                // Clear the mock for next iteration
                vm.clearMockedCalls();
                componentsWithUtilsTested++;
            }
        }

        // Ensure we actually tested some components (currently: opcmV2, opcmMigrator)
        assertGt(componentsWithUtilsTested, 0, "Should have tested at least one component with opcmUtils");
    }

    function _isSuperDisputeGameContractRef(VerifyOPCM.OpcmContractRef memory ref) internal pure returns (bool) {
        return LibString.eq(ref.name, "SuperFaultDisputeGame") || LibString.eq(ref.name, "SuperPermissionedDisputeGame");
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

    /// @notice Helper function to check if a field represents an OPCM component that has opcmUtils().
    /// @param _field The field name to check.
    /// @return True if the field represents an OPCM component with opcmUtils(), false otherwise.
    function _hasOpcmUtils(string memory _field) internal pure returns (bool) {
        // Only opcmV2 and opcmMigrator have opcmUtils() via OPContractsManagerUtilsCaller
        return LibString.eq(_field, "opcmV2") || LibString.eq(_field, "opcmMigrator");
    }

    /// @notice Utility function to mock the first OPCM component's opcmUtils address.
    /// @param _propRefs Array of property references to search through.
    /// @param _mockAddress The address to mock the opcmUtils call to return.
    function _mockFirstOpcmUtilsComponent(
        VerifyOPCM.OpcmContractRef[] memory _propRefs,
        address _mockAddress
    )
        internal
    {
        for (uint256 i = 0; i < _propRefs.length; i++) {
            string memory field = _propRefs[i].field;
            // Check if this is an OPCM component that has opcmUtils()
            if (_hasOpcmUtils(field)) {
                vm.mockCall(
                    _propRefs[i].addr, abi.encodeCall(IOPContractsManagerV2.opcmUtils, ()), abi.encode(_mockAddress)
                );
                return;
            }
        }
    }

    /// @notice Tests that immutable variables are correctly verified in the OPCM contract.
    function test_verifyOpcmImmutableVariables_succeeds() public {
        // Coverage changes bytecode and causes failures, skip.
        skipIfCoverage();

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
    }

    /// @notice Tests that the script fails when OPCM immutable variables are invalid.
    /// We test this by setting expected addresses and mocking OPCM methods to return different addresses.
    function test_verifyOpcmImmutableVariables_mismatch_fails() public {
        // Coverage changes bytecode and causes failures, skip.
        skipIfCoverage();

        // If OPCM V2 is enabled because we do not use environment variables for OPCM V2.
        skipIfDevFeatureEnabled(DevFeatures.OPCM_V2);

        // Test that mocking each individual getter causes verification to fail
        _assertOnOpcmGetter(IOPContractsManager.superchainConfig.selector);
        _assertOnOpcmGetter(IOPContractsManager.protocolVersions.selector);
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

/// @title VerifyOPCM_verifyPortalDelays_Test
/// @notice Tests for the portal delay verification function.
contract VerifyOPCM_verifyPortalDelays_Test is VerifyOPCM_TestInit {
    function setUp() public override {
        super.setUp();
        vm.setEnv("EXPECTED_PROOF_MATURITY_DELAY_SECONDS", vm.toString(optimismPortal2.proofMaturityDelaySeconds()));
    }

    /// @notice Tests that portal delay verification succeeds with correct values.
    function test_verifyPortalDelays_matchingDelay_succeeds() public view {
        bool result = harness.verifyPortalDelays(optimismPortal2);
        assertTrue(result, "Portal delay verification should succeed");
    }

    /// @notice Tests that portal delay verification fails with wrong expected value.
    function test_verifyPortalDelays_mismatchedDelay_fails() public {
        // Mock the portal to return a different delay than expected.
        vm.mockCall(
            address(optimismPortal2),
            abi.encodeCall(IOptimismPortal2.proofMaturityDelaySeconds, ()),
            abi.encode(uint256(12345))
        );
        bool result = harness.verifyPortalDelays(optimismPortal2);
        assertFalse(result, "Portal delay verification should fail with wrong expected value");
    }
}

/// @title VerifyOPCM_verifyAnchorStateRegistryDelays_Test
/// @notice Tests for the anchor state registry delay verification function.
contract VerifyOPCM_verifyAnchorStateRegistryDelays_Test is VerifyOPCM_TestInit {
    function setUp() public override {
        super.setUp();
        vm.setEnv(
            "EXPECTED_DISPUTE_GAME_FINALITY_DELAY_SECONDS",
            vm.toString(anchorStateRegistry.disputeGameFinalityDelaySeconds())
        );
    }

    /// @notice Tests that ASR delay verification succeeds with correct values.
    function test_verifyAnchorStateRegistryDelays_matchingDelay_succeeds() public view {
        bool result = harness.verifyAnchorStateRegistryDelays(anchorStateRegistry);
        assertTrue(result, "ASR delay verification should succeed");
    }

    /// @notice Tests that ASR delay verification fails with wrong expected value.
    function test_verifyAnchorStateRegistryDelays_mismatchedDelay_fails() public {
        // Mock the ASR to return a different delay than expected.
        vm.mockCall(
            address(anchorStateRegistry),
            abi.encodeCall(IAnchorStateRegistry.disputeGameFinalityDelaySeconds, ()),
            abi.encode(uint256(99999))
        );
        bool result = harness.verifyAnchorStateRegistryDelays(anchorStateRegistry);
        assertFalse(result, "ASR delay verification should fail with wrong expected value");
    }
}

/// @title VerifyOPCM_verifyPreimageOracle_Test
/// @notice Tests for the PreimageOracle bytecode verification function.
contract VerifyOPCM_verifyPreimageOracle_Test is VerifyOPCM_TestInit {
    /// @notice Tests that PreimageOracle verification succeeds when bytecode matches.
    function test_verifyPreimageOracle_matchingBytecode_succeeds() public {
        skipIfCoverage();
        IMIPS64 mipsImpl = IMIPS64(opcm.implementations().mipsImpl);
        bool result = harness.verifyPreimageOracle(mipsImpl);
        assertTrue(result, "PreimageOracle verification should succeed");
    }

    /// @notice Tests that PreimageOracle verification fails when bytecode doesn't match.
    function test_verifyPreimageOracle_corruptedBytecode_fails() public {
        skipIfCoverage();
        IMIPS64 mipsImpl = IMIPS64(opcm.implementations().mipsImpl);
        address oracleAddr = address(mipsImpl.oracle());

        bytes memory corruptedCode = oracleAddr.code;
        if (corruptedCode.length > 100) {
            corruptedCode[100] = bytes1(uint8(corruptedCode[100]) ^ 0xFF);
        }
        vm.etch(oracleAddr, corruptedCode);

        bool result = harness.verifyPreimageOracle(mipsImpl);
        assertFalse(result, "PreimageOracle verification should fail with corrupted bytecode");
    }
}
