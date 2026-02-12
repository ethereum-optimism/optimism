// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Interfaces
import { IERC20 } from "@openzeppelin/contracts/token/ERC20/IERC20.sol";

// Libraries
import { SafeERC20 } from "@openzeppelin/contracts/token/ERC20/utils/SafeERC20.sol";
import { Predeploys } from "src/libraries/Predeploys.sol";

/// @title PolicyEngineStaking
/// @notice A periphery contract that enables stake-based transaction ordering in op-rbuilder.
///         Users deposit OP tokens into the contract, and the Policy Engine reads a packed value
///         (effective stake amount and last update timestamp) directly from storage slots to
///         determine transaction priority during block building.
contract PolicyEngineStaking {
    using SafeERC20 for IERC20;

    /// @notice Staking data per account.
    /// @custom:field stakedAmount The amount of OP tokens staked by the account.
    /// @custom:field receivedStake The amount of OP tokens received by the account from others.
    /// @custom:field linkedTo The address to which the account is linked.
    struct StakedData {
        uint256 stakedAmount;
        uint256 receivedStake;
        address linkedTo;
    }

    /// @notice Policy Engine data per account. Packed in one slot for PE reads.
    /// @custom:field effectiveStake The amount of OP tokens that the account has contributed to the Policy Engine.
    /// @custom:field lastUpdate The timestamp of the last update to the Policy Engine data.
    struct PEData {
        uint128 effectiveStake;
        uint128 lastUpdate;
    }

    /// @notice Operation for updating a balance (add or subtract).
    /// @param DECREASE Decrease the balance.
    /// @param INCREASE Increase the balance.
    enum UpdateOperation {
        DECREASE,
        INCREASE
    }

    /// @notice Base storage slot for PE data mapping. Policy Engine reads from keccak256(abi.encode(account,
    /// PE_DATA_SLOT)). Struct is packed: effectiveStake (128 bits) | lastUpdate (128 bits).
    bytes32 public constant PE_DATA_SLOT = bytes32(uint256(0));

    /// @notice The immutable owner of the contract. Can pause and unpause staking.
    address internal immutable OWNER_ADDRESS;

    /// @notice Slot 0: PE data mapping.
    mapping(address => PEData) public peData;

    /// @notice Allowlist: beneficiary => staker => entry.
    mapping(address => mapping(address => bool)) public allowlist;

    /// @notice Staking data mapping.
    mapping(address => StakedData) internal _stakingData;

    /// @notice Paused state.
    bool public paused;

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

    /// @notice Constructs the PolicyEngineStaking contract.
    /// @param _owner The address that can pause and unpause staking.
    constructor(address _owner) {
        OWNER_ADDRESS = _owner;
    }

    /// @notice Modifier that reverts when the staking is paused.
    modifier whenNotPaused() {
        if (paused) revert PolicyEngineStaking_Paused();
        _;
    }

    /// @notice Modifier that reverts when the caller is not the owner.
    modifier onlyOwner() {
        if (msg.sender != OWNER_ADDRESS) revert PolicyEngineStaking_OnlyOwner();
        _;
    }

    /// @notice Returns the owner address.
    function owner() external view returns (address) {
        return OWNER_ADDRESS;
    }

    /// @notice Pauses the contract. Stake and link are disabled while paused.
    function pause() external onlyOwner {
        paused = true;
        emit Paused();
    }

    /// @notice Unpauses the contract.
    function unpause() external onlyOwner {
        paused = false;
        emit Unpaused();
    }

    /// @notice Stakes OP tokens and attributes ordering power to a beneficiary.
    /// @param _amount      The amount of OP tokens to stake.
    /// @param _beneficiary Address that receives ordering power from this stake.
    ///                     Use msg.sender for self-attribution.
    function stake(uint256 _amount, address _beneficiary) external whenNotPaused {
        if (_amount == 0) revert PolicyEngineStaking_ZeroAmount();
        // Check if trying to link to zero address
        if (_beneficiary == address(0)) revert PolicyEngineStaking_ZeroBeneficiary();

        // Get staking data of msg.sender
        StakedData storage stakingData = _stakingData[msg.sender];

        // Check if self-attribution
        bool isSelfAttribution = _beneficiary == msg.sender;

        // If self-attribution, check if already linked to a different beneficiary
        if (isSelfAttribution) {
            // It is not allowed to self-attribute and be linked to a different beneficiary.
            if (stakingData.linkedTo != address(0)) revert PolicyEngineStaking_AlreadyLinked();
            // Update PE data of the staker
            _updatePeData(msg.sender, _amount, UpdateOperation.INCREASE);
        } else {
            // If not self-attribution, check if already staked and not linked to a beneficiary.
            if (stakingData.linkedTo == address(0) && stakingData.stakedAmount > 0) {
                revert PolicyEngineStaking_MustLinkOrUnstakeFirst();
            }
            // Check if already linked to a different beneficiary.
            if (stakingData.linkedTo != _beneficiary && stakingData.linkedTo != address(0)) {
                revert PolicyEngineStaking_AlreadyLinked();
            }
            // Attribute stake to the beneficiary
            _attributeToBeneficiary(msg.sender, _beneficiary, stakingData.receivedStake, _amount, true);
        }

        // Update staking data
        stakingData.stakedAmount += _amount;
        // Update linked beneficiary
        stakingData.linkedTo = isSelfAttribution ? address(0) : _beneficiary;

        // Transfer tokens from caller to contract (interaction last)
        IERC20(Predeploys.GOVERNANCE_TOKEN).safeTransferFrom(msg.sender, address(this), _amount);

        emit Staked(msg.sender, _beneficiary, _amount);
    }

    /// @notice Unstakes all tokens of the caller from the contract.
    function unstake() external {
        // Get staking data of msg.sender
        StakedData storage stakingData = _stakingData[msg.sender];

        // Check if staker has no stake
        if (stakingData.stakedAmount == 0) revert PolicyEngineStaking_NoStake();

        // Get amount of tokens to unstake
        uint256 amount = stakingData.stakedAmount;
        // Get linked beneficiary
        address linkedTo = stakingData.linkedTo;

        // If not linked, decrease PE data and return tokens to the staker
        if (linkedTo == address(0)) {
            _updatePeData(msg.sender, amount, UpdateOperation.DECREASE);
        } else {
            // If linked, unlink and decrease PE data and received stake of the beneficiary
            _stakingData[linkedTo].receivedStake = _stakingData[linkedTo].receivedStake - amount;
            _updatePeData(linkedTo, amount, UpdateOperation.DECREASE);
        }

        // Update staking data
        stakingData.stakedAmount = 0;
        stakingData.linkedTo = address(0);

        // Transfer all tokens back to caller
        IERC20(Predeploys.GOVERNANCE_TOKEN).safeTransfer(msg.sender, amount);

        emit Unstaked(msg.sender, amount);
    }

    /// @notice Links the caller's stake to a beneficiary for ordering power.
    /// @param _beneficiary New beneficiary address.
    function link(address _beneficiary) external whenNotPaused {
        // Check if trying to link to zero address
        if (_beneficiary == address(0)) revert PolicyEngineStaking_ZeroBeneficiary();

        // Check if trying to link to self
        if (_beneficiary == msg.sender) revert PolicyEngineStaking_CannotLinkToSelf();

        StakedData storage stakingData = _stakingData[msg.sender];

        // Check if staker has no stake
        if (stakingData.stakedAmount == 0) revert PolicyEngineStaking_NoStake();

        // Check if already linked to a different beneficiary. As in stake we not allow to do a self-attribution and be
        // linked to a different beneficiary, so we can just check if linked to a different beneficiary.
        if (stakingData.linkedTo != address(0)) revert PolicyEngineStaking_AlreadyLinked();

        // Update linked beneficiary
        stakingData.linkedTo = _beneficiary;
        // Attribute stake to the beneficiary
        _attributeToBeneficiary(msg.sender, _beneficiary, stakingData.receivedStake, stakingData.stakedAmount, false);

        emit Linked(msg.sender, _beneficiary);
    }

    /// @notice Removes the current beneficiary attribution and reverts to self-attribution.
    function unlink() external {
        StakedData storage stakingData = _stakingData[msg.sender];

        // Check if linked
        if (stakingData.linkedTo == address(0)) revert PolicyEngineStaking_NotLinked();

        // Get amount of tokens to unstake
        uint256 amount = stakingData.stakedAmount;
        address linkedTo = stakingData.linkedTo;

        // Update linked beneficiary. receivedStake is 0 for linkers (BeneficiaryIsLinked prevents
        // anyone from staking to a linker).
        stakingData.linkedTo = address(0);
        stakingData.receivedStake = 0;

        // Decrease PE data and received stake of the linked beneficiary
        _updatePeData(linkedTo, amount, UpdateOperation.DECREASE);
        _stakingData[linkedTo].receivedStake = _stakingData[linkedTo].receivedStake - amount;

        // Increase PE data of the staker
        _updatePeData(msg.sender, amount, UpdateOperation.INCREASE);

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

    /// @notice Attributes stake to a beneficiary: updates PE data and beneficiary's receivedStake.
    /// @param _staker      The account whose stake is being attributed.
    /// @param _beneficiary The beneficiary receiving the attribution.
    /// @param _amount      The amount to attribute.
    /// @param _isNewStake  If true, new tokens (no decrease on staker); if false, moving existing stake.
    function _attributeToBeneficiary(
        address _staker,
        address _beneficiary,
        uint256 _receivedStake,
        uint256 _amount,
        bool _isNewStake
    )
        internal
    {
        // Beneficiary must not be a linker; linkers cannot receive stake.
        if (_stakingData[_beneficiary].linkedTo != address(0)) {
            revert PolicyEngineStaking_BeneficiaryIsLinked();
        }
        // If the staker has received stake from another beneficiary , cannot link to a different beneficiary.
        if (_receivedStake > 0) revert PolicyEngineStaking_StakerHasReceivedStake();

        // Check if the staker is allowed to link to the beneficiary.
        if (!allowlist[_beneficiary][_staker]) revert PolicyEngineStaking_NotAllowedToLink();

        // Update beneficiary's received stake
        _stakingData[_beneficiary].receivedStake = _stakingData[_beneficiary].receivedStake + _amount;
        _updatePeData(_beneficiary, _amount, UpdateOperation.INCREASE);
        if (!_isNewStake) {
            _updatePeData(_staker, _amount, UpdateOperation.DECREASE);
        }
    }

    /// @notice Updates PE data (effective stake) and last update timestamp for an account.
    /// @param _account The account address.
    /// @param _amount The amount to add or subtract.
    /// @param _direction Increase or Decrease.
    function _updatePeData(address _account, uint256 _amount, UpdateOperation _direction) internal {
        PEData storage pe = peData[_account];

        // If increasing, add the amount to the effective stake.
        // If decreasing, subtract the amount from the effective stake.
        pe.effectiveStake = _direction == UpdateOperation.INCREASE
            ? pe.effectiveStake + uint128(_amount)
            : pe.effectiveStake - uint128(_amount);

        // Update the last update timestamp.
        pe.lastUpdate = uint128(block.timestamp);
    }
}
