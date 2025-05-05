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

// Interfaces
import { IOPContractsManager } from "interfaces/L1/IOPContractsManager.sol";

/// @title VerifyOPCM
/// @notice Verifies the bytecode of an OPContractsManager instance and all associated blueprints
///         and implementations against locally built artifacts.
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

    /// @notice Thrown when the creation bytecode is not found in an artifact file.
    error VerifyOPCM_CreationBytecodeNotFound(string _artifactPath);

    /// @notice Thrown when the runtime bytecode is not found in an artifact file.
    error VerifyOPCM_RuntimeBytecodeNotFound(string _artifactPath);

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

    /// @notice Setup flag.
    bool internal ready;

    /// @notice Populates override mappings.
    function setUp() public {
        // Overrides for situations where field names do not cleanly map to contract names.
        fieldNameOverrides["optimismPortalImpl"] = "OptimismPortal2";
        fieldNameOverrides["mipsImpl"] = "MIPS64";
        fieldNameOverrides["ethLockboxImpl"] = "ETHLockbox";
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
        fieldNameOverrides["opcmUpgrader"] = "OPContractsManagerUpgrader";
        fieldNameOverrides["opcmInteropMigrator"] = "OPContractsManagerInteropMigrator";

        // Overrides for situations where contracts have differently named source files.
        sourceNameOverrides["OPContractsManagerGameTypeAdder"] = "OPContractsManager";
        sourceNameOverrides["OPContractsManagerDeployer"] = "OPContractsManager";
        sourceNameOverrides["OPContractsManagerUpgrader"] = "OPContractsManager";
        sourceNameOverrides["OPContractsManagerInteropMigrator"] = "OPContractsManager";

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
        _verifyOpcmContractRef(
            OpcmContractRef({ field: _name, name: _name, addr: _addr, blueprint: false }), _skipConstructorVerification
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

        // Collect all the references.
        OpcmContractRef[] memory refs = _collectOpcmContractRefs(opcm);

        // Verify each reference.
        bool success = true;
        for (uint256 i = 0; i < refs.length; i++) {
            success = _verifyOpcmContractRef(refs[i], _skipConstructorVerification) && success;
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
        uint256 extraRefs = 1;
        OpcmContractRef[] memory refs =
            new OpcmContractRef[](propRefs.length + implRefs.length + bpRefs.length + extraRefs);

        // References for OPCM and linked contracts.
        refs[0] = OpcmContractRef({ field: "opcm", name: "OPContractsManager", addr: address(_opcm), blueprint: false });

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

    /// @notice Verifies a single OPCM contract reference (implementation or bytecode).
    /// @param _target The target contract reference to verify.
    /// @param _skipConstructorVerification Whether to skip constructor verification.
    /// @return True if the contract reference is verified, false otherwise.
    function _verifyOpcmContractRef(
        OpcmContractRef memory _target,
        bool _skipConstructorVerification
    )
        internal
        returns (bool)
    {
        console.log();
        console.log(string.concat("Checking Contract: ", _target.field));
        console.log(string.concat("  Type: ", _target.blueprint ? "Blueprint" : "Implementation"));
        console.log(string.concat("  Contract: ", _target.name));
        console.log(string.concat("  Address: ", vm.toString(_target.addr)));

        // Build the expected path to the artifact file.
        string memory artifactPath = _buildArtifactPath(_target.name);
        console.log(string.concat("  Expected Runtime Artifact: ", artifactPath));

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
        bool success = _compareBytecode(actualCode, expectedCode, _target.name, artifact, !_target.blueprint);

        // If requested and this is not a blueprint, we also need to check the creation code.
        if (!_target.blueprint && !_skipConstructorVerification) {
            // Use the Etherscan API to get the creation code.
            bytes memory actualCreationCode = bytes(
                Process.bash(
                    string.concat(
                        "curl -s 'https://api.etherscan.io/v2/api?chainid=",
                        vm.toString(block.chainid),
                        "&module=contract&action=getcontractcreation&contractaddresses=",
                        vm.toString(_target.addr),
                        "&apikey=",
                        Config.etherscanApiKey(),
                        "' | jq -r '.result[0].creationBytecode'"
                    )
                )
            );

            // If we got a creation code, try to grab the constructor arguments from etherscan too.
            if (actualCreationCode.length > 0) {
                // Now try to grab the constructor arguments from etherscan too.
                bytes memory constructorArgs = bytes(
                    Process.bash(
                        string.concat(
                            "curl -s 'https://api.etherscan.io/v2/api?chainid=",
                            vm.toString(block.chainid),
                            "&module=contract&action=getsourcecode&address=",
                            vm.toString(_target.addr),
                            "&apikey=",
                            Config.etherscanApiKey(),
                            "' | jq -r '.result[0].ConstructorArguments'"
                        )
                    )
                );

                // Constructor args might be empty, so we check regardless of the result.
                success = _compareBytecode(
                    actualCreationCode,
                    bytes.concat(artifact.bytecode, constructorArgs),
                    _target.name,
                    artifact,
                    !_target.blueprint
                );
            } else {
                console.log(string.concat("[FAIL] ERROR: Failed to retrieve creation code for ", _target.name));
                success = false;
            }
        }

        // Log final status for this field.
        if (success) {
            console.log(string.concat("Status: [OK] Verified ", _target.name));
        } else {
            console.log(string.concat("Status: [FAIL] Verification failed for ", _target.name));
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
    ///         of contract names and their corresonding addresses for the various immutable
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
                        _buildArtifactPath("OPContractsManager")
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
                        _buildArtifactPath("OPContractsManager")
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
}
