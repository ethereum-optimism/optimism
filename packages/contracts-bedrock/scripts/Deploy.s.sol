// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { VmSafe } from "forge-std/Vm.sol";
import { Script } from "forge-std/Script.sol";

import { console2 as console } from "forge-std/console2.sol";
import { stdJson } from "forge-std/StdJson.sol";

import { Safe } from "safe-contracts/Safe.sol";
import { SafeProxyFactory } from "safe-contracts/proxies/SafeProxyFactory.sol";
import { Enum as SafeOps } from "safe-contracts/common/Enum.sol";

import { Deployer } from "scripts/Deployer.sol";
import { DeployConfig } from "scripts/DeployConfig.s.sol";

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
import { SuperchainConfig } from "src/L1/SuperchainConfig.sol";
import { SystemConfig } from "src/L1/SystemConfig.sol";
import { ResourceMetering } from "src/L1/ResourceMetering.sol";
import { Constants } from "src/libraries/Constants.sol";
import { DisputeGameFactory } from "src/dispute/DisputeGameFactory.sol";
import { FaultDisputeGame } from "src/dispute/FaultDisputeGame.sol";
import { OutputBisectionGame } from "src/dispute/OutputBisectionGame.sol";
import { PreimageOracle } from "src/cannon/PreimageOracle.sol";
import { MIPS } from "src/cannon/MIPS.sol";
import { BlockOracle } from "src/dispute/BlockOracle.sol";
import { L1ERC721Bridge } from "src/L1/L1ERC721Bridge.sol";
import { ProtocolVersions, ProtocolVersion } from "src/L1/ProtocolVersions.sol";
import { StorageSetter } from "src/universal/StorageSetter.sol";
import { Predeploys } from "src/libraries/Predeploys.sol";
import { Chains } from "scripts/Chains.sol";

import { IBigStepper } from "src/dispute/interfaces/IBigStepper.sol";
import { IPreimageOracle } from "src/cannon/interfaces/IPreimageOracle.sol";
import { AlphabetVM } from "test/mocks/AlphabetVM.sol";
import { AlphabetVM2 } from "test/mocks/AlphabetVM2.sol";
import "src/libraries/DisputeTypes.sol";
import { ChainAssertions } from "scripts/ChainAssertions.sol";
import { Types } from "scripts/Types.sol";
import { LibStateDiff } from "scripts/libraries/LibStateDiff.sol";

/// @title Deploy
/// @notice Script used to deploy a bedrock system. The entire system is deployed within the `run` function.
///         To add a new contract to the system, add a public function that deploys that individual contract.
///         Then add a call to that function inside of `run`. Be sure to call the `save` function after each
///         deployment so that hardhat-deploy style artifacts can be generated using a call to `sync()`.
///         The `CONTRACT_ADDRESSES_PATH` environment variable can be set to a path that contains a JSON file full of
///         contract name to address pairs. That enables this script to be much more flexible in the way
///         it is used.
contract Deploy is Deployer {
    DeployConfig public cfg;

    using stdJson for string;

    ////////////////////////////////////////////////////////////////
    //                        Modifiers                           //
    ////////////////////////////////////////////////////////////////

    /// @notice Modifier that wraps a function in broadcasting.
    modifier broadcast() {
        vm.startBroadcast(msg.sender);
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

    /// @notice Modifier that wraps a function with statediff recording.
    ///         The returned AccountAccess[] array is then written to
    ///         the `snapshots/state-diff/<name>.json` output file.
    modifier stateDiff() {
        vm.startStateDiffRecording();
        _;
        VmSafe.AccountAccess[] memory accesses = vm.stopAndReturnStateDiff();
        console.log("Writing %d state diff account accesses to snapshots/state-diff/%s.json", accesses.length, name());
        string memory json = LibStateDiff.encodeAccountAccesses(accesses);
        string memory statediffPath = string.concat(vm.projectRoot(), "/snapshots/state-diff/", name(), ".json");
        vm.writeJson({ json: json, path: statediffPath });
    }

    ////////////////////////////////////////////////////////////////
    //                        Accessors                           //
    ////////////////////////////////////////////////////////////////

    /// @inheritdoc Deployer
    function name() public pure override returns (string memory name_) {
        name_ = "Deploy";
    }

    /// @notice The create2 salt used for deployment of the contract implementations.
    ///         Using this helps to reduce config across networks as the implementation
    ///         addresses will be the same across networks when deployed with create2.
    function _implSalt() internal returns (bytes32) {
        return keccak256(bytes(vm.envOr("IMPL_SALT", string("ethers phoenix"))));
    }

    /// @notice Returns the proxy addresses. If a proxy is not found, it will have address(0).
    function _proxies() private view returns (Types.ContractSet memory proxies_) {
        proxies_ = Types.ContractSet({
            L1CrossDomainMessenger: mustGetAddress("L1CrossDomainMessengerProxy"),
            L1StandardBridge: mustGetAddress("L1StandardBridgeProxy"),
            L2OutputOracle: mustGetAddress("L2OutputOracleProxy"),
            OptimismMintableERC20Factory: mustGetAddress("OptimismMintableERC20FactoryProxy"),
            OptimismPortal: mustGetAddress("OptimismPortalProxy"),
            SystemConfig: mustGetAddress("SystemConfigProxy"),
            L1ERC721Bridge: mustGetAddress("L1ERC721BridgeProxy"),
            ProtocolVersions: mustGetAddress("ProtocolVersionsProxy"),
            SuperchainConfig: mustGetAddress("SuperchainConfigProxy")
        });
    }

    /// @notice Returns the proxy addresses, not reverting if any are unset.
    function _proxiesUnstrict() private view returns (Types.ContractSet memory proxies_) {
        proxies_ = Types.ContractSet({
            L1CrossDomainMessenger: getAddress("L1CrossDomainMessengerProxy"),
            L1StandardBridge: getAddress("L1StandardBridgeProxy"),
            L2OutputOracle: getAddress("L2OutputOracleProxy"),
            OptimismMintableERC20Factory: getAddress("OptimismMintableERC20FactoryProxy"),
            OptimismPortal: getAddress("OptimismPortalProxy"),
            SystemConfig: getAddress("SystemConfigProxy"),
            L1ERC721Bridge: getAddress("L1ERC721BridgeProxy"),
            ProtocolVersions: getAddress("ProtocolVersionsProxy"),
            SuperchainConfig: getAddress("SuperchainConfigProxy")
        });
    }

    ////////////////////////////////////////////////////////////////
    //            State Changing Helper Functions                 //
    ////////////////////////////////////////////////////////////////

    /// @notice Gets the address of the SafeProxyFactory and Safe singleton for use in deploying a new GnosisSafe.
    function _getSafeFactory() internal returns (SafeProxyFactory safeProxyFactory_, Safe safeSingleton_) {
        // These are they standard create2 deployed contracts. First we'll check if they are deployed,
        // if not we'll deploy new ones, though not at these addresses.
        address safeProxyFactory = 0xa6B71E26C5e0845f74c812102Ca7114b6a896AB2;
        address safeSingleton = 0xd9Db270c1B5E3Bd161E8c8503c55cEABeE709552;

        safeProxyFactory.code.length == 0
            ? safeProxyFactory_ = new SafeProxyFactory()
            : safeProxyFactory_ = SafeProxyFactory(safeProxyFactory);

        safeSingleton.code.length == 0 ? safeSingleton_ = new Safe() : safeSingleton_ = Safe(payable(safeSingleton));

        save("SafeProxyFactory", address(safeProxyFactory_));
        save("SafeSingleton", address(safeSingleton_));
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
        address proxyAdmin = mustGetAddress("ProxyAdmin");

        bytes memory data =
            abi.encodeCall(ProxyAdmin.upgradeAndCall, (payable(_proxy), _implementation, _innerCallData));

        _callViaSafe({ _target: proxyAdmin, _data: data });
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

    ////////////////////////////////////////////////////////////////
    //                    SetUp and Run                           //
    ////////////////////////////////////////////////////////////////

    function setUp() public virtual override {
        super.setUp();

        string memory path = string.concat(vm.projectRoot(), "/deploy-config/", deploymentContext, ".json");
        cfg = new DeployConfig(path);

        console.log("Deploying from %s", deployScript);
        console.log("Deployment context: %s", deploymentContext);
    }

    /// @notice Deploy all of the L1 contracts necessary for a full Superchain with a single Op Chain.
    function run() public {
        console.log("Deploying a fresh OP Stack including SuperchainConfig");
        _run();
    }

    /// @notice Deploy all L1 contracts and write the state diff to a file.
    function runWithStateDiff() public stateDiff {
        _run();
    }

    /// @notice Internal function containing the deploy logic.
    function _run() internal {
        deploySafe();
        setupSuperchain();
        setupOpChain();
    }

    ////////////////////////////////////////////////////////////////
    //           High Level Deployment Functions                  //
    ////////////////////////////////////////////////////////////////

    /// @notice Deploy a full system with a new SuperchainConfig
    ///         The Superchain system has 2 singleton contracts which lie outside of an OP Chain:
    ///         1. The SuperchainConfig contract
    ///         2. The ProtocolVersions contract
    function setupSuperchain() public {
        console.log("Setting up Superchain");

        // Deploy a new ProxyAdmin and AddressManager
        // This proxy will be used on the SuperchainConfig and ProtocolVersions contracts, as well as the contracts
        // in the OP Chain system.
        deployAddressManager();
        deployProxyAdmin();
        transferProxyAdminOwnership();

        // Deploy the SuperchainConfigProxy
        deployERC1967Proxy("SuperchainConfigProxy");
        deploySuperchainConfig();
        initializeSuperchainConfig();

        // Deploy the ProtocolVersionsProxy
        deployERC1967Proxy("ProtocolVersionsProxy");
        deployProtocolVersions();
        initializeProtocolVersions();
    }

    /// @notice Deploy a new OP Chain, with an existing SuperchainConfig provided
    function setupOpChain() public {
        console.log("Deploying OP Chain");

        // Ensure that the requisite contracts are deployed
        mustGetAddress("SuperchainConfigProxy");
        mustGetAddress("SystemOwnerSafe");
        mustGetAddress("AddressManager");
        mustGetAddress("ProxyAdmin");

        deployProxies();
        deployImplementations();
        initializeImplementations();

        setCannonOutputBisectionGameImplementation();
        setAlphabetOutputBisectionGameImplementation();

        setAlphabetFaultGameImplementation();
        setCannonFaultGameImplementation();

        transferDisputeGameFactoryOwnership();
    }

    /// @notice Deploy all of the proxies
    function deployProxies() public {
        console.log("Deploying proxies");

        deployERC1967Proxy("OptimismPortalProxy");
        deployERC1967Proxy("L2OutputOracleProxy");
        deployERC1967Proxy("SystemConfigProxy");
        deployL1StandardBridgeProxy();
        deployL1CrossDomainMessengerProxy();
        deployERC1967Proxy("OptimismMintableERC20FactoryProxy");
        deployERC1967Proxy("L1ERC721BridgeProxy");
        deployERC1967Proxy("DisputeGameFactoryProxy");

        transferAddressManagerOwnership(); // to the ProxyAdmin
    }

    /// @notice Deploy all of the implementations
    function deployImplementations() public {
        console.log("Deploying implementations");
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
    }

    /// @notice Initialize all of the implementations
    function initializeImplementations() public {
        console.log("Initializing implementations");
        initializeDisputeGameFactory();
        initializeSystemConfig();
        initializeL1StandardBridge();
        initializeL1ERC721Bridge();
        initializeOptimismMintableERC20Factory();
        initializeL1CrossDomainMessenger();
        initializeL2OutputOracle();
        initializeOptimismPortal();
    }

    ////////////////////////////////////////////////////////////////
    //              Non-Proxied Deployment Functions              //
    ////////////////////////////////////////////////////////////////

    /// @notice Deploy the Safe
    function deploySafe() public broadcast returns (address addr_) {
        console.log("Deploying Safe");
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
        console.log("Deploying AddressManager");
        AddressManager manager = new AddressManager();
        require(manager.owner() == msg.sender);

        save("AddressManager", address(manager));
        console.log("AddressManager deployed at %s", address(manager));
        addr_ = address(manager);
    }

    /// @notice Deploy the ProxyAdmin
    function deployProxyAdmin() public broadcast returns (address addr_) {
        console.log("Deploying ProxyAdmin");
        ProxyAdmin admin = new ProxyAdmin({ _owner: msg.sender });
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

    /// @notice Deploy the StorageSetter contract, used for upgrades.
    function deployStorageSetter() public broadcast returns (address addr_) {
        console.log("Deploying StorageSetter");
        StorageSetter setter = new StorageSetter{ salt: _implSalt() }();
        console.log("StorageSetter deployed at: %s", address(setter));
        string memory version = setter.version();
        console.log("StorageSetter version: %s", version);
        addr_ = address(setter);
    }

    ////////////////////////////////////////////////////////////////
    //                Proxy Deployment Functions                  //
    ////////////////////////////////////////////////////////////////

    /// @notice Deploy the L1StandardBridgeProxy using a ChugSplashProxy
    function deployL1StandardBridgeProxy() public broadcast returns (address addr_) {
        console.log("Deploying proxy for L1StandardBridge");
        address proxyAdmin = mustGetAddress("ProxyAdmin");
        L1ChugSplashProxy proxy = new L1ChugSplashProxy(proxyAdmin);

        address admin = address(uint160(uint256(vm.load(address(proxy), OWNER_KEY))));
        require(admin == proxyAdmin);

        save("L1StandardBridgeProxy", address(proxy));
        console.log("L1StandardBridgeProxy deployed at %s", address(proxy));
        addr_ = address(proxy);
    }

    /// @notice Deploy the L1CrossDomainMessengerProxy using a ResolvedDelegateProxy
    function deployL1CrossDomainMessengerProxy() public broadcast returns (address addr_) {
        console.log("Deploying proxy for L1CrossDomainMessenger");
        AddressManager addressManager = AddressManager(mustGetAddress("AddressManager"));
        ResolvedDelegateProxy proxy = new ResolvedDelegateProxy(addressManager, "OVM_L1CrossDomainMessenger");

        save("L1CrossDomainMessengerProxy", address(proxy));
        console.log("L1CrossDomainMessengerProxy deployed at %s", address(proxy));

        addr_ = address(proxy);
    }

    function deployERC1967Proxy(string memory _name) public broadcast returns (address addr_) {
        console.log(string.concat("Deploying ERC1967 proxy for", _name, ""));
        address proxyAdmin = mustGetAddress("ProxyAdmin");
        Proxy proxy = new Proxy({ _admin: proxyAdmin });

        address admin = address(uint160(uint256(vm.load(address(proxy), OWNER_KEY))));
        require(admin == proxyAdmin);

        save(_name, address(proxy));
        console.log("   at %s", address(proxy));
        addr_ = address(proxy);
    }

    ////////////////////////////////////////////////////////////////
    //             Implementation Deployment Functions            //
    ////////////////////////////////////////////////////////////////

    /// @notice Deploy the SuperchainConfig contract
    function deploySuperchainConfig() public broadcast {
        SuperchainConfig superchainConfig = new SuperchainConfig{ salt: _implSalt() }();

        require(superchainConfig.guardian() == address(0));
        bytes32 initialized = vm.load(address(superchainConfig), bytes32(0));
        require(initialized != 0);

        save("SuperchainConfig", address(superchainConfig));
        console.log("SuperchainConfig deployed at %s", address(superchainConfig));
    }

    /// @notice Deploy the L1CrossDomainMessenger
    function deployL1CrossDomainMessenger() public broadcast returns (address addr_) {
        console.log("Deploying L1CrossDomainMessenger implementation");
        address portal = mustGetAddress("OptimismPortalProxy");
        L1CrossDomainMessenger messenger =
            new L1CrossDomainMessenger{ salt: _implSalt() }({ _portal: OptimismPortal(payable(portal)) });

        save("L1CrossDomainMessenger", address(messenger));
        console.log("L1CrossDomainMessenger deployed at %s", address(messenger));

        // Override the `L1CrossDomainMessenger` contract to the deployed implementation. This is necessary
        // to check the `L1CrossDomainMessenger` implementation alongside dependent contracts, which
        // are always proxies.
        Types.ContractSet memory contracts = _proxiesUnstrict();
        contracts.L1CrossDomainMessenger = address(messenger);
        ChainAssertions.checkL1CrossDomainMessenger({ _contracts: contracts, _vm: vm, _isProxy: false });

        require(loadInitializedSlot("L1CrossDomainMessenger", false) == 1, "L1CrossDomainMessenger is not initialized");

        addr_ = address(messenger);
    }

    /// @notice Deploy the OptimismPortal
    function deployOptimismPortal() public broadcast returns (address addr_) {
        console.log("Deploying OptimismPortal implementation");

        L2OutputOracle l2OutputOracle = L2OutputOracle(mustGetAddress("L2OutputOracleProxy"));
        SystemConfig systemConfig = SystemConfig(mustGetAddress("SystemConfigProxy"));

        OptimismPortal portal =
            new OptimismPortal{ salt: _implSalt() }({ _l2Oracle: l2OutputOracle, _systemConfig: systemConfig });

        save("OptimismPortal", address(portal));
        console.log("OptimismPortal deployed at %s", address(portal));

        // Override the `OptimismPortal` contract to the deployed implementation. This is necessary
        // to check the `OptimismPortal` implementation alongside dependent contracts, which
        // are always proxies.
        Types.ContractSet memory contracts = _proxiesUnstrict();
        contracts.OptimismPortal = address(portal);
        ChainAssertions.checkOptimismPortal({ _contracts: contracts, _cfg: cfg, _isProxy: false });

        require(loadInitializedSlot("OptimismPortal", false) == 1, "OptimismPortal is not initialized");

        addr_ = address(portal);
    }

    /// @notice Deploy the L2OutputOracle
    function deployL2OutputOracle() public broadcast returns (address addr_) {
        console.log("Deploying L2OutputOracle implementation");
        L2OutputOracle oracle = new L2OutputOracle{ salt: _implSalt() }({
            _submissionInterval: cfg.l2OutputOracleSubmissionInterval(),
            _l2BlockTime: cfg.l2BlockTime(),
            _startingBlockNumber: 0,
            _startingTimestamp: 0,
            _proposer: cfg.l2OutputOracleProposer(),
            _challenger: cfg.l2OutputOracleChallenger(),
            _finalizationPeriodSeconds: cfg.finalizationPeriodSeconds()
        });

        save("L2OutputOracle", address(oracle));
        console.log("L2OutputOracle deployed at %s", address(oracle));

        // Override the `L2OutputOracle` contract to the deployed implementation. This is necessary
        // to check the `L2OutputOracle` implementation alongside dependent contracts, which
        // are always proxies.
        Types.ContractSet memory contracts = _proxiesUnstrict();
        contracts.L2OutputOracle = address(oracle);
        ChainAssertions.checkL2OutputOracle(contracts, cfg, 0, 0);

        require(loadInitializedSlot("L2OutputOracle", false) == 1, "L2OutputOracle is not initialized");

        addr_ = address(oracle);
    }

    /// @notice Deploy the OptimismMintableERC20Factory
    function deployOptimismMintableERC20Factory() public broadcast returns (address addr_) {
        console.log("Deploying OptimismMintableERC20Factory implementation");
        address l1standardBridgeProxy = mustGetAddress("L1StandardBridgeProxy");
        OptimismMintableERC20Factory factory =
            new OptimismMintableERC20Factory{ salt: _implSalt() }({ _bridge: l1standardBridgeProxy });

        save("OptimismMintableERC20Factory", address(factory));
        console.log("OptimismMintableERC20Factory deployed at %s", address(factory));

        // Override the `OptimismMintableERC20Factory` contract to the deployed implementation. This is necessary
        // to check the `OptimismMintableERC20Factory` implementation alongside dependent contracts, which
        // are always proxies.
        Types.ContractSet memory contracts = _proxiesUnstrict();
        contracts.OptimismMintableERC20Factory = address(factory);
        ChainAssertions.checkOptimismMintableERC20Factory(contracts);

        addr_ = address(factory);
    }

    /// @notice Deploy the DisputeGameFactory
    function deployDisputeGameFactory() public onlyDevnet broadcast returns (address addr_) {
        console.log("Deploying DisputeGameFactory implementation");
        DisputeGameFactory factory = new DisputeGameFactory{ salt: _implSalt() }();
        save("DisputeGameFactory", address(factory));
        console.log("DisputeGameFactory deployed at %s", address(factory));

        addr_ = address(factory);
    }

    /// @notice Deploy the BlockOracle
    function deployBlockOracle() public onlyDevnet broadcast returns (address addr_) {
        console.log("Deploying BlockOracle implementation");
        BlockOracle oracle = new BlockOracle{ salt: _implSalt() }();
        save("BlockOracle", address(oracle));
        console.log("BlockOracle deployed at %s", address(oracle));

        addr_ = address(oracle);
    }

    /// @notice Deploy the ProtocolVersions
    function deployProtocolVersions() public broadcast returns (address addr_) {
        console.log("Deploying ProtocolVersions implementation");
        ProtocolVersions versions = new ProtocolVersions{ salt: _implSalt() }();
        save("ProtocolVersions", address(versions));
        console.log("ProtocolVersions deployed at %s", address(versions));

        // Override the `ProtocolVersions` contract to the deployed implementation. This is necessary
        // to check the `ProtocolVersions` implementation alongside dependent contracts, which
        // are always proxies.
        Types.ContractSet memory contracts = _proxiesUnstrict();
        contracts.ProtocolVersions = address(versions);
        ChainAssertions.checkProtocolVersions({ _contracts: contracts, _cfg: cfg, _isProxy: false });

        require(loadInitializedSlot("ProtocolVersions", false) == 1, "ProtocolVersions is not initialized");

        addr_ = address(versions);
    }

    /// @notice Deploy the PreimageOracle
    function deployPreimageOracle() public onlyDevnet broadcast returns (address addr_) {
        console.log("Deploying PreimageOracle implementation");
        PreimageOracle preimageOracle = new PreimageOracle{ salt: _implSalt() }();
        save("PreimageOracle", address(preimageOracle));
        console.log("PreimageOracle deployed at %s", address(preimageOracle));

        addr_ = address(preimageOracle);
    }

    /// @notice Deploy Mips
    function deployMips() public onlyDevnet broadcast returns (address addr_) {
        console.log("Deploying Mips implementation");
        MIPS mips = new MIPS{ salt: _implSalt() }(IPreimageOracle(mustGetAddress("PreimageOracle")));
        save("Mips", address(mips));
        console.log("MIPS deployed at %s", address(mips));

        addr_ = address(mips);
    }

    /// @notice Deploy the SystemConfig
    function deploySystemConfig() public broadcast returns (address addr_) {
        console.log("Deploying SystemConfig implementation");
        ResourceMetering.ResourceConfig memory defaultConfig = Constants.DEFAULT_RESOURCE_CONFIG();
        uint64 minimumGasLimit = uint64(defaultConfig.maxResourceLimit) + uint64(defaultConfig.systemTxMaxGas);
        SystemConfig config = new SystemConfig({
            _owner: address(0xdEaD),
            _overhead: 0,
            _scalar: 0,
            _batcherHash: bytes32(0),
            _gasLimit: minimumGasLimit,
            _unsafeBlockSigner: address(0),
            _config: defaultConfig
        });

        save("SystemConfig", address(config));
        console.log("SystemConfig deployed at %s", address(config));

        // Override the `SystemConfig` contract to the deployed implementation. This is necessary
        // to check the `SystemConfig` implementation alongside dependent contracts, which
        // are always proxies.
        Types.ContractSet memory contracts = _proxiesUnstrict();
        contracts.SystemConfig = address(config);
        ChainAssertions.checkSystemConfig({ _contracts: contracts, _cfg: cfg, _isProxy: false });

        require(loadInitializedSlot("SystemConfig", false) == 1, "SystemConfig is not initialized");

        addr_ = address(config);
    }

    /// @notice Deploy the L1StandardBridge
    function deployL1StandardBridge() public broadcast returns (address addr_) {
        console.log("Deploying L1StandardBridge implementation");
        address l1CrossDomainMessengerProxy = mustGetAddress("L1CrossDomainMessengerProxy");

        L1StandardBridge bridge =
            new L1StandardBridge{ salt: _implSalt() }({ _messenger: payable(l1CrossDomainMessengerProxy) });

        save("L1StandardBridge", address(bridge));
        console.log("L1StandardBridge deployed at %s", address(bridge));

        // Override the `L1StandardBridge` contract to the deployed implementation. This is necessary
        // to check the `L1StandardBridge` implementation alongside dependent contracts, which
        // are always proxies.
        Types.ContractSet memory contracts = _proxiesUnstrict();
        contracts.L1StandardBridge = address(bridge);
        ChainAssertions.checkL1StandardBridge({ _contracts: contracts, _isProxy: false });

        addr_ = address(bridge);
    }

    /// @notice Deploy the L1ERC721Bridge
    function deployL1ERC721Bridge() public broadcast returns (address addr_) {
        console.log("Deploying L1ERC721Bridge implementation");
        address l1CrossDomainMessengerProxy = mustGetAddress("L1CrossDomainMessengerProxy");
        L1ERC721Bridge bridge = new L1ERC721Bridge{ salt: _implSalt() }({ _messenger: l1CrossDomainMessengerProxy });

        save("L1ERC721Bridge", address(bridge));
        console.log("L1ERC721Bridge deployed at %s", address(bridge));

        // Override the `L1ERC721Bridge` contract to the deployed implementation. This is necessary
        // to check the `L1ERC721Bridge` implementation alongside dependent contracts, which
        // are always proxies.
        Types.ContractSet memory contracts = _proxiesUnstrict();
        contracts.L1ERC721Bridge = address(bridge);

        ChainAssertions.checkL1ERC721Bridge({ _contracts: contracts, _isProxy: false });

        addr_ = address(bridge);
    }

    /// @notice Transfer ownership of the address manager to the ProxyAdmin
    function transferAddressManagerOwnership() public broadcast {
        console.log("Transferring AddressManager ownership to ProxyAdmin");
        AddressManager addressManager = AddressManager(mustGetAddress("AddressManager"));
        address owner = addressManager.owner();
        address proxyAdmin = mustGetAddress("ProxyAdmin");
        if (owner != proxyAdmin) {
            addressManager.transferOwnership(proxyAdmin);
            console.log("AddressManager ownership transferred to %s", proxyAdmin);
        }

        require(addressManager.owner() == proxyAdmin);
    }

    ////////////////////////////////////////////////////////////////
    //                    Initialize Functions                    //
    ////////////////////////////////////////////////////////////////

    /// @notice Initialize the SuperchainConfig
    function initializeSuperchainConfig() public broadcast {
        address payable superchainConfigProxy = mustGetAddress("SuperchainConfigProxy");
        address payable superchainConfig = mustGetAddress("SuperchainConfig");
        _upgradeAndCallViaSafe({
            _proxy: superchainConfigProxy,
            _implementation: superchainConfig,
            _innerCallData: abi.encodeCall(SuperchainConfig.initialize, (cfg.superchainConfigGuardian(), false))
        });

        ChainAssertions.checkSuperchainConfig({ _contracts: _proxiesUnstrict(), _cfg: cfg, _isPaused: false });
    }

    /// @notice Initialize the DisputeGameFactory
    function initializeDisputeGameFactory() public onlyDevnet broadcast {
        console.log("Upgrading and initializing DisputeGameFactory proxy");
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
        console.log("Upgrading and initializing SystemConfig proxy");
        address systemConfigProxy = mustGetAddress("SystemConfigProxy");
        address systemConfig = mustGetAddress("SystemConfig");

        bytes32 batcherHash = bytes32(uint256(uint160(cfg.batchSenderAddress())));

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
                    Constants.DEFAULT_RESOURCE_CONFIG()
                )
                )
        });

        SystemConfig config = SystemConfig(systemConfigProxy);
        string memory version = config.version();
        console.log("SystemConfig version: %s", version);

        ChainAssertions.checkSystemConfig({ _contracts: _proxies(), _cfg: cfg, _isProxy: true });

        require(loadInitializedSlot("SystemConfig", true) == 1, "SystemConfigProxy is not initialized");
    }

    /// @notice Initialize the L1StandardBridge
    function initializeL1StandardBridge() public broadcast {
        console.log("Upgrading and initializing L1StandardBridge proxy");
        ProxyAdmin proxyAdmin = ProxyAdmin(mustGetAddress("ProxyAdmin"));
        address l1StandardBridgeProxy = mustGetAddress("L1StandardBridgeProxy");
        address l1StandardBridge = mustGetAddress("L1StandardBridge");
        address superchainConfigProxy = mustGetAddress("SuperchainConfigProxy");

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
            _innerCallData: abi.encodeCall(L1StandardBridge.initialize, (SuperchainConfig(superchainConfigProxy)))
        });

        string memory version = L1StandardBridge(payable(l1StandardBridgeProxy)).version();
        console.log("L1StandardBridge version: %s", version);

        ChainAssertions.checkL1StandardBridge({ _contracts: _proxies(), _isProxy: true });
    }

    /// @notice Initialize the L1ERC721Bridge
    function initializeL1ERC721Bridge() public broadcast {
        console.log("Upgrading and initializing L1ERC721Bridge proxy");
        address l1ERC721BridgeProxy = mustGetAddress("L1ERC721BridgeProxy");
        address l1ERC721Bridge = mustGetAddress("L1ERC721Bridge");
        address superchainConfigProxy = mustGetAddress("SuperchainConfigProxy");

        _upgradeAndCallViaSafe({
            _proxy: payable(l1ERC721BridgeProxy),
            _implementation: l1ERC721Bridge,
            _innerCallData: abi.encodeCall(L1ERC721Bridge.initialize, (SuperchainConfig(superchainConfigProxy)))
        });

        L1ERC721Bridge bridge = L1ERC721Bridge(l1ERC721BridgeProxy);
        string memory version = bridge.version();
        console.log("L1ERC721Bridge version: %s", version);

        ChainAssertions.checkL1ERC721Bridge({ _contracts: _proxies(), _isProxy: true });
    }

    /// @notice Ininitialize the OptimismMintableERC20Factory
    function initializeOptimismMintableERC20Factory() public broadcast {
        console.log("Upgrading and initializing OptimismMintableERC20Factory proxy");
        address proxyAdmin = mustGetAddress("ProxyAdmin");
        address optimismMintableERC20FactoryProxy = mustGetAddress("OptimismMintableERC20FactoryProxy");
        address optimismMintableERC20Factory = mustGetAddress("OptimismMintableERC20Factory");

        _callViaSafe({
            _target: proxyAdmin,
            _data: abi.encodeCall(
                ProxyAdmin.upgrade, (payable(optimismMintableERC20FactoryProxy), optimismMintableERC20Factory)
                )
        });

        OptimismMintableERC20Factory factory = OptimismMintableERC20Factory(optimismMintableERC20FactoryProxy);
        string memory version = factory.version();
        console.log("OptimismMintableERC20Factory version: %s", version);

        ChainAssertions.checkOptimismMintableERC20Factory(_proxies());
    }

    /// @notice initializeL1CrossDomainMessenger
    function initializeL1CrossDomainMessenger() public broadcast {
        console.log("Upgrading and initializing L1CrossDomainMessenger proxy");
        ProxyAdmin proxyAdmin = ProxyAdmin(mustGetAddress("ProxyAdmin"));
        address l1CrossDomainMessengerProxy = mustGetAddress("L1CrossDomainMessengerProxy");
        address l1CrossDomainMessenger = mustGetAddress("L1CrossDomainMessenger");
        SuperchainConfig superchainConfigProxy = SuperchainConfig(mustGetAddress("SuperchainConfigProxy"));

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
            _innerCallData: abi.encodeCall(L1CrossDomainMessenger.initialize, (superchainConfigProxy))
        });

        L1CrossDomainMessenger messenger = L1CrossDomainMessenger(l1CrossDomainMessengerProxy);
        string memory version = messenger.version();
        console.log("L1CrossDomainMessenger version: %s", version);

        ChainAssertions.checkL1CrossDomainMessenger({ _contracts: _proxies(), _vm: vm, _isProxy: true });

        require(
            loadInitializedSlot("L1CrossDomainMessenger", true) == 1, "L1CrossDomainMessengerProxy is not initialized"
        );
    }

    /// @notice Initialize the L2OutputOracle
    function initializeL2OutputOracle() public broadcast {
        console.log("Upgrading and initializing L2OutputOracle proxy");
        address l2OutputOracleProxy = mustGetAddress("L2OutputOracleProxy");
        address l2OutputOracle = mustGetAddress("L2OutputOracle");

        _upgradeAndCallViaSafe({
            _proxy: payable(l2OutputOracleProxy),
            _implementation: l2OutputOracle,
            _innerCallData: abi.encodeCall(
                L2OutputOracle.initialize, (cfg.l2OutputOracleStartingBlockNumber(), cfg.l2OutputOracleStartingTimestamp())
                )
        });

        L2OutputOracle oracle = L2OutputOracle(l2OutputOracleProxy);
        string memory version = oracle.version();
        console.log("L2OutputOracle version: %s", version);

        ChainAssertions.checkL2OutputOracle({
            _contracts: _proxies(),
            _cfg: cfg,
            _l2OutputOracleStartingBlockNumber: cfg.l2OutputOracleStartingBlockNumber(),
            _l2OutputOracleStartingTimestamp: cfg.l2OutputOracleStartingTimestamp()
        });

        require(loadInitializedSlot("L2OutputOracle", true) == 1, "L2OutputOracleProxy is not initialized");
    }

    /// @notice Initialize the OptimismPortal
    function initializeOptimismPortal() public broadcast {
        console.log("Upgrading and initializing OptimismPortal proxy");
        address optimismPortalProxy = mustGetAddress("OptimismPortalProxy");
        address optimismPortal = mustGetAddress("OptimismPortal");
        SuperchainConfig superchainConfigProxy = SuperchainConfig(mustGetAddress("SuperchainConfigProxy"));

        _upgradeAndCallViaSafe({
            _proxy: payable(optimismPortalProxy),
            _implementation: optimismPortal,
            _innerCallData: abi.encodeCall(OptimismPortal.initialize, (superchainConfigProxy))
        });

        OptimismPortal portal = OptimismPortal(payable(optimismPortalProxy));
        string memory version = portal.version();
        console.log("OptimismPortal version: %s", version);

        ChainAssertions.checkOptimismPortal({ _contracts: _proxies(), _cfg: cfg, _isProxy: true });

        require(loadInitializedSlot("OptimismPortal", true) == 1, "OptimismPortalProxy is not initialized");
    }

    function initializeProtocolVersions() public broadcast {
        console.log("Upgrading and initializing ProtocolVersions proxy");
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

        ChainAssertions.checkProtocolVersions({ _contracts: _proxiesUnstrict(), _cfg: cfg, _isProxy: true });

        require(loadInitializedSlot("ProtocolVersions", true) == 1, "ProtocolVersionsProxy is not initialized");
    }

    /// @notice Transfer ownership of the DisputeGameFactory contract to the final system owner
    function transferDisputeGameFactoryOwnership() public onlyDevnet broadcast {
        console.log("Transferring DisputeGameFactory ownership to Safe");
        DisputeGameFactory disputeGameFactory = DisputeGameFactory(mustGetAddress("DisputeGameFactoryProxy"));
        address owner = disputeGameFactory.owner();

        address safe = mustGetAddress("SystemOwnerSafe");
        if (owner != safe) {
            disputeGameFactory.transferOwnership(safe);
            console.log("DisputeGameFactory ownership transferred to Safe at: %s", safe);
        }
    }

    /// @notice Loads the mips absolute prestate from the prestate-proof for devnets otherwise
    ///         from the config.
    function loadMipsAbsolutePrestate() internal returns (Claim mipsAbsolutePrestate_) {
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
            mipsAbsolutePrestate_ = Claim.wrap(abi.decode(vm.ffi(commands), (bytes32)));
            console.log(
                "[Cannon Dispute Game] Using devnet MIPS Absolute prestate: %s",
                vm.toString(Claim.unwrap(mipsAbsolutePrestate_))
            );
        } else {
            console.log(
                "[Cannon Dispute Game] Using absolute prestate from config: %s", cfg.faultGameAbsolutePrestate()
            );
            mipsAbsolutePrestate_ = Claim.wrap(bytes32(cfg.faultGameAbsolutePrestate()));
        }
    }

    /// @notice Sets the implementation for the `FAULT` game type in the `DisputeGameFactory`
    function setCannonFaultGameImplementation() public onlyDevnet broadcast {
        console.log("Setting Cannon FaultDisputeGame implementation");
        DisputeGameFactory factory = DisputeGameFactory(mustGetAddress("DisputeGameFactoryProxy"));

        // Set the Cannon FaultDisputeGame implementation in the factory.
        _setFaultGameImplementation({
            _factory: factory,
            _gameType: GameTypes.CANNON,
            _absolutePrestate: loadMipsAbsolutePrestate(),
            _faultVm: IBigStepper(mustGetAddress("Mips")),
            _maxGameDepth: 30 // Hard code depth for legacy game to keep e2e tests fast
         });
    }

    /// @notice Sets the implementation for the `OUTPUT_CANNON` game type in the `DisputeGameFactory`
    function setCannonOutputBisectionGameImplementation() public onlyDevnet broadcast {
        console.log("Setting Cannon OutputBisectionGame implementation");
        DisputeGameFactory factory = DisputeGameFactory(mustGetAddress("DisputeGameFactoryProxy"));

        _setFaultGameImplementation({
            _factory: factory,
            _gameType: GameTypes.OUTPUT_CANNON,
            _absolutePrestate: loadMipsAbsolutePrestate(),
            _faultVm: IBigStepper(mustGetAddress("Mips")),
            _maxGameDepth: cfg.faultGameMaxDepth()
        });
    }

    /// @notice Sets the implementation for the `OUTPUT_ALPHABET` game type in the `DisputeGameFactory`
    function setAlphabetOutputBisectionGameImplementation() public onlyDevnet broadcast {
        console.log("Setting Alphabet OutputBisectionGame implementation");
        DisputeGameFactory factory = DisputeGameFactory(mustGetAddress("DisputeGameFactoryProxy"));

        Claim outputAbsolutePrestate = Claim.wrap(bytes32(cfg.faultGameAbsolutePrestate()));
        _setFaultGameImplementation({
            _factory: factory,
            _gameType: GameTypes.OUTPUT_ALPHABET,
            _absolutePrestate: outputAbsolutePrestate,
            _faultVm: IBigStepper(new AlphabetVM2(outputAbsolutePrestate)),
            _maxGameDepth: cfg.faultGameMaxDepth()
        });
    }

    /// @notice Sets the implementation for the `ALPHABET` game type in the `DisputeGameFactory`
    function setAlphabetFaultGameImplementation() public onlyDevnet broadcast {
        console.log("Setting Alphabet FaultDisputeGame implementation");
        DisputeGameFactory factory = DisputeGameFactory(mustGetAddress("DisputeGameFactoryProxy"));

        // Set the Alphabet FaultDisputeGame implementation in the factory.
        Claim alphabetAbsolutePrestate = Claim.wrap(bytes32(cfg.faultGameAbsolutePrestate()));
        _setFaultGameImplementation({
            _factory: factory,
            _gameType: GameTypes.ALPHABET,
            _absolutePrestate: alphabetAbsolutePrestate,
            _faultVm: IBigStepper(new AlphabetVM(alphabetAbsolutePrestate)),
            _maxGameDepth: 4 // The max game depth of the alphabet game is always 4.
         });
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
        if (address(_factory.gameImpls(_gameType)) != address(0)) {
            console.log(
                "[WARN] DisputeGameFactoryProxy: `FaultDisputeGame` implementation already set for game type: %s",
                vm.toString(GameType.unwrap(_gameType))
            );
            return;
        }

        string memory deployed;
        if (
            GameType.unwrap(_gameType) == GameType.unwrap(GameTypes.OUTPUT_ALPHABET)
                || GameType.unwrap(_gameType) == GameType.unwrap(GameTypes.OUTPUT_CANNON)
        ) {
            deployed = "OutputBisectionGame";
            _factory.setImplementation(
                _gameType,
                new OutputBisectionGame({
                    _gameType: _gameType,
                    _absolutePrestate: _absolutePrestate,
                    _genesisBlockNumber: cfg.outputBisectionGameGenesisBlock(),
                    _genesisOutputRoot: Hash.wrap(cfg.outputBisectionGameGenesisOutputRoot()),
                    _maxGameDepth: _maxGameDepth,
                    _splitDepth: cfg.outputBisectionGameSplitDepth(),
                    _gameDuration: Duration.wrap(uint64(cfg.faultGameMaxDuration())),
                    _vm: _faultVm
                })
            );
        } else {
            deployed = "FaultDisputeGame";
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
        }

        uint8 rawGameType = GameType.unwrap(_gameType);
        string memory gameTypeString;
        if (rawGameType == GameType.unwrap(GameTypes.CANNON)) {
            gameTypeString = "Cannon";
        } else if (rawGameType == GameType.unwrap(GameTypes.OUTPUT_CANNON)) {
            gameTypeString = "OutputBisectionCannon";
        } else if (rawGameType == GameType.unwrap(GameTypes.OUTPUT_ALPHABET)) {
            gameTypeString = "OutputBisectionAlphabet";
        } else if (rawGameType == GameType.unwrap(GameTypes.ALPHABET)) {
            gameTypeString = "Alphabet";
        } else {
            gameTypeString = "Unknown";
        }

        console.log(
            "DisputeGameFactoryProxy: set `%s` implementation (Backend: %s | GameType: %s)",
            deployed,
            gameTypeString,
            vm.toString(rawGameType)
        );
    }
}
