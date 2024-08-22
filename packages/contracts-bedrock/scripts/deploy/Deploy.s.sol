// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { VmSafe } from "forge-std/Vm.sol";
import { Script } from "forge-std/Script.sol";

import { console2 as console } from "forge-std/console2.sol";
import { stdJson } from "forge-std/StdJson.sol";

import { GnosisSafe as Safe } from "safe-contracts/GnosisSafe.sol";
import { OwnerManager } from "safe-contracts/base/OwnerManager.sol";
import { GnosisSafeProxyFactory as SafeProxyFactory } from "safe-contracts/proxies/GnosisSafeProxyFactory.sol";
import { Enum as SafeOps } from "safe-contracts/common/Enum.sol";

import { Deployer } from "scripts/deploy/Deployer.sol";

import { ProxyAdmin } from "src/universal/ProxyAdmin.sol";
import { AddressManager } from "src/legacy/AddressManager.sol";
import { Proxy } from "src/universal/Proxy.sol";
import { L1StandardBridge } from "src/L1/L1StandardBridge.sol";
import { StandardBridge } from "src/universal/StandardBridge.sol";
import { OptimismPortal } from "src/L1/OptimismPortal.sol";
import { OptimismPortal2 } from "src/L1/OptimismPortal2.sol";
import { OptimismPortalInterop } from "src/L1/OptimismPortalInterop.sol";
import { L1ChugSplashProxy } from "src/legacy/L1ChugSplashProxy.sol";
import { ResolvedDelegateProxy } from "src/legacy/ResolvedDelegateProxy.sol";
import { L1CrossDomainMessenger } from "src/L1/L1CrossDomainMessenger.sol";
import { L2OutputOracle } from "src/L1/L2OutputOracle.sol";
import { OptimismMintableERC20Factory } from "src/universal/OptimismMintableERC20Factory.sol";
import { SuperchainConfig } from "src/L1/SuperchainConfig.sol";
import { SystemConfig } from "src/L1/SystemConfig.sol";
import { SystemConfigInterop } from "src/L1/SystemConfigInterop.sol";
import { ResourceMetering } from "src/L1/ResourceMetering.sol";
import { DataAvailabilityChallenge } from "src/L1/DataAvailabilityChallenge.sol";
import { Constants } from "src/libraries/Constants.sol";
import { DisputeGameFactory } from "src/dispute/DisputeGameFactory.sol";
import { FaultDisputeGame } from "src/dispute/FaultDisputeGame.sol";
import { PermissionedDisputeGame } from "src/dispute/PermissionedDisputeGame.sol";
import { DelayedWETH } from "src/dispute/weth/DelayedWETH.sol";
import { AnchorStateRegistry } from "src/dispute/AnchorStateRegistry.sol";
import { PreimageOracle } from "src/cannon/PreimageOracle.sol";
import { MIPS } from "src/cannon/MIPS.sol";
import { L1ERC721Bridge } from "src/L1/L1ERC721Bridge.sol";
import { ProtocolVersions, ProtocolVersion } from "src/L1/ProtocolVersions.sol";
import { StorageSetter } from "src/universal/StorageSetter.sol";
import { Predeploys } from "src/libraries/Predeploys.sol";
import { Chains } from "scripts/libraries/Chains.sol";
import { Config } from "scripts/libraries/Config.sol";

import { IBigStepper } from "src/dispute/interfaces/IBigStepper.sol";
import { IPreimageOracle } from "src/cannon/interfaces/IPreimageOracle.sol";
import { AlphabetVM } from "test/mocks/AlphabetVM.sol";
import "src/dispute/lib/Types.sol";
import { ChainAssertions } from "scripts/deploy/ChainAssertions.sol";
import { Types } from "scripts/libraries/Types.sol";
import { LibStateDiff } from "scripts/libraries/LibStateDiff.sol";
import { EIP1967Helper } from "test/mocks/EIP1967Helper.sol";
import { ForgeArtifacts } from "scripts/libraries/ForgeArtifacts.sol";
import { Process } from "scripts/libraries/Process.sol";

/// @title Deploy
/// @notice Script used to deploy a bedrock system. The entire system is deployed within the `run` function.
///         To add a new contract to the system, add a public function that deploys that individual contract.
///         Then add a call to that function inside of `run`. Be sure to call the `save` function after each
///         deployment so that hardhat-deploy style artifacts can be generated using a call to `sync()`.
///         The `CONTRACT_ADDRESSES_PATH` environment variable can be set to a path that contains a JSON file full of
///         contract name to address pairs. That enables this script to be much more flexible in the way it is used.
///         This contract must not have constructor logic because it is set into state using `etch`.
contract Deploy is Deployer {
    using stdJson for string;

    /// @notice FaultDisputeGameParams is a struct that contains the parameters necessary to call
    ///         the function _setFaultGameImplementation. This struct exists because the EVM needs
    ///         to finally adopt PUSHN and get rid of stack too deep once and for all.
    ///         Someday we will look back and laugh about stack too deep, today is not that day.
    struct FaultDisputeGameParams {
        AnchorStateRegistry anchorStateRegistry;
        DelayedWETH weth;
        GameType gameType;
        Claim absolutePrestate;
        IBigStepper faultVm;
        uint256 maxGameDepth;
        Duration maxClockDuration;
    }

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
        console.log(
            "Writing %d state diff account accesses to snapshots/state-diff/%s.json",
            accesses.length,
            vm.toString(block.chainid)
        );
        string memory json = LibStateDiff.encodeAccountAccesses(accesses);
        string memory statediffPath =
            string.concat(vm.projectRoot(), "/snapshots/state-diff/", vm.toString(block.chainid), ".json");
        vm.writeJson({ json: json, path: statediffPath });
    }

    ////////////////////////////////////////////////////////////////
    //                        Accessors                           //
    ////////////////////////////////////////////////////////////////

    /// @notice The create2 salt used for deployment of the contract implementations.
    ///         Using this helps to reduce config across networks as the implementation
    ///         addresses will be the same across networks when deployed with create2.
    function _implSalt() internal view returns (bytes32) {
        return keccak256(bytes(Config.implSalt()));
    }

    /// @notice Returns the proxy addresses. If a proxy is not found, it will have address(0).
    function _proxies() internal view returns (Types.ContractSet memory proxies_) {
        proxies_ = Types.ContractSet({
            L1CrossDomainMessenger: mustGetAddress("L1CrossDomainMessengerProxy"),
            L1StandardBridge: mustGetAddress("L1StandardBridgeProxy"),
            L2OutputOracle: mustGetAddress("L2OutputOracleProxy"),
            DisputeGameFactory: mustGetAddress("DisputeGameFactoryProxy"),
            DelayedWETH: mustGetAddress("DelayedWETHProxy"),
            PermissionedDelayedWETH: mustGetAddress("PermissionedDelayedWETHProxy"),
            AnchorStateRegistry: mustGetAddress("AnchorStateRegistryProxy"),
            OptimismMintableERC20Factory: mustGetAddress("OptimismMintableERC20FactoryProxy"),
            OptimismPortal: mustGetAddress("OptimismPortalProxy"),
            OptimismPortal2: mustGetAddress("OptimismPortalProxy"),
            SystemConfig: mustGetAddress("SystemConfigProxy"),
            L1ERC721Bridge: mustGetAddress("L1ERC721BridgeProxy"),
            ProtocolVersions: mustGetAddress("ProtocolVersionsProxy"),
            SuperchainConfig: mustGetAddress("SuperchainConfigProxy")
        });
    }

    /// @notice Returns the proxy addresses, not reverting if any are unset.
    function _proxiesUnstrict() internal view returns (Types.ContractSet memory proxies_) {
        proxies_ = Types.ContractSet({
            L1CrossDomainMessenger: getAddress("L1CrossDomainMessengerProxy"),
            L1StandardBridge: getAddress("L1StandardBridgeProxy"),
            L2OutputOracle: getAddress("L2OutputOracleProxy"),
            DisputeGameFactory: getAddress("DisputeGameFactoryProxy"),
            DelayedWETH: getAddress("DelayedWETHProxy"),
            PermissionedDelayedWETH: getAddress("PermissionedDelayedWETHProxy"),
            AnchorStateRegistry: getAddress("AnchorStateRegistryProxy"),
            OptimismMintableERC20Factory: getAddress("OptimismMintableERC20FactoryProxy"),
            OptimismPortal: getAddress("OptimismPortalProxy"),
            OptimismPortal2: getAddress("OptimismPortalProxy"),
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
        if (getAddress("SafeProxyFactory") != address(0)) {
            // The SafeProxyFactory is already saved, we can just use it.
            safeProxyFactory_ = SafeProxyFactory(getAddress("SafeProxyFactory"));
            safeSingleton_ = Safe(getAddress("SafeSingleton"));
            return (safeProxyFactory_, safeSingleton_);
        }

        // These are the standard create2 deployed contracts. First we'll check if they are deployed,
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
    function _callViaSafe(Safe _safe, address _target, bytes memory _data) internal {
        // This is the signature format used when the caller is also the signer.
        bytes memory signature = abi.encodePacked(uint256(uint160(msg.sender)), bytes32(0), uint8(1));

        _safe.execTransaction({
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

        Safe safe = Safe(mustGetAddress("SystemOwnerSafe"));
        _callViaSafe({ _safe: safe, _target: proxyAdmin, _data: data });
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

    /// @notice Transfer ownership of a Proxy to the ProxyAdmin contract
    ///         This is expected to be used in conjusting with deployERC1967ProxyWithOwner after setup actions
    ///         have been performed on the proxy.
    /// @param _name The name of the proxy to transfer ownership of.
    function transferProxyToProxyAdmin(string memory _name) public broadcast {
        Proxy proxy = Proxy(mustGetAddress(_name));
        address proxyAdmin = mustGetAddress("ProxyAdmin");
        proxy.changeAdmin(proxyAdmin);
        console.log("Proxy %s ownership transferred to ProxyAdmin at: %s", _name, proxyAdmin);
    }

    ////////////////////////////////////////////////////////////////
    //                    SetUp and Run                           //
    ////////////////////////////////////////////////////////////////

    /// @notice Deploy all of the L1 contracts necessary for a full Superchain with a single Op Chain.
    function run() public {
        console.log("Deploying a fresh OP Stack including SuperchainConfig");
        _run();
    }

    function runWithStateDump() public {
        vm.chainId(cfg.l1ChainID());
        _run();
        vm.dumpState(Config.stateDumpPath(""));
    }

    /// @notice Deploy all L1 contracts and write the state diff to a file.
    function runWithStateDiff() public stateDiff {
        _run();
    }

    /// @notice Internal function containing the deploy logic.
    function _run() internal virtual {
        console.log("start of L1 Deploy!");
        deploySafe("SystemOwnerSafe");
        console.log("deployed Safe!");
        setupSuperchain();
        console.log("set up superchain!");
        if (cfg.useAltDA()) {
            bytes32 typeHash = keccak256(bytes(cfg.daCommitmentType()));
            bytes32 keccakHash = keccak256(bytes("KeccakCommitment"));
            if (typeHash == keccakHash) {
                setupOpAltDA();
            }
        }
        setupOpChain();
        console.log("set up op chain!");
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

        setAlphabetFaultGameImplementation({ _allowUpgrade: false });
        setFastFaultGameImplementation({ _allowUpgrade: false });
        setCannonFaultGameImplementation({ _allowUpgrade: false });
        setPermissionedCannonFaultGameImplementation({ _allowUpgrade: false });

        transferDisputeGameFactoryOwnership();
        transferDelayedWETHOwnership();
    }

    /// @notice Deploy all of the proxies
    function deployProxies() public {
        console.log("Deploying proxies");

        deployERC1967Proxy("OptimismPortalProxy");
        deployERC1967Proxy("SystemConfigProxy");
        deployL1StandardBridgeProxy();
        deployL1CrossDomainMessengerProxy();
        deployERC1967Proxy("OptimismMintableERC20FactoryProxy");
        deployERC1967Proxy("L1ERC721BridgeProxy");

        // Both the DisputeGameFactory and L2OutputOracle proxies are deployed regardless of whether fault proofs is
        // enabled to prevent a nastier refactor to the deploy scripts. In the future, the L2OutputOracle will be
        // removed. If fault proofs are not enabled, the DisputeGameFactory proxy will be unused.
        deployERC1967Proxy("DisputeGameFactoryProxy");
        deployERC1967Proxy("L2OutputOracleProxy");
        deployERC1967Proxy("DelayedWETHProxy");
        deployERC1967Proxy("PermissionedDelayedWETHProxy");
        deployERC1967Proxy("AnchorStateRegistryProxy");

        transferAddressManagerOwnership(); // to the ProxyAdmin
    }

    /// @notice Deploy all of the implementations
    function deployImplementations() public {
        console.log("Deploying implementations");
        deployL1CrossDomainMessenger();
        deployOptimismMintableERC20Factory();
        deploySystemConfig();
        deployL1StandardBridge();
        deployL1ERC721Bridge();
        deployOptimismPortal();
        deployL2OutputOracle();
        // Fault proofs
        deployOptimismPortal2();
        deployDisputeGameFactory();
        deployDelayedWETH();
        deployPreimageOracle();
        deployMips();
        deployAnchorStateRegistry();
    }

    /// @notice Initialize all of the implementations
    function initializeImplementations() public {
        console.log("Initializing implementations");
        // Selectively initialize either the original OptimismPortal or the new OptimismPortal2. Since this will upgrade
        // the proxy, we cannot initialize both.
        if (cfg.useFaultProofs()) {
            console.log("Fault proofs enabled. Initializing the OptimismPortal proxy with the OptimismPortal2.");
            initializeOptimismPortal2();
        } else {
            initializeOptimismPortal();
        }

        initializeSystemConfig();
        initializeL1StandardBridge();
        initializeL1ERC721Bridge();
        initializeOptimismMintableERC20Factory();
        initializeL1CrossDomainMessenger();
        initializeL2OutputOracle();
        initializeDisputeGameFactory();
        initializeDelayedWETH();
        initializePermissionedDelayedWETH();
        initializeAnchorStateRegistry();
    }

    /// @notice Add AltDA setup to the OP chain
    function setupOpAltDA() public {
        console.log("Deploying OP AltDA");
        deployDataAvailabilityChallengeProxy();
        deployDataAvailabilityChallenge();
        initializeDataAvailabilityChallenge();
    }

    ////////////////////////////////////////////////////////////////
    //              Non-Proxied Deployment Functions              //
    ////////////////////////////////////////////////////////////////

    /// @notice Deploy the Safe
    function deploySafe(string memory _name) public broadcast returns (address addr_) {
        address[] memory owners = new address[](0);
        addr_ = deploySafe(_name, owners, 1, true);
    }

    /// @notice Deploy a new Safe contract. If the keepDeployer option is used to enable further setup actions, then
    ///         the removeDeployerFromSafe() function should be called on that safe after setup is complete.
    ///         Note this function does not have the broadcast modifier.
    /// @param _name The name of the Safe to deploy.
    /// @param _owners The owners of the Safe.
    /// @param _threshold The threshold of the Safe.
    /// @param _keepDeployer Wether or not the deployer address will be added as an owner of the Safe.
    function deploySafe(
        string memory _name,
        address[] memory _owners,
        uint256 _threshold,
        bool _keepDeployer
    )
        public
        returns (address addr_)
    {
        bytes32 salt = keccak256(abi.encode(_name, _implSalt()));
        console.log("Deploying safe: %s with salt %s", _name, vm.toString(salt));
        (SafeProxyFactory safeProxyFactory, Safe safeSingleton) = _getSafeFactory();

        if (_keepDeployer) {
            address[] memory expandedOwners = new address[](_owners.length + 1);
            // By always adding msg.sender first we know that the previousOwner will be SENTINEL_OWNERS, which makes it
            // easier to call removeOwner later.
            expandedOwners[0] = msg.sender;
            for (uint256 i = 0; i < _owners.length; i++) {
                expandedOwners[i + 1] = _owners[i];
            }
            _owners = expandedOwners;
        }

        bytes memory initData = abi.encodeCall(
            Safe.setup, (_owners, _threshold, address(0), hex"", address(0), address(0), 0, payable(address(0)))
        );
        addr_ = address(safeProxyFactory.createProxyWithNonce(address(safeSingleton), initData, uint256(salt)));

        save(_name, addr_);
        console.log("New safe: %s deployed at %s\n    Note that this safe is owned by the deployer key", _name, addr_);
    }

    /// @notice If the keepDeployer option was used with deploySafe(), this function can be used to remove the deployer.
    ///         Note this function does not have the broadcast modifier.
    function removeDeployerFromSafe(string memory _name, uint256 _newThreshold) public {
        Safe safe = Safe(mustGetAddress(_name));

        // The sentinel address is used to mark the start and end of the linked list of owners in the Safe.
        address sentinelOwners = address(0x1);

        // Because deploySafe() always adds msg.sender first (if keepDeployer is true), we know that the previousOwner
        // will be sentinelOwners.
        _callViaSafe({
            _safe: safe,
            _target: address(safe),
            _data: abi.encodeCall(OwnerManager.removeOwner, (sentinelOwners, msg.sender, _newThreshold))
        });
        console.log("Removed deployer owner from ", _name);
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

        require(EIP1967Helper.getAdmin(address(proxy)) == proxyAdmin);

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

    /// @notice Deploys an ERC1967Proxy contract with the ProxyAdmin as the owner.
    /// @param _name The name of the proxy contract to be deployed.
    /// @return addr_ The address of the deployed proxy contract.
    function deployERC1967Proxy(string memory _name) public returns (address addr_) {
        addr_ = deployERC1967ProxyWithOwner(_name, mustGetAddress("ProxyAdmin"));
    }

    /// @notice Deploys an ERC1967Proxy contract with a specified owner.
    /// @param _name The name of the proxy contract to be deployed.
    /// @param _proxyOwner The address of the owner of the proxy contract.
    /// @return addr_ The address of the deployed proxy contract.
    function deployERC1967ProxyWithOwner(
        string memory _name,
        address _proxyOwner
    )
        public
        broadcast
        returns (address addr_)
    {
        console.log(string.concat("Deploying ERC1967 proxy for ", _name));
        Proxy proxy = new Proxy({ _admin: _proxyOwner });

        require(EIP1967Helper.getAdmin(address(proxy)) == _proxyOwner);

        save(_name, address(proxy));
        console.log("   at %s", address(proxy));
        addr_ = address(proxy);
    }

    /// @notice Deploy the DataAvailabilityChallengeProxy
    function deployDataAvailabilityChallengeProxy() public broadcast returns (address addr_) {
        console.log("Deploying proxy for DataAvailabilityChallenge");
        address proxyAdmin = mustGetAddress("ProxyAdmin");
        Proxy proxy = new Proxy({ _admin: proxyAdmin });

        require(EIP1967Helper.getAdmin(address(proxy)) == proxyAdmin);

        save("DataAvailabilityChallengeProxy", address(proxy));
        console.log("DataAvailabilityChallengeProxy deployed at %s", address(proxy));

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
        L1CrossDomainMessenger messenger = new L1CrossDomainMessenger{ salt: _implSalt() }();

        save("L1CrossDomainMessenger", address(messenger));
        console.log("L1CrossDomainMessenger deployed at %s", address(messenger));

        // Override the `L1CrossDomainMessenger` contract to the deployed implementation. This is necessary
        // to check the `L1CrossDomainMessenger` implementation alongside dependent contracts, which
        // are always proxies.
        Types.ContractSet memory contracts = _proxiesUnstrict();
        contracts.L1CrossDomainMessenger = address(messenger);
        ChainAssertions.checkL1CrossDomainMessenger({ _contracts: contracts, _vm: vm, _isProxy: false });

        addr_ = address(messenger);
    }

    /// @notice Deploy the OptimismPortal
    function deployOptimismPortal() public broadcast returns (address addr_) {
        console.log("Deploying OptimismPortal implementation");
        if (cfg.useInterop()) {
            console.log("Attempting to deploy OptimismPortal with interop, this config is a noop");
        }
        addr_ = address(new OptimismPortal{ salt: _implSalt() }());
        save("OptimismPortal", addr_);
        console.log("OptimismPortal deployed at %s", addr_);

        // Override the `OptimismPortal` contract to the deployed implementation. This is necessary
        // to check the `OptimismPortal` implementation alongside dependent contracts, which
        // are always proxies.
        Types.ContractSet memory contracts = _proxiesUnstrict();
        contracts.OptimismPortal = addr_;
        ChainAssertions.checkOptimismPortal({ _contracts: contracts, _cfg: cfg, _isProxy: false });
    }

    /// @notice Deploy the OptimismPortal2
    function deployOptimismPortal2() public broadcast returns (address addr_) {
        console.log("Deploying OptimismPortal2 implementation");

        // Could also verify this inside DeployConfig but doing it here is a bit more reliable.
        require(
            uint32(cfg.respectedGameType()) == cfg.respectedGameType(), "Deploy: respectedGameType must fit into uint32"
        );

        if (cfg.useInterop()) {
            addr_ = address(
                new OptimismPortalInterop{ salt: _implSalt() }({
                    _proofMaturityDelaySeconds: cfg.proofMaturityDelaySeconds(),
                    _disputeGameFinalityDelaySeconds: cfg.disputeGameFinalityDelaySeconds()
                })
            );
        } else {
            addr_ = address(
                new OptimismPortal2{ salt: _implSalt() }({
                    _proofMaturityDelaySeconds: cfg.proofMaturityDelaySeconds(),
                    _disputeGameFinalityDelaySeconds: cfg.disputeGameFinalityDelaySeconds()
                })
            );
        }

        save("OptimismPortal2", addr_);
        console.log("OptimismPortal2 deployed at %s", addr_);

        // Override the `OptimismPortal2` contract to the deployed implementation. This is necessary
        // to check the `OptimismPortal2` implementation alongside dependent contracts, which
        // are always proxies.
        Types.ContractSet memory contracts = _proxiesUnstrict();
        contracts.OptimismPortal2 = addr_;
        ChainAssertions.checkOptimismPortal2({ _contracts: contracts, _cfg: cfg, _isProxy: false });
    }

    /// @notice Deploy the L2OutputOracle
    function deployL2OutputOracle() public broadcast returns (address addr_) {
        console.log("Deploying L2OutputOracle implementation");
        L2OutputOracle oracle = new L2OutputOracle{ salt: _implSalt() }();

        save("L2OutputOracle", address(oracle));
        console.log("L2OutputOracle deployed at %s", address(oracle));

        // Override the `L2OutputOracle` contract to the deployed implementation. This is necessary
        // to check the `L2OutputOracle` implementation alongside dependent contracts, which
        // are always proxies.
        Types.ContractSet memory contracts = _proxiesUnstrict();
        contracts.L2OutputOracle = address(oracle);
        ChainAssertions.checkL2OutputOracle({
            _contracts: contracts,
            _cfg: cfg,
            _l2OutputOracleStartingTimestamp: 0,
            _isProxy: false
        });

        addr_ = address(oracle);
    }

    /// @notice Deploy the OptimismMintableERC20Factory
    function deployOptimismMintableERC20Factory() public broadcast returns (address addr_) {
        console.log("Deploying OptimismMintableERC20Factory implementation");
        OptimismMintableERC20Factory factory = new OptimismMintableERC20Factory{ salt: _implSalt() }();

        save("OptimismMintableERC20Factory", address(factory));
        console.log("OptimismMintableERC20Factory deployed at %s", address(factory));

        // Override the `OptimismMintableERC20Factory` contract to the deployed implementation. This is necessary
        // to check the `OptimismMintableERC20Factory` implementation alongside dependent contracts, which
        // are always proxies.
        Types.ContractSet memory contracts = _proxiesUnstrict();
        contracts.OptimismMintableERC20Factory = address(factory);
        ChainAssertions.checkOptimismMintableERC20Factory({ _contracts: contracts, _isProxy: false });

        addr_ = address(factory);
    }

    /// @notice Deploy the DisputeGameFactory
    function deployDisputeGameFactory() public broadcast returns (address addr_) {
        console.log("Deploying DisputeGameFactory implementation");
        DisputeGameFactory factory = new DisputeGameFactory{ salt: _implSalt() }();
        save("DisputeGameFactory", address(factory));
        console.log("DisputeGameFactory deployed at %s", address(factory));

        // Override the `DisputeGameFactory` contract to the deployed implementation. This is necessary to check the
        // `DisputeGameFactory` implementation alongside dependent contracts, which are always proxies.
        Types.ContractSet memory contracts = _proxiesUnstrict();
        contracts.DisputeGameFactory = address(factory);
        ChainAssertions.checkDisputeGameFactory({ _contracts: contracts, _expectedOwner: address(0) });

        addr_ = address(factory);
    }

    function deployDelayedWETH() public broadcast returns (address addr_) {
        console.log("Deploying DelayedWETH implementation");
        DelayedWETH weth = new DelayedWETH{ salt: _implSalt() }(cfg.faultGameWithdrawalDelay());
        save("DelayedWETH", address(weth));
        console.log("DelayedWETH deployed at %s", address(weth));

        // Override the `DelayedWETH` contract to the deployed implementation. This is necessary
        // to check the `DelayedWETH` implementation alongside dependent contracts, which are
        // always proxies.
        Types.ContractSet memory contracts = _proxiesUnstrict();
        contracts.DelayedWETH = address(weth);
        ChainAssertions.checkDelayedWETH({
            _contracts: contracts,
            _cfg: cfg,
            _isProxy: false,
            _expectedOwner: address(0)
        });

        addr_ = address(weth);
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

        addr_ = address(versions);
    }

    /// @notice Deploy the PreimageOracle
    function deployPreimageOracle() public broadcast returns (address addr_) {
        console.log("Deploying PreimageOracle implementation");
        PreimageOracle preimageOracle = new PreimageOracle{ salt: _implSalt() }({
            _minProposalSize: cfg.preimageOracleMinProposalSize(),
            _challengePeriod: cfg.preimageOracleChallengePeriod()
        });
        save("PreimageOracle", address(preimageOracle));
        console.log("PreimageOracle deployed at %s", address(preimageOracle));

        addr_ = address(preimageOracle);
    }

    /// @notice Deploy Mips
    function deployMips() public broadcast returns (address addr_) {
        console.log("Deploying Mips implementation");
        MIPS mips = new MIPS{ salt: _implSalt() }(IPreimageOracle(mustGetAddress("PreimageOracle")));
        save("Mips", address(mips));
        console.log("MIPS deployed at %s", address(mips));

        addr_ = address(mips);
    }

    /// @notice Deploy the AnchorStateRegistry
    function deployAnchorStateRegistry() public broadcast returns (address addr_) {
        console.log("Deploying AnchorStateRegistry implementation");
        AnchorStateRegistry anchorStateRegistry =
            new AnchorStateRegistry{ salt: _implSalt() }(DisputeGameFactory(mustGetAddress("DisputeGameFactoryProxy")));
        save("AnchorStateRegistry", address(anchorStateRegistry));
        console.log("AnchorStateRegistry deployed at %s", address(anchorStateRegistry));

        addr_ = address(anchorStateRegistry);
    }

    /// @notice Deploy the SystemConfig
    function deploySystemConfig() public broadcast returns (address addr_) {
        console.log("Deploying SystemConfig implementation");
        if (cfg.useInterop()) {
            addr_ = address(new SystemConfigInterop{ salt: _implSalt() }());
        } else {
            addr_ = address(new SystemConfig{ salt: _implSalt() }());
        }
        save("SystemConfig", addr_);
        console.log("SystemConfig deployed at %s", addr_);

        // Override the `SystemConfig` contract to the deployed implementation. This is necessary
        // to check the `SystemConfig` implementation alongside dependent contracts, which
        // are always proxies.
        Types.ContractSet memory contracts = _proxiesUnstrict();
        contracts.SystemConfig = addr_;
        ChainAssertions.checkSystemConfig({ _contracts: contracts, _cfg: cfg, _isProxy: false });
    }

    /// @notice Deploy the L1StandardBridge
    function deployL1StandardBridge() public broadcast returns (address addr_) {
        console.log("Deploying L1StandardBridge implementation");

        L1StandardBridge bridge = new L1StandardBridge{ salt: _implSalt() }();

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
        L1ERC721Bridge bridge = new L1ERC721Bridge{ salt: _implSalt() }();

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

    /// @notice Deploy the DataAvailabilityChallenge
    function deployDataAvailabilityChallenge() public broadcast returns (address addr_) {
        console.log("Deploying DataAvailabilityChallenge implementation");
        DataAvailabilityChallenge dac = new DataAvailabilityChallenge();
        save("DataAvailabilityChallenge", address(dac));
        console.log("DataAvailabilityChallenge deployed at %s", address(dac));

        addr_ = address(dac);
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
    function initializeDisputeGameFactory() public broadcast {
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

        ChainAssertions.checkDisputeGameFactory({ _contracts: _proxiesUnstrict(), _expectedOwner: msg.sender });
    }

    function initializeDelayedWETH() public broadcast {
        console.log("Upgrading and initializing DelayedWETH proxy");
        address delayedWETHProxy = mustGetAddress("DelayedWETHProxy");
        address delayedWETH = mustGetAddress("DelayedWETH");
        address superchainConfigProxy = mustGetAddress("SuperchainConfigProxy");

        _upgradeAndCallViaSafe({
            _proxy: payable(delayedWETHProxy),
            _implementation: delayedWETH,
            _innerCallData: abi.encodeCall(DelayedWETH.initialize, (msg.sender, SuperchainConfig(superchainConfigProxy)))
        });

        string memory version = DelayedWETH(payable(delayedWETHProxy)).version();
        console.log("DelayedWETH version: %s", version);

        ChainAssertions.checkDelayedWETH({
            _contracts: _proxiesUnstrict(),
            _cfg: cfg,
            _isProxy: true,
            _expectedOwner: msg.sender
        });
    }

    function initializePermissionedDelayedWETH() public broadcast {
        console.log("Upgrading and initializing permissioned DelayedWETH proxy");
        address delayedWETHProxy = mustGetAddress("PermissionedDelayedWETHProxy");
        address delayedWETH = mustGetAddress("DelayedWETH");
        address superchainConfigProxy = mustGetAddress("SuperchainConfigProxy");

        _upgradeAndCallViaSafe({
            _proxy: payable(delayedWETHProxy),
            _implementation: delayedWETH,
            _innerCallData: abi.encodeCall(DelayedWETH.initialize, (msg.sender, SuperchainConfig(superchainConfigProxy)))
        });

        string memory version = DelayedWETH(payable(delayedWETHProxy)).version();
        console.log("DelayedWETH version: %s", version);

        ChainAssertions.checkPermissionedDelayedWETH({
            _contracts: _proxiesUnstrict(),
            _cfg: cfg,
            _isProxy: true,
            _expectedOwner: msg.sender
        });
    }

    function initializeAnchorStateRegistry() public broadcast {
        console.log("Upgrading and initializing AnchorStateRegistry proxy");
        address anchorStateRegistryProxy = mustGetAddress("AnchorStateRegistryProxy");
        address anchorStateRegistry = mustGetAddress("AnchorStateRegistry");
        SuperchainConfig superchainConfig = SuperchainConfig(mustGetAddress("SuperchainConfigProxy"));

        AnchorStateRegistry.StartingAnchorRoot[] memory roots = new AnchorStateRegistry.StartingAnchorRoot[](5);
        roots[0] = AnchorStateRegistry.StartingAnchorRoot({
            gameType: GameTypes.CANNON,
            outputRoot: OutputRoot({
                root: Hash.wrap(cfg.faultGameGenesisOutputRoot()),
                l2BlockNumber: cfg.faultGameGenesisBlock()
            })
        });
        roots[1] = AnchorStateRegistry.StartingAnchorRoot({
            gameType: GameTypes.PERMISSIONED_CANNON,
            outputRoot: OutputRoot({
                root: Hash.wrap(cfg.faultGameGenesisOutputRoot()),
                l2BlockNumber: cfg.faultGameGenesisBlock()
            })
        });
        roots[2] = AnchorStateRegistry.StartingAnchorRoot({
            gameType: GameTypes.ALPHABET,
            outputRoot: OutputRoot({
                root: Hash.wrap(cfg.faultGameGenesisOutputRoot()),
                l2BlockNumber: cfg.faultGameGenesisBlock()
            })
        });
        roots[3] = AnchorStateRegistry.StartingAnchorRoot({
            gameType: GameTypes.ASTERISC,
            outputRoot: OutputRoot({
                root: Hash.wrap(cfg.faultGameGenesisOutputRoot()),
                l2BlockNumber: cfg.faultGameGenesisBlock()
            })
        });
        roots[4] = AnchorStateRegistry.StartingAnchorRoot({
            gameType: GameTypes.FAST,
            outputRoot: OutputRoot({
                root: Hash.wrap(cfg.faultGameGenesisOutputRoot()),
                l2BlockNumber: cfg.faultGameGenesisBlock()
            })
        });

        _upgradeAndCallViaSafe({
            _proxy: payable(anchorStateRegistryProxy),
            _implementation: anchorStateRegistry,
            _innerCallData: abi.encodeCall(AnchorStateRegistry.initialize, (roots, superchainConfig))
        });

        string memory version = AnchorStateRegistry(payable(anchorStateRegistryProxy)).version();
        console.log("AnchorStateRegistry version: %s", version);
    }

    /// @notice Initialize the SystemConfig
    function initializeSystemConfig() public broadcast {
        console.log("Upgrading and initializing SystemConfig proxy");
        address systemConfigProxy = mustGetAddress("SystemConfigProxy");
        address systemConfig = mustGetAddress("SystemConfig");

        bytes32 batcherHash = bytes32(uint256(uint160(cfg.batchSenderAddress())));

        address customGasTokenAddress = Constants.ETHER;
        if (cfg.useCustomGasToken()) {
            customGasTokenAddress = cfg.customGasTokenAddress();
        }

        _upgradeAndCallViaSafe({
            _proxy: payable(systemConfigProxy),
            _implementation: systemConfig,
            _innerCallData: abi.encodeCall(
                SystemConfig.initialize,
                (
                    cfg.finalSystemOwner(),
                    cfg.basefeeScalar(),
                    cfg.blobbasefeeScalar(),
                    batcherHash,
                    uint64(cfg.l2GenesisBlockGasLimit()),
                    cfg.p2pSequencerAddress(),
                    Constants.DEFAULT_RESOURCE_CONFIG(),
                    cfg.batchInboxAddress(),
                    SystemConfig.Addresses({
                        l1CrossDomainMessenger: mustGetAddress("L1CrossDomainMessengerProxy"),
                        l1ERC721Bridge: mustGetAddress("L1ERC721BridgeProxy"),
                        l1StandardBridge: mustGetAddress("L1StandardBridgeProxy"),
                        disputeGameFactory: mustGetAddress("DisputeGameFactoryProxy"),
                        optimismPortal: mustGetAddress("OptimismPortalProxy"),
                        optimismMintableERC20Factory: mustGetAddress("OptimismMintableERC20FactoryProxy"),
                        gasPayingToken: customGasTokenAddress
                    })
                )
            )
        });

        SystemConfig config = SystemConfig(systemConfigProxy);
        string memory version = config.version();
        console.log("SystemConfig version: %s", version);

        ChainAssertions.checkSystemConfig({ _contracts: _proxies(), _cfg: cfg, _isProxy: true });
    }

    /// @notice Initialize the L1StandardBridge
    function initializeL1StandardBridge() public broadcast {
        console.log("Upgrading and initializing L1StandardBridge proxy");
        ProxyAdmin proxyAdmin = ProxyAdmin(mustGetAddress("ProxyAdmin"));
        address l1StandardBridgeProxy = mustGetAddress("L1StandardBridgeProxy");
        address l1StandardBridge = mustGetAddress("L1StandardBridge");
        address l1CrossDomainMessengerProxy = mustGetAddress("L1CrossDomainMessengerProxy");
        address superchainConfigProxy = mustGetAddress("SuperchainConfigProxy");
        address systemConfigProxy = mustGetAddress("SystemConfigProxy");

        uint256 proxyType = uint256(proxyAdmin.proxyType(l1StandardBridgeProxy));
        Safe safe = Safe(mustGetAddress("SystemOwnerSafe"));
        if (proxyType != uint256(ProxyAdmin.ProxyType.CHUGSPLASH)) {
            _callViaSafe({
                _safe: safe,
                _target: address(proxyAdmin),
                _data: abi.encodeCall(ProxyAdmin.setProxyType, (l1StandardBridgeProxy, ProxyAdmin.ProxyType.CHUGSPLASH))
            });
        }
        require(uint256(proxyAdmin.proxyType(l1StandardBridgeProxy)) == uint256(ProxyAdmin.ProxyType.CHUGSPLASH));

        _upgradeAndCallViaSafe({
            _proxy: payable(l1StandardBridgeProxy),
            _implementation: l1StandardBridge,
            _innerCallData: abi.encodeCall(
                L1StandardBridge.initialize,
                (
                    L1CrossDomainMessenger(l1CrossDomainMessengerProxy),
                    SuperchainConfig(superchainConfigProxy),
                    SystemConfig(systemConfigProxy)
                )
            )
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
        address l1CrossDomainMessengerProxy = mustGetAddress("L1CrossDomainMessengerProxy");
        address superchainConfigProxy = mustGetAddress("SuperchainConfigProxy");

        _upgradeAndCallViaSafe({
            _proxy: payable(l1ERC721BridgeProxy),
            _implementation: l1ERC721Bridge,
            _innerCallData: abi.encodeCall(
                L1ERC721Bridge.initialize,
                (L1CrossDomainMessenger(payable(l1CrossDomainMessengerProxy)), SuperchainConfig(superchainConfigProxy))
            )
        });

        L1ERC721Bridge bridge = L1ERC721Bridge(l1ERC721BridgeProxy);
        string memory version = bridge.version();
        console.log("L1ERC721Bridge version: %s", version);

        ChainAssertions.checkL1ERC721Bridge({ _contracts: _proxies(), _isProxy: true });
    }

    /// @notice Initialize the OptimismMintableERC20Factory
    function initializeOptimismMintableERC20Factory() public broadcast {
        console.log("Upgrading and initializing OptimismMintableERC20Factory proxy");
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

        ChainAssertions.checkOptimismMintableERC20Factory({ _contracts: _proxies(), _isProxy: true });
    }

    /// @notice initializeL1CrossDomainMessenger
    function initializeL1CrossDomainMessenger() public broadcast {
        console.log("Upgrading and initializing L1CrossDomainMessenger proxy");
        ProxyAdmin proxyAdmin = ProxyAdmin(mustGetAddress("ProxyAdmin"));
        address l1CrossDomainMessengerProxy = mustGetAddress("L1CrossDomainMessengerProxy");
        address l1CrossDomainMessenger = mustGetAddress("L1CrossDomainMessenger");
        address superchainConfigProxy = mustGetAddress("SuperchainConfigProxy");
        address optimismPortalProxy = mustGetAddress("OptimismPortalProxy");
        address systemConfigProxy = mustGetAddress("SystemConfigProxy");

        uint256 proxyType = uint256(proxyAdmin.proxyType(l1CrossDomainMessengerProxy));
        Safe safe = Safe(mustGetAddress("SystemOwnerSafe"));
        if (proxyType != uint256(ProxyAdmin.ProxyType.RESOLVED)) {
            _callViaSafe({
                _safe: safe,
                _target: address(proxyAdmin),
                _data: abi.encodeCall(ProxyAdmin.setProxyType, (l1CrossDomainMessengerProxy, ProxyAdmin.ProxyType.RESOLVED))
            });
        }
        require(uint256(proxyAdmin.proxyType(l1CrossDomainMessengerProxy)) == uint256(ProxyAdmin.ProxyType.RESOLVED));

        string memory contractName = "OVM_L1CrossDomainMessenger";
        string memory implName = proxyAdmin.implementationName(l1CrossDomainMessenger);
        if (keccak256(bytes(contractName)) != keccak256(bytes(implName))) {
            _callViaSafe({
                _safe: safe,
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
                L1CrossDomainMessenger.initialize,
                (
                    SuperchainConfig(superchainConfigProxy),
                    OptimismPortal(payable(optimismPortalProxy)),
                    SystemConfig(systemConfigProxy)
                )
            )
        });

        L1CrossDomainMessenger messenger = L1CrossDomainMessenger(l1CrossDomainMessengerProxy);
        string memory version = messenger.version();
        console.log("L1CrossDomainMessenger version: %s", version);

        ChainAssertions.checkL1CrossDomainMessenger({ _contracts: _proxies(), _vm: vm, _isProxy: true });
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
                L2OutputOracle.initialize,
                (
                    cfg.l2OutputOracleSubmissionInterval(),
                    cfg.l2BlockTime(),
                    cfg.l2OutputOracleStartingBlockNumber(),
                    cfg.l2OutputOracleStartingTimestamp(),
                    cfg.l2OutputOracleProposer(),
                    cfg.l2OutputOracleChallenger(),
                    cfg.finalizationPeriodSeconds()
                )
            )
        });

        L2OutputOracle oracle = L2OutputOracle(l2OutputOracleProxy);
        string memory version = oracle.version();
        console.log("L2OutputOracle version: %s", version);

        ChainAssertions.checkL2OutputOracle({
            _contracts: _proxies(),
            _cfg: cfg,
            _l2OutputOracleStartingTimestamp: cfg.l2OutputOracleStartingTimestamp(),
            _isProxy: true
        });
    }

    /// @notice Initialize the OptimismPortal
    function initializeOptimismPortal() public broadcast {
        console.log("Upgrading and initializing OptimismPortal proxy");
        address optimismPortalProxy = mustGetAddress("OptimismPortalProxy");
        address optimismPortal = mustGetAddress("OptimismPortal");
        address l2OutputOracleProxy = mustGetAddress("L2OutputOracleProxy");
        address systemConfigProxy = mustGetAddress("SystemConfigProxy");
        address superchainConfigProxy = mustGetAddress("SuperchainConfigProxy");

        _upgradeAndCallViaSafe({
            _proxy: payable(optimismPortalProxy),
            _implementation: optimismPortal,
            _innerCallData: abi.encodeCall(
                OptimismPortal.initialize,
                (
                    L2OutputOracle(l2OutputOracleProxy),
                    SystemConfig(systemConfigProxy),
                    SuperchainConfig(superchainConfigProxy)
                )
            )
        });

        OptimismPortal portal = OptimismPortal(payable(optimismPortalProxy));
        string memory version = portal.version();
        console.log("OptimismPortal version: %s", version);

        ChainAssertions.checkOptimismPortal({ _contracts: _proxies(), _cfg: cfg, _isProxy: true });
    }

    /// @notice Initialize the OptimismPortal2
    function initializeOptimismPortal2() public broadcast {
        console.log("Upgrading and initializing OptimismPortal2 proxy");
        address optimismPortalProxy = mustGetAddress("OptimismPortalProxy");
        address optimismPortal2 = mustGetAddress("OptimismPortal2");
        address disputeGameFactoryProxy = mustGetAddress("DisputeGameFactoryProxy");
        address systemConfigProxy = mustGetAddress("SystemConfigProxy");
        address superchainConfigProxy = mustGetAddress("SuperchainConfigProxy");

        _upgradeAndCallViaSafe({
            _proxy: payable(optimismPortalProxy),
            _implementation: optimismPortal2,
            _innerCallData: abi.encodeCall(
                OptimismPortal2.initialize,
                (
                    DisputeGameFactory(disputeGameFactoryProxy),
                    SystemConfig(systemConfigProxy),
                    SuperchainConfig(superchainConfigProxy),
                    GameType.wrap(uint32(cfg.respectedGameType()))
                )
            )
        });

        OptimismPortal2 portal = OptimismPortal2(payable(optimismPortalProxy));
        string memory version = portal.version();
        console.log("OptimismPortal2 version: %s", version);

        ChainAssertions.checkOptimismPortal2({ _contracts: _proxies(), _cfg: cfg, _isProxy: true });
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
    }

    /// @notice Transfer ownership of the DisputeGameFactory contract to the final system owner
    function transferDisputeGameFactoryOwnership() public broadcast {
        console.log("Transferring DisputeGameFactory ownership to Safe");
        DisputeGameFactory disputeGameFactory = DisputeGameFactory(mustGetAddress("DisputeGameFactoryProxy"));
        address owner = disputeGameFactory.owner();

        address safe = mustGetAddress("SystemOwnerSafe");
        if (owner != safe) {
            disputeGameFactory.transferOwnership(safe);
            console.log("DisputeGameFactory ownership transferred to Safe at: %s", safe);
        }
        ChainAssertions.checkDisputeGameFactory({ _contracts: _proxies(), _expectedOwner: safe });
    }

    /// @notice Transfer ownership of the DelayedWETH contract to the final system owner
    function transferDelayedWETHOwnership() public broadcast {
        console.log("Transferring DelayedWETH ownership to Safe");
        DelayedWETH weth = DelayedWETH(mustGetAddress("DelayedWETHProxy"));
        address owner = weth.owner();

        address safe = mustGetAddress("SystemOwnerSafe");
        if (owner != safe) {
            weth.transferOwnership(safe);
            console.log("DelayedWETH ownership transferred to Safe at: %s", safe);
        }
        ChainAssertions.checkDelayedWETH({ _contracts: _proxies(), _cfg: cfg, _isProxy: true, _expectedOwner: safe });
    }

    /// @notice Transfer ownership of the permissioned DelayedWETH contract to the final system owner
    function transferPermissionedDelayedWETHOwnership() public broadcast {
        console.log("Transferring permissioned DelayedWETH ownership to Safe");
        DelayedWETH weth = DelayedWETH(mustGetAddress("PermissionedDelayedWETHProxy"));
        address owner = weth.owner();

        address safe = mustGetAddress("SystemOwnerSafe");
        if (owner != safe) {
            weth.transferOwnership(safe);
            console.log("DelayedWETH ownership transferred to Safe at: %s", safe);
        }
        ChainAssertions.checkPermissionedDelayedWETH({
            _contracts: _proxies(),
            _cfg: cfg,
            _isProxy: true,
            _expectedOwner: safe
        });
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
            if (Process.run(commands).length == 0) {
                revert("Cannon prestate dump not found, generate it with `make cannon-prestate` in the monorepo root.");
            }
            commands[2] = string.concat("cat ", filePath, " | jq -r .pre");
            mipsAbsolutePrestate_ = Claim.wrap(abi.decode(Process.run(commands), (bytes32)));
            console.log(
                "[Cannon Dispute Game] Using devnet MIPS Absolute prestate: %s",
                vm.toString(Claim.unwrap(mipsAbsolutePrestate_))
            );
        } else {
            console.log(
                "[Cannon Dispute Game] Using absolute prestate from config: %x", cfg.faultGameAbsolutePrestate()
            );
            mipsAbsolutePrestate_ = Claim.wrap(bytes32(cfg.faultGameAbsolutePrestate()));
        }
    }

    /// @notice Sets the implementation for the `CANNON` game type in the `DisputeGameFactory`
    function setCannonFaultGameImplementation(bool _allowUpgrade) public broadcast {
        console.log("Setting Cannon FaultDisputeGame implementation");
        DisputeGameFactory factory = DisputeGameFactory(mustGetAddress("DisputeGameFactoryProxy"));
        DelayedWETH weth = DelayedWETH(mustGetAddress("DelayedWETHProxy"));

        // Set the Cannon FaultDisputeGame implementation in the factory.
        _setFaultGameImplementation({
            _factory: factory,
            _allowUpgrade: _allowUpgrade,
            _params: FaultDisputeGameParams({
                anchorStateRegistry: AnchorStateRegistry(mustGetAddress("AnchorStateRegistryProxy")),
                weth: weth,
                gameType: GameTypes.CANNON,
                absolutePrestate: loadMipsAbsolutePrestate(),
                faultVm: IBigStepper(mustGetAddress("Mips")),
                maxGameDepth: cfg.faultGameMaxDepth(),
                maxClockDuration: Duration.wrap(uint64(cfg.faultGameMaxClockDuration()))
            })
        });
    }

    /// @notice Sets the implementation for the `PERMISSIONED_CANNON` game type in the `DisputeGameFactory`
    function setPermissionedCannonFaultGameImplementation(bool _allowUpgrade) public broadcast {
        console.log("Setting Cannon PermissionedDisputeGame implementation");
        DisputeGameFactory factory = DisputeGameFactory(mustGetAddress("DisputeGameFactoryProxy"));
        DelayedWETH weth = DelayedWETH(mustGetAddress("PermissionedDelayedWETHProxy"));

        // Set the Cannon FaultDisputeGame implementation in the factory.
        _setFaultGameImplementation({
            _factory: factory,
            _allowUpgrade: _allowUpgrade,
            _params: FaultDisputeGameParams({
                anchorStateRegistry: AnchorStateRegistry(mustGetAddress("AnchorStateRegistryProxy")),
                weth: weth,
                gameType: GameTypes.PERMISSIONED_CANNON,
                absolutePrestate: loadMipsAbsolutePrestate(),
                faultVm: IBigStepper(mustGetAddress("Mips")),
                maxGameDepth: cfg.faultGameMaxDepth(),
                maxClockDuration: Duration.wrap(uint64(cfg.faultGameMaxClockDuration()))
            })
        });
    }

    /// @notice Sets the implementation for the `ALPHABET` game type in the `DisputeGameFactory`
    function setAlphabetFaultGameImplementation(bool _allowUpgrade) public onlyDevnet broadcast {
        console.log("Setting Alphabet FaultDisputeGame implementation");
        DisputeGameFactory factory = DisputeGameFactory(mustGetAddress("DisputeGameFactoryProxy"));
        DelayedWETH weth = DelayedWETH(mustGetAddress("DelayedWETHProxy"));

        Claim outputAbsolutePrestate = Claim.wrap(bytes32(cfg.faultGameAbsolutePrestate()));
        _setFaultGameImplementation({
            _factory: factory,
            _allowUpgrade: _allowUpgrade,
            _params: FaultDisputeGameParams({
                anchorStateRegistry: AnchorStateRegistry(mustGetAddress("AnchorStateRegistryProxy")),
                weth: weth,
                gameType: GameTypes.ALPHABET,
                absolutePrestate: outputAbsolutePrestate,
                faultVm: IBigStepper(new AlphabetVM(outputAbsolutePrestate, PreimageOracle(mustGetAddress("PreimageOracle")))),
                // The max depth for the alphabet trace is always 3. Add 1 because split depth is fully inclusive.
                maxGameDepth: cfg.faultGameSplitDepth() + 3 + 1,
                maxClockDuration: Duration.wrap(uint64(cfg.faultGameMaxClockDuration()))
            })
        });
    }

    /// @notice Sets the implementation for the `ALPHABET` game type in the `DisputeGameFactory`
    function setFastFaultGameImplementation(bool _allowUpgrade) public onlyDevnet broadcast {
        console.log("Setting Fast FaultDisputeGame implementation");
        DisputeGameFactory factory = DisputeGameFactory(mustGetAddress("DisputeGameFactoryProxy"));
        DelayedWETH weth = DelayedWETH(mustGetAddress("DelayedWETHProxy"));

        Claim outputAbsolutePrestate = Claim.wrap(bytes32(cfg.faultGameAbsolutePrestate()));
        PreimageOracle fastOracle = new PreimageOracle(cfg.preimageOracleMinProposalSize(), 0);
        _setFaultGameImplementation({
            _factory: factory,
            _allowUpgrade: _allowUpgrade,
            _params: FaultDisputeGameParams({
                anchorStateRegistry: AnchorStateRegistry(mustGetAddress("AnchorStateRegistryProxy")),
                weth: weth,
                gameType: GameTypes.FAST,
                absolutePrestate: outputAbsolutePrestate,
                faultVm: IBigStepper(new AlphabetVM(outputAbsolutePrestate, fastOracle)),
                // The max depth for the alphabet trace is always 3. Add 1 because split depth is fully inclusive.
                maxGameDepth: cfg.faultGameSplitDepth() + 3 + 1,
                maxClockDuration: Duration.wrap(0) // Resolvable immediately
             })
        });
    }

    /// @notice Sets the implementation for the given fault game type in the `DisputeGameFactory`.
    function _setFaultGameImplementation(
        DisputeGameFactory _factory,
        bool _allowUpgrade,
        FaultDisputeGameParams memory _params
    )
        internal
    {
        if (address(_factory.gameImpls(_params.gameType)) != address(0) && !_allowUpgrade) {
            console.log(
                "[WARN] DisputeGameFactoryProxy: `FaultDisputeGame` implementation already set for game type: %s",
                vm.toString(GameType.unwrap(_params.gameType))
            );
            return;
        }

        uint32 rawGameType = GameType.unwrap(_params.gameType);
        if (rawGameType != GameTypes.PERMISSIONED_CANNON.raw()) {
            _factory.setImplementation(
                _params.gameType,
                new FaultDisputeGame({
                    _gameType: _params.gameType,
                    _absolutePrestate: _params.absolutePrestate,
                    _maxGameDepth: _params.maxGameDepth,
                    _splitDepth: cfg.faultGameSplitDepth(),
                    _clockExtension: Duration.wrap(uint64(cfg.faultGameClockExtension())),
                    _maxClockDuration: _params.maxClockDuration,
                    _vm: _params.faultVm,
                    _weth: _params.weth,
                    _anchorStateRegistry: _params.anchorStateRegistry,
                    _l2ChainId: cfg.l2ChainID()
                })
            );
        } else {
            _factory.setImplementation(
                _params.gameType,
                new PermissionedDisputeGame({
                    _gameType: _params.gameType,
                    _absolutePrestate: _params.absolutePrestate,
                    _maxGameDepth: _params.maxGameDepth,
                    _splitDepth: cfg.faultGameSplitDepth(),
                    _clockExtension: Duration.wrap(uint64(cfg.faultGameClockExtension())),
                    _maxClockDuration: Duration.wrap(uint64(cfg.faultGameMaxClockDuration())),
                    _vm: _params.faultVm,
                    _weth: _params.weth,
                    _anchorStateRegistry: _params.anchorStateRegistry,
                    _l2ChainId: cfg.l2ChainID(),
                    _proposer: cfg.l2OutputOracleProposer(),
                    _challenger: cfg.l2OutputOracleChallenger()
                })
            );
        }

        string memory gameTypeString;
        if (rawGameType == GameTypes.CANNON.raw()) {
            gameTypeString = "Cannon";
        } else if (rawGameType == GameTypes.PERMISSIONED_CANNON.raw()) {
            gameTypeString = "PermissionedCannon";
        } else if (rawGameType == GameTypes.ALPHABET.raw()) {
            gameTypeString = "Alphabet";
        } else {
            gameTypeString = "Unknown";
        }

        console.log(
            "DisputeGameFactoryProxy: set `FaultDisputeGame` implementation (Backend: %s | GameType: %s)",
            gameTypeString,
            vm.toString(rawGameType)
        );
    }

    /// @notice Initialize the DataAvailabilityChallenge
    function initializeDataAvailabilityChallenge() public broadcast {
        console.log("Upgrading and initializing DataAvailabilityChallenge proxy");
        address dataAvailabilityChallengeProxy = mustGetAddress("DataAvailabilityChallengeProxy");
        address dataAvailabilityChallenge = mustGetAddress("DataAvailabilityChallenge");

        address finalSystemOwner = cfg.finalSystemOwner();
        uint256 daChallengeWindow = cfg.daChallengeWindow();
        uint256 daResolveWindow = cfg.daResolveWindow();
        uint256 daBondSize = cfg.daBondSize();
        uint256 daResolverRefundPercentage = cfg.daResolverRefundPercentage();

        _upgradeAndCallViaSafe({
            _proxy: payable(dataAvailabilityChallengeProxy),
            _implementation: dataAvailabilityChallenge,
            _innerCallData: abi.encodeCall(
                DataAvailabilityChallenge.initialize,
                (finalSystemOwner, daChallengeWindow, daResolveWindow, daBondSize, daResolverRefundPercentage)
            )
        });

        DataAvailabilityChallenge dac = DataAvailabilityChallenge(payable(dataAvailabilityChallengeProxy));
        string memory version = dac.version();
        console.log("DataAvailabilityChallenge version: %s", version);

        require(dac.owner() == finalSystemOwner);
        require(dac.challengeWindow() == daChallengeWindow);
        require(dac.resolveWindow() == daResolveWindow);
        require(dac.bondSize() == daBondSize);
        require(dac.resolverRefundPercentage() == daResolverRefundPercentage);
    }
}
