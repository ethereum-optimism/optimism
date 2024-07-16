// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import "@openzeppelin/contracts/token/ERC20/extensions/ERC20Votes.sol";

contract Alligator {
    mapping(address => bool) public migrated;

    /// @notice Callback called after a token transfer.
    /// @param from   The account sending tokens.
    /// @param to     The account receiving tokens.
    /// @param amount The amount of tokens being transfered.
    function afterTokenTransfer(address from, address to, uint256 amount) public { }

    function checkpoints(address _account, uint32 _pos) public view returns (ERC20Votes.Checkpoint memory) { }

    /// @notice Returns the number of checkpoints for a given account.
    /// @param _account Account to get the number of checkpoints for.
    /// @return Number of checkpoints for the given account.
    function numCheckpoints(address _account) public view returns (uint32) { }

    /// @notice Returns the delegatee of an account.
    /// @param _account Account to get the delegatee of.
    /// @return Delegatee of the given account.
    function delegates(address _account) public view returns (address) { }

    /// @notice Delegates votes from the sender to `delegatee`.
    /// @param _delegatee Account to delegate votes to.
    function delegate(address _delegatee) public { }

    /// @notice Delegates votes from the sender to `delegatee`.
    /// @param _delegatee Account to delegate votes to.
    /// @param _nonce     Nonce of the transaction.
    /// @param _expiry    Expiry of the signature.
    /// @param _v         v of the signature.
    /// @param _r         r of the signature.
    /// @param _s         s of the signature.
    function delegateBySig(
        address _delegatee,
        uint256 _nonce,
        uint256 _expiry,
        uint8 _v,
        bytes32 _r,
        bytes32 _s
    )
        public
    { }
}
