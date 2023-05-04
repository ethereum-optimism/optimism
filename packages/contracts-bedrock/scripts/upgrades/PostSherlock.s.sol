// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { console } from "forge-std/console.sol";
import { SafeBuilder } from "../universal/SafeBuilder.sol";
import { IMulticall3 } from "forge-std/interfaces/IMulticall3.sol";
import { IGnosisSafe, Enum } from "../interfaces/IGnosisSafe.sol";
import { LibSort } from "../libraries/LibSort.sol";
import { ProxyAdmin } from "../../contracts/universal/ProxyAdmin.sol";
import { Constants } from "../../contracts/libraries/Constants.sol";
import { SystemConfig } from "../../contracts/L1/SystemConfig.sol";
import { ResourceMetering } from "../../contracts/L1/ResourceMetering.sol";
import { Semver } from "../../contracts/universal/Semver.sol";

/**
 * @title PostSherlockL1
 * @notice Upgrade script for upgrading the L1 contracts after the sherlock audit.
 */
contract PostSherlockL1 is SafeBuilder {
    /**
     * @notice Address of the ProxyAdmin, passed in via constructor of `run`.
     */
    ProxyAdmin internal PROXY_ADMIN;

    /**
     * @notice Represents a set of L1 contracts. Used to represent a set of
     *         implementations and also a set of proxies.
     */
    struct ContractSet {
        address L1CrossDomainMessenger;
        address L1StandardBridge;
        address L2OutputOracle;
        address OptimismMintableERC20Factory;
        address OptimismPortal;
        address SystemConfig;
        address L1ERC721Bridge;
    }

    /**
     * @notice A mapping of chainid to a ContractSet of implementations.
     */
    mapping(uint256 => ContractSet) internal implementations;

    /**
     * @notice A mapping of chainid to ContractSet of proxy addresses.
     */
    mapping(uint256 => ContractSet) internal proxies;

    /**
     * @notice The expected versions for the contracts to be upgraded to.
     */
    string constant internal L1CrossDomainMessenger_Version = "1.4.0";
    string constant internal L1StandardBridge_Version = "1.1.0";
    string constant internal L2OutputOracle_Version = "1.3.0";
    string constant internal OptimismMintableERC20Factory_Version = "1.1.0";
    string constant internal OptimismPortal_Version = "1.6.0";
    string constant internal SystemConfig_Version = "1.3.0";
    string constant internal L1ERC721Bridge_Version = "1.1.1";

    /**
     * @notice Place the contract addresses in storage so they can be used when building calldata.
     */
    function setUp() external {
        implementations[GOERLI] = ContractSet({
            L1CrossDomainMessenger: 0x9D1dACf9d9299D17EFFE1aAd559c06bb3Fbf9BC4,
            L1StandardBridge: 0x022Fc3EBAA3d53F8f9b270CC4ABe1B0e4A406253,
            L2OutputOracle: 0x0C2b6590De9D61b37094617b5e6f794Ae118176E,
            OptimismMintableERC20Factory: 0x0EebA1A5da867EB3bc0956f6389d490d0F4b8086,
            OptimismPortal: 0x9e760aBd847E48A56b4a348Cba56Ae7267FeCE80,
            SystemConfig: 0x821EE96B88dAA1569F41cD46b0EA87fA89714b45,
            L1ERC721Bridge: 0x015609dC8cBF8f9947ba571432Bc0d9837c583a4
        });

        proxies[GOERLI] = ContractSet({
            L1CrossDomainMessenger: 0x5086d1eEF304eb5284A0f6720f79403b4e9bE294,
            L1StandardBridge: 0x636Af16bf2f682dD3109e60102b8E1A089FedAa8,
            L2OutputOracle: 0xE6Dfba0953616Bacab0c9A8ecb3a9BBa77FC15c0,
            OptimismMintableERC20Factory: 0x883dcF8B05364083D849D8bD226bC8Cb4c42F9C5,
            OptimismPortal: 0x5b47E1A08Ea6d985D6649300584e6722Ec4B1383,
            SystemConfig: 0xAe851f927Ee40dE99aaBb7461C00f9622ab91d60,
            L1ERC721Bridge: 0x8DD330DdE8D9898d43b4dc840Da27A07dF91b3c9
        });
    }

    /**
     * @notice Follow up assertions to ensure that the script ran to completion.
     */
    function _postCheck() internal override view {
        ContractSet memory prox = getProxies();
        require(_versionHash(prox.L1CrossDomainMessenger) == keccak256(bytes(L1CrossDomainMessenger_Version)), "L1CrossDomainMessenger");
        require(_versionHash(prox.L1StandardBridge) == keccak256(bytes(L1StandardBridge_Version)), "L1StandardBridge");
        require(_versionHash(prox.L2OutputOracle) == keccak256(bytes(L2OutputOracle_Version)), "L2OutputOracle");
        require(_versionHash(prox.OptimismMintableERC20Factory) == keccak256(bytes(OptimismMintableERC20Factory_Version)), "OptimismMintableERC20Factory");
        require(_versionHash(prox.OptimismPortal) == keccak256(bytes(OptimismPortal_Version)), "OptimismPortal");
        require(_versionHash(prox.SystemConfig) == keccak256(bytes(SystemConfig_Version)), "SystemConfig");
        require(_versionHash(prox.L1ERC721Bridge) == keccak256(bytes(L1ERC721Bridge_Version)), "L1ERC721Bridge");

        ResourceMetering.ResourceConfig memory rcfg = SystemConfig(prox.SystemConfig).resourceConfig();
        ResourceMetering.ResourceConfig memory dflt = Constants.DEFAULT_RESOURCE_CONFIG();
        require(keccak256(abi.encode(rcfg)) == keccak256(abi.encode(dflt)));

        // Check that the codehashes of all implementations match the proxies set implementations.
        ContractSet memory impl = getImplementations();
        require(PROXY_ADMIN.getProxyImplementation(prox.L1CrossDomainMessenger).codehash == impl.L1CrossDomainMessenger.codehash, "L1CrossDomainMessenger codehash");
        require(PROXY_ADMIN.getProxyImplementation(prox.L1StandardBridge).codehash == impl.L1StandardBridge.codehash, "L1StandardBridge codehash");
        require(PROXY_ADMIN.getProxyImplementation(prox.L2OutputOracle).codehash == impl.L2OutputOracle.codehash, "L2OutputOracle codehash");
        require(PROXY_ADMIN.getProxyImplementation(prox.OptimismMintableERC20Factory).codehash == impl.OptimismMintableERC20Factory.codehash, "OptimismMintableERC20Factory codehash");
        require(PROXY_ADMIN.getProxyImplementation(prox.OptimismPortal).codehash == impl.OptimismPortal.codehash, "OptimismPortal codehash");
        require(PROXY_ADMIN.getProxyImplementation(prox.SystemConfig).codehash == impl.SystemConfig.codehash, "SystemConfig codehash");
        require(PROXY_ADMIN.getProxyImplementation(prox.L1ERC721Bridge).codehash == impl.L1ERC721Bridge.codehash, "L1ERC721Bridge codehash");
    }

    /**
     * @notice Test coverage of the logic. Should only run on goerli but other chains
     *         could be added.
     */
    function test_script_succeeds() skipWhenNotForking external {
        address _safe;
        address _proxyAdmin;

        if (block.chainid == GOERLI) {
            _safe = 0xBc1233d0C3e6B5d53Ab455cF65A6623F6dCd7e4f;
            _proxyAdmin = 0x01d3670863c3F4b24D7b107900f0b75d4BbC6e0d;
            // Set the proxy admin for the `_postCheck` function
            PROXY_ADMIN = ProxyAdmin(_proxyAdmin);
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

    /**
     * @notice Builds the calldata that the multisig needs to make for the upgrade to happen.
     *         A total of 8 calls are made, 7 upgrade implementations and 1 sets the resource
     *         config to the default value in the SystemConfig contract.
     */
    function buildCalldata(address _proxyAdmin) internal override view returns (bytes memory) {
        IMulticall3.Call3[] memory calls = new IMulticall3.Call3[](8);

        ContractSet memory impl = getImplementations();
        ContractSet memory prox = getProxies();

        // Upgrade the L1CrossDomainMessenger
        calls[0] = IMulticall3.Call3({
            target: _proxyAdmin,
            allowFailure: false,
            callData: abi.encodeCall(
                ProxyAdmin.upgrade,
                (payable(prox.L1CrossDomainMessenger), impl.L1CrossDomainMessenger)
            )
        });

        // Upgrade the L1StandardBridge
        calls[1] = IMulticall3.Call3({
            target: _proxyAdmin,
            allowFailure: false,
            callData: abi.encodeCall(
                ProxyAdmin.upgrade,
                (payable(prox.L1StandardBridge), impl.L1StandardBridge)
            )
        });

        // Upgrade the L2OutputOracle
        calls[2] = IMulticall3.Call3({
            target: _proxyAdmin,
            allowFailure: false,
            callData: abi.encodeCall(
                ProxyAdmin.upgrade,
                (payable(prox.L2OutputOracle), impl.L2OutputOracle)
            )
        });

        // Upgrade the OptimismMintableERC20Factory
        calls[3] = IMulticall3.Call3({
            target: _proxyAdmin,
            allowFailure: false,
            callData: abi.encodeCall(
                ProxyAdmin.upgrade,
                (payable(prox.OptimismMintableERC20Factory), impl.OptimismMintableERC20Factory)
            )
        });

        // Upgrade the OptimismPortal
        calls[4] = IMulticall3.Call3({
            target: _proxyAdmin,
            allowFailure: false,
            callData: abi.encodeCall(
                ProxyAdmin.upgrade,
                (payable(prox.OptimismPortal), impl.OptimismPortal)
            )
        });

        // Upgrade the SystemConfig
        calls[5] = IMulticall3.Call3({
            target: _proxyAdmin,
            allowFailure: false,
            callData: abi.encodeCall(
                ProxyAdmin.upgrade,
                (payable(prox.SystemConfig), impl.SystemConfig)
            )
        });

        // Upgrade the L1ERC721Bridge
        calls[6] = IMulticall3.Call3({
            target: _proxyAdmin,
            allowFailure: false,
            callData: abi.encodeCall(
                ProxyAdmin.upgrade,
                (payable(prox.L1ERC721Bridge), impl.L1ERC721Bridge)
            )
        });

        // Set the default resource config
        ResourceMetering.ResourceConfig memory rcfg = Constants.DEFAULT_RESOURCE_CONFIG();
        calls[7] = IMulticall3.Call3({
            target: prox.SystemConfig,
            allowFailure: false,
            callData: abi.encodeCall(SystemConfig.setResourceConfig, (rcfg))
        });

        return abi.encodeCall(IMulticall3.aggregate3, (calls));
    }

    /**
     * @notice Returns the ContractSet that represents the implementations for a given network.
     */
    function getImplementations() internal view returns (ContractSet memory) {
        ContractSet memory set = implementations[block.chainid];
        require(set.L1CrossDomainMessenger != address(0), "no implementations for this network");
        return set;
    }

    /**
     * @notice Returns the ContractSet that represents the proxies for a given network.
     */
    function getProxies() internal view returns (ContractSet memory) {
        ContractSet memory set = proxies[block.chainid];
        require(set.L1CrossDomainMessenger != address(0), "no proxies for this network");
        return set;
    }
}
