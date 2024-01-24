// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { Deploy } from "scripts/Deploy.s.sol";

contract KontrolDeployment is Deploy {
    function runKontrolDeployment() public stateDiff {
        deploySafe();
        setupSuperchain();

        // deployProxies();
        deployERC1967Proxy("OptimismPortalProxy");
        deployERC1967Proxy("L2OutputOracleProxy");
        deployERC1967Proxy("SystemConfigProxy");
        deployL1CrossDomainMessengerProxy();
        transferAddressManagerOwnership(); // to the ProxyAdmin

        // deployImplementations();
        deployOptimismPortal();
        deployL1CrossDomainMessenger();
        deployL2OutputOracle();
        deploySystemConfig();

        // initializeImplementations();
        initializeSystemConfig();
        initializeL1CrossDomainMessenger();
        initializeOptimismPortal();
    }
}
