// SPDX-License-Identifier: MIT
pragma solidity 0.8.25;

// Contracts
import { Ownable } from "@openzeppelin/contracts-v5/access/Ownable.sol";
import { Ownable2Step } from "@openzeppelin/contracts-v5/access/Ownable2Step.sol";
import { Multicallable } from "@solady/utils/Multicallable.sol";

// Libraries
import { BridgeHookHashing } from "src/libraries/BridgeHookHashing.sol";
import { BridgeHookTypes } from "src/libraries/BridgeHookTypes.sol";
import { Predeploys } from "src/libraries/Predeploys.sol";

// Interfaces
import { IComplianceModule } from "interfaces/compliance/IComplianceModule.sol";

/// @title VisionPolicyV1
/// @notice A ComplianceModule policy that screens items against a deny list, an allowlist, a
///         fixed Fee Vault rule and the verdicts its verdict writer records in the module. The
///         rules and their order are fixed. To change a rule, deploy a new policy.
///         multicall preserves msg.sender and applies access checks per call.
///         Any failed call reverts the entire batch.
contract VisionPolicyV1 is Ownable2Step, Multicallable {
    /// @notice The highest value of maxVerdictValidity.
    uint64 internal constant MAX_VERDICT_VALIDITY_CAP = 5 days;

    /// @notice The highest value of holdDelay. The Portal's proof maturity delay can make the
    ///         limit lower.
    uint64 internal constant HOLD_DELAY_CAP = 12 hours;

    /// @notice The module this policy screens for and records verdicts in.
    // nosemgrep: sol-safety-no-immutable-variables
    IComplianceModule internal immutable MODULE;

    /// @notice The only address that can record and revoke verdicts.
    address public verdictWriter;

    /// @notice The maximum validity period, in seconds, for a newly recorded verdict.
    uint64 public maxVerdictValidity;

    /// @notice The time, in seconds, after a withdrawal is finalizable until a withdrawal with
    ///         no verdict is held instead of rejected.
    uint64 public holdDelay;

    /// @notice Senders exempt from entry screening, except for the deny list.
    mapping(address => bool) public allowlisted;

    /// @notice Parties whose new items are held and whose held items cannot complete.
    mapping(address => bool) public denylisted;

    /// @notice Thrown when a withdrawal with no verdict is finalized before its Holdable Time.
    /// @param holdableAt The time from which the withdrawal is held.
    error VisionPolicyV1_NotYetHoldable(uint256 holdableAt);

    /// @notice Thrown when holdDelay is at or above the Portal's proof maturity delay.
    /// @param max The Portal's proof maturity delay.
    error VisionPolicyV1_HoldDelayTooLong(uint64 max);

    /// @notice Thrown when an expiry is not in the future or is more than maxVerdictValidity
    ///         seconds after the current block timestamp.
    error VisionPolicyV1_InvalidExpiry();

    /// @notice Thrown when a parameter is above its cap.
    error VisionPolicyV1_OutOfBounds();

    /// @notice Thrown when the caller is not the module or the verdict writer, as required.
    error VisionPolicyV1_Unauthorized();

    /// @notice Thrown when the module or verdict writer address is zero.
    error VisionPolicyV1_ZeroAddress();

    /// @notice Thrown when any caller tries to renounce ownership.
    error VisionPolicyV1_OwnershipNotRenounceable();

    /// @notice Thrown when the module does not pass exactly two parties.
    error VisionPolicyV1_InvalidPartiesLength();

    /// @notice Emitted when an address's allowlist status is set.
    /// @param account     The address.
    /// @param allowlisted Whether the address is on the allowlist.
    event AllowlistUpdated(address indexed account, bool allowlisted);

    /// @notice Emitted when an address's deny list status is set.
    /// @param account    The address.
    /// @param denylisted Whether the address is on the deny list.
    event DenylistUpdated(address indexed account, bool denylisted);

    /// @notice Emitted when the verdict writer is set.
    /// @param previousVerdictWriter The previous verdict writer.
    /// @param newVerdictWriter      The new verdict writer.
    event VerdictWriterSet(address indexed previousVerdictWriter, address indexed newVerdictWriter);

    /// @notice Emitted when maxVerdictValidity is set.
    /// @param maxVerdictValidity The new value, in seconds.
    event MaxVerdictValiditySet(uint64 maxVerdictValidity);

    /// @notice Emitted when holdDelay is set.
    /// @param holdDelay The new value, in seconds.
    event HoldDelaySet(uint64 holdDelay);

    /// @notice Requires a module configured with its Portal to validate holdDelay.
    /// @param _module             The module this policy serves.
    /// @param _owner              The address that can change the lists, the parameters and the
    ///                            verdict writer.
    /// @param _verdictWriter      The address that can record and revoke verdicts.
    /// @param _maxVerdictValidity The initial maxVerdictValidity, in seconds.
    /// @param _holdDelay          The initial holdDelay, in seconds.
    /// @param _allowlist          The initial allowlist.
    /// @param _denylist           The initial deny list.
    constructor(
        IComplianceModule _module,
        address _owner,
        address _verdictWriter,
        uint64 _maxVerdictValidity,
        uint64 _holdDelay,
        address[] memory _allowlist,
        address[] memory _denylist
    )
        Ownable(_owner)
    {
        if (address(_module) == address(0)) revert VisionPolicyV1_ZeroAddress();

        MODULE = _module;
        _setVerdictWriter(_verdictWriter);
        _setMaxVerdictValidity(_maxVerdictValidity);
        _setHoldDelay(_holdDelay);
        _updateAllowlist(_allowlist, true);
        _updateDenylist(_denylist, true);
    }

    /// @notice Returns the module this policy serves.
    function module() external view returns (IComplianceModule module_) {
        module_ = MODULE;
    }

    /// @notice Screens a deposit at submission. Only the module can call.
    /// @param _parties The sender, then the recipient.
    /// @return pass_ True to proceed, or false to hold.
    function screenDeposit(
        BridgeHookTypes.Item calldata,
        address[] calldata _parties
    )
        external
        view
        returns (bool pass_)
    {
        _assertOnlyModule();
        if (_hasDeniedParty(_parties)) return false;

        pass_ = allowlisted[_parties[0]];
    }

    /// @notice Screens a withdrawal at finalization. Only the module can call.
    /// @param _parties       The sender, then the recipient.
    /// @param _expiresAt     The expiry of the recorded verdict, or zero if there is none.
    /// @param _finalizableAt The time when the withdrawal became finalizable.
    /// @return pass_ True to pay out, or false to hold.
    function screenWithdrawal(
        BridgeHookTypes.Item calldata,
        address[] calldata _parties,
        uint64 _expiresAt,
        uint256 _finalizableAt
    )
        external
        view
        returns (bool pass_)
    {
        _assertOnlyModule();
        if (_hasDeniedParty(_parties)) return false;
        if (allowlisted[_parties[0]] || _isFeeVault(_parties[0])) return true;
        if (_expiresAt > block.timestamp) return true;
        // The writer already answered, so there is no reason to wait.
        if (_expiresAt != 0) return false;

        // Give the writer time to clear the withdrawal without a separate completion transaction.
        uint256 holdableAt = _finalizableAt + holdDelay;
        if (block.timestamp < holdableAt) revert VisionPolicyV1_NotYetHoldable(holdableAt);

        pass_ = false;
    }

    /// @notice Screens a held item at completion. Only the module can call.
    ///         Allowlisted senders and Fee Vaults still need a valid verdict.
    /// @param _parties   The sender, then the recipient.
    /// @param _expiresAt The expiry of the recorded verdict.
    /// @return pass_ True to allow completion; false makes the module revert.
    function screenRelease(
        BridgeHookTypes.Item calldata,
        address[] calldata _parties,
        uint64 _expiresAt
    )
        external
        view
        returns (bool pass_)
    {
        _assertOnlyModule();
        if (_hasDeniedParty(_parties)) return false;

        pass_ = _expiresAt > block.timestamp;
    }

    /// @notice Records a clear verdict in the module. Only the verdict writer can call. The item
    ///         does not need to exist, and an existing verdict is overwritten.
    /// @param _id        The item identifier.
    /// @param _expiresAt The expiry timestamp, in seconds.
    function clear(bytes32 _id, uint64 _expiresAt) external {
        _recordVerdict(_id, _expiresAt);
    }

    /// @notice Removes a verdict from the module. Only the verdict writer can call.
    /// @param _id The item identifier.
    function revoke(bytes32 _id) external {
        _assertOnlyVerdictWriter();
        MODULE.revokeVerdict(_id);
    }

    /// @notice Records a clear verdict for a held deposit, then completes it. Only the verdict
    ///         writer can call.
    /// @param _item      The deposit as the module emitted it in Held.
    /// @param _expiresAt The expiry timestamp, in seconds.
    function clearAndCompleteDeposit(BridgeHookTypes.Item calldata _item, uint64 _expiresAt) external {
        _recordVerdict(BridgeHookHashing.hashItem(_item), _expiresAt);
        MODULE.completeDeposit(_item);
    }

    /// @notice Records a clear verdict for a held withdrawal, then completes it. Only the verdict
    ///         writer can call.
    /// @param _item      The withdrawal as the module emitted it in Held.
    /// @param _expiresAt The expiry timestamp, in seconds.
    function clearAndCompleteWithdrawal(BridgeHookTypes.Item calldata _item, uint64 _expiresAt) external {
        _recordVerdict(BridgeHookHashing.hashItem(_item), _expiresAt);
        MODULE.completeWithdrawal(_item);
    }

    /// @notice Adds senders to the allowlist. Only the owner can call.
    /// @param _accounts The addresses to add.
    function addToAllowlist(address[] calldata _accounts) external onlyOwner {
        _updateAllowlist(_accounts, true);
    }

    /// @notice Removes senders from the allowlist. Only the owner can call.
    /// @param _accounts The addresses to remove.
    function removeFromAllowlist(address[] calldata _accounts) external onlyOwner {
        _updateAllowlist(_accounts, false);
    }

    /// @notice Adds parties to the deny list. Only the owner can call.
    /// @param _accounts The addresses to add.
    function addToDenylist(address[] calldata _accounts) external onlyOwner {
        _updateDenylist(_accounts, true);
    }

    /// @notice Removes parties from the deny list. Only the owner can call.
    /// @param _accounts The addresses to remove.
    function removeFromDenylist(address[] calldata _accounts) external onlyOwner {
        _updateDenylist(_accounts, false);
    }

    /// @notice Sets the verdict writer. Only the owner can call. Recorded verdicts stay in the
    ///         module.
    /// @param _verdictWriter The new verdict writer.
    function setVerdictWriter(address _verdictWriter) external onlyOwner {
        _setVerdictWriter(_verdictWriter);
    }

    /// @notice Sets maxVerdictValidity. Only the owner can call. Recorded verdicts keep their
    ///         expiry.
    /// @param _maxVerdictValidity The new value, in seconds.
    function setMaxVerdictValidity(uint64 _maxVerdictValidity) external onlyOwner {
        _setMaxVerdictValidity(_maxVerdictValidity);
    }

    /// @notice Sets holdDelay. Only the owner can call.
    ///         The delay is read at finalization, not fixed when the withdrawal is proved.
    /// @param _holdDelay The new value, in seconds.
    function setHoldDelay(uint64 _holdDelay) external onlyOwner {
        _setHoldDelay(_holdDelay);
    }

    /// @notice Prevents renunciation so the policy configuration remains editable.
    function renounceOwnership() public pure override {
        revert VisionPolicyV1_OwnershipNotRenounceable();
    }

    function _assertOnlyModule() internal view {
        if (msg.sender != address(MODULE)) revert VisionPolicyV1_Unauthorized();
    }

    function _assertOnlyVerdictWriter() internal view {
        if (msg.sender != verdictWriter) revert VisionPolicyV1_Unauthorized();
    }

    /// @notice Returns whether the sender or the recipient is on the deny list. Reverts unless
    ///         there are exactly two parties.
    /// @param _parties The sender, then the recipient.
    function _hasDeniedParty(address[] calldata _parties) internal view returns (bool) {
        if (_parties.length != 2) revert VisionPolicyV1_InvalidPartiesLength();

        return denylisted[_parties[0]] || denylisted[_parties[1]];
    }

    /// @notice Identifies Fee Vaults so fee withdrawals do not depend on allowlist entries.
    /// @param _account The withdrawal sender.
    function _isFeeVault(address _account) internal pure returns (bool) {
        return _account == Predeploys.SEQUENCER_FEE_WALLET || _account == Predeploys.BASE_FEE_VAULT
            || _account == Predeploys.L1_FEE_VAULT || _account == Predeploys.OPERATOR_FEE_VAULT;
    }

    /// @notice Records a clear verdict in the module. Reverts unless the caller is the verdict
    ///         writer and the expiry is after the current block and within maxVerdictValidity.
    /// @param _id        The item identifier.
    /// @param _expiresAt The expiry timestamp, in seconds.
    function _recordVerdict(bytes32 _id, uint64 _expiresAt) internal {
        _assertOnlyVerdictWriter();
        if (_expiresAt <= block.timestamp || _expiresAt > block.timestamp + maxVerdictValidity) {
            revert VisionPolicyV1_InvalidExpiry();
        }

        MODULE.recordVerdict(_id, _expiresAt);
    }

    /// @notice Sets each address on the allowlist or removes it.
    /// @param _accounts    The addresses.
    /// @param _allowlisted Whether to add or remove them.
    function _updateAllowlist(address[] memory _accounts, bool _allowlisted) internal {
        for (uint256 i = 0; i < _accounts.length; i++) {
            allowlisted[_accounts[i]] = _allowlisted;
            emit AllowlistUpdated(_accounts[i], _allowlisted);
        }
    }

    /// @notice Sets each address on the deny list or removes it.
    /// @param _accounts   The addresses.
    /// @param _denylisted Whether to add or remove them.
    function _updateDenylist(address[] memory _accounts, bool _denylisted) internal {
        for (uint256 i = 0; i < _accounts.length; i++) {
            denylisted[_accounts[i]] = _denylisted;
            emit DenylistUpdated(_accounts[i], _denylisted);
        }
    }

    /// @notice Sets the verdict writer. Reverts if it is the zero address.
    /// @param _verdictWriter The new verdict writer.
    function _setVerdictWriter(address _verdictWriter) internal {
        if (_verdictWriter == address(0)) revert VisionPolicyV1_ZeroAddress();

        address previousVerdictWriter = verdictWriter;
        verdictWriter = _verdictWriter;
        emit VerdictWriterSet(previousVerdictWriter, _verdictWriter);
    }

    /// @notice Sets maxVerdictValidity. Reverts if it is above MAX_VERDICT_VALIDITY_CAP.
    /// @param _maxVerdictValidity The new value, in seconds.
    function _setMaxVerdictValidity(uint64 _maxVerdictValidity) internal {
        if (_maxVerdictValidity > MAX_VERDICT_VALIDITY_CAP) revert VisionPolicyV1_OutOfBounds();

        maxVerdictValidity = _maxVerdictValidity;
        emit MaxVerdictValiditySet(_maxVerdictValidity);
    }

    /// @notice Sets holdDelay. Reverts if it is at or above the Portal's proof maturity delay, or
    ///         above HOLD_DELAY_CAP.
    /// @param _holdDelay The new value, in seconds.
    function _setHoldDelay(uint64 _holdDelay) internal {
        uint256 proofMaturityDelaySeconds = MODULE.portal().proofMaturityDelaySeconds();
        if (_holdDelay >= proofMaturityDelaySeconds) {
            revert VisionPolicyV1_HoldDelayTooLong(uint64(proofMaturityDelaySeconds));
        }
        if (_holdDelay > HOLD_DELAY_CAP) revert VisionPolicyV1_OutOfBounds();

        holdDelay = _holdDelay;
        emit HoldDelaySet(_holdDelay);
    }
}
