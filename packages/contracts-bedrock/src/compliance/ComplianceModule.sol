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
    // nosemgrep: sol-safety-no-immutable-variables
    IOptimismPortal2 internal immutable PORTAL;

    /// @notice The L1CrossDomainMessenger used to resolve the parties of an item.
    // nosemgrep: sol-safety-no-immutable-variables
    IL1CrossDomainMessenger internal immutable L1_CROSS_DOMAIN_MESSENGER;

    /// @notice The L1StandardBridge used to resolve the parties of an item and to restore tokens.
    // nosemgrep: sol-safety-no-immutable-variables
    IL1StandardBridge internal immutable BRIDGE;

    /// @notice Item identifier to hold status and clear verdict.
    mapping(bytes32 => Record) internal _items;

    /// @notice The policy the module consults and accepts verdicts from.
    IPolicy public policy;

    /// @notice Whether deposits and withdrawals at or above cutoffL2Nonce bypass screening.
    ///         Held items still require approval from the policy before release.
    bool public disabled;

    /// @notice The first L2 withdrawal nonce that skips screening while the module is disabled.
    uint256 public cutoffL2Nonce;

    /// @notice Amount of escrowed tokens held for each L1 token and L2 token pair.
    ///         Excludes mintable tokens, which the bridge burns instead of transferring here.
    mapping(address => mapping(address => uint256)) public heldTokens;

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

    /// @notice Deploys an implementation bound to the supplied Portal, messenger, and bridge.
    ///         All three addresses must be nonzero and must belong to the same chain configuration.
    ///         The Portal does not need to be initialized before this implementation is deployed.
    ///         Initialization is only available through a proxy.
    /// @param _portal                 The Portal that will use this module as its bridge hook.
    /// @param _l1CrossDomainMessenger The L1CrossDomainMessenger for the same chain.
    /// @param _l1StandardBridge       The L1StandardBridge for the same chain.
    constructor(
        IOptimismPortal2 _portal,
        IL1CrossDomainMessenger _l1CrossDomainMessenger,
        IL1StandardBridge _l1StandardBridge
    )
        // OpenZeppelin v5 Ownable requires a nonzero owner. This placeholder is stored only in
        // the implementation, whose initializers are disabled. The proxy's owner is set in
        // initialize().
        Ownable(address(0xdEaD))
    {
        if (
            address(_portal) == address(0) || address(_l1CrossDomainMessenger) == address(0)
                || address(_l1StandardBridge) == address(0)
        ) revert ComplianceModule_ZeroAddress();

        PORTAL = _portal;
        L1_CROSS_DOMAIN_MESSENGER = _l1CrossDomainMessenger;
        BRIDGE = _l1StandardBridge;
        _disableInitializers();
    }

    /// @notice Initializes the proxy once. Only its ProxyAdmin or the ProxyAdmin's owner can call.
    ///         The module owner manages the policy and screening bypass, separately from proxy
    ///         upgrades.
    /// @param _owner  The nonzero address that will own the module.
    /// @param _policy The nonzero address of the initial policy.
    function initialize(address _owner, IPolicy _policy) external initializer {
        _assertOnlyProxyAdminOrProxyAdminOwner();
        if (_owner == address(0) || address(_policy) == address(0)) revert ComplianceModule_ZeroAddress();

        _transferOwnership(_owner);
        policy = _policy;
        emit PolicySet(IPolicy(address(0)), _policy);
    }

    /// @notice Returns the OptimismPortal this module is the hook for.
    function portal() external view returns (IOptimismPortal2 portal_) {
        portal_ = PORTAL;
    }

    /// @notice Returns the messenger used to identify the parties of a cross-domain message.
    function l1CrossDomainMessenger() external view returns (IL1CrossDomainMessenger l1CrossDomainMessenger_) {
        l1CrossDomainMessenger_ = L1_CROSS_DOMAIN_MESSENGER;
    }

    /// @notice Returns the bridge used to identify token transfers and restore held tokens.
    function bridge() external view returns (IL1StandardBridge bridge_) {
        bridge_ = BRIDGE;
    }

    /// @notice Returns the record for an item identifier.
    /// @param _id The item identifier.
    function items(bytes32 _id) external view returns (Record memory) {
        return _items[_id];
    }

    /// @notice Returns whether the Portal is paused.
    function paused() external view returns (bool) {
        return PORTAL.paused();
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
    /// @param _expiresAt When the verdict expires. Zero means no verdict.
    function recordVerdict(bytes32 _id, uint64 _expiresAt) external {
        _assertOnlyPolicy();
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
        PORTAL.completeDepositTransaction{ value: _item.ethAmount }(_item);
        emit Completed(id);
    }

    /// @notice Releases a held withdrawal through the Portal. Anyone can call while the Portal is
    ///         not paused. Requires a clear verdict and policy approval.
    /// @param _item The item as emitted in Held.
    function completeWithdrawal(BridgeHookTypes.Item calldata _item) external {
        if (PORTAL.paused()) revert ComplianceModule_Paused();

        bytes32 id = _release(_item);
        PORTAL.completeWithdrawalTransaction{ value: _item.ethAmount }(_item);
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

    /// @notice Reverts. The module owner cannot be removed.
    function renounceOwnership() public pure override {
        revert ComplianceModule_OwnershipNotRenounceable();
    }

    /// @notice Reverts unless the caller is the Portal and this module is its bridge hook.
    ///         Checks both because the Portal can call a former hook as a withdrawal target.
    function _assertOnlyActiveHook() internal view {
        if (msg.sender != address(PORTAL)) revert ComplianceModule_CallerNotPortal();
        if (address(PORTAL.bridgeHook()) != address(this)) revert ComplianceModule_NotActiveHook();
    }

    /// @notice Reverts unless the caller is the policy.
    function _assertOnlyPolicy() internal view {
        if (msg.sender != address(policy)) revert ComplianceModule_CallerNotPolicy();
    }

    /// @notice Records a hold and tracks escrowed token amounts transferred by the bridge.
    ///         Leaves any recorded verdict unchanged.
    /// @param _item The deposit or withdrawal item.
    function _hold(BridgeHookTypes.Item calldata _item) internal {
        _assertOnlyActiveHook();
        if (msg.value != _item.ethAmount) revert ComplianceModule_ETHAmountMismatch();

        bytes32 id = BridgeHookHashing.hashItem(_item);
        _items[id].heldAt = uint64(block.timestamp);

        if (
            _item.asset == BridgeHookTypes.Asset.ERC20
                && BRIDGE.heldTokenKinds(id) == BridgeHookTypes.TokenKind.Escrowed
        ) {
            (, bytes memory message) = BridgeHookDecoder.extractTokenMessage(_item);
            (, address l1Token, address l2Token, uint256 amount) =
                BridgeHookDecoder.decodeTokenTransfer(_item.direction, message);
            heldTokens[l1Token][l2Token] += amount;
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
        if (
            _item.asset == BridgeHookTypes.Asset.ERC20
                && BRIDGE.heldTokenKinds(id_) == BridgeHookTypes.TokenKind.Escrowed
        ) {
            (, bytes memory message) = BridgeHookDecoder.extractTokenMessage(_item);
            (, address l1Token, address l2Token, uint256 amount) =
                BridgeHookDecoder.decodeTokenTransfer(_item.direction, message);
            IERC20(l1Token).forceApprove(address(BRIDGE), amount);
            heldTokens[l1Token][l2Token] -= amount;
        }
    }

    /// @notice Resolves the sender and recipient for both screening and the parties view.
    /// @param _item The deposit or withdrawal item.
    /// @return parties_ The sender, then the recipient.
    function _parties(BridgeHookTypes.Item memory _item) internal view returns (address[] memory parties_) {
        (address initiator, address recipient) =
            BridgeHookDecoder.resolveParties(_item, address(L1_CROSS_DOMAIN_MESSENGER), address(BRIDGE));
        parties_ = new address[](2);
        parties_[0] = initiator;
        parties_[1] = recipient;
    }
}
