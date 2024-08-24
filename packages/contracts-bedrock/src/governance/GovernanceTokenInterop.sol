// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { ERC20 } from "@openzeppelin/contracts/token/ERC20/ERC20.sol";
import { ERC20Votes } from "@openzeppelin/contracts/token/ERC20/extensions/ERC20Votes.sol";
import { Predeploys } from "src/libraries/Predeploys.sol";
import { IGovernanceDelegation } from "src/governance/IGovernanceDelegation.sol";
import { GovernanceToken } from "src/governance/GovernanceToken.sol";

contract GovernanceTokenInterop is GovernanceToken {
    /// @notice Delegates votes from the `_delegator` to `_delegatee`.
    /// @param _delegator The account delegating votes.
    /// @param _delegatee The account to delegate votes to.
    function _delegate(address _delegator, address _delegatee) internal override(ERC20Votes) {
        // GovernanceDelegation will migrate account if necessary.
        IGovernanceDelegation(Predeploys.GOVERNANCE_DELEGATION).delegateFromToken(_delegator, _delegatee);
    }

    /// @notice Callback called after a token transfer. Forwards to the GovernanceDelegation contract,
    ///         independently of whether the account has been migrated.
    /// @param _from The account sending tokens.
    /// @param _to The account receiving tokens.
    /// @param _amount The amount of tokens being transfered.
    function _afterTokenTransfer(address _from, address _to, uint256 _amount) internal override(GovernanceToken) {
        IGovernanceDelegation(Predeploys.GOVERNANCE_DELEGATION).afterTokenTransfer(_from, _to, _amount);
    }

    /// @notice Internal mint function.
    /// @param _account     The account receiving minted tokens.
    /// @param _amount      The amount of tokens to mint.
    function _mint(address _account, uint256 _amount) internal override(GovernanceToken) {
        ERC20._mint(_account, _amount);
        require(totalSupply() <= _maxSupply(), "GovernanceToken: total supply risks overflowing votes");
        // Total supply checkpoint is written by GovernanceDelegation via the hook.
    }

    /// @notice Internal burn function.
    /// @param _account The account that tokens will be burned from.
    /// @param _amount  The amount of tokens that will be burned.
    function _burn(address _account, uint256 _amount) internal override(GovernanceToken) {
        ERC20._burn(_account, _amount);
        // Total supply checkpoint is written by GovernanceDelegation via the hook.
    }
}
