// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { CommonBase } from "forge-std/Base.sol";
import { console2 as console } from "forge-std/console2.sol";

contract Experiment is CommonBase {

    function doThing() public view {
        console.log("contract addr", address(this));
        console.log("contract nonce", vm.getNonce(address(this)));
        console.log("sender addr", address(msg.sender));
        console.log("sender nonce", vm.getNonce(address(msg.sender)));
        console.log("done!");
    }
}
