// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

/// @title IPolicyEngineStaking
/// @notice Interface for the PolicyEngineStaking contract.
interface IPolicyEngineStaking {
    /// @notice Returns the contract owner.
    function OWNER_ADDRESS() external view returns (address);

    /// @notice Returns whether the contract is paused.
    function paused() external view returns (bool);

    /// @notice Base storage slot for PE data mapping. Policy Engine reads from keccak256(abi.encode(account,
    /// PE_DATA_SLOT)).
    function PE_DATA_SLOT() external view returns (bytes32);

    /// @notice Returns Policy Engine data for an account.
    function peData(address _account) external view returns (uint128 effectiveStake_, uint64 lastUpdate_);

    /// @notice Returns allowlist entry for a beneficiary-staker pair.
    function allowlist(address _beneficiary, address _staker) external view returns (bool allowed_);

    /// @notice Returns the contract version.
    function version() external view returns (string memory);

    /// @notice Pauses the contract. Only callable by owner.
    function pause() external;

    /// @notice Unpauses the contract. Only callable by owner.
    function unpause() external;

    /// @notice Stakes OP tokens and attributes ordering power to a beneficiary.
    /// @param _amount      The amount of OP tokens to stake.
    /// @param _beneficiary Address that receives ordering power. Use address(0) for self-attribution.
    function stake(uint256 _amount, address _beneficiary) external;

    /// @notice Unstakes all OP tokens from the contract.
    function unstake() external;

    /// @notice Links the caller's stake to a beneficiary for ordering power.
    /// @param _beneficiary New beneficiary address.
    function link(address _beneficiary) external;

    /// @notice Removes the current beneficiary attribution and reverts to self-attribution.
    function unlink() external;

    /// @notice Sets whether a staker can link to the caller (beneficiary).
    /// @param _staker  The staker address.
    /// @param _allowed Whether the staker is allowed to link.
    function setAllowedStaker(address _staker, bool _allowed) external;

    /// @notice Batch sets allowlist for multiple stakers.
    /// @param _stakers Array of staker addresses.
    /// @param _allowed Whether the stakers are allowed to link.
    function setAllowedStakers(address[] calldata _stakers, bool _allowed) external;
}
