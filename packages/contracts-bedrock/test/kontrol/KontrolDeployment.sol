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
        transferAddressManagerOwnership(); // to the ProxyAdmin

        // deployImplementations();
        deployOptimismPortal();
        deployL2OutputOracle();
        deploySystemConfig();

        // initializeImplementations();
        initializeSystemConfig();
        initializeOptimismPortal();
    }
}
