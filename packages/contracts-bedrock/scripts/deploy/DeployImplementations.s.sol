// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Script } from "forge-std/Script.sol";

// Libraries
import { Chains } from "scripts/libraries/Chains.sol";
import { LibString } from "@solady/utils/LibString.sol";
import { Types } from "scripts/libraries/Types.sol";

// Interfaces
import { ISuperchainConfig } from "interfaces/L1/ISuperchainConfig.sol";
import { IProtocolVersions } from "interfaces/L1/IProtocolVersions.sol";
import { IDelayedWETH } from "interfaces/dispute/IDelayedWETH.sol";
import { IPreimageOracle } from "interfaces/cannon/IPreimageOracle.sol";
import { IMIPS } from "interfaces/cannon/IMIPS.sol";
import { IMIPS2 } from "interfaces/cannon/IMIPS2.sol";
import { IDisputeGameFactory } from "interfaces/dispute/IDisputeGameFactory.sol";
import { IAnchorStateRegistry } from "interfaces/dispute/IAnchorStateRegistry.sol";
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
import { DeployOPChainInput } from "scripts/deploy/DeployOPChain.s.sol";

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

        // Deploy the OP Contracts Manager with the new implementations set.
        deployOPContractsManager(_input, output_);

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
            mipsImpl: address(_output.mipsSingleton)
        });

        deployOPCMBPImplsContainer(_output, _blueprints, implementations);
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
                    _output.opcmStandardValidator,
                    _input.superchainConfigProxy,
                    _input.protocolVersionsProxy,
                    _input.superchainProxyAdmin,
                    _l1ContractsRelease,
                    _input.upgradeController
                )
            )
        );
    }

    function deployOPContractsManager(Input memory _input, Output memory _output) private {
        string memory l1ContractsRelease = _input.l1ContractsRelease;

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

        IOPContractsManager opcm = createOPCMContract(_input, _output, blueprints, l1ContractsRelease);

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
                            _input.upgradeController, // Proxy admin owner
                            _input.challenger,
                            _input.withdrawalDelaySeconds
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
    }

    function assertValidOutput(Input memory _input, Output memory _output) private view {
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

        DeployUtils.assertValidContractAddresses(Solarray.extend(addrs1, addrs2));

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
        assertValidL1ERC721BridgeImpl(_input, _output);
        ChainAssertions.checkL1StandardBridgeImpl(_output.l1StandardBridgeImpl);
        ChainAssertions.checkMIPS(_output.mipsSingleton, _output.preimageOracleSingleton);
        assertValidOpcm(_input, _output);
        assertValidOptimismMintableERC20FactoryImpl(_input, _output);
        assertValidOptimismPortalImpl(_input, _output);
        ChainAssertions.checkETHLockboxImpl(_output.ethLockboxImpl, _output.optimismPortalImpl);
        // We can use DeployOPChainInput(address(0)) here because no method will be called on _doi when isProxy is false
        ChainAssertions.checkSystemConfig(impls, DeployOPChainInput(address(0)), false);
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

    function assertValidL1ERC721BridgeImpl(Input memory, Output memory _output) private view {
        IL1ERC721Bridge bridge = _output.l1ERC721BridgeImpl;

        DeployUtils.assertInitialized({ _contractAddress: address(bridge), _isProxy: false, _slot: 0, _offset: 0 });

        require(address(bridge.OTHER_BRIDGE()) == address(0), "L721B-10");
        require(address(bridge.otherBridge()) == address(0), "L721B-20");
        require(address(bridge.MESSENGER()) == address(0), "L721B-30");
        require(address(bridge.messenger()) == address(0), "L721B-40");
        require(address(bridge.systemConfig()) == address(0), "L721B-50");
    }

    function assertValidOptimismMintableERC20FactoryImpl(Input memory, Output memory _output) private view {
        IOptimismMintableERC20Factory factory = _output.optimismMintableERC20FactoryImpl;

        DeployUtils.assertInitialized({ _contractAddress: address(factory), _isProxy: false, _slot: 0, _offset: 0 });

        require(address(factory.BRIDGE()) == address(0), "MERC20F-10");
        require(address(factory.bridge()) == address(0), "MERC20F-20");
    }
}
