// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

// Testing
import { VmSafe } from "forge-std/Vm.sol";
import { console2 as console } from "forge-std/console2.sol";
import { stdJson } from "forge-std/StdJson.sol";
import { EIP1967Helper } from "test/mocks/EIP1967Helper.sol";

// Scripts
import { Deployer } from "scripts/deploy/Deployer.sol";
import { Chains } from "scripts/libraries/Chains.sol";
import { Config } from "scripts/libraries/Config.sol";
import { StateDiff } from "scripts/libraries/StateDiff.sol";
import { ChainAssertions } from "scripts/deploy/ChainAssertions.sol";
import { DeployUtils } from "scripts/libraries/DeployUtils.sol";
import { DeploySuperchain } from "scripts/deploy/DeploySuperchain.s.sol";
import { DeployImplementations } from "scripts/deploy/DeployImplementations.s.sol";
import { DeployAltDA } from "scripts/deploy/DeployAltDA.s.sol";
import { StandardConstants } from "scripts/deploy/StandardConstants.sol";

// Libraries
import { Types } from "scripts/libraries/Types.sol";
import { Duration } from "src/dispute/lib/LibUDT.sol";
import { GameType, Claim, GameTypes, Proposal, Hash } from "src/dispute/lib/Types.sol";

// Interfaces
import { IOPContractsManager } from "interfaces/L1/IOPContractsManager.sol";
import { IProxy } from "interfaces/universal/IProxy.sol";
import { IProxyAdmin } from "interfaces/universal/IProxyAdmin.sol";
import { ISuperchainConfig } from "interfaces/L1/ISuperchainConfig.sol";
import { IDisputeGameFactory } from "interfaces/dispute/IDisputeGameFactory.sol";
import { IDelayedWETH } from "interfaces/dispute/IDelayedWETH.sol";
import { IAnchorStateRegistry } from "interfaces/dispute/IAnchorStateRegistry.sol";
import { IMIPS } from "interfaces/cannon/IMIPS.sol";
import { IPreimageOracle } from "interfaces/cannon/IPreimageOracle.sol";
import { IProtocolVersions } from "interfaces/L1/IProtocolVersions.sol";
import { IL1CrossDomainMessenger } from "interfaces/L1/IL1CrossDomainMessenger.sol";
import { IETHLockbox } from "interfaces/L1/IETHLockbox.sol";
import { IOptimismPortal2 } from "interfaces/L1/IOptimismPortal2.sol";
import { IL1StandardBridge } from "interfaces/L1/IL1StandardBridge.sol";

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
                console.log("Deploying OP AltDA");

                DeployAltDA da = new DeployAltDA();
                DeployAltDA.Input memory dii = DeployAltDA.Input({
                    salt: _implSalt(),
                    proxyAdmin: IProxyAdmin(artifacts.mustGetAddress("ProxyAdmin")),
                    challengeContractOwner: cfg.finalSystemOwner(),
                    challengeWindow: cfg.daChallengeWindow(),
                    resolveWindow: cfg.daResolveWindow(),
                    bondSize: cfg.daBondSize(),
                    resolverRefundPercentage: cfg.daResolverRefundPercentage()
                });

                DeployAltDA.Output memory dio = da.run(dii);

                artifacts.save("DataAvailabilityChallengeProxy", address(dio.dataAvailabilityChallengeProxy));
                artifacts.save("DataAvailabilityChallengeImpl", address(dio.dataAvailabilityChallengeImpl));
            }
        }

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

        // Run the deployment script.
        DeploySuperchain.Output memory dso = ds.run(
            DeploySuperchain.Input({
                guardian: cfg.superchainConfigGuardian(),
                // TODO: when DeployAuthSystem is done, finalSystemOwner should be replaced with the Foundation Upgrades
                // Safe
                protocolVersionsOwner: cfg.finalSystemOwner(),
                superchainProxyAdminOwner: cfg.finalSystemOwner(),
                paused: false,
                recommendedProtocolVersion: bytes32(cfg.recommendedProtocolVersion()),
                requiredProtocolVersion: bytes32(cfg.requiredProtocolVersion())
            })
        );

        // Store the artifacts
        artifacts.save("SuperchainProxyAdmin", address(dso.superchainProxyAdmin));
        artifacts.save("SuperchainConfigProxy", address(dso.superchainConfigProxy));
        artifacts.save("SuperchainConfigImpl", address(dso.superchainConfigImpl));
        artifacts.save("ProtocolVersionsProxy", address(dso.protocolVersionsProxy));
        artifacts.save("ProtocolVersionsImpl", address(dso.protocolVersionsImpl));

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
        // TODO _isInterop is no longer being used in DeployImplementations, this might no longer be necessary
        require(_isInterop == cfg.useInterop(), "Deploy: Interop setting mismatch.");

        console.log("Deploying implementations");

        DeployImplementations di = new DeployImplementations();

        ISuperchainConfig superchainConfigProxy = ISuperchainConfig(artifacts.mustGetAddress("SuperchainConfigProxy"));
        IProxyAdmin superchainProxyAdmin = IProxyAdmin(EIP1967Helper.getAdmin(address(superchainConfigProxy)));

        DeployImplementations.Output memory dio = di.run(
            DeployImplementations.Input({
                withdrawalDelaySeconds: cfg.faultGameWithdrawalDelay(),
                minProposalSizeBytes: cfg.preimageOracleMinProposalSize(),
                challengePeriodSeconds: cfg.preimageOracleChallengePeriod(),
                proofMaturityDelaySeconds: cfg.proofMaturityDelaySeconds(),
                disputeGameFinalityDelaySeconds: cfg.disputeGameFinalityDelaySeconds(),
                mipsVersion: StandardConstants.MIPS_VERSION,
                l1ContractsRelease: "dev",
                protocolVersionsProxy: IProtocolVersions(artifacts.mustGetAddress("ProtocolVersionsProxy")),
                superchainConfigProxy: superchainConfigProxy,
                superchainProxyAdmin: superchainProxyAdmin,
                upgradeController: superchainProxyAdmin.owner(),
                challenger: cfg.l2OutputOracleChallenger()
            })
        );

        // Save the implementation addresses which are needed outside of this function or script.
        // When called in a fork test, this will overwrite the existing implementations.
        artifacts.save("MipsSingleton", address(dio.mipsSingleton));
        artifacts.save("OPContractsManager", address(dio.opcm));
        artifacts.save("DelayedWETHImpl", address(dio.delayedWETHImpl));
        artifacts.save("PreimageOracle", address(dio.preimageOracleSingleton));

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

        ChainAssertions.checkL1CrossDomainMessenger(IL1CrossDomainMessenger(impls.L1CrossDomainMessenger), vm, false);
        ChainAssertions.checkL1StandardBridgeImpl(IL1StandardBridge(payable(impls.L1StandardBridge)));
        ChainAssertions.checkL1ERC721Bridge({ _contracts: impls, _isProxy: false });
        ChainAssertions.checkOptimismPortal2({ _contracts: impls, _cfg: cfg, _isProxy: false });
        ChainAssertions.checkETHLockboxImpl(
            IETHLockbox(impls.ETHLockbox), IOptimismPortal2(payable(impls.OptimismPortal))
        );
        ChainAssertions.checkOptimismMintableERC20Factory({ _contracts: impls, _isProxy: false });
        ChainAssertions.checkDisputeGameFactory({ _contracts: impls, _expectedOwner: address(0), _isProxy: false });
        ChainAssertions.checkDelayedWETHImpl(IDelayedWETH(payable(impls.DelayedWETH)), cfg.faultGameWithdrawalDelay());
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
    }

    ////////////////////////////////////////////////////////////////
    //                Proxy Deployment Functions                  //
    ////////////////////////////////////////////////////////////////

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
}
