// SPDX-License-Identifier: MIT
pragma solidity 0.8.25;

// Target contracts
import { CrossL2Inbox } from "src/L2/CrossL2Inbox.sol";

/// @title CrossL2InboxWithSlotWarming
/// @dev CrossL2Inbox contract with a method to warm a slot.
contract CrossL2InboxWithSlotWarming is CrossL2Inbox {
    // Getter for warming a slot on tests.
    function warmSlot(bytes32 _slot) external view returns (uint256 res_) {
        assembly {
            res_ := sload(_slot)
        }
    }

    // Getter to expose `_isWarm` function for the tests.
    function isWarm(bytes32 _slot) external view returns (bool isWarm_, uint256 value_) {
        (isWarm_, value_) = _isWarm(_slot);
    }
}
