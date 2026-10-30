// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Libraries
import { SafeERC20 } from "@openzeppelin/contracts/token/ERC20/utils/SafeERC20.sol";
import { BridgeHookTypes } from "src/libraries/BridgeHookTypes.sol";
import { BridgeHookHashing } from "src/libraries/BridgeHookHashing.sol";
import { BridgeHookDecoder } from "src/libraries/BridgeHookDecoder.sol";

// Interfaces
import { IERC20 } from "@openzeppelin/contracts/token/ERC20/IERC20.sol";
import { IComplianceModule } from "interfaces/compliance/IComplianceModule.sol";
import { IPolicy } from "interfaces/compliance/IPolicy.sol";
import { IL1CrossDomainMessenger } from "interfaces/L1/IL1CrossDomainMessenger.sol";
import { IL1StandardBridge } from "interfaces/L1/IL1StandardBridge.sol";
import { IOptimismPortal2 } from "interfaces/L1/IOptimismPortal2.sol";
import { ISystemConfig } from "interfaces/L1/ISystemConfig.sol";

/// @custom:proxied false
/// @title MockComplianceModule
/// @notice Devnet-only bridge hook for the bridge-screener integration. It takes custody of
///         held items, owns the verdict registry, and initiates completion, exactly as the real
///         module will. What is acceptable is decided entirely by the Policy behind it.
/// @dev    NOT FOR PRODUCTION. Faithful to the interface and the event shapes, but unaudited
///         and holding real value on whatever chain it is deployed to.
///
///         What a completion actually does depends on which Portal is behind the proxy.
///         Against the stock Portal, whose completion entry points are still empty stubs, it
///         emits Completed and changes nothing there. Against the mock Portal, or the real one
///         once its hook paths land, it runs the item on its committed terms. The module's own
///         bookkeeping is identical either way, so nothing here changes when they do.
contract MockComplianceModule is IComplianceModule {
    using SafeERC20 for IERC20;

    /// @notice The module's entry for one item.
    /// @custom:field heldAt    Non-zero means the item exists and is held.
    /// @custom:field expiresAt Non-zero means a clear verdict is recorded, and when the policy
    ///                         says it expires.
    struct Record {
        uint64 heldAt;
        uint64 expiresAt;
    }

    /// @notice Portal this module is the hook for.
    IOptimismPortal2 public immutable portal;

    /// @notice Address allowed to set the policy and to disable or enable the module.
    address public owner;

    /// @notice Policy this module consults and accepts verdicts from.
    IPolicy public policy;

    /// @notice Whether the emergency disable switch is set.
    bool public disabled;

    /// @notice L2 withdrawal nonce recorded when the module was last disabled. Withdrawals
    ///         initiated at or after it skip screening; earlier ones are still screened.
    uint256 public cutoffL2Nonce;

    /// @notice Records, keyed by item identifier.
    mapping(bytes32 => Record) internal _items;

    /// @notice Amount held for an L1 token and L2 token pair.
    mapping(address => mapping(address => uint256)) public heldTokens;

    /// @notice Thrown when the Portal no longer has this module as its hook.
    error MockComplianceModule_NotCurrentHook();

    /// @notice Thrown when the caller is not the owner.
    error MockComplianceModule_NotOwner();

    /// @notice Thrown when a verdict's expiry could never let an item move.
    error MockComplianceModule_InvalidExpiry();

    /// @param _portal Portal this module is the hook for.
    /// @param _owner  Address allowed to set the policy and to disable or enable.
    constructor(IOptimismPortal2 _portal, address _owner) {
        if (address(_portal) == address(0) || _owner == address(0)) revert ComplianceModule_ZeroAddress();
        portal = _portal;
        owner = _owner;
    }

    /// @notice Accepts the ETH of a held item, which the Portal forwards with its hold call.
    receive() external payable { }

    /// @notice Reverts unless the caller is the Portal and this module is still its hook. The
    ///         second check matters because a removed hook becomes an ordinary withdrawal
    ///         target, and the Portal would still be the caller.
    modifier onlyPortal() {
        if (msg.sender != address(portal)) revert ComplianceModule_CallerNotPortal();
        if (address(portal.bridgeHook()) != address(this)) revert MockComplianceModule_NotCurrentHook();
        _;
    }

    modifier onlyPolicy() {
        if (msg.sender != address(policy)) revert ComplianceModule_CallerNotPolicy();
        _;
    }

    modifier onlyOwner() {
        if (msg.sender != owner) revert MockComplianceModule_NotOwner();
        _;
    }

    /// @notice The L1StandardBridge, read from the Portal's SystemConfig.
    function bridge() public view returns (IL1StandardBridge) {
        return IL1StandardBridge(payable(_systemConfig().l1StandardBridge()));
    }

    /// @notice The L1CrossDomainMessenger, read from the Portal's SystemConfig.
    function l1CrossDomainMessenger() public view returns (IL1CrossDomainMessenger) {
        return IL1CrossDomainMessenger(_systemConfig().l1CrossDomainMessenger());
    }

    /// @notice The pause state gating completion of held withdrawals.
    function paused() public view returns (bool) {
        return portal.paused();
    }

    /// @notice Returns the record for an identifier. A zero record means the item has never
    ///         been held or cleared.
    function items(bytes32 _id) external view returns (uint64 heldAt_, uint64 expiresAt_) {
        Record memory record = _items[_id];
        return (record.heldAt, record.expiresAt);
    }

    /// @notice The addresses the policy screens for an item. For a message sent through the
    ///         messengers this is the real sender and recipient inside the envelope, not the
    ///         messengers themselves.
    /// @param _item The item.
    /// @return parties_ The initiator followed by the recipient.
    function parties(BridgeHookTypes.Item calldata _item) public view returns (address[] memory parties_) {
        (address initiator, address recipient) =
            BridgeHookDecoder.resolveParties(_item, address(l1CrossDomainMessenger()), address(bridge()));
        parties_ = new address[](2);
        parties_[0] = initiator;
        parties_[1] = recipient;
    }

    /// @notice Answers the Portal for a deposit at submission.
    /// @param _item The deposit.
    /// @return pass_ True to proceed, false to hold. A policy revert refuses the deposit.
    function screenDeposit(BridgeHookTypes.Item calldata _item) external onlyPortal returns (bool pass_) {
        if (disabled) return true;
        pass_ = policy.screen(_item, parties(_item));
    }

    /// @notice Answers the Portal for a withdrawal at finalization.
    /// @param _item          The withdrawal.
    /// @param _finalizableAt When the withdrawal became finalizable.
    /// @return pass_ True to proceed, false to hold. A policy revert refuses the finalization.
    function screenWithdrawal(
        BridgeHookTypes.Item calldata _item,
        uint256 _finalizableAt
    )
        external
        onlyPortal
        returns (bool pass_)
    {
        // Withdrawals initiated before the cutoff are still screened, because the disable is
        // meant to protect new items rather than release the ones already in flight.
        if (disabled && _item.nonce >= cutoffL2Nonce) return true;

        bytes32 id = BridgeHookHashing.hashItem(_item);
        pass_ = policy.screenRelease(_item, parties(_item), _items[id].expiresAt, _finalizableAt);
    }

    /// @notice Takes custody of a held deposit.
    /// @param _item The deposit, as the Portal committed it.
    function holdDeposit(BridgeHookTypes.Item calldata _item) external payable onlyPortal {
        _hold(_item);
    }

    /// @notice Takes custody of a held withdrawal.
    /// @param _item The withdrawal, as the Portal committed it.
    function holdWithdrawal(BridgeHookTypes.Item calldata _item) external payable onlyPortal {
        _hold(_item);
    }

    /// @notice Records a clear verdict against an identifier.
    /// @dev    Does not require the item to exist, so a verdict can be written before the item
    ///         reaches the Portal.
    /// @param _id        The item's identifier.
    /// @param _expiresAt When the verdict stops being valid, as the Verdict Writer chose it.
    ///                   Must not already have passed. Zero is rejected by the same check,
    ///                   because zero is the record's sentinel for carrying no verdict at all.
    function recordVerdict(bytes32 _id, uint64 _expiresAt) external onlyPolicy {
        // Either would store a clearance that can never let the item move, while still emitting
        // VerdictRecorded. A screener watching the event would read that as a success.
        if (_expiresAt < block.timestamp) revert MockComplianceModule_InvalidExpiry();

        _items[_id].expiresAt = _expiresAt;
        emit VerdictRecorded(_id, _expiresAt);
    }

    /// @notice Removes a clear verdict.
    /// @param _id The item's identifier.
    function revokeVerdict(bytes32 _id) external onlyPolicy {
        delete _items[_id].expiresAt;
        emit VerdictRevoked(_id);
    }

    /// @notice Runs a held deposit. Permissionless, so a cleared user never depends on the
    ///         operator to send the transaction.
    /// @param _item The item as it was held.
    function completeDeposit(BridgeHookTypes.Item calldata _item) external {
        (bytes32 id, uint256 ethAmount) = _release(_item);
        portal.completeDepositTransaction{ value: ethAmount }(_item);
        emit Completed(id);
    }

    /// @notice Runs a held withdrawal. Permissionless.
    /// @param _item The item as it was held.
    function completeWithdrawal(BridgeHookTypes.Item calldata _item) external {
        if (paused()) revert ComplianceModule_Paused();
        (bytes32 id, uint256 ethAmount) = _release(_item);
        portal.completeWithdrawalTransaction{ value: ethAmount }(_item);
        emit Completed(id);
    }

    /// @notice Points the module at a different policy. Records and verdicts are untouched, so
    ///         held items survive the change.
    /// @param _policy The new policy.
    function setPolicy(IPolicy _policy) external onlyOwner {
        if (address(_policy) == address(0)) revert ComplianceModule_ZeroAddress();
        policy = _policy;
        emit PolicySet(_policy);
    }

    /// @notice Sets the emergency disable switch, so a failing policy cannot block new items
    ///         indefinitely. It is not a release: items already held stay held and still need a
    ///         clear verdict to complete.
    /// @param _cutoffL2Nonce The L2ToL1MessagePasser nonce at the time of the disable.
    function disable(uint256 _cutoffL2Nonce) external onlyOwner {
        disabled = true;
        cutoffL2Nonce = _cutoffL2Nonce;
        emit Disabled(_cutoffL2Nonce);
    }

    /// @notice Clears the disable switch, so new items are screened again.
    function enable() external onlyOwner {
        disabled = false;
        emit Enabled();
    }

    /// @notice Records a held item and takes custody of its value.
    /// @param _item The item being held.
    function _hold(BridgeHookTypes.Item calldata _item) internal {
        if (msg.value != _item.ethAmount) revert ComplianceModule_ETHAmountMismatch();

        bytes32 id = BridgeHookHashing.hashItem(_item);
        _items[id].heldAt = uint64(block.timestamp);

        if (_item.asset == BridgeHookTypes.Asset.ERC20) {
            (address l1Token, address l2Token, uint256 amount) = _tokenTransfer(_item);
            // Only tokens actually received are accounted for. The bridge sends none for a
            // token native to L2, which is burned and minted rather than escrowed, and none at
            // all while its hold entry points are still empty stubs.
            if (IERC20(l1Token).balanceOf(address(this)) >= heldTokens[l1Token][l2Token] + amount) {
                heldTokens[l1Token][l2Token] += amount;
            }
        }

        // Emitted last, so a reader never sees a Held for an item the module does not hold. It
        // carries the whole item because the module stores only two timestamps, making this log
        // the only on-chain record of the preimage completion needs.
        emit Held(id, _item);
    }

    /// @notice Checks a held item may move, clears its record, and returns its value.
    /// @dev    The record is deleted before the policy or the Portal is called, so a policy that
    ///         calls back cannot move the same item twice.
    /// @param _item The item as it was held.
    /// @return id_        The item's identifier.
    /// @return ethAmount_ The ETH to forward with the completion.
    function _release(BridgeHookTypes.Item calldata _item) internal returns (bytes32 id_, uint256 ethAmount_) {
        id_ = BridgeHookHashing.hashItem(_item);
        Record memory record = _items[id_];

        if (record.heldAt == 0) revert ComplianceModule_ItemNotHeld();
        if (record.expiresAt == 0) revert ComplianceModule_ItemNotCleared();

        delete _items[id_];

        // A zero finalizable time tells the policy this is a completion rather than an item
        // arriving.
        if (!policy.screenRelease(_item, parties(_item), record.expiresAt, 0)) {
            revert ComplianceModule_PolicyRefused();
        }

        if (_item.asset == BridgeHookTypes.Asset.ERC20) {
            (address l1Token, address l2Token, uint256 amount) = _tokenTransfer(_item);
            if (heldTokens[l1Token][l2Token] >= amount) {
                heldTokens[l1Token][l2Token] -= amount;
                IERC20(l1Token).safeIncreaseAllowance(address(bridge()), amount);
            }
        }

        ethAmount_ = _item.ethAmount;
    }

    /// @notice Reads the token pair and amount out of a token transfer item.
    function _tokenTransfer(BridgeHookTypes.Item calldata _item)
        internal
        pure
        returns (address l1Token_, address l2Token_, uint256 amount_)
    {
        (, bytes memory message) = BridgeHookDecoder.extractTokenMessage(_item);
        (, l1Token_, l2Token_, amount_) = BridgeHookDecoder.decodeTokenTransfer(_item.direction, message);
    }

    /// @notice The Portal's SystemConfig, which names the bridge and the messenger.
    function _systemConfig() internal view returns (ISystemConfig) {
        return ISystemConfig(portal.systemConfig());
    }
}
