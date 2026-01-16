// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

// Libraries
import { GameType, Proposal } from "src/dispute/lib/Types.sol";

// Interfaces
import { IProxyAdmin } from "interfaces/universal/IProxyAdmin.sol";
import { ISuperchainConfig } from "interfaces/L1/ISuperchainConfig.sol";
import { ISystemConfig } from "interfaces/L1/ISystemConfig.sol";
import { IL1CrossDomainMessenger } from "interfaces/L1/IL1CrossDomainMessenger.sol";
import { IL1ERC721Bridge } from "interfaces/L1/IL1ERC721Bridge.sol";
import { IL1StandardBridge } from "interfaces/L1/IL1StandardBridge.sol";
import { IOptimismPortal2 } from "interfaces/L1/IOptimismPortal2.sol";
import { IOptimismMintableERC20Factory } from "interfaces/universal/IOptimismMintableERC20Factory.sol";
import { IDisputeGameFactory } from "interfaces/dispute/IDisputeGameFactory.sol";
import { IAnchorStateRegistry } from "interfaces/dispute/IAnchorStateRegistry.sol";
import { IDelayedWETH } from "interfaces/dispute/IDelayedWETH.sol";
import { IAddressManager } from "interfaces/legacy/IAddressManager.sol";
import { IETHLockbox } from "interfaces/L1/IETHLockbox.sol";
import { IResourceMetering } from "interfaces/L1/IResourceMetering.sol";
import { IOPContractsManagerContainer } from "interfaces/L1/opcm/IOPContractsManagerContainer.sol";
import { IOPContractsManagerStandardValidator } from "interfaces/L1/IOPContractsManagerStandardValidator.sol";
import { IOPContractsManagerUtils } from "interfaces/L1/opcm/IOPContractsManagerUtils.sol";
import { IOPContractsManagerMigrator } from "interfaces/L1/opcm/IOPContractsManagerMigrator.sol";

interface IOPContractsManagerV2 {
    /// @notice Contracts that represent the Superchain system.
    struct SuperchainContracts {
        ISuperchainConfig superchainConfig;
    }

    /// @notice Addresses of the deployed and wired contracts for an OP Chain.
    struct ChainContracts {
        ISystemConfig systemConfig;
        IProxyAdmin proxyAdmin;
        IAddressManager addressManager;
        IL1CrossDomainMessenger l1CrossDomainMessenger;
        IL1ERC721Bridge l1ERC721Bridge;
        IL1StandardBridge l1StandardBridge;
        IOptimismPortal2 optimismPortal;
        IETHLockbox ethLockbox;
        IOptimismMintableERC20Factory optimismMintableERC20Factory;
        IDisputeGameFactory disputeGameFactory;
        IAnchorStateRegistry anchorStateRegistry;
        IDelayedWETH delayedWETH;
    }

    /// @notice Full configuration for deploying a new OP Chain.
    struct FullConfig {
        string saltMixer;
        ISuperchainConfig superchainConfig;
        address proxyAdminOwner;
        address systemConfigOwner;
        address unsafeBlockSigner;
        address batcher;
        Proposal startingAnchorRoot;
        GameType startingRespectedGameType;
        uint32 basefeeScalar;
        uint32 blobBasefeeScalar;
        uint64 gasLimit;
        uint256 l2ChainId;
        IResourceMetering.ResourceConfig resourceConfig;
        IOPContractsManagerUtils.DisputeGameConfig[] disputeGameConfigs;
        bool useCustomGasToken;
    }

    struct UpgradeInput {
        ISystemConfig systemConfig;
        IOPContractsManagerUtils.DisputeGameConfig[] disputeGameConfigs;
        IOPContractsManagerUtils.ExtraInstruction[] extraInstructions;
    }

    struct SuperchainUpgradeInput {
        ISuperchainConfig superchainConfig;
        IOPContractsManagerUtils.ExtraInstruction[] extraInstructions;
    }

    error OPContractsManagerV2_InvalidGameConfigs();
    error OPContractsManagerV2_InvalidUpgradeInput();
    error OPContractsManagerV2_SuperchainConfigNeedsUpgrade();
    error OPContractsManagerV2_InvalidUpgradeInstruction(string _key);
    error OPContractsManagerV2_CannotUpgradeToCustomGasToken();
    error OPContractsManagerV2_InvalidUpgradeSequence(string _lastVersion, string _thisVersion);
    error IdentityPrecompileCallFailed();
    error ReservedBitsSet();
    error BytesArrayTooLong();
    error SemverComp_InvalidSemverParts();
    error UnsupportedERCVersion(uint8 version);
    error NotABlueprint();
    error DeploymentFailed();
    error EmptyInitcode();
    error UnexpectedPreambleData(bytes data);

    function __constructor__(
        IOPContractsManagerStandardValidator _standardValidator,
        IOPContractsManagerMigrator _migrator,
        IOPContractsManagerUtils _utils
    )
        external;

    function blueprints() external view returns (IOPContractsManagerContainer.Blueprints memory);

    function implementations() external view returns (IOPContractsManagerContainer.Implementations memory);

    function contractsContainer() external view returns (IOPContractsManagerContainer);

    function opcmStandardValidator() external view returns (IOPContractsManagerStandardValidator);

    function opcmV2() external view returns (IOPContractsManagerV2);

    function opcmMigrator() external view returns (IOPContractsManagerMigrator);

    function opcmUtils() external view returns (IOPContractsManagerUtils);

    function version() external view returns (string memory);

    /// @notice Upgrades Superchain-wide contracts.
    function upgradeSuperchain(SuperchainUpgradeInput memory _inp) external returns (SuperchainContracts memory);

    /// @notice Deploys and wires a complete OP Chain per the provided configuration.
    function deploy(FullConfig memory _cfg) external returns (ChainContracts memory);

    /// @notice Upgrades contracts on an existing OP Chain per the provided input.
    function upgrade(UpgradeInput memory _inp) external returns (ChainContracts memory);

    /// @notice Migrates one or more OP Stack chains to use the Super Root dispute games and shared
    ///         dispute game contracts.
    function migrate(IOPContractsManagerMigrator.MigrateInput calldata _input) external;

    /// @notice Returns whether a development feature is enabled.
    function isDevFeatureEnabled(bytes32 _feature) external view returns (bool);

    /// @notice Checks if the upgrade sequence from the last used OPCM to this OPCM is permitted.
    function isPermittedUpgradeSequence(ISystemConfig _systemConfig) external view returns (bool);

    /// @notice Returns the development feature bitmap.
    function devFeatureBitmap() external view returns (bytes32);
}
