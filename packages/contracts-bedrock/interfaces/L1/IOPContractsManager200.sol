// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

/// @notice Interface for the OPCM v2.0.0 release.
interface IOPContractsManager200 {
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
    }

    function blueprints() external view returns (Blueprints memory);
}
