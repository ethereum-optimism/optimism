// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { console2 as console } from "forge-std/console2.sol";
import { SafeBuilder } from "../universal/SafeBuilder.sol";
import { IGnosisSafe, Enum } from "../interfaces/IGnosisSafe.sol";
import { IMulticall3 } from "forge-std/interfaces/IMulticall3.sol";
import { Predeploys } from "../../src/libraries/Predeploys.sol";
import { ProxyAdmin } from "../../src/universal/ProxyAdmin.sol";
import { Deployer } from "../Deployer.sol";

/// @title EASUpgrader
/// @notice Upgrades the EAS predeploys.
contract EASUpgrader is SafeBuilder, Deployer {
    /// @notice The proxy admin predeploy on L2.
    ProxyAdmin immutable PROXY_ADMIN = ProxyAdmin(Predeploys.PROXY_ADMIN);

    /// @notice Represents the EAS contracts predeploys
    struct ContractSet {
        address EAS;
        address SchemaRegistry;
    }

    /// @notice A mapping of chainid to a ContractSet of implementations.
    mapping(uint256 => ContractSet) internal implementations;

    /// @notice A mapping of chainid to ContractSet of proxy addresses.
    mapping(uint256 => ContractSet) internal proxies;

    /// @notice The expected versions for the contracts to be upgraded to.
    string constant internal EAS_Version = "1.0.0";
    string constant internal SchemaRegistry_Version = "1.0.0";

    /// @notice Place the contract addresses in storage so they can be used when building calldata.
    function setUp() public override {
        super.setUp();

        implementations[OP_GOERLI] = ContractSet({
            EAS: getAddress("EAS"),
            SchemaRegistry: getAddress("SchemaRegistry")
        });

        proxies[OP_GOERLI] = ContractSet({
            EAS: Predeploys.EAS,
            SchemaRegistry: Predeploys.SCHEMA_REGISTRY
        });
    }

    /// @notice
    function name() public override pure returns (string memory) {
        return "EASUpgrader";
    }

    /// @notice Follow up assertions to ensure that the script ran to completion.
    function _postCheck() internal override view {
        ContractSet memory prox = getProxies();
        require(_versionHash(prox.EAS) == keccak256(bytes(EAS_Version)), "EAS");
        require(_versionHash(prox.SchemaRegistry) == keccak256(bytes(SchemaRegistry_Version)), "SchemaRegistry");

        // Check that the codehashes of all implementations match the proxies set implementations.
        ContractSet memory impl = getImplementations();
        require(PROXY_ADMIN.getProxyImplementation(prox.EAS).codehash == impl.EAS.codehash);
        require(PROXY_ADMIN.getProxyImplementation(prox.SchemaRegistry).codehash == impl.SchemaRegistry.codehash);
    }

    /// @notice Test coverage of the logic. Should only run on goerli but other chains
    ///         could be added.
    function test_script_succeeds() skipWhenNotForking external {
        address _safe;
        address _proxyAdmin;

        if (block.chainid == OP_GOERLI) {
            _safe = 0xE534ccA2753aCFbcDBCeB2291F596fc60495257e;
            _proxyAdmin = 0x4200000000000000000000000000000000000018;
        }

        require(_safe != address(0) && _proxyAdmin != address(0));

        address[] memory owners = IGnosisSafe(payable(_safe)).getOwners();

        for (uint256 i; i < owners.length; i++) {
            address owner = owners[i];
            vm.startBroadcast(owner);
            bool success = _run(_safe, _proxyAdmin);
            vm.stopBroadcast();

            if (success) {
                console.log("tx success");
                break;
            }
        }

        _postCheck();
    }

    /// @notice Builds the calldata that the multisig needs to make for the upgrade to happen.
    ///         A total of 9 calls are made to the proxy admin to upgrade the implementations
    ///         of the predeploys.
    function buildCalldata(address  _proxyAdmin) internal override view returns (bytes memory) {
        IMulticall3.Call3[] memory calls = new IMulticall3.Call3[](2);

        ContractSet memory impl = getImplementations();
        ContractSet memory prox = getProxies();

        // Upgrade EAS
        calls[0] = IMulticall3.Call3({
            target: _proxyAdmin,
            allowFailure: false,
            callData: abi.encodeCall(
                ProxyAdmin.upgrade,
                (payable(prox.EAS), impl.EAS)
            )
        });

        // Upgrade SchemaRegistry
        calls[1] = IMulticall3.Call3({
            target: _proxyAdmin,
            allowFailure: false,
            callData: abi.encodeCall(
                ProxyAdmin.upgrade,
                (payable(prox.SchemaRegistry), impl.SchemaRegistry)
            )
        });

        return abi.encodeCall(IMulticall3.aggregate3, (calls));
    }

    /// @notice Returns the ContractSet that represents the implementations for a given network.
    function getImplementations() internal view returns (ContractSet memory) {
        ContractSet memory set = implementations[block.chainid];
        require(set.EAS != address(0), "no implementations for this network");
        return set;
    }

    /// @notice Returns the ContractSet that represents the proxies for a given network.
    function getProxies() internal view returns (ContractSet memory) {
        ContractSet memory set = proxies[block.chainid];
        require(set.EAS != address(0), "no proxies for this network");
        return set;
    }
}
