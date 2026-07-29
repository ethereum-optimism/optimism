// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

/// @title  SafeSend
/// @notice Sends ETH to a recipient account without triggering any code.
/// @dev    `selfdestruct` delivers the contract's entire balance to `_recipient`, not only the
///         value supplied by the caller. Passing this contract's own address as `_recipient` burns
///         the value, the same as choosing any inaccessible recipient.
contract SafeSend {
    /// @param _recipient Account to send ETH to.
    constructor(address payable _recipient) payable {
        selfdestruct(_recipient);
    }
}
