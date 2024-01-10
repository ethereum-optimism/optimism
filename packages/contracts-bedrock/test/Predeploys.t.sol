// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { CommonTest } from "test/setup/CommonTest.sol";
import { EIP1967Helper } from "test/mocks/EIP1967Helper.sol";
import { Predeploys } from "src/libraries/Predeploys.sol";

/// @title PredeploysTest
contract PredeploysTest is CommonTest {
    /// @dev Function to compute the expected address of the predeploy implementation
    ///      in the genesis state.
    function _predeployToCodeNamespace(address _addr) internal pure returns (address) {
        return address(
            uint160(uint256(uint160(_addr)) & 0xffff | uint256(uint160(0xc0D3C0d3C0d3C0D3c0d3C0d3c0D3C0d3c0d30000)))
        );
    }

    /// @dev Returns true if the address is a predeploy.
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

    /// @dev Returns true if the adress is not proxied.
    function _notProxied(address _addr) internal pure returns (bool) {
        return _addr == Predeploys.LEGACY_ERC20_ETH || _addr == Predeploys.GOVERNANCE_TOKEN || _addr == Predeploys.WETH9
            || _addr == Predeploys.MultiCall3 || _addr == Predeploys.Create2Deployer || _addr == Predeploys.Safe_v130
            || _addr == Predeploys.SafeL2_v130 || _addr == Predeploys.MultiSendCallOnly_v130
            || _addr == Predeploys.SafeSingletonFactory || _addr == Predeploys.DeterministicDeploymentProxy
            || _addr == Predeploys.MultiSend_v130 || _addr == Predeploys.Permit2 || _addr == Predeploys.SenderCreator
            || _addr == Predeploys.EntryPoint;
    }

    /// @dev Tests that the predeploy addresses are set correctly. They have code
    ///      and the proxied accounts have the correct admin.
    function test_predeploysSet_succeeds() external {
        uint256 count = 2048;
        uint160 prefix = uint160(0x420) << 148;
        for (uint256 i = 0; i < count; i++) {
            address addr = address(prefix | uint160(i));
            bytes memory code = addr.code;
            assertTrue(code.length > 0);

            bool proxied = _notProxied(addr) == false;
            bool isPredeploy = _isPredeploy(addr);

            // Skip the accounts that do not have a proxy
            if (proxied == false) {
                continue;
            }

            // Only the defined predeploys have their implementation slot set
            if (proxied && isPredeploy) {
                assertEq(
                    EIP1967Helper.getImplementation(addr),
                    _predeployToCodeNamespace(addr),
                    string.concat("Implementation mismatch for ", vm.toString(addr))
                );
            }

            // The code is a proxy
            assertEq(code, vm.getDeployedCode("Proxy.sol"));

            // All of the defined predeploys have their admin set to the proxy admin
            assertEq(EIP1967Helper.getAdmin(addr), Predeploys.PROXY_ADMIN, "Admin mismatch");
        }
    }
}
