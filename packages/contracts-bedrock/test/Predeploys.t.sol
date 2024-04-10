// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { CommonTest } from "test/setup/CommonTest.sol";
import { EIP1967Helper } from "test/mocks/EIP1967Helper.sol";
import { console2 as console } from "forge-std/console2.sol";
import { Predeploys } from "src/libraries/Predeploys.sol";

/// @title PredeploysTest
contract PredeploysTest is CommonTest {
    //////////////////////////////////////////////////////
    /// Internal helpers
    //////////////////////////////////////////////////////

    /// @dev Returns true if the address is a predeploy that is active (i.e. embedded in L2 genesis).
    function _isPredeploy(address _addr) internal pure returns (bool) {
        return _addr == Predeploys.L2_TO_L1_MESSAGE_PASSER || _addr == Predeploys.L2_CROSS_DOMAIN_MESSENGER
            || _addr == Predeploys.L2_STANDARD_BRIDGE || _addr == Predeploys.L2_ERC721_BRIDGE
            || _addr == Predeploys.SEQUENCER_FEE_WALLET || _addr == Predeploys.OPTIMISM_MINTABLE_ERC20_FACTORY
            || _addr == Predeploys.OPTIMISM_MINTABLE_ERC721_FACTORY || _addr == Predeploys.L1_BLOCK_ATTRIBUTES
            || _addr == Predeploys.GAS_PRICE_ORACLE || _addr == Predeploys.DEPLOYER_WHITELIST || _addr == Predeploys.WETH9
            || _addr == Predeploys.L1_BLOCK_NUMBER || _addr == Predeploys.LEGACY_MESSAGE_PASSER
            || _addr == Predeploys.PROXY_ADMIN || _addr == Predeploys.BASE_FEE_VAULT || _addr == Predeploys.L1_FEE_VAULT
            || _addr == Predeploys.GOVERNANCE_TOKEN || _addr == Predeploys.SCHEMA_REGISTRY || _addr == Predeploys.EAS;
    }

    /// @dev Returns true if the address is not proxied.
    function _notProxied(address _addr) internal pure returns (bool) {
        return _addr == Predeploys.GOVERNANCE_TOKEN || _addr == Predeploys.WETH9;
    }

    /// @dev Returns true if the account is not meant to be in the L2 genesis anymore.
    function _isOmitted(address _addr) internal pure returns (bool) {
        return _addr == Predeploys.L1_MESSAGE_SENDER;
    }

    function _isInitializable(address _addr) internal pure returns (bool) {
        return !(
            _addr == Predeploys.LEGACY_MESSAGE_PASSER || _addr == Predeploys.DEPLOYER_WHITELIST
                || _addr == Predeploys.GAS_PRICE_ORACLE || _addr == Predeploys.SEQUENCER_FEE_WALLET
                || _addr == Predeploys.BASE_FEE_VAULT || _addr == Predeploys.L1_FEE_VAULT
                || _addr == Predeploys.L1_BLOCK_NUMBER || _addr == Predeploys.L1_BLOCK_ATTRIBUTES
                || _addr == Predeploys.L2_TO_L1_MESSAGE_PASSER || _addr == Predeploys.OPTIMISM_MINTABLE_ERC721_FACTORY
                || _addr == Predeploys.PROXY_ADMIN || _addr == Predeploys.SCHEMA_REGISTRY || _addr == Predeploys.EAS
                || _addr == Predeploys.GOVERNANCE_TOKEN
        );
    }

    function _usesImmutables(address _addr) internal pure returns (bool) {
        return _addr == Predeploys.OPTIMISM_MINTABLE_ERC721_FACTORY || _addr == Predeploys.SEQUENCER_FEE_WALLET
            || _addr == Predeploys.BASE_FEE_VAULT || _addr == Predeploys.L1_FEE_VAULT || _addr == Predeploys.EAS
            || _addr == Predeploys.GOVERNANCE_TOKEN;
    }

    function test_predeployToCodeNamespace() external pure {
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

    /// @dev Tests that the predeploy addresses are set correctly. They have code
    ///      and the proxied accounts have the correct admin.
    function test_predeploys_succeeds() external {
        uint256 count = 2048;
        uint160 prefix = uint160(0x420) << 148;

        bytes memory proxyCode = vm.getDeployedCode("Proxy.sol:Proxy");

        for (uint256 i = 0; i < count; i++) {
            console.log("asserting predeploy %d", i);
            address addr = address(prefix | uint160(i));
            bytes memory code = addr.code;
            assertTrue(code.length > 0);

            address implAddr = Predeploys.predeployToCodeNamespace(addr);

            if (_isOmitted(addr)) {
                assertEq(implAddr.code.length, 0, "must have no code");
                console.log("predeploy %d is intentionally not in genesis", i);
                continue;
            }
            bool isPredeploy = _isPredeploy(addr);

            if (!isPredeploy) {
                // All of the predeploys, even if inactive, have their admin set to the proxy admin
                assertEq(EIP1967Helper.getAdmin(addr), Predeploys.PROXY_ADMIN, "Admin mismatch");
                continue;
            }
            bool proxied = _notProxied(addr) == false;

            string memory cname = Predeploys.getName(addr);
            assertNotEq(cname, "", "must have a name");

            console.log("finding supposed code");
            bytes memory supposedCode = vm.getDeployedCode(string.concat(cname, ".sol:", cname));
            assertNotEq(supposedCode.length, 0, "must have supposed code");

            if (proxied == false) {
                if (!_usesImmutables(addr)) {
                    // can't check bytecode if it's modified with immutables in genesis.
                    assertEq(code, supposedCode, "non-proxy contract should be deployed in-place");
                }
                console.log("predeploy %d is not a proxy", i);
                // TODO: should we still test the "initialized" slot?
                continue;
            }

            console.log("checking proxy properties of active predeploy");

            // The code is a proxy
            assertEq(code, proxyCode);

            assertEq(
                EIP1967Helper.getImplementation(addr),
                implAddr,
                string.concat("Implementation mismatch for ", vm.toString(addr))
            );
            assertNotEq(implAddr.code.length, 0, "predeploy implementation account must have code");
            if (!_usesImmutables(addr)) {
                // can't check bytecode if it's modified with immutables in genesis.
                assertEq(implAddr.code, supposedCode, "proxy implementation contract should match contract source");
            }

            if (_isInitializable(addr)) {
                console.log("loading initialized slot of %s", cname);
                uint8 initializedSlot = l2Genesis.loadInitializedSlot(cname);
                bytes32 implInitialized = vm.load(implAddr, bytes32(uint256(initializedSlot)));
                bytes32 proxyInitialized = vm.load(addr, bytes32(uint256(initializedSlot)));
                console.log("loaded initialized slot of %s: slot=%s, impl=%s, proxy=%s", cname, initializedSlot);
                console.log("slot values: impl=%s, proxy=%s", uint256(implInitialized), uint256(proxyInitialized));
                // TODO: should we be asserting a global initialized value as constant? Or can they differ?
                // Storage packing causes some edge cases in this assertion
                //                assertNotEq(bytes32(0), implInitialized, "implementation must be initialized, even
                // though behind a proxy");
                //                assertNotEq(bytes32(0), proxyInitialized , "proxy must be initialized");
            }
        }
    }
}
