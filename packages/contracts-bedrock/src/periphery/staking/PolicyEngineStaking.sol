// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Interfaces
import { IERC20 } from "@openzeppelin/contracts/token/ERC20/IERC20.sol";
import { SafeERC20 } from "@openzeppelin/contracts/token/ERC20/utils/SafeERC20.sol";
import { ISemver } from "interfaces/universal/ISemver.sol";

/// @title PolicyEngineStaking
/// @notice A periphery contract that enables stake-based transaction ordering in op-rbuilder.
///         Users deposit OP tokens into the contract, and the Policy Engine reads a packed value
///         (effective stake amount and last update timestamp) directly from storage slots to
///         determine transaction priority during block building.
contract PolicyEngineStaking is ISemver {
    using SafeERC20 for IERC20;

    /// @notice Semantic version.
    /// @custom:semver 1.0.0
    string public constant override version = "1.0.0";

    /// @notice Base storage slot for PE data mapping. Policy Engine reads from keccak256(abi.encode(account,
    /// PE_DATA_SLOT)). Struct is packed: effectiveStake (128 bits) | lastUpdate (64 bits).
    bytes32 public constant PE_DATA_SLOT = bytes32(uint256(0));

    /// @notice The address of the OP token on OP Mainnet
    /// @custom:network-specific
    address internal constant OP_TOKEN_ADDRESS = 0x4200000000000000000000000000000000000042;

    /// @notice Staking data per account.
    struct StakedData {
        uint256 stakedAmount;
        uint256 receivedStake;
        address linkedTo;
    }

    /// @notice Policy Engine data per account. Packed in one slot for PE reads.
    struct PEData {
        uint128 effectiveStake;
        uint64 lastUpdate;
    }

    /// @notice Slot 0: PE data mapping.
    mapping(address => PEData) public peData;

    /// @notice Staking data mapping.
    mapping(address => StakedData) public stakingData;

    /// @notice Allowlist: beneficiary => staker => entry.
    mapping(address => mapping(address => bool)) public allowlist;

    /// @notice Paused state.
    bool public paused;

    /// @notice The immutable owner of the contract. Can pause and unpause.
    address public immutable owner;

    /// @notice Emitted when a user stakes OP tokens.
    /// @param account     The address that staked tokens.
    /// @param beneficiary The address receiving ordering power.
    /// @param amount      The amount of tokens staked.
    event Staked(address indexed account, address indexed beneficiary, uint256 amount);

    /// @notice Emitted when a user unstakes OP tokens.
    /// @param account The address that unstaked tokens.
    /// @param amount  The amount of tokens unstaked.
    event Unstaked(address indexed account, uint256 amount);

    /// @notice Emitted when a staker links their stake to a beneficiary.
    /// @param staker      The address linking their stake.
    /// @param beneficiary The address receiving ordering power.
    event Linked(address indexed staker, address indexed beneficiary);

    /// @notice Emitted when a staker unlinks and reverts to self-attribution.
    /// @param staker              The address unlinking.
    /// @param previousBeneficiary The previous beneficiary.
    event Unlinked(address indexed staker, address indexed previousBeneficiary);

    /// @notice Emitted when a beneficiary updates their allowlist.
    /// @param beneficiary The address controlling the allowlist.
    /// @param staker      The staker whose permission changed.
    /// @param allowed     The new permission state.
    event BeneficiaryAllowlistUpdated(address indexed beneficiary, address indexed staker, bool allowed);

    /// @notice Emitted when the contract is paused.
    event Paused();

    /// @notice Emitted when the contract is unpaused.
    event Unpaused();

    /// @notice Thrown when the caller is not the owner.
    error PolicyEngineStaking_OnlyOwner();

    /// @notice Thrown when the contract is paused.
    error PolicyEngineStaking_Paused();

    /// @notice Thrown when the amount is zero.
    error PolicyEngineStaking_ZeroAmount();

    /// @notice Thrown when the beneficiary address is zero.
    error PolicyEngineStaking_ZeroBeneficiary();

    /// @notice Thrown when trying to link to self. Use address(0) when staking for self-attribution.
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

    /// @notice Thrown when amount exceeds uint128 max (PE effectiveStake limit).
    error PolicyEngineStaking_AmountExceedsEffectiveStakeLimit();

    /// @notice Constructs the PolicyEngineStaking contract.
    /// @param _owner The address that can pause and unpause the contract.
    constructor(address _owner) {
        owner = _owner;
    }

    /// @notice Modifier that reverts when the contract is paused.
    modifier whenNotPaused() {
        if (paused) revert PolicyEngineStaking_Paused();
        _;
    }

    /// @notice Pauses the contract. Stake and link are disabled while paused.
    function pause() external {
        if (msg.sender != owner) revert PolicyEngineStaking_OnlyOwner();
        paused = true;
        emit Paused();
    }

    /// @notice Unpauses the contract.
    function unpause() external {
        if (msg.sender != owner) revert PolicyEngineStaking_OnlyOwner();
        paused = false;
        emit Unpaused();
    }

    /// @notice Stakes OP tokens and attributes ordering power to a beneficiary.
    /// @param _amount      The amount of OP tokens to stake.
    /// @param _beneficiary Address that receives ordering power from this stake.
    ///                     Use address(0) for self-attribution.
    function stake(uint256 _amount, address _beneficiary) external whenNotPaused {
        if (_amount == 0) revert PolicyEngineStaking_ZeroAmount();
        if (_amount > type(uint128).max) revert PolicyEngineStaking_AmountExceedsEffectiveStakeLimit();
        if (_beneficiary == msg.sender) revert PolicyEngineStaking_CannotLinkToSelf();

        (uint256 stakedAmount, uint256 receivedStake, address linkedTo) = _getStakedData(msg.sender);

        // Check if self-attribution
        bool isSelfAttribution = _beneficiary == address(0);

        // If self-attribution, check if already linked to a different beneficiary
        if (isSelfAttribution) {
            if (linkedTo != address(0)) revert PolicyEngineStaking_AlreadyLinked();
            _increasePeData(msg.sender, _amount);
        } else {
            if (linkedTo == address(0) && stakedAmount > 0) revert PolicyEngineStaking_MustLinkOrUnstakeFirst();
            if (linkedTo != _beneficiary && linkedTo != address(0)) revert PolicyEngineStaking_AlreadyLinked();
            if (!allowlist[_beneficiary][msg.sender]) revert PolicyEngineStaking_NotAllowedToLink();
            _addStakeToBeneficiary(_beneficiary, _amount);
            _increasePeData(_beneficiary, _amount);
        }

        // Update staking data
        _setStakedData(msg.sender, stakedAmount + _amount, receivedStake, _beneficiary);

        // Transfer tokens from caller to contract (interaction last)
        IERC20(OP_TOKEN_ADDRESS).safeTransferFrom(msg.sender, address(this), _amount);

        emit Staked(msg.sender, _beneficiary, _amount);
    }

    /// @notice Unstakes all OP tokens from the contract.
    function unstake() external {
        (uint256 stakedAmount, uint256 receivedStake, address linkedTo) = _getStakedData(msg.sender);

        if (stakedAmount == 0) revert PolicyEngineStaking_NoStake();

        if (linkedTo == address(0)) {
            _decreasePeData(msg.sender, stakedAmount);
        } else {
            _removeStakeFromBeneficiary(linkedTo, stakedAmount);
            _decreasePeData(linkedTo, stakedAmount);
        }

        // Clear staking data (stakedAmount = 0, linkedTo = address(0))
        _setStakedData(msg.sender, 0, receivedStake, address(0));

        // Transfer all tokens back to caller
        IERC20(OP_TOKEN_ADDRESS).safeTransfer(msg.sender, stakedAmount);

        emit Unstaked(msg.sender, stakedAmount);
    }

    /// @notice Links the caller's stake to a beneficiary for ordering power.
    /// @param _beneficiary New beneficiary address.
    function link(address _beneficiary) external whenNotPaused {
        // Check if trying to link to zero address
        if (_beneficiary == address(0)) revert PolicyEngineStaking_ZeroBeneficiary();

        // Check if trying to link to self
        if (_beneficiary == msg.sender) revert PolicyEngineStaking_CannotLinkToSelf();

        (uint256 stakedAmount, uint256 receivedStake, address linkedTo) = _getStakedData(msg.sender);

        // Check if staker has no stake
        if (stakedAmount == 0) revert PolicyEngineStaking_NoStake();

        // Check if already linked
        if (linkedTo != address(0)) revert PolicyEngineStaking_AlreadyLinked();

        // Check if staker is allowed to link to the beneficiary
        if (!allowlist[_beneficiary][msg.sender]) revert PolicyEngineStaking_NotAllowedToLink();

        _setStakedData(msg.sender, stakedAmount, receivedStake, _beneficiary);
        _addStakeToBeneficiary(_beneficiary, stakedAmount);
        _increasePeData(_beneficiary, stakedAmount);
        _decreasePeData(msg.sender, stakedAmount);

        emit Linked(msg.sender, _beneficiary);
    }

    /// @notice Removes the current beneficiary attribution and reverts to self-attribution.
    function unlink() external {
        (uint256 stakedAmount, uint256 receivedStake, address linkedTo) = _getStakedData(msg.sender);

        // Check if linked
        if (linkedTo == address(0)) revert PolicyEngineStaking_NotLinked();

        _setStakedData(msg.sender, stakedAmount, receivedStake, address(0));
        _removeStakeFromBeneficiary(linkedTo, stakedAmount);
        _decreasePeData(linkedTo, stakedAmount);
        _increasePeData(msg.sender, stakedAmount);

        emit Unlinked(msg.sender, linkedTo);
    }

    /// @notice Allows or denies a staker to attribute ordering power to the caller.
    /// @param _staker The staker to allow or deny.
    /// @param _allowed The allowed state.
    function setAllowedStaker(address _staker, bool _allowed) external {
        allowlist[msg.sender][_staker] = _allowed;
        emit BeneficiaryAllowlistUpdated(msg.sender, _staker, _allowed);
    }

    /// @notice Batch allows or denies stakers to attribute ordering power to the caller.
    /// @param _stakers The stakers to allow or deny.
    /// @param _allowed The allowed state.
    function setAllowedStakers(address[] calldata _stakers, bool _allowed) external {
        for (uint256 i; i < _stakers.length;) {
            allowlist[msg.sender][_stakers[i]] = _allowed;
            emit BeneficiaryAllowlistUpdated(msg.sender, _stakers[i], _allowed);
            unchecked {
                ++i;
            }
        }
    }

    /// @notice Increases beneficiary's received stake (staking data only, not PE data).
    /// @param _beneficiary The beneficiary address.
    /// @param _amount The amount to increase.
    function _addStakeToBeneficiary(address _beneficiary, uint256 _amount) internal {
        (uint256 staked, uint256 received, address linkedTo) = _getStakedData(_beneficiary);
        _setStakedData(_beneficiary, staked, received + _amount, linkedTo);
    }

    /// @notice Decreases beneficiary's received stake (staking data only, not PE data).
    /// @param _beneficiary The beneficiary address.
    /// @param _amount The amount to decrease.
    function _removeStakeFromBeneficiary(address _beneficiary, uint256 _amount) internal {
        (uint256 staked, uint256 received, address linkedTo) = _getStakedData(_beneficiary);
        _setStakedData(_beneficiary, staked, received - _amount, linkedTo);
    }

    /// @notice Sets staking data for an account.
    /// @param _account         The account address.
    /// @param _stakedAmount    The staked amount.
    /// @param _receivedStake   The received stake from others.
    /// @param _linkedAddressTo The linked beneficiary address.
    function _setStakedData(
        address _account,
        uint256 _stakedAmount,
        uint256 _receivedStake,
        address _linkedAddressTo
    )
        internal
    {
        stakingData[_account] =
            StakedData({ stakedAmount: _stakedAmount, receivedStake: _receivedStake, linkedTo: _linkedAddressTo });
    }

    /// @notice Gets staking data for an account.
    /// @param _account The account address.
    /// @return The staked amount.
    /// @return The received stake from others.
    /// @return The linked beneficiary address.
    function _getStakedData(address _account) internal view returns (uint256, uint256, address) {
        StakedData memory d = stakingData[_account];
        return (d.stakedAmount, d.receivedStake, d.linkedTo);
    }

    /// @notice Increases PE data (effective stake) and updates last update timestamp for an account.
    /// @param _account The account address.
    /// @param _amount The amount to increase.
    function _increasePeData(address _account, uint256 _amount) internal {
        if (_amount > type(uint128).max) revert PolicyEngineStaking_AmountExceedsEffectiveStakeLimit();
        PEData storage pe = peData[_account];
        pe.effectiveStake += uint128(_amount);
        pe.lastUpdate = uint64(block.timestamp);
    }

    /// @notice Decreases PE data (effective stake) and updates last update timestamp for an account.
    /// @param _account The account address.
    /// @param _amount The amount to decrease.
    function _decreasePeData(address _account, uint256 _amount) internal {
        if (_amount > type(uint128).max) revert PolicyEngineStaking_AmountExceedsEffectiveStakeLimit();
        PEData storage pe = peData[_account];
        pe.effectiveStake -= uint128(_amount);
        pe.lastUpdate = uint64(block.timestamp);
    }
}
