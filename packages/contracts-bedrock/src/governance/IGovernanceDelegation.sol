// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { ERC20Votes } from "@openzeppelin/contracts/token/ERC20/extensions/ERC20Votes.sol";

/// @title IGovernanceDelegation
/// @notice Interface for the GovernanceDelegation contract.
interface IGovernanceDelegation {
    /// @notice Allowance type of a delegation.
    /// @param Absolute The amount of votes delegated is fixed.
    /// @param Relative The amount of votes delegated is relative to the total amount of votes the delegator has.
    enum AllowanceType {
        Absolute,
        Relative
    }

    /// @notice Delegation of voting power.
    /// @param delegatee              The address to delegate to.
    /// @param allowanceType          Type of allowance.
    /// @param amount                 Amount of votes delegated. If `allowanceType` is Relative, `amount` acts
    ///                               as a numerator and `DENOMINATOR` as a denominator. For example, 100% of allowance
    ///                               corresponds to 1e4. Otherwise, this is the exact amount of votes delegated.
    struct Delegation {
        AllowanceType allowanceType;
        address delegatee;
        uint256 amount;
    }

    /// @notice Adjustment of delegation.
    /// @param delegatee              The address to delegate to.
    /// @param amount                 Amount of votes delegated.
    struct DelegationAdjustment {
        address delegatee;
        uint208 amount;
    }

    /// @notice Operations for delegation adjustments.
    /// @param ADD      Add votes to the delegatee.
    /// @param SUBTRACT Subtract votes from the delegatee.
    enum Op {
        ADD,
        SUBTRACT
    }

    /// @notice Returns the maximum number of delegations per delegator.
    /// @return MAX_DELEGATIONS The maximum number of delegations.
    function MAX_DELEGATIONS() external view returns (uint256);

    /// @notice Returns the denominator for relative delegations.
    /// @return DENOMINATOR The denominator for relative delegations.
    function DENOMINATOR() external view returns (uint96);

    /// @notice Returns the version of the contract.
    function version() external view returns (string memory);

    /// @notice Returns the migration status of an account.
    /// @param _account The account to check the migration status.
    /// @return _status True if the account has been migrated, false otherwise.
    function migrated(address _account) external view returns (bool _status);

    /// @notice Returns the delegations for a given account.
    /// @param _account The account to get the delegations for.
    /// @return _delegations The delegations.
    function delegations(address _account) external view returns (Delegation[] memory _delegations);

    /// @notice Returns the checkpoints for a given account.
    /// @param _account The account to get the checkpoints for.
    /// @return _checkpoints The checkpoints.
    function checkpoints(address _account) external view returns (ERC20Votes.Checkpoint[] memory _checkpoints);

    /// @notice Returns the number of checkpoints for a account.
    /// @param _account The account to get the total supply checkpoints for.
    /// @return _checkpoints The total supply checkpoints.
    function numCheckpoints(address _account) external view returns (uint32 _checkpoints);

    /// @notice Returns the delegatee with the most voting power for a given account.
    /// @param _account The account to get the delegatee for.
    /// @return _delegatee The delegatee with the most voting power.
    function delegates(address _account) external view returns (address _delegatee);

    /// @notice Returns the number of votes for a given account.
    /// @param _account     The account to get the number of votes for.
    /// @return _votes The number of votes.
    function getVotes(address _account) external view returns (uint256 _votes);

    /// @notice Returns the number of votes for `_account` at the end of `_blockNumber`.
    /// @param _account     The address of the account to get the number of votes for.
    /// @param _blockNumber The block number to get the number of votes for.
    /// @return _votes The number of votes.
    function getPastVotes(address _account, uint256 _blockNumber) external view returns (uint256 _votes);

    /// @notice Returns the total supply at a block.
    /// @param _blockNumber The block number to get the total supply.
    /// @return _totalSupply The total supply of the token for the given block.
    function getPastTotalSupply(uint256 _blockNumber) external view returns (uint256 _totalSupply);

    /// @notice Apply a delegation.
    /// @param _delegation         The delegeation to apply.
    function delegate(Delegation calldata _delegation) external;

    /// @notice Apply a basic delegation from `_delegator` to `_delegatee`.
    /// @param _delegator          The address delegating.
    /// @param _delegatee          The address to delegate to.
    function delegateFromToken(address _delegator, address _delegatee) external;

    /// @notice Apply multiple delegations.
    /// @param _delegations The delegations to apply.
    function delegateBatched(Delegation[] calldata _delegations) external;

    /// @notice Callback called after token transfer in the GovernanceToken contract.
    /// @param _from   The account sending tokens.
    /// @param _to     The account receiving tokens.
    /// @param _amount The amount of tokens being transfered.
    function afterTokenTransfer(address _from, address _to, uint256 _amount) external;

    /// @notice Migrate accounts' delegation state from the GovernanceToken contract to the
    /// GovernanceDelegation contract.
    /// @param _accounts The accounts to migrate.
    function migrateAccounts(address[] calldata _accounts) external;
}
