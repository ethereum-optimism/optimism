// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

contract Alligator {
    /// @notice Callback called after a token transfer.
    /// @param from   The account sending tokens.
    /// @param to     The account receiving tokens.
    /// @param amount The amount of tokens being transfered.
    function afterTokenTransfer(address from, address to, uint256 amount) internal { }
}
