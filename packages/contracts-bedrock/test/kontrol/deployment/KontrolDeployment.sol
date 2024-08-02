// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { Deploy } from "scripts/deploy/Deploy.s.sol";

contract KontrolDeployment is Deploy {
    function runKontrolDeployment() public {
        runWithStateDiff();
    }

    function runKontrolDeploymentFaultProofs() public {
        cfg.setUseFaultProofs(true);
        runWithStateDiff();
    }
}
