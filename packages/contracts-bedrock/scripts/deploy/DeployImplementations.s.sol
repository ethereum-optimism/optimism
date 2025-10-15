// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Script } from "forge-std/Script.sol";

// Libraries
import { Chains } from "scripts/libraries/Chains.sol";
import { Types } from "scripts/libraries/Types.sol";

// Interfaces
import { ISuperchainConfig } from "interfaces/L1/ISuperchainConfig.sol";
import { IProtocolVersions } from "interfaces/L1/IProtocolVersions.sol";
import { IDelayedWETH } from "interfaces/dispute/IDelayedWETH.sol";
import { IPreimageOracle } from "interfaces/cannon/IPreimageOracle.sol";
import { IMIPS64 } from "interfaces/cannon/IMIPS64.sol";
import { IDisputeGameFactory } from "interfaces/dispute/IDisputeGameFactory.sol";
import { IAnchorStateRegistry } from "interfaces/dispute/IAnchorStateRegistry.sol";
import { IFaultDisputeGameV2 } from "interfaces/dispute/v2/IFaultDisputeGameV2.sol";
import { IPermissionedDisputeGameV2 } from "interfaces/dispute/v2/IPermissionedDisputeGameV2.sol";
import { GameTypes, Duration } from "src/dispute/lib/Types.sol";
import {
    IOPContractsManager,
    IOPContractsManagerGameTypeAdder,
    IOPContractsManagerDeployer,
    IOPContractsManagerUpgrader,
    IOPContractsManagerContractsContainer,
    IOPContractsManagerInteropMigrator,
    IOPContractsManagerStandardValidator
} from "interfaces/L1/IOPContractsManager.sol";
import { IOptimismPortal2 as IOptimismPortal } from "interfaces/L1/IOptimismPortal2.sol";
import { IOptimismPortalInterop } from "interfaces/L1/IOptimismPortalInterop.sol";
import { IETHLockbox } from "interfaces/L1/IETHLockbox.sol";
import { ISystemConfig } from "interfaces/L1/ISystemConfig.sol";
import { IL1CrossDomainMessenger } from "interfaces/L1/IL1CrossDomainMessenger.sol";
import { IL1ERC721Bridge } from "interfaces/L1/IL1ERC721Bridge.sol";
import { IL1StandardBridge } from "interfaces/L1/IL1StandardBridge.sol";
import { IOptimismMintableERC20Factory } from "interfaces/universal/IOptimismMintableERC20Factory.sol";
import { IProxyAdmin } from "interfaces/universal/IProxyAdmin.sol";
import { IOPContractsManagerStandardValidator } from "interfaces/L1/IOPContractsManagerStandardValidator.sol";
import { DeployUtils } from "scripts/libraries/DeployUtils.sol";
import { Solarray } from "scripts/libraries/Solarray.sol";
import { ChainAssertions } from "scripts/deploy/ChainAssertions.sol";
import { DevFeatures } from "src/libraries/DevFeatures.sol";

contract DeployImplementations is Script {
    struct Input {
        uint256 withdrawalDelaySeconds;
        uint256 minProposalSizeBytes;
        uint256 challengePeriodSeconds;
        uint256 proofMaturityDelaySeconds;
        uint256 disputeGameFinalityDelaySeconds;
        uint256 mipsVersion;
        bytes32 devFeatureBitmap;
        // V2 Dispute Game parameters
        uint256 faultGameV2MaxGameDepth;
        uint256 faultGameV2SplitDepth;
        uint256 faultGameV2ClockExtension;
        uint256 faultGameV2MaxClockDuration;
        // Outputs from DeploySuperchain.s.sol.
        ISuperchainConfig superchainConfigProxy;
        IProtocolVersions protocolVersionsProxy;
        IProxyAdmin superchainProxyAdmin;
        address l1ProxyAdminOwner;
        address challenger;
    }

    struct Output {
        IOPContractsManager opcm;
        IOPContractsManagerContractsContainer opcmContractsContainer;
        IOPContractsManagerGameTypeAdder opcmGameTypeAdder;
        IOPContractsManagerDeployer opcmDeployer;
        IOPContractsManagerUpgrader opcmUpgrader;
        IOPContractsManagerInteropMigrator opcmInteropMigrator;
        IOPContractsManagerStandardValidator opcmStandardValidator;
        IDelayedWETH delayedWETHImpl;
        IOptimismPortal optimismPortalImpl;
        IOptimismPortalInterop optimismPortalInteropImpl;
        IETHLockbox ethLockboxImpl;
        IPreimageOracle preimageOracleSingleton;
        IMIPS64 mipsSingleton;
        ISystemConfig systemConfigImpl;
        IL1CrossDomainMessenger l1CrossDomainMessengerImpl;
        IL1ERC721Bridge l1ERC721BridgeImpl;
        IL1StandardBridge l1StandardBridgeImpl;
        IOptimismMintableERC20Factory optimismMintableERC20FactoryImpl;
        IDisputeGameFactory disputeGameFactoryImpl;
        IAnchorStateRegistry anchorStateRegistryImpl;
        ISuperchainConfig superchainConfigImpl;
        IProtocolVersions protocolVersionsImpl;
        IFaultDisputeGameV2 faultDisputeGameV2Impl;
        IPermissionedDisputeGameV2 permissionedDisputeGameV2Impl;
    }

    bytes32 internal _salt = DeployUtils.DEFAULT_SALT;

    // -------- Core Deployment Methods --------

    function runWithBytes(bytes memory _input) public returns (bytes memory) {
        Input memory input = abi.decode(_input, (Input));
        Output memory output = run(input);
        return abi.encode(output);
    }

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
        deployOptimismPortalInteropImpl(_input, output_);
        deployETHLockboxImpl(output_);
        deployDelayedWETHImpl(_input, output_);
        deployPreimageOracleSingleton(_input, output_);
        deployMipsSingleton(_input, output_);
        deployDisputeGameFactoryImpl(output_);
        deployAnchorStateRegistryImpl(_input, output_);
        if (DevFeatures.isDevFeatureEnabled(_input.devFeatureBitmap, DevFeatures.DEPLOY_V2_DISPUTE_GAMES)) {
            deployFaultDisputeGameV2Impl(_input, output_);
            deployPermissionedDisputeGameV2Impl(_input, output_);
        }

        // Deploy the OP Contracts Manager with the new implementations set.
        deployOPContractsManager(_input, output_);

        assertValidOutput(_input, output_);
    }

    // -------- Deployment Steps --------

    // --- OP Contracts Manager ---

    function createOPCMContract(
        Input memory _input,
        Output memory _output,
        IOPContractsManager.Blueprints memory _blueprints
    )
        private
        returns (IOPContractsManager opcm_)
    {
        IOPContractsManager.Implementations memory implementations = IOPContractsManager.Implementations({
            superchainConfigImpl: address(_output.superchainConfigImpl),
            protocolVersionsImpl: address(_output.protocolVersionsImpl),
            l1ERC721BridgeImpl: address(_output.l1ERC721BridgeImpl),
            optimismPortalImpl: address(_output.optimismPortalImpl),
            optimismPortalInteropImpl: address(_output.optimismPortalInteropImpl),
            ethLockboxImpl: address(_output.ethLockboxImpl),
            systemConfigImpl: address(_output.systemConfigImpl),
            optimismMintableERC20FactoryImpl: address(_output.optimismMintableERC20FactoryImpl),
            l1CrossDomainMessengerImpl: address(_output.l1CrossDomainMessengerImpl),
            l1StandardBridgeImpl: address(_output.l1StandardBridgeImpl),
            disputeGameFactoryImpl: address(_output.disputeGameFactoryImpl),
            anchorStateRegistryImpl: address(_output.anchorStateRegistryImpl),
            delayedWETHImpl: address(_output.delayedWETHImpl),
            mipsImpl: address(_output.mipsSingleton)
        });

        deployOPCMBPImplsContainer(_input, _output, _blueprints, implementations);
        deployOPCMGameTypeAdder(_output);
        deployOPCMDeployer(_input, _output);
        deployOPCMUpgrader(_output);
        deployOPCMInteropMigrator(_output);
        deployOPCMStandardValidator(_input, _output, implementations);

        // Semgrep rule will fail because the arguments are encoded inside of a separate function.
        opcm_ = IOPContractsManager(
            // nosemgrep: sol-safety-deployutils-args
            DeployUtils.createDeterministic({
                _name: "OPContractsManager",
                _args: encodeOPCMConstructor(_input, _output),
                _salt: _salt
            })
        );

        vm.label(address(opcm_), "OPContractsManager");
        _output.opcm = opcm_;
    }

    /// @notice Encodes the constructor of the OPContractsManager contract. Used to avoid stack too
    ///         deep errors inside of the createOPCMContract function.
    /// @param _input The deployment input parameters.
    /// @param _output The deployment output parameters.
    /// @return encoded_ The encoded constructor.
    function encodeOPCMConstructor(
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
                    _output.opcmStandardValidator,
                    _input.superchainConfigProxy,
                    _input.protocolVersionsProxy,
                    _input.superchainProxyAdmin
                )
            )
        );
    }

    function deployOPContractsManager(Input memory _input, Output memory _output) private {
        // First we deploy the blueprints for the singletons deployed by OPCM.
        // forgefmt: disable-start
        IOPContractsManager.Blueprints memory blueprints;
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
        // The max initcode/runtimecode size is 48KB/24KB.
        // But for Blueprint, the initcode is stored as runtime code, that's why it's necessary to split into 2 parts.
        (blueprints.permissionedDisputeGame1, blueprints.permissionedDisputeGame2) = DeployUtils.createDeterministicBlueprint(vm.getCode("PermissionedDisputeGame"), _salt);
        (blueprints.permissionlessDisputeGame1, blueprints.permissionlessDisputeGame2) = DeployUtils.createDeterministicBlueprint(vm.getCode("FaultDisputeGame"), _salt);
        (blueprints.superPermissionedDisputeGame1, blueprints.superPermissionedDisputeGame2) = DeployUtils.createDeterministicBlueprint(vm.getCode("SuperPermissionedDisputeGame"), _salt);
        (blueprints.superPermissionlessDisputeGame1, blueprints.superPermissionlessDisputeGame2) = DeployUtils.createDeterministicBlueprint(vm.getCode("SuperFaultDisputeGame"), _salt);
        // forgefmt: disable-end
        vm.stopBroadcast();

        IOPContractsManager opcm = createOPCMContract(_input, _output, blueprints);

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

    function deployOptimismPortalInteropImpl(Input memory _input, Output memory _output) private {
        uint256 proofMaturityDelaySeconds = _input.proofMaturityDelaySeconds;
        IOptimismPortalInterop impl = IOptimismPortalInterop(
            DeployUtils.createDeterministic({
                _name: "OptimismPortalInterop",
                _args: DeployUtils.encodeConstructor(
                    abi.encodeCall(IOptimismPortalInterop.__constructor__, (proofMaturityDelaySeconds))
                ),
                _salt: _salt
            })
        );
        vm.label(address(impl), "OptimismPortalInteropImpl");
        _output.optimismPortalInteropImpl = impl;
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

        IMIPS64 singleton = IMIPS64(
            DeployUtils.createDeterministic({
                _name: "MIPS64",
                _args: DeployUtils.encodeConstructor(abi.encodeCall(IMIPS64.__constructor__, (preimageOracle, mipsVersion))),
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

    function deployFaultDisputeGameV2Impl(Input memory _input, Output memory _output) private {
        IFaultDisputeGameV2.GameConstructorParams memory params;
        params.gameType = GameTypes.CANNON;
        params.maxGameDepth = _input.faultGameV2MaxGameDepth;
        params.splitDepth = _input.faultGameV2SplitDepth;
        params.clockExtension = Duration.wrap(uint64(_input.faultGameV2ClockExtension));
        params.maxClockDuration = Duration.wrap(uint64(_input.faultGameV2MaxClockDuration));

        IFaultDisputeGameV2 impl = IFaultDisputeGameV2(
            DeployUtils.createDeterministic({
                _name: "FaultDisputeGameV2",
                _args: DeployUtils.encodeConstructor(abi.encodeCall(IFaultDisputeGameV2.__constructor__, (params))),
                _salt: _salt
            })
        );
        vm.label(address(impl), "FaultDisputeGameV2Impl");
        _output.faultDisputeGameV2Impl = impl;
    }

    function deployPermissionedDisputeGameV2Impl(Input memory _input, Output memory _output) private {
        IFaultDisputeGameV2.GameConstructorParams memory params;
        params.gameType = GameTypes.PERMISSIONED_CANNON;
        params.maxGameDepth = _input.faultGameV2MaxGameDepth;
        params.splitDepth = _input.faultGameV2SplitDepth;
        params.clockExtension = Duration.wrap(uint64(_input.faultGameV2ClockExtension));
        params.maxClockDuration = Duration.wrap(uint64(_input.faultGameV2MaxClockDuration));

        IPermissionedDisputeGameV2 impl = IPermissionedDisputeGameV2(
            DeployUtils.createDeterministic({
                _name: "PermissionedDisputeGameV2",
                _args: DeployUtils.encodeConstructor(abi.encodeCall(IPermissionedDisputeGameV2.__constructor__, (params))),
                _salt: _salt
            })
        );
        vm.label(address(impl), "PermissionedDisputeGameV2Impl");
        _output.permissionedDisputeGameV2Impl = impl;
    }

    function deployOPCMBPImplsContainer(
        Input memory _input,
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
                    abi.encodeCall(
                        IOPContractsManagerContractsContainer.__constructor__,
                        (_blueprints, _implementations, _input.devFeatureBitmap)
                    )
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

    function deployOPCMStandardValidator(
        Input memory _input,
        Output memory _output,
        IOPContractsManager.Implementations memory _implementations
    )
        private
    {
        IOPContractsManagerStandardValidator.Implementations memory opcmImplementations;
        opcmImplementations.l1ERC721BridgeImpl = _implementations.l1ERC721BridgeImpl;
        opcmImplementations.optimismPortalImpl = _implementations.optimismPortalImpl;
        opcmImplementations.optimismPortalInteropImpl = _implementations.optimismPortalInteropImpl;
        opcmImplementations.ethLockboxImpl = _implementations.ethLockboxImpl;
        opcmImplementations.systemConfigImpl = _implementations.systemConfigImpl;
        opcmImplementations.optimismMintableERC20FactoryImpl = _implementations.optimismMintableERC20FactoryImpl;
        opcmImplementations.l1CrossDomainMessengerImpl = _implementations.l1CrossDomainMessengerImpl;
        opcmImplementations.l1StandardBridgeImpl = _implementations.l1StandardBridgeImpl;
        opcmImplementations.disputeGameFactoryImpl = _implementations.disputeGameFactoryImpl;
        opcmImplementations.anchorStateRegistryImpl = _implementations.anchorStateRegistryImpl;
        opcmImplementations.delayedWETHImpl = _implementations.delayedWETHImpl;
        opcmImplementations.mipsImpl = _implementations.mipsImpl;

        IOPContractsManagerStandardValidator impl = IOPContractsManagerStandardValidator(
            DeployUtils.createDeterministic({
                _name: "OPContractsManagerStandardValidator.sol:OPContractsManagerStandardValidator",
                _args: DeployUtils.encodeConstructor(
                    abi.encodeCall(
                        IOPContractsManagerStandardValidator.__constructor__,
                        (
                            opcmImplementations,
                            _input.superchainConfigProxy,
                            _input.l1ProxyAdminOwner,
                            _input.challenger,
                            _input.withdrawalDelaySeconds,
                            _input.devFeatureBitmap
                        )
                    )
                ),
                _salt: _salt
            })
        );
        vm.label(address(impl), "OPContractsManagerStandardValidatorImpl");
        _output.opcmStandardValidator = impl;
    }

    function assertValidInput(Input memory _input) private pure {
        if (DevFeatures.isDevFeatureEnabled(_input.devFeatureBitmap, DevFeatures.DEPLOY_V2_DISPUTE_GAMES)) {
            // Validate V2 game depth parameters are sensible
            require(
                _input.faultGameV2MaxGameDepth > 0 && _input.faultGameV2MaxGameDepth <= 125,
                "DeployImplementations: faultGameV2MaxGameDepth out of valid range (1-125)"
            );
            // V2 contract requires splitDepth >= 2 and splitDepth + 1 < maxGameDepth
            require(
                _input.faultGameV2SplitDepth >= 2 && _input.faultGameV2SplitDepth + 1 < _input.faultGameV2MaxGameDepth,
                "DeployImplementations: faultGameV2SplitDepth must be >= 2 and splitDepth + 1 < maxGameDepth"
            );

            // Validate V2 clock parameters fit in uint64 before deployment
            require(
                _input.faultGameV2ClockExtension <= type(uint64).max,
                "DeployImplementations: faultGameV2ClockExtension too large for uint64"
            );
            require(
                _input.faultGameV2MaxClockDuration <= type(uint64).max,
                "DeployImplementations: faultGameV2MaxClockDuration too large for uint64"
            );
            require(
                _input.faultGameV2MaxClockDuration >= _input.faultGameV2ClockExtension,
                "DeployImplementations: maxClockDuration must be >= clockExtension"
            );
            require(
                _input.faultGameV2ClockExtension > 0, "DeployImplementations: faultGameV2ClockExtension must be > 0"
            );
        }
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
        require(
            address(_input.superchainConfigProxy) != address(0), "DeployImplementations: superchainConfigProxy not set"
        );
        require(
            address(_input.protocolVersionsProxy) != address(0), "DeployImplementations: protocolVersionsProxy not set"
        );
        require(
            address(_input.superchainProxyAdmin) != address(0), "DeployImplementations: superchainProxyAdmin not set"
        );
        require(address(_input.l1ProxyAdminOwner) != address(0), "DeployImplementations: L1ProxyAdminOwner not set");
    }

    function assertValidOutput(Input memory _input, Output memory _output) private {
        // With 12 addresses, we'd get a stack too deep error if we tried to do this inline as a
        // single call to `Solarray.addresses`. So we split it into two calls.
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

        // Only include V2 contracts in validation if they were deployed
        if (DevFeatures.isDevFeatureEnabled(_input.devFeatureBitmap, DevFeatures.DEPLOY_V2_DISPUTE_GAMES)) {
            address[] memory v2Addrs = Solarray.addresses(
                address(_output.faultDisputeGameV2Impl), address(_output.permissionedDisputeGameV2Impl)
            );
            addrs2 = Solarray.extend(addrs2, v2Addrs);
        }

        DeployUtils.assertValidContractAddresses(Solarray.extend(addrs1, addrs2));

        // Validate V2 contracts not deployed when flag is disabled
        if (!DevFeatures.isDevFeatureEnabled(_input.devFeatureBitmap, DevFeatures.DEPLOY_V2_DISPUTE_GAMES)) {
            require(
                address(_output.faultDisputeGameV2Impl) == address(0),
                "DeployImplementations: V2 flag disabled but FaultDisputeGameV2 was deployed"
            );
            require(
                address(_output.permissionedDisputeGameV2Impl) == address(0),
                "DeployImplementations: V2 flag disabled but PermissionedDisputeGameV2 was deployed"
            );
        }

        Types.ContractSet memory impls = ChainAssertions.dioToContractSet(_output);

        ChainAssertions.checkDelayedWETHImpl(_output.delayedWETHImpl, _input.withdrawalDelaySeconds);
        ChainAssertions.checkDisputeGameFactory(_output.disputeGameFactoryImpl, address(0), address(0), false);
        DeployUtils.assertInitialized({
            _contractAddress: address(_output.anchorStateRegistryImpl),
            _isProxy: false,
            _slot: 0,
            _offset: 0
        });
        ChainAssertions.checkL1CrossDomainMessenger(IL1CrossDomainMessenger(impls.L1CrossDomainMessenger), vm, false);
        ChainAssertions.checkL1ERC721BridgeImpl(_output.l1ERC721BridgeImpl);
        ChainAssertions.checkL1StandardBridgeImpl(_output.l1StandardBridgeImpl);
        ChainAssertions.checkMIPS(_output.mipsSingleton, _output.preimageOracleSingleton);

        Types.ContractSet memory proxies;
        proxies.SuperchainConfig = address(_input.superchainConfigProxy);
        proxies.ProtocolVersions = address(_input.protocolVersionsProxy);
        ChainAssertions.checkOPContractsManager({
            _impls: impls,
            _proxies: proxies,
            _opcm: IOPContractsManager(address(_output.opcm)),
            _mips: IMIPS64(address(_output.mipsSingleton)),
            _superchainProxyAdmin: _input.superchainProxyAdmin
        });

        ChainAssertions.checkOptimismMintableERC20FactoryImpl(_output.optimismMintableERC20FactoryImpl);
        ChainAssertions.checkOptimismPortal2({
            _contracts: impls,
            _superchainConfig: ISuperchainConfig(address(_input.superchainConfigProxy)),
            _opChainProxyAdminOwner: address(0),
            _isProxy: false
        });
        ChainAssertions.checkETHLockboxImpl(_output.ethLockboxImpl, _output.optimismPortalImpl);
        ChainAssertions.checkSystemConfigImpls(impls);
        ChainAssertions.checkAnchorStateRegistryProxy(IAnchorStateRegistry(impls.AnchorStateRegistry), false);
    }
}
