// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { console } from "forge-std/console.sol";
import { SafeBuilder } from "../universal/SafeBuilder.sol";
import { IGnosisSafe, Enum } from "../interfaces/IGnosisSafe.sol";
import { IMulticall3 } from "forge-std/interfaces/IMulticall3.sol";
import { Predeploys } from "../../contracts/libraries/Predeploys.sol";
import { ProxyAdmin } from "../../contracts/universal/ProxyAdmin.sol";

/**
 * @title PostSherlockL2
 * @notice Upgrades the L2 contracts.
 */
contract PostSherlockL2 is SafeBuilder {
    /**
     * @notice The proxy admin predeploy on L2.
     */
    ProxyAdmin constant PROXY_ADMIN = ProxyAdmin(0x4200000000000000000000000000000000000018);

    /**
     * @notice Represents a set of L2 predepploy contracts. Used to represent a set of
     *         implementations and also a set of proxies.
     */
    struct ContractSet {
        address BaseFeeVault;
        address GasPriceOracle;
        address L1Block;
        address L1FeeVault;
        address L2CrossDomainMessenger;
        address L2ERC721Bridge;
        address L2StandardBridge;
        address L2ToL1MessagePasser;
        address SequencerFeeVault;
        address OptimismMintableERC20Factory;
        address OptimismMintableERC721Factory;
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
    string constant internal BaseFeeVault_Version = "1.1.0";
    string constant internal GasPriceOracle_Version = "1.0.0";
    string constant internal L1Block_Version = "1.0.0";
    string constant internal L1FeeVault_Version = "1.1.0";
    string constant internal L2CrossDomainMessenger_Version = "1.4.0";
    string constant internal L2ERC721Bridge_Version = "1.1.0";
    string constant internal L2StandardBridge_Version = "1.1.0";
    string constant internal L2ToL1MessagePasser_Version = "1.0.0";
    string constant internal SequencerFeeVault_Version = "1.1.0";
    string constant internal OptimismMintableERC20Factory_Version = "1.1.0";
    string constant internal OptimismMintableERC721Factory_Version = "1.2.0";

    /**
     * @notice Place the contract addresses in storage so they can be used when building calldata.
     */
    function setUp() external {
        implementations[OP_GOERLI] = ContractSet({
            BaseFeeVault: 0x984eBeFb32A5c2862e92ce90EA0C81Ab69F026B5,
            GasPriceOracle: 0xb43C412454f5D1e58Fe895B1a832B6700ADB5FA7,
            L1Block: 0x6dF83A19647A398d48e77a6835F4A28EB7e2f7c0,
            L1FeeVault: 0xC7d3389726374B8BFF53116585e20A483415f6f6,
            L2CrossDomainMessenger: 0x3305a8110469eB7168870126b26BDAD56067C679,
            L2ERC721Bridge: 0x5Eb0EE8d7f29856F50b5c97F9CF1225491404bF1,
            L2StandardBridge: 0x26A77636eD9A97BBDE9740bed362bFCE5CaB8e10,
            L2ToL1MessagePasser: 0x50CcA47c1e06084459dc83c9E964F4a158cB28Ae,
            SequencerFeeVault: 0x9eE472aB07Aa92bAe20a9d3E2d29beD3248b9075,
            OptimismMintableERC20Factory: 0x3F94732CFd48eE3597d7cEDfb853cfB2De31219c,
            OptimismMintableERC721Factory: 0x13DcfC403eCEF3E8Eab66C00dC64e793dc40Be1d
        });

        proxies[OP_GOERLI] = ContractSet({
            BaseFeeVault: Predeploys.BASE_FEE_VAULT,
            GasPriceOracle: Predeploys.GAS_PRICE_ORACLE,
            L1Block: Predeploys.L1_BLOCK_ATTRIBUTES,
            L1FeeVault: Predeploys.L1_FEE_VAULT,
            L2CrossDomainMessenger: Predeploys.L2_CROSS_DOMAIN_MESSENGER,
            L2ERC721Bridge: Predeploys.L2_ERC721_BRIDGE,
            L2StandardBridge: Predeploys.L2_STANDARD_BRIDGE,
            L2ToL1MessagePasser: Predeploys.L2_TO_L1_MESSAGE_PASSER,
            SequencerFeeVault: Predeploys.SEQUENCER_FEE_WALLET,
            OptimismMintableERC20Factory: Predeploys.OPTIMISM_MINTABLE_ERC20_FACTORY,
            OptimismMintableERC721Factory: Predeploys.OPTIMISM_MINTABLE_ERC721_FACTORY
        });
    }

    /**
     * @notice Follow up assertions to ensure that the script ran to completion.
     */
    function _postCheck() internal override view {
        ContractSet memory prox = getProxies();
        require(_versionHash(prox.BaseFeeVault) == keccak256(bytes(BaseFeeVault_Version)), "BaseFeeVault");
        require(_versionHash(prox.GasPriceOracle) == keccak256(bytes(GasPriceOracle_Version)), "GasPriceOracle");
        require(_versionHash(prox.L1Block) == keccak256(bytes(L1Block_Version)), "L1Block");
        require(_versionHash(prox.L1FeeVault) == keccak256(bytes(L1FeeVault_Version)), "L1FeeVault");
        require(_versionHash(prox.L2CrossDomainMessenger) == keccak256(bytes(L2CrossDomainMessenger_Version)), "L2CrossDomainMessenger");
        require(_versionHash(prox.L2ERC721Bridge) == keccak256(bytes(L2ERC721Bridge_Version)), "L2ERC721Bridge");
        require(_versionHash(prox.L2StandardBridge) == keccak256(bytes(L2StandardBridge_Version)), "L2StandardBridge");
        require(_versionHash(prox.L2ToL1MessagePasser) == keccak256(bytes(L2ToL1MessagePasser_Version)), "L2ToL1MessagePasser");
        require(_versionHash(prox.SequencerFeeVault) == keccak256(bytes(SequencerFeeVault_Version)), "SequencerFeeVault");
        require(_versionHash(prox.OptimismMintableERC20Factory) == keccak256(bytes(OptimismMintableERC20Factory_Version)), "OptimismMintableERC20Factory");
        require(_versionHash(prox.OptimismMintableERC721Factory) == keccak256(bytes(OptimismMintableERC721Factory_Version)), "OptimismMintableERC721Factory");

        // Check that the codehashes of all implementations match the proxies set implementations.
        ContractSet memory impl = getImplementations();
        require(PROXY_ADMIN.getProxyImplementation(prox.BaseFeeVault).codehash == impl.BaseFeeVault.codehash);
        require(PROXY_ADMIN.getProxyImplementation(prox.GasPriceOracle).codehash == impl.GasPriceOracle.codehash);
        require(PROXY_ADMIN.getProxyImplementation(prox.L1Block).codehash == impl.L1Block.codehash);
        require(PROXY_ADMIN.getProxyImplementation(prox.L1FeeVault).codehash == impl.L1FeeVault.codehash);
        require(PROXY_ADMIN.getProxyImplementation(prox.L2CrossDomainMessenger).codehash == impl.L2CrossDomainMessenger.codehash);
        require(PROXY_ADMIN.getProxyImplementation(prox.L2ERC721Bridge).codehash == impl.L2ERC721Bridge.codehash);
        require(PROXY_ADMIN.getProxyImplementation(prox.L2StandardBridge).codehash == impl.L2StandardBridge.codehash);
        require(PROXY_ADMIN.getProxyImplementation(prox.L2ToL1MessagePasser).codehash == impl.L2ToL1MessagePasser.codehash);
        require(PROXY_ADMIN.getProxyImplementation(prox.SequencerFeeVault).codehash == impl.SequencerFeeVault.codehash);
        require(PROXY_ADMIN.getProxyImplementation(prox.OptimismMintableERC20Factory).codehash == impl.OptimismMintableERC20Factory.codehash);
        require(PROXY_ADMIN.getProxyImplementation(prox.OptimismMintableERC721Factory).codehash == impl.OptimismMintableERC721Factory.codehash);
    }

    /**
     * @notice Test coverage of the logic. Should only run on goerli but other chains
     *         could be added.
     */
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

    /**
     * @notice Builds the calldata that the multisig needs to make for the upgrade to happen.
     *         A total of 9 calls are made to the proxy admin to upgrade the implementations
     *         of the predeploys.
     */
    function buildCalldata(address  _proxyAdmin) internal override view returns (bytes memory) {
        IMulticall3.Call3[] memory calls = new IMulticall3.Call3[](11);

        ContractSet memory impl = getImplementations();
        ContractSet memory prox = getProxies();

        // Upgrade the BaseFeeVault
        calls[0] = IMulticall3.Call3({
            target: _proxyAdmin,
            allowFailure: false,
            callData: abi.encodeCall(
                ProxyAdmin.upgrade,
                (payable(prox.BaseFeeVault), impl.BaseFeeVault)
            )
        });

        // Upgrade the GasPriceOracle
        calls[1] = IMulticall3.Call3({
            target: _proxyAdmin,
            allowFailure: false,
            callData: abi.encodeCall(
                ProxyAdmin.upgrade,
                (payable(prox.GasPriceOracle), impl.GasPriceOracle)
            )
        });

        // Upgrade the L1Block predeploy
        calls[2] = IMulticall3.Call3({
            target: _proxyAdmin,
            allowFailure: false,
            callData: abi.encodeCall(
                ProxyAdmin.upgrade,
                (payable(prox.L1Block), impl.L1Block)
            )
        });

        // Upgrade the L1FeeVault
        calls[3] = IMulticall3.Call3({
            target: _proxyAdmin,
            allowFailure: false,
            callData: abi.encodeCall(
                ProxyAdmin.upgrade,
                (payable(prox.L1FeeVault), impl.L1FeeVault)
            )
        });

        // Upgrade the L2CrossDomainMessenger
        calls[4] = IMulticall3.Call3({
            target: _proxyAdmin,
            allowFailure: false,
            callData: abi.encodeCall(
                ProxyAdmin.upgrade,
                (payable(prox.L2CrossDomainMessenger), impl.L2CrossDomainMessenger)
            )
        });

        // Upgrade the L2ERC721Bridge
        calls[5] = IMulticall3.Call3({
            target: _proxyAdmin,
            allowFailure: false,
            callData: abi.encodeCall(
                ProxyAdmin.upgrade,
                (payable(prox.L2ERC721Bridge), impl.L2ERC721Bridge)
            )
        });

        // Upgrade the L2StandardBridge
        calls[6] = IMulticall3.Call3({
            target: _proxyAdmin,
            allowFailure: false,
            callData: abi.encodeCall(
                ProxyAdmin.upgrade,
                (payable(prox.L2StandardBridge), impl.L2StandardBridge)
            )
        });

        // Upgrade the L2ToL1MessagePasser
        calls[7] = IMulticall3.Call3({
            target: _proxyAdmin,
            allowFailure: false,
            callData: abi.encodeCall(
                ProxyAdmin.upgrade,
                (payable(prox.L2ToL1MessagePasser), impl.L2ToL1MessagePasser)
            )
        });

        // Upgrade the SequencerFeeVault
        calls[8] = IMulticall3.Call3({
            target: _proxyAdmin,
            allowFailure: false,
            callData: abi.encodeCall(
                ProxyAdmin.upgrade,
                (payable(prox.SequencerFeeVault), impl.SequencerFeeVault)
            )
        });

        // Upgrade the OptimismMintableERC20Factory
        calls[9] = IMulticall3.Call3({
            target: _proxyAdmin,
            allowFailure: false,
            callData: abi.encodeCall(
                ProxyAdmin.upgrade,
                (payable(prox.OptimismMintableERC20Factory), impl.OptimismMintableERC20Factory)
            )
        });

        // Upgrade the OptimismMintableERC721Factory
        calls[10] = IMulticall3.Call3({
            target: _proxyAdmin,
            allowFailure: false,
            callData: abi.encodeCall(
                ProxyAdmin.upgrade,
                (payable(prox.OptimismMintableERC721Factory), impl.OptimismMintableERC721Factory)
            )
        });

        return abi.encodeCall(IMulticall3.aggregate3, (calls));
    }

    /**
     * @notice Returns the ContractSet that represents the implementations for a given network.
     */
    function getImplementations() internal view returns (ContractSet memory) {
        ContractSet memory set = implementations[block.chainid];
        require(set.BaseFeeVault != address(0), "no implementations for this network");
        return set;
    }

    /**
     * @notice Returns the ContractSet that represents the proxies for a given network.
     */
    function getProxies() internal view returns (ContractSet memory) {
        ContractSet memory set = proxies[block.chainid];
        require(set.BaseFeeVault != address(0), "no proxies for this network");
        return set;
    }
}
