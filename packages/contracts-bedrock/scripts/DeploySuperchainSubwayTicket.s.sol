// SPDX-License-Identifier: MIT
pragma solidity ^0.8.15;

import "forge-std/Script.sol";

import { SuperchainSubwayTicket } from "src/periphery/SuperchainSubwayTicket.sol";

contract DeploySuperchainSubwayTicket is Script {
    function name() public pure returns (string memory name_) {
        name_ = "DeploySuperchainSubwayTicket";
    }

    function run() external {
        console.log("Deploying Superchain Subway Ticket");
        vm.startBroadcast();

        string memory name = "Superchain Subway Ticket";
        string memory symbol = "SUPER";
        string memory baseURI = "";

        SuperchainSubwayTicket nft = new SuperchainSubwayTicket(name, symbol, baseURI);

        vm.stopBroadcast();
    }
}
