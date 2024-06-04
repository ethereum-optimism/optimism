// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { Deploy } from "scripts/Deploy.s.sol";

contract KontrolDeployment is Deploy {
    function runKontrolDeployment() public stateDiff {
        deploySafe("SystemOwnerSafe");
        setupSuperchain();

        // deployProxies();
        deployERC1967Proxy("OptimismPortalProxy");
        deployERC1967Proxy("L2OutputOracleProxy");
        deployERC1967Proxy("SystemConfigProxy");
        deployL1StandardBridgeProxy();
        deployL1CrossDomainMessengerProxy();
        deployERC1967Proxy("L1ERC721BridgeProxy");
        transferAddressManagerOwnership(); // to the ProxyAdmin

        // deployImplementations();
        deployOptimismPortal();
        deployL1CrossDomainMessenger();
        deployL2OutputOracle();
        deploySystemConfig();
        deployL1StandardBridge();
        deployL1ERC721Bridge();

        // initializeImplementations();
        initializeSystemConfig();
        initializeL1StandardBridge();
        initializeL1ERC721Bridge();
        initializeL1CrossDomainMessenger();
        initializeOptimismPortal();
    }

    function runKontrolDeploymentFaultProofs() public stateDiff {
        deploySafe("SystemOwnerSafe");
        setupSuperchain();

        // deployProxies();
        deployERC1967Proxy("OptimismPortalProxy");
        deployERC1967Proxy("DisputeGameFactoryProxy");
        deployERC1967Proxy("AnchorStateRegistryProxy");
        deployERC1967Proxy("DelayedWETHProxy");
        deployERC1967Proxy("SystemConfigProxy");
        deployL1StandardBridgeProxy();
        deployL1CrossDomainMessengerProxy();
        deployERC1967Proxy("L1ERC721BridgeProxy");
        transferAddressManagerOwnership(); // to the ProxyAdmin

        // deployImplementations();
        deployOptimismPortal2();
        deployL1CrossDomainMessenger();
        deploySystemConfig();
        deployL1StandardBridge();
        deployL1ERC721Bridge();
        deployDisputeGameFactory();
        deployDelayedWETH();
        deployPreimageOracle();
        deployMips();
        deployAnchorStateRegistry();

        // initializeImplementations();
        initializeSystemConfig();
        initializeL1StandardBridge();
        initializeL1ERC721Bridge();
        initializeL1CrossDomainMessenger();
        initializeDisputeGameFactory();
        initializeDelayedWETH();
        initializeAnchorStateRegistry();
        initializeOptimismPortal2();

        // Set dispute game implementations in DGF
        setCannonFaultGameImplementation({ _allowUpgrade: false });
        setPermissionedCannonFaultGameImplementation({ _allowUpgrade: false });
    }
}
