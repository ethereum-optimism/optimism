// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

interface IOPContractsManagerContainer {
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
        address storageSetterImpl;
    }

    error OPContractsManagerContainer_DevFeatureInProd();

    function blueprints() external view returns (Blueprints memory);
    function implementations() external view returns (Implementations memory);
    function isDevFeatureEnabled(bytes32 _feature) external view returns (bool);
    function devFeatureBitmap() external view returns (bytes32);
    function __constructor__(Blueprints memory _blueprints, Implementations memory _implementations, bytes32 _devFeatureBitmap) external;
}
