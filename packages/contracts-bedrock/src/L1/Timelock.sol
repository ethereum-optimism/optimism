// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { GnosisSafe } from "safe-contracts/GnosisSafe.sol";
import { EnumerableSet } from "@openzeppelin/contracts/utils/structs/EnumerableSet.sol";
import { Initializable } from "@openzeppelin/contracts/proxy/utils/Initializable.sol";

import { ISemver } from "interfaces/universal/ISemver.sol";
import { ReinitializableBase } from "src/universal/ReinitializableBase.sol";

/// @custom:proxied true
/// @title Timelock
/// @notice The Timelock contract is used as the owner for OP Stack contracts.
contract Timelock is ISemver, Initializable, ReinitializableBase {
    using EnumerableSet for EnumerableSet.AddressSet;

    /// @notice Parameters for a transaction to be approved and executed.
    /// @custom:field salt Acts as a unique identifier for the call.
    /// @custom:field target The address of the contract to call.
    /// @custom:field value The value to send with the call.
    /// @custom:field data The data to send with the call.
    struct Call {
        bytes32 salt;
        address target;
        uint256 value;
        bytes data;
    }

    /// @notice Status of a transaction to be approved and executed.
    /// @custom:field executed True if the transaction has been executed.
    /// @custom:field cancelled True if the transaction has been cancelled.
    /// @custom:field eta The earliest timestamp at which the transaction can be executed.
    /// @custom:field approvals The addresses of the controllers that have approved the transaction.
    struct CallStatus {
        bool executed;
        bool cancelled;
        uint256 eta;
        EnumerableSet.AddressSet approvals;
    }

    /// @notice Semantic version.
    /// @custom:semver 0.0.1
    string public constant version = "0.0.1";

    /// @notice Standard longer delay, in seconds, before a call can be executed.
    uint256 public longDelay;

    /// @notice Shorter delay, in seconds, before a call can be executed if all controllers have approved.
    uint256 public shortDelay;

    /// @notice The set of controllers that can approve calls.
    EnumerableSet.AddressSet internal controllers;

    /// @notice A mapping of call hashes to their status.
    mapping(bytes32 => CallStatus) internal calls;

    /// @notice Thrown when controllers array is empty.
    error Timelock_EmptyControllers();

    /// @notice Thrown when controllers array has only one controller.
    error Timelock_SingleController();

    /// @notice Thrown when long delay is not greater than short delay.
    error Timelock_ReversedDelays();

    /// @notice Thrown when long delay is zero.
    error Timelock_LongDelayZero();

    /// @notice Thrown when short delay is zero.
    error Timelock_ShortDelayZero();

    /// @notice Thrown when long delay exceeds maximum allowed value.
    error Timelock_LongDelayTooLarge();

    /// @notice Thrown when short delay exceeds maximum allowed value.
    error Timelock_ShortDelayTooLarge();

    /// @notice Thrown when controllers are not sorted or not unique.
    error Timelock_ControllersNotSortedOrUnique();

    /// @notice Thrown when trying to approve a call that has already been approved by the caller.
    error Timelock_AlreadyApproved();

    /// @notice Thrown when trying to cancel a call that has already been cancelled.
    error Timelock_AlreadyCancelled();

    /// @notice Thrown when trying to approve or execute a call that has already been executed.
    error Timelock_AlreadyExecuted();

    /// @notice Thrown when a call execution fails.
    error Timelock_CallFailed(bytes revertData);

    /// @notice Thrown when trying to execute a call that has been cancelled.
    error Timelock_CallCancelled();

    /// @notice Thrown when trying to execute a call before its ETA.
    error Timelock_EtaNotReached();

    /// @notice Thrown when an invalid controller address is provided.
    error Timelock_InvalidController();

    /// @notice Thrown when the caller is not authorized to perform the action.
    error Timelock_NotAuthorized();

    /// @notice Emitted when a call is approved by a controller.
    /// @param txHash The hash of the approved transaction.
    /// @param call The call that was approved.
    /// @param eta The earliest time the call can be executed.
    event Approved(bytes32 indexed txHash, Call call, uint256 eta);

    /// @notice Emitted when a call is cancelled by a controller.
    /// @param txHash The hash of the cancelled transaction.
    event Cancelled(bytes32 indexed txHash);

    /// @notice Emitted when a call is executed.
    /// @param txHash The hash of the executed transaction.
    /// @param call The call that was executed.
    event Executed(bytes32 indexed txHash, Call call);

    /// @notice Constructs the Timelock contract.
    ///         Disables initializers to prevent the implementation contract from being initialized.
    constructor() ReinitializableBase(1) {
        _disableInitializers();
    }

    /// @notice Initializes the Timelock contract.
    /// @param _controllers The addresses of the controllers that can approve and cancel calls.
    /// @param _longDelay The standard longer delay, in seconds, before a call can be executed.
    /// @param _shortDelay The shorter delay, in seconds, before a call can be executed if all controllers have
    /// approved.
    function initialize(
        address[] memory _controllers,
        uint256 _longDelay,
        uint256 _shortDelay
    )
        external
        reinitializer(initVersion())
    {
        if (_controllers.length == 0) revert Timelock_EmptyControllers();

        // We disallow the single-controller case because every approval would immediately trigger
        // the "all controllers approved" short delay case, which is not the intended use case.
        if (_controllers.length == 1) revert Timelock_SingleController();

        if (_longDelay <= _shortDelay) revert Timelock_ReversedDelays();
        if (_longDelay == 0) revert Timelock_LongDelayZero();
        if (_shortDelay == 0) revert Timelock_ShortDelayZero();

        if (_longDelay > 180 days) revert Timelock_LongDelayTooLarge();
        if (_shortDelay > 30 days) revert Timelock_ShortDelayTooLarge();

        for (uint256 i = 0; i < _controllers.length; i++) {
            if (_controllers[i] == address(0)) revert Timelock_InvalidController();
            controllers.add(_controllers[i]);
            if (i > 0) {
                if (_controllers[i] <= _controllers[i - 1]) {
                    revert Timelock_ControllersNotSortedOrUnique();
                }
            }
        }

        longDelay = _longDelay;
        shortDelay = _shortDelay;
    }

    /// @notice Computes the unique hash for a given call. The hash is used as the unique identifier
    ///         for tracking call status and approvals.
    /// @param _call The call struct to compute the hash for.
    /// @return txHash_ The computed hash of the call.
    function hash(Call memory _call) public pure returns (bytes32 txHash_) {
        txHash_ = keccak256(abi.encode(_call));
    }

    /// @notice Approves a call for execution and sets its execution time (ETA). Only controllers
    ///         can approve calls. Each controller can only approve a call once. The ETA is set
    ///         based on the number of approvals:
    ///         - First approval: sets ETA to current time + longDelay
    ///         - All controllers approved: changes ETA to current time + shortDelay
    /// @param _call The call to approve.
    /// @return eta_ The earliest time at which the call can be executed.
    function approve(Call calldata _call) external returns (uint256 eta_) {
        if (!controllers.contains(msg.sender)) revert Timelock_NotAuthorized();

        bytes32 txHash = hash(_call);
        CallStatus storage callStatus = calls[txHash];

        // Safety checks.
        if (callStatus.approvals.contains(msg.sender)) revert Timelock_AlreadyApproved();
        if (callStatus.executed) revert Timelock_AlreadyExecuted();
        if (callStatus.cancelled) revert Timelock_AlreadyCancelled();

        // Add the approval first.
        callStatus.approvals.add(msg.sender);

        // Set ETA based on approval status. If all controllers have approved, use short delay,
        // otherwise use long delay. If a long delay already exists, it remains unchanged. This is
        // important to ensure that the long delay is not unnecessarily extended when a new
        // approval is added.
        if (callStatus.approvals.length() == controllers.length()) {
            // All controllers have approved, so we use the short delay. If the current ETA is
            // closer than the short delay, we use the current ETA to avoid extending the ETA.
            // The invariant here that ETA must never increase.
            if (block.timestamp + shortDelay < callStatus.eta) {
                callStatus.eta = block.timestamp + shortDelay;
            }
        } else if (callStatus.eta == 0) {
            // First approval, so we set the long delay.
            callStatus.eta = block.timestamp + longDelay;
        }

        emit Approved(txHash, _call, callStatus.eta);
        return callStatus.eta;
    }

    /// @notice Cancels a call, making it permanently un-executable. Any controller can cancel a
    ///         call directly, or Gnosis Safe owners can cancel on behalf of their Safe controller.
    ///         Once cancelled, a call cannot be executed.
    /// @param _txHash The hash of the call to cancel.
    /// @param _controller The controller address that is cancelling the call.
    function cancel(bytes32 _txHash, address _controller) public {
        if (!controllers.contains(_controller)) revert Timelock_InvalidController();
        if (_controller != msg.sender && !isCallerGnosisSafeOwner(_controller, msg.sender)) {
            revert Timelock_NotAuthorized();
        }

        CallStatus storage callStatus = calls[_txHash];
        if (callStatus.executed) revert Timelock_AlreadyExecuted();
        if (callStatus.cancelled) revert Timelock_AlreadyCancelled();

        callStatus.cancelled = true;
        emit Cancelled(_txHash);
    }

    /// @notice Cancels a call by its call struct hash. This is a convenience function that computes
    ///         the hash from the call struct and delegates to the main cancel function.
    /// @param _call The call struct to cancel.
    /// @param _controller The controller address that is cancelling the call.
    function cancel(Call memory _call, address _controller) external {
        cancel(hash(_call), _controller);
    }

    /// @notice Executes a call that has been approved and reached its execution time.
    /// @param _call The call to execute.
    /// @return result_ The return data or revert data from the executed call.
    function execute(Call calldata _call) external returns (bytes memory result_) {
        bytes32 txHash = hash(_call);
        CallStatus storage callStatus = calls[txHash];

        if (block.timestamp < callStatus.eta) revert Timelock_EtaNotReached();
        if (callStatus.cancelled) revert Timelock_CallCancelled();
        if (callStatus.executed) revert Timelock_AlreadyExecuted();

        callStatus.executed = true;
        emit Executed(txHash, _call);

        (bool success, bytes memory result) = _call.target.call{ value: _call.value }(_call.data);
        if (!success) revert Timelock_CallFailed(result);
        return result;
    }

    /// @notice Returns the total number of controllers.
    /// @return length_ The number of controllers in the set.
    function getControllersLength() external view returns (uint256 length_) {
        return controllers.length();
    }

    /// @notice Returns the controller address at the specified index.
    /// @param _index The index of the controller to retrieve (0-based).
    /// @return controller_ The controller address at the given index.
    function getController(uint256 _index) external view returns (address controller_) {
        return controllers.at(_index);
    }

    /// @notice Returns all controller addresses in an array.
    /// @return controllers_ Array containing all controller addresses.
    function getControllers() external view returns (address[] memory controllers_) {
        return controllers.values();
    }

    /// @notice Returns the complete status information for a call.
    /// @param _txHash The hash of the call to query.
    /// @return executed_ True if the call has been executed.
    /// @return cancelled_ True if the call has been cancelled.
    /// @return eta_ The earliest execution time for the call (0 if not approved).
    /// @return approvals_ Array of addresses that have approved the call.
    function getCall(bytes32 _txHash)
        external
        view
        returns (bool executed_, bool cancelled_, uint256 eta_, address[] memory approvals_)
    {
        CallStatus storage callStatus = calls[_txHash];
        executed_ = callStatus.executed;
        cancelled_ = callStatus.cancelled;
        eta_ = callStatus.eta;
        approvals_ = callStatus.approvals.values();
    }

    /// @notice Checks if the given address appears to be a Gnosis Safe contract. Performs a
    ///         heuristic check by calling getThreshold() and verifying the call succeeds and
    ///         returns 32 bytes of data. This is not a perfect check, but because the controllers
    ///         are trusted entities (i.e. we are not worried about a malicious controller), this
    ///         is a sufficient heuristic.
    /// @param _who The address to check.
    /// @return isGnosis_ True if the address appears to be a Gnosis Safe.
    function isGnosisSafe(address _who) internal view returns (bool isGnosis_) {
        bytes memory callData = abi.encodeWithSelector(bytes4(keccak256("getThreshold()")));
        (bool ok, bytes memory data) = _who.staticcall(callData);
        return ok && data.length == 32;
    }

    /// @notice Checks if a specific address is an owner of a Gnosis Safe controller. First verifies
    ///         the controller is a Gnosis Safe, then checks if the specified address is listed as
    ///         an owner of that Safe.
    /// @param _controller The Gnosis Safe controller address to check.
    /// @param _who The address to check for ownership.
    /// @return isOwner_ True if the address is an owner of the Gnosis Safe controller.
    function isCallerGnosisSafeOwner(address _controller, address _who) internal view returns (bool isOwner_) {
        if (!isGnosisSafe(_controller)) return false;
        GnosisSafe safe = GnosisSafe(payable(_controller));
        return safe.isOwner(_who);
    }
}
