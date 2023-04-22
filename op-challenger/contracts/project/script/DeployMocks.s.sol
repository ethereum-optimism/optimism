// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.13;

import "forge-std/Script.sol";
import { MockDisputeGameFactory } from "src/MockDisputeGameFactory.sol";
import { MockAttestationDisputeGame } from "src/MockAttestationDisputeGame.sol";

/// @notice This script deploys the mock dispute game factory.
contract DeployMocks is Script {
    function run() public {
        // Deploy the mock dispute game factory
        vm.broadcast();
        MockDisputeGameFactory mock = new MockDisputeGameFactory();

        // Write the address to the devnet ENV
        console.log(string.concat("ENV Variable: export OP_CHALLENGER_DGF=\"", vm.toString(address(mock)), "\""));
    }
}
