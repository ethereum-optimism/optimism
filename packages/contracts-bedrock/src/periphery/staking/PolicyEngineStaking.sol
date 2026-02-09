// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Contracts
import { Initializable } from "@openzeppelin/contracts/proxy/utils/Initializable.sol";
import { ReinitializableBase } from "src/universal/ReinitializableBase.sol";
import { ProxyAdminOwnedBase } from "src/L1/ProxyAdminOwnedBase.sol";

// Libraries
import { Storage } from "src/libraries/Storage.sol";

// Interfaces
import { IERC20 } from "@openzeppelin/contracts/token/ERC20/IERC20.sol";
import { SafeERC20 } from "@openzeppelin/contracts/token/ERC20/utils/SafeERC20.sol";
import { ISemver } from "interfaces/universal/ISemver.sol";

/// @title PolicyEngineStaking
/// @notice A periphery contract that enables stake-based transaction ordering in op-rbuilder.
///         Users deposit OP tokens into the contract, and the Policy Engine reads a packed value
///         (effective stake amount and last update timestamp) directly from storage slots to
///         determine transaction priority during block building.
/// @custom:proxied
contract PolicyEngineStaking is Initializable, ReinitializableBase, ProxyAdminOwnedBase, ISemver {
    using SafeERC20 for IERC20;

    /// @notice Semantic version.
    /// @custom:semver 1.0.0
    string public constant override version = "1.0.0";

    /// @notice Storage slot for PE-facing packed stake data (effective stake + timestamp)
    bytes32 public constant PE_DATA_SLOT = bytes32(uint256(keccak256("pe-stakingcontract.peData")) - 1);

    /// @notice Storage slot for contract-side staking data mapping base
    bytes32 public constant STAKING_DATA_SLOT = bytes32(uint256(keccak256("pe-stakingcontract.stakingData")) - 1);

    /// @notice Storage slot for beneficiary-controlled allowlist: beneficiary => (staker => allowed)
    bytes32 public constant BENEFICIARY_ALLOWLIST_SLOT =
        bytes32(uint256(keccak256("pe-stakingcontract.beneficiaryAllowlist")) - 1);

    /// @notice Number of bits allocated for effective stake in packed PE data
    uint256 internal constant PE_STAKE_BITS = 128;

    /// @notice Number of bits allocated for last update timestamp in packed PE data
    uint256 internal constant PE_TIME_BITS = 64;

    /// @notice Bit shift for the timestamp field in packed PE data
    uint256 internal constant PE_TIME_SHIFT = 128;

    /// @notice Mask for extracting effective stake from packed PE data
    uint256 internal constant PE_STAKE_MASK = (1 << PE_STAKE_BITS) - 1;

    /// @notice Mask for extracting timestamp from packed PE data
    uint256 internal constant PE_TIME_MASK = ((1 << PE_TIME_BITS) - 1) << PE_TIME_SHIFT;

    /// @notice Offset for stakedAmount in StakedData struct
    uint256 internal constant SD_OFFSET_STAKED_AMOUNT = 0;

    /// @notice Offset for linkedAmountFromTotal (receivedStake) in StakedData struct
    uint256 internal constant SD_OFFSET_LINKED_AMOUNT = 1;

    /// @notice Offset for linkedAddressTo in StakedData struct
    uint256 internal constant SD_OFFSET_LINKED_TO = 2;

    /// @notice The address of the OP token on OP Mainnet
    /// @custom:network-specific
    address internal constant OP_TOKEN_ADDRESS = 0x4200000000000000000000000000000000000042;

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

    /// @notice Thrown when the amount is zero.
    error PolicyEngineStaking_ZeroAmount();

    /// @notice Thrown when the beneficiary address is zero.
    error PolicyEngineStaking_ZeroBeneficiary();

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

    /// @notice Constructs the PolicyEngineStaking contract.
    constructor() ReinitializableBase(1) {
        _disableInitializers();
    }

    /// @notice Initializes the contract.
    function initialize() external reinitializer(initVersion()) {
        _assertOnlyProxyAdminOrProxyAdminOwner();
    }

    /// @notice Stakes OP tokens and attributes ordering power to a beneficiary.
    /// @param _amount      The amount of OP tokens to stake.
    /// @param _beneficiary Address that receives ordering power from this stake.
    ///                     Use address(0) for self-attribution.
    function stake(uint256 _amount, address _beneficiary) external {
        if (_amount == 0) revert PolicyEngineStaking_ZeroAmount();

        // Get current staking data
        (uint256 stakedAmount, uint256 receivedStake, address linkedTo) = _getStakedData(msg.sender);

        // Check if self-attribution
        bool isSelfAttribution = _beneficiary == address(0);

        // Transfer tokens from caller to contract
        IERC20(OP_TOKEN_ADDRESS).safeTransferFrom(msg.sender, address(this), _amount);

        // If self-attribution, check if already linked to a different beneficiary
        if (isSelfAttribution) {
            // If self-attribution, check if already linked to a different beneficiary
            if (linkedTo != address(0)) revert PolicyEngineStaking_AlreadyLinked();
            // Update PE data
            _increasePeData(msg.sender, _amount);
        } else {
            if (linkedTo == address(0) && stakedAmount > 0) revert PolicyEngineStaking_MustLinkOrUnstakeFirst();
            // If not self-attribution, check if already linked to a different beneficiary
            if (linkedTo != _beneficiary && linkedTo != address(0)) revert PolicyEngineStaking_AlreadyLinked();
            // If not self-attribution, check if staker is allowed to link to the beneficiary
            if (!_isAllowedToLink(_beneficiary, msg.sender)) revert PolicyEngineStaking_NotAllowedToLink();
            // If not self-attribution, add stake to beneficiary
            _addStakeToBeneficiary(_beneficiary, _amount);
            // Update PE data
            _increasePeData(_beneficiary, _amount);
        }

        // Update staking data
        _setStakedData(msg.sender, stakedAmount + _amount, receivedStake, _beneficiary);

        emit Staked(msg.sender, _beneficiary, _amount);
    }

    /// @notice Unstakes all OP tokens from the contract.
    function unstake() external {
        // Get current staking data
        (uint256 stakedAmount, uint256 receivedStake, address linkedTo) = _getStakedData(msg.sender);

        if (stakedAmount == 0) revert PolicyEngineStaking_NoStake();

        // Check if linked
        if (linkedTo == address(0)) {
            // If not linked, update PE data for self
            _decreasePeData(msg.sender, stakedAmount);
        } else {
            // If linked, remove stake from beneficiary's received stake
            _removeStakeFromBeneficiary(linkedTo, stakedAmount);
            // Update PE data for beneficiary
            _decreasePeData(linkedTo, stakedAmount);
        }

        // Clear staking data (stakedAmount = 0, linkedTo = address(0))
        _setStakedData(msg.sender, 0, receivedStake, address(0));

        // Transfer all tokens back to caller
        IERC20(OP_TOKEN_ADDRESS).safeTransfer(msg.sender, stakedAmount);

        emit Unstaked(msg.sender, stakedAmount);
    }

    /// @notice Links the caller's stake to a beneficiary for ordering power.
    /// @param _beneficiary New beneficiary address. Use msg.sender for self-attribution.
    function link(address _beneficiary) external {
        if (_beneficiary == address(0)) revert PolicyEngineStaking_ZeroBeneficiary();

        // Get current staking data
        (uint256 stakedAmount, uint256 receivedStake, address linkedTo) = _getStakedData(msg.sender);

        // Check if staker has no stake
        if (stakedAmount == 0) revert PolicyEngineStaking_NoStake();

        // Check if already linked
        if (linkedTo != address(0)) revert PolicyEngineStaking_AlreadyLinked();

        // Check allowlist if linking to another beneficiary
        if (_beneficiary != msg.sender && !_isAllowedToLink(_beneficiary, msg.sender)) {
            revert PolicyEngineStaking_NotAllowedToLink();
        }

        // Update caller's linked address
        _setStakedData(msg.sender, stakedAmount, receivedStake, _beneficiary);

        // Update beneficiary's received stake and PE data
        _addStakeToBeneficiary(_beneficiary, stakedAmount);
        _increasePeData(_beneficiary, stakedAmount);

        // Decrease caller's PE data (stake moves to beneficiary)
        _decreasePeData(msg.sender, stakedAmount);

        emit Linked(msg.sender, _beneficiary);
    }

    /// @notice Removes the current beneficiary attribution and reverts to self-attribution.
    function unlink() external {
        // Get current staking data
        (uint256 stakedAmount, uint256 receivedStake, address linkedTo) = _getStakedData(msg.sender);

        // Check if linked
        if (linkedTo == address(0)) revert PolicyEngineStaking_NotLinked();

        // Clear linked address
        _setStakedData(msg.sender, stakedAmount, receivedStake, address(0));

        // Update previous beneficiary's received stake and PE data
        _removeStakeFromBeneficiary(linkedTo, stakedAmount);
        _decreasePeData(linkedTo, stakedAmount);

        // Increase caller's PE data (stake returns to self)
        _increasePeData(msg.sender, stakedAmount);

        emit Unlinked(msg.sender, linkedTo);
    }

    /// @notice Returns contract-side raw staking data for an account.
    /// @param _account Address to query.
    /// @return stakedAmount_    Amount staked by _account.
    /// @return receivedStake_   Total stake attributed to _account by other stakers.
    /// @return linkedAddressTo_ Current beneficiary that _account attributes its stake to.
    function getStakedData(address _account)
        external
        view
        returns (uint256 stakedAmount_, uint256 receivedStake_, address linkedAddressTo_)
    {
        return _getStakedData(_account);
    }

    /// @notice Returns the Policy Engine data for an account.
    /// @param _account The account to query.
    /// @return effectiveStake_ The effective stake used by the Policy Engine.
    /// @return lastUpdate_    Last update timestamp.
    function getPEData(address _account) external view returns (uint128 effectiveStake_, uint64 lastUpdate_) {
        return _getPeData(_account);
    }

    /// @notice Returns whether a staker is allowed to attribute stake to a beneficiary.
    /// @param _beneficiary The address that would receive ordering power.
    /// @param _staker      The staker attempting to attribute ordering power.
    /// @return allowed_ Whether the staker is allowed.
    function isAllowedToLink(address _beneficiary, address _staker) external view returns (bool allowed_) {
        return _isAllowedToLink(_beneficiary, _staker);
    }

    /// @notice Allows or denies a staker to attribute ordering power to the caller.
    /// @param _staker  Staker address being allowlisted.
    /// @param _allowed Whether the staker is allowed.
    function setAllowedStaker(address _staker, bool _allowed) external {
        _setAllowlistEntry(msg.sender, _staker, _allowed);
        emit BeneficiaryAllowlistUpdated(msg.sender, _staker, _allowed);
    }

    /// @notice Batch allows or denies stakers to attribute ordering power to the caller.
    /// @param _stakers List of staker addresses.
    /// @param _allowed Whether each staker is allowed.
    function setAllowedStakers(address[] calldata _stakers, bool _allowed) external {
        uint256 length = _stakers.length;
        for (uint256 i = 0; i < length;) {
            _setAllowlistEntry(msg.sender, _stakers[i], _allowed);
            emit BeneficiaryAllowlistUpdated(msg.sender, _stakers[i], _allowed);
            unchecked {
                ++i;
            }
        }
    }

    /// @notice Increases beneficiary's received stake (staking data only, not PE data).
    /// @param _beneficiary The beneficiary address.
    /// @param _amount      The amount to add.
    function _addStakeToBeneficiary(address _beneficiary, uint256 _amount) internal {
        (uint256 beneficiaryStaked, uint256 beneficiaryReceived, address beneficiaryLinkedTo) =
            _getStakedData(_beneficiary);
        beneficiaryReceived += _amount;
        _setStakedData(_beneficiary, beneficiaryStaked, beneficiaryReceived, beneficiaryLinkedTo);
    }

    /// @notice Decreases beneficiary's received stake (staking data only, not PE data).
    /// @param _beneficiary The beneficiary address.
    /// @param _amount      The amount to remove.
    function _removeStakeFromBeneficiary(address _beneficiary, uint256 _amount) internal {
        (uint256 beneficiaryStaked, uint256 beneficiaryReceived, address beneficiaryLinkedTo) =
            _getStakedData(_beneficiary);
        beneficiaryReceived -= _amount;
        _setStakedData(_beneficiary, beneficiaryStaked, beneficiaryReceived, beneficiaryLinkedTo);
    }

    /// @notice Increases PE data (effective stake) for an account.
    /// @param _account The account address.
    /// @param _amount  The amount to add.
    function _increasePeData(address _account, uint256 _amount) internal {
        (uint128 effectiveStake,) = _getPeData(_account);
        _setPeData(_account, effectiveStake + uint128(_amount), uint64(block.timestamp));
    }

    /// @notice Decreases PE data (effective stake) for an account.
    /// @param _account The account address.
    /// @param _amount  The amount to remove.
    function _decreasePeData(address _account, uint256 _amount) internal {
        (uint128 effectiveStake,) = _getPeData(_account);
        _setPeData(_account, effectiveStake - uint128(_amount), uint64(block.timestamp));
    }

    /// @notice Computes the storage slot for PE data of an account.
    /// @param _account The account address.
    /// @return slot_ The storage slot.
    function _peSlot(address _account) internal pure returns (bytes32 slot_) {
        return keccak256(abi.encode(_account, PE_DATA_SLOT));
    }

    /// @notice Computes the base storage slot for staking data of an account.
    /// @param _account The account address.
    /// @return base_ The base storage slot.
    function _sdBase(address _account) internal pure returns (bytes32 base_) {
        return keccak256(abi.encode(_account, STAKING_DATA_SLOT));
    }

    /// @notice Computes the storage slot for an allowlist entry.
    /// @param _beneficiary The beneficiary address.
    /// @param _staker      The staker address.
    /// @return slot_ The storage slot.
    function _allowlistSlot(address _beneficiary, address _staker) internal pure returns (bytes32 slot_) {
        bytes32 outer = keccak256(abi.encode(_beneficiary, BENEFICIARY_ALLOWLIST_SLOT));
        return keccak256(abi.encode(_staker, outer));
    }

    /// @notice Packs effective stake and timestamp into a single uint256.
    /// @param _effectiveStake The effective stake amount.
    /// @param _lastUpdate     The last update timestamp.
    /// @return packed_ The packed value.
    function _packPeData(uint128 _effectiveStake, uint64 _lastUpdate) internal pure returns (uint256 packed_) {
        packed_ = uint256(_effectiveStake) | (uint256(_lastUpdate) << PE_TIME_SHIFT);
    }

    /// @notice Unpacks effective stake and timestamp from a packed uint256.
    /// @param _packed The packed value.
    /// @return effectiveStake_ The effective stake amount.
    /// @return lastUpdate_     The last update timestamp.
    function _unpackPeData(uint256 _packed) internal pure returns (uint128 effectiveStake_, uint64 lastUpdate_) {
        effectiveStake_ = uint128(_packed & PE_STAKE_MASK);
        lastUpdate_ = uint64((_packed & PE_TIME_MASK) >> PE_TIME_SHIFT);
    }

    /// @notice Sets PE data for an account.
    /// @param _account        The account address.
    /// @param _effectiveStake The effective stake amount.
    /// @param _lastUpdate     The last update timestamp.
    function _setPeData(address _account, uint128 _effectiveStake, uint64 _lastUpdate) internal {
        Storage.setUint(_peSlot(_account), _packPeData(_effectiveStake, _lastUpdate));
    }

    /// @notice Gets PE data for an account.
    /// @param _account The account address.
    /// @return effectiveStake_ The effective stake amount.
    /// @return lastUpdate_     The last update timestamp.
    function _getPeData(address _account) internal view returns (uint128 effectiveStake_, uint64 lastUpdate_) {
        return _unpackPeData(Storage.getUint(_peSlot(_account)));
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
        bytes32 base = _sdBase(_account);
        Storage.setUint(bytes32(uint256(base) + SD_OFFSET_STAKED_AMOUNT), _stakedAmount);
        Storage.setUint(bytes32(uint256(base) + SD_OFFSET_LINKED_AMOUNT), _receivedStake);
        Storage.setUint(bytes32(uint256(base) + SD_OFFSET_LINKED_TO), uint256(uint160(_linkedAddressTo)));
    }

    /// @notice Gets staking data for an account.
    /// @param _account The account address.
    /// @return stakedAmount_    The staked amount.
    /// @return receivedStake_   The received stake from others.
    /// @return linkedAddressTo_ The linked beneficiary address.
    function _getStakedData(address _account)
        internal
        view
        returns (uint256 stakedAmount_, uint256 receivedStake_, address linkedAddressTo_)
    {
        bytes32 base = _sdBase(_account);
        stakedAmount_ = Storage.getUint(bytes32(uint256(base) + SD_OFFSET_STAKED_AMOUNT));
        receivedStake_ = Storage.getUint(bytes32(uint256(base) + SD_OFFSET_LINKED_AMOUNT));
        linkedAddressTo_ = address(uint160(Storage.getUint(bytes32(uint256(base) + SD_OFFSET_LINKED_TO))));
    }

    /// @notice Checks if a staker is allowed to link to a beneficiary.
    /// @param _beneficiary The beneficiary address.
    /// @param _staker      The staker address.
    /// @return allowed_ Whether the staker is allowed.
    function _isAllowedToLink(address _beneficiary, address _staker) internal view returns (bool allowed_) {
        return Storage.getBool(_allowlistSlot(_beneficiary, _staker));
    }

    /// @notice Sets an allowlist entry.
    /// @param _beneficiary The beneficiary address.
    /// @param _staker      The staker address.
    /// @param _allowed     Whether the staker is allowed.
    function _setAllowlistEntry(address _beneficiary, address _staker, bool _allowed) internal {
        Storage.setBool(_allowlistSlot(_beneficiary, _staker), _allowed);
    }
}
