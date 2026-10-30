// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Libraries
import { BridgeHookTypes } from "src/libraries/BridgeHookTypes.sol";
import { BridgeHookHashing } from "src/libraries/BridgeHookHashing.sol";

// Interfaces
import { IPolicy } from "interfaces/compliance/IPolicy.sol";
import { IComplianceModule } from "interfaces/compliance/IComplianceModule.sol";

/// @custom:proxied false
/// @title MockPolicy
/// @notice Devnet-only policy for the bridge-screener integration. It decides nothing on its
///         own: an operator sets what each direction should answer, and a per-item override
///         pins individual items so fixtures are reproducible under concurrency.
/// @dev    NOT FOR PRODUCTION. A real policy screens the parties it is handed; this one ignores
///         them entirely and answers from storage.
contract MockPolicy is IPolicy {
    /// @notice The three screening outcomes a policy can return.
    /// @custom:value Proceed The item takes the normal path.
    /// @custom:value Hold    The item is deferred into the hook's custody.
    /// @custom:value Reject  The call reverts and nothing is consumed.
    enum Outcome {
        Proceed,
        Hold,
        Reject
    }

    /// @notice A per-item outcome. Unset falls through to the direction's default, which is why
    ///         this cannot simply reuse Outcome.
    enum Override {
        Unset,
        Proceed,
        Hold,
        Reject
    }

    /// @notice Which call produced an observation, carried on ItemObserved.
    /// @custom:value 0 screen, a deposit arriving.
    /// @custom:value 1 screenRelease with a real finalizable time, a withdrawal arriving.
    /// @custom:value 2 screenRelease with a zero finalizable time, a held item completing.
    uint8 internal constant KIND_DEPOSIT = 0;
    uint8 internal constant KIND_WITHDRAWAL = 1;
    uint8 internal constant KIND_RELEASE = 2;

    /// @notice Module this policy answers for and records verdicts through.
    IComplianceModule public module;

    /// @notice Address allowed to change the defaults, the overrides and the screener.
    address public owner;

    /// @notice Address allowed to clear and revoke verdicts. The off-chain Verdict Writer.
    address public screener;

    /// @notice Answer for a deposit when it has no override.
    Outcome public defaultDeposit;

    /// @notice Answer for a withdrawal when it has no override.
    Outcome public defaultWithdrawal;

    /// @notice Answer when a held item tries to complete and has no override.
    Outcome public defaultRelease;

    /// @notice Per-item answers, which take precedence over the defaults. Settable before the
    ///         item exists, which is what allows a fixture to be scripted up front.
    mapping(bytes32 => Override) public overrides;

    /// @notice Emitted on every screening call, whatever it answers.
    /// @dev    MOCK ONLY. A proceeding deposit leaves no other on-chain record of its item ID,
    ///         because TransactionDeposited does not carry the deposit counter. This gives the
    ///         screener a complete ordered feed of every ID the Portal computed, which is only
    ///         true because screening is restricted to the module: an open entry point would let
    ///         anyone forge entries in it. A rejected item is absent by construction, since the
    ///         revert discards the log with everything else.
    event ItemObserved(bytes32 indexed id, BridgeHookTypes.Item item, uint8 kind, bool pass);

    /// @notice Emitted when a direction's default answer changes.
    event DefaultOutcomeSet(uint8 indexed kind, Outcome outcome);

    /// @notice Emitted when an item's override changes.
    event OverrideSet(bytes32 indexed id, Override outcome);

    /// @notice Emitted when the module is set.
    event ModuleSet(IComplianceModule indexed module);

    /// @notice Emitted when the screener address is set.
    event ScreenerSet(address indexed screener);

    /// @notice Thrown when an item's outcome is Reject. Refuses the item, consuming nothing.
    error MockPolicy_Rejected(bytes32 id);

    /// @notice Thrown when the caller is not the owner.
    error MockPolicy_NotOwner();

    /// @notice Thrown when the caller is not the screener.
    error MockPolicy_NotScreener();

    /// @notice Thrown when the caller is not the module.
    error MockPolicy_NotModule();

    /// @notice Thrown when a verdict is written before the module is set.
    error MockPolicy_ModuleNotSet();

    modifier onlyOwner() {
        if (msg.sender != owner) revert MockPolicy_NotOwner();
        _;
    }

    modifier onlyScreener() {
        if (msg.sender != screener) revert MockPolicy_NotScreener();
        _;
    }

    /// @notice Restricts screening to the module. Without it anyone could call these and emit
    ///         ItemObserved for an item the Portal never saw, and an indexer that trusts the
    ///         feed would ingest it.
    modifier onlyModule() {
        if (msg.sender != address(module)) revert MockPolicy_NotModule();
        _;
    }

    /// @param _owner    Address allowed to change the defaults and overrides.
    /// @param _screener Address allowed to clear and revoke verdicts.
    constructor(address _owner, address _screener) {
        owner = _owner;
        screener = _screener;
        emit ScreenerSet(_screener);
    }

    /// @notice Answers for a deposit arriving at the Portal.
    /// @param _item The item being screened.
    /// @return pass_ True to proceed, false to hold. Reverts to reject.
    function screen(BridgeHookTypes.Item calldata _item, address[] calldata) external onlyModule returns (bool pass_) {
        pass_ = _decide(_item, defaultDeposit, KIND_DEPOSIT);
    }

    /// @notice Answers when value moves: a withdrawal arriving, or a held item completing.
    /// @param _item          The item being screened.
    /// @param _expiresAt     When the item's clearance expires, as the module recorded it.
    /// @param _finalizableAt When the withdrawal became finalizable. Zero on a completion, which
    ///                       is how the two callers are told apart.
    /// @return pass_ True to proceed, false to hold. Reverts to reject.
    function screenRelease(
        BridgeHookTypes.Item calldata _item,
        address[] calldata,
        uint64 _expiresAt,
        uint256 _finalizableAt
    )
        external
        onlyModule
        returns (bool pass_)
    {
        bool isCompletion = _finalizableAt == 0;
        Outcome fallbackOutcome = isCompletion ? defaultRelease : defaultWithdrawal;
        uint8 kind = isCompletion ? KIND_RELEASE : KIND_WITHDRAWAL;

        pass_ = _decide(_item, fallbackOutcome, kind);

        // An expired clearance counts as none, so an item that would otherwise proceed is held
        // like any other uncleared one.
        if (pass_ && _expiresAt != 0 && block.timestamp > _expiresAt) pass_ = false;
    }

    /// @notice Records a clear verdict for an item, with the expiry the screener chose. How long
    ///         a verdict stays valid is decided entirely off chain.
    /// @param _id        Identifier of the item to clear.
    /// @param _expiresAt When the clearance stops counting.
    function clear(bytes32 _id, uint64 _expiresAt) public onlyScreener {
        if (address(module) == address(0)) revert MockPolicy_ModuleNotSet();

        // The module owns the verdict registry and announces the write itself, so there is
        // nothing for the policy to emit here.
        module.recordVerdict(_id, _expiresAt);
    }

    /// @notice Removes an item's clear verdict.
    /// @param _id Identifier of the item.
    function revoke(bytes32 _id) external onlyScreener {
        if (address(module) == address(0)) revert MockPolicy_ModuleNotSet();
        module.revokeVerdict(_id);
    }

    /// @notice Clears a held deposit and completes it in one transaction.
    /// @dev    The real policy derives the expiry from its own configuration, so its signature
    ///         carries no expiry. This one takes it, because the screener owns the expiry.
    /// @param _item      The item as it was held.
    /// @param _expiresAt When the clearance stops counting.
    function clearAndCompleteDeposit(BridgeHookTypes.Item calldata _item, uint64 _expiresAt) external {
        clear(BridgeHookHashing.hashItem(_item), _expiresAt);
        module.completeDeposit(_item);
    }

    /// @notice Clears a held withdrawal and completes it in one transaction.
    /// @param _item      The item as it was held.
    /// @param _expiresAt When the clearance stops counting.
    function clearAndCompleteWithdrawal(BridgeHookTypes.Item calldata _item, uint64 _expiresAt) external {
        clear(BridgeHookHashing.hashItem(_item), _expiresAt);
        module.completeWithdrawal(_item);
    }

    /// @notice Points the policy at the module it records verdicts through.
    /// @param _module The module.
    function setModule(IComplianceModule _module) external onlyOwner {
        module = _module;
        emit ModuleSet(_module);
    }

    /// @notice Sets the address allowed to clear and revoke verdicts.
    /// @param _screener The screener.
    function setScreener(address _screener) external onlyOwner {
        screener = _screener;
        emit ScreenerSet(_screener);
    }

    /// @notice Sets the answer for deposits that have no override.
    /// @param _outcome The answer.
    function setDefaultDeposit(Outcome _outcome) external onlyOwner {
        defaultDeposit = _outcome;
        emit DefaultOutcomeSet(KIND_DEPOSIT, _outcome);
    }

    /// @notice Sets the answer for withdrawals that have no override.
    /// @param _outcome The answer.
    function setDefaultWithdrawal(Outcome _outcome) external onlyOwner {
        defaultWithdrawal = _outcome;
        emit DefaultOutcomeSet(KIND_WITHDRAWAL, _outcome);
    }

    /// @notice Sets the answer for completions that have no override.
    /// @param _outcome The answer.
    function setDefaultRelease(Outcome _outcome) external onlyOwner {
        defaultRelease = _outcome;
        emit DefaultOutcomeSet(KIND_RELEASE, _outcome);
    }

    /// @notice Pins one item's answer, overriding the direction's default. Can be set before the
    ///         item exists.
    /// @param _id      Identifier of the item.
    /// @param _outcome The answer, or Unset to fall back to the default.
    function setOverride(bytes32 _id, Override _outcome) external onlyOwner {
        overrides[_id] = _outcome;
        emit OverrideSet(_id, _outcome);
    }

    /// @notice Resolves an item's outcome and records the observation.
    /// @param _item     The item being screened.
    /// @param _fallback The answer to use when the item has no override.
    /// @param _kind     Which call is asking.
    /// @return pass_ True to proceed, false to hold.
    function _decide(BridgeHookTypes.Item calldata _item, Outcome _fallback, uint8 _kind)
        internal
        returns (bool pass_)
    {
        bytes32 id = BridgeHookHashing.hashItem(_item);

        Override pinned = overrides[id];
        Outcome outcome = pinned == Override.Unset ? _fallback : Outcome(uint8(pinned) - 1);

        // Nothing is emitted on the reject path: the revert takes the log with it.
        if (outcome == Outcome.Reject) revert MockPolicy_Rejected(id);

        pass_ = outcome == Outcome.Proceed;
        emit ItemObserved(id, _item, _kind, pass_);
    }
}
