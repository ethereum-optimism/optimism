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

/// @custom:legacy
/// @title IOPContractsManager
/// @notice Legacy interface preserved for type compatibility. The OPCMv1 implementation has been
///         removed. Only struct and error definitions remain. New code should use
///         IOPContractsManagerV2 instead.
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
}

/// @custom:legacy
/// @notice Legacy interface for the v1 interop migrator. Preserved for type compatibility.
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
}

/// @custom:legacy
/// @notice Legacy interface for the v1 upgrader. Preserved for type compatibility.
interface IOPContractsManagerUpgrader {
    event Upgraded(uint256 indexed l2ChainId, address indexed systemConfig, address indexed upgrader);
    error OPContractsManagerUpgrader_SuperchainConfigNeedsUpgrade(uint256 index);
    error OPContractsManagerUpgrader_SuperchainConfigAlreadyUpToDate();
}
