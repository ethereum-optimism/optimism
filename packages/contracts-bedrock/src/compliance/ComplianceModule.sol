// SPDX-License-Identifier: MIT
pragma solidity 0.8.25;

// Contracts
import { ProxyAdminOwnedBase } from "src/universal/ProxyAdminOwnedBase.sol";
import { Initializable } from "@openzeppelin/contracts-v5/proxy/utils/Initializable.sol";
import { Ownable } from "@openzeppelin/contracts-v5/access/Ownable.sol";
import { Ownable2Step } from "@openzeppelin/contracts-v5/access/Ownable2Step.sol";

// Libraries
import { SafeERC20 } from "@openzeppelin/contracts-v5/token/ERC20/utils/SafeERC20.sol";
import { BridgeHookDecoder } from "src/libraries/BridgeHookDecoder.sol";
import { BridgeHookHashing } from "src/libraries/BridgeHookHashing.sol";
import { BridgeHookTypes } from "src/libraries/BridgeHookTypes.sol";

// Interfaces
import { IERC20 } from "@openzeppelin/contracts-v5/token/ERC20/IERC20.sol";
import { ISemver } from "interfaces/universal/ISemver.sol";
import { IOptimismPortal2 } from "interfaces/L1/IOptimismPortal2.sol";
import { IL1CrossDomainMessenger } from "interfaces/L1/IL1CrossDomainMessenger.sol";
import { IL1StandardBridge } from "interfaces/L1/IL1StandardBridge.sol";
import { IPolicy } from "interfaces/compliance/IPolicy.sol";

/// @custom:proxied true
/// @title ComplianceModule
/// @notice A bridge hook that uses a policy to screen deposits and withdrawals.
///         Holds assets when the policy requires a hold. Anyone can request completion of a held
///         item, but the policy must approve its release.
contract ComplianceModule is ProxyAdminOwnedBase, Initializable, Ownable2Step, ISemver {
    using SafeERC20 for IERC20;

    /// @notice Hold status and clear verdict for an item. A withdrawal verdict can be recorded
    ///         before the withdrawal is held. Deposit screening does not use a recorded verdict.
    /// @custom:field heldAt    Time when the item was held. Zero means the item is not held.
    /// @custom:field expiresAt Expiration time of the clear verdict. Zero means no verdict is
    ///                         recorded. The policy checks whether the verdict has expired.
    struct Record {
        uint64 heldAt;
        uint64 expiresAt;
    }

    /// @notice Semantic version.
    /// @custom:semver 1.0.0
    string public constant version = "1.0.0";

    /// @notice The OptimismPortal this module is the hook for.
    IOptimismPortal2 public portal;

    /// @notice The L1CrossDomainMessenger used to resolve the parties of an item.
    IL1CrossDomainMessenger public l1CrossDomainMessenger;

    /// @notice The L1StandardBridge used to resolve the parties of an item and to restore tokens.
    IL1StandardBridge public bridge;

    /// @notice The policy the module consults and accepts verdicts from.
    IPolicy public policy;

    /// @notice Whether deposits and withdrawals at or above cutoffL2Nonce bypass screening.
    ///         Held items still require approval from the policy before release.
    bool public disabled;

    /// @notice The first L2 withdrawal nonce that skips screening while the module is disabled.
    uint256 public cutoffL2Nonce;

    /// @notice Item identifier to hold status and clear verdict.
    mapping(bytes32 => Record) internal _items;

    /// @notice Amount of escrowed tokens held for each L1 token and L2 token pair.
    ///         Excludes mintable tokens, which the bridge burns instead of transferring here.
    mapping(address l1Token => mapping(address l2Token => uint256 amount)) public heldTokens;

    /// @notice Thrown when the caller is not the Portal.
    error ComplianceModule_CallerNotPortal();

    /// @notice Thrown when the Portal's bridge hook is not this module.
    error ComplianceModule_NotActiveHook();

    /// @notice Thrown when the caller is not the policy.
    error ComplianceModule_CallerNotPolicy();

    /// @notice Thrown when the item is not held.
    error ComplianceModule_ItemNotHeld();

    /// @notice Thrown when the item has no clear verdict.
    error ComplianceModule_ItemNotCleared();

    /// @notice Thrown when the policy does not approve a release.
    error ComplianceModule_PolicyRefused();

    /// @notice Thrown when the ETH sent with a hold request differs from the item's ethAmount.
    error ComplianceModule_ETHAmountMismatch();

    /// @notice Thrown when an address argument is zero.
    error ComplianceModule_ZeroAddress();

    /// @notice Thrown when the module's token balance does not cover heldTokens after a hold.
    error ComplianceModule_InsufficientBalance();

    /// @notice Thrown when the policy records a verdict with a zero expiry.
    error ComplianceModule_ZeroExpiry();

    /// @notice Thrown when withdrawal completion is requested while the Portal is paused.
    error ComplianceModule_Paused();

    /// @notice Thrown when any caller tries to renounce ownership.
    error ComplianceModule_OwnershipNotRenounceable();

    /// @notice Emitted when an item is held.
    /// @param id   The item identifier.
    /// @param item The full item.
    event Held(bytes32 indexed id, BridgeHookTypes.Item item);

    /// @notice Emitted when the policy records a clear verdict.
    /// @param id        The item identifier.
    /// @param expiresAt When the verdict expires.
    event VerdictRecorded(bytes32 indexed id, uint64 expiresAt);

    /// @notice Emitted when the policy revokes a verdict.
    /// @param id The item identifier.
    event VerdictRevoked(bytes32 indexed id);

    /// @notice Emitted when a held item completes.
    /// @param id The item identifier.
    event Completed(bytes32 indexed id);

    /// @notice Emitted when the policy changes.
    /// @param previousPolicy The previous policy.
    /// @param newPolicy      The new policy.
    event PolicySet(IPolicy indexed previousPolicy, IPolicy indexed newPolicy);

    /// @notice Emitted when screening is disabled.
    /// @param cutoffL2Nonce The first L2 withdrawal nonce that skips screening.
    event Disabled(uint256 cutoffL2Nonce);

    /// @notice Emitted when screening is enabled.
    event Enabled();

    /// @notice Deploys an implementation whose initializers are disabled. Initialization is only
    ///         available through a proxy.
    constructor()
        // OpenZeppelin v5 Ownable requires a nonzero owner. This placeholder is stored only in
        // the implementation, whose initializers are disabled. The proxy's owner is set in
        // initialize().
        Ownable(address(0xdEaD))
    {
        _disableInitializers();
    }

    /// @notice Initializes the proxy once. Only its ProxyAdmin or the ProxyAdmin's owner can call.
    ///         The module owner manages the policy and screening bypass, separately from proxy
    ///         upgrades. The Portal does not need to be initialized first.
    /// @param _owner                  The nonzero address that will own the module.
    /// @param _policy                 The nonzero address of the initial policy.
    /// @param _portal                 The Portal that will use this module as its bridge hook.
    /// @param _l1CrossDomainMessenger The L1CrossDomainMessenger for the same chain.
    /// @param _l1StandardBridge       The L1StandardBridge for the same chain.
    function initialize(
        address _owner,
        IPolicy _policy,
        IOptimismPortal2 _portal,
        IL1CrossDomainMessenger _l1CrossDomainMessenger,
        IL1StandardBridge _l1StandardBridge
    )
        external
        initializer
    {
        _assertOnlyProxyAdminOrProxyAdminOwner();
        if (
            _owner == address(0) || address(_policy) == address(0) || address(_portal) == address(0)
                || address(_l1CrossDomainMessenger) == address(0) || address(_l1StandardBridge) == address(0)
        ) revert ComplianceModule_ZeroAddress();

        _transferOwnership(_owner);
        policy = _policy;
        portal = _portal;
        l1CrossDomainMessenger = _l1CrossDomainMessenger;
        bridge = _l1StandardBridge;
        emit PolicySet(IPolicy(address(0)), _policy);
    }

    /// @notice Returns the record for an item identifier.
    /// @param _id The item identifier.
    function items(bytes32 _id) external view returns (Record memory) {
        return _items[_id];
    }

    /// @notice Returns whether the Portal is paused.
    function paused() external view returns (bool) {
        return portal.paused();
    }

    /// @notice Returns the sender and recipient that the policy screens for an item. Decodes
    ///         canonical bridge and messenger messages. Otherwise uses the outer addresses and
    ///         unaliases an aliased deposit sender.
    /// @param _item The deposit or withdrawal item.
    /// @return parties_ The sender, then the recipient.
    function parties(BridgeHookTypes.Item calldata _item) external view returns (address[] memory parties_) {
        parties_ = _parties(_item);
    }

    /// @notice Asks the policy whether a deposit can proceed. Returns true when screening is
    ///         disabled. Only the Portal can call while this module is its hook.
    /// @param _item The deposit item.
    /// @return pass_ True to proceed, or false to hold.
    function screenDeposit(BridgeHookTypes.Item calldata _item) external returns (bool pass_) {
        _assertOnlyActiveHook();
        if (disabled) return true;

        pass_ = policy.screenDeposit(_item, _parties(_item));
    }

    /// @notice Records a deposit hold and receives its ETH. Only the Portal can call while this
    ///         module is its hook. The bridge must process any token hold first.
    /// @param _item The deposit item.
    function holdDeposit(BridgeHookTypes.Item calldata _item) external payable {
        _hold(_item);
    }

    /// @notice Asks the policy whether a withdrawal can proceed. Returns true when screening is
    ///         disabled and the nonce is at or above cutoffL2Nonce. Only the Portal can call while
    ///         this module is its hook.
    /// @param _item          The withdrawal item.
    /// @param _finalizableAt One second after the later of the proof maturity and game finality end times.
    /// @return pass_ True to proceed, or false to hold.
    function screenWithdrawal(
        BridgeHookTypes.Item calldata _item,
        uint256 _finalizableAt
    )
        external
        returns (bool pass_)
    {
        _assertOnlyActiveHook();
        if (disabled && _item.nonce >= cutoffL2Nonce) return true;

        bytes32 id = BridgeHookHashing.hashItem(_item);
        pass_ = policy.screenWithdrawal(_item, _parties(_item), _items[id].expiresAt, _finalizableAt);
    }

    /// @notice Records a withdrawal hold and receives its ETH. Only the Portal can call while this
    ///         module is its hook. The bridge must process any token hold first.
    /// @param _item The withdrawal item.
    function holdWithdrawal(BridgeHookTypes.Item calldata _item) external payable {
        _hold(_item);
    }

    /// @notice Records a clear verdict for an item, held or not. Only the policy can call.
    /// @param _id        The item identifier.
    /// @param _expiresAt When the verdict expires. Must not be zero, because zero means no verdict.
    function recordVerdict(bytes32 _id, uint64 _expiresAt) external {
        _assertOnlyPolicy();
        if (_expiresAt == 0) revert ComplianceModule_ZeroExpiry();
        _items[_id].expiresAt = _expiresAt;
        emit VerdictRecorded(_id, _expiresAt);
    }

    /// @notice Removes the clear verdict for an item. Only the policy can call.
    /// @param _id The item identifier.
    function revokeVerdict(bytes32 _id) external {
        _assertOnlyPolicy();
        _items[_id].expiresAt = 0;
        emit VerdictRevoked(_id);
    }

    /// @notice Releases a held deposit through the Portal. Anyone can call. Requires a clear
    ///         verdict and policy approval.
    /// @param _item The item as emitted in Held.
    function completeDeposit(BridgeHookTypes.Item calldata _item) external {
        bytes32 id = _release(_item);
        portal.completeDepositTransaction{ value: _item.ethAmount }(_item);
        emit Completed(id);
    }

    /// @notice Releases a held withdrawal through the Portal. Anyone can call while the Portal is
    ///         not paused. Requires a clear verdict and policy approval.
    /// @param _item The item as emitted in Held.
    function completeWithdrawal(BridgeHookTypes.Item calldata _item) external {
        if (portal.paused()) revert ComplianceModule_Paused();

        bytes32 id = _release(_item);
        portal.completeWithdrawalTransaction{ value: _item.ethAmount }(_item);
        emit Completed(id);
    }

    /// @notice Sets the policy. Only the owner can call.
    /// @param _policy The new policy.
    function setPolicy(IPolicy _policy) external onlyOwner {
        if (address(_policy) == address(0)) revert ComplianceModule_ZeroAddress();

        IPolicy previousPolicy = policy;
        policy = _policy;
        emit PolicySet(previousPolicy, _policy);
    }

    /// @notice Disables screening for deposits, and for withdrawals with a nonce at or above
    ///         _cutoffL2Nonce. Only the owner can call. Held items stay held.
    /// @param _cutoffL2Nonce The first withdrawal nonce that skips screening, as returned by
    ///                       L2ToL1MessagePasser.messageNonce().
    function disable(uint256 _cutoffL2Nonce) external onlyOwner {
        disabled = true;
        cutoffL2Nonce = _cutoffL2Nonce;
        emit Disabled(_cutoffL2Nonce);
    }

    /// @notice Enables screening and clears the cutoff. Only the owner can call.
    function enable() external onlyOwner {
        disabled = false;
        cutoffL2Nonce = 0;
        emit Enabled();
    }

    /// @notice Prevents renunciation so the owner can always replace the policy and toggle screening.
    function renounceOwnership() public pure override {
        revert ComplianceModule_OwnershipNotRenounceable();
    }

    /// @notice Reverts unless the caller is the Portal and this module is its bridge hook.
    ///         Checks both because the Portal can call a former hook as a withdrawal target.
    function _assertOnlyActiveHook() internal view {
        if (msg.sender != address(portal)) revert ComplianceModule_CallerNotPortal();
        if (address(portal.bridgeHook()) != address(this)) revert ComplianceModule_NotActiveHook();
    }

    /// @notice Reverts unless the caller is the policy.
    function _assertOnlyPolicy() internal view {
        if (msg.sender != address(policy)) revert ComplianceModule_CallerNotPolicy();
    }

    /// @notice Records a hold and tracks escrowed token amounts transferred by the bridge.
    ///         Reverts if the module's token balance does not cover heldTokens. Leaves any
    ///         recorded verdict unchanged.
    /// @param _item The deposit or withdrawal item.
    function _hold(BridgeHookTypes.Item calldata _item) internal {
        _assertOnlyActiveHook();
        if (msg.value != _item.ethAmount) revert ComplianceModule_ETHAmountMismatch();

        bytes32 id = BridgeHookHashing.hashItem(_item);
        _items[id].heldAt = uint64(block.timestamp);

        (bool escrowed, address l1Token, address l2Token, uint256 amount) = _escrowedTransfer(_item, id);
        if (escrowed) {
            uint256 held = heldTokens[l1Token][l2Token] + amount;
            if (IERC20(l1Token).balanceOf(address(this)) < held) revert ComplianceModule_InsufficientBalance();
            heldTokens[l1Token][l2Token] = held;
        }

        emit Held(id, _item);
    }

    /// @notice Prepares a held item with a clear verdict for completion. Requires policy approval.
    ///         Deletes the record before calling the policy. For escrowed tokens, approves the
    ///         bridge to transfer the item's amount and reduces heldTokens.
    /// @param _item The deposit or withdrawal item.
    /// @return id_ The item identifier.
    function _release(BridgeHookTypes.Item calldata _item) internal returns (bytes32 id_) {
        id_ = BridgeHookHashing.hashItem(_item);
        Record memory record = _items[id_];
        if (record.heldAt == 0) revert ComplianceModule_ItemNotHeld();
        if (record.expiresAt == 0) revert ComplianceModule_ItemNotCleared();

        // Delete the record before external calls so reentrant completion of the same item reverts.
        delete _items[id_];

        if (!policy.screenRelease(_item, _parties(_item), record.expiresAt)) revert ComplianceModule_PolicyRefused();

        // The token kind remains available until the Portal calls the bridge's restoreERC20.
        (bool escrowed, address l1Token, address l2Token, uint256 amount) = _escrowedTransfer(_item, id_);
        if (escrowed) {
            IERC20(l1Token).forceApprove(address(bridge), amount);
            heldTokens[l1Token][l2Token] -= amount;
        }
    }

    /// @notice Decodes the token transfer of an item whose escrowed tokens this module holds.
    /// @param _item The deposit or withdrawal item.
    /// @param _id   The item identifier.
    /// @return escrowed_ Whether the item carries an escrowed token transfer.
    /// @return l1Token_  The L1 token. Zero unless escrowed.
    /// @return l2Token_  The L2 token. Zero unless escrowed.
    /// @return amount_   The token amount. Zero unless escrowed.
    function _escrowedTransfer(
        BridgeHookTypes.Item calldata _item,
        bytes32 _id
    )
        internal
        view
        returns (bool escrowed_, address l1Token_, address l2Token_, uint256 amount_)
    {
        if (
            _item.asset != BridgeHookTypes.Asset.ERC20
                || bridge.heldTokenKinds(_id) != BridgeHookTypes.TokenKind.Escrowed
        ) return (false, address(0), address(0), 0);

        (, bytes memory message) = BridgeHookDecoder.extractTokenMessage(_item);
        (, l1Token_, l2Token_, amount_) = BridgeHookDecoder.decodeTokenTransfer(_item.direction, message);
        escrowed_ = true;
    }

    /// @notice Resolves the sender and recipient for both screening and the parties view.
    /// @param _item The deposit or withdrawal item.
    /// @return parties_ The sender, then the recipient.
    function _parties(BridgeHookTypes.Item memory _item) internal view returns (address[] memory parties_) {
        (address initiator, address recipient) =
            BridgeHookDecoder.resolveParties(_item, address(l1CrossDomainMessenger), address(bridge));
        parties_ = new address[](2);
        parties_[0] = initiator;
        parties_[1] = recipient;
    }
}
