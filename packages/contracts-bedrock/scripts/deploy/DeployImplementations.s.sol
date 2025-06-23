// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Script } from "forge-std/Script.sol";

// Libraries
import { Chains } from "scripts/libraries/Chains.sol";
import { LibString } from "@solady/utils/LibString.sol";
import { GameType, GameTypes, Duration } from "src/dispute/lib/Types.sol";

// Interfaces
import { IResourceMetering } from "interfaces/L1/IResourceMetering.sol";
import { ISuperchainConfig } from "interfaces/L1/ISuperchainConfig.sol";
import { IProtocolVersions } from "interfaces/L1/IProtocolVersions.sol";
import { IDelayedWETH } from "interfaces/dispute/IDelayedWETH.sol";
import { IPreimageOracle } from "interfaces/cannon/IPreimageOracle.sol";
import { IMIPS } from "interfaces/cannon/IMIPS.sol";
import { IMIPS2 } from "interfaces/cannon/IMIPS2.sol";
import { IDisputeGameFactory } from "interfaces/dispute/IDisputeGameFactory.sol";
import { IAnchorStateRegistry } from "interfaces/dispute/IAnchorStateRegistry.sol";
import { IFaultDisputeGame } from "interfaces/dispute/IFaultDisputeGame.sol";
import { IPermissionedDisputeGame } from "interfaces/dispute/IPermissionedDisputeGame.sol";
import { ISuperFaultDisputeGame } from "interfaces/dispute/ISuperFaultDisputeGame.sol";
import { ISuperPermissionedDisputeGame } from "interfaces/dispute/ISuperPermissionedDisputeGame.sol";
import {
    IOPContractsManager,
    IOPContractsManagerGameTypeAdder,
    IOPContractsManagerDeployer,
    IOPContractsManagerUpgrader,
    IOPContractsManagerContractsContainer,
    IOPContractsManagerInteropMigrator
} from "interfaces/L1/IOPContractsManager.sol";
import { IOptimismPortal2 as IOptimismPortal } from "interfaces/L1/IOptimismPortal2.sol";
import { IETHLockbox } from "interfaces/L1/IETHLockbox.sol";
import { ISystemConfig } from "interfaces/L1/ISystemConfig.sol";
import { IL1CrossDomainMessenger } from "interfaces/L1/IL1CrossDomainMessenger.sol";
import { IL1ERC721Bridge } from "interfaces/L1/IL1ERC721Bridge.sol";
import { IL1StandardBridge } from "interfaces/L1/IL1StandardBridge.sol";
import { IOptimismMintableERC20Factory } from "interfaces/universal/IOptimismMintableERC20Factory.sol";
import { IProxyAdmin } from "interfaces/universal/IProxyAdmin.sol";
import { DeployUtils } from "scripts/libraries/DeployUtils.sol";
import { Solarray } from "scripts/libraries/Solarray.sol";

contract DeployImplementations is Script {
    struct Input {
        uint256 withdrawalDelaySeconds;
        uint256 minProposalSizeBytes;
        uint256 challengePeriodSeconds;
        uint256 proofMaturityDelaySeconds;
        uint256 disputeGameFinalityDelaySeconds;
        uint256 mipsVersion;
        // This is used in opcm to signal which version of the L1 smart contracts is deployed.
        // It takes the format of `op-contracts/v*.*.*`.
        string l1ContractsRelease;
        // Outputs from DeploySuperchain.s.sol.
        ISuperchainConfig superchainConfigProxy;
        IProtocolVersions protocolVersionsProxy;
        IProxyAdmin superchainProxyAdmin;
        address upgradeController;
        // Game implementation template parameters
        uint256 gameMaxGameDepth;
        uint256 gameSplitDepth;
        uint256 gameClockExtension;
        uint256 gameMaxClockDuration;
    }

    struct Output {
        IOPContractsManager opcm;
        IOPContractsManagerContractsContainer opcmContractsContainer;
        IOPContractsManagerGameTypeAdder opcmGameTypeAdder;
        IOPContractsManagerDeployer opcmDeployer;
        IOPContractsManagerUpgrader opcmUpgrader;
        IOPContractsManagerInteropMigrator opcmInteropMigrator;
        IDelayedWETH delayedWETHImpl;
        IOptimismPortal optimismPortalImpl;
        IETHLockbox ethLockboxImpl;
        IPreimageOracle preimageOracleSingleton;
        IMIPS mipsSingleton;
        ISystemConfig systemConfigImpl;
        IL1CrossDomainMessenger l1CrossDomainMessengerImpl;
        IL1ERC721Bridge l1ERC721BridgeImpl;
        IL1StandardBridge l1StandardBridgeImpl;
        IOptimismMintableERC20Factory optimismMintableERC20FactoryImpl;
        IDisputeGameFactory disputeGameFactoryImpl;
        IAnchorStateRegistry anchorStateRegistryImpl;
        ISuperchainConfig superchainConfigImpl;
        IProtocolVersions protocolVersionsImpl;
        IFaultDisputeGame faultDisputeGameImpl;
        IPermissionedDisputeGame permissionedDisputeGameImpl;
        ISuperFaultDisputeGame superFaultDisputeGameImpl;
        ISuperPermissionedDisputeGame superPermissionedDisputeGameImpl;
    }

    bytes32 internal _salt = DeployUtils.DEFAULT_SALT;

    // -------- Core Deployment Methods --------

    function run(Input memory _input) public returns (Output memory output_) {
        assertValidInput(_input);

        // Deploy the implementations.
        deploySuperchainConfigImpl(output_);
        deployProtocolVersionsImpl(output_);
        deploySystemConfigImpl(output_);
        deployL1CrossDomainMessengerImpl(output_);
        deployL1ERC721BridgeImpl(output_);
        deployL1StandardBridgeImpl(output_);
        deployOptimismMintableERC20FactoryImpl(output_);
        deployOptimismPortalImpl(_input, output_);
        deployETHLockboxImpl(output_);
        deployDelayedWETHImpl(_input, output_);
        deployPreimageOracleSingleton(_input, output_);
        deployMipsSingleton(_input, output_);
        deployDisputeGameFactoryImpl(output_);
        deployAnchorStateRegistryImpl(_input, output_);
        deployFaultDisputeGameImpl(_input, output_);
        deployPermissionedDisputeGameImpl(_input, output_);
        deploySuperFaultDisputeGameImpl(_input, output_);
        deploySuperPermissionedDisputeGameImpl(_input, output_);

        // Create blueprints for OPCM.
        IOPContractsManager.Blueprints memory blueprints = createBlueprints();

        // Deploy the OP Contracts Manager with the new implementations set.
        deployOPContractsManager(_input, output_, blueprints);

        assertValidOutput(_input, output_);
    }

    // -------- Deployment Steps --------

    // --- OP Contracts Manager ---

    function createOPCMContract(
        Input memory _input,
        Output memory _output,
        IOPContractsManager.Blueprints memory _blueprints,
        string memory _l1ContractsRelease
    )
        private
        returns (IOPContractsManager opcm_)
    {
        IOPContractsManager.Implementations memory implementations = IOPContractsManager.Implementations({
            superchainConfigImpl: address(_output.superchainConfigImpl),
            protocolVersionsImpl: address(_output.protocolVersionsImpl),
            l1ERC721BridgeImpl: address(_output.l1ERC721BridgeImpl),
            optimismPortalImpl: address(_output.optimismPortalImpl),
            ethLockboxImpl: address(_output.ethLockboxImpl),
            systemConfigImpl: address(_output.systemConfigImpl),
            optimismMintableERC20FactoryImpl: address(_output.optimismMintableERC20FactoryImpl),
            l1CrossDomainMessengerImpl: address(_output.l1CrossDomainMessengerImpl),
            l1StandardBridgeImpl: address(_output.l1StandardBridgeImpl),
            disputeGameFactoryImpl: address(_output.disputeGameFactoryImpl),
            anchorStateRegistryImpl: address(_output.anchorStateRegistryImpl),
            delayedWETHImpl: address(_output.delayedWETHImpl),
            mipsImpl: address(_output.mipsSingleton),
            faultDisputeGameImpl: address(_output.faultDisputeGameImpl),
            permissionedDisputeGameImpl: address(_output.permissionedDisputeGameImpl),
            superFaultDisputeGameImpl: address(_output.superFaultDisputeGameImpl),
            superPermissionedDisputeGameImpl: address(_output.superPermissionedDisputeGameImpl)
        });

        deployOPCMBPImplsContainer(_output, _blueprints, implementations);
        deployOPCMGameTypeAdder(_output);
        deployOPCMDeployer(_input, _output);
        deployOPCMUpgrader(_output);
        deployOPCMInteropMigrator(_output);

        // Semgrep rule will fail because the arguments are encoded inside of a separate function.
        opcm_ = IOPContractsManager(
            // nosemgrep: sol-safety-deployutils-args
            DeployUtils.createDeterministic({
                _name: "OPContractsManager",
                _args: encodeOPCMConstructor(_l1ContractsRelease, _input, _output),
                _salt: _salt
            })
        );

        vm.label(address(opcm_), "OPContractsManager");
        _output.opcm = opcm_;
    }

    /// @notice Encodes the constructor of the OPContractsManager contract. Used to avoid stack too
    ///         deep errors inside of the createOPCMContract function.
    /// @param _l1ContractsRelease The release of the L1 contracts.
    /// @param _input The deployment input parameters.
    /// @param _output The deployment output parameters.
    /// @return encoded_ The encoded constructor.
    function encodeOPCMConstructor(
        string memory _l1ContractsRelease,
        Input memory _input,
        Output memory _output
    )
        private
        pure
        returns (bytes memory encoded_)
    {
        encoded_ = DeployUtils.encodeConstructor(
            abi.encodeCall(
                IOPContractsManager.__constructor__,
                (
                    _output.opcmGameTypeAdder,
                    _output.opcmDeployer,
                    _output.opcmUpgrader,
                    _output.opcmInteropMigrator,
                    _input.superchainConfigProxy,
                    _input.protocolVersionsProxy,
                    _input.superchainProxyAdmin,
                    _l1ContractsRelease,
                    _input.upgradeController
                )
            )
        );
    }

    function createBlueprints() private returns (IOPContractsManager.Blueprints memory blueprints) {
        // OPCM uses Blueprints and stores references to their addresses because it deploys these as implementations for
        // OP Stack Chains
        /// Blueprints prevent these child contracts from bloating the runtime bytecode size of OPCM
        // First we deploy the blueprints for the singletons deployed by OPCM.
        /// TODO: snevins - Unintuitive for why we use start broadcast here and it's not in the
        /// createDeterministicBlueprint function
        // forgefmt: disable-start
        vm.startBroadcast(msg.sender);
        address checkAddress;
        (blueprints.addressManager, checkAddress) = DeployUtils.createDeterministicBlueprint(vm.getCode("AddressManager"), _salt);
        require(checkAddress == address(0), "OPCM-10");
        (blueprints.proxy, checkAddress) = DeployUtils.createDeterministicBlueprint(vm.getCode("Proxy"), _salt);
        require(checkAddress == address(0), "OPCM-20");
        (blueprints.proxyAdmin, checkAddress) = DeployUtils.createDeterministicBlueprint(vm.getCode("ProxyAdmin"), _salt);
        require(checkAddress == address(0), "OPCM-30");
        (blueprints.l1ChugSplashProxy, checkAddress) = DeployUtils.createDeterministicBlueprint(vm.getCode("L1ChugSplashProxy"), _salt);
        require(checkAddress == address(0), "OPCM-40");
        (blueprints.resolvedDelegateProxy, checkAddress) = DeployUtils.createDeterministicBlueprint(vm.getCode("ResolvedDelegateProxy"), _salt);
        require(checkAddress == address(0), "OPCM-50");
        // forgefmt: disable-end
        vm.stopBroadcast();
    }

    function deployOPContractsManager(
        Input memory _input,
        Output memory _output,
        IOPContractsManager.Blueprints memory _blueprints
    )
        private
    {
        string memory l1ContractsRelease = _input.l1ContractsRelease;

        IOPContractsManager opcm = createOPCMContract(_input, _output, _blueprints, l1ContractsRelease);

        vm.label(address(opcm), "OPContractsManager");
        _output.opcm = opcm;
    }

    // --- Core Contracts ---

    function deploySuperchainConfigImpl(Output memory _output) private {
        ISuperchainConfig impl = ISuperchainConfig(
            DeployUtils.createDeterministic({
                _name: "SuperchainConfig",
                _args: DeployUtils.encodeConstructor(abi.encodeCall(ISuperchainConfig.__constructor__, ())),
                _salt: _salt
            })
        );
        vm.label(address(impl), "SuperchainConfigImpl");
        _output.superchainConfigImpl = impl;
    }

    function deployProtocolVersionsImpl(Output memory _output) private {
        IProtocolVersions impl = IProtocolVersions(
            DeployUtils.createDeterministic({
                _name: "ProtocolVersions",
                _args: DeployUtils.encodeConstructor(abi.encodeCall(IProtocolVersions.__constructor__, ())),
                _salt: _salt
            })
        );
        vm.label(address(impl), "ProtocolVersionsImpl");
        _output.protocolVersionsImpl = impl;
    }

    function deploySystemConfigImpl(Output memory _output) private {
        ISystemConfig impl = ISystemConfig(
            DeployUtils.createDeterministic({
                _name: "SystemConfig",
                _args: DeployUtils.encodeConstructor(abi.encodeCall(ISystemConfig.__constructor__, ())),
                _salt: _salt
            })
        );
        vm.label(address(impl), "SystemConfigImpl");
        _output.systemConfigImpl = impl;
    }

    function deployL1CrossDomainMessengerImpl(Output memory _output) private {
        IL1CrossDomainMessenger impl = IL1CrossDomainMessenger(
            DeployUtils.createDeterministic({
                _name: "L1CrossDomainMessenger",
                _args: DeployUtils.encodeConstructor(abi.encodeCall(IL1CrossDomainMessenger.__constructor__, ())),
                _salt: _salt
            })
        );
        vm.label(address(impl), "L1CrossDomainMessengerImpl");
        _output.l1CrossDomainMessengerImpl = impl;
    }

    function deployL1ERC721BridgeImpl(Output memory _output) private {
        IL1ERC721Bridge impl = IL1ERC721Bridge(
            DeployUtils.createDeterministic({
                _name: "L1ERC721Bridge",
                _args: DeployUtils.encodeConstructor(abi.encodeCall(IL1ERC721Bridge.__constructor__, ())),
                _salt: _salt
            })
        );
        vm.label(address(impl), "L1ERC721BridgeImpl");
        _output.l1ERC721BridgeImpl = impl;
    }

    function deployL1StandardBridgeImpl(Output memory _output) private {
        IL1StandardBridge impl = IL1StandardBridge(
            DeployUtils.createDeterministic({
                _name: "L1StandardBridge",
                _args: DeployUtils.encodeConstructor(abi.encodeCall(IL1StandardBridge.__constructor__, ())),
                _salt: _salt
            })
        );
        vm.label(address(impl), "L1StandardBridgeImpl");
        _output.l1StandardBridgeImpl = impl;
    }

    function deployOptimismMintableERC20FactoryImpl(Output memory _output) private {
        IOptimismMintableERC20Factory impl = IOptimismMintableERC20Factory(
            DeployUtils.createDeterministic({
                _name: "OptimismMintableERC20Factory",
                _args: DeployUtils.encodeConstructor(abi.encodeCall(IOptimismMintableERC20Factory.__constructor__, ())),
                _salt: _salt
            })
        );
        vm.label(address(impl), "OptimismMintableERC20FactoryImpl");
        _output.optimismMintableERC20FactoryImpl = impl;
    }

    function deployETHLockboxImpl(Output memory _output) private {
        IETHLockbox impl = IETHLockbox(
            DeployUtils.createDeterministic({
                _name: "ETHLockbox",
                _args: DeployUtils.encodeConstructor(abi.encodeCall(IETHLockbox.__constructor__, ())),
                _salt: _salt
            })
        );
        vm.label(address(impl), "ETHLockboxImpl");
        _output.ethLockboxImpl = impl;
    }

    // --- Fault Proofs Contracts ---

    // The fault proofs contracts are configured as follows:
    // | Contract                | Proxied | Deployment                        | MCP Ready  |
    // |-------------------------|---------|-----------------------------------|------------|
    // | DisputeGameFactory      | Yes     | Bespoke                           | Yes        |
    // | AnchorStateRegistry     | Yes     | Bespoke                           | Yes         |
    // | FaultDisputeGame        | No      | Bespoke                           | No         | Not yet supported by OPCM
    // | PermissionedDisputeGame | No      | Bespoke                           | No         |
    // | DelayedWETH             | Yes     | Two bespoke (one per DisputeGame) | Yes *️⃣     |
    // | PreimageOracle          | No      | Shared                            | N/A        |
    // | MIPS                    | No      | Shared                            | N/A        |
    // | OptimismPortal2         | Yes     | Shared                            | Yes *️⃣     |
    //
    // - *️⃣ These contracts have immutable values which are intended to be constant for all contracts within a
    //   Superchain, and are therefore MCP ready for any chain using the Standard Configuration.
    //
    // This script only deploys the shared contracts. The bespoke contracts are deployed by
    // `DeployOPChain.s.sol`. When the shared contracts are proxied, the contracts deployed here are
    // "implementations", and when shared contracts are not proxied, they are "singletons". So
    // here we deploy:
    //
    //   - DisputeGameFactory (implementation)
    //   - AnchorStateRegistry (implementation)
    //   - OptimismPortal2 (implementation)
    //   - DelayedWETH (implementation)
    //   - PreimageOracle (singleton)
    //   - MIPS (singleton)
    //
    // For contracts which are not MCP ready neither the Proxy nor the implementation can be shared, therefore they
    // are deployed by `DeployOpChain.s.sol`.
    // These are:
    // - FaultDisputeGame (not proxied)
    // - PermissionedDisputeGame (not proxied)
    // - DelayedWeth (proxies only)
    // - OptimismPortal2 (proxies only)

    function deployOptimismPortalImpl(Input memory _input, Output memory _output) private {
        uint256 proofMaturityDelaySeconds = _input.proofMaturityDelaySeconds;
        IOptimismPortal impl = IOptimismPortal(
            DeployUtils.createDeterministic({
                _name: "OptimismPortal2",
                _args: DeployUtils.encodeConstructor(
                    abi.encodeCall(IOptimismPortal.__constructor__, (proofMaturityDelaySeconds))
                ),
                _salt: _salt
            })
        );
        vm.label(address(impl), "OptimismPortalImpl");
        _output.optimismPortalImpl = impl;
    }

    function deployDelayedWETHImpl(Input memory _input, Output memory _output) private {
        uint256 withdrawalDelaySeconds = _input.withdrawalDelaySeconds;
        IDelayedWETH impl = IDelayedWETH(
            DeployUtils.createDeterministic({
                _name: "DelayedWETH",
                _args: DeployUtils.encodeConstructor(abi.encodeCall(IDelayedWETH.__constructor__, (withdrawalDelaySeconds))),
                _salt: _salt
            })
        );
        vm.label(address(impl), "DelayedWETHImpl");
        _output.delayedWETHImpl = impl;
    }

    function deployPreimageOracleSingleton(Input memory _input, Output memory _output) private {
        uint256 minProposalSizeBytes = _input.minProposalSizeBytes;
        uint256 challengePeriodSeconds = _input.challengePeriodSeconds;
        IPreimageOracle singleton = IPreimageOracle(
            DeployUtils.createDeterministic({
                _name: "PreimageOracle",
                _args: DeployUtils.encodeConstructor(
                    abi.encodeCall(IPreimageOracle.__constructor__, (minProposalSizeBytes, challengePeriodSeconds))
                ),
                _salt: _salt
            })
        );
        vm.label(address(singleton), "PreimageOracleSingleton");
        _output.preimageOracleSingleton = singleton;
    }

    function deployMipsSingleton(Input memory _input, Output memory _output) private {
        uint256 mipsVersion = _input.mipsVersion;
        IPreimageOracle preimageOracle = IPreimageOracle(address(_output.preimageOracleSingleton));

        // We want to ensure that the OPCM for upgrade 13 is deployed with Mips32 on production networks.
        if (mipsVersion < 2) {
            if (block.chainid == Chains.Mainnet || block.chainid == Chains.Sepolia) {
                revert("DeployImplementations: Only Mips64 should be deployed on Mainnet or Sepolia");
            }
        }

        IMIPS singleton = IMIPS(
            DeployUtils.createDeterministic({
                _name: "MIPS64",
                _args: DeployUtils.encodeConstructor(abi.encodeCall(IMIPS2.__constructor__, (preimageOracle, mipsVersion))),
                _salt: DeployUtils.DEFAULT_SALT
            })
        );
        vm.label(address(singleton), "MIPSSingleton");
        _output.mipsSingleton = singleton;
    }

    function deployDisputeGameFactoryImpl(Output memory _output) private {
        IDisputeGameFactory impl = IDisputeGameFactory(
            DeployUtils.createDeterministic({
                _name: "DisputeGameFactory",
                _args: DeployUtils.encodeConstructor(abi.encodeCall(IDisputeGameFactory.__constructor__, ())),
                _salt: _salt
            })
        );
        vm.label(address(impl), "DisputeGameFactoryImpl");
        _output.disputeGameFactoryImpl = impl;
    }

    function deployAnchorStateRegistryImpl(Input memory _input, Output memory _output) private {
        uint256 disputeGameFinalityDelaySeconds = _input.disputeGameFinalityDelaySeconds;
        IAnchorStateRegistry impl = IAnchorStateRegistry(
            DeployUtils.createDeterministic({
                _name: "AnchorStateRegistry",
                _args: DeployUtils.encodeConstructor(
                    abi.encodeCall(IAnchorStateRegistry.__constructor__, (disputeGameFinalityDelaySeconds))
                ),
                _salt: _salt
            })
        );
        vm.label(address(impl), "AnchorStateRegistryImpl");
        _output.anchorStateRegistryImpl = impl;
    }

    function deployFaultDisputeGameImpl(Input memory _input, Output memory _output) private {
        // Create minimal constructor params for the implementation template
        IFaultDisputeGame.GameConstructorParams memory params = IFaultDisputeGame.GameConstructorParams({
            gameType: GameTypes.CANNON,
            maxGameDepth: _input.gameMaxGameDepth,
            splitDepth: _input.gameSplitDepth,
            clockExtension: Duration.wrap(uint64(_input.gameClockExtension)),
            maxClockDuration: Duration.wrap(uint64(_input.gameMaxClockDuration)),
            weth: IDelayedWETH(payable(address(0))), // Will be set during clone initialization
            l2ChainId: 0 // Will be set during clone initialization
         });

        IFaultDisputeGame impl = IFaultDisputeGame(
            DeployUtils.createDeterministic({
                _name: "FaultDisputeGame",
                _args: DeployUtils.encodeConstructor(abi.encodeCall(IFaultDisputeGame.__constructor__, (params))),
                _salt: _salt
            })
        );
        vm.label(address(impl), "FaultDisputeGameImpl");
        _output.faultDisputeGameImpl = impl;
    }

    function deployPermissionedDisputeGameImpl(Input memory _input, Output memory _output) private {
        // Create minimal constructor params for the implementation template
        IFaultDisputeGame.GameConstructorParams memory params = IFaultDisputeGame.GameConstructorParams({
            gameType: GameTypes.PERMISSIONED_CANNON,
            maxGameDepth: _input.gameMaxGameDepth,
            splitDepth: _input.gameSplitDepth,
            clockExtension: Duration.wrap(uint64(_input.gameClockExtension)),
            maxClockDuration: Duration.wrap(uint64(_input.gameMaxClockDuration)),
            weth: IDelayedWETH(payable(address(0))), // Will be set during clone initialization
            l2ChainId: 0 // Will be set during clone initialization
         });

        IPermissionedDisputeGame impl = IPermissionedDisputeGame(
            DeployUtils.createDeterministic({
                _name: "PermissionedDisputeGame",
                _args: DeployUtils.encodeConstructor(
                    abi.encodeCall(IPermissionedDisputeGame.__constructor__, (params, address(0), address(0)))
                ),
                _salt: _salt
            })
        );
        vm.label(address(impl), "PermissionedDisputeGameImpl");
        _output.permissionedDisputeGameImpl = impl;
    }

    function deploySuperFaultDisputeGameImpl(Input memory _input, Output memory _output) private {
        // Create minimal constructor params for the implementation template
        ISuperFaultDisputeGame.GameConstructorParams memory params = ISuperFaultDisputeGame.GameConstructorParams({
            gameType: GameTypes.SUPER_CANNON,
            maxGameDepth: _input.gameMaxGameDepth,
            splitDepth: _input.gameSplitDepth,
            clockExtension: Duration.wrap(uint64(_input.gameClockExtension)),
            maxClockDuration: Duration.wrap(uint64(_input.gameMaxClockDuration)),
            weth: IDelayedWETH(payable(address(0))), // Will be set during clone initialization
            l2ChainId: 0 // Will be set during clone initialization
         });

        ISuperFaultDisputeGame impl = ISuperFaultDisputeGame(
            DeployUtils.createDeterministic({
                _name: "SuperFaultDisputeGame",
                _args: DeployUtils.encodeConstructor(abi.encodeCall(ISuperFaultDisputeGame.__constructor__, (params))),
                _salt: _salt
            })
        );
        vm.label(address(impl), "SuperFaultDisputeGameImpl");
        _output.superFaultDisputeGameImpl = impl;
    }

    function deploySuperPermissionedDisputeGameImpl(Input memory _input, Output memory _output) private {
        // Create minimal constructor params for the implementation template
        ISuperFaultDisputeGame.GameConstructorParams memory params = ISuperFaultDisputeGame.GameConstructorParams({
            gameType: GameTypes.SUPER_PERMISSIONED_CANNON,
            maxGameDepth: _input.gameMaxGameDepth,
            splitDepth: _input.gameSplitDepth,
            clockExtension: Duration.wrap(uint64(_input.gameClockExtension)),
            maxClockDuration: Duration.wrap(uint64(_input.gameMaxClockDuration)),
            weth: IDelayedWETH(payable(address(0))), // Will be set during clone initialization
            l2ChainId: 0 // Will be set during clone initialization
         });

        ISuperPermissionedDisputeGame impl = ISuperPermissionedDisputeGame(
            DeployUtils.createDeterministic({
                _name: "SuperPermissionedDisputeGame",
                _args: DeployUtils.encodeConstructor(
                    abi.encodeCall(ISuperPermissionedDisputeGame.__constructor__, (params, address(0), address(0)))
                ),
                _salt: _salt
            })
        );
        vm.label(address(impl), "SuperPermissionedDisputeGameImpl");
        _output.superPermissionedDisputeGameImpl = impl;
    }

    function deployOPCMBPImplsContainer(
        Output memory _output,
        IOPContractsManager.Blueprints memory _blueprints,
        IOPContractsManager.Implementations memory _implementations
    )
        private
    {
        IOPContractsManagerContractsContainer impl = IOPContractsManagerContractsContainer(
            DeployUtils.createDeterministic({
                _name: "OPContractsManager.sol:OPContractsManagerContractsContainer",
                _args: DeployUtils.encodeConstructor(
                    abi.encodeCall(IOPContractsManagerContractsContainer.__constructor__, (_blueprints, _implementations))
                ),
                _salt: _salt
            })
        );
        vm.label(address(impl), "OPContractsManagerBPImplsContainerImpl");
        _output.opcmContractsContainer = impl;
    }

    function deployOPCMGameTypeAdder(Output memory _output) private {
        IOPContractsManagerGameTypeAdder impl = IOPContractsManagerGameTypeAdder(
            DeployUtils.createDeterministic({
                _name: "OPContractsManager.sol:OPContractsManagerGameTypeAdder",
                _args: DeployUtils.encodeConstructor(
                    abi.encodeCall(IOPContractsManagerGameTypeAdder.__constructor__, (_output.opcmContractsContainer))
                ),
                _salt: _salt
            })
        );
        vm.label(address(impl), "OPContractsManagerGameTypeAdderImpl");
        _output.opcmGameTypeAdder = impl;
    }

    function deployOPCMDeployer(Input memory, Output memory _output) private {
        IOPContractsManagerDeployer impl = IOPContractsManagerDeployer(
            DeployUtils.createDeterministic({
                _name: "OPContractsManager.sol:OPContractsManagerDeployer",
                _args: DeployUtils.encodeConstructor(
                    abi.encodeCall(IOPContractsManagerDeployer.__constructor__, (_output.opcmContractsContainer))
                ),
                _salt: _salt
            })
        );
        vm.label(address(impl), "OPContractsManagerDeployerImpl");
        _output.opcmDeployer = impl;
    }

    function deployOPCMUpgrader(Output memory _output) private {
        IOPContractsManagerUpgrader impl = IOPContractsManagerUpgrader(
            DeployUtils.createDeterministic({
                _name: "OPContractsManager.sol:OPContractsManagerUpgrader",
                _args: DeployUtils.encodeConstructor(
                    abi.encodeCall(IOPContractsManagerUpgrader.__constructor__, (_output.opcmContractsContainer))
                ),
                _salt: _salt
            })
        );
        vm.label(address(impl), "OPContractsManagerUpgraderImpl");
        _output.opcmUpgrader = impl;
    }

    function deployOPCMInteropMigrator(Output memory _output) private {
        IOPContractsManagerInteropMigrator impl = IOPContractsManagerInteropMigrator(
            DeployUtils.createDeterministic({
                _name: "OPContractsManager.sol:OPContractsManagerInteropMigrator",
                _args: DeployUtils.encodeConstructor(
                    abi.encodeCall(IOPContractsManagerInteropMigrator.__constructor__, (_output.opcmContractsContainer))
                ),
                _salt: _salt
            })
        );
        vm.label(address(impl), "OPContractsManagerInteropMigratorImpl");
        _output.opcmInteropMigrator = impl;
    }

    function assertValidInput(Input memory _input) private pure {
        require(_input.withdrawalDelaySeconds != 0, "DeployImplementations: withdrawalDelaySeconds not set");
        require(_input.minProposalSizeBytes != 0, "DeployImplementations: minProposalSizeBytes not set");
        require(_input.challengePeriodSeconds != 0, "DeployImplementations: challengePeriodSeconds not set");
        require(
            _input.challengePeriodSeconds <= type(uint64).max, "DeployImplementations: challengePeriodSeconds too large"
        );
        require(_input.proofMaturityDelaySeconds != 0, "DeployImplementations: proofMaturityDelaySeconds not set");
        require(
            _input.disputeGameFinalityDelaySeconds != 0,
            "DeployImplementations: disputeGameFinalityDelaySeconds not set"
        );
        require(_input.mipsVersion != 0, "DeployImplementations: mipsVersion not set");
        require(!LibString.eq(_input.l1ContractsRelease, ""), "DeployImplementations: l1ContractsRelease not set");
        require(
            address(_input.superchainConfigProxy) != address(0), "DeployImplementations: superchainConfigProxy not set"
        );
        require(
            address(_input.protocolVersionsProxy) != address(0), "DeployImplementations: protocolVersionsProxy not set"
        );
        require(
            address(_input.superchainProxyAdmin) != address(0), "DeployImplementations: superchainProxyAdmin not set"
        );
        require(address(_input.upgradeController) != address(0), "DeployImplementations: upgradeController not set");
        require(_input.gameMaxGameDepth != 0, "DeployImplementations: gameMaxGameDepth not set");
        require(_input.gameSplitDepth != 0, "DeployImplementations: gameSplitDepth not set");
        require(_input.gameClockExtension != 0, "DeployImplementations: gameClockExtension not set");
        require(_input.gameMaxClockDuration != 0, "DeployImplementations: gameMaxClockDuration not set");
    }

    function assertValidOutput(Input memory _input, Output memory _output) private view {
        // With more addresses, we'd get a stack too deep error if we tried to do this inline as a
        // single call to `Solarray.addresses`. So we split it into three calls.
        address[] memory addrs1 = Solarray.addresses(
            address(_output.opcm),
            address(_output.optimismPortalImpl),
            address(_output.delayedWETHImpl),
            address(_output.preimageOracleSingleton),
            address(_output.mipsSingleton),
            address(_output.superchainConfigImpl),
            address(_output.protocolVersionsImpl)
        );

        address[] memory addrs2 = Solarray.addresses(
            address(_output.systemConfigImpl),
            address(_output.l1CrossDomainMessengerImpl),
            address(_output.l1ERC721BridgeImpl),
            address(_output.l1StandardBridgeImpl),
            address(_output.optimismMintableERC20FactoryImpl),
            address(_output.disputeGameFactoryImpl),
            address(_output.anchorStateRegistryImpl),
            address(_output.ethLockboxImpl)
        );

        address[] memory addrs3 = Solarray.addresses(
            address(_output.faultDisputeGameImpl),
            address(_output.permissionedDisputeGameImpl),
            address(_output.superFaultDisputeGameImpl),
            address(_output.superPermissionedDisputeGameImpl)
        );

        DeployUtils.assertValidContractAddresses(Solarray.extend(Solarray.extend(addrs1, addrs2), addrs3));

        assertValidDelayedWETHImpl(_input, _output);
        assertValidDisputeGameFactoryImpl(_input, _output);
        assertValidAnchorStateRegistryImpl(_input, _output);
        assertValidL1CrossDomainMessengerImpl(_input, _output);
        assertValidL1ERC721BridgeImpl(_input, _output);
        assertValidL1StandardBridgeImpl(_input, _output);
        assertValidMipsSingleton(_input, _output);
        assertValidOpcm(_input, _output);
        assertValidOptimismMintableERC20FactoryImpl(_input, _output);
        assertValidOptimismPortalImpl(_input, _output);
        assertValidETHLockboxImpl(_input, _output);
        assertValidPreimageOracleSingleton(_input, _output);
        assertValidSystemConfigImpl(_input, _output);
        assertValidFaultDisputeGameImpl(_input, _output);
        assertValidPermissionedDisputeGameImpl(_input, _output);
        assertValidSuperFaultDisputeGameImpl(_input, _output);
        assertValidSuperPermissionedDisputeGameImpl(_input, _output);
    }

    function assertValidOpcm(Input memory _input, Output memory _output) private view {
        IOPContractsManager impl = IOPContractsManager(address(_output.opcm));
        require(address(impl.superchainConfig()) == address(_input.superchainConfigProxy), "OPCMI-10");
        require(address(impl.protocolVersions()) == address(_input.protocolVersionsProxy), "OPCMI-20");
        require(impl.upgradeController() == _input.upgradeController, "OPCMI-30");
    }

    function assertValidOptimismPortalImpl(Input memory, Output memory _output) private view {
        IOptimismPortal portal = _output.optimismPortalImpl;

        DeployUtils.assertInitialized({ _contractAddress: address(portal), _isProxy: false, _slot: 0, _offset: 0 });

        require(address(portal.anchorStateRegistry()) == address(0), "PORTAL-10");
        require(address(portal.systemConfig()) == address(0), "PORTAL-20");
        require(portal.l2Sender() == address(0), "PORTAL-30");

        // This slot is the custom gas token _balance and this check ensures
        // that it stays unset for forwards compatibility with custom gas token.
        require(vm.load(address(portal), bytes32(uint256(61))) == bytes32(0), "PORTAL-40");

        require(address(portal.ethLockbox()) == address(0), "PORTAL-50");
    }

    function assertValidETHLockboxImpl(Input memory, Output memory _output) private view {
        IETHLockbox lockbox = _output.ethLockboxImpl;

        DeployUtils.assertInitialized({ _contractAddress: address(lockbox), _isProxy: false, _slot: 0, _offset: 0 });

        require(address(lockbox.systemConfig()) == address(0), "ELB-10");
        require(lockbox.authorizedPortals(_output.optimismPortalImpl) == false, "ELB-20");
    }

    function assertValidDelayedWETHImpl(Input memory _input, Output memory _output) private view {
        IDelayedWETH delayedWETH = _output.delayedWETHImpl;

        DeployUtils.assertInitialized({ _contractAddress: address(delayedWETH), _isProxy: false, _slot: 0, _offset: 0 });

        require(delayedWETH.delay() == _input.withdrawalDelaySeconds, "DW-10");
        require(delayedWETH.systemConfig() == ISystemConfig(address(0)), "DW-20");
    }

    function assertValidPreimageOracleSingleton(Input memory _input, Output memory _output) private view {
        IPreimageOracle oracle = _output.preimageOracleSingleton;

        require(oracle.minProposalSize() == _input.minProposalSizeBytes, "PO-10");
        require(oracle.challengePeriod() == _input.challengePeriodSeconds, "PO-20");
    }

    function assertValidMipsSingleton(Input memory, Output memory _output) private view {
        IMIPS mips = _output.mipsSingleton;
        require(address(mips.oracle()) == address(_output.preimageOracleSingleton), "MIPS-10");
    }

    function assertValidSystemConfigImpl(Input memory, Output memory _output) private view {
        ISystemConfig systemConfig = _output.systemConfigImpl;

        DeployUtils.assertInitialized({ _contractAddress: address(systemConfig), _isProxy: false, _slot: 0, _offset: 0 });

        require(systemConfig.owner() == address(0), "SYSCON-10");
        require(systemConfig.overhead() == 0, "SYSCON-20");
        require(systemConfig.scalar() == 0, "SYSCON-30");
        require(systemConfig.basefeeScalar() == 0, "SYSCON-40");
        require(systemConfig.blobbasefeeScalar() == 0, "SYSCON-50");
        require(systemConfig.batcherHash() == bytes32(0), "SYSCON-60");
        require(systemConfig.gasLimit() == 0, "SYSCON-70");
        require(systemConfig.unsafeBlockSigner() == address(0), "SYSCON-80");

        IResourceMetering.ResourceConfig memory resourceConfig = systemConfig.resourceConfig();
        require(resourceConfig.maxResourceLimit == 0, "SYSCON-90");
        require(resourceConfig.elasticityMultiplier == 0, "SYSCON-100");
        require(resourceConfig.baseFeeMaxChangeDenominator == 0, "SYSCON-110");
        require(resourceConfig.systemTxMaxGas == 0, "SYSCON-120");
        require(resourceConfig.minimumBaseFee == 0, "SYSCON-130");
        require(resourceConfig.maximumBaseFee == 0, "SYSCON-140");

        require(systemConfig.startBlock() == type(uint256).max, "SYSCON-150");
        require(systemConfig.batchInbox() == address(0), "SYSCON-160");
        require(systemConfig.l1CrossDomainMessenger() == address(0), "SYSCON-170");
        require(systemConfig.l1ERC721Bridge() == address(0), "SYSCON-180");
        require(systemConfig.l1StandardBridge() == address(0), "SYSCON-190");
        require(systemConfig.optimismPortal() == address(0), "SYSCON-200");
        require(systemConfig.optimismMintableERC20Factory() == address(0), "SYSCON-210");
    }

    function assertValidL1CrossDomainMessengerImpl(Input memory, Output memory _output) private view {
        IL1CrossDomainMessenger messenger = _output.l1CrossDomainMessengerImpl;

        DeployUtils.assertInitialized({ _contractAddress: address(messenger), _isProxy: false, _slot: 0, _offset: 20 });

        require(address(messenger.OTHER_MESSENGER()) == address(0), "L1xDM-10");
        require(address(messenger.otherMessenger()) == address(0), "L1xDM-20");
        require(address(messenger.PORTAL()) == address(0), "L1xDM-30");
        require(address(messenger.portal()) == address(0), "L1xDM-40");
        require(address(messenger.systemConfig()) == address(0), "L1xDM-50");

        bytes32 xdmSenderSlot = vm.load(address(messenger), bytes32(uint256(204)));
        require(address(uint160(uint256(xdmSenderSlot))) == address(0), "L1xDM-60");
    }

    function assertValidL1ERC721BridgeImpl(Input memory, Output memory _output) private view {
        IL1ERC721Bridge bridge = _output.l1ERC721BridgeImpl;

        DeployUtils.assertInitialized({ _contractAddress: address(bridge), _isProxy: false, _slot: 0, _offset: 0 });

        require(address(bridge.OTHER_BRIDGE()) == address(0), "L721B-10");
        require(address(bridge.otherBridge()) == address(0), "L721B-20");
        require(address(bridge.MESSENGER()) == address(0), "L721B-30");
        require(address(bridge.messenger()) == address(0), "L721B-40");
        require(address(bridge.systemConfig()) == address(0), "L721B-50");
    }

    function assertValidL1StandardBridgeImpl(Input memory, Output memory _output) private view {
        IL1StandardBridge bridge = _output.l1StandardBridgeImpl;

        DeployUtils.assertInitialized({ _contractAddress: address(bridge), _isProxy: false, _slot: 0, _offset: 0 });

        require(address(bridge.MESSENGER()) == address(0), "L1SB-10");
        require(address(bridge.messenger()) == address(0), "L1SB-20");
        require(address(bridge.OTHER_BRIDGE()) == address(0), "L1SB-30");
        require(address(bridge.otherBridge()) == address(0), "L1SB-40");
        require(address(bridge.systemConfig()) == address(0), "L1SB-50");
    }

    function assertValidOptimismMintableERC20FactoryImpl(Input memory, Output memory _output) private view {
        IOptimismMintableERC20Factory factory = _output.optimismMintableERC20FactoryImpl;

        DeployUtils.assertInitialized({ _contractAddress: address(factory), _isProxy: false, _slot: 0, _offset: 0 });

        require(address(factory.BRIDGE()) == address(0), "MERC20F-10");
        require(address(factory.bridge()) == address(0), "MERC20F-20");
    }

    function assertValidDisputeGameFactoryImpl(Input memory, Output memory _output) private view {
        IDisputeGameFactory factory = _output.disputeGameFactoryImpl;

        DeployUtils.assertInitialized({ _contractAddress: address(factory), _isProxy: false, _slot: 0, _offset: 0 });

        require(address(factory.owner()) == address(0), "DG-10");
    }

    function assertValidAnchorStateRegistryImpl(Input memory, Output memory _output) private view {
        IAnchorStateRegistry registry = _output.anchorStateRegistryImpl;

        DeployUtils.assertInitialized({ _contractAddress: address(registry), _isProxy: false, _slot: 0, _offset: 0 });
    }

    function assertValidFaultDisputeGameImpl(Input memory, Output memory _output) private view {
        IFaultDisputeGame game = _output.faultDisputeGameImpl;
        require(address(game).code.length > 0, "FDG-10");
    }

    function assertValidPermissionedDisputeGameImpl(Input memory, Output memory _output) private view {
        IPermissionedDisputeGame game = _output.permissionedDisputeGameImpl;
        require(address(game).code.length > 0, "PDG-10");
    }

    function assertValidSuperFaultDisputeGameImpl(Input memory, Output memory _output) private view {
        ISuperFaultDisputeGame game = _output.superFaultDisputeGameImpl;
        require(address(game).code.length > 0, "SFDG-10");
    }

    function assertValidSuperPermissionedDisputeGameImpl(Input memory, Output memory _output) private view {
        ISuperPermissionedDisputeGame game = _output.superPermissionedDisputeGameImpl;
        require(address(game).code.length > 0, "SPDG-10");
    }
}
