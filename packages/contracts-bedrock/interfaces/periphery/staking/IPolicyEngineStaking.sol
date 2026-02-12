// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

/// @title IPolicyEngineStaking
/// @notice Interface for the PolicyEngineStaking contract.
interface IPolicyEngineStaking {
    /// @notice Emitted when a user stakes OP tokens.
    event Staked(address indexed account, address indexed beneficiary, uint256 amount);

    /// @notice Emitted when a user unstakes OP tokens.
    event Unstaked(address indexed account, uint256 amount);

    /// @notice Emitted when a staker links their stake to a beneficiary.
    event Linked(address indexed staker, address indexed beneficiary);

    /// @notice Emitted when a staker unlinks and reverts to self-attribution.
    event Unlinked(address indexed staker, address indexed previousBeneficiary);

    /// @notice Emitted when a beneficiary updates their allowlist.
    event BeneficiaryAllowlistUpdated(address indexed beneficiary, address indexed staker, bool allowed);

    /// @notice Emitted when staking is paused.
    event Paused();

    /// @notice Emitted when the staking is unpaused.
    event Unpaused();

    /// @notice Thrown when the caller is not the owner.
    error PolicyEngineStaking_OnlyOwner();

    /// @notice Thrown when the staking is paused.
    error PolicyEngineStaking_Paused();

    /// @notice Thrown when the amount is zero.
    error PolicyEngineStaking_ZeroAmount();

    /// @notice Thrown when the beneficiary address is zero.
    error PolicyEngineStaking_ZeroBeneficiary();

    /// @notice Thrown when trying to link to self. Use msg.sender when staking for self-attribution.
    error PolicyEngineStaking_CannotLinkToSelf();

    /// @notice Thrown when the staker is not allowed to link to the beneficiary.
    error PolicyEngineStaking_NotAllowedToLink();

    /// @notice Thrown when trying to link while already linked to a different address.
    error PolicyEngineStaking_AlreadyLinked();

    /// @notice Thrown when trying to unlink but not linked.
    error PolicyEngineStaking_NotLinked();

    /// @notice Thrown when trying to link with no stake.
    error PolicyEngineStaking_NoStake();

    /// @notice Thrown when trying to stake while not linked and having stake.
    error PolicyEngineStaking_MustLinkOrUnstakeFirst();

    /// @notice Thrown when the staker has received stake from another beneficiary.
    error PolicyEngineStaking_StakerHasReceivedStake();

    /// @notice Thrown when staking to a beneficiary who is themselves linked to another (linkers cannot receive stake).
    error PolicyEngineStaking_BeneficiaryIsLinked();

    /// @notice Returns the contract owner.
    function owner() external view returns (address);

    /// @notice Returns whether the contract is paused.
    function paused() external view returns (bool);

    /// @notice Base storage slot for PE data mapping. Policy Engine reads from keccak256(abi.encode(account,
    /// PE_DATA_SLOT)).
    function PE_DATA_SLOT() external view returns (bytes32);

    /// @notice Returns Policy Engine data for an account.
    function peData(address _account) external view returns (uint128 effectiveStake_, uint128 lastUpdate_);

    /// @notice Returns allowlist entry for a beneficiary-staker pair.
    function allowlist(address _beneficiary, address _staker) external view returns (bool allowed_);

    /// @notice Pauses the contract. Only callable by owner.
    function pause() external;

    /// @notice Unpauses the contract. Only callable by owner.
    function unpause() external;

    /// @notice Stakes OP tokens and attributes ordering power to a beneficiary.
    /// @param _amount      The amount of OP tokens to stake.
    /// @param _beneficiary Address that receives ordering power. Use msg.sender for self-attribution.
    function stake(uint256 _amount, address _beneficiary) external;

    /// @notice Unstakes all tokens of the caller from the contract.
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
