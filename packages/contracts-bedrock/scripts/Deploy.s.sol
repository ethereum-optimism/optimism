// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Script } from "forge-std/Script.sol";
import { Test } from "forge-std/Test.sol";
import { console2 as console } from "forge-std/console2.sol";
import { stdJson } from "forge-std/StdJson.sol";

import { Deployer } from "./Deployer.sol";
import { DeployConfig } from "./DeployConfig.s.sol";

import { ProxyAdmin } from "../contracts/universal/ProxyAdmin.sol";
import { AddressManager } from "../contracts/legacy/AddressManager.sol";
import { Proxy } from "../contracts/universal/Proxy.sol";
import { L1StandardBridge } from "../contracts/L1/L1StandardBridge.sol";
import { OptimismPortal } from "../contracts/L1/OptimismPortal.sol";
import { L1ChugSplashProxy } from "../contracts/legacy/L1ChugSplashProxy.sol";
import { ResolvedDelegateProxy } from "../contracts/legacy/ResolvedDelegateProxy.sol";
import { L1CrossDomainMessenger } from "../contracts/L1/L1CrossDomainMessenger.sol";
import { L2OutputOracle } from "../contracts/L1/L2OutputOracle.sol";
import { OptimismMintableERC20Factory } from "../contracts/universal/OptimismMintableERC20Factory.sol";
import { SystemConfig } from "../contracts/L1/SystemConfig.sol";
import { ResourceMetering } from "../contracts/L1/ResourceMetering.sol";
import { Constants } from "../contracts/libraries/Constants.sol";
import { DisputeGameFactory } from "../contracts/dispute/DisputeGameFactory.sol";
import { L1ERC721Bridge } from "../contracts/L1/L1ERC721Bridge.sol";
import { Predeploys } from "../contracts/libraries/Predeploys.sol";

/// @title Deploy
/// @notice Script used to deploy a bedrock system. The entire system is deployed within the `run` function.
///         To add a new contract to the system, add a public function that deploys that individual contract.
///         Then add a call to that function inside of `run`. Be sure to call the `save` function after each
///         deployment so that hardhat-deploy style artifacts can be generated using a call to `sync()`.
contract Deploy is Deployer {
    DeployConfig cfg;

    /// @notice The name of the script, used to ensure the right deploy artifacts
    ///         are used.
    function name() public pure override returns (string memory) {
        return "Deploy";
    }

    function setUp() public override {
        super.setUp();

        string memory path = string.concat(vm.projectRoot(), "/deploy-config/", deploymentContext, ".json");
        cfg = new DeployConfig(path);

        console.log("Deploying from %s", deployScript);
        console.log("Deployment context: %s", deploymentContext);
    }

    /// @notice Deploy all of the L1 contracts
    function run() public {
        console.log("Deploying L1 system");

        deployAddressManager();
        deployProxyAdmin();

        deployOptimismPortalProxy();
        deployL2OutputOracleProxy();
        deploySystemConfigProxy();
        deployL1StandardBridgeProxy();
        deployL1CrossDomainMessengerProxy();
        deployOptimismMintableERC20FactoryProxy();
        deployL1ERC721BridgeProxy();
        deployDisputeGameFactoryProxy();

        deployOptimismPortal();
        deployL1CrossDomainMessenger();
        deployL2OutputOracle();
        deployOptimismMintableERC20Factory();
        deploySystemConfig();
        deployL1StandardBridge();
        deployL1ERC721Bridge();
        deployDisputeGameFactory();

        transferAddressManagerOwnership();

        initializeDisputeGameFactory();
        initializeSystemConfig();
        initializeL1StandardBridge();
        initializeL1ERC721Bridge();
        initializeOptimismMintableERC20Factory();
        initializeL1CrossDomainMessenger();
        initializeL2OutputOracle();
        initializeOptimismPortal();

        transferProxyAdminOwnership();
    }

    /// @notice Modifier that wraps a function in broadcasting.
    modifier broadcast() {
        vm.startBroadcast();
        _;
        vm.stopBroadcast();
    }

    /// @notice Deploy the AddressManager
    function deployAddressManager() broadcast() public returns (address) {
        AddressManager manager = new AddressManager();
        require(manager.owner() == msg.sender);

        save("AddressManager", address(manager));
        console.log("AddressManager deployed at %s", address(manager));
        return address(manager);
    }

    /// @notice Deploy the ProxyAdmin
    function deployProxyAdmin() broadcast() public returns (address) {
        ProxyAdmin admin = new ProxyAdmin({
            _owner: msg.sender
        });
        require(admin.owner() == msg.sender);

        AddressManager addressManager = AddressManager(mustGetAddress("AddressManager"));
        if (admin.addressManager() != addressManager) {
            admin.setAddressManager(addressManager);
        }

        require(admin.addressManager() == addressManager);

        save("ProxyAdmin", address(admin));
        console.log("ProxyAdmin deployed at %s", address(admin));
        return address(admin);
    }

    /// @notice Deploy the L1StandardBridgeProxy
    function deployL1StandardBridgeProxy() broadcast() public returns (address) {
        address proxyAdmin = mustGetAddress("ProxyAdmin");
        L1ChugSplashProxy proxy = new L1ChugSplashProxy(proxyAdmin);

        address admin = address(uint160(uint256(vm.load(address(proxy), OWNER_KEY))));
        require(admin == proxyAdmin);

        save("L1StandardBridgeProxy", address(proxy));
        console.log("L1StandardBridgeProxy deployed at %s", address(proxy));
        return address(proxy);
    }

    /// @notice Deploy the L2OutputOracleProxy
    function deployL2OutputOracleProxy() broadcast() public returns (address) {
        address proxyAdmin = mustGetAddress("ProxyAdmin");
        Proxy proxy = new Proxy({
            _admin: proxyAdmin
        });

        address admin = address(uint160(uint256(vm.load(address(proxy), OWNER_KEY))));
        require(admin == proxyAdmin);

        save("L2OutputOracleProxy", address(proxy));
        console.log("L2OutputOracleProxy deployed at %s", address(proxy));
        return address(proxy);
    }

    /// @notice Deploy the L1CrossDomainMessengerProxy
    function deployL1CrossDomainMessengerProxy() broadcast() public returns (address) {
        AddressManager addressManager = AddressManager(mustGetAddress("AddressManager"));
        string memory contractName = "OVM_L1CrossDomainMessenger";
        ResolvedDelegateProxy proxy = new ResolvedDelegateProxy(addressManager, contractName);

        save("L1CrossDomainMessengerProxy", address(proxy));
        console.log("L1CrossDomainMessengerProxy deployed at %s", address(proxy));

        address addr = addressManager.getAddress(contractName);
        if (addr != address(proxy)) {
            addressManager.setAddress(contractName, address(proxy));
        }

        require(addressManager.getAddress(contractName) == address(proxy));

        return address(proxy);
    }

    /// @notice Deploy the OptimismPortalProxy
    function deployOptimismPortalProxy() broadcast() public returns (address) {
        address proxyAdmin = mustGetAddress("ProxyAdmin");
        Proxy proxy = new Proxy({
            _admin: proxyAdmin
        });

        address admin = address(uint160(uint256(vm.load(address(proxy), OWNER_KEY))));
        require(admin == proxyAdmin);

        save("OptimismPortalProxy", address(proxy));
        console.log("OptimismPortalProxy deployed at %s", address(proxy));

        return address(proxy);
    }

    /// @notice Deploy the OptimismMintableERC20FactoryProxy
    function deployOptimismMintableERC20FactoryProxy() broadcast() public returns (address) {
        address proxyAdmin = mustGetAddress("ProxyAdmin");
        Proxy proxy = new Proxy({
            _admin: proxyAdmin
        });

        address admin = address(uint160(uint256(vm.load(address(proxy), OWNER_KEY))));
        require(admin == proxyAdmin);

        save("OptimismMintableERC20FactoryProxy", address(proxy));
        console.log("OptimismMintableERC20FactoryProxy deployed at %s", address(proxy));

        return address(proxy);
    }

    /// @notice Deploy the L1ERC721BridgeProxy
    function deployL1ERC721BridgeProxy() broadcast() public returns (address) {
        address proxyAdmin = mustGetAddress("ProxyAdmin");
        Proxy proxy = new Proxy({
            _admin: proxyAdmin
        });

        address admin = address(uint160(uint256(vm.load(address(proxy), OWNER_KEY))));
        require(admin == proxyAdmin);

        save("L1ERC721BridgeProxy", address(proxy));
        console.log("L1ERC721BridgeProxy deployed at %s", address(proxy));

        return address(proxy);
    }

    /// @notice Deploy the SystemConfigProxy
    function deploySystemConfigProxy() broadcast() public returns (address) {
        address proxyAdmin = mustGetAddress("ProxyAdmin");
        Proxy proxy = new Proxy({
            _admin: proxyAdmin
        });

        address admin = address(uint160(uint256(vm.load(address(proxy), OWNER_KEY))));
        require(admin == proxyAdmin);

        save("SystemConfigProxy", address(proxy));
        console.log("SystemConfigProxy deployed at %s", address(proxy));

        return address(proxy);
    }

    /// @notice Deploy the DisputeGameFactoryProxy
    function deployDisputeGameFactoryProxy() broadcast() public returns (address) {
        if (block.chainid == 900) {
            address proxyAdmin = mustGetAddress("ProxyAdmin");
            Proxy proxy = new Proxy({
                _admin: proxyAdmin
            });

            address admin = address(uint160(uint256(vm.load(address(proxy), OWNER_KEY))));
            require(admin == proxyAdmin);

            save("DisputeGameFactoryProxy", address(proxy));
            console.log("DisputeGameFactoryProxy deployed at %s", address(proxy));

            return address(proxy);
        }
        return address(0);
    }

    /// @notice Deploy the L1CrossDomainMessenger
    function deployL1CrossDomainMessenger() broadcast() public returns (address) {
        address portal = mustGetAddress("OptimismPortalProxy");
        L1CrossDomainMessenger messenger = new L1CrossDomainMessenger({
            _portal: OptimismPortal(payable(portal))
        });

        require(address(messenger.PORTAL()) == portal);

        save("L1CrossDomainMessenger", address(messenger));
        console.log("L1CrossDomainMessenger deployed at %s", address(messenger));

        return address(messenger);
    }

    /// @notice Deploy the OptimismPortal
    function deployOptimismPortal() broadcast() public returns (address) {
        address l2OutputOracleProxy = mustGetAddress("L2OutputOracleProxy");
        address systemConfigProxy = mustGetAddress("SystemConfigProxy");

        address guardian = cfg.portalGuardian();
        if (guardian.code.length == 0) {
            console.log("Portal guardian has no code: %s", guardian);
        }

        OptimismPortal portal = new OptimismPortal({
            _l2Oracle: L2OutputOracle(l2OutputOracleProxy),
            _guardian: guardian,
            _paused: true,
            _config: SystemConfig(systemConfigProxy)
        });

        require(address(portal.L2_ORACLE()) == l2OutputOracleProxy);
        require(portal.GUARDIAN() == guardian);
        require(address(portal.SYSTEM_CONFIG()) == systemConfigProxy);
        require(portal.paused() == true);

        save("OptimismPortal", address(portal));
        console.log("OptimismPortal deployed at %s", address(portal));

        return address(portal);
    }

    /// @notice Deploy the L2OutputOracle
    function deployL2OutputOracle() broadcast() public returns (address) {
        L2OutputOracle oracle = new L2OutputOracle({
            _submissionInterval: cfg.l2OutputOracleSubmissionInterval(),
            _l2BlockTime: cfg.l2BlockTime(),
            _startingBlockNumber: cfg.l2OutputOracleStartingBlockNumber(),
            _startingTimestamp: cfg.l2OutputOracleStartingTimestamp(),
            _proposer: cfg.l2OutputOracleProposer(),
            _challenger: cfg.l2OutputOracleChallenger(),
            _finalizationPeriodSeconds: cfg.finalizationPeriodSeconds()
        });

        require(oracle.SUBMISSION_INTERVAL() == cfg.l2OutputOracleSubmissionInterval());
        require(oracle.L2_BLOCK_TIME() == cfg.l2BlockTime());
        require(oracle.PROPOSER() == cfg.l2OutputOracleProposer());
        require(oracle.CHALLENGER() == cfg.l2OutputOracleChallenger());
        require(oracle.FINALIZATION_PERIOD_SECONDS() == cfg.finalizationPeriodSeconds());
        require(oracle.startingBlockNumber() == cfg.l2OutputOracleStartingBlockNumber());
        require(oracle.startingTimestamp() == cfg.l2OutputOracleStartingTimestamp());

        save("L2OutputOracle", address(oracle));
        console.log("L2OutputOracle deployed at %s", address(oracle));

        return address(oracle);
    }

    /// @notice Deploy the OptimismMintableERC20Factory
    function deployOptimismMintableERC20Factory() broadcast() public returns (address) {
        address l1StandardBridgeProxy = mustGetAddress("L1StandardBridgeProxy");
        OptimismMintableERC20Factory factory = new OptimismMintableERC20Factory(l1StandardBridgeProxy);

        require(factory.BRIDGE() == l1StandardBridgeProxy);

        save("OptimismMintableERC20Factory", address(factory));
        console.log("OptimismMintableERC20Factory deployed at %s", address(factory));

        return address(factory);
    }

    /// @notice Deploy the DisputeGameFactory
    function deployDisputeGameFactory() broadcast() public returns (address) {
        if (block.chainid == 900) {
            DisputeGameFactory factory = new DisputeGameFactory();
            save("DisputeGameFactory", address(factory));
            console.log("DisputeGameFactory deployed at %s", address(factory));

            return address(factory);
        }
        return address(0);
    }

    /// @notice Deploy the SystemConfig
    function deploySystemConfig() broadcast() public returns (address) {
        bytes32 batcherHash = bytes32(uint256(uint160(cfg.batchSenderAddress())));

        SystemConfig config = new SystemConfig({
            _owner: cfg.finalSystemOwner(),
            _overhead: cfg.gasPriceOracleOverhead(),
            _scalar: cfg.gasPriceOracleScalar(),
            _batcherHash: batcherHash,
            _gasLimit: uint64(cfg.l2GenesisBlockGasLimit()),
            _unsafeBlockSigner: cfg.p2pSequencerAddress(),
            _config: Constants.DEFAULT_RESOURCE_CONFIG()
        });

        require(config.owner() == cfg.finalSystemOwner());
        require(config.overhead() == cfg.gasPriceOracleOverhead());
        require(config.scalar() == cfg.gasPriceOracleScalar());
        require(config.unsafeBlockSigner() == cfg.p2pSequencerAddress());
        require(config.batcherHash() == batcherHash);

        ResourceMetering.ResourceConfig memory rconfig = Constants.DEFAULT_RESOURCE_CONFIG();
        ResourceMetering.ResourceConfig memory resourceConfig = config.resourceConfig();
        require(resourceConfig.maxResourceLimit == rconfig.maxResourceLimit);
        require(resourceConfig.elasticityMultiplier == rconfig.elasticityMultiplier);
        require(resourceConfig.baseFeeMaxChangeDenominator == rconfig.baseFeeMaxChangeDenominator);
        require(resourceConfig.systemTxMaxGas == rconfig.systemTxMaxGas);
        require(resourceConfig.minimumBaseFee == rconfig.minimumBaseFee);
        require(resourceConfig.maximumBaseFee == rconfig.maximumBaseFee);

        save("SystemConfig", address(config));
        console.log("SystemConfig deployed at %s", address(config));

        return address(config);
    }

    /// @notice Deploy the L1StandardBridge
    function deployL1StandardBridge() broadcast() public returns (address) {
        address l1CrossDomainMessengerProxy = mustGetAddress("L1CrossDomainMessengerProxy");

        L1StandardBridge bridge = new L1StandardBridge({
            _messenger: payable(l1CrossDomainMessengerProxy)
        });

        require(address(bridge.MESSENGER()) == l1CrossDomainMessengerProxy);
        require(address(bridge.OTHER_BRIDGE()) == Predeploys.L2_STANDARD_BRIDGE);

        save("L1StandardBridge", address(bridge));
        console.log("L1StandardBridge deployed at %s", address(bridge));

        return address(bridge);
    }

    /// @notice Deploy the L1ERC721Bridge
    function deployL1ERC721Bridge() broadcast() public returns (address) {
        address l1CrossDomainMessengerProxy = mustGetAddress("L1CrossDomainMessengerProxy");

        L1ERC721Bridge bridge = new L1ERC721Bridge({
            _messenger: l1CrossDomainMessengerProxy,
            _otherBridge: Predeploys.L2_ERC721_BRIDGE
        });

        require(address(bridge.MESSENGER()) == l1CrossDomainMessengerProxy);
        require(bridge.OTHER_BRIDGE() == Predeploys.L2_ERC721_BRIDGE);

        save("L1ERC721Bridge", address(bridge));
        console.log("L1ERC721Bridge deployed at %s", address(bridge));

        return address(bridge);
    }

    /// @notice Transfer ownership of the address manager to the ProxyAdmin
    function transferAddressManagerOwnership() broadcast() public {
        AddressManager addressManager = AddressManager(mustGetAddress("AddressManager"));
        address owner = addressManager.owner();
        address proxyAdmin = mustGetAddress("ProxyAdmin");
        if (owner != proxyAdmin) {
            addressManager.transferOwnership(proxyAdmin);
            console.log("AddressManager ownership transferred to %s", proxyAdmin);
        }

        require(addressManager.owner() == proxyAdmin);
    }

    /// @notice Initialize the DisputeGameFactory
    function initializeDisputeGameFactory() broadcast() public {
        if (block.chainid == 900) {
            ProxyAdmin proxyAdmin = ProxyAdmin(mustGetAddress("ProxyAdmin"));
            address disputeGameFactoryProxy = mustGetAddress("DisputeGameFactoryProxy");
            address disputeGameFactory = mustGetAddress("DisputeGameFactory");

            proxyAdmin.upgradeAndCall({
                _proxy: payable(disputeGameFactoryProxy),
                _implementation: disputeGameFactory,
                _data: abi.encodeCall(
                    DisputeGameFactory.initialize,
                    (cfg.finalSystemOwner())
                )
            });

            string memory version = DisputeGameFactory(disputeGameFactoryProxy).version();
            console.log("DisputeGameFactory version: %s", version);
        }
    }

    /// @notice Initialize the SystemConfig
    function initializeSystemConfig() broadcast() public {
        ProxyAdmin proxyAdmin = ProxyAdmin(mustGetAddress("ProxyAdmin"));
        address systemConfigProxy = mustGetAddress("SystemConfigProxy");
        address systemConfig = mustGetAddress("SystemConfig");

        bytes32 batcherHash = bytes32(uint256(uint160(cfg.batchSenderAddress())));

        proxyAdmin.upgradeAndCall({
            _proxy: payable(systemConfigProxy),
            _implementation: systemConfig,
            _data: abi.encodeCall(
                SystemConfig.initialize,
                (
                    cfg.finalSystemOwner(),
                    cfg.gasPriceOracleOverhead(),
                    cfg.gasPriceOracleScalar(),
                    batcherHash,
                    uint64(cfg.l2GenesisBlockGasLimit()),
                    cfg.p2pSequencerAddress(),
                    Constants.DEFAULT_RESOURCE_CONFIG()
                )
            )
        });

        SystemConfig config = SystemConfig(systemConfigProxy);
        string memory version = config.version();
        console.log("SystemConfig version: %s", version);

        require(config.owner() == cfg.finalSystemOwner());
        require(config.overhead() == cfg.gasPriceOracleOverhead());
        require(config.scalar() == cfg.gasPriceOracleScalar());
        require(config.unsafeBlockSigner() == cfg.p2pSequencerAddress());
        require(config.batcherHash() == batcherHash);

        ResourceMetering.ResourceConfig memory rconfig = Constants.DEFAULT_RESOURCE_CONFIG();
        ResourceMetering.ResourceConfig memory resourceConfig = config.resourceConfig();
        require(resourceConfig.maxResourceLimit == rconfig.maxResourceLimit);
        require(resourceConfig.elasticityMultiplier == rconfig.elasticityMultiplier);
        require(resourceConfig.baseFeeMaxChangeDenominator == rconfig.baseFeeMaxChangeDenominator);
        require(resourceConfig.systemTxMaxGas == rconfig.systemTxMaxGas);
        require(resourceConfig.minimumBaseFee == rconfig.minimumBaseFee);
        require(resourceConfig.maximumBaseFee == rconfig.maximumBaseFee);

    }

    /// @notice Initialize the L1StandardBridge
    function initializeL1StandardBridge() broadcast() public {
        ProxyAdmin proxyAdmin = ProxyAdmin(mustGetAddress("ProxyAdmin"));
        address l1StandardBridgeProxy = mustGetAddress("L1StandardBridgeProxy");
        address l1StandardBridge = mustGetAddress("L1StandardBridge");
        address l1CrossDomainMessengerProxy = mustGetAddress("L1CrossDomainMessengerProxy");

        uint256 proxyType = uint256(proxyAdmin.proxyType(l1StandardBridgeProxy));
        if (proxyType != uint256(ProxyAdmin.ProxyType.CHUGSPLASH)) {
            proxyAdmin.setProxyType(l1StandardBridgeProxy, ProxyAdmin.ProxyType.CHUGSPLASH);
        }
        require(uint256(proxyAdmin.proxyType(l1StandardBridgeProxy)) == uint256(ProxyAdmin.ProxyType.CHUGSPLASH));

        proxyAdmin.upgrade({
            _proxy: payable(l1StandardBridgeProxy),
            _implementation: l1StandardBridge
        });

        string memory version = L1StandardBridge(payable(l1StandardBridgeProxy)).version();
        console.log("L1StandardBridge version: %s", version);

        L1StandardBridge bridge = L1StandardBridge(payable(l1StandardBridgeProxy));
        require(address(bridge.MESSENGER()) == l1CrossDomainMessengerProxy);
        require(address(bridge.OTHER_BRIDGE()) == Predeploys.L2_STANDARD_BRIDGE);

    }

    /// @notice Initialize the L1ERC721Bridge
    function initializeL1ERC721Bridge() broadcast() public {
        ProxyAdmin proxyAdmin = ProxyAdmin(mustGetAddress("ProxyAdmin"));
        address l1ERC721BridgeProxy = mustGetAddress("L1ERC721BridgeProxy");
        address l1ERC721Bridge = mustGetAddress("L1ERC721Bridge");
        address l1CrossDomainMessengerProxy = mustGetAddress("L1CrossDomainMessengerProxy");

        proxyAdmin.upgrade({
            _proxy: payable(l1ERC721BridgeProxy),
            _implementation: l1ERC721Bridge
        });

        L1ERC721Bridge bridge = L1ERC721Bridge(l1ERC721BridgeProxy);
        string memory version = bridge.version();
        console.log("L1ERC721Bridge version: %s", version);

        require(address(bridge.MESSENGER()) == l1CrossDomainMessengerProxy);
        require(bridge.OTHER_BRIDGE() == Predeploys.L2_ERC721_BRIDGE);
    }

    /// @notice Ininitialize the OptimismMintableERC20Factory
    function initializeOptimismMintableERC20Factory() broadcast() public {
        ProxyAdmin proxyAdmin = ProxyAdmin(mustGetAddress("ProxyAdmin"));
        address optimismMintableERC20FactoryProxy = mustGetAddress("OptimismMintableERC20FactoryProxy");
        address optimismMintableERC20Factory = mustGetAddress("OptimismMintableERC20Factory");
        address l1StandardBridgeProxy = mustGetAddress("L1StandardBridgeProxy");

        proxyAdmin.upgrade({
            _proxy: payable(optimismMintableERC20FactoryProxy),
            _implementation: optimismMintableERC20Factory
        });

        OptimismMintableERC20Factory factory = OptimismMintableERC20Factory(optimismMintableERC20FactoryProxy);
        string memory version = factory.version();
        console.log("OptimismMintableERC20Factory version: %s", version);

        require(factory.BRIDGE() == l1StandardBridgeProxy);
    }

    /// @notice initializeL1CrossDomainMessenger
    function initializeL1CrossDomainMessenger() broadcast() public {
        ProxyAdmin proxyAdmin = ProxyAdmin(mustGetAddress("ProxyAdmin"));
        address l1CrossDomainMessengerProxy = mustGetAddress("L1CrossDomainMessengerProxy");
        address l1CrossDomainMessenger = mustGetAddress("L1CrossDomainMessenger");
        address optimismPortalProxy = mustGetAddress("OptimismPortalProxy");

        uint256 proxyType = uint256(proxyAdmin.proxyType(l1CrossDomainMessengerProxy));
        if (proxyType != uint256(ProxyAdmin.ProxyType.RESOLVED)) {
            proxyAdmin.setProxyType(l1CrossDomainMessengerProxy, ProxyAdmin.ProxyType.RESOLVED);
        }
        require(uint256(proxyAdmin.proxyType(l1CrossDomainMessengerProxy)) == uint256(ProxyAdmin.ProxyType.RESOLVED));

        string memory contractName = "OVM_L1CrossDomainMessenger";
        string memory implName = proxyAdmin.implementationName(l1CrossDomainMessenger);
        if (keccak256(bytes(contractName)) != keccak256(bytes(implName))) {
            proxyAdmin.setImplementationName(l1CrossDomainMessengerProxy, contractName);
        }
        require(
            keccak256(bytes(proxyAdmin.implementationName(l1CrossDomainMessengerProxy))) == keccak256(bytes(contractName))
        );

        proxyAdmin.upgradeAndCall({
            _proxy: payable(l1CrossDomainMessengerProxy),
            _implementation: l1CrossDomainMessenger,
            _data: abi.encodeCall(L1CrossDomainMessenger.initialize, ())
        });

        L1CrossDomainMessenger messenger = L1CrossDomainMessenger(l1CrossDomainMessengerProxy);
        string memory version = messenger.version();
        console.log("L1CrossDomainMessenger version: %s", version);

        require(address(messenger.PORTAL()) == optimismPortalProxy);
    }

    /// @notice Initialize the L2OutputOracle
    function initializeL2OutputOracle() broadcast() public {
        ProxyAdmin proxyAdmin = ProxyAdmin(mustGetAddress("ProxyAdmin"));
        address l2OutputOracleProxy = mustGetAddress("L2OutputOracleProxy");
        address l2OutputOracle = mustGetAddress("L2OutputOracle");

        proxyAdmin.upgradeAndCall({
            _proxy: payable(l2OutputOracleProxy),
            _implementation: l2OutputOracle,
            _data: abi.encodeCall(
                L2OutputOracle.initialize,
                (
                    cfg.l2OutputOracleStartingBlockNumber(),
                    cfg.l2OutputOracleStartingTimestamp()
                )
            )
        });

        L2OutputOracle oracle = L2OutputOracle(l2OutputOracleProxy);
        string memory version = oracle.version();
        console.log("L2OutputOracle version: %s", version);

        require(oracle.SUBMISSION_INTERVAL() == cfg.l2OutputOracleSubmissionInterval());
        require(oracle.L2_BLOCK_TIME() == cfg.l2BlockTime());
        require(oracle.PROPOSER() == cfg.l2OutputOracleProposer());
        require(oracle.CHALLENGER() == cfg.l2OutputOracleChallenger());
        require(oracle.FINALIZATION_PERIOD_SECONDS() == cfg.finalizationPeriodSeconds());
        require(oracle.startingBlockNumber() == cfg.l2OutputOracleStartingBlockNumber());
        require(oracle.startingTimestamp() == cfg.l2OutputOracleStartingTimestamp());
    }

    /// @notice Initialize the OptimismPortal
    function initializeOptimismPortal() broadcast() public {
        ProxyAdmin proxyAdmin = ProxyAdmin(mustGetAddress("ProxyAdmin"));
        address optimismPortalProxy = mustGetAddress("OptimismPortalProxy");
        address optimismPortal = mustGetAddress("OptimismPortal");
        address l2OutputOracleProxy = mustGetAddress("L2OutputOracleProxy");
        address systemConfigProxy = mustGetAddress("SystemConfigProxy");

        proxyAdmin.upgradeAndCall({
            _proxy: payable(optimismPortalProxy),
            _implementation: optimismPortal,
            _data: abi.encodeCall(OptimismPortal.initialize, (false))
        });

        OptimismPortal portal = OptimismPortal(payable(optimismPortalProxy));
        string memory version = portal.version();
        console.log("OptimismPortal version: %s", version);

        require(address(portal.L2_ORACLE()) == l2OutputOracleProxy);
        require(portal.GUARDIAN() == cfg.portalGuardian());
        require(address(portal.SYSTEM_CONFIG()) == systemConfigProxy);
        require(portal.paused() == false);
    }

    /// @notice Transfer ownership of the ProxyAdmin contract to the final system owner
    function transferProxyAdminOwnership() broadcast() public {
        ProxyAdmin proxyAdmin = ProxyAdmin(mustGetAddress("ProxyAdmin"));
        address owner = proxyAdmin.owner();
        address finalSystemOwner = cfg.finalSystemOwner();
        if (owner != finalSystemOwner) {
            proxyAdmin.transferOwnership(finalSystemOwner);
            console.log("ProxyAdmin ownership transferred to: %s", finalSystemOwner);
        }
    }
}

