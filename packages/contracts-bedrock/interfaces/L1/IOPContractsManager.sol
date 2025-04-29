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
import { ISuperchainConfig } from "interfaces/L1/ISuperchainConfig.sol";
import { IL1CrossDomainMessenger } from "interfaces/L1/IL1CrossDomainMessenger.sol";
import { IL1ERC721Bridge } from "interfaces/L1/IL1ERC721Bridge.sol";
import { IL1StandardBridge } from "interfaces/L1/IL1StandardBridge.sol";
import { IOptimismMintableERC20Factory } from "interfaces/universal/IOptimismMintableERC20Factory.sol";
import { IETHLockbox } from "interfaces/L1/IETHLockbox.sol";

interface IOPContractsManagerContractsContainer {
    function __constructor__(
        IOPContractsManager.Blueprints memory _blueprints,
        IOPContractsManager.Implementations memory _implementations
    )
        external;

    function blueprints() external view returns (IOPContractsManager.Blueprints memory);
    function implementations() external view returns (IOPContractsManager.Implementations memory);
}

interface IOPContractsManagerGameTypeAdder {
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
        IOPContractsManager.OpChainConfig[] memory _prestateUpdateInputs,
        address _superchainConfig
    )
        external;

    function contractsContainer() external view returns (IOPContractsManagerContractsContainer);
}

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

interface IOPContractsManagerUpgrader {
    event Upgraded(uint256 indexed l2ChainId, address indexed systemConfig, address indexed upgrader);

    function __constructor__(IOPContractsManagerContractsContainer _contractsContainer) external;

    function upgrade(IOPContractsManager.OpChainConfig[] memory _opChainConfigs) external;

    function contractsContainer() external view returns (IOPContractsManagerContractsContainer);
}

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

interface IOPContractsManager {
    // -------- Structs --------

    /// @notice Represents the roles that can be set when deploying a standard OP Stack chain.
    struct Roles {
        address opChainProxyAdminOwner;
        address systemConfigOwner;
        address batcher;
        address unsafeBlockSigner;
        address proposer;
        address challenger;
    }

    /// @notice The full set of inputs to deploy a new OP Stack chain.
    struct DeployInput {
        Roles roles;
        uint32 basefeeScalar;
        uint32 blobBasefeeScalar;
        uint256 l2ChainId;
        // The correct type is OutputRoot memory but OP Deployer does not yet support structs.
        bytes startingAnchorRoot;
        // The salt mixer is used as part of making the resulting salt unique.
        string saltMixer;
        uint64 gasLimit;
        // Configurable dispute game parameters.
        GameType disputeGameType;
        Claim disputeAbsolutePrestate;
        uint256 disputeMaxGameDepth;
        uint256 disputeSplitDepth;
        Duration disputeClockExtension;
        Duration disputeMaxClockDuration;
    }

    /// @notice The full set of outputs from deploying a new OP Stack chain.
    struct DeployOutput {
        IProxyAdmin opChainProxyAdmin;
        IAddressManager addressManager;
        IL1ERC721Bridge l1ERC721BridgeProxy;
        ISystemConfig systemConfigProxy;
        IOptimismMintableERC20Factory optimismMintableERC20FactoryProxy;
        IL1StandardBridge l1StandardBridgeProxy;
        IL1CrossDomainMessenger l1CrossDomainMessengerProxy;
        IETHLockbox ethLockboxProxy;
        // Fault proof contracts below.
        IOptimismPortal2 optimismPortalProxy;
        IDisputeGameFactory disputeGameFactoryProxy;
        IAnchorStateRegistry anchorStateRegistryProxy;
        IFaultDisputeGame faultDisputeGame;
        IPermissionedDisputeGame permissionedDisputeGame;
        IDelayedWETH delayedWETHPermissionedGameProxy;
        IDelayedWETH delayedWETHPermissionlessGameProxy;
    }

    /// @notice Addresses of ERC-5202 Blueprint contracts. There are used for deploying full size
    /// contracts, to reduce the code size of this factory contract. If it deployed full contracts
    /// using the `new Proxy()` syntax, the code size would get large fast, since this contract would
    /// contain the bytecode of every contract it deploys. Therefore we instead use Blueprints to
    /// reduce the code size of this contract.
    struct Blueprints {
        address addressManager;
        address proxy;
        address proxyAdmin;
        address l1ChugSplashProxy;
        address resolvedDelegateProxy;
        address permissionedDisputeGame1;
        address permissionedDisputeGame2;
        address permissionlessDisputeGame1;
        address permissionlessDisputeGame2;
        address superPermissionedDisputeGame1;
        address superPermissionedDisputeGame2;
        address superPermissionlessDisputeGame1;
        address superPermissionlessDisputeGame2;
    }

    /// @notice The latest implementation contracts for the OP Stack.
    struct Implementations {
        address superchainConfigImpl;
        address protocolVersionsImpl;
        address l1ERC721BridgeImpl;
        address optimismPortalImpl;
        address ethLockboxImpl;
        address systemConfigImpl;
        address optimismMintableERC20FactoryImpl;
        address l1CrossDomainMessengerImpl;
        address l1StandardBridgeImpl;
        address disputeGameFactoryImpl;
        address anchorStateRegistryImpl;
        address delayedWETHImpl;
        address mipsImpl;
    }

    /// @notice The input required to identify a chain for upgrading.
    struct OpChainConfig {
        ISystemConfig systemConfigProxy;
        IProxyAdmin proxyAdmin;
        Claim absolutePrestate;
    }

    struct AddGameInput {
        string saltMixer;
        ISystemConfig systemConfig;
        IProxyAdmin proxyAdmin;
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

    // -------- Constants and Variables --------

    function version() external pure returns (string memory);

    /// @notice Address of the SuperchainConfig contract shared by all chains.
    function superchainConfig() external view returns (ISuperchainConfig);

    /// @notice Address of the ProtocolVersions contract shared by all chains.
    function protocolVersions() external view returns (IProtocolVersions);

    /// @notice Address of the ProxyAdmin contract shared by all chains.
    function superchainProxyAdmin() external view returns (IProxyAdmin);

    /// @notice L1 smart contracts release deployed by this version of OPCM. This is used in opcm to signal which
    /// version of the L1 smart contracts is deployed. It takes the format of `op-contracts/vX.Y.Z`.
    function l1ContractsRelease() external view returns (string memory);

    // -------- Errors --------

    error OnlyUpgradeController();

    /// @notice Thrown when an address is the zero address.
    error AddressNotFound(address who);

    /// @notice Throw when a contract address has no code.
    error AddressHasNoCode(address who);

    /// @notice Thrown when a release version is already set.
    error AlreadyReleased();

    /// @notice Thrown when an invalid `l2ChainId` is provided to `deploy`.
    error InvalidChainId();

    /// @notice Thrown when a role's address is not valid.
    error InvalidRoleAddress(string role);

    /// @notice Thrown when the latest release is not set upon initialization.
    error LatestReleaseNotSet();

    /// @notice Thrown when the starting anchor root is not provided.
    error InvalidStartingAnchorRoot();

    /// @notice Thrown when certain methods are called outside of a DELEGATECALL.
    error OnlyDelegatecall();

    /// @notice Thrown when game configs passed to addGameType are invalid.
    error InvalidGameConfigs();

    /// @notice Thrown when the SuperchainConfig of the chain does not match the SuperchainConfig of this OPCM.
    error SuperchainConfigMismatch(ISystemConfig systemConfig);

    error SuperchainProxyAdminMismatch();

    error PrestateNotSet();

    error PrestateRequired();

    // -------- Methods --------

    function __constructor__(
        IOPContractsManagerGameTypeAdder _opcmGameTypeAdder,
        IOPContractsManagerDeployer _opcmDeployer,
        IOPContractsManagerUpgrader _opcmUpgrader,
        IOPContractsManagerInteropMigrator _opcmInteropMigrator,
        ISuperchainConfig _superchainConfig,
        IProtocolVersions _protocolVersions,
        IProxyAdmin _superchainProxyAdmin,
        string memory _l1ContractsRelease,
        address _upgradeController
    )
        external;

    function deploy(DeployInput calldata _input) external returns (DeployOutput memory);

    /// @notice Upgrades the implementation of all proxies in the specified chains
    /// @param _opChainConfigs The chains to upgrade
    function upgrade(OpChainConfig[] memory _opChainConfigs) external;

    /// @notice addGameType deploys a new dispute game and links it to the DisputeGameFactory. The inputted _gameConfigs
    /// must be added in ascending GameType order.
    function addGameType(AddGameInput[] memory _gameConfigs) external returns (AddGameOutput[] memory);

    /// @notice Updates the prestate hash for a new game type while keeping all other parameters the same
    /// @param _prestateUpdateInputs The new prestate hash to use
    function updatePrestate(OpChainConfig[] memory _prestateUpdateInputs) external;

    /// @notice Migrates one or more OP Stack chains to use the Super Root dispute games and shared
    ///         dispute game contracts.
    /// @param _input The input parameters for the migration.
    function migrate(IOPContractsManagerInteropMigrator.MigrateInput calldata _input) external;

    /// @notice Maps an L2 chain ID to an L1 batch inbox address as defined by the standard
    /// configuration's convention. This convention is `versionByte || keccak256(bytes32(chainId))[:19]`,
    /// where || denotes concatenation`, versionByte is 0x00, and chainId is a uint256.
    /// https://specs.optimism.io/protocol/configurability.html#consensus-parameters
    function chainIdToBatchInboxAddress(uint256 _l2ChainId) external pure returns (address);

    /// @notice Returns the blueprint contract addresses.
    function blueprints() external view returns (Blueprints memory);

    function opcmDeployer() external view returns (IOPContractsManagerDeployer);

    function opcmUpgrader() external view returns (IOPContractsManagerUpgrader);

    function opcmGameTypeAdder() external view returns (IOPContractsManagerGameTypeAdder);

    function opcmInteropMigrator() external view returns (IOPContractsManagerInteropMigrator);

    /// @notice Returns the implementation contract addresses.
    function implementations() external view returns (Implementations memory);

    function upgradeController() external view returns (address);

    function isRC() external view returns (bool);

    function setRC(bool _isRC) external;
}

/// @notice Minimal interface only used for calling `implementations()` method but without retrieving the ETHLockbox
///         on it, since the OPCM contracts already deployed on mainnet don't have it.
/// @dev    Only used for testing.
interface IOPCMImplementationsWithoutLockbox {
    /// @notice The implementation contracts for the OP Stack, without the newly added ETHLockbox.
    struct Implementations {
        address superchainConfigImpl;
        address protocolVersionsImpl;
        address l1ERC721BridgeImpl;
        address optimismPortalImpl;
        address systemConfigImpl;
        address optimismMintableERC20FactoryImpl;
        address l1CrossDomainMessengerImpl;
        address l1StandardBridgeImpl;
        address disputeGameFactoryImpl;
        address anchorStateRegistryImpl;
        address delayedWETHImpl;
        address mipsImpl;
    }

    /// @notice Returns the implementation contracts without the ETHLockbox.
    function implementations() external view returns (Implementations memory);
}
