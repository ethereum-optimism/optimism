// SPDX-License-Identifier: GPL-3
pragma solidity ^0.8.24;

import { console } from "forge-std/console.sol";

/// @title CompatibleAssert
/// @notice meant to add compatibility between medusa assertion tests and
/// foundry invariant test's required architecture
contract CompatibleAssert {
    bool public failed;

    function compatibleAssert(bool condition) internal {
        compatibleAssert(condition, "");
    }

    function compatibleAssert(bool condition, string memory message) internal {
        if (!condition) {
            if (bytes(message).length != 0) console.log("Assertion failed: ", message);
            else console.log("Assertion failed");

            // for foundry to call & check
            failed = true;

            // for medusa
            assert(false);
        }
    }
}
