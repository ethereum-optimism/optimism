// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { ERC20 } from "@openzeppelin/contracts/token/ERC20/ERC20.sol";
import { ERC20Votes } from "@openzeppelin/contracts/token/ERC20/extensions/ERC20Votes.sol";
import { Predeploys } from "src/libraries/Predeploys.sol";
import { IGovernanceDelegation } from "src/governance/IGovernanceDelegation.sol";
import { GovernanceToken } from "src/governance/GovernanceToken.sol";

contract GovernanceTokenInterop is GovernanceToken {
    /// @notice Returns the checkpoint for a given account at a given position.
    /// @param _account The account to get the checkpoints for.
    /// @param _pos     The psition to get the checkpoints at.
    /// @return         The checkpoint at the given position.
    function checkpoints(address _account, uint32 _pos) public view override(ERC20Votes) returns (Checkpoint memory) {
        if (_migrated(_account)) {
            return IGovernanceDelegation(Predeploys.GOVERNANCE_DELEGATION).checkpoints(_account)[_pos];
        } else {
            return super.checkpoints(_account, _pos);
        }
    }

    /// @notice Returns the number of checkpoints for a given account.
    /// @param _account The account to get the number of checkpoints for.
    /// @return         The number of checkpoints for the given account.
    function numCheckpoints(address _account) public view override(ERC20Votes) returns (uint32) {
        if (_migrated(_account)) {
            return IGovernanceDelegation(Predeploys.GOVERNANCE_DELEGATION).numCheckpoints(_account);
        } else {
            return super.numCheckpoints(_account);
        }
    }

    /// @notice Returns the delegatee of an account. This function is unavailable post migration,
    ///         because the GovernanceDelegation may hold more than one delegatee for an account, conflicting
    ///         the return type of this function.
    /// @param _account The account to get the delegatee of.
    /// @return         The delegatee of the given account.
    function delegates(address _account) public view override(ERC20Votes) returns (address) {
        if (_migrated(_account)) {
            return IGovernanceDelegation(Predeploys.GOVERNANCE_DELEGATION).delegates(_account);
        } else {
            return super.delegates(_account);
        }
    }

    /// @notice Returns the number of votes for a given account.
    /// @param _account The account to get the number of votess for.
    /// @return         The number of votes for the given account.
    function getVotes(address _account) public view override(ERC20Votes) returns (uint256) {
        if (_migrated(_account)) {
            return IGovernanceDelegation(Predeploys.GOVERNANCE_DELEGATION).getVotes(_account);
        } else {
            return super.getVotes(_account);
        }
    }

    /// @notice Returns the number of votes for a given account at a block.
    /// @param _account The account to get the number of checkpoints for.
    /// @param _blockNumber The block number to get the number of votes for.
    /// @return         The number of votes for the given account and block number.
    function getPastVotes(address _account, uint256 _blockNumber) public view override(ERC20Votes) returns (uint256) {
        if (_migrated(_account)) {
            return IGovernanceDelegation(Predeploys.GOVERNANCE_DELEGATION).getPastVotes(_account, _blockNumber);
        } else {
            return super.getPastVotes(_account, _blockNumber);
        }
    }

    /// @notice Returns the total supply at a block.
    /// @param _blockNumber The block number to get the total supply.
    /// @return         The total supply of the token for the given block.
    function getPastTotalSupply(uint256 _blockNumber) public view override(ERC20Votes) returns (uint256) {
        return IGovernanceDelegation(Predeploys.GOVERNANCE_DELEGATION).getPastTotalSupply(_blockNumber);
    }

    /// @notice Delegates votes from the `_delegator` to `_delegatee`.
    /// @param _delegator The account delegating votes.
    /// @param _delegatee The account to delegate votes to.
    function _delegate(address _delegator, address _delegatee) internal override(ERC20Votes) {
        // GovernanceDelegation will migrate account if necessary.
        IGovernanceDelegation(Predeploys.GOVERNANCE_DELEGATION).delegateFromToken(_delegator, _delegatee);
    }

    /// @notice Callback called after a token transfer. Forwards to the GovernanceDelegation contract,
    ///         independently of whether the account has been migrated.
    /// @param from   The account sending tokens.
    /// @param to     The account receiving tokens.
    /// @param amount The amount of tokens being transfered.
    function _afterTokenTransfer(address from, address to, uint256 amount) internal override(GovernanceToken) {
        IGovernanceDelegation(Predeploys.GOVERNANCE_DELEGATION).afterTokenTransfer(from, to, amount);
    }

    /// @notice Determines whether an account has been migrated.
    /// @param _account The account to check if it has been migrated.
    /// @return         True if the given account has been migrated, and false otherwise.
    function _migrated(address _account) internal view returns (bool) {
        return IGovernanceDelegation(Predeploys.GOVERNANCE_DELEGATION).migrated(_account);
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
