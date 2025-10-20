// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Test } from "forge-std/Test.sol";
import { console } from "forge-std/console.sol";
import { GnosisSafeProxyFactory } from "safe-contracts/proxies/GnosisSafeProxyFactory.sol";
import { GnosisSafeProxy } from "safe-contracts/proxies/GnosisSafeProxy.sol";
import { GnosisSafe } from "safe-contracts/GnosisSafe.sol";
import { Enum } from "safe-contracts/common/Enum.sol";

/// @title SafeCompatibilityTest
/// @notice Base test contract for testing Safe compatibility across ALL versions on Mainnet fork
/// @dev This contract creates Safe wallets for ALL major versions (1.0.0 through 1.5.0) on Mainnet fork
contract SafeCompatibilityTest is Test {
    // Safe version information
    struct SafeVersion {
        string version;
        address singleton;
        address proxyFactory;
        GnosisSafe safe;
    }

    // Official Safe deployments on Ethereum Mainnet
    // Source: https://github.com/safe-global/safe-deployments

    // Safe v1.0.0
    address constant SAFE_SINGLETON_V1_0_0 = 0xb6029EA3B2c51D09a50B53CA8012FeEB05bDa35A;
    address constant SAFE_PROXY_FACTORY_V1_0_0 = 0x12302fE9c02ff50939BaAaaf415fc226C078613C;

    // Safe v1.1.1
    address constant SAFE_SINGLETON_V1_1_1 = 0x34CfAC646f301356fAa8B21e94227e3583Fe3F5F;
    address constant SAFE_PROXY_FACTORY_V1_1_1 = 0x76E2cFc1F5Fa8F6a5b3fC4c8F4788F0116861F9B;

    // Safe v1.2.0
    address constant SAFE_SINGLETON_V1_2_0 = 0x6851D6fDFAfD08c0295C392436245E5bc78B0185;
    address constant SAFE_PROXY_FACTORY_V1_2_0 = 0x76E2cFc1F5Fa8F6a5b3fC4c8F4788F0116861F9B;

    // Safe v1.3.0 (L1)
    address constant SAFE_SINGLETON_V1_3_0 = 0xd9Db270c1B5E3Bd161E8c8503c55cEABeE709552;
    address constant SAFE_PROXY_FACTORY_V1_3_0 = 0xa6B71E26C5e0845f74c812102Ca7114b6a896AB2;

    // Safe v1.4.1 (L1)
    address constant SAFE_SINGLETON_V1_4_1 = 0x41675C099F32341bf84BFc5382aF534df5C7461a;
    address constant SAFE_PROXY_FACTORY_V1_4_1 = 0x4e1DCf7AD4e460CfD30791CCC4F9c8a4f820ec67;

    // Safe v.1.5.0 (L1)
    address constant SAFE_SINGLETON_V1_5_0 = 0xFf51A5898e281Db6DfC7855790607438dF2ca44b;
    address constant SAFE_PROXY_FACTORY_V1_5_0 = 0x14F2982D601c9458F93bd70B218933A6f8165e7b;

    SafeVersion[] public safeVersions;

    // Test owners
    address[] public owners;
    uint256 public threshold = 2;

    // Owner private keys for signing transactions
    uint256 constant OWNER1_PK = 0x1;
    uint256 constant OWNER2_PK = 0x2;
    uint256 constant OWNER3_PK = 0x3;

    function setUp() public virtual {
        // Only run on fork tests (Mainnet)
        if (!isForkTest()) {
            vm.skip(true);
            return;
        }

        // Setup owners (sorted by address for Safe compatibility)
        owners = new address[](3);
        owners[0] = vm.addr(OWNER1_PK);
        owners[1] = vm.addr(OWNER2_PK);
        owners[2] = vm.addr(OWNER3_PK);

        // Sort owners
        owners = _sortAddresses(owners);

        // Label owners
        vm.label(owners[0], "Owner1");
        vm.label(owners[1], "Owner2");
        vm.label(owners[2], "Owner3");

        // Create Safe instances for each version
        _createSafeVersion("v1.0.0", SAFE_SINGLETON_V1_0_0, SAFE_PROXY_FACTORY_V1_0_0);
        _createSafeVersion("v1.1.1", SAFE_SINGLETON_V1_1_1, SAFE_PROXY_FACTORY_V1_1_1);
        _createSafeVersion("v1.2.0", SAFE_SINGLETON_V1_2_0, SAFE_PROXY_FACTORY_V1_2_0);
        _createSafeVersion("v1.3.0", SAFE_SINGLETON_V1_3_0, SAFE_PROXY_FACTORY_V1_3_0);
        _createSafeVersion("v1.4.1", SAFE_SINGLETON_V1_4_1, SAFE_PROXY_FACTORY_V1_4_1);
        _createSafeVersion("v1.5.0", SAFE_SINGLETON_V1_5_0, SAFE_PROXY_FACTORY_V1_5_0);
    }

    /// @notice Checks if this is a fork test
    /// @dev This should match your project's fork detection mechanism
    function isForkTest() public view virtual returns (bool) {
        // Check if we're on Mainnet (chainId 1)
        return block.chainid == 1;
    }

    /// @notice Creates a Safe proxy for a specific version
    /// @param version Version label for this Safe
    /// @param singleton Address of the Safe singleton contract
    /// @param proxyFactory Address of the Safe proxy factory
    function _createSafeVersion(string memory version, address singleton, address proxyFactory) internal {
        // Verify contracts exist at these addresses
        uint256 singletonSize;
        uint256 factorySize;
        assembly {
            singletonSize := extcodesize(singleton)
            factorySize := extcodesize(proxyFactory)
        }

        if (singletonSize == 0 || factorySize == 0) {
            // Skip if contracts don't exist (might not be deployed on this network)
            emit log_named_string("Skipping Safe version (not deployed)", version);
            return;
        }

        GnosisSafeProxyFactory factory = GnosisSafeProxyFactory(proxyFactory);

        // Prepare initialization data
        bytes memory initializer = abi.encodeCall(
            GnosisSafe.setup,
            (
                owners,
                threshold,
                address(0), // to (for module setup)
                "", // data (for module setup)
                address(0), // fallbackHandler
                address(0), // paymentToken
                0, // payment
                payable(address(0)) // paymentReceiver
            )
        );

        // Create proxy
        GnosisSafeProxy proxy = factory.createProxyWithNonce(
            singleton, initializer, uint256(keccak256(abi.encodePacked(version, block.timestamp)))
        );

        // Cast to GnosisSafe
        GnosisSafe safe = GnosisSafe(payable(address(proxy)));

        // Fund the safe with some ETH
        vm.deal(address(safe), 10 ether);

        // Store version info
        safeVersions.push(
            SafeVersion({ version: version, singleton: singleton, proxyFactory: proxyFactory, safe: safe })
        );

        vm.label(address(safe), string.concat("Safe_", version));
        emit log_named_string("Created Safe version", version);
        emit log_named_address("Safe address", address(safe));
    }

    /// @notice Helper function to test custom logic against all Safe versions (STRICT MODE)
    /// @dev This version will FAIL the test if any version reverts
    /// @param testLogic Function pointer to test logic
    function forEachSafeVersion(function(SafeVersion memory) external testLogic) internal {
        string[] memory emptySkipList;
        forEachSafeVersion(testLogic, emptySkipList);
    }

    /// @notice Helper function to test custom logic against all Safe versions with version skipping (STRICT MODE)
    /// @dev This version will FAIL the test if any version reverts
    /// @param testLogic Function pointer to test logic
    /// @param skipVersions Array of version strings to skip (e.g., ["v1.0.0", "v1.1.1"])
    function forEachSafeVersion(
        function(SafeVersion memory) external testLogic,
        string[] memory skipVersions
    )
        internal
    {
        for (uint256 i = 0; i < safeVersions.length; i++) {
            // Check if this version should be skipped
            if (_shouldSkipVersion(safeVersions[i].version, skipVersions)) {
                emit log_named_string("Skipping Safe version", safeVersions[i].version);
                continue;
            }

            emit log_named_string("Testing Safe version", safeVersions[i].version);
            testLogic(safeVersions[i]);
        }
    }

    /// @notice Helper function to test custom logic against all Safe versions (VERBOSE MODE)
    /// @dev This version will CATCH reverts and log them instead of failing
    ///      Use this to explore what works/doesn't work on each Safe version
    /// @param testLogic Function pointer to test logic
    function forEachSafeVersionVerbose(function(SafeVersion memory) external testLogic) internal {
        string[] memory emptySkipList;
        forEachSafeVersionVerbose(testLogic, emptySkipList);
    }

    /// @notice Helper function to test custom logic against all Safe versions with version skipping (VERBOSE MODE)
    /// @dev This version will CATCH reverts and log them instead of failing
    ///      Use this to explore what works/doesn't work on each Safe version
    /// @param testLogic Function pointer to test logic
    /// @param skipVersions Array of version strings to skip (e.g., ["v1.0.0", "v1.1.1"])
    function forEachSafeVersionVerbose(
        function(SafeVersion memory) external testLogic,
        string[] memory skipVersions
    )
        internal
    {
        console.log("\n=== VERBOSE MODE: Testing across all Safe versions ===\n");

        if (skipVersions.length > 0) {
            console.log("Skipping versions:");
            for (uint256 i = 0; i < skipVersions.length; i++) {
                console.log(string.concat("  - ", skipVersions[i]));
            }
            console.log("");
        }

        uint256 successCount = 0;
        uint256 failCount = 0;
        uint256 skippedCount = 0;

        for (uint256 i = 0; i < safeVersions.length; i++) {
            SafeVersion memory sv = safeVersions[i];

            // Check if this version should be skipped
            if (_shouldSkipVersion(sv.version, skipVersions)) {
                console.log("--------------------------------------");
                console.log(string.concat("Skipping Safe ", sv.version));
                console.log("--------------------------------------\n");
                skippedCount++;
                continue;
            }

            console.log("--------------------------------------");
            console.log(string.concat("Testing Safe ", sv.version));
            console.log("--------------------------------------");

            // Use try/catch to capture reverts
            try testLogic(sv) {
                console.log(string.concat("[SUCCESS] ", sv.version, " - Test passed"));
                successCount++;
            } catch Error(string memory reason) {
                console.log(string.concat("[FAILED] ", sv.version, " - Reverted with: ", reason));
                failCount++;
            } catch (bytes memory lowLevelData) {
                console.log(string.concat("[FAILED] ", sv.version, " - Reverted with low-level error"));
                console.logBytes(lowLevelData);
                failCount++;
            }
            console.log("");
        }

        console.log("======================================");
        console.log("SUMMARY:");
        uint256 totalTested = safeVersions.length - skippedCount;
        console.log(string.concat("  Passed: ", vm.toString(successCount), "/", vm.toString(totalTested)));
        console.log(string.concat("  Failed: ", vm.toString(failCount), "/", vm.toString(totalTested)));
        if (skippedCount > 0) {
            console.log(string.concat("  Skipped: ", vm.toString(skippedCount)));
        }
        console.log("======================================\n");
    }

    /// @notice Check if a version should be skipped
    /// @param version The version string to check
    /// @param skipVersions Array of version strings to skip
    /// @return True if the version should be skipped
    function _shouldSkipVersion(string memory version, string[] memory skipVersions) internal pure returns (bool) {
        for (uint256 i = 0; i < skipVersions.length; i++) {
            if (keccak256(bytes(version)) == keccak256(bytes(skipVersions[i]))) {
                return true;
            }
        }
        return false;
    }

    /// @notice Helper to execute a Safe transaction with signatures from owners
    /// @param safe The Safe to execute on
    /// @param to Target address
    /// @param value ETH value
    /// @param data Transaction data
    /// @param operation Call or DelegateCall
    function _executeSafeTransaction(
        GnosisSafe safe,
        address to,
        uint256 value,
        bytes memory data,
        Enum.Operation operation
    )
        internal
    {
        // Get transaction hash
        uint256 nonce = safe.nonce();
        bytes32 txHash = _getTransactionHash(safe, to, value, data, operation, nonce);

        // Generate signatures from threshold number of owners
        bytes memory signatures = _generateSignatures(txHash, threshold);

        // Execute transaction
        bool success = safe.execTransaction(
            to,
            value,
            data,
            operation,
            0, // safeTxGas
            0, // baseGas
            0, // gasPrice
            address(0), // gasToken
            payable(address(0)), // refundReceiver
            signatures
        );

        assertTrue(success, "Safe transaction execution failed");
    }

    /// @notice Helper to get transaction hash
    function _getTransactionHash(
        GnosisSafe safe,
        address to,
        uint256 value,
        bytes memory data,
        Enum.Operation operation,
        uint256 nonce
    )
        internal
        view
        returns (bytes32)
    {
        return safe.getTransactionHash(
            to,
            value,
            data,
            operation,
            0, // safeTxGas
            0, // baseGas
            0, // gasPrice
            address(0), // gasToken
            address(0), // refundReceiver
            nonce
        );
    }

    /// @notice Generate signatures from multiple owners
    /// @param txHash Hash to sign
    /// @param numSignatures Number of signatures to generate
    /// @return Packed signatures
    function _generateSignatures(bytes32 txHash, uint256 numSignatures) internal pure returns (bytes memory) {
        bytes memory signatures;

        uint256[3] memory pks = [OWNER1_PK, OWNER2_PK, OWNER3_PK];
        address[] memory signers = new address[](numSignatures);

        // Get signers
        for (uint256 i = 0; i < numSignatures; i++) {
            signers[i] = vm.addr(pks[i]);
        }

        // Sort signers (Safe requires sorted signatures)
        signers = _sortAddresses(signers);

        // Generate signatures in order
        for (uint256 i = 0; i < numSignatures; i++) {
            // Find the private key for this signer
            uint256 pk;
            for (uint256 j = 0; j < 3; j++) {
                if (vm.addr(pks[j]) == signers[i]) {
                    pk = pks[j];
                    break;
                }
            }

            (uint8 v, bytes32 r, bytes32 s) = vm.sign(pk, txHash);
            signatures = abi.encodePacked(signatures, r, s, v);
        }

        return signatures;
    }

    /// @notice Sort addresses in ascending order (required for Safe)
    /// @param addrs Array of addresses to sort
    /// @return Sorted array
    function _sortAddresses(address[] memory addrs) internal pure returns (address[] memory) {
        uint256 length = addrs.length;
        for (uint256 i = 0; i < length; i++) {
            for (uint256 j = i + 1; j < length; j++) {
                if (addrs[i] > addrs[j]) {
                    address temp = addrs[i];
                    addrs[i] = addrs[j];
                    addrs[j] = temp;
                }
            }
        }
        return addrs;
    }

    /// @notice Get a specific Safe version by index
    /// @param index Index of the version
    /// @return SafeVersion struct
    function getSafeVersion(uint256 index) public view returns (SafeVersion memory) {
        if (index >= safeVersions.length) {
            revert("SafeCompatibilityTest: invalid version index");
        }
        return safeVersions[index];
    }

    /// @notice Get total number of Safe versions created
    /// @return Number of versions
    function getSafeVersionCount() public view returns (uint256) {
        return safeVersions.length;
    }
}
