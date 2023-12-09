// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { VmSafe } from "forge-std/Vm.sol";
import { Script } from "forge-std/Script.sol";

import { console2 as console } from "forge-std/console2.sol";
import { stdJson } from "forge-std/StdJson.sol";

import {Counter} from "./Counter.sol";
import { LibStateDiff } from "scripts/libraries/LibStateDiff.sol";
import { OptimismPortal } from "src/L1/OptimismPortal.sol";
import { Deploy } from "scripts/Deploy.s.sol";

contract MakeStateDiff is Deploy {

    function testStateDiff() public stateDiff /* broadcast */ {
        /* Counter counter = new Counter(); */
        /* console.log("COUNTER", address(counter)); */
        /* Counter counter2 = new Counter(); */
        /* console.log("COUNTER2", address(counter2)); */
        /* counter.setNumber(3); */
        /* counter.setNumber(42); */
        /* counter2.setNumber(777); */
        /* OptimismPortal portal = new OptimismPortal */
        deploySafe();
        setupSuperchain();

        /* deployProxies(); */
        deployERC1967Proxy("OptimismPortalProxy");
        deployERC1967Proxy("L2OutputOracleProxy");
        deployERC1967Proxy("SystemConfigProxy");
        transferAddressManagerOwnership(); // to the ProxyAdmin

        /* deployImplementations(); */
        deployOptimismPortal();
        deployL2OutputOracle();
        deploySystemConfig();

        /* initializeImplementations(); */
        initializeSystemConfig();
        initializeOptimismPortal();
    }
}
