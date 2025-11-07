// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

/// @notice Helper recipient that always reverts on receiving ETH
contract RevertingRecipient {
    receive() external payable {
        revert("RevertingRecipient: cannot accept ETH");
    }
}
