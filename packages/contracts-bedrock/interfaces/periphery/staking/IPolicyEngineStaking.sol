// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { IERC20 } from "@openzeppelin/contracts/token/ERC20/IERC20.sol";
import { ISemver } from "interfaces/universal/ISemver.sol";

/// @title IPolicyEngineStaking
/// @notice Interface for the PolicyEngineStaking contract.
interface IPolicyEngineStaking is ISemver {
    /// @notice Emitted when a user stakes OP tokens.
    event Staked(address indexed account, uint128 amount);

    /// @notice Emitted when a user unstakes OP tokens.
    event Unstaked(address indexed account, uint128 amount);

    /// @notice Emitted when a staker sets their beneficiary.
    event BeneficiarySet(address indexed staker, address indexed beneficiary);

    /// @notice Emitted when a staker's beneficiary is removed (on change or full unstake).
    event BeneficiaryRemoved(address indexed staker, address indexed previousBeneficiary);

    /// @notice Emitted when effective stake changes for an account.
    event EffectiveStakeChanged(address indexed account, uint256 newEffectiveStake);

    /// @notice Emitted when a beneficiary updates their allowlist.
    event BeneficiaryAllowlistUpdated(address indexed beneficiary, address indexed staker, bool allowed);

    /// @notice Emitted when staking is paused.
    event Paused();

    /// @notice Emitted when the staking is unpaused.
    event Unpaused();

    /// @notice Emitted when ownership is transferred.
    event OwnershipTransferred(address indexed previousOwner, address indexed newOwner);

    /// @notice Thrown when the caller is not the owner.
    error PolicyEngineStaking_OnlyOwner();

    /// @notice Thrown when the staking is paused.
    error PolicyEngineStaking_Paused();

    /// @notice Thrown when the amount is zero.
    error PolicyEngineStaking_ZeroAmount();

    /// @notice Thrown when the beneficiary address is zero.
    error PolicyEngineStaking_ZeroBeneficiary();

    /// @notice Thrown when the staker is not allowed to set the beneficiary.
    error PolicyEngineStaking_NotAllowedToSetBeneficiary();

    /// @notice Thrown when trying to operate with no stake.
    error PolicyEngineStaking_NoStake();

    /// @notice Thrown when trying to unstake more than the staked amount.
    error PolicyEngineStaking_InsufficientStake();

    /// @notice Thrown when a zero address is provided where it is not allowed.
    error PolicyEngineStaking_ZeroAddress();

    /// @notice Thrown when trying to change beneficiary to the current beneficiary.
    error PolicyEngineStaking_SameBeneficiary();

    /// @notice Thrown when trying to allowlist/disallow yourself.
    error PolicyEngineStaking_SelfAllowlist();

    function __constructor__(address _ownerAddr, address _token) external;

    /// @notice Returns the contract owner.
    function owner() external view returns (address);

    /// @notice Transfers ownership of the contract to a new account. Only callable by owner.
    /// @param _newOwner The address of the new owner.
    function transferOwnership(address _newOwner) external;

    /// @notice Returns whether the contract is paused.
    function paused() external view returns (bool);

    /// @notice Base storage slot for PE data mapping. Policy Engine reads from
    ///         keccak256(abi.encode(account, PE_DATA_SLOT)).
    function PE_DATA_SLOT() external view returns (bytes32);

    /// @notice Returns Policy Engine data for an account.
    function peData(address account) external view returns (uint128 effectiveStake, uint128 lastUpdate);

    /// @notice Returns allowlist entry for a beneficiary-staker pair.
    function allowlist(address beneficiary, address staker) external view returns (bool allowed);

    /// @notice Returns staking data for an account.
    function stakingData(address account) external view returns (uint128 stakedAmount, address beneficiary);

    /// @notice Returns the ERC20 token used for staking.
    function stakingToken() external view returns (IERC20);

    /// @notice Pauses the contract. Only callable by owner.
    function pause() external;

    /// @notice Unpauses the contract. Only callable by owner.
    function unpause() external;

    /// @notice Stakes tokens and sets beneficiary atomically.
    /// @param _amount      The amount of tokens to stake.
    /// @param _beneficiary Address that receives ordering power. Use msg.sender for self-attribution.
    function stake(uint128 _amount, address _beneficiary) external;

    /// @notice Changes the beneficiary for existing stake.
    /// @param _beneficiary New beneficiary address.
    function changeBeneficiary(address _beneficiary) external;

    /// @notice Unstakes OP tokens. Supports partial and full unstake.
    /// @param _amount The amount of OP tokens to unstake.
    function unstake(uint128 _amount) external;

    /// @notice Sets whether a staker can set the caller as beneficiary. When disallowing,
    ///         if the staker's current beneficiary is the caller, their stake attribution is
    ///         moved back to the staker (beneficiary reset to self).
    ///
    /// @param _staker  The staker address.
    /// @param _allowed Whether the staker is allowed to set the caller as beneficiary.
    function setAllowedStaker(address _staker, bool _allowed) external;

    /// @notice Batch sets allowlist for multiple stakers.
    /// @param _stakers Array of staker addresses.
    /// @param _allowed Whether the stakers are allowed to set the caller as beneficiary.
    function setAllowedStakers(address[] calldata _stakers, bool _allowed) external;
}
