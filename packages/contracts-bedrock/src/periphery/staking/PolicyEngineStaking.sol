// SPDX-License-Identifier: MIT
pragma solidity 0.8.30;

// Interfaces
import { IERC20 } from "@openzeppelin/contracts/token/ERC20/IERC20.sol";

// Libraries
import { SafeERC20 } from "@openzeppelin/contracts/token/ERC20/utils/SafeERC20.sol";

/// @title PolicyEngineStaking
/// @notice A simplified stake-based transaction ordering contract for op-rbuilder.
///         Separates stake and link operations, supports partial unstake, and enforces
///         the invariant that every staked token always has a beneficiary.
///         No `receivedStake` tracking, no dormant state, no `unlink()`.
contract PolicyEngineStaking {
    using SafeERC20 for IERC20;

    /// @notice Staking data per account.
    /// @custom:field stakedAmount The amount of OP tokens staked by the account.
    /// @custom:field linkedTo The address to which the account's stake is attributed.
    struct StakedData {
        uint128 stakedAmount;
        address linkedTo;
    }

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

    /// @notice The ERC20 token used for staking.
    // nosemgrep: sol-safety-no-immutable-variables
    IERC20 public immutable STAKING_TOKEN;

    /// @notice Slot 0: PE data mapping.
    mapping(address account => PEData) public peData;

    /// @notice Allowlist: beneficiary => staker => allowed.
    mapping(address beneficiary => mapping(address staker => bool allowed)) public allowlist;

    /// @notice Staking data mapping.
    mapping(address account => StakedData) public stakingData;

    /// @notice Paused state.
    bool public paused;

    /// @notice The owner of the contract. Can pause, unpause, and transfer ownership.
    address private _owner;

    /// @notice Emitted when a user stakes OP tokens.
    /// @param account The address that staked tokens.
    /// @param amount  The amount of tokens staked.
    event Staked(address indexed account, uint128 amount);

    /// @notice Emitted when a user unstakes OP tokens.
    /// @param account The address that unstaked tokens.
    /// @param amount  The amount of tokens unstaked.
    event Unstaked(address indexed account, uint128 amount);

    /// @notice Emitted when a staker links their stake to a beneficiary.
    /// @param staker      The address linking their stake.
    /// @param beneficiary The address receiving ordering power.
    event Linked(address indexed staker, address indexed beneficiary);

    /// @notice Emitted when a staker is unlinked from a beneficiary (on re-link or full unstake).
    /// @param staker              The address being unlinked.
    /// @param previousBeneficiary The previous beneficiary.
    event Unlinked(address indexed staker, address indexed previousBeneficiary);

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

    /// @notice Emitted when ownership is transferred.
    /// @param previousOwner The address of the previous owner.
    /// @param newOwner      The address of the new owner.
    event OwnershipTransferred(address indexed previousOwner, address indexed newOwner);

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

    /// @notice Thrown when a zero address is provided where it is not allowed.
    error PolicyEngineStaking_ZeroAddress();

    /// @notice Thrown when trying to change beneficiary to the current beneficiary.
    error PolicyEngineStaking_AlreadyLinked();

    /// @notice Constructs the PolicyEngineStaking contract.
    /// @param _ownerAddr The address that can pause and unpause staking.
    /// @param _token The ERC20 token used for staking.
    constructor(address _ownerAddr, address _token) {
        if (_ownerAddr == address(0)) revert PolicyEngineStaking_ZeroAddress();
        if (_token == address(0)) revert PolicyEngineStaking_ZeroAddress();
        _owner = _ownerAddr;
        STAKING_TOKEN = IERC20(_token);
    }

    /// @notice Modifier that reverts when the staking is paused.
    modifier whenNotPaused() {
        if (paused) revert PolicyEngineStaking_Paused();
        _;
    }

    /// @notice Modifier that reverts when the caller is not the owner.
    modifier onlyOwner() {
        if (msg.sender != _owner) revert PolicyEngineStaking_OnlyOwner();
        _;
    }

    /// @notice Returns the owner address.
    function owner() external view returns (address) {
        return _owner;
    }

    /// @notice Transfers ownership of the contract to a new account.
    /// @param _newOwner The address of the new owner.
    function transferOwnership(address _newOwner) external onlyOwner {
        if (_newOwner == address(0)) revert PolicyEngineStaking_ZeroAddress();
        emit OwnershipTransferred(_owner, _newOwner);
        _owner = _newOwner;
    }

    /// @notice Pauses the contract. Stake and changeBeneficiary are disabled while paused.
    function pause() external onlyOwner {
        paused = true;
        emit Paused();
    }

    /// @notice Unpauses the contract.
    function unpause() external onlyOwner {
        paused = false;
        emit Unpaused();
    }

    /// @notice Stakes tokens and links to a beneficiary atomically.
    ///         This is the entry point for staking. Handles first-time staking,
    ///         adding to same beneficiary, and re-linking to a new beneficiary.
    /// @param _amount      The amount of tokens to stake.
    /// @param _beneficiary Address that receives ordering power from this stake.
    ///                     Use msg.sender for self-attribution.
    function stake(uint128 _amount, address _beneficiary) external whenNotPaused {
        if (_amount == 0) revert PolicyEngineStaking_ZeroAmount();
        if (_beneficiary == address(0)) revert PolicyEngineStaking_ZeroBeneficiary();
        if (_beneficiary != msg.sender && !allowlist[_beneficiary][msg.sender]) {
            revert PolicyEngineStaking_NotAllowedToLink();
        }

        StakedData storage data = stakingData[msg.sender];
        address currentLink = data.linkedTo;
        data.stakedAmount += _amount;

        if (currentLink != _beneficiary) {
            if (currentLink != address(0)) {
                _decreasePeData(currentLink, data.stakedAmount - _amount);
                emit Unlinked(msg.sender, currentLink);
            }
            data.linkedTo = _beneficiary;
            emit Linked(msg.sender, _beneficiary);
        }

        uint128 peDelta = currentLink == _beneficiary ? _amount : data.stakedAmount;
        _increasePeData(_beneficiary, peDelta);

        STAKING_TOKEN.safeTransferFrom(msg.sender, address(this), uint256(_amount));

        emit Staked(msg.sender, _amount);
    }

    /// @notice Re-links existing stake to a new beneficiary. No-op if already linked
    ///         to the same beneficiary.
    /// @param _beneficiary New beneficiary address.
    function changeBeneficiary(address _beneficiary) external {
        if (_beneficiary == address(0)) revert PolicyEngineStaking_ZeroBeneficiary();

        StakedData storage data = stakingData[msg.sender];
        if (data.stakedAmount == 0) revert PolicyEngineStaking_NoStake();

        address currentLink = data.linkedTo;
        if (currentLink == _beneficiary) revert PolicyEngineStaking_AlreadyLinked();

        // Move existing stake from old beneficiary to new
        _decreasePeData(currentLink, data.stakedAmount);
        emit Unlinked(msg.sender, currentLink);

        if (_beneficiary != msg.sender && !allowlist[_beneficiary][msg.sender]) {
            revert PolicyEngineStaking_NotAllowedToLink();
        }

        data.linkedTo = _beneficiary;
        _increasePeData(_beneficiary, data.stakedAmount);

        emit Linked(msg.sender, _beneficiary);
    }

    /// @notice Unstakes OP tokens. Supports partial and full unstake.
    ///         On full unstake, the link is automatically cleared.
    /// @param _amount The amount of OP tokens to unstake.
    function unstake(uint128 _amount) external {
        if (_amount == 0) revert PolicyEngineStaking_ZeroAmount();

        StakedData storage data = stakingData[msg.sender];
        if (data.stakedAmount < _amount) revert PolicyEngineStaking_InsufficientStake();

        address linkedTo = data.linkedTo;
        _decreasePeData(linkedTo, _amount);
        data.stakedAmount -= _amount;

        // Auto-unlink on full unstake
        if (data.stakedAmount == 0) {
            data.linkedTo = address(0);
            emit Unlinked(msg.sender, linkedTo);
        }

        STAKING_TOKEN.safeTransfer(msg.sender, uint256(_amount));

        emit Unstaked(msg.sender, _amount);
    }

    /// @notice Allows or denies a staker to attribute ordering power to the caller.
    /// @param _staker The staker to allow or deny.
    /// @param _allowed The allowed state.
    function setAllowedStaker(address _staker, bool _allowed) public {
        allowlist[msg.sender][_staker] = _allowed;
        emit BeneficiaryAllowlistUpdated(msg.sender, _staker, _allowed);
    }

    /// @notice Batch allows or denies stakers to attribute ordering power to the caller.
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

    /// @notice Decreases effective stake for an account and updates timestamp.
    /// @param _account The account address.
    /// @param _amount  The amount to subtract.
    function _decreasePeData(address _account, uint128 _amount) internal {
        PEData storage pe = peData[_account];
        pe.effectiveStake -= _amount;
        pe.lastUpdate = uint128(block.timestamp);
        emit EffectiveStakeChanged(_account, pe.effectiveStake);
    }
}
