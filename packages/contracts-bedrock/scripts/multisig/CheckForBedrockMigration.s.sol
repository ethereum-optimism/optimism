// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { console2 } from "forge-std/console2.sol";
import { Script } from "forge-std/Script.sol";
import { StdAssertions } from "forge-std/StdAssertions.sol";

/**
 * @title BedrockMigrationChecker
 * @notice A script to check safety of multisig operations for Bedrock.
 *         The usage is as follows:
 *         $ forge script scripts/CheckForBedrockMigration.s.sol \
 *             --rpc-url $ETH_RPC_URL
 */

contract BedrockMigrationChecker is Script, StdAssertions {

    struct ContractSet {
        // Please keep these sorted by name.
        address AddressManager;
        address L1CrossDomainMessengerImpl;
        address L1CrossDomainMessengerProxy;
        address L1ERC721BridgeImpl;
        address L1ERC721BridgeProxy;
        address L1ProxyAdmin;
        address L1StandardBridgeImpl;
        address L1StandardBridgeProxy;
        address L1UpgradeKey;
        address L2OutputOracleImpl;
        address L2OutputOracleProxy;
        address OptimismMintableERC20FactoryImpl;
        address OptimismMintableERC20FactoryProxy;
        address OptimismPortalImpl;
        address OptimismPortalProxy;
        address PortalSender;
        address SystemConfigProxy;
        address SystemDictatorImpl;
        address SystemDictatorProxy;
    }

    /**
     * @notice The entrypoint function.
     */
    function run() external {
        string memory bedrockJsonDir = vm.envString("BEDROCK_JSON_DIR");
        console2.log("BEDROCK_JSON_DIR = %s", bedrockJsonDir);
        ContractSet memory contracts = getContracts(bedrockJsonDir);
        checkAddressManager(contracts);
        checkL1CrossDomainMessengerImpl(contracts);
        checkL1CrossDomainMessengerProxy(contracts);
        checkL1ERC721BridgeImpl(contracts);
        checkL1ERC721BridgeProxy(contracts);
        checkL1ProxyAdmin(contracts);
        checkL1StandardBridgeImpl(contracts);
        checkL1StandardBridgeProxy(contracts);
        checkL1UpgradeKey(contracts);
        checkL2OutputOracleImpl(contracts);
        checkL2OutputOracleProxy(contracts);
        checkOptimismMintableERC20FactoryImpl(contracts);
        checkOptimismMintableERC20FactoryProxy(contracts);
        checkOptimismPortalImpl(contracts);
        checkOptimismPortalProxy(contracts);
        checkPortalSender(contracts);
        checkSystemConfigProxy(contracts);
        checkSystemDictatorImpl(contracts);
        checkSystemDictatorProxy(contracts);
    }

    function checkAddressManager(ContractSet memory contracts) internal {
        console2.log("Checking AddressManager %s", contracts.AddressManager);
        checkAddressIsExpected(contracts.L1UpgradeKey, contracts.AddressManager, "owner()");
    }

    function checkL1CrossDomainMessengerImpl(ContractSet memory contracts) internal {
        console2.log("Checking L1CrossDomainMessenger %s", contracts.L1CrossDomainMessengerImpl);
        checkAddressIsExpected(contracts.OptimismPortalProxy, contracts.L1CrossDomainMessengerImpl, "PORTAL()");
    }

    function checkL1CrossDomainMessengerProxy(ContractSet memory contracts) internal {
        console2.log("Checking L1CrossDomainMessengerProxy %s", contracts.L1CrossDomainMessengerProxy);
        checkAddressIsExpected(contracts.L1UpgradeKey, contracts.L1CrossDomainMessengerProxy, "owner()");
        checkAddressIsExpected(contracts.AddressManager, contracts.L1CrossDomainMessengerProxy, "libAddressManager()");
    }

    function checkL1ERC721BridgeImpl(ContractSet memory contracts) internal {
        console2.log("Checking L1ERC721Bridge %s", contracts.L1ERC721BridgeImpl);
        checkAddressIsExpected(contracts.L1CrossDomainMessengerProxy, contracts.L1ERC721BridgeImpl, "messenger()");
    }

    function checkL1ERC721BridgeProxy(ContractSet memory contracts) internal {
        console2.log("Checking L1ERC721BridgeProxy %s", contracts.L1ERC721BridgeProxy);
        checkAddressIsExpected(contracts.L1UpgradeKey, contracts.L1ERC721BridgeProxy, "admin()");
        checkAddressIsExpected(contracts.L1CrossDomainMessengerProxy, contracts.L1ERC721BridgeProxy, "messenger()");
    }

    function checkL1ProxyAdmin(ContractSet memory contracts) internal {
        console2.log("Checking L1ProxyAdmin %s", contracts.L1ProxyAdmin);
        checkAddressIsExpected(contracts.L1UpgradeKey, contracts.L1ProxyAdmin, "owner()");
    }

    function checkL1StandardBridgeImpl(ContractSet memory contracts) internal {
        console2.log("Checking L1StandardBridge %s", contracts.L1StandardBridgeImpl);
        checkAddressIsExpected(contracts.L1CrossDomainMessengerProxy, contracts.L1StandardBridgeImpl, "messenger()");
    }

    function checkL1StandardBridgeProxy(ContractSet memory contracts) internal {
        console2.log("Checking L1StandardBridgeProxy %s", contracts.L1StandardBridgeProxy);
        checkAddressIsExpected(contracts.L1UpgradeKey, contracts.L1StandardBridgeProxy, "getOwner()");
        checkAddressIsExpected(contracts.L1CrossDomainMessengerProxy, contracts.L1StandardBridgeProxy, "messenger()");
    }

    function checkL1UpgradeKey(ContractSet memory contracts) internal {
        console2.log("Checking L1UpgradeKeyAddress %s", contracts.L1UpgradeKey);
        // No need to check anything here, so just printing the address.
    }

    function checkL2OutputOracleImpl(ContractSet memory contracts) internal {
        console2.log("Checking L2OutputOracle %s", contracts.L2OutputOracleImpl);
        checkAddressIsExpected(contracts.L1UpgradeKey, contracts.L2OutputOracleImpl, "CHALLENGER()");
        // 604800 seconds = 7 days, reusing the logic in
        // checkAddressIsExpected for simplicity.
        checkAddressIsExpected(address(604800), contracts.L2OutputOracleImpl, "FINALIZATION_PERIOD_SECONDS()");
    }

    function checkL2OutputOracleProxy(ContractSet memory contracts) internal {
        console2.log("Checking L2OutputOracleProxy %s", contracts.L2OutputOracleProxy);
        checkAddressIsExpected(contracts.L1ProxyAdmin, contracts.L2OutputOracleProxy, "admin()");
    }

    function checkOptimismMintableERC20FactoryImpl(ContractSet memory contracts) internal {
        console2.log("Checking OptimismMintableERC20Factory %s", contracts.OptimismMintableERC20FactoryImpl);
        checkAddressIsExpected(contracts.L1StandardBridgeProxy, contracts.OptimismMintableERC20FactoryImpl, "BRIDGE()");
    }

    function checkOptimismMintableERC20FactoryProxy(ContractSet memory contracts) internal {
        console2.log("Checking OptimismMintableERC20FactoryProxy %s", contracts.OptimismMintableERC20FactoryProxy);
        checkAddressIsExpected(contracts.L1ProxyAdmin, contracts.OptimismMintableERC20FactoryProxy, "admin()");
    }

    function checkOptimismPortalImpl(ContractSet memory contracts) internal {
        console2.log("Checking OptimismPortal %s", contracts.OptimismPortalImpl);
        checkAddressIsExpected(contracts.L2OutputOracleProxy, contracts.OptimismPortalImpl, "L2_ORACLE()");
    }

    function checkOptimismPortalProxy(ContractSet memory contracts) internal {
        console2.log("Checking OptimismPortalProxy %s", contracts.OptimismPortalProxy);
        checkAddressIsExpected(contracts.L1ProxyAdmin, contracts.OptimismPortalProxy, "admin()");
    }

    function checkPortalSender(ContractSet memory contracts) internal {
        console2.log("Checking PortalSender %s", contracts.PortalSender);
        checkAddressIsExpected(contracts.OptimismPortalProxy, contracts.PortalSender, "PORTAL()");
    }

    function checkSystemConfigProxy(ContractSet memory contracts) internal {
        console2.log("Checking SystemConfigProxy %s", contracts.SystemConfigProxy);
        checkAddressIsExpected(contracts.L1ProxyAdmin, contracts.SystemConfigProxy, "admin()");
    }

    function checkSystemDictatorImpl(ContractSet memory contracts) internal {
        console2.log("Checking SystemDictator %s", contracts.SystemDictatorImpl);
        checkAddressIsExpected(address(0), contracts.SystemDictatorImpl, "owner()");
    }

    function checkSystemDictatorProxy(ContractSet memory contracts) internal {
        console2.log("Checking SystemDictatorProxy %s", contracts.SystemDictatorProxy);
        checkAddressIsExpected(contracts.SystemDictatorImpl, contracts.SystemDictatorProxy, "implementation()");
        checkAddressIsExpected(contracts.L1UpgradeKey, contracts.SystemDictatorProxy, "owner()");
        checkAddressIsExpected(contracts.L1UpgradeKey, contracts.SystemDictatorProxy, "admin()");
    }

    function checkAddressIsExpected(address expectedAddr, address contractAddr, string memory signature) internal {
        address actual = getAddressFromCall(contractAddr, signature);
        if (expectedAddr != actual) {
            console2.log("  !! Error: %s != %s.%s, ", expectedAddr, contractAddr, signature);
            console2.log("           which is %s", actual);
        } else {
            console2.log("  -- Success: %s == %s.%s.", expectedAddr, contractAddr, signature);
        }
    }

    function getAddressFromCall(address contractAddr, string memory signature) internal returns (address) {
        vm.prank(address(0));
        (bool success, bytes memory addrBytes) = contractAddr.staticcall(abi.encodeWithSignature(signature));
        if (!success) {
            console2.log("  !! Error calling %s.%s", contractAddr, signature);
            return address(0);
        }
        return abi.decode(addrBytes, (address));
    }

    function getContracts(string memory bedrockJsonDir) internal returns (ContractSet memory) {
        return ContractSet({
            AddressManager: getAddressFromJson(string.concat(bedrockJsonDir, "/AddressManager.json")),
                    L1CrossDomainMessengerImpl: getAddressFromJson(string.concat(bedrockJsonDir, "/L1CrossDomainMessenger.json")),
                    L1CrossDomainMessengerProxy: getAddressFromJson(string.concat(bedrockJsonDir, "/L1CrossDomainMessengerProxy.json")),
                L1ERC721BridgeImpl: getAddressFromJson(string.concat(bedrockJsonDir, "/L1ERC721Bridge.json")),
                L1ERC721BridgeProxy: getAddressFromJson(string.concat(bedrockJsonDir, "/L1ERC721BridgeProxy.json")),
                L1ProxyAdmin: getAddressFromJson(string.concat(bedrockJsonDir, "/ProxyAdmin.json")),
                L1StandardBridgeImpl: getAddressFromJson(string.concat(bedrockJsonDir, "/L1StandardBridge.json")),
                L1StandardBridgeProxy: getAddressFromJson(string.concat(bedrockJsonDir, "/L1StandardBridgeProxy.json")),
                L1UpgradeKey: vm.envAddress("L1_UPGRADE_KEY"),
                L2OutputOracleImpl: getAddressFromJson(string.concat(bedrockJsonDir, "/L2OutputOracle.json")),
                L2OutputOracleProxy: getAddressFromJson(string.concat(bedrockJsonDir, "/L2OutputOracleProxy.json")),
                OptimismMintableERC20FactoryImpl: getAddressFromJson(string.concat(bedrockJsonDir, "/OptimismMintableERC20Factory.json")),
                OptimismMintableERC20FactoryProxy: getAddressFromJson(string.concat(bedrockJsonDir, "/OptimismMintableERC20FactoryProxy.json")),
                OptimismPortalImpl: getAddressFromJson(string.concat(bedrockJsonDir, "/OptimismPortal.json")),
                OptimismPortalProxy: getAddressFromJson(string.concat(bedrockJsonDir, "/OptimismPortalProxy.json")),
                PortalSender: getAddressFromJson(string.concat(bedrockJsonDir, "/PortalSender.json")),
                SystemConfigProxy: getAddressFromJson(string.concat(bedrockJsonDir, "/SystemConfigProxy.json")),
                SystemDictatorImpl: getAddressFromJson(string.concat(bedrockJsonDir, "/SystemDictator.json")),
                SystemDictatorProxy: getAddressFromJson(string.concat(bedrockJsonDir, "/SystemDictatorProxy.json"))
            });
    }

    function getAddressFromJson(string memory jsonPath) internal returns (address) {
        string memory json = vm.readFile(jsonPath);
        return vm.parseJsonAddress(json, ".address");
    }

}
