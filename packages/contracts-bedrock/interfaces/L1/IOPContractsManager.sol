// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

// Libraries
import { Claim, Duration, GameType, Proposal } from "src/dispute/lib/Types.sol";

// Interfaces
import { IBigStepper } from "interfaces/dispute/IBigStepper.sol";
import { IDelayedWETH } from "interfaces/dispute/IDelayedWETH.sol";
import { IAnchorStateRegistry } from "interfaces/dispute/IAnchorStateRegistry.sol";
import { IAddressManager } from "interfaces/legacy/IAddressManager.sol";
import { IProxyAdmin } from "interfaces/universal/IProxyAdmin.sol";
import { ISuperchainConfig } from "interfaces/L1/ISuperchainConfig.sol";
import { IDisputeGameFactory } from "interfaces/dispute/IDisputeGameFactory.sol";
import { IFaultDisputeGame } from "interfaces/dispute/IFaultDisputeGame.sol";
import { IPermissionedDisputeGame } from "interfaces/dispute/IPermissionedDisputeGame.sol";
import { IProtocolVersions } from "interfaces/L1/IProtocolVersions.sol";
import { IOptimismPortal2 } from "interfaces/L1/IOptimismPortal2.sol";
import { ISystemConfig } from "interfaces/L1/ISystemConfig.sol";
import { IL1CrossDomainMessenger } from "interfaces/L1/IL1CrossDomainMessenger.sol";
import { IL1ERC721Bridge } from "interfaces/L1/IL1ERC721Bridge.sol";
import { IL1StandardBridge } from "interfaces/L1/IL1StandardBridge.sol";
import { IOptimismMintableERC20Factory } from "interfaces/universal/IOptimismMintableERC20Factory.sol";
import { IETHLockbox } from "interfaces/L1/IETHLockbox.sol";
import { IOPContractsManagerStandardValidator } from "interfaces/L1/IOPContractsManagerStandardValidator.sol";

/// @custom:legacy
/// @title IOPContractsManagerContractsContainer
/// @notice Legacy stub interface. The v1 ContractsContainer has been removed.
interface IOPContractsManagerContractsContainer {
    error OPContractsManagerContractsContainer_DevFeatureInProd();

    function __constructor__(
        IOPContractsManager.Blueprints memory _blueprints,
        IOPContractsManager.Implementations memory _implementations,
        bytes32 _devFeatureBitmap
    )
        external;

    function blueprints() external view returns (IOPContractsManager.Blueprints memory);
    function implementations() external view returns (IOPContractsManager.Implementations memory);
    function devFeatureBitmap() external view returns (bytes32);
    function isDevFeatureEnabled(bytes32 _feature) external view returns (bool);
}

/// @custom:legacy
/// @title IOPContractsManagerGameTypeAdder
/// @notice Legacy stub interface. The v1 GameTypeAdder has been removed.
interface IOPContractsManagerGameTypeAdder {
    error OPContractsManagerGameTypeAdder_UnsupportedGameType();
    error OPContractsManagerGameTypeAdder_MixedGameTypes();

    event GameTypeAdded(
        uint256 indexed l2ChainId, GameType indexed gameType, address newDisputeGame, address oldDisputeGame
    );

    function __constructor__(IOPContractsManagerContractsContainer _contractsContainer) external;

    function addGameType(
        IOPContractsManager.AddGameInput[] memory _gameConfigs,
        address _superchainConfig
    )
        external
        returns (IOPContractsManager.AddGameOutput[] memory);

    function updatePrestate(
        IOPContractsManager.UpdatePrestateInput[] memory _prestateUpdateInputs,
        address _superchainConfig
    )
        external;

    function contractsContainer() external view returns (IOPContractsManagerContractsContainer);
}

/// @custom:legacy
/// @title IOPContractsManagerDeployer
/// @notice Legacy stub interface. The v1 Deployer has been removed.
interface IOPContractsManagerDeployer {
    event Deployed(uint256 indexed l2ChainId, address indexed deployer, bytes deployOutput);

    function __constructor__(IOPContractsManagerContractsContainer _contractsContainer) external;

    function deploy(
        IOPContractsManager.DeployInput memory _input,
        ISuperchainConfig _superchainConfig,
        address _deployer
    )
        external
        returns (IOPContractsManager.DeployOutput memory);

    function contractsContainer() external view returns (IOPContractsManagerContractsContainer);
}

/// @custom:legacy
/// @title IOPContractsManagerUpgrader
/// @notice Legacy stub interface. The v1 Upgrader has been removed.
interface IOPContractsManagerUpgrader {
    event Upgraded(uint256 indexed l2ChainId, address indexed systemConfig, address indexed upgrader);

    error OPContractsManagerUpgrader_SuperchainConfigNeedsUpgrade(uint256 index);

    error OPContractsManagerUpgrader_SuperchainConfigAlreadyUpToDate();

    function __constructor__(IOPContractsManagerContractsContainer _contractsContainer) external;

    function upgrade(IOPContractsManager.OpChainConfig[] memory _opChainConfigs) external;

    function upgradeSuperchainConfig(ISuperchainConfig _superchainConfig) external;

    function contractsContainer() external view returns (IOPContractsManagerContractsContainer);
}

/// @custom:legacy
/// @title IOPContractsManagerInteropMigrator
/// @notice Legacy stub interface. The v1 InteropMigrator has been removed.
interface IOPContractsManagerInteropMigrator {
    error OPContractsManagerInteropMigrator_ProxyAdminOwnerMismatch();
    error OPContractsManagerInteropMigrator_SuperchainConfigMismatch();
    error OPContractsManagerInteropMigrator_AbsolutePrestateMismatch();

    struct GameParameters {
        address proposer;
        address challenger;
        uint256 maxGameDepth;
        uint256 splitDepth;
        uint256 initBond;
        Duration clockExtension;
        Duration maxClockDuration;
    }

    struct MigrateInput {
        bool usePermissionlessGame;
        Proposal startingAnchorRoot;
        GameParameters gameParameters;
        IOPContractsManager.OpChainConfig[] opChainConfigs;
    }

    function __constructor__(IOPContractsManagerContractsContainer _contractsContainer) external;

    function migrate(MigrateInput calldata _input) external;
}

/// @custom:legacy
/// @title IOPContractsManager
/// @notice Legacy interface preserved for type compatibility. The OPCMv1 implementation has been
///         removed. Only struct/error definitions and function signatures remain so that external
///         code using IOPContractsManager types/selectors continues to compile. New code should
///         use IOPContractsManagerV2 instead.
interface IOPContractsManager {
    // -------- Structs --------

    struct Roles {
        address opChainProxyAdminOwner;
        address systemConfigOwner;
        address batcher;
        address unsafeBlockSigner;
        address proposer;
        address challenger;
    }

    struct DeployInput {
        Roles roles;
        uint32 basefeeScalar;
        uint32 blobBasefeeScalar;
        uint256 l2ChainId;
        bytes startingAnchorRoot;
        string saltMixer;
        uint64 gasLimit;
        GameType disputeGameType;
        Claim disputeAbsolutePrestate;
        uint256 disputeMaxGameDepth;
        uint256 disputeSplitDepth;
        Duration disputeClockExtension;
        Duration disputeMaxClockDuration;
        bool useCustomGasToken;
    }

    struct DeployOutput {
        IProxyAdmin opChainProxyAdmin;
        IAddressManager addressManager;
        IL1ERC721Bridge l1ERC721BridgeProxy;
        ISystemConfig systemConfigProxy;
        IOptimismMintableERC20Factory optimismMintableERC20FactoryProxy;
        IL1StandardBridge l1StandardBridgeProxy;
        IL1CrossDomainMessenger l1CrossDomainMessengerProxy;
        IETHLockbox ethLockboxProxy;
        IOptimismPortal2 optimismPortalProxy;
        IDisputeGameFactory disputeGameFactoryProxy;
        IAnchorStateRegistry anchorStateRegistryProxy;
        IFaultDisputeGame faultDisputeGame;
        IPermissionedDisputeGame permissionedDisputeGame;
        IDelayedWETH delayedWETHPermissionedGameProxy;
        IDelayedWETH delayedWETHPermissionlessGameProxy;
    }

    struct Blueprints {
        address addressManager;
        address proxy;
        address proxyAdmin;
        address l1ChugSplashProxy;
        address resolvedDelegateProxy;
    }

    struct Implementations {
        address superchainConfigImpl;
        address protocolVersionsImpl;
        address l1ERC721BridgeImpl;
        address optimismPortalImpl;
        address optimismPortalInteropImpl;
        address ethLockboxImpl;
        address systemConfigImpl;
        address optimismMintableERC20FactoryImpl;
        address l1CrossDomainMessengerImpl;
        address l1StandardBridgeImpl;
        address disputeGameFactoryImpl;
        address anchorStateRegistryImpl;
        address delayedWETHImpl;
        address mipsImpl;
        address faultDisputeGameImpl;
        address permissionedDisputeGameImpl;
        address superFaultDisputeGameImpl;
        address superPermissionedDisputeGameImpl;
    }

    struct OpChainConfig {
        ISystemConfig systemConfigProxy;
        Claim cannonPrestate;
        Claim cannonKonaPrestate;
    }

    struct UpdatePrestateInput {
        ISystemConfig systemConfigProxy;
        Claim cannonPrestate;
        Claim cannonKonaPrestate;
    }

    struct AddGameInput {
        string saltMixer;
        ISystemConfig systemConfig;
        IDelayedWETH delayedWETH;
        GameType disputeGameType;
        Claim disputeAbsolutePrestate;
        uint256 disputeMaxGameDepth;
        uint256 disputeSplitDepth;
        Duration disputeClockExtension;
        Duration disputeMaxClockDuration;
        uint256 initialBond;
        IBigStepper vm;
        bool permissioned;
    }

    struct AddGameOutput {
        IDelayedWETH delayedWETH;
        IFaultDisputeGame faultDisputeGame;
    }

    // -------- Errors --------

    error AddressNotFound(address who);
    error AddressHasNoCode(address who);
    error AlreadyReleased();
    error InvalidChainId();
    error InvalidRoleAddress(string role);
    error LatestReleaseNotSet();
    error InvalidStartingAnchorRoot();
    error OnlyDelegatecall();
    error InvalidGameConfigs();
    error SuperchainConfigMismatch(ISystemConfig systemConfig);
    error SuperchainProxyAdminMismatch();
    error PrestateNotSet();
    error PrestateRequired();
    error InvalidDevFeatureAccess(bytes32 devFeature);
    error OPContractsManager_V2Enabled();

    // -------- Function signatures (legacy, kept for selector references) --------

    function __constructor__(
        IOPContractsManagerGameTypeAdder _opcmGameTypeAdder,
        IOPContractsManagerDeployer _opcmDeployer,
        IOPContractsManagerUpgrader _opcmUpgrader,
        IOPContractsManagerInteropMigrator _opcmInteropMigrator,
        IOPContractsManagerStandardValidator _opcmStandardValidator,
        ISuperchainConfig _superchainConfig,
        IProtocolVersions _protocolVersions
    )
        external;

    function version() external pure returns (string memory);
    function superchainConfig() external view returns (ISuperchainConfig);
    function protocolVersions() external view returns (IProtocolVersions);

    function validate(
        IOPContractsManagerStandardValidator.ValidationInput calldata _input,
        bool _allowFailure
    )
        external
        view
        returns (string memory);

    function validateWithOverrides(
        IOPContractsManagerStandardValidator.ValidationInput calldata _input,
        bool _allowFailure,
        IOPContractsManagerStandardValidator.ValidationOverrides calldata _overrides
    )
        external
        view
        returns (string memory);

    function validate(
        IOPContractsManagerStandardValidator.ValidationInputDev calldata _input,
        bool _allowFailure
    )
        external
        view
        returns (string memory);

    function validateWithOverrides(
        IOPContractsManagerStandardValidator.ValidationInputDev calldata _input,
        bool _allowFailure,
        IOPContractsManagerStandardValidator.ValidationOverrides calldata _overrides
    )
        external
        view
        returns (string memory);

    function deploy(DeployInput calldata _input) external returns (DeployOutput memory);
    function upgrade(OpChainConfig[] memory _opChainConfigs) external;
    function upgradeSuperchainConfig(ISuperchainConfig _superchainConfig) external;
    function addGameType(AddGameInput[] memory _gameConfigs) external returns (AddGameOutput[] memory);
    function updatePrestate(UpdatePrestateInput[] memory _prestateUpdateInputs) external;
    function migrate(IOPContractsManagerInteropMigrator.MigrateInput calldata _input) external;
    function chainIdToBatchInboxAddress(uint256 _l2ChainId) external pure returns (address);
    function blueprints() external view returns (Blueprints memory);
    function implementations() external view returns (Implementations memory);
    function opcmDeployer() external view returns (IOPContractsManagerDeployer);
    function opcmUpgrader() external view returns (IOPContractsManagerUpgrader);
    function opcmGameTypeAdder() external view returns (IOPContractsManagerGameTypeAdder);
    function opcmInteropMigrator() external view returns (IOPContractsManagerInteropMigrator);
    function opcmStandardValidator() external view returns (IOPContractsManagerStandardValidator);
    function devFeatureBitmap() external view returns (bytes32);
    function isDevFeatureEnabled(bytes32 _feature) external view returns (bool);
}
