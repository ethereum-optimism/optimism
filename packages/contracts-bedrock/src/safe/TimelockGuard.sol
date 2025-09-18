// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Safe
import { GnosisSafe as Safe } from "safe-contracts/GnosisSafe.sol";
import { Enum } from "safe-contracts/common/Enum.sol";
import { GuardManager, Guard as IGuard } from "safe-contracts/base/GuardManager.sol";
import { ExecTransactionParams } from "src/safe/Types.sol";

// Interfaces
import { ISemver } from "interfaces/universal/ISemver.sol";

/// @title TimelockGuard
/// @notice This guard provides timelock functionality for Safe transactions
/// @dev This is a singleton contract. To use it:
///      1. The Safe must first enable this guard using GuardManager.setGuard()
///      2. The Safe must then configure the guard by calling configureTimelockGuard()
contract TimelockGuard is IGuard, ISemver {
    /// @notice Configuration for a Safe's timelock guard
    struct GuardConfig {
        uint256 timelockDelay;
        bool configured;
    }

    /// @notice Scheduled transaction
    struct ScheduledTransaction {
        uint256 executionTime;
        bool cancelled;
        bool executed;
    }

    /// @notice Mapping from Safe address to its guard configuration
    mapping(Safe => GuardConfig) public safeConfigs;

    /// @notice Mapping from Safe and tx id to scheduled transaction.
    mapping(Safe => mapping(bytes32 => ScheduledTransaction)) internal scheduledTransactions;

    /// @notice Mapping from Safe to cancellation threshold.
    mapping(Safe => uint256) internal safeCancellationThreshold;

    /// @notice Error for when guard is not enabled for the Safe
    error TimelockGuard_GuardNotEnabled();

    /// @notice Error for when Safe is not configured for this guard
    error TimelockGuard_GuardNotConfigured();

    /// @notice Error for when attempt to clear guard while it is still enabled for the Safe
    error TimelockGuard_GuardStillEnabled();

    /// @notice Error for invalid timelock delay
    error TimelockGuard_InvalidTimelockDelay();

    /// @notice Error for when a transaction is already scheduled
    error TimelockGuard_TransactionAlreadyScheduled();

    /// @notice Error for when a transaction is already cancelled
    error TimelockGuard_TransactionAlreadyCancelled();

    /// @notice Error for when a transaction is not scheduled
    error TimelockGuard_TransactionNotScheduled();

    /// @notice Error for when a transaction is not ready to execute (timelock delay not passed)
    error TimelockGuard_TransactionNotReady();

    /// @notice Emitted when a Safe configures the guard
    event GuardConfigured(Safe indexed safe, uint256 timelockDelay);

    /// @notice Emitted when a Safe clears the guard configuration
    event GuardCleared(Safe indexed safe);

    /// @notice Emitted when a transaction is scheduled for a Safe.
    /// @param safe The Safe whose transaction is scheduled.
    /// @param txId The identifier of the scheduled transaction (nonce-independent).
    /// @param when The timestamp when execution becomes valid.
    event TransactionScheduled(Safe indexed safe, bytes32 indexed txId, uint256 when);

    /// @notice Emitted when a transaction is cancelled for a Safe.
    /// @param safe The Safe whose transaction is cancelled.
    /// @param txId The identifier of the cancelled transaction (nonce-independent).
    event TransactionCancelled(Safe indexed safe, bytes32 indexed txId);

    /// @notice Emitted when the cancellation threshold is updated
    event CancellationThresholdUpdated(Safe indexed safe, uint256 oldThreshold, uint256 newThreshold);

    /// @notice Emitted when a transaction is executed for a Safe.
    /// @param safe The Safe whose transaction is executed.
    /// @param nonce The nonce of the Safe for the transaction being executed.
    /// @param txHash The identifier of the executed transaction (nonce-independent).
    event TransactionExecuted(Safe indexed safe, uint256 indexed nonce, bytes32 txHash);

    /// @notice Semantic version.
    /// @custom:semver 1.0.0
    string public constant version = "1.0.0";

    /// @notice Returns the timelock delay for a given Safe
    /// @dev MUST never revert
    /// @param _safe The Safe address to query
    /// @return The timelock delay in seconds
    function viewTimelockGuardConfiguration(Safe _safe) public view returns (GuardConfig memory) {
        return safeConfigs[_safe];
    }

    /// @notice Returns the scheduled transaction for a given Safe and tx hash
    /// @dev This function is necessary to properly expose the scheduledTransactions mapping, as
    ///      simply making the mapping public will return a tuple instead of a struct.
    function getScheduledTransaction(Safe _safe, bytes32 _txHash) public view returns (ScheduledTransaction memory) {
        return scheduledTransactions[_safe][_txHash];
    }

    /// @notice Configure the contract as a timelock guard by setting the timelock delay
    /// @dev MUST allow an arbitrary number of Safe contracts to use the contract as a guard
    /// @dev MUST revert if the contract is not enabled as a guard for the Safe
    /// @dev MUST revert if timelock_delay is longer than 1 year
    /// @dev MUST set the caller as a Safe
    /// @dev MUST take timelock_delay as a parameter and store it as related to the Safe
    /// @dev MUST emit a GuardConfigured event with at least timelock_delay as a parameter
    /// @param _timelockDelay The timelock delay in seconds
    function configureTimelockGuard(uint256 _timelockDelay) external {
        Safe callingSafe = Safe(payable(msg.sender));
        // Validate timelock delay - must be non-zero and not longer than 1 year
        if (_timelockDelay == 0 || _timelockDelay > 365 days) {
            revert TimelockGuard_InvalidTimelockDelay();
        }

        // Check that this guard is enabled on the calling Safe
        if (!_isGuardEnabled(callingSafe)) {
            revert TimelockGuard_GuardNotEnabled();
        }

        // Store the configuration for this safe
        safeConfigs[callingSafe].timelockDelay = _timelockDelay;
        safeConfigs[callingSafe].configured = true;

        // Initialize cancellation threshold to 1
        safeCancellationThreshold[callingSafe] = 1;

        emit GuardConfigured(callingSafe, _timelockDelay);
    }

    /// @notice Remove the timelock guard configuration by a previously enabled Safe
    /// @dev MUST revert if the contract is not enabled as a guard for the Safe
    /// @dev MUST erase the existing timelock_delay data related to the calling Safe
    /// @dev MUST emit a GuardCleared event
    function clearTimelockGuard() external {
        Safe callingSafe = Safe(payable(msg.sender));
        // Check if the calling safe has configuration set
        if (safeConfigs[callingSafe].configured == false) {
            revert TimelockGuard_GuardNotConfigured();
        }

        // Check that this guard is NOT enabled on the calling Safe
        if (_isGuardEnabled(callingSafe)) {
            revert TimelockGuard_GuardStillEnabled();
        }

        // Erase the configuration data for this safe
        delete safeConfigs[callingSafe];

        emit GuardCleared(callingSafe);
    }

    /// @notice Returns the cancellation threshold for a given safe
    /// @dev MUST NOT revert
    /// @dev MUST return 0 if the contract is not enabled as a guard for the safe
    /// @param _safe The Safe address to query
    /// @return The current cancellation threshold
    function cancellationThreshold(Safe _safe) public view returns (uint256) {
        // Return 0 if guard is not enabled
        if (!_isGuardEnabled(_safe)) {
            return 0;
        }

        return safeCancellationThreshold[_safe];
    }

    /// @notice Returns the blocking threshold threshold for a given safe
    /// @dev MUST NOT revert
    /// @return The current blocking threshold
    function blockingThreshold(Safe /* _safe */ ) public pure returns (uint256) {
        // TODO: Implement this
        return 10;
        // return min(quorum, total_owners - quorum + 1) for _safe;
    }

    /// @notice Internal helper to get the guard address from a Safe
    /// @param _safe The Safe address
    /// @return The current guard address
    function _isGuardEnabled(Safe _safe) internal view returns (bool) {
        // keccak256("guard_manager.guard.address") from GuardManager
        bytes32 guardSlot = 0x4a204f620c8c5ccdca3fd54d003badd85ba500436a431f0cbda4f558c93c34c8;
        address guard = abi.decode(_safe.getStorageAt(uint256(guardSlot), 1), (address));
        return guard == address(this);
    }

    /// @notice Schedule a transaction for execution after the timelock delay.
    /// @dev Minimal implementation: checks enabled+configured, uniqueness, cancellation, stores execution time and
    /// emits.
    /// @dev The txId is computed independent of Safe nonce using all exec params (with keccak(data)).
    function scheduleTransaction(
        Safe _safe,
        uint256 _nonce,
        ExecTransactionParams memory _params,
        bytes memory _signatures
    )
        external
    {
        // Check that this guard is enabled on the calling Safe
        if (!_isGuardEnabled(_safe)) {
            revert TimelockGuard_GuardNotEnabled();
        }

        // Check that the guard has been configured for the Safe
        if (!safeConfigs[_safe].configured) {
            revert TimelockGuard_GuardNotConfigured();
        }

        // Get the encoded transaction data as defined in the Safe
        // The format of the string returned is: "0x1901{domainSeparator}{safeTxHash}"
        bytes memory txHashData = _safe.encodeTransactionData(
            _params.to,
            _params.value,
            _params.data,
            _params.operation,
            _params.safeTxGas,
            _params.baseGas,
            _params.gasPrice,
            _params.gasToken,
            _params.refundReceiver,
            _nonce
        );

        // Get the transaction hash and data as defined in the Safe
        // This value is identical to keccak256(txHashData), but we prefer to use the Safe's own
        // internal logic as it is more future-proof in case future versions of the Safe change
        // the transaction hash derivation.
        bytes32 txHash = _safe.getTransactionHash(
            _params.to,
            _params.value,
            _params.data,
            _params.operation,
            _params.safeTxGas,
            _params.baseGas,
            _params.gasPrice,
            _params.gasToken,
            _params.refundReceiver,
            _nonce
        );

        // Check if the transaction exists
        // A transaction can only be scheduled once, regardless of whether it has been cancelled or not.
        if (scheduledTransactions[_safe][txHash].executionTime != 0) {
            revert TimelockGuard_TransactionAlreadyScheduled();
        }

        // Verify signatures using the Safe's signature checking logic
        // This function call reverts if the signatures are invalid.
        _safe.checkSignatures(txHash, txHashData, _signatures);

        // Calculate the execution time
        uint256 executionTime = block.timestamp + safeConfigs[_safe].timelockDelay;

        // Schedule the transaction
        scheduledTransactions[_safe][txHash] =
            ScheduledTransaction({ executionTime: executionTime, cancelled: false, executed: false });

        emit TransactionScheduled(_safe, txHash, executionTime);
    }

    /// @notice Returns the list of all scheduled but not cancelled transactions for a given safe
    /// @dev MUST NOT revert - NOT IMPLEMENTED YET
    /// @return List of pending transaction hashes
    function checkPendingTransactions(address) external pure returns (bytes32[] memory) {
        return new bytes32[](0);
    }

    /// @notice Cancel a scheduled transaction if cancellation threshold is met
    /// @dev This function aims to mimic the approach which would be used by a quorum of signers to
    ///      cancel a partially signed transaction, which would be to sign and execute an empty
    ///      transaction at the same nonce.
    ///      This enables us to deterministically generate the transaction inputs for a cancellation
    ///      transaction from the transaction being cancelled.
    ///      In this case however we cannot use a completely empty transaction (with all inputs other than the nonce
    /// being null),
    ///      as that would allow for the signatures used to cancel one transaction at nonce X to
    ///      be used to cancel all transactions at nonce X.
    ///
    ///      Therefore we define a custom set of inputs for a cancellation transaction, based on the
    ///      Safe's address as well as the nonce and hash of the transaction being cancelled.
    ///
    ///      Since the Safe's checkNSignatures function is used, the owner can use any method
    ///      to sign the cancellation transaction inputs, including signing with a private key,
    ///      calling the Safe's approveHash function, or EIP1271 contract signatures.
    function cancelTransaction(Safe _safe, bytes32 _txHash, uint256 _nonce, bytes memory _signatures) external {
        if (scheduledTransactions[_safe][_txHash].cancelled) {
            revert TimelockGuard_TransactionAlreadyCancelled();
        }
        if (scheduledTransactions[_safe][_txHash].executionTime == 0) {
            revert TimelockGuard_TransactionNotScheduled();
        }

        // Generate the cancellation transaction data
        bytes memory txData = abi.encodeWithSignature("cancelTransaction(bytes32)", _txHash);
        bytes memory cancellationTxData = _safe.encodeTransactionData(
            address(_safe), 0, txData, Enum.Operation.Call, 0, 0, 0, address(0), address(0), _nonce
        );
        bytes32 cancellationTxHash = _safe.getTransactionHash(
            address(_safe), 0, txData, Enum.Operation.Call, 0, 0, 0, address(0), address(0), _nonce
        );

        // Verify signatures using the Safe's signature checking logic
        // This function call reverts if the signatures are invalid.
        _safe.checkNSignatures(cancellationTxHash, cancellationTxData, _signatures, safeCancellationThreshold[_safe]);

        scheduledTransactions[_safe][_txHash].cancelled = true;
        increaseCancellationThreshold(_safe);

        emit TransactionCancelled(_safe, _txHash);
    }

    /// @notice Increase the cancellation threshold for a safe
    /// @dev This function must be caled only once and only when calling cancel
    function increaseCancellationThreshold(Safe _safe) internal {
        if (safeCancellationThreshold[_safe] < blockingThreshold(_safe)) {
            uint256 oldThreshold = safeCancellationThreshold[_safe];
            safeCancellationThreshold[_safe]++;
            emit CancellationThresholdUpdated(_safe, oldThreshold, safeCancellationThreshold[_safe]);
        }
    }

    /// @notice Reset the cancellation threshold for a safe
    /// @dev This function must be called only once and only when calling checkAfterExecution
    function resetCancellationThreshold(Safe _safe) internal {
        uint256 oldThreshold = safeCancellationThreshold[_safe];
        safeCancellationThreshold[_safe] = 1;
        emit CancellationThresholdUpdated(_safe, oldThreshold, 1);
    }

    /// @notice Called by the Safe before executing a transaction
    /// @dev Implementation of IGuard interface
    function checkTransaction(
        address _to,
        uint256 _value,
        bytes memory _data,
        Enum.Operation _operation,
        uint256 _safeTxGas,
        uint256 _baseGas,
        uint256 _gasPrice,
        address _gasToken,
        address payable _refundReceiver,
        bytes memory,
        address
    )
        external
        override
    {
        Safe callingSafe = Safe(payable(msg.sender));

        if (safeConfigs[callingSafe].configured == false) {
            // We return immediately. This is important in order to allow a Safe which has the
            // guard set, but not configured to complete the setup process.
            // It is also just a reasonable thing to do, since an unconfigured Safe must have a
            // delay of zero.
            return;
        }

        // Get the nonce of the Safe for the transaction being executed,
        // since the Safe's nonce is incremented before the transaction is executed,
        // we must subtract 1.
        uint256 nonce = callingSafe.nonce() - 1;

        // Get the transaction hash from the Safe's getTransactionHash function
        bytes32 txHash = callingSafe.getTransactionHash(
            _to, _value, _data, _operation, _safeTxGas, _baseGas, _gasPrice, _gasToken, _refundReceiver, nonce
        );

        // Get the scheduled transaction
        ScheduledTransaction storage scheduledTx = scheduledTransactions[callingSafe][txHash];

        // Check if the transaction has been scheduled
        if (scheduledTx.executionTime == 0) {
            revert TimelockGuard_TransactionNotScheduled();
        }

        // Check if the transaction was cancelled
        if (scheduledTx.cancelled) {
            revert TimelockGuard_TransactionAlreadyCancelled();
        }

        // Check if the timelock delay has passed
        if (scheduledTx.executionTime > block.timestamp) {
            revert TimelockGuard_TransactionNotReady();
        }

        // Set the transaction as executed
        scheduledTx.executed = true;

        // Reset the cancellation threshold
        resetCancellationThreshold(callingSafe);

        emit TransactionExecuted(callingSafe, nonce, txHash);
    }

    /// @notice Called by the Safe after executing a transaction
    /// @dev Implementation of IGuard interface
    function checkAfterExecution(bytes32, bool) external override {
        // TODO: Implement
        // extract txHash
        // resetCancellationThreshold(_safe, txHash)
        // scheduledTransactions[_safe][txHash].executed = true
    }
}
