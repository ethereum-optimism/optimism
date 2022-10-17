// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { HuffDeployer } from "foundry-huff/HuffDeployer.sol";
import { Test } from "forge-std/Test.sol";
import { console } from "forge-std/console.sol";

contract BatchPortal_Test is Test {
    //

    function setUp() {
        address addr = HuffDeployer.deploy("L1/BatchPortal");
        console.log(addr);
    }
}
