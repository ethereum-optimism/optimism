// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { CommonTest } from "test/setup/CommonTest.sol";
import { EIP1967Helper } from "test/mocks/EIP1967Helper.sol";

// Libraries
import { Predeploys } from "src/libraries/Predeploys.sol";
import { ForgeArtifacts } from "scripts/libraries/ForgeArtifacts.sol";
import { Fork } from "scripts/libraries/Config.sol";
import { Features } from "src/libraries/Features.sol";
import { DevFeatures } from "src/libraries/DevFeatures.sol";

/// @title Predeploys_TestInit
/// @notice Reusable test initialization for `Predeploys` tests.
abstract contract Predeploys_TestInit is CommonTest {
    //////////////////////////////////////////////////////
    /// Internal helpers
    //////////////////////////////////////////////////////

    /// @notice Returns true if the predeploy has a custom gas token implementation variant.
    function _customGasTokenCodeDiffer(address _addr) internal pure returns (bool) {
        Predeploys.PredeployRecord[] memory records = Predeploys.getAllRecords();
        for (uint256 i = 0; i < records.length; i++) {
            if (records[i].proxy == _addr && records[i].variants.length > uint256(Predeploys.VariantKind.CGT)) {
                return true;
            }
        }
        return false;
    }

    /// @notice Returns true if the account is not meant to be in the L2 genesis anymore.
    function _isOmitted(address _addr) internal pure returns (bool) {
        return _addr == Predeploys.L1_MESSAGE_SENDER;
    }

    /// @notice Returns true if the predeploy is initializable and uses OpenZeppelin v4 storage pattern.
    ///         These contracts have _initialized in the regular storage layout.
    function _isInitializableV4(address _addr) internal pure returns (bool) {
        return _addr == Predeploys.L2_CROSS_DOMAIN_MESSENGER || _addr == Predeploys.L2_STANDARD_BRIDGE
            || _addr == Predeploys.L2_ERC721_BRIDGE || _addr == Predeploys.OPTIMISM_MINTABLE_ERC20_FACTORY;
    }

    /// @notice Returns true if the predeploy is initializable and uses OpenZeppelin v5 namespaced storage (EIP-7201).
    ///         These contracts store _initialized in a namespaced slot, not in the regular storage layout.
    function _isInitializableV5(address _addr) internal pure returns (bool) {
        return _addr == Predeploys.SEQUENCER_FEE_WALLET || _addr == Predeploys.BASE_FEE_VAULT
            || _addr == Predeploys.L1_FEE_VAULT || _addr == Predeploys.OPERATOR_FEE_VAULT;
    }

    /// @notice Returns true if the predeploy uses immutables.
    function _usesImmutables(address _addr) internal pure returns (bool) {
        return _addr == Predeploys.OPTIMISM_MINTABLE_ERC721_FACTORY || _addr == Predeploys.EAS
            || _addr == Predeploys.GOVERNANCE_TOKEN;
    }

    /// @notice Returns true if the record's feature gates are satisfied for the given fork/flags.
    function _isRecordActive(
        Predeploys.PredeployRecord memory _r,
        Fork _fork,
        bool _isCustomGasToken,
        bool _isInterop
    )
        internal
        view
        returns (bool)
    {
        if (_r.devFeatureGate != bytes32(0) && !DevFeatures.isDevFeatureEnabled(devFeatureBitmap, _r.devFeatureGate)) {
            return false;
        }
        if (_r.isInterop) {
            return _isInterop && uint256(_fork) >= uint256(Fork.INTEROP);
        }
        if (_r.isCustomGasToken) return _isCustomGasToken;
        return true;
    }

    /// @notice Internal test function for predeploys validation across different forks.
    function _test_predeploys(Fork _fork, bool _isCustomGasToken, bool _isInterop) internal {
        uint256 count = 2048;
        uint160 prefix = uint160(0x420) << 148;

        bytes memory proxyCode = vm.getDeployedCode("Proxy.sol:Proxy");

        for (uint256 i = 0; i < count; i++) {
            address addr = address(prefix | uint160(i));
            address implAddr = Predeploys.predeployToCodeNamespace(addr);

            if (_isOmitted(addr)) {
                assertEq(implAddr.code.length, 0, "must have no code");
                continue;
            }

            bool isPredeploy =
                Predeploys.isSupportedPredeploy(addr, uint256(_fork), _isCustomGasToken, _isInterop, devFeatureBitmap);

            bytes memory code = addr.code;
            if (isPredeploy) assertTrue(code.length > 0);

            bool proxied = Predeploys.notProxied(addr) == false;

            if (!isPredeploy) {
                // All of the predeploys, even if inactive, have their admin set to the proxy admin
                if (proxied) assertEq(EIP1967Helper.getAdmin(addr), Predeploys.PROXY_ADMIN, "Admin mismatch");
                continue;
            }

            string memory cname = Predeploys.getName(addr);
            assertNotEq(cname, "", "must have a name");

            bytes memory supposedCode = vm.getDeployedCode(string.concat(cname, ".sol:", cname));
            assertNotEq(supposedCode.length, 0, "must have supposed code");

            if (proxied == false) {
                // can't check bytecode if it's modified with immutables in genesis.
                if (!_usesImmutables(addr)) {
                    assertEq(code, supposedCode, "non-proxy contract should be deployed in-place");
                }
                continue;
            }

            // The code is a proxy
            assertEq(code, proxyCode);

            assertEq(
                EIP1967Helper.getImplementation(addr),
                implAddr,
                string.concat("Implementation mismatch for ", vm.toString(addr))
            );
            assertNotEq(implAddr.code.length, 0, "predeploy implementation account must have code");
            if (!_usesImmutables(addr) && !_customGasTokenCodeDiffer(addr)) {
                // can't check bytecode if it's modified with immutables in genesis.
                assertEq(implAddr.code, supposedCode, "proxy implementation contract should match contract source");
            }

            if (_isInitializableV4(addr)) {
                assertTrue(ForgeArtifacts.isInitialized({ _name: cname, _address: addr }));
                assertTrue(ForgeArtifacts.isInitialized({ _name: cname, _address: implAddr }));
            }

            if (_isInitializableV5(addr)) {
                assertTrue(
                    ForgeArtifacts.isInitializedV5(addr), string.concat("V5 proxy not initialized: ", vm.toString(addr))
                );
                assertTrue(
                    ForgeArtifacts.isInitializedV5(implAddr),
                    string.concat("V5 implementation not initialized: ", vm.toString(implAddr))
                );
            }
        }
    }
}

/// @title Predeploys_PredeployToCodeNamespace_Test
/// @notice Tests the `predeployToCodeNamespace` function of the `Predeploys` contract.
contract Predeploys_PredeployToCodeNamespace_Test is Predeploys_TestInit {
    /// @notice Tests that predeployToCodeNamespace correctly computes namespace addresses.
    function test_predeployToCodeNamespace_works() external pure {
        assertEq(
            address(0xc0D3C0d3C0d3C0D3c0d3C0d3c0D3C0d3c0d30000),
            Predeploys.predeployToCodeNamespace(Predeploys.LEGACY_MESSAGE_PASSER)
        );
        assertEq(
            address(0xc0d3C0d3C0d3c0D3C0D3C0d3C0d3C0D3C0D3000f),
            Predeploys.predeployToCodeNamespace(Predeploys.GAS_PRICE_ORACLE)
        );
        assertEq(
            address(0xC0d3C0d3c0d3c0d3C0d3c0D3c0D3c0D3C0d30420),
            Predeploys.predeployToCodeNamespace(address(0x4200000000000000000000000000000000000420))
        );
    }
}

/// @title Predeploys_GetName_Test
/// @notice Tests the `getName` function of the `Predeploys` library.
contract Predeploys_GetName_Test is Predeploys_TestInit {
    /// @notice Tests that for addresses with CGT variants which share their proxy address with a primary
    ///         record getName always return the primary name.
    function test_getName_forVariantProxy_succeeds() external pure {
        assertEq(Predeploys.getName(Predeploys.L1_BLOCK_ATTRIBUTES), "L1Block");
        assertEq(Predeploys.getName(Predeploys.L2_TO_L1_MESSAGE_PASSER), "L2ToL1MessagePasser");
    }
}

/// @title Predeploys_Uncategorized_Test
/// @notice General tests that are not testing any function directly of the `Predeploys` contract
///         or are testing multiple functions at once.
contract Predeploys_Uncategorized_Test is Predeploys_TestInit {
    /// @notice Tests that the predeploy addresses are set correctly. They have code
    ///         and the proxied accounts have the correct admin.
    function test_predeploys_succeeds() external {
        _test_predeploys(Fork.ISTHMUS, false, false);
    }

    /// @notice Tests that the predeploy addresses are set correctly. They have code
    ///         and the proxied accounts have the correct admin. Using custom gas token.
    function test_predeploys_customGasToken_succeeds() external {
        skipIfSysFeatureDisabled(Features.CUSTOM_GAS_TOKEN);
        _test_predeploys(Fork.ISTHMUS, true, false);
    }

    /// @notice Tests that the predeploy addresses are set correctly. They have code
    ///         and the proxied accounts have the correct admin. Using l2cm.
    function test_predeploys_l2cm_succeeds() external {
        skipIfDevFeatureDisabled(DevFeatures.L2CM);
        _test_predeploys(Fork.ISTHMUS, false, false);
    }
}

/// @title Predeploys_GetAllRecords_Test
/// @notice Tests ordering invariants of the `getAllRecords` return value.
contract Predeploys_GetAllRecords_Test is Predeploys_TestInit {
    /// @notice Test that once a non-proxied record appears at index i, every subsequent record j > i must
    ///         also be non-proxied or deprecated.
    function test_getAllRecords_nonProxiedRecordsAtEnd_succeeds() external pure {
        Predeploys.PredeployRecord[] memory records = Predeploys.getAllRecords();
        bool seenNonProxied = false;
        for (uint256 i = 0; i < records.length; i++) {
            if (!records[i].isProxied) seenNonProxied = true;
            if (seenNonProxied) {
                assertTrue(
                    !records[i].isProxied || records[i].isDeprecated,
                    string.concat(Predeploys.implName(records[i]), ": proxied record appears after non-proxied record")
                );
            }
        }
    }

    /// @notice Test that once a deprecated record appears at index i, every subsequent record j > i must
    ///         also be deprecated.
    function test_getAllRecords_deprecatedRecordsAtEnd_succeeds() external pure {
        Predeploys.PredeployRecord[] memory records = Predeploys.getAllRecords();
        bool seenDeprecated = false;
        for (uint256 i = 0; i < records.length; i++) {
            if (records[i].isDeprecated) seenDeprecated = true;
            if (seenDeprecated) {
                assertTrue(
                    records[i].isDeprecated,
                    string.concat(
                        Predeploys.implName(records[i]), ": non-deprecated record appears after deprecated record"
                    )
                );
            }
        }
    }
}

/// @title Predeploys_Interop_Uncategorized_Test
/// @notice General tests that are not testing any function directly of the `Predeploys` contract
///         or are testing multiple functions at once, using interop mode.
contract Predeploys_UncategorizedInterop_Test is Predeploys_TestInit {
    /// @notice Test setup. Enabling interop to get all predeploys.
    function setUp() public virtual override {
        super.enableInterop();
        super.setUp();
    }

    /// @notice Tests that the predeploy addresses are set correctly. They have code and the
    ///         proxied accounts have the correct admin. Using interop.
    function test_predeploys_succeeds() external {
        _test_predeploys(Fork.INTEROP, false, true);
    }

    /// @notice Tests that the predeploy addresses are set correctly. They have code and the
    ///         proxied accounts have the correct admin. Using interop with custom gas token.
    function test_predeploys_customGasToken_succeeds() external {
        skipIfSysFeatureDisabled(Features.CUSTOM_GAS_TOKEN);
        _test_predeploys(Fork.INTEROP, true, true);
    }
}
