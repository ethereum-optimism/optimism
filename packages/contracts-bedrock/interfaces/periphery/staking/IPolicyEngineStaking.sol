// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { IERC20 } from "@openzeppelin/contracts/token/ERC20/IERC20.sol";

/// @title IPolicyEngineStaking
/// @notice Interface for the PolicyEngineStaking contract.
interface IPolicyEngineStaking {
    /// @notice Emitted when a user stakes OP tokens.
    event Staked(address indexed account, uint256 amount);

    /// @notice Emitted when a user unstakes OP tokens.
    event Unstaked(address indexed account, uint256 amount);

    /// @notice Emitted when a staker links their stake to a beneficiary.
    event Linked(address indexed staker, address indexed beneficiary);

    /// @notice Emitted when a staker is unlinked from a beneficiary (on re-link or full unstake).
    event Unlinked(address indexed staker, address indexed previousBeneficiary);

    /// @notice Emitted when effective stake changes for an account.
    event EffectiveStakeChanged(address indexed account, uint256 newEffectiveStake);

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

    /// @notice Thrown when the staker is not allowed to link to the beneficiary.
    error PolicyEngineStaking_NotAllowedToLink();

    /// @notice Thrown when trying to operate with no stake.
    error PolicyEngineStaking_NoStake();

    /// @notice Thrown when trying to unstake more than the staked amount.
    error PolicyEngineStaking_InsufficientStake();

    /// @notice The immutable owner of the contract. Can pause and unpause staking.
    function OWNER_ADDRESS() external view returns (address);

    /// @notice Returns the contract owner.
    function owner() external view returns (address);

    /// @notice Returns whether the contract is paused.
    function paused() external view returns (bool);

    /// @notice Base storage slot for PE data mapping. Policy Engine reads from
    ///         keccak256(abi.encode(account, PE_DATA_SLOT)).
    function PE_DATA_SLOT() external view returns (bytes32);

    /// @notice Returns Policy Engine data for an account.
    function peData(address _account) external view returns (uint128 effectiveStake_, uint128 lastUpdate_);

    /// @notice Returns allowlist entry for a beneficiary-staker pair.
    function allowlist(address _beneficiary, address _staker) external view returns (bool allowed_);

    /// @notice Returns staking data for an account.
    function stakingData(address _account) external view returns (uint256 stakedAmount_, address linkedTo_);

    /// @notice Returns the ERC20 token used for staking.
    function STAKING_TOKEN() external view returns (IERC20);

    /// @notice Pauses the contract. Only callable by owner.
    function pause() external;

    /// @notice Unpauses the contract. Only callable by owner.
    function unpause() external;

    /// @notice Stakes tokens and links to a beneficiary atomically.
    /// @param _amount      The amount of tokens to stake.
    /// @param _beneficiary Address that receives ordering power. Use msg.sender for self-attribution.
    function stake(uint256 _amount, address _beneficiary) external;

    /// @notice Re-links existing stake to a new beneficiary.
    /// @param _beneficiary New beneficiary address.
    function changeBeneficiary(address _beneficiary) external;

    /// @notice Unstakes OP tokens. Supports partial and full unstake.
    /// @param _amount The amount of OP tokens to unstake.
    function unstake(uint256 _amount) external;

    /// @notice Sets whether a staker can link to the caller (beneficiary).
    /// @param _staker  The staker address.
    /// @param _allowed Whether the staker is allowed to link.
    function setAllowedStaker(address _staker, bool _allowed) external;

    /// @notice Batch sets allowlist for multiple stakers.
    /// @param _stakers Array of staker addresses.
    /// @param _allowed Whether the stakers are allowed to link.
    function setAllowedStakers(address[] calldata _stakers, bool _allowed) external;
}
