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
import { IDisputeGame } from "interfaces/dispute/IDisputeGame.sol";
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
import { IOPContractsManagerStandardValidator } from "interfaces/L1/IOPContractsManagerStandardValidator.sol";
import { IResourceMetering } from "interfaces/L1/IResourceMetering.sol";

interface IOPContractsManager {
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

    struct FaultDisputeGameConfig {
        Claim absolutePrestate;
    }

    struct PermissionedDisputeGameConfig {
        Claim absolutePrestate;
        address proposer;
        address challenger;
    }

    struct DisputeGameConfig {
        GameType gameType;
        bytes gameArgs;
    }

    struct SystemRoles {
        address proxyAdminOwner;
        address systemConfigOwner;
        address unsafeBlockSigner;
        bytes32 batcherHash;
    }

    struct L2SystemConfig {
        uint32 basefeeScalar;
        uint32 blobBasefeeScalar;
        uint64 gasLimit;
        uint256 l2ChainId;
        IResourceMetering.ResourceConfig resourceConfig;
    }

    struct AnchorStateConfig {
        bytes startingAnchorRoot;
        GameType startingRespectedGameType;
    }

    struct UpgradeInput {
        ISystemConfig systemConfigProxy;
        DisputeGameConfig[] disputeGameConfigs;
    }

    struct DisputeGameContracts {
        IDisputeGame disputeGame;
        IDelayedWETH delayedWETH;
    }

    struct ChainContracts {
        ISystemConfig systemConfig;
        IProxyAdmin proxyAdmin;
        IAddressManager addressManager;
        ISuperchainConfig superchainConfig;
        IL1CrossDomainMessenger l1CrossDomainMessenger;
        IL1ERC721Bridge l1ERC721Bridge;
        IL1StandardBridge l1StandardBridge;
        IOptimismPortal optimismPortal;
        IETHLockbox ethLockbox;
        IOptimismMintableERC20Factory optimismMintableERC20Factory;
        IDisputeGameFactory disputeGameFactory;
        IAnchorStateRegistry anchorStateRegistry;
        IDelayedWETH delayedWETH;
    }

    struct FullConfig {
        string saltMixer;
        SystemRoles roles;
        L2SystemConfig l2SystemConfig;
        DisputeGameConfig[] disputeGameConfigs;
        AnchorStateConfig anchorStateConfig;
    }

    struct ExecutionInput {
        ChainContracts cts;
        FullConfig cfg;
    }

    struct ExecutionOutput {
        ChainContracts cts;
    }

    struct ProxyDeployArgs {
        uint256 l2ChainId;
        IProxyAdmin proxyAdmin;
        string saltMixer;
    }

    error OPContractsManager_SuperchainConfigNeedsUpgrade();
    error OPContractsManager_UnknownGameType();

    function version() external view returns (string memory);
    function blueprints() external view returns (Blueprints memory);
    function implementations() external view returns (Implementations memory);

    function deploy(FullConfig memory _input) external returns (ExecutionOutput memory);
    function upgrade(UpgradeInput memory _input) external returns (ExecutionOutput memory);
}
