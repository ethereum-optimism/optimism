// SPDX-License-Identifier: MIT
pragma solidity 0.8.25;

// Interfaces
import { IERC20 } from "@openzeppelin/contracts/token/ERC20/IERC20.sol";
import { ISemver } from "interfaces/universal/ISemver.sol";

// Libraries
import { SafeERC20 } from "@openzeppelin/contracts/token/ERC20/utils/SafeERC20.sol";

// Inheritance
import { Ownable } from "@openzeppelin/contracts-v5/access/Ownable.sol";
import { Ownable2Step } from "@openzeppelin/contracts-v5/access/Ownable2Step.sol";

/// @title PolicyEngineStakingMapping
/// @notice Holds the `peData` mapping at storage slot 0 so that `op-rbuilder` can read
///         effective-stake data at a known, stable location. Inherited first by
///         `PolicyEngineStaking` to guarantee the slot assignment.
contract PolicyEngineStakingMapping {
    /// @notice Policy Engine data per account. Packed in one slot for PE reads.
    /// @custom:field effectiveStake The exact stake amount used for ordering.
    /// @custom:field lastUpdate The timestamp of the latest change on their effective stake.
    struct PEData {
        uint128 effectiveStake;
        uint128 lastUpdate;
    }

    /// @notice Base storage slot for PE data mapping. Policy Engine reads from
    ///         keccak256(abi.encode(account, PE_DATA_SLOT)).
    bytes32 public constant PE_DATA_SLOT = 0;

    /// @notice Slot 0: PE data mapping.
    mapping(address account => PEData) public peData;
}

/// @title PolicyEngineStaking
/// @notice Periphery contract for stake-based transaction ordering in op-rbuilder. Users stake governance tokens
///         and optionally link to a beneficiary who receives ordering power. Supports partial unstake.
///         Invariant: every staked token has a beneficiary (self or linked). No receivedStake tracking or unlink().
contract PolicyEngineStaking is PolicyEngineStakingMapping, Ownable2Step, ISemver {
    using SafeERC20 for IERC20;

    /// @notice Staking stakingData per account.
    /// @custom:field stakedAmount The amount of OP tokens staked by the account.
    /// @custom:field beneficiary The address to which the account's stake is attributed.
    struct StakedData {
        uint128 stakedAmount;
        address beneficiary;
    }

    /// @notice Semantic version.
    /// @custom:semver 1.0.0
    string public constant version = "1.0.0";

    /// @notice The ERC20 token used for staking.
    // nosemgrep: sol-safety-no-immutable-variables
    IERC20 internal immutable STAKING_TOKEN;

    /// @notice Allowlist: beneficiary => staker => allowed.
    mapping(address beneficiary => mapping(address staker => bool allowed)) public allowlist;

    /// @notice Staking stakingData mapping.
    mapping(address account => StakedData) public stakingData;

    /// @notice Paused state.
    bool public paused;

    /// @notice Emitted when a user stakes OP tokens.
    /// @param account The address that staked tokens.
    /// @param amount  The amount of tokens staked.
    event Staked(address indexed account, uint128 amount);

    /// @notice Emitted when a user unstakes OP tokens.
    /// @param account The address that unstaked tokens.
    /// @param amount  The amount of tokens unstaked.
    event Unstaked(address indexed account, uint128 amount);

    /// @notice Emitted when a staker sets their beneficiary.
    /// @param staker      The address setting their beneficiary.
    /// @param beneficiary The address receiving ordering power.
    event BeneficiarySet(address indexed staker, address indexed beneficiary);

    /// @notice Emitted when a staker's beneficiary is removed (on change or full unstake).
    /// @param staker              The address whose beneficiary was removed.
    /// @param previousBeneficiary The previous beneficiary.
    event BeneficiaryRemoved(address indexed staker, address indexed previousBeneficiary);

    /// @notice Emitted when effective stake changes for an account.
    /// @param account           The account whose effective stake changed.
    /// @param newEffectiveStake The new effective stake value.
    event EffectiveStakeChanged(address indexed account, uint256 newEffectiveStake);

    /// @notice Emitted when a beneficiary updates their allowlist.
    /// @param beneficiary The address controlling the allowlist.
    /// @param staker      The staker whose permission changed.
    /// @param allowed     The new permission state.
    event BeneficiaryAllowlistUpdated(address indexed beneficiary, address indexed staker, bool allowed);

    /// @notice Emitted when staking is paused.
    event Paused();

    /// @notice Emitted when the staking is unpaused.
    event Unpaused();

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

    /// @notice Constructs the PolicyEngineStaking contract.
    /// @param _ownerAddr The address that can pause and unpause staking.
    /// @param _token The ERC20 token used for staking.
    constructor(address _ownerAddr, address _token) Ownable(_ownerAddr) {
        if (_token == address(0)) revert PolicyEngineStaking_ZeroAddress();
        STAKING_TOKEN = IERC20(_token);
    }

    /// @notice Modifier that reverts when the staking is paused.
    modifier whenNotPaused() {
        if (paused) revert PolicyEngineStaking_Paused();
        _;
    }

    /// @notice Returns the staking token address.
    ///
    /// @return The ERC20 token used for staking.
    function stakingToken() external view returns (IERC20) {
        return STAKING_TOKEN;
    }

    /// @notice Pauses the contract. Stake is disabled while paused.
    function pause() external onlyOwner {
        paused = true;
        emit Paused();
    }

    /// @notice Unpauses the contract.
    function unpause() external onlyOwner {
        paused = false;
        emit Unpaused();
    }

    /// @notice Stakes tokens and sets beneficiary atomically.
    ///         This is the entry point for staking. Handles first-time staking,
    ///         adding to same beneficiary, and changing to a new beneficiary.
    /// @param _amount      The amount of tokens to stake.
    /// @param _beneficiary Address that receives ordering power from this stake.
    ///                     Use msg.sender for self-attribution.
    function stake(uint128 _amount, address _beneficiary) external whenNotPaused {
        if (_amount == 0) revert PolicyEngineStaking_ZeroAmount();
        if (_beneficiary == address(0)) revert PolicyEngineStaking_ZeroBeneficiary();
        if (_beneficiary != msg.sender && !allowlist[_beneficiary][msg.sender]) {
            revert PolicyEngineStaking_NotAllowedToSetBeneficiary();
        }

        StakedData storage stakedData = stakingData[msg.sender];
        address currentBeneficiary = stakedData.beneficiary;

        // Remove previous beneficiary
        if (currentBeneficiary != _beneficiary) {
            if (currentBeneficiary != address(0)) {
                _decreasePeData(currentBeneficiary, stakedData.stakedAmount);
                emit BeneficiaryRemoved(msg.sender, currentBeneficiary);
            }
            stakedData.beneficiary = _beneficiary;
            emit BeneficiarySet(msg.sender, _beneficiary);
        }

        stakedData.stakedAmount += _amount;

        // If the beneficiary hasn't changed, peDelta is just the new amount staked.
        // If the beneficiary changed, peDelta is the full total stake amount (previous + new stake),
        // since the new beneficiary now receives ordering power for the entire position.
        uint128 peDelta = currentBeneficiary == _beneficiary ? _amount : stakedData.stakedAmount;
        _increasePeData(_beneficiary, peDelta);

        STAKING_TOKEN.safeTransferFrom(msg.sender, address(this), uint256(_amount));

        emit Staked(msg.sender, _amount);
    }

    /// @notice Changes the beneficiary for existing stake. Reverts if already set
    ///         to the same beneficiary.
    /// @param _beneficiary New beneficiary address.
    function changeBeneficiary(address _beneficiary) external whenNotPaused {
        if (_beneficiary == address(0)) revert PolicyEngineStaking_ZeroBeneficiary();
        if (_beneficiary != msg.sender && !allowlist[_beneficiary][msg.sender]) {
            revert PolicyEngineStaking_NotAllowedToSetBeneficiary();
        }

        StakedData storage stakedData = stakingData[msg.sender];
        if (stakedData.stakedAmount == 0) revert PolicyEngineStaking_NoStake();

        address currentBeneficiary = stakedData.beneficiary;
        if (currentBeneficiary == _beneficiary) revert PolicyEngineStaking_SameBeneficiary();

        // Move existing stake from old beneficiary to new
        _decreasePeData(currentBeneficiary, stakedData.stakedAmount);
        emit BeneficiaryRemoved(msg.sender, currentBeneficiary);

        stakedData.beneficiary = _beneficiary;
        _increasePeData(_beneficiary, stakedData.stakedAmount);

        emit BeneficiarySet(msg.sender, _beneficiary);
    }

    /// @notice Unstakes OP tokens. Supports partial and full unstake.
    ///         On full unstake, the beneficiary is automatically cleared.
    /// @param _amount The amount of OP tokens to unstake.
    function unstake(uint128 _amount) external {
        if (_amount == 0) revert PolicyEngineStaking_ZeroAmount();

        StakedData storage stakedData = stakingData[msg.sender];
        if (stakedData.stakedAmount < _amount) revert PolicyEngineStaking_InsufficientStake();

        address beneficiary = stakedData.beneficiary;
        _decreasePeData(beneficiary, _amount);
        stakedData.stakedAmount -= _amount;

        // Auto-clear beneficiary on full unstake
        if (stakedData.stakedAmount == 0) {
            stakedData.beneficiary = address(0);
            emit BeneficiaryRemoved(msg.sender, beneficiary);
        }

        STAKING_TOKEN.safeTransfer(msg.sender, uint256(_amount));

        emit Unstaked(msg.sender, _amount);
    }

    /// @notice Sets whether a staker can set the caller as beneficiary. When disallowing,
    ///         if the staker's current beneficiary is the caller, their stake attribution is
    ///         moved back to the staker (beneficiary reset to self).
    /// @dev    This function is intentionally NOT gated by `whenNotPaused`. Allowlist
    ///         mutations remain available during pause so that beneficiaries can revoke
    ///         stakers at any time. Note that disallowing a staker who is currently
    ///         delegated to the caller will move effective stake attribution back to the
    ///         staker, changing ordering-power state even while the contract is paused.
    /// @dev    Trust assumption: stakers who delegate to a beneficiary implicitly trust
    ///         that the beneficiary will not remove them from the allowlist at a
    ///         disadvantageous time. Removal triggers `_increasePeData` on the staker,
    ///         which resets their `lastUpdate` and thus their accumulated staking weight.
    /// @param _staker The staker to allow or deny.
    /// @param _allowed The allowed state.
    function setAllowedStaker(address _staker, bool _allowed) public {
        if (_staker == msg.sender) revert PolicyEngineStaking_SelfAllowlist();

        allowlist[msg.sender][_staker] = _allowed;
        emit BeneficiaryAllowlistUpdated(msg.sender, _staker, _allowed);

        if (!_allowed) {
            StakedData storage stakedData = stakingData[_staker];
            if (stakedData.beneficiary == msg.sender) {
                _decreasePeData(msg.sender, stakedData.stakedAmount);
                emit BeneficiaryRemoved(_staker, msg.sender);

                stakedData.beneficiary = _staker;
                _increasePeData(_staker, stakedData.stakedAmount);
                emit BeneficiarySet(_staker, _staker);
            }
        }
    }

    /// @notice Batch sets allowlist for multiple stakers.
    /// @param _stakers The stakers to allow or deny.
    /// @param _allowed The allowed state.
    function setAllowedStakers(address[] calldata _stakers, bool _allowed) external {
        uint256 stakersLength = _stakers.length;

        for (uint256 i; i < stakersLength; ++i) {
            setAllowedStaker(_stakers[i], _allowed);
        }
    }

    /// @notice Increases effective stake for an account and updates timestamp.
    /// @param _account The account address.
    /// @param _amount  The amount to add.
    function _increasePeData(address _account, uint128 _amount) internal {
        PEData storage pe = peData[_account];
        pe.effectiveStake += _amount;
        pe.lastUpdate = uint128(block.timestamp);
        emit EffectiveStakeChanged(_account, pe.effectiveStake);
    }

    /// @notice Decreases effective stake for an account. Only resets `lastUpdate` when
    ///         the effective stake reaches zero to avoid stale timestamps; otherwise the
    ///         existing timestamp is preserved so remaining stake keeps its staking weight.
    ///
    /// @param _account The account address.
    /// @param _amount  The amount to subtract.
    function _decreasePeData(address _account, uint128 _amount) internal {
        PEData storage pe = peData[_account];
        pe.effectiveStake -= _amount;
        if (pe.effectiveStake == 0) {
            pe.lastUpdate = uint128(block.timestamp);
        }
        emit EffectiveStakeChanged(_account, pe.effectiveStake);
    }
}
