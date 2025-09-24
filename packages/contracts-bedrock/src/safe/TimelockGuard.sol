// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Safe
import { GnosisSafe as Safe } from "safe-contracts/GnosisSafe.sol";
import { Enum } from "safe-contracts/common/Enum.sol";
import { Guard as IGuard } from "safe-contracts/base/GuardManager.sol";
import { ExecTransactionParams } from "src/safe/Types.sol";

// Libraries
import { EnumerableSet } from "@openzeppelin/contracts/utils/structs/EnumerableSet.sol";

// Interfaces
import { ISemver } from "interfaces/universal/ISemver.sol";

/// @title TimelockGuard
/// @notice This guard provides timelock functionality for Safe transactions
/// @dev This is a singleton contract. To use it:
///      1. The Safe must first enable this guard using GuardManager.setGuard()
///      2. The Safe must then configure the guard by calling configureTimelockGuard()
contract TimelockGuard is IGuard, ISemver {
    using EnumerableSet for EnumerableSet.Bytes32Set;

    /// @notice Configuration for a Safe's timelock guard
    struct GuardConfig {
        uint256 timelockDelay;
        uint256 safetyDelay;
        bool safetyDelayEnabled;
    }

    /// @notice Scheduled transaction
    struct ScheduledTransaction {
        uint256 executionTime;
        bool cancelled;
        bool executed;
        ExecTransactionParams params;
    }

    /// @notice Mapping from Safe address to its guard configuration
    mapping(Safe => GuardConfig) internal _timelockSafeConfiguration;

    /// @notice Mapping from Safe and tx id to scheduled transaction.
    mapping(Safe => mapping(bytes32 => ScheduledTransaction)) internal _scheduledTransactions;

    /// @notice Mapping from a Safe to an enumerable set of tx hashes used to store the list of tx
    ///         hashes which have been scheduled, but not yet exeuted or cancelled.
    mapping(Safe => EnumerableSet.Bytes32Set) internal _safePendingTxHashes;

    /// @notice Mapping from Safe to cancellation threshold.
    mapping(Safe => uint256) internal _safeCancellationThreshold;

    /// @notice Error for when guard is not enabled for the Safe
    error TimelockGuard_GuardNotEnabled();

    /// @notice Error for when Safe is not configured for this guard
    error TimelockGuard_GuardNotConfigured();

    /// @notice Error for invalid timelock delay
    error TimelockGuard_InvalidTimelockDelay();

    /// @notice Error for invalid safety delay
    error TimelockGuard_InvalidSafetyDelay();

    /// @notice Error for when a transaction is already scheduled
    error TimelockGuard_TransactionAlreadyScheduled();

    /// @notice Error for when a transaction is already cancelled
    error TimelockGuard_TransactionAlreadyCancelled();

    /// @notice Error for when a transaction is not scheduled
    error TimelockGuard_TransactionNotScheduled();

    /// @notice Error for when a transaction is not ready to execute (timelock delay not passed)
    error TimelockGuard_TransactionNotReady();

    /// @notice Error for when a transaction has already been executed
    error TimelockGuard_TransactionAlreadyExecuted();

    /// @notice Emitted when a Safe configures the guard
    event GuardConfigured(Safe indexed safe, uint256 timelockDelay, uint256 safetyDelay);

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

    /// @notice Emitted when the safety delay is toggled for a Safe
    /// @param safe The Safe whose safety delay is toggled.
    /// @param enabled Whether the safety delay is enabled.
    event SafetyDelayToggled(Safe indexed safe, bool enabled);

    /// @notice Semantic version.
    /// @custom:semver 1.0.0
    string public constant version = "1.0.0";

    /// @notice Returns the timelock delay for a given Safe
    /// @param _safe The Safe address to query
    /// @return The timelock delay in seconds
    function timelockConfigurationForSafe(Safe _safe) public view returns (GuardConfig memory) {
        return _timelockSafeConfiguration[_safe];
    }

    /// @notice Returns the scheduled transaction for a given Safe and tx hash
    /// @dev This function is necessary to properly expose the scheduledTransactions mapping, as
    ///      simply making the mapping public will return a tuple instead of a struct.
    function scheduledTransactionForSafe(
        Safe _safe,
        bytes32 _txHash
    )
        public
        view
        returns (ScheduledTransaction memory)
    {
        return _scheduledTransactions[_safe][_txHash];
    }

    /// @notice Returns the list of all scheduled but not cancelled or executed transactions for
    /// for a given safe
    /// @dev WARNING: This operation will copy the entire set of pending transactions to memory,
    /// which can be quite expensive. This is designed only to be used by view accessors that are
    /// queried without any gas fees. Developers should keep in mind that this function has an
    /// unbounded cost, and using it as part of a state-changing function may render the function
    /// uncallable if the set grows to a point where copying to memory consumes too much gas to fit
    /// in a block.
    /// @return List of pending transaction hashes
    function pendingTransactionsForSafe(Safe _safe) external view returns (ScheduledTransaction[] memory) {
        bytes32[] memory hashes = _safePendingTxHashes[_safe].values();
        ScheduledTransaction[] memory scheduled = new ScheduledTransaction[](hashes.length);
        for (uint256 i = 0; i < hashes.length; i++) {
            scheduled[i] = _scheduledTransactions[_safe][hashes[i]];
        }
        return scheduled;
    }

    /// @notice Configure the contract as a timelock guard by setting the timelock delay
    /// @param _timelockDelay The timelock delay in seconds (0 to clear configuration)
    function configureTimelockGuard(uint256 _timelockDelay, uint256 _safetyDelay) external {
        Safe callingSafe = Safe(payable(msg.sender));

        // Check that this guard is enabled on the calling Safe
        if (!_isGuardEnabled(callingSafe)) {
            revert TimelockGuard_GuardNotEnabled();
        }

        // Check that the timelock delay is not longer than 1 year
        if (_timelockDelay > 365 days) {
            revert TimelockGuard_InvalidTimelockDelay();
        }

        // Check that the safety delay is not longer than 1 year
        if (_safetyDelay > 365 days) {
            revert TimelockGuard_InvalidSafetyDelay();
        }

        GuardConfig storage config = _timelockSafeConfiguration[callingSafe];
        config.timelockDelay = _timelockDelay;
        config.safetyDelay = _safetyDelay;

        _safeCancellationThreshold[callingSafe] = 1;

        emit GuardConfigured(callingSafe, _timelockDelay, _safetyDelay);
    }

    /// @notice Returns the blocking threshold threshold for a given safe
    /// @return The current blocking threshold
    function _blockingThreshold(Safe _safe) internal view returns (uint256) {
        // The blocking threshold is the number of owners who can coordinate to block a transaction
        // from being executed by refusing to sign.
        return _safe.getOwners().length - _safe.getThreshold() + 1;
    }

    /// @notice Returns the cancellation threshold for a given safe
    /// @param _safe The Safe address to query
    /// @return The current cancellation threshold
    function cancellationThresholdForSafe(Safe _safe) public view returns (uint256) {
        // Return 0 if guard is not enabled
        if (!_isGuardEnabled(_safe)) {
            return 0;
        }

        return _safeCancellationThreshold[_safe];
    }

    /// @notice Returns the maximum cancellation threshold for a given safe
    /// @return The maximum cancellation threshold
    function maxCancellationThreshold(Safe _safe) public view returns (uint256) {
        uint256 blockingThreshold = _blockingThreshold(_safe);
        uint256 quorum = _safe.getThreshold();
        // Return the minimum of the blocking threshold and the quorum
        return (blockingThreshold < quorum ? blockingThreshold : quorum) - 1;
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
        if (_timelockSafeConfiguration[_safe].timelockDelay == 0) {
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
        if (_scheduledTransactions[_safe][txHash].executionTime != 0) {
            revert TimelockGuard_TransactionAlreadyScheduled();
        }

        // Verify signatures using the Safe's signature checking logic
        // This function call reverts if the signatures are invalid.
        _safe.checkSignatures(txHash, txHashData, _signatures);

        // Calculate the execution time
        uint256 executionTime = block.timestamp + _timelockSafeConfiguration[_safe].timelockDelay;

        // Schedule the transaction
        _scheduledTransactions[_safe][txHash] =
            ScheduledTransaction({ executionTime: executionTime, cancelled: false, executed: false, params: _params });
        _safePendingTxHashes[_safe].add(txHash);

        emit TransactionScheduled(_safe, txHash, executionTime);
    }

    /// @notice Cancel a scheduled transaction if cancellation threshold is met
    /// @dev This function aims to mimic the approach which would be used by a quorum of signers to
    ///      cancel a partially signed transaction, by signing and executing an empty
    ///      transaction at the same nonce.
    ///      This enables us to define a standard "cancellation transaction" format using the Safe address, nonce,
    ///      and hash of the transaction being cancelled. This is necessary to ensure that the cancellation transaction
    ///      is unique and cannot be used to cancel another transaction at the same nonce.
    ///
    ///      Signature verificiation uses the Safe's checkNSignatures function, so that the number of signatures required
    ///      can be set by the Safe's current cancellation threshold. Another benefit of checkNSignatures is that owners
    ///      can use any method to sign the cancellation transaction inputs, including signing with a private key,
    ///      calling the Safe's approveHash function, or EIP1271 contract signatures.
    function cancelTransaction(Safe _safe, bytes32 _txHash, uint256 _nonce, bytes memory _signatures) external {
        if (_scheduledTransactions[_safe][_txHash].cancelled) {
            revert TimelockGuard_TransactionAlreadyCancelled();
        }
        if (_scheduledTransactions[_safe][_txHash].executed) {
            revert TimelockGuard_TransactionAlreadyExecuted();
        }
        if (_scheduledTransactions[_safe][_txHash].executionTime == 0) {
            revert TimelockGuard_TransactionNotScheduled();
        }

        // Generate the cancellation transaction data
        bytes memory data = abi.encodeWithSignature("cancelTransaction(bytes32)", _txHash);
        bytes memory cancellationTxData = _safe.encodeTransactionData(
            address(_safe), 0, data, Enum.Operation.Call, 0, 0, 0, address(0), address(0), _nonce
        );
        bytes32 cancellationTxHash = _safe.getTransactionHash(
            address(_safe), 0, data, Enum.Operation.Call, 0, 0, 0, address(0), address(0), _nonce
        );

        // Verify signatures using the Safe's signature checking logic
        // This function call reverts if the signatures are invalid.
        _safe.checkNSignatures(cancellationTxHash, cancellationTxData, _signatures, _safeCancellationThreshold[_safe]);

        _scheduledTransactions[_safe][_txHash].cancelled = true;
        _safePendingTxHashes[_safe].remove(_txHash);
        _increaseCancellationThreshold(_safe);

        emit TransactionCancelled(_safe, _txHash);
    }

    /// @notice Toggle the safety delay for a Safe
    /// @dev This function is used to add or remove a safety delay to all transactions executed by a Safe. Similar to
    ///      cancelTransaction, it uses a custom format for the transaction data and uses the Safe's
    ///      signature checking logic to verify the signatures. The current nonce of the Safe is used to
    ///      ensure that the safety delay transaction is unique and cannot be replayed in the future.
    /// @param _safe The Safe address
    /// @param _signatures The signatures of the owners
    function toggleSafetyDelay(Safe _safe, bool _enableSafetyDelay, bytes memory _signatures) external {
        // Get the current Safe nonce
        uint256 nonce = _safe.nonce();

        // Generate the safety delay transaction data
        bytes memory data = abi.encodeWithSignature("toggleSafetyDelay(bool)", _enableSafetyDelay);
        bytes memory safetyDelayTxData = _safe.encodeTransactionData(
            address(_safe), 0, data, Enum.Operation.Call, 0, 0, 0, address(0), address(0), nonce
        );
        bytes32 safetyDelayTxHash = _safe.getTransactionHash(
            address(_safe), 0, data, Enum.Operation.Call, 0, 0, 0, address(0), address(0), nonce
        );

        // Verify signatures using the Safe's signature checking logic
        // This function call reverts if the signatures are invalid.
        _safe.checkSignatures(safetyDelayTxHash, safetyDelayTxData, _signatures);

        _timelockSafeConfiguration[_safe].safetyDelayEnabled = _enableSafetyDelay;

        emit SafetyDelayToggled(_safe, _enableSafetyDelay);
    }

    /// @notice Increase the cancellation threshold for a safe
    /// @dev This function must be called only once and only when calling cancel
    function _increaseCancellationThreshold(Safe _safe) internal {
        if (_safeCancellationThreshold[_safe] < maxCancellationThreshold(_safe)) {
            uint256 oldThreshold = _safeCancellationThreshold[_safe];
            _safeCancellationThreshold[_safe]++;
            emit CancellationThresholdUpdated(_safe, oldThreshold, _safeCancellationThreshold[_safe]);
        }
    }

    /// @notice Reset the cancellation threshold for a safe
    /// @dev This function must be called only once and only when calling checkAfterExecution
    function _resetCancellationThreshold(Safe _safe) internal {
        uint256 oldThreshold = _safeCancellationThreshold[_safe];
        _safeCancellationThreshold[_safe] = 1;
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

        if (_timelockSafeConfiguration[callingSafe].timelockDelay == 0) {
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
        ScheduledTransaction storage scheduledTx = _scheduledTransactions[callingSafe][txHash];

        // Check if the transaction was cancelled
        if (scheduledTx.cancelled) {
            revert TimelockGuard_TransactionAlreadyCancelled();
        }

        // Check if the transaction has been scheduled
        if (scheduledTx.executionTime == 0) {
            revert TimelockGuard_TransactionNotScheduled();
        }

        // Check if the timelock delay has passed
        if (scheduledTx.executionTime > block.timestamp) {
            revert TimelockGuard_TransactionNotReady();
        }

        // Check if the transaction has already been executed
        // Note: this is of course enforced by the Safe itself, but we check it here for
        // completeness
        if (scheduledTx.executed) {
            revert TimelockGuard_TransactionAlreadyExecuted();
        }

        // Set the transaction as executed
        scheduledTx.executed = true;
        _safePendingTxHashes[callingSafe].remove(txHash);

        // Reset the cancellation threshold
        _resetCancellationThreshold(callingSafe);

        emit TransactionExecuted(callingSafe, nonce, txHash);
    }

    /// @notice Called by the Safe after executing a transaction
    /// @dev Implementation of IGuard interface
    function checkAfterExecution(bytes32, bool) external override {
        // Do nothing
        // In order to follow the Checks-Effects-Interactions pattern,
        // all logic should be done in the checkTransaction function.
    }
}
