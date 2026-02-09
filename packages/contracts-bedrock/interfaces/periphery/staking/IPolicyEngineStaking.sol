// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

/// @title IPolicyEngineStaking
/// @notice Interface for the PolicyEngineStaking contract.
interface IPolicyEngineStaking {
    /// @notice Returns the staked amount, received amount, and linked beneficiary for an account.
    /// @param _account The account to query.
    /// @return staked   Amount the account has staked.
    /// @return received Amount the account has received from linked stakers.
    /// @return linkedTo Address the account is linked to (address(0) if not linked).
    function getStakedData(address _account)
        external
        view
        returns (uint256 staked, uint256 received, address linkedTo);

    /// @notice Returns the Policy Engine data for an account.
    /// @param _account The account to query.
    /// @return effectiveStake The effective stake used by the Policy Engine.
    /// @return lastUpdate    Last update timestamp.
    function getPEData(address _account) external view returns (uint128 effectiveStake, uint64 lastUpdate);

    /// @notice Returns whether a staker is allowed to link to a beneficiary.
    /// @param _beneficiary The beneficiary address.
    /// @param _staker      The staker address.
    /// @return True if the staker is allowed to link to the beneficiary.
    function isAllowedToLink(address _beneficiary, address _staker) external view returns (bool);

    /// @notice Returns the contract version.
    function version() external view returns (string memory);
}
