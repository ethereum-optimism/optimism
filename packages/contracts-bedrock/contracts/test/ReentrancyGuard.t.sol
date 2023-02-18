// SPDX-License-Identifier: Unlicense
pragma solidity 0.8.15;

import { Test } from "forge-std/Test.sol";
import { ReentrancyGuard } from "../libraries/ReentrancyGuard.sol";

contract ReentrancyGuard_Test is Test {
    /// @dev The message hash passed to `noReentrance`
    bytes32 internal constant MSG_HASH = keccak256(abi.encode("MESSAGE_HASH"));

    NonReentrant internal reentrant;

    function setUp() public {
        reentrant = new NonReentrant();
    }

    function test_perMessageNonReentrant_diffHash_succeeds() public {
        reentrant.noReentrance(bytes32(0));
    }

    function test_perMessageNonReentrant_sameHash_reverts() public {
        vm.expectRevert("ReentrancyGuard: reentrant call");
        reentrant.noReentrance(MSG_HASH);
    }

    fallback() external {
        reentrant.noReentrance(MSG_HASH);
    }
}

contract NonReentrant is ReentrancyGuard {
    bool onlyOnce;

    function noReentrance(bytes32 _hash) external perMessageNonReentrant(_hash) {
        // Only allow the following call back to the sender occur once.
        if (onlyOnce) return;
        onlyOnce = true;

        assembly {
            let success := call(gas(), caller(), 0, 0, 0, 0, 0)
            returndatacopy(0x00, 0x00, returndatasize())
            switch success
            case 0 {
                revert(0x00, returndatasize())
            }
            default {
                return(0x00, returndatasize())
            }
        }
    }
}
