// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

// Testing
import { VmSafe } from "forge-std/Vm.sol";
import { Script } from "forge-std/Script.sol";
import { console2 as console } from "forge-std/console2.sol";
import { stdJson } from "forge-std/StdJson.sol";
import { AlphabetVM } from "test/mocks/AlphabetVM.sol";
import { EIP1967Helper } from "test/mocks/EIP1967Helper.sol";

// Scripts
import { Deployer } from "scripts/deploy/Deployer.sol";
import { Chains } from "scripts/libraries/Chains.sol";
import { Config } from "scripts/libraries/Config.sol";
import { LibStateDiff } from "scripts/libraries/LibStateDiff.sol";
import { Process } from "scripts/libraries/Process.sol";
import { ChainAssertions } from "scripts/deploy/ChainAssertions.sol";
import { DeployUtils } from "scripts/libraries/DeployUtils.sol";
import { DeploySuperchainInput, DeploySuperchain, DeploySuperchainOutput } from "scripts/DeploySuperchain.s.sol";
import {
    DeployImplementationsInput,
    DeployImplementations,
    DeployImplementationsInterop,
    DeployImplementationsOutput
} from "scripts/DeployImplementations.s.sol";

// Contracts
import { AddressManager } from "src/legacy/AddressManager.sol";
import { StorageSetter } from "src/universal/StorageSetter.sol";

// Libraries
import { Constants } from "src/libraries/Constants.sol";
import { Types } from "scripts/libraries/Types.sol";
import { Duration } from "src/dispute/lib/LibUDT.sol";
import "src/dispute/lib/Types.sol";

// Interfaces
import { IProxy } from "src/universal/interfaces/IProxy.sol";
import { IProxyAdmin } from "src/universal/interfaces/IProxyAdmin.sol";
import { IOptimismPortal } from "src/L1/interfaces/IOptimismPortal.sol";
import { IOptimismPortal2 } from "src/L1/interfaces/IOptimismPortal2.sol";
import { IOptimismPortalInterop } from "src/L1/interfaces/IOptimismPortalInterop.sol";
import { ICrossDomainMessenger } from "src/universal/interfaces/ICrossDomainMessenger.sol";
import { IL1CrossDomainMessenger } from "src/L1/interfaces/IL1CrossDomainMessenger.sol";
import { IL2OutputOracle } from "src/L1/interfaces/IL2OutputOracle.sol";
import { ISuperchainConfig } from "src/L1/interfaces/ISuperchainConfig.sol";
import { ISystemConfig } from "src/L1/interfaces/ISystemConfig.sol";
import { ISystemConfigInterop } from "src/L1/interfaces/ISystemConfigInterop.sol";
import { IDataAvailabilityChallenge } from "src/L1/interfaces/IDataAvailabilityChallenge.sol";
import { IL1ERC721Bridge } from "src/L1/interfaces/IL1ERC721Bridge.sol";
import { IL1StandardBridge } from "src/L1/interfaces/IL1StandardBridge.sol";
import { IProtocolVersions, ProtocolVersion } from "src/L1/interfaces/IProtocolVersions.sol";
import { IBigStepper } from "src/dispute/interfaces/IBigStepper.sol";
import { IDisputeGameFactory } from "src/dispute/interfaces/IDisputeGameFactory.sol";
import { IDisputeGame } from "src/dispute/interfaces/IDisputeGame.sol";
import { IFaultDisputeGame } from "src/dispute/interfaces/IFaultDisputeGame.sol";
import { IPermissionedDisputeGame } from "src/dispute/interfaces/IPermissionedDisputeGame.sol";
import { IDelayedWETH } from "src/dispute/interfaces/IDelayedWETH.sol";
import { IAnchorStateRegistry } from "src/dispute/interfaces/IAnchorStateRegistry.sol";
import { IMIPS } from "src/cannon/interfaces/IMIPS.sol";
import { IMIPS2 } from "src/cannon/interfaces/IMIPS2.sol";
import { IPreimageOracle } from "src/cannon/interfaces/IPreimageOracle.sol";
import { IOptimismMintableERC20Factory } from "src/universal/interfaces/IOptimismMintableERC20Factory.sol";
import { IAddressManager } from "src/legacy/interfaces/IAddressManager.sol";
import { IL1ChugSplashProxy } from "src/legacy/interfaces/IL1ChugSplashProxy.sol";
import { IResolvedDelegateProxy } from "src/legacy/interfaces/IResolvedDelegateProxy.sol";

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
        IAnchorStateRegistry anchorStateRegistry;
        IDelayedWETH weth;
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

    /// @notice Returns the proxy addresses, not reverting if any are unset.
    function _proxies() internal view returns (Types.ContractSet memory proxies_) {
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
            SuperchainConfig: getAddress("SuperchainConfigProxy"),
            OPContractsManager: getAddress("OPContractsManagerProxy")
        });
    }

    /// @notice Returns the impl addresses, not reverting if any are unset.
    function _impls() internal view returns (Types.ContractSet memory proxies_) {
        proxies_ = Types.ContractSet({
            L1CrossDomainMessenger: getAddress("L1CrossDomainMessenger"),
            L1StandardBridge: getAddress("L1StandardBridge"),
            L2OutputOracle: getAddress("L2OutputOracle"),
            DisputeGameFactory: getAddress("DisputeGameFactory"),
            DelayedWETH: getAddress("DelayedWETH"),
            PermissionedDelayedWETH: getAddress("PermissionedDelayedWETH"),
            AnchorStateRegistry: getAddress("AnchorStateRegistry"),
            OptimismMintableERC20Factory: getAddress("OptimismMintableERC20Factory"),
            OptimismPortal: getAddress("OptimismPortal"),
            OptimismPortal2: getAddress("OptimismPortal2"),
            SystemConfig: getAddress("SystemConfig"),
            L1ERC721Bridge: getAddress("L1ERC721Bridge"),
            ProtocolVersions: getAddress("ProtocolVersions"),
            SuperchainConfig: getAddress("SuperchainConfig"),
            OPContractsManager: getAddress("OPContractsManager")
        });
    }

    ////////////////////////////////////////////////////////////////
    //            State Changing Helper Functions                 //
    ////////////////////////////////////////////////////////////////

    /// @notice Transfer ownership of the ProxyAdmin contract to the final system owner
    function transferProxyAdminOwnership() public broadcast {
        // Get the ProxyAdmin contract.
        IProxyAdmin proxyAdmin = IProxyAdmin(mustGetAddress("ProxyAdmin"));

        // Transfer ownership to the final system owner if necessary.
        address owner = proxyAdmin.owner();
        address finalSystemOwner = cfg.finalSystemOwner();
        if (owner != finalSystemOwner) {
            proxyAdmin.transferOwnership(finalSystemOwner);
            console.log("ProxyAdmin ownership transferred to final system owner at: %s", finalSystemOwner);
        }

        // Make sure the ProxyAdmin owner is set to the final system owner.
        owner = proxyAdmin.owner();
        require(owner == finalSystemOwner, "Deploy: ProxyAdmin ownership not transferred to final system owner");
    }

    ////////////////////////////////////////////////////////////////
    //                    SetUp and Run                           //
    ////////////////////////////////////////////////////////////////

    /// @notice Deploy all of the L1 contracts necessary for a full Superchain with a single Op Chain.
    function run() public {
        console.log("Deploying a fresh OP Stack including SuperchainConfig");
        _run();
    }

    /// @notice Deploy a new OP Chain using an existing SuperchainConfig and ProtocolVersions
    /// @param _superchainConfigProxy Address of the existing SuperchainConfig proxy
    /// @param _protocolVersionsProxy Address of the existing ProtocolVersions proxy
    /// @param _includeDump Whether to include a state dump after deployment
    function runWithSuperchain(
        address payable _superchainConfigProxy,
        address payable _protocolVersionsProxy,
        bool _includeDump
    )
        public
    {
        require(_superchainConfigProxy != address(0), "Deploy: must specify address for superchain config proxy");
        require(_protocolVersionsProxy != address(0), "Deploy: must specify address for protocol versions proxy");

        vm.chainId(cfg.l1ChainID());

        console.log("Deploying a fresh OP Stack with existing SuperchainConfig and ProtocolVersions");

        IProxy scProxy = IProxy(_superchainConfigProxy);
        save("SuperchainConfig", scProxy.implementation());
        save("SuperchainConfigProxy", _superchainConfigProxy);

        IProxy pvProxy = IProxy(_protocolVersionsProxy);
        save("ProtocolVersions", pvProxy.implementation());
        save("ProtocolVersionsProxy", _protocolVersionsProxy);

        _run(false);

        if (_includeDump) {
            vm.dumpState(Config.stateDumpPath(""));
        }
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

    /// @notice Compatibility function for tests that override _run().
    function _run() internal virtual {
        _run(true);
    }

    /// @notice Internal function containing the deploy logic.
    function _run(bool _needsSuperchain) internal {
        console.log("start of L1 Deploy!");

        // Set up the Superchain if needed.
        if (_needsSuperchain) {
            deploySuperchain();
        }

        deployImplementations({ _isInterop: cfg.useInterop() });

        // Deploy Current OPChain Contracts
        deployOpChain();

        if (cfg.useAltDA()) {
            bytes32 typeHash = keccak256(bytes(cfg.daCommitmentType()));
            bytes32 keccakHash = keccak256(bytes("KeccakCommitment"));
            if (typeHash == keccakHash) {
                setupOpAltDA();
            }
        }

        transferProxyAdminOwnership();
        console.log("set up op chain!");
    }

    ////////////////////////////////////////////////////////////////
    //           High Level Deployment Functions                  //
    ////////////////////////////////////////////////////////////////

    /// @notice Deploy a full system with a new SuperchainConfig
    ///         The Superchain system has 2 singleton contracts which lie outside of an OP Chain:
    ///         1. The SuperchainConfig contract
    ///         2. The ProtocolVersions contract
    function deploySuperchain() public {
        console.log("Setting up Superchain");
        DeploySuperchain ds = new DeploySuperchain();
        (DeploySuperchainInput dsi, DeploySuperchainOutput dso) = ds.etchIOContracts();

        // Set the input values on the input contract.
        // TODO: when DeployAuthSystem is done, finalSystemOwner should be replaced with the Foundation Upgrades Safe
        dsi.set(dsi.protocolVersionsOwner.selector, cfg.finalSystemOwner());
        dsi.set(dsi.superchainProxyAdminOwner.selector, cfg.finalSystemOwner());
        dsi.set(dsi.guardian.selector, cfg.superchainConfigGuardian());
        dsi.set(dsi.paused.selector, false);
        dsi.set(dsi.requiredProtocolVersion.selector, ProtocolVersion.wrap(cfg.requiredProtocolVersion()));
        dsi.set(dsi.recommendedProtocolVersion.selector, ProtocolVersion.wrap(cfg.recommendedProtocolVersion()));

        // Run the deployment script.
        ds.run(dsi, dso);
        save("SuperchainProxyAdmin", address(dso.superchainProxyAdmin()));
        save("SuperchainConfigProxy", address(dso.superchainConfigProxy()));
        save("SuperchainConfig", address(dso.superchainConfigImpl()));
        save("ProtocolVersionsProxy", address(dso.protocolVersionsProxy()));
        save("ProtocolVersions", address(dso.protocolVersionsImpl()));

        // First run assertions for the ProtocolVersions and SuperchainConfig proxy contracts.
        Types.ContractSet memory contracts = _proxies();
        ChainAssertions.checkProtocolVersions({ _contracts: contracts, _cfg: cfg, _isProxy: true });
        ChainAssertions.checkSuperchainConfig({ _contracts: contracts, _cfg: cfg, _isProxy: true, _isPaused: false });

        // Then replace the ProtocolVersions proxy with the implementation address and run assertions on it.
        contracts.ProtocolVersions = mustGetAddress("ProtocolVersions");
        ChainAssertions.checkProtocolVersions({ _contracts: contracts, _cfg: cfg, _isProxy: false });

        // Finally replace the SuperchainConfig proxy with the implementation address and run assertions on it.
        contracts.SuperchainConfig = mustGetAddress("SuperchainConfig");
        ChainAssertions.checkSuperchainConfig({ _contracts: contracts, _cfg: cfg, _isPaused: false, _isProxy: false });
    }

    /// @notice Deploy all of the implementations
    function deployImplementations(bool _isInterop) public {
        require(_isInterop == cfg.useInterop(), "Deploy: Interop setting mismatch.");

        console.log("Deploying implementations");
        DeployImplementations di = new DeployImplementations();
        (DeployImplementationsInput dii, DeployImplementationsOutput dio) = di.etchIOContracts();

        dii.set(dii.withdrawalDelaySeconds.selector, cfg.faultGameWithdrawalDelay());
        dii.set(dii.minProposalSizeBytes.selector, cfg.preimageOracleMinProposalSize());
        dii.set(dii.challengePeriodSeconds.selector, cfg.preimageOracleChallengePeriod());
        dii.set(dii.proofMaturityDelaySeconds.selector, cfg.proofMaturityDelaySeconds());
        dii.set(dii.disputeGameFinalityDelaySeconds.selector, cfg.disputeGameFinalityDelaySeconds());
        string memory release = "dev";
        dii.set(dii.release.selector, release);
        dii.set(
            dii.standardVersionsToml.selector, string.concat(vm.projectRoot(), "/test/fixtures/standard-versions.toml")
        );
        dii.set(dii.superchainConfigProxy.selector, mustGetAddress("SuperchainConfigProxy"));
        dii.set(dii.protocolVersionsProxy.selector, mustGetAddress("ProtocolVersionsProxy"));
        dii.set(dii.opcmProxyOwner.selector, cfg.finalSystemOwner());

        if (_isInterop) {
            di = DeployImplementations(new DeployImplementationsInterop());
        }
        di.run(dii, dio);

        // Temporary patch for legacy system
        if (!cfg.useFaultProofs()) {
            deployOptimismPortal();
            deployL2OutputOracle();
        }

        save("L1CrossDomainMessenger", address(dio.l1CrossDomainMessengerImpl()));
        save("OptimismMintableERC20Factory", address(dio.optimismMintableERC20FactoryImpl()));
        save("SystemConfig", address(dio.systemConfigImpl()));
        save("L1StandardBridge", address(dio.l1StandardBridgeImpl()));
        save("L1ERC721Bridge", address(dio.l1ERC721BridgeImpl()));

        // Fault proofs
        save("OptimismPortal2", address(dio.optimismPortalImpl()));
        save("DisputeGameFactory", address(dio.disputeGameFactoryImpl()));
        save("DelayedWETH", address(dio.delayedWETHImpl()));
        save("PreimageOracle", address(dio.preimageOracleSingleton()));
        save("Mips", address(dio.mipsSingleton()));
        save("OPContractsManagerProxy", address(dio.opcmProxy()));
        save("OPContractsManager", address(dio.opcmImpl()));

        Types.ContractSet memory contracts = _impls();
        ChainAssertions.checkL1CrossDomainMessenger({ _contracts: contracts, _vm: vm, _isProxy: false });
        ChainAssertions.checkL1StandardBridge({ _contracts: contracts, _isProxy: false });
        ChainAssertions.checkL1ERC721Bridge({ _contracts: contracts, _isProxy: false });
        ChainAssertions.checkOptimismPortal2({ _contracts: contracts, _cfg: cfg, _isProxy: false });
        ChainAssertions.checkOptimismMintableERC20Factory({ _contracts: contracts, _isProxy: false });
        ChainAssertions.checkDisputeGameFactory({ _contracts: contracts, _expectedOwner: address(0), _isProxy: false });
        ChainAssertions.checkDelayedWETH({
            _contracts: contracts,
            _cfg: cfg,
            _isProxy: false,
            _expectedOwner: address(0)
        });
        ChainAssertions.checkPreimageOracle({
            _oracle: IPreimageOracle(address(dio.preimageOracleSingleton())),
            _cfg: cfg
        });
        ChainAssertions.checkMIPS({
            _mips: IMIPS(address(dio.mipsSingleton())),
            _oracle: IPreimageOracle(address(dio.preimageOracleSingleton()))
        });
        if (_isInterop) {
            ChainAssertions.checkSystemConfigInterop({ _contracts: contracts, _cfg: cfg, _isProxy: false });
        } else {
            ChainAssertions.checkSystemConfig({ _contracts: contracts, _cfg: cfg, _isProxy: false });
        }
    }

    /// @notice Deploy all of the OP Chain specific contracts
    function deployOpChain() public {
        console.log("Deploying OP Chain");
        deployAddressManager();
        deployProxyAdmin();
        transferAddressManagerOwnership(); // to the ProxyAdmin

        // Ensure that the requisite contracts are deployed
        mustGetAddress("SuperchainConfigProxy");
        mustGetAddress("AddressManager");
        mustGetAddress("ProxyAdmin");

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
        deployERC1967Proxy("DelayedWETHProxy");
        deployERC1967Proxy("PermissionedDelayedWETHProxy");
        deployERC1967Proxy("AnchorStateRegistryProxy");

        deployAnchorStateRegistry();

        // Deploy and setup the legacy (pre-faultproofs) contracts
        if (!cfg.useFaultProofs()) {
            deployERC1967Proxy("L2OutputOracleProxy");
        }

        initializeOpChain();

        setAlphabetFaultGameImplementation({ _allowUpgrade: false });
        setFastFaultGameImplementation({ _allowUpgrade: false });
        setCannonFaultGameImplementation({ _allowUpgrade: false });
        setPermissionedCannonFaultGameImplementation({ _allowUpgrade: false });

        transferDisputeGameFactoryOwnership();
        transferDelayedWETHOwnership();
    }

    /// @notice Initialize all of the proxies in an OP Chain by upgrading to the correct proxy and calling the
    /// initialize function
    function initializeOpChain() public {
        console.log("Initializing Op Chain proxies");

        initializeOptimismPortal();
        initializeSystemConfig();
        initializeL1StandardBridge();
        initializeL1ERC721Bridge();
        initializeOptimismMintableERC20Factory();
        initializeL1CrossDomainMessenger();
        initializeDisputeGameFactory();
        initializeDelayedWETH();
        initializePermissionedDelayedWETH();
        initializeAnchorStateRegistry();

        if (!cfg.useFaultProofs()) {
            initializeL2OutputOracle();
        }
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

    /// @notice Deploy the AddressManager
    function deployAddressManager() public broadcast returns (address addr_) {
        console.log("Deploying AddressManager");
        AddressManager manager = new AddressManager();
        require(manager.owner() == msg.sender);

        save("AddressManager", address(manager));
        console.log("AddressManager deployed at %s", address(manager));
        addr_ = address(manager);
    }

    /// @notice Deploys the ProxyAdmin contract. Should NOT be used for the Superchain.
    function deployProxyAdmin() public broadcast returns (address addr_) {
        // Deploy the ProxyAdmin contract.
        IProxyAdmin admin = IProxyAdmin(
            DeployUtils.create2AndSave({
                _save: this,
                _salt: _implSalt(),
                _name: "ProxyAdmin",
                _args: DeployUtils.encodeConstructor(abi.encodeCall(IProxyAdmin.__constructor__, (msg.sender)))
            })
        );

        // Make sure the owner was set to the deployer.
        require(admin.owner() == msg.sender);

        // Set the address manager if it is not already set.
        IAddressManager addressManager = IAddressManager(mustGetAddress("AddressManager"));
        if (admin.addressManager() != addressManager) {
            admin.setAddressManager(addressManager);
        }

        // Make sure the address manager is set properly.
        require(admin.addressManager() == addressManager);

        // Return the address of the deployed contract.
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
        address proxyAdmin = mustGetAddress("ProxyAdmin");
        IL1ChugSplashProxy proxy = IL1ChugSplashProxy(
            DeployUtils.create2AndSave({
                _save: this,
                _salt: _implSalt(),
                _name: "L1ChugSplashProxy",
                _nick: "L1StandardBridgeProxy",
                _args: DeployUtils.encodeConstructor(abi.encodeCall(IL1ChugSplashProxy.__constructor__, (proxyAdmin)))
            })
        );
        require(EIP1967Helper.getAdmin(address(proxy)) == proxyAdmin);
        addr_ = address(proxy);
    }

    /// @notice Deploy the L1CrossDomainMessengerProxy using a ResolvedDelegateProxy
    function deployL1CrossDomainMessengerProxy() public broadcast returns (address addr_) {
        IResolvedDelegateProxy proxy = IResolvedDelegateProxy(
            DeployUtils.create2AndSave({
                _save: this,
                _salt: _implSalt(),
                _name: "ResolvedDelegateProxy",
                _nick: "L1CrossDomainMessengerProxy",
                _args: DeployUtils.encodeConstructor(
                    abi.encodeCall(
                        IResolvedDelegateProxy.__constructor__,
                        (IAddressManager(mustGetAddress("AddressManager")), "OVM_L1CrossDomainMessenger")
                    )
                )
            })
        );
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
        IProxy proxy = IProxy(
            DeployUtils.create2AndSave({
                _save: this,
                _salt: keccak256(abi.encode(_implSalt(), _name)),
                _name: "Proxy",
                _nick: _name,
                _args: DeployUtils.encodeConstructor(abi.encodeCall(IProxy.__constructor__, (_proxyOwner)))
            })
        );
        require(EIP1967Helper.getAdmin(address(proxy)) == _proxyOwner);
        addr_ = address(proxy);
    }

    /// @notice Deploy the DataAvailabilityChallengeProxy
    function deployDataAvailabilityChallengeProxy() public broadcast returns (address addr_) {
        address proxyAdmin = mustGetAddress("ProxyAdmin");
        IProxy proxy = IProxy(
            DeployUtils.create2AndSave({
                _save: this,
                _salt: _implSalt(),
                _name: "Proxy",
                _nick: "DataAvailabilityChallengeProxy",
                _args: DeployUtils.encodeConstructor(abi.encodeCall(IProxy.__constructor__, (proxyAdmin)))
            })
        );
        require(EIP1967Helper.getAdmin(address(proxy)) == proxyAdmin);
        addr_ = address(proxy);
    }

    ////////////////////////////////////////////////////////////////
    //             Implementation Deployment Functions            //
    ////////////////////////////////////////////////////////////////

    /// @notice Deploy the OptimismPortal
    function deployOptimismPortal() public broadcast returns (address addr_) {
        if (cfg.useFaultProofs()) {
            // Could also verify this inside DeployConfig but doing it here is a bit more reliable.
            require(
                uint32(cfg.respectedGameType()) == cfg.respectedGameType(),
                "Deploy: respectedGameType must fit into uint32"
            );

            addr_ = DeployUtils.create2AndSave({
                _save: this,
                _salt: _implSalt(),
                _name: "OptimismPortal2",
                _args: DeployUtils.encodeConstructor(
                    abi.encodeCall(
                        IOptimismPortal2.__constructor__,
                        (cfg.proofMaturityDelaySeconds(), cfg.disputeGameFinalityDelaySeconds())
                    )
                )
            });

            // Override the `OptimismPortal2` contract to the deployed implementation. This is necessary
            // to check the `OptimismPortal2` implementation alongside dependent contracts, which
            // are always proxies.
            Types.ContractSet memory contracts = _proxies();
            contracts.OptimismPortal2 = addr_;
            ChainAssertions.checkOptimismPortal2({ _contracts: contracts, _cfg: cfg, _isProxy: false });
        } else {
            if (cfg.useInterop()) {
                console.log("Attempting to deploy OptimismPortal with interop, this config is a noop");
            }

            addr_ = DeployUtils.create2AndSave({
                _save: this,
                _salt: _implSalt(),
                _name: "OptimismPortal",
                _args: DeployUtils.encodeConstructor(abi.encodeCall(IOptimismPortal.__constructor__, ()))
            });

            // Override the `OptimismPortal` contract to the deployed implementation. This is necessary
            // to check the `OptimismPortal` implementation alongside dependent contracts, which
            // are always proxies.
            Types.ContractSet memory contracts = _proxies();
            contracts.OptimismPortal = addr_;
            ChainAssertions.checkOptimismPortal({ _contracts: contracts, _cfg: cfg, _isProxy: false });
        }
    }

    /// @notice Deploy the L2OutputOracle
    function deployL2OutputOracle() public broadcast returns (address addr_) {
        IL2OutputOracle oracle = IL2OutputOracle(
            DeployUtils.create2AndSave({
                _save: this,
                _salt: _implSalt(),
                _name: "L2OutputOracle",
                _args: DeployUtils.encodeConstructor(abi.encodeCall(IL2OutputOracle.__constructor__, ()))
            })
        );

        // Override the `L2OutputOracle` contract to the deployed implementation. This is necessary
        // to check the `L2OutputOracle` implementation alongside dependent contracts, which
        // are always proxies.
        Types.ContractSet memory contracts = _proxies();
        contracts.L2OutputOracle = address(oracle);
        ChainAssertions.checkL2OutputOracle({
            _contracts: contracts,
            _cfg: cfg,
            _l2OutputOracleStartingTimestamp: 0,
            _isProxy: false
        });

        addr_ = address(oracle);
    }

    /// @notice Deploy Mips VM. Deploys either MIPS or MIPS2 depending on the environment
    function deployMips() public broadcast returns (address addr_) {
        addr_ = DeployUtils.create2AndSave({
            _save: this,
            _salt: _implSalt(),
            _name: Config.useMultithreadedCannon() ? "MIPS2" : "MIPS",
            _args: DeployUtils.encodeConstructor(
                abi.encodeCall(IMIPS2.__constructor__, (IPreimageOracle(mustGetAddress("PreimageOracle"))))
            )
        });
        save("Mips", address(addr_));
    }

    /// @notice Deploy the AnchorStateRegistry
    function deployAnchorStateRegistry() public broadcast returns (address addr_) {
        IAnchorStateRegistry anchorStateRegistry = IAnchorStateRegistry(
            DeployUtils.create2AndSave({
                _save: this,
                _salt: _implSalt(),
                _name: "AnchorStateRegistry",
                _args: DeployUtils.encodeConstructor(
                    abi.encodeCall(
                        IAnchorStateRegistry.__constructor__,
                        (IDisputeGameFactory(mustGetAddress("DisputeGameFactoryProxy")))
                    )
                )
            })
        );

        addr_ = address(anchorStateRegistry);
    }

    /// @notice Deploy the L1StandardBridge
    function deployL1StandardBridge() public broadcast returns (address addr_) {
        IL1StandardBridge bridge = IL1StandardBridge(
            DeployUtils.create2AndSave({
                _save: this,
                _salt: _implSalt(),
                _name: "L1StandardBridge",
                _args: DeployUtils.encodeConstructor(abi.encodeCall(IL1StandardBridge.__constructor__, ()))
            })
        );

        // Override the `L1StandardBridge` contract to the deployed implementation. This is necessary
        // to check the `L1StandardBridge` implementation alongside dependent contracts, which
        // are always proxies.
        Types.ContractSet memory contracts = _proxies();
        contracts.L1StandardBridge = address(bridge);
        ChainAssertions.checkL1StandardBridge({ _contracts: contracts, _isProxy: false });

        addr_ = address(bridge);
    }

    /// @notice Deploy the L1ERC721Bridge
    function deployL1ERC721Bridge() public broadcast returns (address addr_) {
        IL1ERC721Bridge bridge = IL1ERC721Bridge(
            DeployUtils.create2AndSave({
                _save: this,
                _salt: _implSalt(),
                _name: "L1ERC721Bridge",
                _args: DeployUtils.encodeConstructor(abi.encodeCall(IL1ERC721Bridge.__constructor__, ()))
            })
        );

        // Override the `L1ERC721Bridge` contract to the deployed implementation. This is necessary
        // to check the `L1ERC721Bridge` implementation alongside dependent contracts, which
        // are always proxies.
        Types.ContractSet memory contracts = _proxies();
        contracts.L1ERC721Bridge = address(bridge);

        ChainAssertions.checkL1ERC721Bridge({ _contracts: contracts, _isProxy: false });

        addr_ = address(bridge);
    }

    /// @notice Transfer ownership of the address manager to the ProxyAdmin
    function transferAddressManagerOwnership() public broadcast {
        console.log("Transferring AddressManager ownership to IProxyAdmin");
        IAddressManager addressManager = IAddressManager(mustGetAddress("AddressManager"));
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
        IDataAvailabilityChallenge dac = IDataAvailabilityChallenge(
            DeployUtils.create2AndSave({
                _save: this,
                _salt: _implSalt(),
                _name: "DataAvailabilityChallenge",
                _args: DeployUtils.encodeConstructor(abi.encodeCall(IDataAvailabilityChallenge.__constructor__, ()))
            })
        );
        addr_ = address(dac);
    }

    ////////////////////////////////////////////////////////////////
    //                    Initialize Functions                    //
    ////////////////////////////////////////////////////////////////

    /// @notice Initialize the DisputeGameFactory
    function initializeDisputeGameFactory() public broadcast {
        console.log("Upgrading and initializing DisputeGameFactory proxy");
        address disputeGameFactoryProxy = mustGetAddress("DisputeGameFactoryProxy");
        address disputeGameFactory = mustGetAddress("DisputeGameFactory");

        IProxyAdmin proxyAdmin = IProxyAdmin(payable(mustGetAddress("ProxyAdmin")));
        proxyAdmin.upgradeAndCall({
            _proxy: payable(disputeGameFactoryProxy),
            _implementation: disputeGameFactory,
            _data: abi.encodeCall(IDisputeGameFactory.initialize, (msg.sender))
        });

        string memory version = IDisputeGameFactory(disputeGameFactoryProxy).version();
        console.log("DisputeGameFactory version: %s", version);

        ChainAssertions.checkDisputeGameFactory({ _contracts: _proxies(), _expectedOwner: msg.sender, _isProxy: true });
    }

    function initializeDelayedWETH() public broadcast {
        console.log("Upgrading and initializing DelayedWETH proxy");
        address delayedWETHProxy = mustGetAddress("DelayedWETHProxy");
        address delayedWETH = mustGetAddress("DelayedWETH");
        address superchainConfigProxy = mustGetAddress("SuperchainConfigProxy");

        IProxyAdmin proxyAdmin = IProxyAdmin(payable(mustGetAddress("ProxyAdmin")));
        proxyAdmin.upgradeAndCall({
            _proxy: payable(delayedWETHProxy),
            _implementation: delayedWETH,
            _data: abi.encodeCall(IDelayedWETH.initialize, (msg.sender, ISuperchainConfig(superchainConfigProxy)))
        });

        string memory version = IDelayedWETH(payable(delayedWETHProxy)).version();
        console.log("DelayedWETH version: %s", version);

        ChainAssertions.checkDelayedWETH({
            _contracts: _proxies(),
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

        IProxyAdmin proxyAdmin = IProxyAdmin(payable(mustGetAddress("ProxyAdmin")));
        proxyAdmin.upgradeAndCall({
            _proxy: payable(delayedWETHProxy),
            _implementation: delayedWETH,
            _data: abi.encodeCall(IDelayedWETH.initialize, (msg.sender, ISuperchainConfig(superchainConfigProxy)))
        });

        string memory version = IDelayedWETH(payable(delayedWETHProxy)).version();
        console.log("DelayedWETH version: %s", version);

        ChainAssertions.checkPermissionedDelayedWETH({
            _contracts: _proxies(),
            _cfg: cfg,
            _isProxy: true,
            _expectedOwner: msg.sender
        });
    }

    function initializeAnchorStateRegistry() public broadcast {
        console.log("Upgrading and initializing AnchorStateRegistry proxy");
        address anchorStateRegistryProxy = mustGetAddress("AnchorStateRegistryProxy");
        address anchorStateRegistry = mustGetAddress("AnchorStateRegistry");
        ISuperchainConfig superchainConfig = ISuperchainConfig(mustGetAddress("SuperchainConfigProxy"));

        IAnchorStateRegistry.StartingAnchorRoot[] memory roots = new IAnchorStateRegistry.StartingAnchorRoot[](5);
        roots[0] = IAnchorStateRegistry.StartingAnchorRoot({
            gameType: GameTypes.CANNON,
            outputRoot: OutputRoot({
                root: Hash.wrap(cfg.faultGameGenesisOutputRoot()),
                l2BlockNumber: cfg.faultGameGenesisBlock()
            })
        });
        roots[1] = IAnchorStateRegistry.StartingAnchorRoot({
            gameType: GameTypes.PERMISSIONED_CANNON,
            outputRoot: OutputRoot({
                root: Hash.wrap(cfg.faultGameGenesisOutputRoot()),
                l2BlockNumber: cfg.faultGameGenesisBlock()
            })
        });
        roots[2] = IAnchorStateRegistry.StartingAnchorRoot({
            gameType: GameTypes.ALPHABET,
            outputRoot: OutputRoot({
                root: Hash.wrap(cfg.faultGameGenesisOutputRoot()),
                l2BlockNumber: cfg.faultGameGenesisBlock()
            })
        });
        roots[3] = IAnchorStateRegistry.StartingAnchorRoot({
            gameType: GameTypes.ASTERISC,
            outputRoot: OutputRoot({
                root: Hash.wrap(cfg.faultGameGenesisOutputRoot()),
                l2BlockNumber: cfg.faultGameGenesisBlock()
            })
        });
        roots[4] = IAnchorStateRegistry.StartingAnchorRoot({
            gameType: GameTypes.FAST,
            outputRoot: OutputRoot({
                root: Hash.wrap(cfg.faultGameGenesisOutputRoot()),
                l2BlockNumber: cfg.faultGameGenesisBlock()
            })
        });

        IProxyAdmin proxyAdmin = IProxyAdmin(payable(mustGetAddress("ProxyAdmin")));
        proxyAdmin.upgradeAndCall({
            _proxy: payable(anchorStateRegistryProxy),
            _implementation: anchorStateRegistry,
            _data: abi.encodeCall(IAnchorStateRegistry.initialize, (roots, superchainConfig))
        });

        string memory version = IAnchorStateRegistry(payable(anchorStateRegistryProxy)).version();
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

        IProxyAdmin proxyAdmin = IProxyAdmin(payable(mustGetAddress("ProxyAdmin")));
        proxyAdmin.upgradeAndCall({
            _proxy: payable(systemConfigProxy),
            _implementation: systemConfig,
            _data: abi.encodeCall(
                ISystemConfig.initialize,
                (
                    cfg.finalSystemOwner(),
                    cfg.basefeeScalar(),
                    cfg.blobbasefeeScalar(),
                    batcherHash,
                    uint64(cfg.l2GenesisBlockGasLimit()),
                    cfg.p2pSequencerAddress(),
                    Constants.DEFAULT_RESOURCE_CONFIG(),
                    cfg.batchInboxAddress(),
                    ISystemConfig.Addresses({
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

        ISystemConfig config = ISystemConfig(systemConfigProxy);
        string memory version = config.version();
        console.log("SystemConfig version: %s", version);

        ChainAssertions.checkSystemConfig({ _contracts: _proxies(), _cfg: cfg, _isProxy: true });
    }

    /// @notice Initialize the L1StandardBridge
    function initializeL1StandardBridge() public broadcast {
        console.log("Upgrading and initializing L1StandardBridge proxy");
        IProxyAdmin proxyAdmin = IProxyAdmin(mustGetAddress("ProxyAdmin"));
        address l1StandardBridgeProxy = mustGetAddress("L1StandardBridgeProxy");
        address l1StandardBridge = mustGetAddress("L1StandardBridge");
        address l1CrossDomainMessengerProxy = mustGetAddress("L1CrossDomainMessengerProxy");
        address superchainConfigProxy = mustGetAddress("SuperchainConfigProxy");
        address systemConfigProxy = mustGetAddress("SystemConfigProxy");

        uint256 proxyType = uint256(proxyAdmin.proxyType(l1StandardBridgeProxy));
        if (proxyType != uint256(IProxyAdmin.ProxyType.CHUGSPLASH)) {
            proxyAdmin.setProxyType(l1StandardBridgeProxy, IProxyAdmin.ProxyType.CHUGSPLASH);
        }
        require(uint256(proxyAdmin.proxyType(l1StandardBridgeProxy)) == uint256(IProxyAdmin.ProxyType.CHUGSPLASH));

        proxyAdmin.upgradeAndCall({
            _proxy: payable(l1StandardBridgeProxy),
            _implementation: l1StandardBridge,
            _data: abi.encodeCall(
                IL1StandardBridge.initialize,
                (
                    ICrossDomainMessenger(l1CrossDomainMessengerProxy),
                    ISuperchainConfig(superchainConfigProxy),
                    ISystemConfig(systemConfigProxy)
                )
            )
        });

        string memory version = IL1StandardBridge(payable(l1StandardBridgeProxy)).version();
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

        IProxyAdmin proxyAdmin = IProxyAdmin(payable(mustGetAddress("ProxyAdmin")));
        proxyAdmin.upgradeAndCall({
            _proxy: payable(l1ERC721BridgeProxy),
            _implementation: l1ERC721Bridge,
            _data: abi.encodeCall(
                IL1ERC721Bridge.initialize,
                (ICrossDomainMessenger(payable(l1CrossDomainMessengerProxy)), ISuperchainConfig(superchainConfigProxy))
            )
        });

        IL1ERC721Bridge bridge = IL1ERC721Bridge(l1ERC721BridgeProxy);
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

        IProxyAdmin proxyAdmin = IProxyAdmin(payable(mustGetAddress("ProxyAdmin")));
        proxyAdmin.upgradeAndCall({
            _proxy: payable(optimismMintableERC20FactoryProxy),
            _implementation: optimismMintableERC20Factory,
            _data: abi.encodeCall(IOptimismMintableERC20Factory.initialize, (l1StandardBridgeProxy))
        });

        IOptimismMintableERC20Factory factory = IOptimismMintableERC20Factory(optimismMintableERC20FactoryProxy);
        string memory version = factory.version();
        console.log("OptimismMintableERC20Factory version: %s", version);

        ChainAssertions.checkOptimismMintableERC20Factory({ _contracts: _proxies(), _isProxy: true });
    }

    /// @notice initializeL1CrossDomainMessenger
    function initializeL1CrossDomainMessenger() public broadcast {
        console.log("Upgrading and initializing L1CrossDomainMessenger proxy");
        IProxyAdmin proxyAdmin = IProxyAdmin(mustGetAddress("ProxyAdmin"));
        address l1CrossDomainMessengerProxy = mustGetAddress("L1CrossDomainMessengerProxy");
        address l1CrossDomainMessenger = mustGetAddress("L1CrossDomainMessenger");
        address superchainConfigProxy = mustGetAddress("SuperchainConfigProxy");
        address optimismPortalProxy = mustGetAddress("OptimismPortalProxy");
        address systemConfigProxy = mustGetAddress("SystemConfigProxy");

        uint256 proxyType = uint256(proxyAdmin.proxyType(l1CrossDomainMessengerProxy));
        if (proxyType != uint256(IProxyAdmin.ProxyType.RESOLVED)) {
            proxyAdmin.setProxyType(l1CrossDomainMessengerProxy, IProxyAdmin.ProxyType.RESOLVED);
        }
        require(uint256(proxyAdmin.proxyType(l1CrossDomainMessengerProxy)) == uint256(IProxyAdmin.ProxyType.RESOLVED));

        string memory contractName = "OVM_L1CrossDomainMessenger";
        string memory implName = proxyAdmin.implementationName(l1CrossDomainMessenger);
        if (keccak256(bytes(contractName)) != keccak256(bytes(implName))) {
            proxyAdmin.setImplementationName(l1CrossDomainMessengerProxy, contractName);
        }
        require(
            keccak256(bytes(proxyAdmin.implementationName(l1CrossDomainMessengerProxy)))
                == keccak256(bytes(contractName))
        );

        proxyAdmin.upgradeAndCall({
            _proxy: payable(l1CrossDomainMessengerProxy),
            _implementation: l1CrossDomainMessenger,
            _data: abi.encodeCall(
                IL1CrossDomainMessenger.initialize,
                (
                    ISuperchainConfig(superchainConfigProxy),
                    IOptimismPortal(payable(optimismPortalProxy)),
                    ISystemConfig(systemConfigProxy)
                )
            )
        });

        IL1CrossDomainMessenger messenger = IL1CrossDomainMessenger(l1CrossDomainMessengerProxy);
        string memory version = messenger.version();
        console.log("L1CrossDomainMessenger version: %s", version);

        ChainAssertions.checkL1CrossDomainMessenger({ _contracts: _proxies(), _vm: vm, _isProxy: true });
    }

    /// @notice Initialize the L2OutputOracle
    function initializeL2OutputOracle() public broadcast {
        console.log("Upgrading and initializing L2OutputOracle proxy");
        address l2OutputOracleProxy = mustGetAddress("L2OutputOracleProxy");
        address l2OutputOracle = mustGetAddress("L2OutputOracle");

        IProxyAdmin proxyAdmin = IProxyAdmin(payable(mustGetAddress("ProxyAdmin")));
        proxyAdmin.upgradeAndCall({
            _proxy: payable(l2OutputOracleProxy),
            _implementation: l2OutputOracle,
            _data: abi.encodeCall(
                IL2OutputOracle.initialize,
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

        IL2OutputOracle oracle = IL2OutputOracle(l2OutputOracleProxy);
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
        address optimismPortalProxy = mustGetAddress("OptimismPortalProxy");
        address systemConfigProxy = mustGetAddress("SystemConfigProxy");
        address superchainConfigProxy = mustGetAddress("SuperchainConfigProxy");
        if (cfg.useFaultProofs()) {
            console.log("Upgrading and initializing OptimismPortal2 proxy");
            address optimismPortal2 = mustGetAddress("OptimismPortal2");
            address disputeGameFactoryProxy = mustGetAddress("DisputeGameFactoryProxy");

            IProxyAdmin proxyAdmin = IProxyAdmin(payable(mustGetAddress("ProxyAdmin")));
            proxyAdmin.upgradeAndCall({
                _proxy: payable(optimismPortalProxy),
                _implementation: optimismPortal2,
                _data: abi.encodeCall(
                    IOptimismPortal2.initialize,
                    (
                        IDisputeGameFactory(disputeGameFactoryProxy),
                        ISystemConfig(systemConfigProxy),
                        ISuperchainConfig(superchainConfigProxy),
                        GameType.wrap(uint32(cfg.respectedGameType()))
                    )
                )
            });

            IOptimismPortal2 portal = IOptimismPortal2(payable(optimismPortalProxy));
            string memory version = portal.version();
            console.log("OptimismPortal2 version: %s", version);

            ChainAssertions.checkOptimismPortal2({ _contracts: _proxies(), _cfg: cfg, _isProxy: true });
        } else {
            console.log("Upgrading and initializing OptimismPortal proxy");
            address optimismPortal = mustGetAddress("OptimismPortal");
            address l2OutputOracleProxy = mustGetAddress("L2OutputOracleProxy");

            IProxyAdmin proxyAdmin = IProxyAdmin(payable(mustGetAddress("ProxyAdmin")));
            proxyAdmin.upgradeAndCall({
                _proxy: payable(optimismPortalProxy),
                _implementation: optimismPortal,
                _data: abi.encodeCall(
                    IOptimismPortal.initialize,
                    (
                        IL2OutputOracle(l2OutputOracleProxy),
                        ISystemConfig(systemConfigProxy),
                        ISuperchainConfig(superchainConfigProxy)
                    )
                )
            });

            IOptimismPortal portal = IOptimismPortal(payable(optimismPortalProxy));
            string memory version = portal.version();
            console.log("OptimismPortal version: %s", version);

            ChainAssertions.checkOptimismPortal({ _contracts: _proxies(), _cfg: cfg, _isProxy: true });
        }
    }

    /// @notice Transfer ownership of the DisputeGameFactory contract to the final system owner
    function transferDisputeGameFactoryOwnership() public broadcast {
        console.log("Transferring DisputeGameFactory ownership to Safe");
        IDisputeGameFactory disputeGameFactory = IDisputeGameFactory(mustGetAddress("DisputeGameFactoryProxy"));
        address owner = disputeGameFactory.owner();
        address finalSystemOwner = cfg.finalSystemOwner();

        if (owner != finalSystemOwner) {
            disputeGameFactory.transferOwnership(finalSystemOwner);
            console.log("DisputeGameFactory ownership transferred to final system owner at: %s", finalSystemOwner);
        }
        ChainAssertions.checkDisputeGameFactory({
            _contracts: _proxies(),
            _expectedOwner: finalSystemOwner,
            _isProxy: true
        });
    }

    /// @notice Transfer ownership of the DelayedWETH contract to the final system owner
    function transferDelayedWETHOwnership() public broadcast {
        console.log("Transferring DelayedWETH ownership to Safe");
        IDelayedWETH weth = IDelayedWETH(mustGetAddress("DelayedWETHProxy"));
        address owner = weth.owner();

        address finalSystemOwner = cfg.finalSystemOwner();
        if (owner != finalSystemOwner) {
            weth.transferOwnership(finalSystemOwner);
            console.log("DelayedWETH ownership transferred to final system owner at: %s", finalSystemOwner);
        }
        ChainAssertions.checkDelayedWETH({
            _contracts: _proxies(),
            _cfg: cfg,
            _isProxy: true,
            _expectedOwner: finalSystemOwner
        });
    }

    /// @notice Transfer ownership of the permissioned DelayedWETH contract to the final system owner
    function transferPermissionedDelayedWETHOwnership() public broadcast {
        console.log("Transferring permissioned DelayedWETH ownership to Safe");
        IDelayedWETH weth = IDelayedWETH(mustGetAddress("PermissionedDelayedWETHProxy"));
        address owner = weth.owner();

        address finalSystemOwner = cfg.finalSystemOwner();
        if (owner != finalSystemOwner) {
            weth.transferOwnership(finalSystemOwner);
            console.log("DelayedWETH ownership transferred to final system owner at: %s", finalSystemOwner);
        }
        ChainAssertions.checkPermissionedDelayedWETH({
            _contracts: _proxies(),
            _cfg: cfg,
            _isProxy: true,
            _expectedOwner: finalSystemOwner
        });
    }

    /// @notice Load the appropriate mips absolute prestate for devenets depending on config environment.
    function loadMipsAbsolutePrestate() internal returns (Claim mipsAbsolutePrestate_) {
        if (block.chainid == Chains.LocalDevnet || block.chainid == Chains.GethDevnet) {
            if (Config.useMultithreadedCannon()) {
                return _loadDevnetMtMipsAbsolutePrestate();
            } else {
                return _loadDevnetStMipsAbsolutePrestate();
            }
        } else {
            console.log(
                "[Cannon Dispute Game] Using absolute prestate from config: %x", cfg.faultGameAbsolutePrestate()
            );
            mipsAbsolutePrestate_ = Claim.wrap(bytes32(cfg.faultGameAbsolutePrestate()));
        }
    }

    /// @notice Loads the singlethreaded mips absolute prestate from the prestate-proof for devnets otherwise
    ///         from the config.
    function _loadDevnetStMipsAbsolutePrestate() internal returns (Claim mipsAbsolutePrestate_) {
        // Fetch the absolute prestate dump
        string memory filePath = string.concat(vm.projectRoot(), "/../../op-program/bin/prestate-proof.json");
        string[] memory commands = new string[](3);
        commands[0] = "bash";
        commands[1] = "-c";
        commands[2] = string.concat("[[ -f ", filePath, " ]] && echo \"present\"");
        if (Process.run(commands).length == 0) {
            revert(
                "Deploy: cannon prestate dump not found, generate it with `make cannon-prestate` in the monorepo root"
            );
        }
        commands[2] = string.concat("cat ", filePath, " | jq -r .pre");
        mipsAbsolutePrestate_ = Claim.wrap(abi.decode(Process.run(commands), (bytes32)));
        console.log(
            "[Cannon Dispute Game] Using devnet MIPS Absolute prestate: %s",
            vm.toString(Claim.unwrap(mipsAbsolutePrestate_))
        );
    }

    /// @notice Loads the multithreaded mips absolute prestate from the prestate-proof-mt for devnets otherwise
    ///         from the config.
    function _loadDevnetMtMipsAbsolutePrestate() internal returns (Claim mipsAbsolutePrestate_) {
        // Fetch the absolute prestate dump
        string memory filePath = string.concat(vm.projectRoot(), "/../../op-program/bin/prestate-proof-mt.json");
        string[] memory commands = new string[](3);
        commands[0] = "bash";
        commands[1] = "-c";
        commands[2] = string.concat("[[ -f ", filePath, " ]] && echo \"present\"");
        if (Process.run(commands).length == 0) {
            revert(
                "Deploy: MT-Cannon prestate dump not found, generate it with `make cannon-prestate-mt` in the monorepo root"
            );
        }
        commands[2] = string.concat("cat ", filePath, " | jq -r .pre");
        mipsAbsolutePrestate_ = Claim.wrap(abi.decode(Process.run(commands), (bytes32)));
        console.log(
            "[MT-Cannon Dispute Game] Using devnet MIPS2 Absolute prestate: %s",
            vm.toString(Claim.unwrap(mipsAbsolutePrestate_))
        );
    }

    /// @notice Sets the implementation for the `CANNON` game type in the `DisputeGameFactory`
    function setCannonFaultGameImplementation(bool _allowUpgrade) public broadcast {
        console.log("Setting Cannon FaultDisputeGame implementation");
        IDisputeGameFactory factory = IDisputeGameFactory(mustGetAddress("DisputeGameFactoryProxy"));
        IDelayedWETH weth = IDelayedWETH(mustGetAddress("DelayedWETHProxy"));

        // Set the Cannon FaultDisputeGame implementation in the factory.
        _setFaultGameImplementation({
            _factory: factory,
            _allowUpgrade: _allowUpgrade,
            _params: FaultDisputeGameParams({
                anchorStateRegistry: IAnchorStateRegistry(mustGetAddress("AnchorStateRegistryProxy")),
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
        IDisputeGameFactory factory = IDisputeGameFactory(mustGetAddress("DisputeGameFactoryProxy"));
        IDelayedWETH weth = IDelayedWETH(mustGetAddress("PermissionedDelayedWETHProxy"));

        // Set the Cannon FaultDisputeGame implementation in the factory.
        _setFaultGameImplementation({
            _factory: factory,
            _allowUpgrade: _allowUpgrade,
            _params: FaultDisputeGameParams({
                anchorStateRegistry: IAnchorStateRegistry(mustGetAddress("AnchorStateRegistryProxy")),
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
        IDisputeGameFactory factory = IDisputeGameFactory(mustGetAddress("DisputeGameFactoryProxy"));
        IDelayedWETH weth = IDelayedWETH(mustGetAddress("DelayedWETHProxy"));

        Claim outputAbsolutePrestate = Claim.wrap(bytes32(cfg.faultGameAbsolutePrestate()));
        _setFaultGameImplementation({
            _factory: factory,
            _allowUpgrade: _allowUpgrade,
            _params: FaultDisputeGameParams({
                anchorStateRegistry: IAnchorStateRegistry(mustGetAddress("AnchorStateRegistryProxy")),
                weth: weth,
                gameType: GameTypes.ALPHABET,
                absolutePrestate: outputAbsolutePrestate,
                faultVm: IBigStepper(new AlphabetVM(outputAbsolutePrestate, IPreimageOracle(mustGetAddress("PreimageOracle")))),
                // The max depth for the alphabet trace is always 3. Add 1 because split depth is fully inclusive.
                maxGameDepth: cfg.faultGameSplitDepth() + 3 + 1,
                maxClockDuration: Duration.wrap(uint64(cfg.faultGameMaxClockDuration()))
            })
        });
    }

    /// @notice Sets the implementation for the `ALPHABET` game type in the `DisputeGameFactory`
    function setFastFaultGameImplementation(bool _allowUpgrade) public onlyDevnet broadcast {
        console.log("Setting Fast FaultDisputeGame implementation");
        IDisputeGameFactory factory = IDisputeGameFactory(mustGetAddress("DisputeGameFactoryProxy"));
        IDelayedWETH weth = IDelayedWETH(mustGetAddress("DelayedWETHProxy"));

        Claim outputAbsolutePrestate = Claim.wrap(bytes32(cfg.faultGameAbsolutePrestate()));
        IPreimageOracle fastOracle = IPreimageOracle(
            DeployUtils.create2AndSave({
                _save: this,
                _salt: _implSalt(),
                _name: "PreimageOracle",
                _nick: "FastPreimageOracle",
                _args: DeployUtils.encodeConstructor(
                    abi.encodeCall(IPreimageOracle.__constructor__, (cfg.preimageOracleMinProposalSize(), 0))
                )
            })
        );
        _setFaultGameImplementation({
            _factory: factory,
            _allowUpgrade: _allowUpgrade,
            _params: FaultDisputeGameParams({
                anchorStateRegistry: IAnchorStateRegistry(mustGetAddress("AnchorStateRegistryProxy")),
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
        IDisputeGameFactory _factory,
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

        // Redefine _param variable to avoid stack too deep error during compilation
        FaultDisputeGameParams memory _params_ = _params;
        if (rawGameType != GameTypes.PERMISSIONED_CANNON.raw()) {
            _factory.setImplementation(
                _params_.gameType,
                IDisputeGame(
                    DeployUtils.create2AndSave({
                        _save: this,
                        _salt: _implSalt(),
                        _name: "FaultDisputeGame",
                        _nick: string.concat("FaultDisputeGame_", vm.toString(rawGameType)),
                        _args: DeployUtils.encodeConstructor(
                            abi.encodeCall(
                                IFaultDisputeGame.__constructor__,
                                (
                                    _params_.gameType,
                                    _params_.absolutePrestate,
                                    _params_.maxGameDepth,
                                    cfg.faultGameSplitDepth(),
                                    Duration.wrap(uint64(cfg.faultGameClockExtension())),
                                    _params_.maxClockDuration,
                                    _params_.faultVm,
                                    _params_.weth,
                                    IAnchorStateRegistry(mustGetAddress("AnchorStateRegistryProxy")),
                                    cfg.l2ChainID()
                                )
                            )
                        )
                    })
                )
            );
        } else {
            _factory.setImplementation(
                _params_.gameType,
                IDisputeGame(
                    DeployUtils.create2AndSave({
                        _save: this,
                        _salt: _implSalt(),
                        _name: "PermissionedDisputeGame",
                        _args: DeployUtils.encodeConstructor(
                            abi.encodeCall(
                                IPermissionedDisputeGame.__constructor__,
                                (
                                    _params_.gameType,
                                    _params_.absolutePrestate,
                                    _params_.maxGameDepth,
                                    cfg.faultGameSplitDepth(),
                                    Duration.wrap(uint64(cfg.faultGameClockExtension())),
                                    _params_.maxClockDuration,
                                    _params_.faultVm,
                                    _params_.weth,
                                    _params_.anchorStateRegistry,
                                    cfg.l2ChainID(),
                                    cfg.l2OutputOracleProposer(),
                                    cfg.l2OutputOracleChallenger()
                                )
                            )
                        )
                    })
                )
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

        IProxyAdmin proxyAdmin = IProxyAdmin(payable(mustGetAddress("ProxyAdmin")));
        proxyAdmin.upgradeAndCall({
            _proxy: payable(dataAvailabilityChallengeProxy),
            _implementation: dataAvailabilityChallenge,
            _data: abi.encodeCall(
                IDataAvailabilityChallenge.initialize,
                (finalSystemOwner, daChallengeWindow, daResolveWindow, daBondSize, daResolverRefundPercentage)
            )
        });

        IDataAvailabilityChallenge dac = IDataAvailabilityChallenge(payable(dataAvailabilityChallengeProxy));
        string memory version = dac.version();
        console.log("DataAvailabilityChallenge version: %s", version);

        require(dac.owner() == finalSystemOwner);
        require(dac.challengeWindow() == daChallengeWindow);
        require(dac.resolveWindow() == daResolveWindow);
        require(dac.bondSize() == daBondSize);
        require(dac.resolverRefundPercentage() == daResolverRefundPercentage);
    }
}
