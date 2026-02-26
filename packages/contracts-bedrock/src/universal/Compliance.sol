// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Contracts
import { Ownable } from "@solady-v0.0.245/auth/Ownable.sol";
import { Initializable } from "@openzeppelin/contracts/proxy/utils/Initializable.sol";
import { ReentrancyGuard } from "@solady-v0.0.245/utils/ReentrancyGuard.sol";
import { ProxyAdminOwnedBase } from "src/L1/ProxyAdminOwnedBase.sol";
import { ReinitializableBase } from "src/universal/ReinitializableBase.sol";

// Libraries
import { EnumerableSetLib } from "@solady-v0.0.245/utils/EnumerableSetLib.sol";
import { SafeCall } from "src/libraries/SafeCall.sol";

// Interfaces
import { ISemver } from "interfaces/universal/ISemver.sol";
import { ICompliance } from "interfaces/universal/ICompliance.sol";
import { IDonatable } from "interfaces/universal/IDonatable.sol";
import { IRule } from "interfaces/universal/IRule.sol";

/// @custom:proxied true
/// @title Compliance
/// @notice Abstract compliance screening layer for cross-chain transactions. When enabled,
///         transactions pass through configurable rules before execution. Flagged transactions
///         are held pending; compliant transactions proceed without delay. Concrete implementations
///         (L1Compliance, L2Compliance) override `_executeApproved` to call the correct bridge.
abstract contract Compliance is ProxyAdminOwnedBase, ReinitializableBase, Ownable, Initializable, ReentrancyGuard, ISemver {
    using EnumerableSetLib for EnumerableSetLib.AddressSet;

    /// @notice Emitted when a transaction is flagged as pending by compliance rules.
    event Pending(
        bytes32 indexed id,
        address indexed from,
        address indexed to,
        uint256 value,
        uint256 mint,
        uint64 gasLimit,
        uint256 nonce,
        bytes data
    );

    /// @notice Emitted when a transaction is rejected by compliance rules.
    event Rejected(
        bytes32 indexed id,
        address indexed from,
        address indexed to,
        uint256 value,
        uint256 mint,
        uint64 gasLimit,
        uint256 nonce,
        bytes data
    );

    /// @notice Emitted when a transaction is approved (either automatically or by the owner).
    event Approved(bytes32 indexed id);

    /// @notice Emitted when a rejected transaction's ETH is refunded to the sender.
    event Refunded(bytes32 indexed id);

    /// @notice Address of the bridge contract (OptimismPortal2 or L2ToL1MessagePasser).
    address payable public bridge;

    /// @notice Packed status storage. Bit 255: owner-override flag. Bits 254..0: Status enum value.
    mapping(bytes32 => uint256) private _status;

    /// @notice Bitmask for the owner-override flag (bit 255).
    uint256 private constant OVERRIDE_BIT = 1 << 255;

    /// @notice Bitmask for extracting the Status enum value (bits 254..0).
    uint256 private constant STATUS_MASK = type(uint256).max >> 1;

    /// @notice Set of compliance rule addresses.
    EnumerableSetLib.AddressSet private _rules;

    /// @notice Constructs the Compliance base contract.
    constructor() ReinitializableBase(1) {
        _disableInitializers();
    }

    /// @notice Semantic version.
    /// @custom:semver 1.0.0
    function version() public pure virtual returns (string memory) {
        return "1.0.0";
    }

    /// @notice Initializes the compliance contract.
    /// @param _bridge Address of the bridge contract.
    /// @param _owner  Address of the initial owner.
    function initialize(address _bridge, address _owner) external reinitializer(initVersion()) {
        _assertOnlyProxyAdminOrProxyAdminOwner();
        bridge = payable(_bridge);
        _initializeOwner(_owner);
    }

    /// @notice Modifier that restricts access to the bridge contract.
    modifier onlyBridge() {
        if (msg.sender != bridge) revert ICompliance.Compliance_OnlyBridge();
        _;
    }

    /// @notice Returns the Status of a cross chain message and whether it is final.
    /// @dev    Masks off bit 255 (the owner-override flag) and returns the Status
    ///         enum value from the least significant bits. A status is considered
    ///         final when it is Refunded (always final) or when the owner-override
    ///         flag (bit 255) is set (Pending or Rejected with an explicit owner
    ///         decision). Rule re-evaluation in settle() must respect finality:
    ///         a finalized status takes precedence even if rules would return a
    ///         stricter result.
    /// @param _id The hashed cross chain message identifier.
    /// @return isFinal_ True if the status is final and cannot be changed by rule re-evaluation.
    /// @return status_ The current Status of the message.
    function status(bytes32 _id) public view returns (bool isFinal_, ICompliance.Status status_) {
        uint256 raw = _status[_id];
        status_ = ICompliance.Status(raw & STATUS_MASK);
        bool hasOverride = (raw & OVERRIDE_BIT) != 0;
        isFinal_ = status_ == ICompliance.Status.Refunded || hasOverride;
    }

    /// @notice Returns all rule addresses.
    /// @return The array of rule contract addresses.
    function rules() public view returns (address[] memory) {
        return _rules.values();
    }

    /// @notice Returns whether the given address is a registered rule.
    /// @param _rule The address to check.
    /// @return True if the address is a registered rule.
    function hasRule(address _rule) public view returns (bool) {
        return _rules.contains(_rule);
    }

    /// @notice Evaluates all rules against a transaction and returns the strictest result.
    ///         Reverts if any rule returns Refunded.
    /// @param _from       The sender of the transaction.
    /// @param _to         The recipient of the transaction.
    /// @param _value      The ETH value of the transaction.
    /// @param _gasLimit   The gas limit for execution.
    /// @param _isCreation Whether the transaction creates a contract.
    /// @param _data       The calldata of the transaction.
    /// @param _nonce      The nonce of the transaction.
    /// @return strictest_ The strictest status returned by any rule.
    function _evaluateRules(
        address _from,
        address _to,
        uint256 _value,
        uint64 _gasLimit,
        bool _isCreation,
        bytes memory _data,
        uint256 _nonce
    )
        private
        returns (ICompliance.Status strictest_)
    {
        strictest_ = ICompliance.Status.Approved;
        address[] memory ruleAddrs = _rules.values();
        uint256 numRules = ruleAddrs.length;
        for (uint256 i = 0; i < numRules; i++) {
            ICompliance.Status result = IRule(ruleAddrs[i]).check({
                _from: _from,
                _to: _to,
                _value: _value,
                _gasLimit: _gasLimit,
                _isCreation: _isCreation,
                _data: _data,
                _nonce: _nonce
            });
            if (result == ICompliance.Status.Refunded) revert ICompliance.Compliance_InvalidRuleStatus();
            if (result > strictest_) {
                strictest_ = result;
            }
        }
    }

    /// @notice Checks a transaction against all configured compliance rules. Only callable
    ///         by the bridge contract. If all rules approve, returns ETH to bridge via
    ///         donateETH() and returns true. Otherwise stores only the status entry keyed
    ///         by hash — no preimage data is persisted on-chain. Emits a Pending or
    ///         Rejected event with the full preimage fields for off-chain reconstruction.
    /// @param _from       The sender of the transaction.
    /// @param _to         The recipient of the transaction.
    /// @param _value      The ETH value of the transaction.
    /// @param _gasLimit   The gas limit for execution.
    /// @param _isCreation Whether the transaction creates a contract.
    /// @param _data       The calldata of the transaction.
    /// @param _nonce      The nonce of the transaction (used for L2 withdrawals).
    /// @return allowed_   True if the transaction is approved to proceed immediately.
    function check(
        address _from,
        address _to,
        uint256 _value,
        uint64 _gasLimit,
        bool _isCreation,
        bytes calldata _data,
        uint256 _nonce
    )
        external
        payable
        onlyBridge
        returns (bool allowed_)
    {
        // Compute a unique ID for this transaction.
        bytes32 id = keccak256(abi.encode(_from, _to, _value, msg.value, _gasLimit, _isCreation, _data, _nonce));

        // Evaluate all rules and find the strictest result.
        ICompliance.Status strictest = _evaluateRules({
            _from: _from,
            _to: _to,
            _value: _value,
            _gasLimit: _gasLimit,
            _isCreation: _isCreation,
            _data: _data,
            _nonce: _nonce
        });

        if (strictest == ICompliance.Status.Approved) {
            // Return ETH to bridge via donateETH() to avoid triggering deposits/withdrawals.
            if (msg.value > 0) {
                IDonatable(bridge).donateETH{ value: msg.value }();
            }
            emit Approved(id);
            return true;
        }

        // Transaction was flagged. Store status and keep the ETH.
        // No preimage data is persisted — only the hash-to-status mapping.
        _status[id] = uint256(strictest);

        if (strictest == ICompliance.Status.Pending) {
            emit Pending(id, _from, _to, _value, msg.value, _gasLimit, _nonce, _data);
        } else if (strictest == ICompliance.Status.Rejected) {
            emit Rejected(id, _from, _to, _value, msg.value, _gasLimit, _nonce, _data);
        }

        return false;
    }

    /// @notice Owner approves a pending transaction. Sets the override bit (bit 255).
    ///         Reverts if the current status is not Pending.
    /// @param _id The unique identifier of the transaction.
    function approve(bytes32 _id) external onlyOwner {
        (, ICompliance.Status current) = status(_id);
        if (current != ICompliance.Status.Pending) {
            revert ICompliance.Compliance_NotPending();
        }
        // Set override bit (bit 255) and Approved status.
        _status[_id] = OVERRIDE_BIT | uint256(ICompliance.Status.Approved);
        emit Approved(_id);
    }

    /// @notice Owner rejects a transaction. Sets the override bit (bit 255).
    ///         Callable when the current status is Pending or Approved.
    ///         Reverts if the status is Rejected or Refunded.
    /// @param _id The unique identifier of the transaction.
    function reject(bytes32 _id) external onlyOwner {
        (, ICompliance.Status current) = status(_id);
        if (current != ICompliance.Status.Pending && current != ICompliance.Status.Approved) {
            revert ICompliance.Compliance_NotPending();
        }
        // Set override bit (bit 255) and Rejected status.
        _status[_id] = OVERRIDE_BIT | uint256(ICompliance.Status.Rejected);
    }

    /// @notice Called by anybody to progress the state of the deposit.
    /// @dev The caller provides the full preimage fields. The contract computes
    ///      the hash on-chain via keccak256(abi.encode(...)) and uses it to look
    ///      up the stored status. Reverts if the hash has no stored status entry
    ///      (i.e., _status[id] == 0), which covers both unknown transactions and
    ///      previously-settled approved transactions whose status was deleted.
    ///
    ///      Calls status(id) to obtain (isFinal_, currentStatus). If the status
    ///      is final (Refunded, or Pending/Rejected with the owner-override bit
    ///      set), the stored status is used directly — rule re-evaluation is
    ///      skipped. If the status is NOT final, all configured rules are
    ///      re-evaluated and the strictest outcome is applied. However, if
    ///      re-evaluation produces a status that is stricter than the stored
    ///      finalized status, the finalized status still takes precedence:
    ///      finality wins over rule escalation.
    ///
    ///      ETH flow per resolved status:
    ///      - Approved: deletes the status entry (_status[id] = 0) and calls
    ///        bridge.approved{value: held}(...) to execute the held transaction
    ///        using the caller-provided preimage fields (verified by hash). For
    ///        deposits, _mint is the forwarded ETH. For withdrawals, _value is
    ///        the forwarded ETH. Deleting the status entry prevents
    ///        double-execution (a subsequent settle on the same preimage will
    ///        find _status[id] == 0 and revert).
    ///      - Rejected: marks status as Refunded and sends the held ETH back to
    ///        the original sender (_from).
    ///      - Pending: no-op, the transaction remains held.
    ///      - Refunded: reverts (already settled, prevents double-claim).
    ///
    /// @param _from       The original depositor/withdrawer address.
    /// @param _to         The recipient address on the remote chain.
    /// @param _value      The ETH value.
    /// @param _mint       The ETH to mint on L2 (msg.value for deposits, always 0 for withdrawals).
    /// @param _gasLimit   The gas limit for remote execution.
    /// @param _isCreation Whether this is a contract creation (always false for withdrawals).
    /// @param _data       The calldata.
    /// @param _nonce      The reserved nonce (always 0 for deposits).
    function settle(
        address _from,
        address _to,
        uint256 _value,
        uint256 _mint,
        uint64 _gasLimit,
        bool _isCreation,
        bytes calldata _data,
        uint256 _nonce
    )
        external
        nonReentrant
    {
        // Recompute the hash from the caller-provided preimage.
        bytes32 id = keccak256(abi.encode(_from, _to, _value, _mint, _gasLimit, _isCreation, _data, _nonce));

        // Must have a stored status entry. _status[id] == 0 covers unknown transactions
        // and previously-settled approved transactions whose status was deleted.
        if (_status[id] == 0) revert ICompliance.Compliance_NotPending();

        (bool isFinal, ICompliance.Status current) = status(id);

        // Already settled.
        if (current == ICompliance.Status.Refunded) revert ICompliance.Compliance_NotPending();

        ICompliance.Status resolvedStatus;
        if (isFinal) {
            // Status is final (owner override set): use the stored status directly.
            resolvedStatus = current;
        } else {
            // Status is not final: re-evaluate all rules.
            resolvedStatus = _evaluateRules({
                _from: _from,
                _to: _to,
                _value: _value,
                _gasLimit: _gasLimit,
                _isCreation: _isCreation,
                _data: _data,
                _nonce: _nonce
            });
        }

        if (resolvedStatus == ICompliance.Status.Approved) {
            // Delete the status entry to prevent double-execution.
            _status[id] = 0;
            emit Approved(id);

            _executeApproved(_from, _to, _value, _mint, _gasLimit, _isCreation, _data, _nonce);
        } else if (resolvedStatus == ICompliance.Status.Rejected) {
            // Mark as Refunded and send ETH back to the sender.
            _status[id] = uint256(ICompliance.Status.Refunded);
            emit Refunded(id);

            // Refund ETH to the original sender.
            if (_mint > 0) {
                if (!SafeCall.send(_from, _mint)) revert ICompliance.Compliance_TransferFailed();
            }
        }
        // If still Pending, do nothing — leave it for later settlement.
    }

    /// @notice Executes an approved transaction by forwarding it to the bridge.
    ///         Concrete implementations override this to call the correct bridge function.
    /// @param _from       The original depositor/withdrawer address.
    /// @param _to         The recipient address on the remote chain.
    /// @param _value      The ETH value.
    /// @param _mint       The ETH to mint on L2 (msg.value for deposits, always 0 for withdrawals).
    /// @param _gasLimit   The gas limit for remote execution.
    /// @param _isCreation Whether this is a contract creation (always false for withdrawals).
    /// @param _data       The calldata.
    /// @param _nonce      The reserved nonce (always 0 for deposits).
    function _executeApproved(
        address _from,
        address _to,
        uint256 _value,
        uint256 _mint,
        uint64 _gasLimit,
        bool _isCreation,
        bytes calldata _data,
        uint256 _nonce
    )
        internal
        virtual;

    /// @notice Adds a compliance rule to the set. Reverts if already present.
    /// @param _rule The address of the rule contract.
    function addRule(address _rule) external onlyOwner {
        if (!_rules.add(_rule)) revert ICompliance.Compliance_DuplicateRule();
    }

    /// @notice Removes a compliance rule from the set. Reverts if not found.
    /// @param _rule The address of the rule contract to remove.
    function removeRule(address _rule) external onlyOwner {
        if (!_rules.remove(_rule)) revert ICompliance.Compliance_RuleNotFound();
    }
}
