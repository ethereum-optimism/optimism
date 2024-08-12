// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { CommonBase } from "forge-std/Base.sol";
import { console2 as console } from "forge-std/console2.sol";

/// @title ScriptExample
/// @notice ScriptExample is an example script. The Go forge script code tests that it can run this.
contract ScriptExample is CommonBase {

    /// @notice example function, runs through basic cheat-codes and console logs.
    function run() public view {
        console.log("contract addr", address(this));
        console.log("contract nonce", vm.getNonce(address(this)));
        console.log("sender addr", address(msg.sender));
        console.log("sender nonce", vm.getNonce(address(msg.sender)));
        console.log("done!");
    }
}
