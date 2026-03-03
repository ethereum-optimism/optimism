// SPDX-License-Identifier: MIT
pragma solidity ^0.8.15;

// Foundry
import { Script } from "forge-std/Script.sol";
import { console2 as console } from "forge-std/console2.sol";
import { stdJson } from "forge-std/StdJson.sol";

// Libraries
import { Math } from "openzeppelin-contracts/contracts/utils/math/Math.sol";
import { LibString } from "@solady/utils/LibString.sol";
import { Process } from "scripts/libraries/Process.sol";
import { Config } from "scripts/libraries/Config.sol";
import { Bytes } from "src/libraries/Bytes.sol";
import { DevFeatures } from "src/libraries/DevFeatures.sol";
import { SemverComp } from "src/libraries/SemverComp.sol";
import { Constants } from "src/libraries/Constants.sol";

// Interfaces
import { IOPContractsManager } from "interfaces/L1/IOPContractsManager.sol";
import { IOptimismPortal2 } from "interfaces/L1/IOptimismPortal2.sol";
import { IAnchorStateRegistry } from "interfaces/dispute/IAnchorStateRegistry.sol";
import { IMIPS64 } from "interfaces/cannon/IMIPS64.sol";

/// @title VerifyOPCM
/// @notice Verifies the bytecode of an OPContractsManager instance and all associated blueprints
///         and implementations against locally built artifacts.
/// @dev SECURITY MODEL
///
///      This script verifies that deployed contracts match expected bytecode and configuration.
///      Understanding what this script can and cannot detect is critical for security.
///
///      Attacker Capabilities (what the attacker controls):
///      - Deployment of all contracts (OPCM, Container, StandardValidator, implementations)
///      - All constructor arguments and immutable values
///      - Contract deployment addresses
///
///      Trust Assumptions (what we assume is honest):
///      - Local artifacts are compiled from correct, audited source code
///      - Environment variables contain correct expected values from trusted sources
///      - Block explorer API returns authentic creation bytecode (for constructor verification)
///      - The RPC endpoint returns authentic on-chain bytecode and state
///
///      What This Script Verifies:
///      - Runtime bytecode matches local artifacts (ignoring immutable slots)
///      - Creation bytecode matches local artifacts (when constructor verification enabled)
///      - Security-critical immutable values (delays, addresses) match expected values
///      - PreimageOracle bytecode referenced by MIPS64 is correct
///      - StandardValidator configuration matches Container implementations
///
///      What This Script Does NOT Verify:
///      - Source code correctness (assumes artifacts are from audited code)
///      - Environment variable correctness (must be set from trusted governance/config)
///      - Proxy storage slot contents (only verifies implementation bytecode)
///      - Runtime behavior or logic correctness
contract VerifyOPCM is Script {
    using stdJson for string;

    /// @notice Thrown when the top-level verification fails.
    error VerifyOPCM_Failed();

    /// @notice Thrown when no properties are found in the OPCM.
    error VerifyOPCM_NoProperties();

    /// @notice Thrown when no implementations are found in the OPCM.
    error VerifyOPCM_NoImplementations();

    /// @notice Thrown when no blueprints are found in the OPCM.
    error VerifyOPCM_NoBlueprints();

    /// @notice Thrown when an unexpected part number is found in the blueprint.
    error VerifyOPCM_UnexpectedPart();

    /// @notice Thrown when an artifact file is empty.
    error VerifyOPCM_EmptyArtifactFile(string _artifactPath);

    /// @notice Thrown when contractsContainer addresses are not the same across all OPCM components.
    error VerifyOPCM_ContractsContainerMismatch();

    /// @notice Thrown when opcmUtils addresses are not the same across all OPCM components that have it.
    error VerifyOPCM_OpcmUtilsMismatch();

    /// @notice Thrown when the creation bytecode is not found in an artifact file.
    error VerifyOPCM_CreationBytecodeNotFound(string _artifactPath);

    /// @notice Thrown when the runtime bytecode is not found in an artifact file.
    error VerifyOPCM_RuntimeBytecodeNotFound(string _artifactPath);

    /// @notice Thrown when there are getter functions in the ABI that are not being checked.
    error VerifyOPCM_UnaccountedGetters(string[] _unaccountedGetters);

    /// @notice Thrown when the dev feature bitmap is not empty on mainnet.
    error VerifyOPCM_DevFeatureBitmapNotEmpty();

    /// @notice Thrown when a security-critical value doesn't match expected.
    error VerifyOPCM_SecurityCriticalValueMismatch(string name, uint256 expected, uint256 actual);

    /// @notice Thrown when a staticcall to a validator getter fails.
    error VerifyOPCM_ValidatorCallFailed(string sig);

    /// @notice Thrown when _findChar is called with a multi-character string.
    error VerifyOPCM_MustBeSingleChar();

    /// @notice Preamble used for blueprint contracts.
    bytes constant BLUEPRINT_PREAMBLE = hex"FE7100";

    /// @notice Maximum init code size for blueprints.
    uint256 constant MAX_INIT_CODE_SIZE = 23500;

    /// @notice Represents a contract name and its corresponding address.
    /// @param field Name of the field the address was extracted from.
    /// @param name  Name of the contract.
    /// @param addr  Address of the contract.
    struct OpcmContractRef {
        string field;
        string name;
        address addr;
        bool blueprint;
    }

    /// @notice Represents an immutable reference within bytecode.
    /// @param length Length of the immutable reference in bytes.
    /// @param offset Offset of the immutable reference within the bytecode.
    struct ImmutableRef {
        uint256 length;
        uint256 offset;
    }

    /// @notice Represents info loaded from a contract artifact JSON file.
    /// @param bytecode         The creation bytecode.
    /// @param deployedBytecode The runtime bytecode.
    /// @param immutableRefs    Array of immutable references found in the deployed bytecode.
    struct ArtifactInfo {
        bytes bytecode;
        bytes deployedBytecode;
        ImmutableRef[] immutableRefs;
    }

    /// @notice Maps OPCM field names (as strings) to an overriding contract name.
    mapping(string => string) internal fieldNameOverrides;

    /// @notice Maps contract names to an overriding source file name.
    mapping(string => string) internal sourceNameOverrides;

    /// @notice Maps expected getter function names to their verification method.
    /// Value can be either:
    /// - An environment variable name (e.g., "EXPECTED_SUPERCHAIN_CONFIG") for getters verified via env vars
    /// - "SKIP" for getters verified elsewhere in the verification process
    /// WARNING: Do NOT add new getters without understanding their verification method!
    mapping(string => string) internal expectedGetters;

    /// @notice Maps StandardValidator getter names to their verification method.
    /// Value can be:
    /// - "CONTAINER_IMPL" - verify against Container's implementations struct
    /// - "ENV:ADDRESS:<VAR_NAME>" - verify against environment variable (address)
    /// - "ENV:UINT256:<VAR_NAME>" - verify against environment variable (uint256)
    /// - "ZERO_ON_MAINNET" - verify is zero/empty on mainnet
    /// - "SKIP" - explicitly skip (e.g., version)
    mapping(string => string) internal validatorGetterChecks;

    /// @notice Setup flag.
    bool internal ready;

    /// @notice Returns whether to skip security-critical value checks.
    ///         Public to allow tests to mock via vm.mockCall.
    function skipSecurityValueChecks() public view virtual returns (bool) {
        // nosemgrep: sol-style-vm-env-only-in-config-sol
        return vm.envOr("SKIP_SECURITY_VALUE_CHECKS", false);
    }

    /// @notice Populates override mappings.
    function setUp() public {
        // Overrides for situations where field names do not cleanly map to contract names.
        fieldNameOverrides["optimismPortalImpl"] = "OptimismPortal2";
        fieldNameOverrides["optimismPortalInteropImpl"] = "OptimismPortalInterop";
        fieldNameOverrides["mipsImpl"] = "MIPS64";
        fieldNameOverrides["ethLockboxImpl"] = "ETHLockbox";
        fieldNameOverrides["faultDisputeGameImpl"] = "FaultDisputeGame";
        fieldNameOverrides["permissionedDisputeGameImpl"] = "PermissionedDisputeGame";
        fieldNameOverrides["permissionlessDisputeGame1"] = "FaultDisputeGame";
        fieldNameOverrides["permissionlessDisputeGame2"] = "FaultDisputeGame";
        fieldNameOverrides["permissionedDisputeGame1"] = "PermissionedDisputeGame";
        fieldNameOverrides["permissionedDisputeGame2"] = "PermissionedDisputeGame";
        fieldNameOverrides["superPermissionlessDisputeGame1"] = "SuperFaultDisputeGame";
        fieldNameOverrides["superPermissionlessDisputeGame2"] = "SuperFaultDisputeGame";
        fieldNameOverrides["superPermissionedDisputeGame1"] = "SuperPermissionedDisputeGame";
        fieldNameOverrides["superPermissionedDisputeGame2"] = "SuperPermissionedDisputeGame";
        fieldNameOverrides["opcmGameTypeAdder"] = "OPContractsManagerGameTypeAdder";
        fieldNameOverrides["opcmDeployer"] = "OPContractsManagerDeployer";
        fieldNameOverrides["opcmMigrator"] = "OPContractsManagerMigrator";
        fieldNameOverrides["opcmUpgrader"] = "OPContractsManagerUpgrader";
        fieldNameOverrides["opcmInteropMigrator"] = "OPContractsManagerInteropMigrator";
        fieldNameOverrides["opcmStandardValidator"] = "OPContractsManagerStandardValidator";

        // Since both OPCM V1 and V2 have contractsContainer var and they point to different contract file names,
        // in the code logic, we rename any occurrences of it to "contractsContainerV1" or "contractsContainerV2" before
        // using it to read the mapping.
        fieldNameOverrides["contractsContainerV1"] = "OPContractsManagerContractsContainer";
        fieldNameOverrides["contractsContainerV2"] = "OPContractsManagerContainer";

        // OPCM V2 Specific field name overrides.
        fieldNameOverrides["standardValidator"] = "OPContractsManagerStandardValidator";
        fieldNameOverrides["storageSetterImpl"] = "StorageSetter";
        fieldNameOverrides["opcmV2"] = "OPContractsManagerV2";
        fieldNameOverrides["opcmUtils"] = "OPContractsManagerUtils";

        // Overrides for situations where contracts have differently named source files.
        sourceNameOverrides["OPContractsManagerGameTypeAdder"] = "OPContractsManager";
        sourceNameOverrides["OPContractsManagerDeployer"] = "OPContractsManager";
        sourceNameOverrides["OPContractsManagerUpgrader"] = "OPContractsManager";
        sourceNameOverrides["OPContractsManagerInteropMigrator"] = "OPContractsManager";
        sourceNameOverrides["OPContractsManagerContractsContainer"] = "OPContractsManager";

        // Expected getter functions and their verification methods.
        // CRITICAL: Any getter in the ABI that's not in this list will cause verification to fail.
        // NEVER add a getter without understanding HOW it's being verified!

        // Getters verified via bytecode comparison (blueprints/implementations contain addresses)
        expectedGetters["blueprints"] = "SKIP"; // Verified via bytecode comparison of blueprint contracts
        expectedGetters["implementations"] = "SKIP"; // Verified via bytecode comparison of implementation contracts

        // Getters verified via environment variables in _verifyOpcmImmutableVariables()
        expectedGetters["protocolVersions"] = "EXPECTED_PROTOCOL_VERSIONS";
        expectedGetters["superchainConfig"] = "EXPECTED_SUPERCHAIN_CONFIG";

        // Getters for OPCM sub-contracts (addresses verified via bytecode comparison)
        expectedGetters["opcmDeployer"] = "SKIP"; // Address verified via bytecode comparison
        expectedGetters["opcmGameTypeAdder"] = "SKIP"; // Address verified via bytecode comparison
        expectedGetters["opcmInteropMigrator"] = "SKIP"; // Address verified via bytecode comparison
        expectedGetters["opcmMigrator"] = "SKIP"; // Address verified via bytecode comparison
        expectedGetters["opcmStandardValidator"] = "SKIP"; // Address verified via bytecode comparison
        expectedGetters["opcmUpgrader"] = "SKIP"; // Address verified via bytecode comparison

        // OPCM V2 Specific expected getters overrides
        expectedGetters["opcmV2"] = "SKIP"; // Address verified via bytecode comparison
        expectedGetters["opcmUtils"] = "SKIP"; // Address verified via bytecode comparison
        expectedGetters["contractsContainer"] = "SKIP"; // Address verified via bytecode comparison

        // Getters that don't need any sort of verification
        expectedGetters["devFeatureBitmap"] = "SKIP";
        expectedGetters["isDevFeatureEnabled"] = "SKIP";
        expectedGetters["version"] = "SKIP";

        // StandardValidator getter verification methods
        // Implementation addresses - verify against Container
        validatorGetterChecks["l1ERC721BridgeImpl"] = "CONTAINER_IMPL";
        validatorGetterChecks["optimismPortalImpl"] = "CONTAINER_IMPL";
        validatorGetterChecks["optimismPortalInteropImpl"] = "CONTAINER_IMPL";
        validatorGetterChecks["ethLockboxImpl"] = "CONTAINER_IMPL";
        validatorGetterChecks["systemConfigImpl"] = "CONTAINER_IMPL";
        validatorGetterChecks["optimismMintableERC20FactoryImpl"] = "CONTAINER_IMPL";
        validatorGetterChecks["l1CrossDomainMessengerImpl"] = "CONTAINER_IMPL";
        validatorGetterChecks["l1StandardBridgeImpl"] = "CONTAINER_IMPL";
        validatorGetterChecks["disputeGameFactoryImpl"] = "CONTAINER_IMPL";
        validatorGetterChecks["anchorStateRegistryImpl"] = "CONTAINER_IMPL";
        validatorGetterChecks["delayedWETHImpl"] = "CONTAINER_IMPL";
        validatorGetterChecks["mipsImpl"] = "CONTAINER_IMPL";
        validatorGetterChecks["faultDisputeGameImpl"] = "CONTAINER_IMPL";
        validatorGetterChecks["permissionedDisputeGameImpl"] = "CONTAINER_IMPL";

        // Verify against env vars
        validatorGetterChecks["superchainConfig"] = "ENV:ADDRESS:EXPECTED_SUPERCHAIN_CONFIG";
        validatorGetterChecks["l1PAOMultisig"] = "ENV:ADDRESS:EXPECTED_L1_PAO_MULTISIG";
        validatorGetterChecks["challenger"] = "ENV:ADDRESS:EXPECTED_CHALLENGER";
        validatorGetterChecks["withdrawalDelaySeconds"] = "ENV:UINT256:EXPECTED_WITHDRAWAL_DELAY_SECONDS";

        // Must be empty on mainnet
        validatorGetterChecks["devFeatureBitmap"] = "ZERO_ON_MAINNET";

        // Skip - no security relevance or verified elsewhere
        validatorGetterChecks["version"] = "SKIP";
        validatorGetterChecks["preimageOracleVersion"] = "SKIP";

        // Mark as ready.
        ready = true;
    }

    /// @notice Entry point for the script when run via `forge script`, reads the OPCM address from
    ///         the environment variable OPCM_ADDRESS. Use run(address) if you want to specify the
    ///         address as an argument instead. Running in this mode will not allow you to skip
    ///         constructor verification.
    function run() external {
        // nosemgrep: sol-style-vm-env-only-in-config-sol
        run(vm.envAddress("OPCM_ADDRESS"), false);
    }

    /// @notice Entry point for the script when trying to verify a single contract by name.
    /// @param _name Name of the contract to verify.
    /// @param _addr Address of the contract to verify.
    /// @param _skipConstructorVerification Whether to skip constructor verification.
    function runSingle(string memory _name, address _addr, bool _skipConstructorVerification) public {
        // This function is used as part of the release checklist to verify new contracts.
        // Rather than requiring an opcm input parameter, just pass in an empty reference
        // as we really only need this for features that are in development.
        IOPContractsManager emptyOpcm = IOPContractsManager(address(0));
        _verifyOpcmContractRef(
            emptyOpcm,
            OpcmContractRef({ field: _name, name: _name, addr: _addr, blueprint: false }),
            _skipConstructorVerification
        );
    }

    /// @notice Main verification logic.
    /// @param _opcmAddress Address of the OPContractsManager contract to verify.
    /// @param _skipConstructorVerification Whether to skip constructor verification.
    function run(address _opcmAddress, bool _skipConstructorVerification) public {
        // Make sure the setup function has been called.
        if (!ready) {
            setUp();
        }

        // Log a warning if constructor verification is being skipped.
        if (_skipConstructorVerification) {
            console.log("WARNING: Constructor verification is being skipped");
            console.log("         ONLY to be used in test environments");
            console.log("         Do NOT do this in production");
        }

        // Fetch Implementations & Blueprints from OPCM
        IOPContractsManager opcm = IOPContractsManager(_opcmAddress);

        // Validate that all ABI getters are accounted for.
        _validateAllGettersAccounted();

        // Validate that the dev feature bitmap is empty on mainnet.
        _validateDevFeatureBitmap(opcm);

        // Collect all the references.
        OpcmContractRef[] memory refs = _collectOpcmContractRefs(opcm);

        // Verify each reference.
        bool success = true;
        for (uint256 i = 0; i < refs.length; i++) {
            success = _verifyOpcmContractRef(opcm, refs[i], _skipConstructorVerification) && success;
        }

        // Final Result
        console.log();
        if (success) {
            console.log("Overall Verification Status: SUCCESS");
        } else {
            console.log("Overall Verification Status: FAILED");
            revert VerifyOPCM_Failed();
        }
    }

    /// @notice Collects all the references from the OPCM contract.
    /// @param _opcm The live OPCM contract.
    /// @return Array of OpcmContractRef structs containing contract names/addresses.
    function _collectOpcmContractRefs(IOPContractsManager _opcm) internal returns (OpcmContractRef[] memory) {
        // Collect property references.
        OpcmContractRef[] memory propRefs = _getOpcmPropertyRefs(_opcm);
        if (propRefs.length == 0) {
            revert VerifyOPCM_NoProperties();
        }

        // Verify that all component contracts have the same contractsContainer address.
        _verifyContractsContainerConsistency(propRefs);

        // Verify that all component contracts that have opcmUtils() have the same address.
        _verifyOpcmUtilsConsistency(propRefs);

        // Get the ContractsContainer address from the first component (they're all the same)
        address contractsContainerAddr = address(0);
        for (uint256 i = 0; i < propRefs.length; i++) {
            string memory field = propRefs[i].field;
            if (_hasContractsContainer(field)) {
                contractsContainerAddr = _getContractsContainerAddress(propRefs[i].addr);
                break;
            }
        }

        // Collect implementation references.
        OpcmContractRef[] memory implRefs = _getOpcmContractRefs(_opcm, "implementations", false);
        if (implRefs.length == 0) {
            revert VerifyOPCM_NoImplementations();
        }

        // Collect blueprint references.
        OpcmContractRef[] memory bpRefs = _getOpcmContractRefs(_opcm, "blueprints", true);
        if (bpRefs.length == 0) {
            revert VerifyOPCM_NoBlueprints();
        }

        // Create a single array to join everything together.
        uint256 extraRefs = 2; // OPCM + ContractsContainer
        OpcmContractRef[] memory refs =
            new OpcmContractRef[](propRefs.length + implRefs.length + bpRefs.length + extraRefs);

        // References for OPCM and linked contracts.
        refs[0] = OpcmContractRef({ field: "opcm", name: _opcmContractName(), addr: address(_opcm), blueprint: false });
        refs[1] = OpcmContractRef({
            field: "contractsContainer",
            name: _isOPCMV2() ? "OPContractsManagerContainer" : "OPContractsManagerContractsContainer",
            addr: contractsContainerAddr,
            blueprint: false
        });

        // Add the property references.
        for (uint256 i = 0; i < propRefs.length; i++) {
            refs[i + extraRefs] = propRefs[i];
        }

        // Add the implementation references.
        for (uint256 i = 0; i < implRefs.length; i++) {
            refs[i + extraRefs + propRefs.length] = implRefs[i];
        }

        // Add the blueprint references.
        for (uint256 i = 0; i < bpRefs.length; i++) {
            refs[i + extraRefs + propRefs.length + implRefs.length] = bpRefs[i];
        }

        // Return the combined references.
        return refs;
    }

    /// @notice Verifies that all OPCM component contracts have the same contractsContainer address.
    /// @param _propRefs Array of property references containing component addresses.
    function _verifyContractsContainerConsistency(OpcmContractRef[] memory _propRefs) internal view {
        // Process components that have contractsContainer(), validate addresses, and verify consistency
        OpcmContractRef[] memory components = new OpcmContractRef[](_propRefs.length);
        address[] memory containerAddresses = new address[](_propRefs.length);
        uint256 componentCount = 0;
        address expectedContainer = address(0);

        for (uint256 i = 0; i < _propRefs.length; i++) {
            OpcmContractRef memory propRef = _propRefs[i];

            if (!_hasContractsContainer(propRef.field)) {
                continue;
            }

            components[componentCount] = propRef;
            address containerAddr = _getContractsContainerAddress(propRef.addr);

            if (containerAddr == address(0)) {
                console.log(string.concat("ERROR: Failed to retrieve contractsContainer address from ", propRef.field));
                revert VerifyOPCM_ContractsContainerMismatch();
            }

            containerAddresses[componentCount] = containerAddr;

            if (componentCount == 0) {
                expectedContainer = containerAddr;
            } else if (containerAddr != expectedContainer) {
                _logContainerAddressMismatch(components, containerAddresses, componentCount);
                revert VerifyOPCM_ContractsContainerMismatch();
            }

            componentCount++;
        }

        // Ensure we found at least one component
        if (componentCount == 0) {
            console.log("ERROR: No OPCM components found for contractsContainer verification");
            revert VerifyOPCM_ContractsContainerMismatch();
        }

        console.log(
            string.concat(
                "OK: All ", vm.toString(componentCount), " components have the same contractsContainer address"
            )
        );
        console.log(string.concat("  contractsContainer: ", vm.toString(expectedContainer)));
    }

    /// @notice Logs container address mismatch details for debugging.
    /// @param _components Array of components found so far.
    /// @param _containerAddresses Array of container addresses for each component.
    /// @param _componentCount Number of components processed.
    function _logContainerAddressMismatch(
        OpcmContractRef[] memory _components,
        address[] memory _containerAddresses,
        uint256 _componentCount
    )
        internal
        pure
    {
        console.log("ERROR: contractsContainer addresses are not consistent across all components");
        for (uint256 j = 0; j <= _componentCount; j++) {
            console.log(string.concat("  ", _components[j].field, ": ", vm.toString(_containerAddresses[j])));
        }
    }

    /// @notice Gets the contractsContainer address from a contract.
    /// @param _contract The contract address to query.
    /// @return The contractsContainer address.
    function _getContractsContainerAddress(address _contract) internal view returns (address) {
        // Call the contractsContainer() function on the contract.
        // nosemgrep: sol-style-use-abi-encodecall
        (bool success, bytes memory returnData) = _contract.staticcall(abi.encodeWithSignature("contractsContainer()"));
        if (!success) {
            console.log(
                string.concat(
                    "[FAIL] ERROR: Failed to call contractsContainer() function on contract ", vm.toString(_contract)
                )
            );
            return address(0);
        }
        return abi.decode(returnData, (address));
    }

    /// @notice Checks if a field name represents an OPCM component contract that has opcmUtils().
    /// @param _field The field name to check.
    /// @return True if the field represents an OPCM component with opcmUtils(), false otherwise.
    function _hasOpcmUtils(string memory _field) internal pure returns (bool) {
        // Only opcmV2 and opcmMigrator have opcmUtils() via OPContractsManagerUtilsCaller
        return LibString.eq(_field, "opcmV2") || LibString.eq(_field, "opcmMigrator");
    }

    /// @notice Gets the opcmUtils address from a contract.
    /// @param _contract The contract address to query.
    /// @return The opcmUtils address.
    function _getOpcmUtilsAddress(address _contract) internal view returns (address) {
        // Call the opcmUtils() function on the contract.
        // nosemgrep: sol-style-use-abi-encodecall
        (bool success, bytes memory returnData) = _contract.staticcall(abi.encodeWithSignature("opcmUtils()"));
        if (!success) {
            console.log(
                string.concat("[FAIL] ERROR: Failed to call opcmUtils() function on contract ", vm.toString(_contract))
            );
            return address(0);
        }
        return abi.decode(returnData, (address));
    }

    /// @notice Verifies that all OPCM component contracts that have opcmUtils() have the same address.
    /// @param _propRefs Array of property references containing component addresses.
    function _verifyOpcmUtilsConsistency(OpcmContractRef[] memory _propRefs) internal view {
        // Process components that have opcmUtils(), validate addresses, and verify consistency
        OpcmContractRef[] memory components = new OpcmContractRef[](_propRefs.length);
        address[] memory utilsAddresses = new address[](_propRefs.length);
        uint256 componentCount = 0;
        address expectedUtils = address(0);

        for (uint256 i = 0; i < _propRefs.length; i++) {
            OpcmContractRef memory propRef = _propRefs[i];

            if (!_hasOpcmUtils(propRef.field)) {
                continue;
            }

            components[componentCount] = propRef;
            address utilsAddr = _getOpcmUtilsAddress(propRef.addr);

            if (utilsAddr == address(0)) {
                console.log(string.concat("ERROR: Failed to retrieve opcmUtils address from ", propRef.field));
                revert VerifyOPCM_OpcmUtilsMismatch();
            }

            utilsAddresses[componentCount] = utilsAddr;

            if (componentCount == 0) {
                expectedUtils = utilsAddr;
            } else if (utilsAddr != expectedUtils) {
                console.log("ERROR: opcmUtils addresses are not consistent across all components");
                for (uint256 j = 0; j <= componentCount; j++) {
                    console.log(string.concat("  ", components[j].field, ": ", vm.toString(utilsAddresses[j])));
                }
                revert VerifyOPCM_OpcmUtilsMismatch();
            }

            componentCount++;
        }

        // Ensure we found at least one component
        if (componentCount == 0) {
            console.log("OK: No OPCM components with opcmUtils() found (skipping verification)");
            return;
        }

        console.log(
            string.concat("OK: All ", vm.toString(componentCount), " components have the same opcmUtils address")
        );
        console.log(string.concat("  opcmUtils: ", vm.toString(expectedUtils)));
    }

    /// @notice Verifies a single OPCM contract reference (implementation or bytecode).
    /// @param _opcm The OPCM contract that contains the target contract reference.
    /// @param _target The target contract reference to verify.
    /// @param _skipConstructorVerification Whether to skip constructor verification.
    /// @return True if the contract reference is verified, false otherwise.
    function _verifyOpcmContractRef(
        IOPContractsManager _opcm,
        OpcmContractRef memory _target,
        bool _skipConstructorVerification
    )
        internal
        returns (bool)
    {
        bool success = true;

        console.log();
        console.log(string.concat("Checking Contract: ", _target.field));
        console.log(string.concat("  Type: ", _target.blueprint ? "Blueprint" : "Implementation"));
        console.log(string.concat("  Contract: ", _target.name));
        console.log(string.concat("  Address: ", vm.toString(_target.addr)));

        // Build the expected path to the artifact file.
        string memory artifactPath = _buildArtifactPath(_target.name);
        console.log(string.concat("  Expected Runtime Artifact: ", artifactPath));

        // Check if this is a Super dispute game that should be skipped
        if (_isSuperDisputeGameImplementation(_target.name)) {
            if (!_isSuperDisputeGamesEnabled(_opcm)) {
                if (_target.addr == address(0)) {
                    console.log("[SKIP] Super game not deployed (feature disabled)");
                    return true; // Consider this "verified" when feature is off
                } else {
                    console.log("[FAIL] ERROR: Super game deployed but feature disabled");
                    success = false;
                }
            }
            // If feature is enabled, continue with normal verification
        }

        // Load artifact information (bytecode, immutable refs) for detailed comparison
        ArtifactInfo memory artifact = _loadArtifactInfo(artifactPath);

        // Grab the actual code.
        bytes memory actualCode = _target.addr.code;

        // Figure out expected code.
        bytes memory expectedCode;
        if (_target.blueprint) {
            // Determine which part of the blueprint this is using final digit as signifier.
            uint8 partNumber = 1;
            bytes memory fieldBytes = bytes(_target.field);
            if (fieldBytes.length > 0) {
                uint8 lastChar = uint8(fieldBytes[fieldBytes.length - 1]);
                if (lastChar >= uint8(bytes1("1")) && lastChar <= uint8(bytes1("9"))) {
                    partNumber = lastChar - uint8(bytes1("0"));
                }
            }

            // Split the creation code.
            bytes memory creationCodePart;
            if (partNumber == 1) {
                // First part: take initial MAX_INIT_CODE_SIZE bytes.
                creationCodePart =
                    Bytes.slice(artifact.bytecode, 0, Math.min(MAX_INIT_CODE_SIZE, artifact.bytecode.length));
            } else if (partNumber == 2) {
                // Second part: take remaining bytes.
                creationCodePart =
                    Bytes.slice(artifact.bytecode, MAX_INIT_CODE_SIZE, artifact.bytecode.length - MAX_INIT_CODE_SIZE);
            } else {
                // We don't support >2 parts for now, this is an explicit error.
                revert VerifyOPCM_UnexpectedPart();
            }

            // Create expected blueprint code for this part.
            expectedCode = abi.encodePacked(BLUEPRINT_PREAMBLE, creationCodePart);
        } else {
            expectedCode = artifact.deployedBytecode;
        }

        // Perform detailed bytecode comparison.
        success = _compareBytecode(actualCode, expectedCode, _target.name, artifact, !_target.blueprint) && success;

        // For implementations, verify security-critical values.
        if (!_target.blueprint) {
            success = _verifySecurityCriticalValues(_opcm, _target, artifact) && success;
        }

        // If requested and this is not a blueprint, we also need to check the creation code.
        if (!_target.blueprint && !_skipConstructorVerification) {
            // Get the creation code from the selected block explorer.
            bytes memory actualCreationCode = _getCreationCode(_target.addr);

            // Verify that the artifact bytecode is a prefix of the actual creation code and
            // extract any remaining bytes so we can verify the constructor arguments.
            if (Bytes.equal(Bytes.slice(actualCreationCode, 0, artifact.bytecode.length), artifact.bytecode)) {
                // Extract the constructor arguments.
                bytes memory constructorArgs = Bytes.slice(
                    actualCreationCode, artifact.bytecode.length, actualCreationCode.length - artifact.bytecode.length
                );

                // Make sure the constructor args are valid.
                if (_isValidConstructorArgs(_target.name, constructorArgs)) {
                    console.log(string.concat("[OK] Constructor arguments are valid"));
                } else {
                    console.log(string.concat("[FAIL] ERROR: Constructor arguments are invalid"));
                    success = false;
                }
            } else {
                console.log(string.concat("[FAIL] ERROR: Creation code mismatch for ", _target.name));
                success = false;
            }
        }

        // If this is the OPCM contract itself, verify the immutable variables as well.
        if (keccak256(bytes(_target.field)) == keccak256(bytes("opcm"))) {
            success = _verifyOpcmImmutableVariables(IOPContractsManager(_target.addr)) && success;
        }

        // Log final status for this field.
        if (success) {
            console.log(string.concat("Status: [OK] Verified ", _target.name));
        } else {
            console.log(string.concat("Status: [FAIL] Verification failed for ", _target.name));
        }

        return success;
    }

    /// @notice Gets the creation code for a given contract address from the configured block explorer.
    /// @param _addr The address of the contract to get the creation code for.
    /// @return The creation code of the contract.
    function _getCreationCode(address _addr) internal returns (bytes memory) {
        // Prepare the command to execute.
        string memory cmd;

        // Check which block explorer to use.
        if (LibString.eq(Config.blockExplorer(), "blockscout")) {
            console.log("  Fetching creation code from Blockscout...");
            cmd = string.concat(
                "curl -s '",
                Config.blockscoutApiUrl(),
                "/api?module=contract&action=getcontractcreation&contractaddresses=",
                vm.toString(_addr),
                "' | jq -r '.result[0].creationBytecode'"
            );
        } else {
            console.log("  Fetching creation code from Etherscan...");
            cmd = string.concat(
                "curl -s 'https://api.etherscan.io/v2/api?chainid=",
                vm.toString(block.chainid),
                "&module=contract&action=getcontractcreation&contractaddresses=",
                vm.toString(_addr),
                "&apikey=",
                Config.etherscanApiKey(),
                "' | jq -r '.result[0].creationBytecode'"
            );
        }

        // Execute the command.
        return bytes(Process.bash(cmd));
    }

    /// @notice Checks if super dispute games feature is enabled in the dev feature bitmap.
    /// @param _opcm The OPContractsManager to check.
    /// @return True if super dispute games are enabled.
    function _isSuperDisputeGamesEnabled(IOPContractsManager _opcm) internal view returns (bool) {
        bytes32 bitmap = _opcm.devFeatureBitmap();
        return DevFeatures.isDevFeatureEnabled(bitmap, DevFeatures.OPTIMISM_PORTAL_INTEROP);
    }

    /// @notice Checks if a contract is a Super dispute game implementation.
    /// @param _contractName The name to check.
    /// @return True if this is a V2 dispute game.
    function _isSuperDisputeGameImplementation(string memory _contractName) internal pure returns (bool) {
        return LibString.eq(_contractName, "SuperFaultDisputeGame")
            || LibString.eq(_contractName, "SuperPermissionedDisputeGame");
    }

    /// @notice Verifies that the immutable variables in the OPCM contract match expected values.
    /// @param _opcm The OPCM contract to verify immutable variables for.
    /// @return True if all immutable variables are verified, false otherwise.
    function _verifyOpcmImmutableVariables(IOPContractsManager _opcm) internal returns (bool) {
        console.log("  Verifying OPCM immutable variables...");

        bool success = true;

        // Get all OPCM getters and iterate over them
        // Note: We use the pattern `success = false; continue;` for failures to ensure
        // comprehensive reporting. Once success is false, it should never be reset to true.
        // This allows us to collect and report ALL issues in a single verification run.
        string[] memory allGetters = _getOpcmGetters();

        for (uint256 i = 0; i < allGetters.length; i++) {
            string memory functionName = allGetters[i];
            string memory verificationMethod = expectedGetters[functionName];

            // All getters must be accounted for in expectedGetters mapping
            if (bytes(verificationMethod).length == 0) {
                console.log("ERROR: Getter '%s' is not accounted for in expectedGetters mapping", functionName);
                success = false;
                continue;
            }

            // Skip getters that don't need env var verification
            if (keccak256(bytes(verificationMethod)) == keccak256(bytes("SKIP"))) {
                continue;
            }

            // Get expected address from environment variable
            // nosemgrep: sol-style-vm-env-only-in-config-sol
            address expectedAddress = vm.envAddress(verificationMethod);

            // Call the function to retrieve the actual address
            // nosemgrep: sol-style-use-abi-encodecall
            (bool callSuccess, bytes memory returnedData) =
                address(_opcm).staticcall(abi.encodeWithSignature(string.concat(functionName, "()")));

            if (!callSuccess) {
                console.log(string.concat("    [FAIL] ERROR: Failed to call ", functionName, "() function on OPCM."));
                success = false;
                continue;
            }

            // Decode as an address
            address actualAddress = abi.decode(returnedData, (address));

            // Log the comparison
            console.log(string.concat("    ", functionName, ": ", vm.toString(actualAddress)));
            console.log(string.concat("    expected: ", vm.toString(expectedAddress)));

            if (actualAddress != expectedAddress) {
                console.log(string.concat("    [FAIL] ERROR: ", functionName, " mismatch"));
                success = false;
            } else {
                console.log(string.concat("    [OK] ", functionName, " verified"));
            }
        }

        return success;
    }

    /// @notice Loads artifact info from a JSON file using Foundry's parsing capabilities.
    /// @param _artifactPath Path to the artifact JSON file.
    /// @return info The parsed artifact information containing bytecode and immutable references.
    function _loadArtifactInfo(string memory _artifactPath) internal view returns (ArtifactInfo memory) {
        // Read and parse the artifact file.
        string memory artifactJson = vm.readFile(_artifactPath);
        if (bytes(artifactJson).length == 0) {
            revert VerifyOPCM_EmptyArtifactFile(_artifactPath);
        }

        // Parse the creation bytecode.
        bytes memory bytecode = vm.parseBytes(artifactJson.readString(".bytecode.object"));
        if (bytecode.length == 0) {
            revert VerifyOPCM_CreationBytecodeNotFound(_artifactPath);
        }

        // Parse the runtime bytecode.
        bytes memory deployedBytecode = vm.parseBytes(artifactJson.readString(".deployedBytecode.object"));
        if (deployedBytecode.length == 0) {
            revert VerifyOPCM_RuntimeBytecodeNotFound(_artifactPath);
        }

        // Put together the artifact info struct.
        return ArtifactInfo({
            bytecode: bytecode,
            deployedBytecode: deployedBytecode,
            immutableRefs: _parseImmutableRefs(artifactJson)
        });
    }

    /// @notice Parses immutable references from the artifact JSON.
    /// @param _artifactJson Complete artifact JSON string.
    /// @return Array of parsed immutable reference structs {offset, length}.
    function _parseImmutableRefs(string memory _artifactJson) internal view returns (ImmutableRef[] memory) {
        // Check if immutableReferences exists, skip if not.
        if (!vm.keyExistsJson(_artifactJson, ".deployedBytecode.immutableReferences")) {
            return new ImmutableRef[](0);
        }

        // Grab all keys (AST node IDs) from the immutableReferences object.
        string[] memory keys = vm.parseJsonKeys(_artifactJson, ".deployedBytecode.immutableReferences");
        if (keys.length == 0) {
            return new ImmutableRef[](0);
        }

        // Count the total number of individual references across all keys.
        uint256 totalRefs = 0;
        for (uint256 i = 0; i < keys.length; i++) {
            string memory key = keys[i];
            string memory refsPath = string.concat(".deployedBytecode.immutableReferences.", key);
            ImmutableRef[] memory positions = abi.decode(vm.parseJson(_artifactJson, refsPath), (ImmutableRef[]));
            totalRefs += positions.length;
        }

        // Allocate the final array to hold all references.
        ImmutableRef[] memory refs = new ImmutableRef[](totalRefs);
        uint256 refIdx = 0;

        // Populate the final array with references from each key.
        for (uint256 i = 0; i < keys.length; i++) {
            string memory key = keys[i];
            string memory refsPath = string.concat(".deployedBytecode.immutableReferences.", key);
            ImmutableRef[] memory positions = abi.decode(vm.parseJson(_artifactJson, refsPath), (ImmutableRef[]));
            for (uint256 j = 0; j < positions.length; j++) {
                refs[refIdx++] = positions[j];
            }
        }

        return refs;
    }

    /// @notice Compares two bytecode arrays for differences.
    /// @param _actual The actual bytecode obtained from the chain.
    /// @param _expected The expected bytecode from the local artifact.
    /// @param _contractName The name of the contract being compared (for logging).
    /// @param _artifact Additional artifact info (used for immutable reference checking).
    /// @param _allowImmutables True if immutables are allowed to be different, false otherwise.
    /// @return True if bytecodes match exactly or if differences only occur within known immutables.
    function _compareBytecode(
        bytes memory _actual,
        bytes memory _expected,
        string memory _contractName,
        ArtifactInfo memory _artifact,
        bool _allowImmutables
    )
        internal
        pure
        returns (bool)
    {
        // Basic length check
        if (_actual.length != _expected.length) {
            console.log(string.concat("[FAIL] ERROR: Bytecode length mismatch for ", _contractName));
            console.log(string.concat("  Expected length: ", vm.toString(_expected.length)));
            console.log(string.concat("  Actual length:   ", vm.toString(_actual.length)));
            return false;
        }

        // Simplified logic, compare each byte individually, check if that difference falls within
        // an immutable range (if immutables are allowed) or if it's a code difference.
        for (uint256 i = 0; i < _actual.length; i++) {
            if (_actual[i] != _expected[i] && (!_allowImmutables || !_posInsideImmutable(i, _artifact))) {
                console.log(string.concat("[FAIL] ERROR: Bytecode difference found for ", _contractName));
                console.log(string.concat("  Offset: ", vm.toString(i)));
                console.log(string.concat("  Expected: ", vm.toString(_expected[i])));
                console.log(string.concat("  Actual:   ", vm.toString(_actual[i])));
                return false;
            }
        }

        // If we're here, the bytecode is identical.
        console.log("Status: [OK] Exact Match");
        return true;
    }

    /// @notice Uses the OPContractsManager ABI JSON and the live OPCM contract to extract a list
    ///         of contract names and their corresponding addresses for the various immutable
    ///         references to other OPCM contracts.
    /// @param _opcm The live OPCM contract.
    /// @return Array of OpcmContractRef structs containing contract names/addresses.
    function _getOpcmPropertyRefs(IOPContractsManager _opcm) internal returns (OpcmContractRef[] memory) {
        // Find all functions that start with "opcm".
        string[] memory functionNames = abi.decode(
            vm.parseJson(
                Process.bash(
                    string.concat(
                        "jq -r '[.abi[] | select(.name? and (.name | type == \"string\") and (.name | startswith(\"opcm\"))) | .name]' ",
                        _buildArtifactPath(_opcmContractName())
                    )
                )
            ),
            (string[])
        );

        // For each of these, turn into a contract reference.
        OpcmContractRef[] memory refs = new OpcmContractRef[](functionNames.length);
        for (uint256 i = 0; i < functionNames.length; i++) {
            // Get the function name.
            string memory functionName = functionNames[i];

            // Call the function to retrieve the encoded address.
            // nosemgrep: sol-style-use-abi-encodecall
            (bool callSuccess, bytes memory returnedData) =
                address(_opcm).staticcall(abi.encodeWithSignature(string.concat(functionName, "()")));
            if (!callSuccess) {
                console.log(string.concat("[FAIL] ERROR: Failed to call ", functionName, "() function on OPCM."));
                return new OpcmContractRef[](0);
            }

            // Decode as an address.
            address implAddress = abi.decode(returnedData, (address));

            // Add to the list.
            string memory contractName = _getContractNameFromFieldName(functionName);
            refs[i] = OpcmContractRef({ field: functionName, name: contractName, addr: implAddress, blueprint: false });
        }

        // Return the results.
        return refs;
    }

    /// @notice Uses the OPContractsManager ABI JSON and the live OPCM contract to extract a list
    ///         of contract names and their corresponding addresses for a given property/struct on
    ///         the OPCM contract.
    /// @param _opcm The live OPCM contract.
    /// @param _property The property/struct to extract contract names and addresses from.
    /// @param _blueprint Whether this is a blueprint or an implementation.
    /// @return Array of OpcmContractRef structs containing contract names/addresses.
    function _getOpcmContractRefs(
        IOPContractsManager _opcm,
        string memory _property,
        bool _blueprint
    )
        internal
        returns (OpcmContractRef[] memory)
    {
        // Use jq to grab the field names from the ABI.
        string[] memory fieldNames = abi.decode(
            vm.parseJson(
                Process.bash(
                    string.concat(
                        "jq -r '[.abi[] | select(.name == \"",
                        _property,
                        "\") | .outputs[0].components[].name]' ",
                        _buildArtifactPath(_opcmContractName())
                    )
                )
            ),
            (string[])
        );

        // Call the corresponding function on the OPCM contract.
        // nosemgrep: sol-style-use-abi-encodecall
        (bool callSuccess, bytes memory returnedData) =
            address(_opcm).staticcall(abi.encodeWithSignature(string.concat(_property, "()")));
        if (!callSuccess) {
            console.log(string.concat("[FAIL] ERROR: Failed to call ", _property, "() function on OPCM."));
            return new OpcmContractRef[](0);
        }

        // Expected length check: numFields * 32 bytes/address.
        uint256 expectedDataLength = fieldNames.length * 32;
        if (returnedData.length != expectedDataLength) {
            console.log(string.concat("[FAIL] ERROR: Returned data length mismatch from ", _property, "() call."));
            console.log(string.concat("  Expected length: ", vm.toString(expectedDataLength)));
            console.log(string.concat("  Actual length:   ", vm.toString(returnedData.length)));
            return new OpcmContractRef[](0);
        }

        // Extract the addresses from the returned data.
        OpcmContractRef[] memory opcmContractRefs = new OpcmContractRef[](fieldNames.length);
        for (uint256 i = 0; i < fieldNames.length; i++) {
            string memory fieldName = fieldNames[i];
            uint256 offset = i * 32;
            address implAddress = abi.decode(Bytes.slice(returnedData, offset, 32), (address));
            string memory contractName = _getContractNameFromFieldName(fieldName);
            opcmContractRefs[i] =
                OpcmContractRef({ field: fieldName, name: contractName, addr: implAddress, blueprint: _blueprint });
        }

        // Return the extracted addresses.
        return opcmContractRefs;
    }

    /// @notice Converts an OPCM field name to a contract name. Not 100% reliable, so use overrides
    ///         if necessary. Works most of the time though.
    /// @param _fieldName The field name to convert.
    /// @return The contract name.
    function _getContractNameFromFieldName(string memory _fieldName) internal view returns (string memory) {
        if (LibString.eq(_fieldName, "contractsContainer")) {
            _fieldName = _isOPCMV2() ? "contractsContainerV2" : "contractsContainerV1";
        }

        // Check for an explicit override
        string memory overrideName = fieldNameOverrides[_fieldName];
        if (bytes(overrideName).length > 0) {
            return overrideName;
        }

        // Make a copy of the field name.
        string memory fieldName = LibString.slice(_fieldName, 0, bytes(_fieldName).length);

        // Uppercase the first character
        bytes memory fieldBytes = bytes(fieldName);
        fieldBytes[0] = bytes1(uint8(bytes1("A")) + uint8(fieldBytes[0]) - uint8(bytes1("a")));

        // If it ends in impl, strip that.
        if (LibString.endsWith(_fieldName, "Impl")) {
            fieldBytes = Bytes.slice(fieldBytes, 0, fieldBytes.length - 4);
        }

        // Return the field name with the first character uppercase
        return string(fieldBytes);
    }

    /// @notice Checks if a position is inside an immutable reference.
    /// @param _pos The position to check.
    /// @param _artifact The artifact info.
    /// @return True if the position is inside an immutable reference, false otherwise.
    function _posInsideImmutable(uint256 _pos, ArtifactInfo memory _artifact) internal pure returns (bool) {
        for (uint256 i = 0; i < _artifact.immutableRefs.length; i++) {
            ImmutableRef memory ref = _artifact.immutableRefs[i];
            if (_pos >= ref.offset && _pos < ref.offset + ref.length) {
                return true;
            }
        }
        return false;
    }

    /// @notice Checks if the constructor args that came back from Etherscan are valid for this
    ///         contract. Essentially decodes and then re-encodes the same arguments to make sure
    ///         they parse correctly for the provided constructor ABI and there's no extra data.
    /// @param _contractName The name of the contract.
    /// @param _constructorArgs The constructor arguments to check.
    /// @return True if the constructor arguments are valid, false otherwise.
    function _isValidConstructorArgs(
        string memory _contractName,
        bytes memory _constructorArgs
    )
        internal
        returns (bool)
    {
        // Grab the constructor ABI types.
        string memory types = Process.bash(
            string.concat(
                "forge inspect ",
                _contractName,
                " abi --json | jq -r '.[] | select(.type == \"constructor\") | .inputs | map(if .type == \"tuple\" then \"(\" + (.components | map(.type) | join(\",\")) + \")\" else .type end) | join(\",\")'"
            )
        );

        // Decode, then re-encode the same args and make sure they match the original input.
        bytes memory encodedArgs = bytes(
            Process.bash(
                string.concat(
                    "cast abi-encode \"constructor(",
                    types,
                    ")\" ",
                    "$(cast decode-abi --input \"constructor(",
                    types,
                    ")\" ",
                    vm.toString(_constructorArgs),
                    " --json | jq -r 'map(if type == \"string\" and startswith(\"(\") then gsub(\", \"; \",\") else . end) | join(\" \")')"
                )
            )
        );

        // Compare with original input.
        return Bytes.equal(_constructorArgs, encodedArgs);
    }

    /// @notice Constructs the expected path to Foundry artifact JSON file based on contract name.
    /// @param _contractName The simple contract name (e.g., "SystemConfig", "FaultDisputeGame").
    /// @return Path to the artifact file.
    function _buildArtifactPath(string memory _contractName) internal view returns (string memory) {
        // Potentially need to override the source name if multiple contracts are defined in the same file.
        string memory sourceName = _contractName;
        if (bytes(sourceNameOverrides[_contractName]).length > 0) {
            sourceName = sourceNameOverrides[_contractName];
        }

        // Return computed path, relative to the contracts-bedrock directory.
        return string.concat("forge-artifacts/", sourceName, ".sol/", _contractName, ".json");
    }

    /// @notice Checks if a field name represents an OPCM component contract that has contractsContainer().
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

    /// @notice Gets all OPCM getter function names from the ABI.
    /// @return Array of getter function names found in the OPContractsManager ABI.
    function _getOpcmGetters() internal returns (string[] memory) {
        return abi.decode(
            vm.parseJson(
                Process.bash(
                    string.concat(
                        "jq -r '[.abi[] | select(.type == \"function\" and .stateMutability == \"view\" and (.inputs | length) == 0) | .name]' ",
                        _buildArtifactPath(_opcmContractName())
                    )
                )
            ),
            (string[])
        );
    }

    /// @notice Validates that the dev feature bitmap is empty on mainnet.
    /// @param _opcm The OPCM contract.
    function _validateDevFeatureBitmap(IOPContractsManager _opcm) internal view {
        // Get the dev feature bitmap.
        bytes32 devFeatureBitmap = _opcm.devFeatureBitmap();

        // Check if we're in a testing environment.
        bool isTestingEnvironment = address(0xbeefcafe).code.length > 0;

        // Check if any dev features are enabled.
        if (block.chainid == 1 && !isTestingEnvironment && devFeatureBitmap != bytes32(0)) {
            revert VerifyOPCM_DevFeatureBitmapNotEmpty();
        }
    }

    /// @notice Validates that all getter functions in the OPContractsManager ABI are accounted for
    ///         in the expectedGetters mapping. This ensures we don't miss any new getters that
    ///         might be added to the contract.
    function _validateAllGettersAccounted() internal {
        // Get all function names from the OPContractsManager ABI
        string[] memory allFunctions = _getOpcmGetters();

        // Check for any functions that are not in our expectedGetters mapping
        string[] memory unaccountedGetters = new string[](allFunctions.length);
        uint256 unaccountedCount = 0;

        for (uint256 i = 0; i < allFunctions.length; i++) {
            string memory functionName = allFunctions[i];
            // Check if the getter is not in our mapping (empty string means not set)
            if (bytes(expectedGetters[functionName]).length == 0) {
                unaccountedGetters[unaccountedCount] = functionName;
                unaccountedCount++;
            }
        }

        // If there are unaccounted getters, revert with the list
        if (unaccountedCount > 0) {
            // Create a trimmed array with only the unaccounted getters
            string[] memory trimmedUnaccounted = new string[](unaccountedCount);
            for (uint256 i = 0; i < unaccountedCount; i++) {
                trimmedUnaccounted[i] = unaccountedGetters[i];
            }
            revert VerifyOPCM_UnaccountedGetters(trimmedUnaccounted);
        }
    }

    /// @notice Returns the name of the OPCM contract depending on whether the OPCM is V2.
    /// @return The name of the OPCM contract.
    function _opcmContractName() internal view returns (string memory) {
        return _isOPCMV2() ? "OPContractsManagerV2" : "OPContractsManager";
    }

    /// @notice Checks if the OPCM is V2.
    /// @dev If the OPCM address is not set, default to false.
    /// @return True if the OPCM is V2, false otherwise.
    function _isOPCMV2() internal view returns (bool) {
        // Get the OPCM contract address from the environment variables.
        address opcmAddress = _getOPCMAddress();

        // If the OPCM contract address is not set, default to V1.
        if (opcmAddress == address(0)) {
            return false;
        }

        // If the OPCM contract version is greater than or equal to 7.0.0, then it is OPCM V2.
        return SemverComp.gte(IOPContractsManager(opcmAddress).version(), Constants.OPCM_V2_MIN_VERSION);
    }

    /// @notice Gets the address of the OPCM contract from the environment variables.
    /// @dev If not set, default to address(0).
    /// @return The address of the OPCM contract.
    function _getOPCMAddress() internal view returns (address) {
        // nosemgrep: sol-style-vm-env-only-in-config-sol
        return vm.envOr("OPCM_ADDRESS", address(0));
    }

    /// @notice Verifies security-critical values for contracts where immutables matter.
    /// @param _opcm The OPCM contract that contains the target contract reference.
    /// @param _target The contract reference being verified.
    /// @param _artifact The artifact info for the contract.
    /// @return True if all security-critical values are correct.
    function _verifySecurityCriticalValues(
        IOPContractsManager _opcm,
        OpcmContractRef memory _target,
        ArtifactInfo memory _artifact
    )
        internal
        returns (bool)
    {
        // Silence unused variable warning - artifact is available for future use
        _artifact;

        // Allow skipping security-critical value checks (for tests that modify immutables)
        if (skipSecurityValueChecks()) {
            return true;
        }

        bool success = true;

        // MIPS64: Verify the PreimageOracle it points to
        if (LibString.eq(_target.name, "MIPS64")) {
            success = _verifyPreimageOracle(IMIPS64(_target.addr)) && success;
        }

        // OptimismPortal2: Verify PROOF_MATURITY_DELAY_SECONDS
        if (LibString.eq(_target.name, "OptimismPortal2") || LibString.eq(_target.name, "OptimismPortalInterop")) {
            success = _verifyPortalDelays(IOptimismPortal2(payable(_target.addr))) && success;
        }

        // AnchorStateRegistry: Verify DISPUTE_GAME_FINALITY_DELAY_SECONDS
        if (LibString.eq(_target.name, "AnchorStateRegistry")) {
            success = _verifyAnchorStateRegistryDelays(IAnchorStateRegistry(_target.addr)) && success;
        }

        // OPContractsManagerStandardValidator: Verify all constructor arg values
        if (LibString.eq(_target.name, "OPContractsManagerStandardValidator")) {
            success = _verifyStandardValidatorArgs(_opcm, _target.addr) && success;
        }

        return success;
    }

    /// @notice Verifies the PreimageOracle bytecode that MIPS64 points to.
    /// @param _mips The MIPS64 contract.
    /// @return True if the PreimageOracle bytecode matches expected.
    function _verifyPreimageOracle(IMIPS64 _mips) internal view returns (bool) {
        address oracleAddr = address(_mips.oracle());
        console.log("  Verifying PreimageOracle bytecode...");
        console.log(string.concat("    Address: ", vm.toString(oracleAddr)));

        ArtifactInfo memory oracleArtifact = _loadArtifactInfo(_buildArtifactPath("PreimageOracle"));
        return _compareBytecode(
            oracleAddr.code,
            oracleArtifact.deployedBytecode,
            "PreimageOracle",
            oracleArtifact,
            true // allow immutables for challengePeriod/minProposalSize
        );
    }

    /// @notice Verifies OptimismPortal2 security-critical delay values.
    /// @param _portal The OptimismPortal2 contract.
    /// @return True if delay values match expected.
    function _verifyPortalDelays(IOptimismPortal2 _portal) internal view returns (bool) {
        // nosemgrep: sol-style-vm-env-only-in-config-sol
        uint256 expectedDelay = vm.envOr("EXPECTED_PROOF_MATURITY_DELAY_SECONDS", uint256(604800));
        uint256 actualDelay = _portal.proofMaturityDelaySeconds();

        console.log("  Verifying PROOF_MATURITY_DELAY_SECONDS...");
        console.log(string.concat("    Expected: ", vm.toString(expectedDelay)));
        console.log(string.concat("    Actual: ", vm.toString(actualDelay)));

        if (actualDelay != expectedDelay) {
            console.log("    [FAIL] PROOF_MATURITY_DELAY_SECONDS mismatch");
            return false;
        }
        console.log("    [OK] PROOF_MATURITY_DELAY_SECONDS verified");
        return true;
    }

    /// @notice Verifies AnchorStateRegistry security-critical delay values.
    /// @param _asr The AnchorStateRegistry contract.
    /// @return True if delay values match expected.
    function _verifyAnchorStateRegistryDelays(IAnchorStateRegistry _asr) internal view returns (bool) {
        // nosemgrep: sol-style-vm-env-only-in-config-sol
        uint256 expectedDelay = vm.envOr("EXPECTED_DISPUTE_GAME_FINALITY_DELAY_SECONDS", uint256(302400));
        uint256 actualDelay = _asr.disputeGameFinalityDelaySeconds();

        console.log("  Verifying DISPUTE_GAME_FINALITY_DELAY_SECONDS...");
        console.log(string.concat("    Expected: ", vm.toString(expectedDelay)));
        console.log(string.concat("    Actual: ", vm.toString(actualDelay)));

        if (actualDelay != expectedDelay) {
            console.log("    [FAIL] DISPUTE_GAME_FINALITY_DELAY_SECONDS mismatch");
            return false;
        }
        console.log("    [OK] DISPUTE_GAME_FINALITY_DELAY_SECONDS verified");
        return true;
    }

    /// @notice Verifies all StandardValidator getters are properly validated.
    /// @param _opcm The OPCM contract.
    /// @param _validator The StandardValidator contract address.
    /// @return True if all getters are valid.
    function _verifyStandardValidatorArgs(IOPContractsManager _opcm, address _validator) internal returns (bool) {
        bool success = true;
        console.log("  Verifying StandardValidator args...");

        // Get ALL zero-arg view getters from ABI
        string[] memory allGetters = abi.decode(
            vm.parseJson(
                Process.bash(
                    string.concat(
                        "jq -r '[.abi[] | select(.type == \"function\" and .stateMutability == \"view\" and (.inputs | length) == 0) | .name]' ",
                        _buildArtifactPath("OPContractsManagerStandardValidator")
                    )
                )
            ),
            (string[])
        );

        // Load Container impls for comparison
        // nosemgrep: sol-style-use-abi-encodecall
        (bool callOk, bytes memory containerData) =
            address(_opcm).staticcall(abi.encodeWithSignature("implementations()"));
        if (!callOk) {
            console.log("    [FAIL] Could not fetch implementations from OPCM");
            return false;
        }

        // Get container impl field names
        string[] memory containerFields = _getContainerImplFields();

        // Verify each getter
        for (uint256 i = 0; i < allGetters.length; i++) {
            string memory getter = allGetters[i];
            string memory check = validatorGetterChecks[getter];

            // Fail if getter is unaccounted for
            if (bytes(check).length == 0) {
                console.log(string.concat("    [FAIL] Unaccounted getter: ", getter));
                success = false;
                continue;
            }

            // Skip explicitly skipped getters
            if (LibString.eq(check, "SKIP")) {
                continue;
            }

            // Handle each check type
            if (LibString.eq(check, "CONTAINER_IMPL")) {
                success = _verifyContainerImpl(_validator, getter, containerFields, containerData) && success;
            } else if (LibString.startsWith(check, "ENV:ADDRESS:")) {
                string memory envVar = LibString.slice(check, bytes("ENV:ADDRESS:").length, bytes(check).length);
                success = _verifyEnvAddress(_validator, getter, envVar) && success;
            } else if (LibString.startsWith(check, "ENV:UINT256:")) {
                string memory envVar = LibString.slice(check, bytes("ENV:UINT256:").length, bytes(check).length);
                success = _verifyEnvUint256(_validator, getter, envVar) && success;
            } else if (LibString.eq(check, "ZERO_ON_MAINNET")) {
                success = _verifyZeroOnMainnet(_validator, getter) && success;
            }
        }

        if (success) {
            console.log("    [OK] All StandardValidator args verified");
        }
        return success;
    }

    /// @notice Gets the field names from the Container implementations struct.
    /// @return Array of field names.
    function _getContainerImplFields() internal returns (string[] memory) {
        return abi.decode(
            vm.parseJson(
                Process.bash(
                    string.concat(
                        "jq -r '[.abi[] | select(.name == \"implementations\") | .outputs[0].components[].name]' ",
                        _buildArtifactPath(_opcmContractName())
                    )
                )
            ),
            (string[])
        );
    }

    /// @notice Verifies a StandardValidator getter matches the corresponding Container impl.
    /// @param _validator The StandardValidator address.
    /// @param _getter The getter name.
    /// @param _containerFields Array of Container field names.
    /// @param _containerData ABI-encoded Container implementations struct.
    /// @return True if the values match.
    function _verifyContainerImpl(
        address _validator,
        string memory _getter,
        string[] memory _containerFields,
        bytes memory _containerData
    )
        internal
        view
        returns (bool)
    {
        address actual = _getAddressFromValidator(_validator, string.concat(_getter, "()"));
        address expected = _findContainerImpl(_getter, _containerFields, _containerData);

        if (actual != expected) {
            console.log(string.concat("    [FAIL] ", _getter));
            console.log(string.concat("      Container: ", vm.toString(expected)));
            console.log(string.concat("      Validator: ", vm.toString(actual)));
            return false;
        }
        return true;
    }

    /// @notice Verifies a StandardValidator getter matches an environment variable address.
    /// @param _validator The StandardValidator address.
    /// @param _getter The getter name.
    /// @param _envVar The environment variable name.
    /// @return True if the values match.
    function _verifyEnvAddress(
        address _validator,
        string memory _getter,
        string memory _envVar
    )
        internal
        view
        returns (bool)
    {
        address actual = _getAddressFromValidator(_validator, string.concat(_getter, "()"));
        // nosemgrep: sol-style-vm-env-only-in-config-sol
        address expected = vm.envAddress(_envVar);

        if (actual != expected) {
            console.log(string.concat("    [FAIL] ", _getter));
            console.log(string.concat("      Expected (", _envVar, "): ", vm.toString(expected)));
            console.log(string.concat("      Actual: ", vm.toString(actual)));
            return false;
        }
        return true;
    }

    /// @notice Verifies a StandardValidator getter matches an environment variable uint256.
    /// @param _validator The StandardValidator address.
    /// @param _getter The getter name.
    /// @param _envVar The environment variable name.
    /// @return True if the values match.
    function _verifyEnvUint256(
        address _validator,
        string memory _getter,
        string memory _envVar
    )
        internal
        view
        returns (bool)
    {
        uint256 actual = _getUintFromValidator(_validator, string.concat(_getter, "()"));
        // nosemgrep: sol-style-vm-env-only-in-config-sol
        uint256 expected = vm.envUint(_envVar);

        if (actual != expected) {
            console.log(string.concat("    [FAIL] ", _getter));
            console.log(string.concat("      Expected (", _envVar, "): ", vm.toString(expected)));
            console.log(string.concat("      Actual: ", vm.toString(actual)));
            return false;
        }
        return true;
    }

    /// @notice Verifies a StandardValidator getter is zero on mainnet.
    /// @param _validator The StandardValidator address.
    /// @param _getter The getter name.
    /// @return True if zero on mainnet (or not mainnet).
    function _verifyZeroOnMainnet(address _validator, string memory _getter) internal view returns (bool) {
        // Skip check if not mainnet or if in a testing environment
        // Testing environment is detected by code at the TESTING_ENVIRONMENT_ADDRESS
        if (block.chainid != 1 || Constants.TESTING_ENVIRONMENT_ADDRESS.code.length > 0) {
            return true;
        }

        bytes32 actual = _getBytes32FromValidator(_validator, string.concat(_getter, "()"));

        if (actual != bytes32(0)) {
            console.log(string.concat("    [FAIL] ", _getter, " must be zero on mainnet"));
            return false;
        }
        return true;
    }

    /// @notice Finds the address of a field in the Container implementations struct.
    /// @param _getter The field name to find.
    /// @param _containerFields Array of field names.
    /// @param _containerData ABI-encoded implementations struct.
    /// @return The address at the matching field, or address(0) if not found.
    function _findContainerImpl(
        string memory _getter,
        string[] memory _containerFields,
        bytes memory _containerData
    )
        internal
        pure
        returns (address)
    {
        for (uint256 i = 0; i < _containerFields.length; i++) {
            if (LibString.eq(_getter, _containerFields[i])) {
                return abi.decode(Bytes.slice(_containerData, i * 32, 32), (address));
            }
        }
        return address(0);
    }

    /// @notice Gets an address value from a StandardValidator getter.
    /// @param _validator The StandardValidator address.
    /// @param _sig The function signature (e.g., "superchainConfig()").
    /// @return The address returned by the getter.
    function _getAddressFromValidator(address _validator, string memory _sig) internal view returns (address) {
        // nosemgrep: sol-style-use-abi-encodecall
        (bool ok, bytes memory data) = _validator.staticcall(abi.encodeWithSignature(_sig));
        if (!ok) revert VerifyOPCM_ValidatorCallFailed(_sig);
        return abi.decode(data, (address));
    }

    /// @notice Gets a uint256 value from a StandardValidator getter.
    /// @param _validator The StandardValidator address.
    /// @param _sig The function signature.
    /// @return The uint256 returned by the getter.
    function _getUintFromValidator(address _validator, string memory _sig) internal view returns (uint256) {
        // nosemgrep: sol-style-use-abi-encodecall
        (bool ok, bytes memory data) = _validator.staticcall(abi.encodeWithSignature(_sig));
        if (!ok) revert VerifyOPCM_ValidatorCallFailed(_sig);
        return abi.decode(data, (uint256));
    }

    /// @notice Gets a bytes32 value from a StandardValidator getter.
    /// @param _validator The StandardValidator address.
    /// @param _sig The function signature.
    /// @return The bytes32 returned by the getter.
    function _getBytes32FromValidator(address _validator, string memory _sig) internal view returns (bytes32) {
        // nosemgrep: sol-style-use-abi-encodecall
        (bool ok, bytes memory data) = _validator.staticcall(abi.encodeWithSignature(_sig));
        if (!ok) revert VerifyOPCM_ValidatorCallFailed(_sig);
        return abi.decode(data, (bytes32));
    }

    /// @notice Finds the position of a character in a string.
    /// @param _str The string to search.
    /// @param _char The character to find (as a single-char string).
    /// @return The index of the first occurrence, or string length if not found.
    function _findChar(string memory _str, string memory _char) internal pure returns (uint256) {
        bytes memory strBytes = bytes(_str);
        bytes memory charBytes = bytes(_char);
        if (charBytes.length != 1) revert VerifyOPCM_MustBeSingleChar();
        bytes1 target = charBytes[0];
        for (uint256 i = 0; i < strBytes.length; i++) {
            if (strBytes[i] == target) {
                return i;
            }
        }
        return strBytes.length;
    }
}
