// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { Script } from "forge-std/Script.sol";

import { console2 as console } from "forge-std/console2.sol";
import { stdJson } from "forge-std/StdJson.sol";

import { Safe } from "safe-contracts/Safe.sol";
import { SafeProxyFactory } from "safe-contracts/proxies/SafeProxyFactory.sol";
import { Enum as SafeOps } from "safe-contracts/common/Enum.sol";

import { Deployer } from "./Deployer.sol";
import { DeployConfig } from "./DeployConfig.s.sol";

import { Safe } from "safe-contracts/Safe.sol";
import { SafeProxyFactory } from "safe-contracts/proxies/SafeProxyFactory.sol";
import { ProxyAdmin } from "src/universal/ProxyAdmin.sol";
import { AddressManager } from "src/legacy/AddressManager.sol";
import { Proxy } from "src/universal/Proxy.sol";
import { L1StandardBridge } from "src/L1/L1StandardBridge.sol";
import { OptimismPortal } from "src/L1/OptimismPortal.sol";
import { L1ChugSplashProxy } from "src/legacy/L1ChugSplashProxy.sol";
import { ResolvedDelegateProxy } from "src/legacy/ResolvedDelegateProxy.sol";
import { L1CrossDomainMessenger } from "src/L1/L1CrossDomainMessenger.sol";
import { L2OutputOracle } from "src/L1/L2OutputOracle.sol";
import { OptimismMintableERC20Factory } from "src/universal/OptimismMintableERC20Factory.sol";
import { SystemConfig } from "src/L1/SystemConfig.sol";
import { ResourceMetering } from "src/L1/ResourceMetering.sol";
import { Constants } from "src/libraries/Constants.sol";
import { DisputeGameFactory } from "src/dispute/DisputeGameFactory.sol";
import { FaultDisputeGame } from "src/dispute/FaultDisputeGame.sol";
import { PreimageOracle } from "src/cannon/PreimageOracle.sol";
import { MIPS } from "src/cannon/MIPS.sol";
import { BlockOracle } from "src/dispute/BlockOracle.sol";
import { L1ERC721Bridge } from "src/L1/L1ERC721Bridge.sol";
import { ProtocolVersions, ProtocolVersion } from "src/L1/ProtocolVersions.sol";
import { Predeploys } from "src/libraries/Predeploys.sol";
import { Chains } from "./Chains.sol";

import { IBigStepper } from "src/dispute/interfaces/IBigStepper.sol";
import { IPreimageOracle } from "src/cannon/interfaces/IPreimageOracle.sol";
import { AlphabetVM } from "../test/FaultDisputeGame.t.sol";
import "src/libraries/DisputeTypes.sol";

/// @title Deploy
/// @notice Script used to deploy a bedrock system. The entire system is deployed within the `run` function.
///         To add a new contract to the system, add a public function that deploys that individual contract.
///         Then add a call to that function inside of `run`. Be sure to call the `save` function after each
///         deployment so that hardhat-deploy style artifacts can be generated using a call to `sync()`.
contract Deploy is Deployer {
    DeployConfig cfg;

    /// @notice The name of the script, used to ensure the right deploy artifacts
    ///         are used.
    function name() public pure override returns (string memory name_) {
        name_ = "Deploy";
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

        deployProxies();
        deployImplementations();

        deploySafe();
        transferProxyAdminOwnership(); // to the Safe

        initializeDisputeGameFactory();
        initializeSystemConfig();
        initializeL1StandardBridge();
        initializeL1ERC721Bridge();
        initializeOptimismMintableERC20Factory();
        initializeL1CrossDomainMessenger();
        initializeL2OutputOracle();
        initializeOptimismPortal();
        initializeProtocolVersions();

        setAlphabetFaultGameImplementation();
        setCannonFaultGameImplementation();

        transferDisputeGameFactoryOwnership();
    }

    /// @notice The create2 salt used for deployment of the contract implementations.
    ///         Using this helps to reduce config across networks as the implementation
    ///         addresses will be the same across networks when deployed with create2.
    function implSalt() public returns (bytes32) {
        return keccak256(bytes(vm.envOr("IMPL_SALT", string("ether's phoenix"))));
    }

    /// @notice Modifier that wraps a function in broadcasting.
    modifier broadcast() {
        vm.startBroadcast();
        _;
        vm.stopBroadcast();
    }

    /// @notice Modifier that will only allow a function to be called on devnet.
    modifier onlyDevnet() {
        uint256 chainid = block.chainid;
        if (chainid == Chains.LocalDevnet || chainid == Chains.GethDevnet) {
            _;
        }
    }

    /// @notice Modifier that will only allow a function to be called on a public
    ///         testnet or devnet.
    modifier onlyTestnetOrDevnet() {
        uint256 chainid = block.chainid;
        if (
            chainid == Chains.Goerli || chainid == Chains.Sepolia || chainid == Chains.LocalDevnet
                || chainid == Chains.GethDevnet
        ) {
            _;
        }
    }

    /// @notice Deploy all of the proxies
    function deployProxies() public {
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
        deployProtocolVersionsProxy();

        transferAddressManagerOwnership(); // to the ProxyAdmin
    }

    /// @notice Deploy all of the implementations
    function deployImplementations() public {
        deployOptimismPortal();
        deployL1CrossDomainMessenger();
        deployL2OutputOracle();
        deployOptimismMintableERC20Factory();
        deploySystemConfig();
        deployL1StandardBridge();
        deployL1ERC721Bridge();
        deployDisputeGameFactory();
        deployBlockOracle();
        deployPreimageOracle();
        deployMips();
        deployProtocolVersions();
    }

    // @notice Gets the address of the SafeProxyFactory and Safe singleton for use in deploying a new GnosisSafe.
    function _getSafeFactory() internal returns (SafeProxyFactory safeProxyFactory_, Safe safeSingleton_) {
        // These are they standard create2 deployed contracts. First we'll check if they are deployed,
        // if not we'll deploy new ones, though not at these addresses.
        address safeProxyFactory = 0xa6B71E26C5e0845f74c812102Ca7114b6a896AB2;
        address safeSingleton = 0xd9Db270c1B5E3Bd161E8c8503c55cEABeE709552;

        safeProxyFactory.code.length == 0
            ? safeProxyFactory_ = new SafeProxyFactory()
            : safeProxyFactory_ = SafeProxyFactory(safeProxyFactory);

        safeSingleton.code.length == 0 ? safeSingleton_ = new Safe() : safeSingleton_ = Safe(payable(safeSingleton_));

        save("SafeProxyFactory", address(safeProxyFactory_));
        save("SafeSingleton", address(safeSingleton_));
    }

    /// @notice Deploy the Safe
    function deploySafe() public broadcast returns (address addr_) {
        (SafeProxyFactory safeProxyFactory, Safe safeSingleton) = _getSafeFactory();

        address[] memory signers = new address[](1);
        signers[0] = msg.sender;

        bytes memory initData = abi.encodeWithSelector(
            Safe.setup.selector, signers, 1, address(0), hex"", address(0), address(0), 0, address(0)
        );
        address safe = address(safeProxyFactory.createProxyWithNonce(address(safeSingleton), initData, block.timestamp));

        save("SystemOwnerSafe", address(safe));
        console.log("New SystemOwnerSafe deployed at %s", address(safe));
        addr_ = safe;
    }

    /// @notice Deploy the AddressManager
    function deployAddressManager() public broadcast returns (address addr_) {
        AddressManager manager = new AddressManager();
        require(manager.owner() == msg.sender);

        save("AddressManager", address(manager));
        console.log("AddressManager deployed at %s", address(manager));
        addr_ = address(manager);
    }

    /// @notice Deploy the ProxyAdmin
    function deployProxyAdmin() public broadcast returns (address addr_) {
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
        addr_ = address(admin);
    }

    /// @notice Deploy the L1StandardBridgeProxy
    function deployL1StandardBridgeProxy() public broadcast returns (address addr_) {
        address proxyAdmin = mustGetAddress("ProxyAdmin");
        L1ChugSplashProxy proxy = new L1ChugSplashProxy(proxyAdmin);

        address admin = address(uint160(uint256(vm.load(address(proxy), OWNER_KEY))));
        require(admin == proxyAdmin);

        save("L1StandardBridgeProxy", address(proxy));
        console.log("L1StandardBridgeProxy deployed at %s", address(proxy));
        addr_ = address(proxy);
    }

    /// @notice Deploy the L2OutputOracleProxy
    function deployL2OutputOracleProxy() public broadcast returns (address addr_) {
        address proxyAdmin = mustGetAddress("ProxyAdmin");
        Proxy proxy = new Proxy({
            _admin: proxyAdmin
        });

        address admin = address(uint160(uint256(vm.load(address(proxy), OWNER_KEY))));
        require(admin == proxyAdmin);

        save("L2OutputOracleProxy", address(proxy));
        console.log("L2OutputOracleProxy deployed at %s", address(proxy));
        addr_ = address(proxy);
    }

    /// @notice Deploy the L1CrossDomainMessengerProxy
    function deployL1CrossDomainMessengerProxy() public broadcast returns (address addr_) {
        AddressManager addressManager = AddressManager(mustGetAddress("AddressManager"));
        string memory contractName = "OVM_L1CrossDomainMessenger";
        ResolvedDelegateProxy proxy = new ResolvedDelegateProxy(addressManager, contractName);

        save("L1CrossDomainMessengerProxy", address(proxy));
        console.log("L1CrossDomainMessengerProxy deployed at %s", address(proxy));

        address contractAddr = addressManager.getAddress(contractName);
        if (contractAddr != address(proxy)) {
            addressManager.setAddress(contractName, address(proxy));
        }

        require(addressManager.getAddress(contractName) == address(proxy));

        addr_ = address(proxy);
    }

    /// @notice Deploy the OptimismPortalProxy
    function deployOptimismPortalProxy() public broadcast returns (address addr_) {
        address proxyAdmin = mustGetAddress("ProxyAdmin");
        Proxy proxy = new Proxy({
            _admin: proxyAdmin
        });

        address admin = address(uint160(uint256(vm.load(address(proxy), OWNER_KEY))));
        require(admin == proxyAdmin);

        save("OptimismPortalProxy", address(proxy));
        console.log("OptimismPortalProxy deployed at %s", address(proxy));

        addr_ = address(proxy);
    }

    /// @notice Deploy the OptimismMintableERC20FactoryProxy
    function deployOptimismMintableERC20FactoryProxy() public broadcast returns (address addr_) {
        address proxyAdmin = mustGetAddress("ProxyAdmin");
        Proxy proxy = new Proxy({
            _admin: proxyAdmin
        });

        address admin = address(uint160(uint256(vm.load(address(proxy), OWNER_KEY))));
        require(admin == proxyAdmin);

        save("OptimismMintableERC20FactoryProxy", address(proxy));
        console.log("OptimismMintableERC20FactoryProxy deployed at %s", address(proxy));

        addr_ = address(proxy);
    }

    /// @notice Deploy the L1ERC721BridgeProxy
    function deployL1ERC721BridgeProxy() public broadcast returns (address addr_) {
        address proxyAdmin = mustGetAddress("ProxyAdmin");
        Proxy proxy = new Proxy({
            _admin: proxyAdmin
        });

        address admin = address(uint160(uint256(vm.load(address(proxy), OWNER_KEY))));
        require(admin == proxyAdmin);

        save("L1ERC721BridgeProxy", address(proxy));
        console.log("L1ERC721BridgeProxy deployed at %s", address(proxy));

        addr_ = address(proxy);
    }

    /// @notice Deploy the SystemConfigProxy
    function deploySystemConfigProxy() public broadcast returns (address addr_) {
        address proxyAdmin = mustGetAddress("ProxyAdmin");
        Proxy proxy = new Proxy({
            _admin: proxyAdmin
        });

        address admin = address(uint160(uint256(vm.load(address(proxy), OWNER_KEY))));
        require(admin == proxyAdmin);

        save("SystemConfigProxy", address(proxy));
        console.log("SystemConfigProxy deployed at %s", address(proxy));

        addr_ = address(proxy);
    }

    /// @notice Deploy the DisputeGameFactoryProxy
    function deployDisputeGameFactoryProxy() public onlyDevnet broadcast returns (address addr_) {
        address proxyAdmin = mustGetAddress("ProxyAdmin");
        Proxy proxy = new Proxy({
            _admin: proxyAdmin
        });

        address admin = address(uint160(uint256(vm.load(address(proxy), OWNER_KEY))));
        require(admin == proxyAdmin);

        save("DisputeGameFactoryProxy", address(proxy));
        console.log("DisputeGameFactoryProxy deployed at %s", address(proxy));

        addr_ = address(proxy);
    }

    /// @notice Deploy the ProtocolVersionsProxy
    function deployProtocolVersionsProxy() public onlyTestnetOrDevnet broadcast returns (address addr_) {
        address proxyAdmin = mustGetAddress("ProxyAdmin");
        Proxy proxy = new Proxy({
            _admin: proxyAdmin
        });

        address admin = address(uint160(uint256(vm.load(address(proxy), OWNER_KEY))));
        require(admin == proxyAdmin);

        save("ProtocolVersionsProxy", address(proxy));
        console.log("ProtocolVersionsProxy deployed at %s", address(proxy));

        addr_ = address(proxy);
    }

    /// @notice Deploy the L1CrossDomainMessenger
    function deployL1CrossDomainMessenger() public broadcast returns (address addr_) {
        L1CrossDomainMessenger messenger = new L1CrossDomainMessenger{ salt: implSalt() }();

        require(address(messenger.PORTAL()) == address(0));
        require(address(messenger.portal()) == address(0));

        bytes32 xdmSenderSlot = vm.load(address(messenger), bytes32(uint256(204)));
        require(address(uint160(uint256(xdmSenderSlot))) == Constants.DEFAULT_L2_SENDER);

        save("L1CrossDomainMessenger", address(messenger));
        console.log("L1CrossDomainMessenger deployed at %s", address(messenger));

        addr_ = address(messenger);
    }

    /// @notice Deploy the OptimismPortal
    function deployOptimismPortal() public broadcast returns (address addr_) {
        OptimismPortal portal = new OptimismPortal{ salt: implSalt() }();

        require(address(portal.L2_ORACLE()) == address(0));
        require(portal.GUARDIAN() == address(0));
        require(address(portal.SYSTEM_CONFIG()) == address(0));
        require(portal.paused() == true);

        save("OptimismPortal", address(portal));
        console.log("OptimismPortal deployed at %s", address(portal));

        addr_ = address(portal);
    }

    /// @notice Deploy the L2OutputOracle
    function deployL2OutputOracle() public broadcast returns (address addr_) {
        L2OutputOracle oracle = new L2OutputOracle{ salt: implSalt() }({
            _submissionInterval: cfg.l2OutputOracleSubmissionInterval(),
            _l2BlockTime: cfg.l2BlockTime(),
            _finalizationPeriodSeconds: cfg.finalizationPeriodSeconds()
        });

        require(oracle.SUBMISSION_INTERVAL() == cfg.l2OutputOracleSubmissionInterval());
        require(oracle.submissionInterval() == cfg.l2OutputOracleSubmissionInterval());
        require(oracle.L2_BLOCK_TIME() == cfg.l2BlockTime());
        require(oracle.l2BlockTime() == cfg.l2BlockTime());
        require(oracle.PROPOSER() == address(0));
        require(oracle.proposer() == address(0));
        require(oracle.CHALLENGER() == address(0));
        require(oracle.challenger() == address(0));
        require(oracle.FINALIZATION_PERIOD_SECONDS() == cfg.finalizationPeriodSeconds());
        require(oracle.finalizationPeriodSeconds() == cfg.finalizationPeriodSeconds());
        require(oracle.startingBlockNumber() == 0);
        require(oracle.startingTimestamp() == 0);

        save("L2OutputOracle", address(oracle));
        console.log("L2OutputOracle deployed at %s", address(oracle));

        addr_ = address(oracle);
    }

    /// @notice Deploy the OptimismMintableERC20Factory
    function deployOptimismMintableERC20Factory() public broadcast returns (address addr_) {
        OptimismMintableERC20Factory factory = new OptimismMintableERC20Factory{ salt: implSalt() }();

        require(factory.BRIDGE() == address(0));
        require(factory.bridge() == address(0));

        save("OptimismMintableERC20Factory", address(factory));
        console.log("OptimismMintableERC20Factory deployed at %s", address(factory));

        addr_ = address(factory);
    }

    /// @notice Deploy the DisputeGameFactory
    function deployDisputeGameFactory() public onlyDevnet broadcast returns (address addr_) {
        DisputeGameFactory factory = new DisputeGameFactory{ salt: implSalt() }();
        save("DisputeGameFactory", address(factory));
        console.log("DisputeGameFactory deployed at %s", address(factory));

        addr_ = address(factory);
    }

    /// @notice Deploy the BlockOracle
    function deployBlockOracle() public onlyDevnet broadcast returns (address addr_) {
        BlockOracle oracle = new BlockOracle{ salt: implSalt() }();
        save("BlockOracle", address(oracle));
        console.log("BlockOracle deployed at %s", address(oracle));

        addr_ = address(oracle);
    }

    /// @notice Deploy the ProtocolVersions
    function deployProtocolVersions() public onlyTestnetOrDevnet broadcast returns (address addr_) {
        ProtocolVersions versions = new ProtocolVersions{ salt: implSalt() }();
        save("ProtocolVersions", address(versions));
        console.log("ProtocolVersions deployed at %s", address(versions));

        addr_ = address(versions);
    }

    /// @notice Deploy the PreimageOracle
    function deployPreimageOracle() public onlyDevnet broadcast returns (address addr_) {
        PreimageOracle preimageOracle = new PreimageOracle{ salt: implSalt() }();
        save("PreimageOracle", address(preimageOracle));
        console.log("PreimageOracle deployed at %s", address(preimageOracle));

        addr_ = address(preimageOracle);
    }

    /// @notice Deploy Mips
    function deployMips() public onlyDevnet broadcast returns (address addr_) {
        MIPS mips = new MIPS{ salt: implSalt() }(IPreimageOracle(mustGetAddress("PreimageOracle")));
        save("Mips", address(mips));
        console.log("MIPS deployed at %s", address(mips));

        addr_ = address(mips);
    }

    /// @notice Deploy the SystemConfig
    function deploySystemConfig() public broadcast returns (address addr_) {
        SystemConfig config = new SystemConfig{ salt: implSalt() }();

        require(config.owner() == address(0xdEaD));
        require(config.overhead() == 0);
        require(config.scalar() == 0);
        require(config.unsafeBlockSigner() == address(0));
        require(config.batcherHash() == bytes32(0));
        require(config.gasLimit() == 1);

        ResourceMetering.ResourceConfig memory resourceConfig = config.resourceConfig();
        require(resourceConfig.maxResourceLimit == 1);
        require(resourceConfig.elasticityMultiplier == 1);
        require(resourceConfig.baseFeeMaxChangeDenominator == 2);
        require(resourceConfig.systemTxMaxGas == 0);
        require(resourceConfig.minimumBaseFee == 0);
        require(resourceConfig.maximumBaseFee == 0);

        require(config.l1ERC721Bridge() == address(0));
        require(config.l1StandardBridge() == address(0));
        require(config.l2OutputOracle() == address(0));
        require(config.optimismPortal() == address(0));
        require(config.l1CrossDomainMessenger() == address(0));
        require(config.optimismMintableERC20Factory() == address(0));
        require(config.startBlock() == type(uint256).max);

        save("SystemConfig", address(config));
        console.log("SystemConfig deployed at %s", address(config));

        addr_ = address(config);
    }

    /// @notice Deploy the L1StandardBridge
    function deployL1StandardBridge() public broadcast returns (address addr_) {
        L1StandardBridge bridge = new L1StandardBridge{ salt: implSalt() }();

        require(address(bridge.MESSENGER()) == address(0));
        require(address(bridge.messenger()) == address(0));
        require(address(bridge.OTHER_BRIDGE()) == Predeploys.L2_STANDARD_BRIDGE);
        require(address(bridge.otherBridge()) == Predeploys.L2_STANDARD_BRIDGE);

        save("L1StandardBridge", address(bridge));
        console.log("L1StandardBridge deployed at %s", address(bridge));

        addr_ = address(bridge);
    }

    /// @notice Deploy the L1ERC721Bridge
    function deployL1ERC721Bridge() public broadcast returns (address addr_) {
        L1ERC721Bridge bridge = new L1ERC721Bridge{ salt: implSalt() }();

        require(address(bridge.MESSENGER()) == address(0));
        require(bridge.OTHER_BRIDGE() == Predeploys.L2_ERC721_BRIDGE);

        save("L1ERC721Bridge", address(bridge));
        console.log("L1ERC721Bridge deployed at %s", address(bridge));

        addr_ = address(bridge);
    }

    /// @notice Transfer ownership of the address manager to the ProxyAdmin
    function transferAddressManagerOwnership() public broadcast {
        AddressManager addressManager = AddressManager(mustGetAddress("AddressManager"));
        address owner = addressManager.owner();
        address proxyAdmin = mustGetAddress("ProxyAdmin");
        if (owner != proxyAdmin) {
            addressManager.transferOwnership(proxyAdmin);
            console.log("AddressManager ownership transferred to %s", proxyAdmin);
        }

        require(addressManager.owner() == proxyAdmin);
    }

    /// @notice Make a call from the Safe contract to an arbitrary address with arbitrary data
    function _callViaSafe(address _target, bytes memory _data) internal {
        Safe safe = Safe(mustGetAddress("SystemOwnerSafe"));

        // This is the signature format used the caller is also the signer.
        bytes memory signature = abi.encodePacked(uint256(uint160(msg.sender)), bytes32(0), uint8(1));

        safe.execTransaction({
            to: _target,
            value: 0,
            data: _data,
            operation: SafeOps.Operation.Call,
            safeTxGas: 0,
            baseGas: 0,
            gasPrice: 0,
            gasToken: address(0),
            refundReceiver: payable(address(0)),
            signatures: signature
        });
    }

    /// @notice Call from the Safe contract to the Proxy Admin's upgrade and call method
    function _upgradeAndCallViaSafe(address _proxy, address _implementation, bytes memory _innerCallData) internal {
        Safe safe = Safe(mustGetAddress("SystemOwnerSafe"));
        address proxyAdmin = mustGetAddress("ProxyAdmin");

        bytes memory data =
            abi.encodeCall(ProxyAdmin.upgradeAndCall, (payable(_proxy), _implementation, _innerCallData));

        _callViaSafe({ _target: proxyAdmin, _data: data });
    }

    /// @notice Initialize the DisputeGameFactory
    function initializeDisputeGameFactory() public onlyDevnet broadcast {
        address disputeGameFactoryProxy = mustGetAddress("DisputeGameFactoryProxy");
        address disputeGameFactory = mustGetAddress("DisputeGameFactory");

        _upgradeAndCallViaSafe({
            _proxy: payable(disputeGameFactoryProxy),
            _implementation: disputeGameFactory,
            _innerCallData: abi.encodeCall(DisputeGameFactory.initialize, (msg.sender))
        });

        string memory version = DisputeGameFactory(disputeGameFactoryProxy).version();
        console.log("DisputeGameFactory version: %s", version);
    }

    /// @notice Initialize the SystemConfig
    function initializeSystemConfig() public broadcast {
        address systemConfigProxy = mustGetAddress("SystemConfigProxy");
        address systemConfig = mustGetAddress("SystemConfig");

        bytes32 batcherHash = bytes32(uint256(uint160(cfg.batchSenderAddress())));
        uint256 startBlock = cfg.systemConfigStartBlock();

        _upgradeAndCallViaSafe({
            _proxy: payable(systemConfigProxy),
            _implementation: systemConfig,
            _innerCallData: abi.encodeCall(
                SystemConfig.initialize,
                (
                    cfg.finalSystemOwner(),
                    cfg.gasPriceOracleOverhead(),
                    cfg.gasPriceOracleScalar(),
                    batcherHash,
                    uint64(cfg.l2GenesisBlockGasLimit()),
                    cfg.p2pSequencerAddress(),
                    Constants.DEFAULT_RESOURCE_CONFIG(),
                    startBlock,
                    cfg.batchInboxAddress(),
                    SystemConfig.Addresses({
                        l1CrossDomainMessenger: mustGetAddress("L1CrossDomainMessengerProxy"),
                        l1ERC721Bridge: mustGetAddress("L1ERC721BridgeProxy"),
                        l1StandardBridge: mustGetAddress("L1StandardBridgeProxy"),
                        l2OutputOracle: mustGetAddress("L2OutputOracleProxy"),
                        optimismPortal: mustGetAddress("OptimismPortalProxy"),
                        optimismMintableERC20Factory: mustGetAddress("OptimismMintableERC20FactoryProxy")
                    })
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

        require(config.l1ERC721Bridge() == mustGetAddress("L1ERC721BridgeProxy"));
        require(config.l1StandardBridge() == mustGetAddress("L1StandardBridgeProxy"));
        require(config.l2OutputOracle() == mustGetAddress("L2OutputOracleProxy"));
        require(config.optimismPortal() == mustGetAddress("OptimismPortalProxy"));
        require(config.l1CrossDomainMessenger() == mustGetAddress("L1CrossDomainMessengerProxy"));

        // A non zero start block is an override
        if (startBlock != 0) {
            require(config.startBlock() == startBlock);
        } else {
            require(config.startBlock() == block.number);
        }
    }

    /// @notice Initialize the L1StandardBridge
    function initializeL1StandardBridge() public broadcast {
        ProxyAdmin proxyAdmin = ProxyAdmin(mustGetAddress("ProxyAdmin"));
        address l1StandardBridgeProxy = mustGetAddress("L1StandardBridgeProxy");
        address l1StandardBridge = mustGetAddress("L1StandardBridge");
        address l1CrossDomainMessengerProxy = mustGetAddress("L1CrossDomainMessengerProxy");

        uint256 proxyType = uint256(proxyAdmin.proxyType(l1StandardBridgeProxy));
        if (proxyType != uint256(ProxyAdmin.ProxyType.CHUGSPLASH)) {
            _callViaSafe({
                _target: address(proxyAdmin),
                _data: abi.encodeCall(ProxyAdmin.setProxyType, (l1StandardBridgeProxy, ProxyAdmin.ProxyType.CHUGSPLASH))
            });
        }
        require(uint256(proxyAdmin.proxyType(l1StandardBridgeProxy)) == uint256(ProxyAdmin.ProxyType.CHUGSPLASH));

        _upgradeAndCallViaSafe({
            _proxy: payable(l1StandardBridgeProxy),
            _implementation: l1StandardBridge,
            _innerCallData: abi.encodeCall(
                L1StandardBridge.initialize, (L1CrossDomainMessenger(l1CrossDomainMessengerProxy))
                )
        });

        string memory version = L1StandardBridge(payable(l1StandardBridgeProxy)).version();
        console.log("L1StandardBridge version: %s", version);

        L1StandardBridge bridge = L1StandardBridge(payable(l1StandardBridgeProxy));
        require(address(bridge.MESSENGER()) == l1CrossDomainMessengerProxy);
        require(address(bridge.messenger()) == l1CrossDomainMessengerProxy);
        require(address(bridge.OTHER_BRIDGE()) == Predeploys.L2_STANDARD_BRIDGE);
        require(address(bridge.otherBridge()) == Predeploys.L2_STANDARD_BRIDGE);

        // Ensures that the legacy slot is modified correctly. This will fail
        // during predeployment simulation on OP Mainnet if there is a bug.
        bytes32 slot0 = vm.load(address(bridge), bytes32(uint256(0)));
        require(slot0 == bytes32(uint256(2)));
    }

    /// @notice Initialize the L1ERC721Bridge
    function initializeL1ERC721Bridge() public broadcast {
        address l1ERC721BridgeProxy = mustGetAddress("L1ERC721BridgeProxy");
        address l1ERC721Bridge = mustGetAddress("L1ERC721Bridge");
        address l1CrossDomainMessengerProxy = mustGetAddress("L1CrossDomainMessengerProxy");

        _upgradeAndCallViaSafe({
            _proxy: payable(l1ERC721BridgeProxy),
            _implementation: l1ERC721Bridge,
            _innerCallData: abi.encodeCall(L1ERC721Bridge.initialize, (L1CrossDomainMessenger(l1CrossDomainMessengerProxy)))
        });

        L1ERC721Bridge bridge = L1ERC721Bridge(l1ERC721BridgeProxy);
        string memory version = bridge.version();
        console.log("L1ERC721Bridge version: %s", version);

        require(address(bridge.MESSENGER()) == l1CrossDomainMessengerProxy);
        require(bridge.OTHER_BRIDGE() == Predeploys.L2_ERC721_BRIDGE);
    }

    /// @notice Ininitialize the OptimismMintableERC20Factory
    function initializeOptimismMintableERC20Factory() public broadcast {
        address optimismMintableERC20FactoryProxy = mustGetAddress("OptimismMintableERC20FactoryProxy");
        address optimismMintableERC20Factory = mustGetAddress("OptimismMintableERC20Factory");
        address l1StandardBridgeProxy = mustGetAddress("L1StandardBridgeProxy");

        _upgradeAndCallViaSafe({
            _proxy: payable(optimismMintableERC20FactoryProxy),
            _implementation: optimismMintableERC20Factory,
            _innerCallData: abi.encodeCall(OptimismMintableERC20Factory.initialize, (l1StandardBridgeProxy))
        });

        OptimismMintableERC20Factory factory = OptimismMintableERC20Factory(optimismMintableERC20FactoryProxy);
        string memory version = factory.version();
        console.log("OptimismMintableERC20Factory version: %s", version);

        require(factory.BRIDGE() == l1StandardBridgeProxy);
        require(factory.bridge() == l1StandardBridgeProxy);
    }

    /// @notice initializeL1CrossDomainMessenger
    function initializeL1CrossDomainMessenger() public broadcast {
        ProxyAdmin proxyAdmin = ProxyAdmin(mustGetAddress("ProxyAdmin"));
        address l1CrossDomainMessengerProxy = mustGetAddress("L1CrossDomainMessengerProxy");
        address l1CrossDomainMessenger = mustGetAddress("L1CrossDomainMessenger");
        address optimismPortalProxy = mustGetAddress("OptimismPortalProxy");

        uint256 proxyType = uint256(proxyAdmin.proxyType(l1CrossDomainMessengerProxy));
        if (proxyType != uint256(ProxyAdmin.ProxyType.RESOLVED)) {
            _callViaSafe({
                _target: address(proxyAdmin),
                _data: abi.encodeCall(ProxyAdmin.setProxyType, (l1CrossDomainMessengerProxy, ProxyAdmin.ProxyType.RESOLVED))
            });
        }
        require(uint256(proxyAdmin.proxyType(l1CrossDomainMessengerProxy)) == uint256(ProxyAdmin.ProxyType.RESOLVED));

        string memory contractName = "OVM_L1CrossDomainMessenger";
        string memory implName = proxyAdmin.implementationName(l1CrossDomainMessenger);
        if (keccak256(bytes(contractName)) != keccak256(bytes(implName))) {
            _callViaSafe({
                _target: address(proxyAdmin),
                _data: abi.encodeCall(ProxyAdmin.setImplementationName, (l1CrossDomainMessengerProxy, contractName))
            });
        }
        require(
            keccak256(bytes(proxyAdmin.implementationName(l1CrossDomainMessengerProxy)))
                == keccak256(bytes(contractName))
        );

        _upgradeAndCallViaSafe({
            _proxy: payable(l1CrossDomainMessengerProxy),
            _implementation: l1CrossDomainMessenger,
            _innerCallData: abi.encodeCall(
                L1CrossDomainMessenger.initialize, (OptimismPortal(payable(optimismPortalProxy)))
                )
        });

        L1CrossDomainMessenger messenger = L1CrossDomainMessenger(l1CrossDomainMessengerProxy);
        string memory version = messenger.version();
        console.log("L1CrossDomainMessenger version: %s", version);

        require(address(messenger.PORTAL()) == optimismPortalProxy);
        require(address(messenger.portal()) == optimismPortalProxy);
        bytes32 xdmSenderSlot = vm.load(address(messenger), bytes32(uint256(204)));
        require(address(uint160(uint256(xdmSenderSlot))) == Constants.DEFAULT_L2_SENDER);
    }

    /// @notice Initialize the L2OutputOracle
    function initializeL2OutputOracle() public broadcast {
        address l2OutputOracleProxy = mustGetAddress("L2OutputOracleProxy");
        address l2OutputOracle = mustGetAddress("L2OutputOracle");

        _upgradeAndCallViaSafe({
            _proxy: payable(l2OutputOracleProxy),
            _implementation: l2OutputOracle,
            _innerCallData: abi.encodeCall(
                L2OutputOracle.initialize,
                (
                    cfg.l2OutputOracleStartingBlockNumber(),
                    cfg.l2OutputOracleStartingTimestamp(),
                    cfg.l2OutputOracleProposer(),
                    cfg.l2OutputOracleChallenger()
                )
                )
        });

        L2OutputOracle oracle = L2OutputOracle(l2OutputOracleProxy);
        string memory version = oracle.version();
        console.log("L2OutputOracle version: %s", version);

        require(oracle.SUBMISSION_INTERVAL() == cfg.l2OutputOracleSubmissionInterval());
        require(oracle.submissionInterval() == cfg.l2OutputOracleSubmissionInterval());
        require(oracle.L2_BLOCK_TIME() == cfg.l2BlockTime());
        require(oracle.l2BlockTime() == cfg.l2BlockTime());
        require(oracle.PROPOSER() == cfg.l2OutputOracleProposer());
        require(oracle.proposer() == cfg.l2OutputOracleProposer());
        require(oracle.CHALLENGER() == cfg.l2OutputOracleChallenger());
        require(oracle.challenger() == cfg.l2OutputOracleChallenger());
        require(oracle.FINALIZATION_PERIOD_SECONDS() == cfg.finalizationPeriodSeconds());
        require(oracle.finalizationPeriodSeconds() == cfg.finalizationPeriodSeconds());
        require(oracle.startingBlockNumber() == cfg.l2OutputOracleStartingBlockNumber());
        require(oracle.startingTimestamp() == cfg.l2OutputOracleStartingTimestamp());
    }

    /// @notice Initialize the OptimismPortal
    function initializeOptimismPortal() public broadcast {
        address optimismPortalProxy = mustGetAddress("OptimismPortalProxy");
        address optimismPortal = mustGetAddress("OptimismPortal");
        address l2OutputOracleProxy = mustGetAddress("L2OutputOracleProxy");
        address systemConfigProxy = mustGetAddress("SystemConfigProxy");

        address guardian = cfg.portalGuardian();
        if (guardian.code.length == 0) {
            console.log("Portal guardian has no code: %s", guardian);
        }

        _upgradeAndCallViaSafe({
            _proxy: payable(optimismPortalProxy),
            _implementation: optimismPortal,
            _innerCallData: abi.encodeCall(
                OptimismPortal.initialize,
                (L2OutputOracle(l2OutputOracleProxy), guardian, SystemConfig(systemConfigProxy), false)
                )
        });

        OptimismPortal portal = OptimismPortal(payable(optimismPortalProxy));
        string memory version = portal.version();
        console.log("OptimismPortal version: %s", version);

        require(address(portal.L2_ORACLE()) == l2OutputOracleProxy);
        require(portal.GUARDIAN() == cfg.portalGuardian());
        require(address(portal.SYSTEM_CONFIG()) == systemConfigProxy);
        require(portal.paused() == false);
    }

    function initializeProtocolVersions() public onlyTestnetOrDevnet broadcast {
        address protocolVersionsProxy = mustGetAddress("ProtocolVersionsProxy");
        address protocolVersions = mustGetAddress("ProtocolVersions");

        address finalSystemOwner = cfg.finalSystemOwner();
        uint256 requiredProtocolVersion = cfg.requiredProtocolVersion();
        uint256 recommendedProtocolVersion = cfg.recommendedProtocolVersion();

        _upgradeAndCallViaSafe({
            _proxy: payable(protocolVersionsProxy),
            _implementation: protocolVersions,
            _innerCallData: abi.encodeCall(
                ProtocolVersions.initialize,
                (
                    finalSystemOwner,
                    ProtocolVersion.wrap(requiredProtocolVersion),
                    ProtocolVersion.wrap(recommendedProtocolVersion)
                )
                )
        });

        ProtocolVersions versions = ProtocolVersions(protocolVersionsProxy);
        string memory version = versions.version();
        console.log("ProtocolVersions version: %s", version);

        require(versions.owner() == finalSystemOwner);
        require(ProtocolVersion.unwrap(versions.required()) == requiredProtocolVersion);
        require(ProtocolVersion.unwrap(versions.recommended()) == recommendedProtocolVersion);
    }

    /// @notice Transfer ownership of the ProxyAdmin contract to the final system owner
    function transferProxyAdminOwnership() public broadcast {
        ProxyAdmin proxyAdmin = ProxyAdmin(mustGetAddress("ProxyAdmin"));
        address owner = proxyAdmin.owner();
        address safe = mustGetAddress("SystemOwnerSafe");
        if (owner != safe) {
            proxyAdmin.transferOwnership(safe);
            console.log("ProxyAdmin ownership transferred to Safe at: %s", safe);
        }
    }

    /// @notice Transfer ownership of the DisputeGameFactory contract to the final system owner
    function transferDisputeGameFactoryOwnership() public onlyDevnet broadcast {
        DisputeGameFactory disputeGameFactory = DisputeGameFactory(mustGetAddress("DisputeGameFactoryProxy"));
        address owner = disputeGameFactory.owner();

        address safe = mustGetAddress("SystemOwnerSafe");
        if (owner != safe) {
            disputeGameFactory.transferOwnership(safe);
            console.log("DisputeGameFactory ownership transferred to Safe at: %s", safe);
        }
    }

    /// @notice Sets the implementation for the `FAULT` game type in the `DisputeGameFactory`
    function setCannonFaultGameImplementation() public onlyDevnet broadcast {
        DisputeGameFactory factory = DisputeGameFactory(mustGetAddress("DisputeGameFactoryProxy"));

        Claim mipsAbsolutePrestate;
        if (block.chainid == Chains.LocalDevnet || block.chainid == Chains.GethDevnet) {
            // Fetch the absolute prestate dump
            string memory filePath = string.concat(vm.projectRoot(), "/../../op-program/bin/prestate-proof.json");
            string[] memory commands = new string[](3);
            commands[0] = "bash";
            commands[1] = "-c";
            commands[2] = string.concat("[[ -f ", filePath, " ]] && echo \"present\"");
            if (vm.ffi(commands).length == 0) {
                revert("Cannon prestate dump not found, generate it with `make cannon-prestate` in the monorepo root.");
            }
            commands[2] = string.concat("cat ", filePath, " | jq -r .pre");
            mipsAbsolutePrestate = Claim.wrap(abi.decode(vm.ffi(commands), (bytes32)));
            console.log(
                "[Cannon Dispute Game] Using devnet MIPS Absolute prestate: %s",
                vm.toString(Claim.unwrap(mipsAbsolutePrestate))
            );
        } else {
            console.log(
                "[Cannon Dispute Game] Using absolute prestate from config: %s", cfg.faultGameAbsolutePrestate()
            );
            mipsAbsolutePrestate = Claim.wrap(bytes32(cfg.faultGameAbsolutePrestate()));
        }

        // Set the Cannon FaultDisputeGame implementation in the factory.
        _setFaultGameImplementation(
            factory, GameTypes.FAULT, mipsAbsolutePrestate, IBigStepper(mustGetAddress("Mips")), cfg.faultGameMaxDepth()
        );
    }

    /// @notice Sets the implementation for the alphabet game type in the `DisputeGameFactory`
    function setAlphabetFaultGameImplementation() public onlyDevnet broadcast {
        DisputeGameFactory factory = DisputeGameFactory(mustGetAddress("DisputeGameFactoryProxy"));

        // Set the Alphabet FaultDisputeGame implementation in the factory.
        Claim alphabetAbsolutePrestate = Claim.wrap(bytes32(cfg.faultGameAbsolutePrestate()));
        _setFaultGameImplementation(
            factory,
            GameType.wrap(255),
            alphabetAbsolutePrestate,
            IBigStepper(new AlphabetVM(alphabetAbsolutePrestate)),
            4 // The max game depth of the alphabet game is always 4.
        );
    }

    /// @notice Sets the implementation for the given fault game type in the `DisputeGameFactory`.
    function _setFaultGameImplementation(
        DisputeGameFactory _factory,
        GameType _gameType,
        Claim _absolutePrestate,
        IBigStepper _faultVm,
        uint256 _maxGameDepth
    )
        internal
    {
        if (address(_factory.gameImpls(_gameType)) == address(0)) {
            _factory.setImplementation(
                _gameType,
                new FaultDisputeGame({
                    _gameType: _gameType,
                    _absolutePrestate: _absolutePrestate,
                    _maxGameDepth: _maxGameDepth,
                    _gameDuration: Duration.wrap(uint64(cfg.faultGameMaxDuration())),
                    _vm: _faultVm,
                    _l2oo: L2OutputOracle(mustGetAddress("L2OutputOracleProxy")),
                    _blockOracle: BlockOracle(mustGetAddress("BlockOracle"))
                })
            );

            uint8 rawGameType = GameType.unwrap(_gameType);
            console.log(
                "DisputeGameFactoryProxy: set `FaultDisputeGame` implementation (Backend: %s | GameType: %s)",
                rawGameType == 0 ? "Cannon" : "Alphabet",
                vm.toString(rawGameType)
            );
        } else {
            console.log(
                "[WARN] DisputeGameFactoryProxy: `FaultDisputeGame` implementation already set for game type: %s",
                vm.toString(GameType.unwrap(_gameType))
            );
        }
    }
}
