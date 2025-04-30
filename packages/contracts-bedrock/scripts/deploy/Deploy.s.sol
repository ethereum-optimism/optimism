// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

// Testing
import { VmSafe } from "forge-std/Vm.sol";
import { console2 as console } from "forge-std/console2.sol";
import { stdJson } from "forge-std/StdJson.sol";
import { AlphabetVM } from "test/mocks/AlphabetVM.sol";
import { EIP1967Helper } from "test/mocks/EIP1967Helper.sol";

// Scripts
import { Deployer } from "scripts/deploy/Deployer.sol";
import { Chains } from "scripts/libraries/Chains.sol";
import { Config } from "scripts/libraries/Config.sol";
import { StateDiff } from "scripts/libraries/StateDiff.sol";
import { Process } from "scripts/libraries/Process.sol";
import { ChainAssertions } from "scripts/deploy/ChainAssertions.sol";
import { DeployUtils } from "scripts/libraries/DeployUtils.sol";
import { DeploySuperchainInput, DeploySuperchain, DeploySuperchainOutput } from "scripts/deploy/DeploySuperchain.s.sol";
import { DeployImplementations2 } from "scripts/deploy/DeployImplementations2.s.sol";

// Libraries
import { Constants } from "src/libraries/Constants.sol";
import { Types } from "scripts/libraries/Types.sol";
import { Duration } from "src/dispute/lib/LibUDT.sol";
import { StorageSlot, ForgeArtifacts } from "scripts/libraries/ForgeArtifacts.sol";
import { GameType, Claim, GameTypes, Proposal, Hash } from "src/dispute/lib/Types.sol";

// Interfaces
import { IOPContractsManager } from "interfaces/L1/IOPContractsManager.sol";
import { IProxy } from "interfaces/universal/IProxy.sol";
import { IProxyAdmin } from "interfaces/universal/IProxyAdmin.sol";
import { ISuperchainConfig } from "interfaces/L1/ISuperchainConfig.sol";
import { ISystemConfig } from "interfaces/L1/ISystemConfig.sol";
import { IDataAvailabilityChallenge } from "interfaces/L1/IDataAvailabilityChallenge.sol";
import { ProtocolVersion } from "interfaces/L1/IProtocolVersions.sol";
import { IBigStepper } from "interfaces/dispute/IBigStepper.sol";
import { IDisputeGameFactory } from "interfaces/dispute/IDisputeGameFactory.sol";
import { IDisputeGame } from "interfaces/dispute/IDisputeGame.sol";
import { IFaultDisputeGame } from "interfaces/dispute/IFaultDisputeGame.sol";
import { IDelayedWETH } from "interfaces/dispute/IDelayedWETH.sol";
import { IAnchorStateRegistry } from "interfaces/dispute/IAnchorStateRegistry.sol";
import { IMIPS } from "interfaces/cannon/IMIPS.sol";
import { IPermissionedDisputeGame } from "interfaces/dispute/IPermissionedDisputeGame.sol";
import { IPreimageOracle } from "interfaces/cannon/IPreimageOracle.sol";
import { IProtocolVersions } from "interfaces/L1/IProtocolVersions.sol";

/// @title Deploy
/// @notice Script used to deploy a bedrock system. The entire system is deployed within the `run` function.
///         To add a new contract to the system, add a public function that deploys that individual contract.
///         Then add a call to that function inside of `run`. Be sure to call the `save` function after each
///         deployment so that hardhat-deploy style artifacts can be generated using a call to `sync()`.
///         This contract must not have constructor logic because it is set into state using `etch`.
contract Deploy is Deployer {
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
        string memory json = StateDiff.encodeAccountAccesses(accesses);
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
            L1CrossDomainMessenger: artifacts.getAddress("L1CrossDomainMessengerProxy"),
            L1StandardBridge: artifacts.getAddress("L1StandardBridgeProxy"),
            L2OutputOracle: artifacts.getAddress("L2OutputOracleProxy"),
            DisputeGameFactory: artifacts.getAddress("DisputeGameFactoryProxy"),
            DelayedWETH: artifacts.getAddress("DelayedWETHProxy"),
            PermissionedDelayedWETH: artifacts.getAddress("PermissionedDelayedWETHProxy"),
            AnchorStateRegistry: artifacts.getAddress("AnchorStateRegistryProxy"),
            OptimismMintableERC20Factory: artifacts.getAddress("OptimismMintableERC20FactoryProxy"),
            OptimismPortal: artifacts.getAddress("OptimismPortalProxy"),
            ETHLockbox: artifacts.getAddress("ETHLockboxProxy"),
            SystemConfig: artifacts.getAddress("SystemConfigProxy"),
            L1ERC721Bridge: artifacts.getAddress("L1ERC721BridgeProxy"),
            ProtocolVersions: artifacts.getAddress("ProtocolVersionsProxy"),
            SuperchainConfig: artifacts.getAddress("SuperchainConfigProxy")
        });
    }

    ////////////////////////////////////////////////////////////////
    //                    SetUp and Run                           //
    ////////////////////////////////////////////////////////////////

    /// @notice Deploy all of the L1 contracts necessary for a full Superchain with a single Op Chain.
    function run() public {
        console.log("Deploying a fresh OP Stack including SuperchainConfig");
        _run({ _needsSuperchain: true });
    }

    /// @notice Deploy a new OP Chain using an existing SuperchainConfig and ProtocolVersions
    /// @param _superchainConfigProxy Address of the existing SuperchainConfig proxy
    /// @param _protocolVersionsProxy Address of the existing ProtocolVersions proxy
    function runWithSuperchain(address payable _superchainConfigProxy, address payable _protocolVersionsProxy) public {
        require(_superchainConfigProxy != address(0), "Deploy: must specify address for superchain config proxy");
        require(_protocolVersionsProxy != address(0), "Deploy: must specify address for protocol versions proxy");

        vm.chainId(cfg.l1ChainID());

        console.log("Deploying a fresh OP Stack with existing SuperchainConfig and ProtocolVersions");

        IProxy scProxy = IProxy(_superchainConfigProxy);
        artifacts.save("SuperchainConfigImpl", scProxy.implementation());
        artifacts.save("SuperchainConfigProxy", _superchainConfigProxy);

        IProxy pvProxy = IProxy(_protocolVersionsProxy);
        artifacts.save("ProtocolVersionsImpl", pvProxy.implementation());
        artifacts.save("ProtocolVersionsProxy", _protocolVersionsProxy);

        _run({ _needsSuperchain: false });
    }

    /// @notice Used for L1 alloc generation.
    function runWithStateDump() public {
        vm.chainId(cfg.l1ChainID());
        _run({ _needsSuperchain: true });
        vm.dumpState(Config.stateDumpPath(""));
    }

    /// @notice Deploy all L1 contracts and write the state diff to a file.
    ///         Used to generate kontrol tests.
    function runWithStateDiff() public stateDiff {
        _run({ _needsSuperchain: true });
    }

    /// @notice Internal function containing the deploy logic.
    function _run(bool _needsSuperchain) internal virtual {
        console.log("start of L1 Deploy!");

        // Set up the Superchain if needed.
        if (_needsSuperchain) {
            deploySuperchain();
        }

        deployImplementations({ _isInterop: cfg.useInterop() });

        // Deploy Current OPChain Contracts
        deployOpChain();

        // Set the respected game type according to the deploy config
        vm.startPrank(ISuperchainConfig(artifacts.mustGetAddress("SuperchainConfigProxy")).guardian());
        IAnchorStateRegistry(artifacts.mustGetAddress("AnchorStateRegistryProxy")).setRespectedGameType(
            GameType.wrap(uint32(cfg.respectedGameType()))
        );
        vm.stopPrank();

        if (cfg.useAltDA()) {
            bytes32 typeHash = keccak256(bytes(cfg.daCommitmentType()));
            bytes32 keccakHash = keccak256(bytes("KeccakCommitment"));
            if (typeHash == keccakHash) {
                deployOpAltDA();
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
        dsi.set(dsi.requiredProtocolVersion.selector, ProtocolVersion.wrap(cfg.requiredProtocolVersion()));
        dsi.set(dsi.recommendedProtocolVersion.selector, ProtocolVersion.wrap(cfg.recommendedProtocolVersion()));

        // Run the deployment script.
        ds.run(dsi, dso);
        artifacts.save("SuperchainProxyAdmin", address(dso.superchainProxyAdmin()));
        artifacts.save("SuperchainConfigProxy", address(dso.superchainConfigProxy()));
        artifacts.save("SuperchainConfigImpl", address(dso.superchainConfigImpl()));
        artifacts.save("ProtocolVersionsProxy", address(dso.protocolVersionsProxy()));
        artifacts.save("ProtocolVersionsImpl", address(dso.protocolVersionsImpl()));

        // First run assertions for the ProtocolVersions and SuperchainConfig proxy contracts.
        Types.ContractSet memory contracts = _proxies();
        ChainAssertions.checkProtocolVersions({ _contracts: contracts, _cfg: cfg, _isProxy: true });
        ChainAssertions.checkSuperchainConfig({ _contracts: contracts, _cfg: cfg, _isProxy: true });

        // Then replace the ProtocolVersions proxy with the implementation address and run assertions on it.
        contracts.ProtocolVersions = artifacts.mustGetAddress("ProtocolVersionsImpl");
        ChainAssertions.checkProtocolVersions({ _contracts: contracts, _cfg: cfg, _isProxy: false });

        // Finally replace the SuperchainConfig proxy with the implementation address and run assertions on it.
        contracts.SuperchainConfig = artifacts.mustGetAddress("SuperchainConfigImpl");
        ChainAssertions.checkSuperchainConfig({ _contracts: contracts, _cfg: cfg, _isProxy: false });
    }

    /// @notice Deploy all of the implementations
    /// @param _isInterop Whether to use interop
    function deployImplementations(bool _isInterop) public {
        require(_isInterop == cfg.useInterop(), "Deploy: Interop setting mismatch.");

        console.log("Deploying implementations");

        ISuperchainConfig superchainConfig = ISuperchainConfig(artifacts.mustGetAddress("SuperchainConfigProxy"));
        IProxyAdmin superchainProxyAdmin = IProxyAdmin(EIP1967Helper.getAdmin(address(superchainConfig)));

        DeployImplementations2 di2 = new DeployImplementations2();
        DeployImplementations2.Input memory dii = DeployImplementations2.Input({
            withdrawalDelaySeconds: cfg.faultGameWithdrawalDelay(),
            minProposalSizeBytes: cfg.preimageOracleMinProposalSize(),
            challengePeriodSeconds: cfg.preimageOracleChallengePeriod(),
            proofMaturityDelaySeconds: cfg.proofMaturityDelaySeconds(),
            disputeGameFinalityDelaySeconds: cfg.disputeGameFinalityDelaySeconds(),
            mipsVersion: 6,
            l1ContractsRelease: "dev",
            superchainConfigProxy: superchainConfig,
            protocolVersionsProxy: IProtocolVersions(artifacts.mustGetAddress("ProtocolVersionsProxy")),
            superchainProxyAdmin: superchainProxyAdmin,
            upgradeController: superchainProxyAdmin.owner()
        });

        DeployImplementations2.Output memory dio = di2.run(dii);

        // Save the implementation addresses which are needed outside of this function or script.
        // When called in a fork test, this will overwrite the existing implementations.
        artifacts.save("MipsSingleton", address(dio.mipsSingleton));
        artifacts.save("OPContractsManager", address(dio.opcm));
        artifacts.save("DelayedWETHImpl", address(dio.delayedWETHImpl));

        // Get a contract set from the implementation addresses which were just deployed.
        Types.ContractSet memory impls = Types.ContractSet({
            L1CrossDomainMessenger: address(dio.l1CrossDomainMessengerImpl),
            L1StandardBridge: address(dio.l1StandardBridgeImpl),
            L2OutputOracle: address(0),
            DisputeGameFactory: address(dio.disputeGameFactoryImpl),
            DelayedWETH: address(dio.delayedWETHImpl),
            PermissionedDelayedWETH: address(dio.delayedWETHImpl),
            AnchorStateRegistry: address(0),
            OptimismMintableERC20Factory: address(dio.optimismMintableERC20FactoryImpl),
            OptimismPortal: address(dio.optimismPortalImpl),
            ETHLockbox: address(dio.ethLockboxImpl),
            SystemConfig: address(dio.systemConfigImpl),
            L1ERC721Bridge: address(dio.l1ERC721BridgeImpl),
            ProtocolVersions: address(dio.protocolVersionsImpl),
            SuperchainConfig: address(dio.superchainConfigImpl)
        });

        ChainAssertions.checkL1CrossDomainMessenger({ _contracts: impls, _vm: vm, _isProxy: false });
        ChainAssertions.checkL1StandardBridge({ _contracts: impls, _isProxy: false });
        ChainAssertions.checkL1ERC721Bridge({ _contracts: impls, _isProxy: false });
        ChainAssertions.checkOptimismPortal2({ _contracts: impls, _cfg: cfg, _isProxy: false });
        ChainAssertions.checkETHLockbox({ _contracts: impls, _cfg: cfg, _isProxy: false });
        ChainAssertions.checkOptimismMintableERC20Factory({ _contracts: impls, _isProxy: false });
        ChainAssertions.checkDisputeGameFactory({ _contracts: impls, _expectedOwner: address(0), _isProxy: false });
        ChainAssertions.checkDelayedWETH({ _contracts: impls, _cfg: cfg, _isProxy: false, _expectedOwner: address(0) });
        ChainAssertions.checkPreimageOracle({ _oracle: IPreimageOracle(address(dio.preimageOracleSingleton)), _cfg: cfg });
        ChainAssertions.checkMIPS({
            _mips: IMIPS(address(dio.mipsSingleton)),
            _oracle: IPreimageOracle(address(dio.preimageOracleSingleton))
        });
        ChainAssertions.checkOPContractsManager({
            _impls: impls,
            _proxies: _proxies(),
            _opcm: IOPContractsManager(address(dio.opcm)),
            _mips: IMIPS(address(dio.mipsSingleton)),
            _superchainProxyAdmin: superchainProxyAdmin
        });
        ChainAssertions.checkSystemConfig({ _contracts: impls, _cfg: cfg, _isProxy: false });
    }

    /// @notice Deploy all of the OP Chain specific contracts
    function deployOpChain() public {
        console.log("Deploying OP Chain");

        // Ensure that the requisite contracts are deployed
        IOPContractsManager opcm = IOPContractsManager(artifacts.mustGetAddress("OPContractsManager"));

        IOPContractsManager.DeployInput memory deployInput = getDeployInput();
        IOPContractsManager.DeployOutput memory deployOutput = opcm.deploy(deployInput);

        // Store code in the Final system owner address so that it can be used for prank delegatecalls
        // Store "fe" opcode so that accidental calls to this address revert
        vm.etch(cfg.finalSystemOwner(), hex"fe");

        // Save all deploy outputs from the OPCM, in the order they are declared in the DeployOutput struct
        artifacts.save("ProxyAdmin", address(deployOutput.opChainProxyAdmin));
        artifacts.save("AddressManager", address(deployOutput.addressManager));
        artifacts.save("L1ERC721BridgeProxy", address(deployOutput.l1ERC721BridgeProxy));
        artifacts.save("SystemConfigProxy", address(deployOutput.systemConfigProxy));
        artifacts.save("OptimismMintableERC20FactoryProxy", address(deployOutput.optimismMintableERC20FactoryProxy));
        artifacts.save("L1StandardBridgeProxy", address(deployOutput.l1StandardBridgeProxy));
        artifacts.save("L1CrossDomainMessengerProxy", address(deployOutput.l1CrossDomainMessengerProxy));
        artifacts.save("ETHLockboxProxy", address(deployOutput.ethLockboxProxy));

        // Fault Proof contracts
        artifacts.save("DisputeGameFactoryProxy", address(deployOutput.disputeGameFactoryProxy));
        artifacts.save("PermissionedDelayedWETHProxy", address(deployOutput.delayedWETHPermissionedGameProxy));
        artifacts.save("AnchorStateRegistryProxy", address(deployOutput.anchorStateRegistryProxy));
        artifacts.save("PermissionedDisputeGame", address(deployOutput.permissionedDisputeGame));
        artifacts.save("OptimismPortalProxy", address(deployOutput.optimismPortalProxy));
        artifacts.save("OptimismPortal2Proxy", address(deployOutput.optimismPortalProxy));
        // Check if the permissionless game implementation is already set
        IDisputeGameFactory factory = IDisputeGameFactory(artifacts.mustGetAddress("DisputeGameFactoryProxy"));
        address permissionlessGameImpl = address(factory.gameImpls(GameTypes.CANNON));

        // Deploy and setup the PermissionlessDelayedWeth not provided by the OPCM.
        // If the following require statement is hit, you can delete the block of code after it.
        require(
            permissionlessGameImpl == address(0),
            "Deploy: The PermissionlessDelayedWETH is already set by the OPCM, it is no longer necessary to deploy it separately."
        );
        address delayedWETHImpl = artifacts.mustGetAddress("DelayedWETHImpl");
        address delayedWETHPermissionlessGameProxy =
            deployERC1967ProxyWithOwner("DelayedWETHProxy", address(deployOutput.opChainProxyAdmin));
        vm.broadcast(address(deployOutput.opChainProxyAdmin));
        IProxy(payable(delayedWETHPermissionlessGameProxy)).upgradeToAndCall({
            _implementation: delayedWETHImpl,
            _data: abi.encodeCall(IDelayedWETH.initialize, (deployOutput.systemConfigProxy))
        });

        setAlphabetFaultGameImplementation();
        setSuperFaultGameImplementation();
        setSuperPermissionedGameImplementation();
        setFastFaultGameImplementation();
        setCannonFaultGameImplementation();
    }

    /// @notice Add AltDA setup to the OP chain
    function deployOpAltDA() public {
        console.log("Deploying OP AltDA");
        deployDataAvailabilityChallengeProxy();
        deployDataAvailabilityChallenge();
        initializeDataAvailabilityChallenge();
    }

    ////////////////////////////////////////////////////////////////
    //                Proxy Deployment Functions                  //
    ////////////////////////////////////////////////////////////////

    /// @notice Deploys an ERC1967Proxy contract with the ProxyAdmin as the owner.
    /// @param _name The name of the proxy contract to be deployed.
    /// @return addr_ The address of the deployed proxy contract.
    function deployERC1967Proxy(string memory _name) public returns (address addr_) {
        addr_ = deployERC1967ProxyWithOwner(_name, artifacts.mustGetAddress("ProxyAdmin"));
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
                _save: artifacts,
                _salt: keccak256(abi.encode(_implSalt(), _name)),
                _name: "Proxy",
                _nick: _name,
                _args: DeployUtils.encodeConstructor(abi.encodeCall(IProxy.__constructor__, (_proxyOwner)))
            })
        );
        require(EIP1967Helper.getAdmin(address(proxy)) == _proxyOwner, "Deploy: EIP1967Proxy admin not set");
        addr_ = address(proxy);
    }

    /// @notice Deploy the DataAvailabilityChallengeProxy
    function deployDataAvailabilityChallengeProxy() public broadcast returns (address addr_) {
        address proxyAdmin = artifacts.mustGetAddress("ProxyAdmin");
        IProxy proxy = IProxy(
            DeployUtils.create2AndSave({
                _save: artifacts,
                _salt: _implSalt(),
                _name: "Proxy",
                _nick: "DataAvailabilityChallengeProxy",
                _args: DeployUtils.encodeConstructor(abi.encodeCall(IProxy.__constructor__, (proxyAdmin)))
            })
        );
        require(
            EIP1967Helper.getAdmin(address(proxy)) == proxyAdmin, "Deploy: DataAvailabilityChallengeProxy admin not set"
        );
        addr_ = address(proxy);
    }

    ////////////////////////////////////////////////////////////////
    //             Implementation Deployment Functions            //
    ////////////////////////////////////////////////////////////////

    /// @notice Deploy the DataAvailabilityChallenge
    function deployDataAvailabilityChallenge() public broadcast returns (address addr_) {
        IDataAvailabilityChallenge dac = IDataAvailabilityChallenge(
            DeployUtils.create2AndSave({
                _save: artifacts,
                _salt: _implSalt(),
                _name: "DataAvailabilityChallenge",
                _nick: "DataAvailabilityChallengeImpl",
                _args: DeployUtils.encodeConstructor(abi.encodeCall(IDataAvailabilityChallenge.__constructor__, ()))
            })
        );
        addr_ = address(dac);
    }

    ////////////////////////////////////////////////////////////////
    //                    Initialize Functions                    //
    ////////////////////////////////////////////////////////////////

    /// @notice Initialize the SystemConfig
    function initializeSystemConfig() public broadcast {
        console.log("Upgrading and initializing SystemConfig proxy");
        address systemConfigProxy = artifacts.mustGetAddress("SystemConfigProxy");
        address systemConfig = artifacts.mustGetAddress("SystemConfigImpl");

        bytes32 batcherHash = bytes32(uint256(uint160(cfg.batchSenderAddress())));

        IProxyAdmin proxyAdmin = IProxyAdmin(payable(artifacts.mustGetAddress("ProxyAdmin")));
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
                        l1CrossDomainMessenger: artifacts.mustGetAddress("L1CrossDomainMessengerProxy"),
                        l1ERC721Bridge: artifacts.mustGetAddress("L1ERC721BridgeProxy"),
                        l1StandardBridge: artifacts.mustGetAddress("L1StandardBridgeProxy"),
                        optimismPortal: artifacts.mustGetAddress("OptimismPortalProxy"),
                        optimismMintableERC20Factory: artifacts.mustGetAddress("OptimismMintableERC20FactoryProxy")
                    }),
                    cfg.l2ChainID(),
                    ISuperchainConfig(artifacts.mustGetAddress("SuperchainConfigProxy"))
                )
            )
        });

        ISystemConfig config = ISystemConfig(systemConfigProxy);
        string memory version = config.version();
        console.log("SystemConfig version: %s", version);

        ChainAssertions.checkSystemConfig({ _contracts: _proxies(), _cfg: cfg, _isProxy: true });
    }

    /// @notice Initialize the DataAvailabilityChallenge
    function initializeDataAvailabilityChallenge() public {
        console.log("Upgrading and initializing DataAvailabilityChallenge proxy");
        address dataAvailabilityChallengeProxy = artifacts.mustGetAddress("DataAvailabilityChallengeProxy");
        address dataAvailabilityChallenge = artifacts.mustGetAddress("DataAvailabilityChallengeImpl");

        address finalSystemOwner = cfg.finalSystemOwner();
        uint256 daChallengeWindow = cfg.daChallengeWindow();
        uint256 daResolveWindow = cfg.daResolveWindow();
        uint256 daBondSize = cfg.daBondSize();
        uint256 daResolverRefundPercentage = cfg.daResolverRefundPercentage();

        IProxyAdmin proxyAdmin = IProxyAdmin(payable(artifacts.mustGetAddress("ProxyAdmin")));
        vm.prank(proxyAdmin.owner());
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

        require(dac.owner() == finalSystemOwner, "Deploy: DataAvailabilityChallenge owner not set");
        require(
            dac.challengeWindow() == daChallengeWindow, "Deploy: DataAvailabilityChallenge challenge window not set"
        );
        require(dac.resolveWindow() == daResolveWindow, "Deploy: DataAvailabilityChallenge resolve window not set");
        require(dac.bondSize() == daBondSize, "Deploy: DataAvailabilityChallenge bond size not set");
        require(
            dac.resolverRefundPercentage() == daResolverRefundPercentage,
            "Deploy: DataAvailabilityChallenge resolver refund percentage not set"
        );
    }

    ////////////////////////////////////////////////////////////////
    //         Ownership Transfer Helper Functions                //
    ////////////////////////////////////////////////////////////////

    /// @notice Transfer ownership of the ProxyAdmin contract to the final system owner
    function transferProxyAdminOwnership() public broadcast {
        // Get the ProxyAdmin contract.
        IProxyAdmin proxyAdmin = IProxyAdmin(artifacts.mustGetAddress("ProxyAdmin"));

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

    ///////////////////////////////////////////////////////////
    //         Proofs setup helper functions                 //
    ///////////////////////////////////////////////////////////

    /// @notice Load the appropriate mips absolute prestate for devenets depending on config environment.
    function loadMipsAbsolutePrestate() internal returns (Claim mipsAbsolutePrestate_) {
        if (block.chainid == Chains.LocalDevnet || block.chainid == Chains.GethDevnet) {
            return _loadDevnetMtMipsAbsolutePrestate();
        } else {
            console.log(
                "[Cannon Dispute Game] Using absolute prestate from config: %x", cfg.faultGameAbsolutePrestate()
            );
            mipsAbsolutePrestate_ = Claim.wrap(bytes32(cfg.faultGameAbsolutePrestate()));
        }
    }

    function loadInteropDevnetAbsolutePrestate() internal returns (Claim interopDevnetAbsolutePrestate_) {
        string memory filePath = string.concat(vm.projectRoot(), "/../../op-program/bin/prestate-proof-interop.json");
        if (bytes(Process.bash(string.concat("[[ -f ", filePath, " ]] && echo \"present\""))).length == 0) {
            revert(
                "Deploy: cannon prestate dump not found, generate it with `make cannon-prestate` in the monorepo root"
            );
        }
        interopDevnetAbsolutePrestate_ =
            Claim.wrap(abi.decode(bytes(Process.bash(string.concat("cat ", filePath, " | jq -r .pre"))), (bytes32)));
        console.log(
            "[Cannon Dispute Game] Using devnet Interop Devnet Absolute Prestate: %s",
            vm.toString(Claim.unwrap(interopDevnetAbsolutePrestate_))
        );
    }

    /// @notice Loads the multithreaded mips absolute prestate from the prestate-proof-mt64 for devnets otherwise
    ///         from the config.
    function _loadDevnetMtMipsAbsolutePrestate() internal returns (Claim mipsAbsolutePrestate_) {
        // Fetch the absolute prestate dump
        string memory filePath = string.concat(vm.projectRoot(), "/../../op-program/bin/prestate-proof-mt64.json");
        if (bytes(Process.bash(string.concat("[[ -f ", filePath, " ]] && echo \"present\""))).length == 0) {
            revert(
                "Deploy: MT-Cannon prestate dump not found, generate it with `make cannon-prestate-mt64` in the monorepo root"
            );
        }
        mipsAbsolutePrestate_ =
            Claim.wrap(abi.decode(bytes(Process.bash(string.concat("cat ", filePath, " | jq -r .pre"))), (bytes32)));
        console.log(
            "[MT-Cannon Dispute Game] Using devnet MIPS64 Absolute prestate: %s",
            vm.toString(Claim.unwrap(mipsAbsolutePrestate_))
        );
    }

    /// @notice Sets the implementation for the `CANNON` game type in the `DisputeGameFactory`
    function setCannonFaultGameImplementation() public {
        console.log("Setting Cannon FaultDisputeGame implementation");
        address opcm = artifacts.mustGetAddress("OPContractsManager");
        IProxyAdmin proxyAdmin = IProxyAdmin(artifacts.mustGetAddress("ProxyAdmin"));

        IOPContractsManager.AddGameInput[] memory addGameInput = new IOPContractsManager.AddGameInput[](1);
        addGameInput[0] = IOPContractsManager.AddGameInput({
            saltMixer: "CannonFaultGame",
            systemConfig: ISystemConfig(artifacts.mustGetAddress("SystemConfigProxy")),
            proxyAdmin: proxyAdmin,
            delayedWETH: IDelayedWETH(artifacts.mustGetAddress("DelayedWETHProxy")),
            disputeGameType: GameTypes.CANNON,
            disputeAbsolutePrestate: loadMipsAbsolutePrestate(),
            disputeMaxGameDepth: cfg.faultGameMaxDepth(),
            disputeSplitDepth: cfg.faultGameSplitDepth(),
            disputeClockExtension: Duration.wrap(uint64(cfg.faultGameClockExtension())),
            disputeMaxClockDuration: Duration.wrap(uint64(cfg.faultGameMaxClockDuration())),
            initialBond: 0.08 ether,
            vm: IBigStepper(artifacts.mustGetAddress("MipsSingleton")),
            permissioned: false
        });

        vm.prank(cfg.finalSystemOwner(), true);
        (bool success,) = opcm.delegatecall(abi.encodeCall(IOPContractsManager.addGameType, (addGameInput)));
        require(success, "Deploy: Cannon FaultDisputeGame implementation not set");
    }

    /// @notice Sets the implementation for the `ALPHABET` game type in the `DisputeGameFactory`
    function setAlphabetFaultGameImplementation() public onlyDevnet broadcast {
        console.log("Setting Alphabet FaultDisputeGame implementation");
        IDisputeGameFactory factory = IDisputeGameFactory(artifacts.mustGetAddress("DisputeGameFactoryProxy"));
        IDelayedWETH weth = IDelayedWETH(artifacts.mustGetAddress("DelayedWETHProxy"));

        Claim outputAbsolutePrestate = Claim.wrap(bytes32(cfg.faultGameAbsolutePrestate()));
        _setFaultGameImplementation({
            _factory: factory,
            _params: IFaultDisputeGame.GameConstructorParams({
                gameType: GameTypes.ALPHABET,
                absolutePrestate: outputAbsolutePrestate,
                // The max depth for the alphabet trace is always 3. Add 1 because split depth is fully inclusive.
                maxGameDepth: cfg.faultGameSplitDepth() + 3 + 1,
                splitDepth: cfg.faultGameSplitDepth(),
                clockExtension: Duration.wrap(uint64(cfg.faultGameClockExtension())),
                maxClockDuration: Duration.wrap(uint64(cfg.faultGameMaxClockDuration())),
                vm: IBigStepper(artifacts.mustGetAddress("MipsSingleton")),
                weth: weth,
                anchorStateRegistry: IAnchorStateRegistry(artifacts.mustGetAddress("AnchorStateRegistryProxy")),
                l2ChainId: cfg.l2ChainID()
            })
        });
    }

    /// @notice Sets the implementation for the `PERMISSIONED_SUPER_CANNON` game type in the `DisputeGameFactory`
    function setSuperPermissionedGameImplementation() public onlyDevnet broadcast {
        console.log("Setting SuperPermissionedDisputeGame implementation");
        IDisputeGameFactory factory = IDisputeGameFactory(artifacts.mustGetAddress("DisputeGameFactoryProxy"));
        IDelayedWETH weth = IDelayedWETH(artifacts.mustGetAddress("DelayedWETHProxy"));

        _setFaultGameImplementation({
            _factory: factory,
            _params: IFaultDisputeGame.GameConstructorParams({
                gameType: GameType.wrap(4),
                absolutePrestate: loadInteropDevnetAbsolutePrestate(),
                maxGameDepth: cfg.faultGameMaxDepth(),
                splitDepth: cfg.faultGameSplitDepth(),
                clockExtension: Duration.wrap(uint64(cfg.faultGameClockExtension())),
                maxClockDuration: Duration.wrap(uint64(cfg.faultGameMaxClockDuration())),
                vm: IBigStepper(artifacts.mustGetAddress("MipsSingleton")),
                weth: weth,
                anchorStateRegistry: IAnchorStateRegistry(artifacts.mustGetAddress("AnchorStateRegistryProxy")),
                l2ChainId: 0 // Unused Param on SuperDisputeGame
             })
        });
    }

    /// @notice Sets the implementation for the `SUPER_CANNON` game type in the `DisputeGameFactory`
    function setSuperFaultGameImplementation() public onlyDevnet broadcast {
        console.log("Setting SuperFaultDisputeGame implementation");
        IDisputeGameFactory factory = IDisputeGameFactory(artifacts.mustGetAddress("DisputeGameFactoryProxy"));
        IDelayedWETH weth = IDelayedWETH(artifacts.mustGetAddress("DelayedWETHProxy"));

        _setFaultGameImplementation({
            _factory: factory,
            _params: IFaultDisputeGame.GameConstructorParams({
                gameType: GameType.wrap(4),
                absolutePrestate: loadInteropDevnetAbsolutePrestate(),
                maxGameDepth: cfg.faultGameMaxDepth(),
                splitDepth: cfg.faultGameSplitDepth(),
                clockExtension: Duration.wrap(uint64(cfg.faultGameClockExtension())),
                maxClockDuration: Duration.wrap(uint64(cfg.faultGameMaxClockDuration())),
                vm: IBigStepper(artifacts.mustGetAddress("MipsSingleton")),
                weth: weth,
                anchorStateRegistry: IAnchorStateRegistry(artifacts.mustGetAddress("AnchorStateRegistryProxy")),
                l2ChainId: 0 // Unused Param on SuperDisputeGame
             })
        });
    }

    /// @notice Sets the implementation for the `ALPHABET` game type in the `DisputeGameFactory`
    function setFastFaultGameImplementation() public onlyDevnet broadcast {
        console.log("Setting Fast FaultDisputeGame implementation");
        IDisputeGameFactory factory = IDisputeGameFactory(artifacts.mustGetAddress("DisputeGameFactoryProxy"));
        IDelayedWETH weth = IDelayedWETH(artifacts.mustGetAddress("DelayedWETHProxy"));

        Claim outputAbsolutePrestate = Claim.wrap(bytes32(cfg.faultGameAbsolutePrestate()));
        IPreimageOracle fastOracle = IPreimageOracle(
            DeployUtils.create2AndSave({
                _save: artifacts,
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
            _params: IFaultDisputeGame.GameConstructorParams({
                gameType: GameTypes.FAST,
                absolutePrestate: outputAbsolutePrestate,
                // The max depth for the alphabet trace is always 3. Add 1 because split depth is fully inclusive.
                maxGameDepth: cfg.faultGameSplitDepth() + 3 + 1,
                splitDepth: cfg.faultGameSplitDepth(),
                clockExtension: Duration.wrap(uint64(cfg.faultGameClockExtension())),
                maxClockDuration: Duration.wrap(0), // Resolvable immediately
                vm: IBigStepper(new AlphabetVM(outputAbsolutePrestate, fastOracle)),
                weth: weth,
                anchorStateRegistry: IAnchorStateRegistry(artifacts.mustGetAddress("AnchorStateRegistryProxy")),
                l2ChainId: cfg.l2ChainID()
            })
        });
    }

    /// @notice Sets the implementation for the given fault game type in the `DisputeGameFactory`.
    function _setFaultGameImplementation(
        IDisputeGameFactory _factory,
        IFaultDisputeGame.GameConstructorParams memory _params
    )
        internal
    {
        if (address(_factory.gameImpls(_params.gameType)) != address(0)) {
            console.log(
                "[WARN] DisputeGameFactoryProxy: `FaultDisputeGame` implementation already set for game type: %s",
                vm.toString(GameType.unwrap(_params.gameType))
            );
            return;
        }

        uint32 rawGameType = GameType.unwrap(_params.gameType);
        require(
            rawGameType != GameTypes.PERMISSIONED_CANNON.raw(), "Deploy: Permissioned Game should be deployed by OPCM"
        );

        if (rawGameType == 4) {
            _factory.setImplementation(
                _params.gameType,
                IDisputeGame(
                    DeployUtils.create2AndSave({
                        _save: artifacts,
                        _salt: _implSalt(),
                        _name: "SuperFaultDisputeGame",
                        _nick: string.concat("SuperFaultDisputeGame_", vm.toString(rawGameType)),
                        _args: DeployUtils.encodeConstructor(abi.encodeCall(IFaultDisputeGame.__constructor__, (_params)))
                    })
                )
            );
        } else if (rawGameType == 5) {
            _factory.setImplementation(
                _params.gameType,
                IDisputeGame(
                    DeployUtils.create2AndSave({
                        _save: artifacts,
                        _salt: _implSalt(),
                        _name: "SuperPermissionedDisputeGame",
                        _nick: string.concat("SuperFaultDisputeGame_", vm.toString(rawGameType)),
                        _args: DeployUtils.encodeConstructor(
                            abi.encodeCall(
                                IPermissionedDisputeGame.__constructor__,
                                (_params, cfg.l2OutputOracleProposer(), cfg.l2OutputOracleChallenger())
                            )
                        )
                    })
                )
            );
        } else {
            _factory.setImplementation(
                _params.gameType,
                IDisputeGame(
                    DeployUtils.create2AndSave({
                        _save: artifacts,
                        _salt: _implSalt(),
                        _name: "FaultDisputeGame",
                        _nick: string.concat("FaultDisputeGame_", vm.toString(rawGameType)),
                        _args: DeployUtils.encodeConstructor(abi.encodeCall(IFaultDisputeGame.__constructor__, (_params)))
                    })
                )
            );
        }

        string memory gameTypeString;
        if (rawGameType == GameTypes.CANNON.raw()) {
            gameTypeString = "Cannon";
        } else if (rawGameType == GameTypes.ALPHABET.raw()) {
            gameTypeString = "Alphabet";
        } else if (rawGameType == GameTypes.OP_SUCCINCT.raw()) {
            gameTypeString = "OP Succinct";
        } else if (rawGameType == GameTypes.KAILUA.raw()) {
            gameTypeString = "Kailua";
        } else if (rawGameType == 4) {
            gameTypeString = "Super Cannon";
        } else if (rawGameType == 5) {
            gameTypeString = "Permissioned Super Cannon";
        } else {
            gameTypeString = "Unknown";
        }

        console.log(
            "DisputeGameFactoryProxy: set `FaultDisputeGame` implementation (Backend: %s | GameType: %s)",
            gameTypeString,
            vm.toString(rawGameType)
        );
    }

    /// @notice Get the DeployInput struct to use for testing
    function getDeployInput() public view returns (IOPContractsManager.DeployInput memory) {
        string memory saltMixer = "salt mixer";
        return IOPContractsManager.DeployInput({
            roles: IOPContractsManager.Roles({
                opChainProxyAdminOwner: cfg.finalSystemOwner(),
                systemConfigOwner: cfg.finalSystemOwner(),
                batcher: cfg.batchSenderAddress(),
                unsafeBlockSigner: cfg.p2pSequencerAddress(),
                proposer: cfg.l2OutputOracleProposer(),
                challenger: cfg.l2OutputOracleChallenger()
            }),
            basefeeScalar: cfg.basefeeScalar(),
            blobBasefeeScalar: cfg.blobbasefeeScalar(),
            l2ChainId: cfg.l2ChainID(),
            startingAnchorRoot: abi.encode(
                Proposal({ root: Hash.wrap(cfg.faultGameGenesisOutputRoot()), l2SequenceNumber: cfg.faultGameGenesisBlock() })
            ),
            saltMixer: saltMixer,
            gasLimit: uint64(cfg.l2GenesisBlockGasLimit()),
            disputeGameType: GameTypes.PERMISSIONED_CANNON,
            disputeAbsolutePrestate: Claim.wrap(bytes32(cfg.faultGameAbsolutePrestate())),
            disputeMaxGameDepth: cfg.faultGameMaxDepth(),
            disputeSplitDepth: cfg.faultGameSplitDepth(),
            disputeClockExtension: Duration.wrap(uint64(cfg.faultGameClockExtension())),
            disputeMaxClockDuration: Duration.wrap(uint64(cfg.faultGameMaxClockDuration()))
        });
    }

    /// @notice Reset the initialized value on a proxy contract so that it can be initialized again
    function resetInitializedProxy(string memory _contractName) internal {
        console.log("resetting initialized value on %s Proxy", _contractName);
        address proxy = artifacts.mustGetAddress(string.concat(_contractName, "Proxy"));
        StorageSlot memory slot = ForgeArtifacts.getInitializedSlot(_contractName);
        bytes32 slotVal = vm.load(proxy, bytes32(slot.slot));
        uint256 value = uint256(slotVal);
        value = value & ~(0xFF << (slot.offset * 8));
        slotVal = bytes32(value);
        vm.store(proxy, bytes32(slot.slot), slotVal);
    }
}
