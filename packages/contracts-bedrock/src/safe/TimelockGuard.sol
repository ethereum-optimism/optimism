// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Safe
import { Safe } from "safe-contracts/Safe.sol";
import { Enum } from "safe-contracts/common/Enum.sol";
import { BaseGuard } from "safe-contracts/base/GuardManager.sol";

// Libraries
import { EnumerableSet } from "@openzeppelin/contracts/utils/structs/EnumerableSet.sol";
import { SemverComp } from "src/libraries/SemverComp.sol";
import { Constants } from "src/libraries/Constants.sol";

/// @title TimelockGuard
/// @notice This guard provides timelock functionality for Safe transactions
/// @dev This is a singleton contract, any Safe on the network can use this guard to enforce a
///      timelock delay, and allow a subset of signers to cancel a transaction if they do not agree
///      with the execution. This provides significant security improvements over the Safe's
///      default execution mechanism, which will allow any transaction to be executed as long as it
///      is fully signed, and with no mechanism for revealing the existence of said signatures.
/// Usage:
///     In order to use this guard, the Safe must first enable it using Safe.setGuard(), and then
///     configure it by calling TimelockGuard.configureTimelockGuard().
/// Scheduling and executing transactions:
///     Once enabled and configured, all transactions executed by the Safe's execTransaction()
///     function will revert, unless the transaction has first been scheduled by calling
///     scheduleTransaction() on this contract. Because scheduleTransaction() uses the Safe's own
///     signature verification logic, the same signatures used to execute a transaction can be
///     used to schedule it.
///     Note: this guard does not apply a delay to transactions executed by modules which are
///     installed on the Safe.
/// Cancelling transactions:
///     Once a transaction has been scheduled, so long as it has not already been executed, it can
///     be cancelled by calling cancelTransaction() on this contract.
///     This mechanism allows for a subset of signers to cancel a transaction if they do not agree
///     with the execution.
///     As an 'anti-griefing' mechanism, the cancellation threshold (the number of signatures
///     required to cancel a transaction) starts at 1, and is automatically increased by 1 after
///     each cancellation.
///     The cancellation threshold is reset to 1 after any transaction is executed successfully.
/// Failed transactions:
///     The execTransaction call by the Safe doesn't revert if the called transaction fails and it
///     returns a false success value instead, bumping the nonce as with successful transactions.
///     The TimelockGuard matches this behaviour by marking failed transactions as Executed,
///     removing them from the pending transactions queue, and resetting the cancellation
///     threshold.
/// Safe Version Compatibility:
///     This guard is compatible only with Safe version 1.4.1.
/// Threats Mitigated and Integration With LivenessModule:
///     This Guard is designed to protect against a number of well-defined scenarios, defined on
///     the two axes of amount of keys compromised, and type of compromise.
///     For scenarios where the keys compromised don't amount to a blocking threshold (the number
///     of signers who must refuse to sign a transaction in order to block it from being executed),
///     regular transactions from the multisig for removal or rotation is the preferred solution.
///     For scenarios where the keys compromised are at least a blocking threshold, but not as much
///     as quorum, the LivenessModule would be used. If there is a quorum of absent keys, but no
///     significant malicious control, the LivenessModule would also be used.
///     The TimelockGuard acts when there is malicious control of a quorum of keys. If the control
///     is temporary, for example by phishing a single set of signatures, then the TimelockGuard's
///     cancellation is enough to stop the attack entirely. If the malicious control would be
///     permanent, then the TimelockGuard will buy some time to execute remediations external to
///     the compromised safe.
///     The following table summarizes the various scenarios and the course of action to take in
///     each case.
///                       +-------------------------------------------------------------------+
///                       |                        Course of action when X Number of keys...  |
/// +-----------------------------------------------------------------------------------------+
/// |                     | ... are Absent                 |  ... are Maliciously Controlled  |
/// | X Number of keys    | (Honest signers cannot sign)   |  (Malicious signers can sign)    |
/// +-----------------------------------------------------------------------------------------+
/// | 1+                  | swapOwner                      | swapOwner                        |
/// +-----------------------------------------------------------------------------------------+
/// | Blocking Threshold+ | challenge +                    | challenge +                      |
/// |                     | changeOwnershipToFallback      | changeOwnershipToFallback        |
/// +-----------------------------------------------------------------------------------------+
/// | Quorum+             | challenge +                    | cancelTransaction                |
/// |                     | changeOwnershipToFallback      |                                  |
/// +-----------------------------------------------------------------------------------------+
abstract contract TimelockGuard is BaseGuard {
    using EnumerableSet for EnumerableSet.Bytes32Set;

    /// @notice Allowed states of a transaction
    enum TransactionState {
        NotScheduled,
        Pending,
        Cancelled,
        Executed
    }

    /// @notice Scheduled transaction
    /// @custom:field txHash The hash of the transaction.
    /// @custom:field executionTime The timestamp when execution becomes valid.
    /// @custom:field state The state of the transaction.
    /// @custom:field params The parameters of the transaction.
    /// @custom:field nonce The nonce of the transaction.
    struct ScheduledTransaction {
        bytes32 txHash;
        uint256 executionTime;
        TransactionState state;
        ExecTransactionParams params;
        uint256 nonce;
    }

    /// @notice Parameters for the Safe's execTransaction function
    /// @custom:field to The address of the contract to call.
    /// @custom:field value The value to send with the transaction.
    /// @custom:field data The data to send with the transaction.
    /// @custom:field operation The operation to perform with the transaction.
    /// @custom:field safeTxGas The gas to use for the transaction.
    /// @custom:field baseGas The base gas to use for the transaction.
    /// @custom:field gasPrice The gas price to use for the transaction.
    /// @custom:field gasToken The token to use for the transaction.
    /// @custom:field refundReceiver The address to receive the refund for the transaction.
    struct ExecTransactionParams {
        address to;
        uint256 value;
        bytes data;
        Enum.Operation operation;
        uint256 safeTxGas;
        uint256 baseGas;
        uint256 gasPrice;
        address gasToken;
        address payable refundReceiver;
    }

    /// @notice Aggregated state for each Safe using this guard.
    /// @dev We have chosen for operational reasons to keep a list of pending transactions that can
    ///      be easily retrieved via a function call. This is done by maintaining a separate
    ///      EnumerableSet with the hashes of pending transactions. Transactions in the enumerable
    ///      set need to be updated along with updates to the ScheduledTransactions mapping.
    struct SafeState {
        uint256 timelockDelay;
        uint256 cancellationThreshold;
        mapping(bytes32 => ScheduledTransaction) scheduledTransactions;
        EnumerableSet.Bytes32Set pendingTxHashes;
    }

    /// @notice Mapping from Safe address to a mapping of configuration nonce to its state.
    mapping(Safe => mapping(uint256 => SafeState)) internal _safeStates;

    /// @notice Mapping from Safe address to the current configuration nonce.
    mapping(Safe => uint256) internal _safeConfigNonces;

    /// @notice Error for when guard is not enabled for the Safe
    error TimelockGuard_GuardNotEnabled();

    /// @notice Error for when Safe is not configured for this guard
    error TimelockGuard_GuardNotConfigured();

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

    /// @notice Error for when a transaction has already been executed
    error TimelockGuard_TransactionAlreadyExecuted();

    /// @notice Error for when the contract is not 1.4.1
    error TimelockGuard_InvalidVersion();

    /// @notice Error for when trying to clear guard while it is still enabled
    error TimelockGuard_GuardStillEnabled();

    /// @notice Error for when the caller is not an owner of the Safe
    error TimelockGuard_NotOwner();

    /// @notice Emitted when a Safe configures the guard
    /// @param safe The Safe whose guard is configured.
    /// @param timelockDelay The timelock delay in seconds.
    event GuardConfigured(Safe indexed safe, uint256 timelockDelay);

    /// @notice Emitted when a transaction is scheduled for a Safe.
    /// @param safe The Safe whose transaction is scheduled.
    /// @param txHash The identifier of the scheduled transaction.
    /// @param executionTime The timestamp when execution becomes valid.
    event TransactionScheduled(Safe indexed safe, bytes32 indexed txHash, uint256 executionTime);

    /// @notice Emitted when a transaction is cancelled for a Safe.
    /// @param safe The Safe whose transaction is cancelled.
    /// @param txHash The identifier of the cancelled transaction.
    event TransactionCancelled(Safe indexed safe, bytes32 indexed txHash);

    /// @notice Emitted when a transaction is executed for a Safe.
    /// @param safe The Safe whose transaction is executed.
    /// @param txHash The identifier of the executed transaction.
    event TransactionExecuted(Safe indexed safe, bytes32 indexed txHash);

    /// @notice Emitted when the cancellation threshold is updated
    /// @param safe The Safe whose cancellation threshold is updated.
    /// @param oldThreshold The old cancellation threshold.
    /// @param newThreshold The new cancellation threshold.
    event CancellationThresholdUpdated(Safe indexed safe, uint256 oldThreshold, uint256 newThreshold);

    /// @notice Used to emit a message, primarily to ensure that the cancelTransaction function is
    ///         is not labelled as view so that it is treated as a state-changing function.
    event Message(string message);

    ////////////////////////////////////////////////////////////////
    //                  Internal View Functions                   //
    ////////////////////////////////////////////////////////////////

    /// @notice Returns the blocking threshold, which is defined as the minimum number of owners
    ///         that must coordinate to block a transaction from being executed by refusing to
    ///         sign.
    /// @dev Because `_safe.getOwners()` loops through the owners list, it could run out of gas if
    ///      there are a lot of owners.
    /// @param _safe The Safe address to query
    /// @return The current blocking threshold
    function _blockingThreshold(Safe _safe) internal view returns (uint256) {
        return _safe.getOwners().length - _safe.getThreshold() + 1;
    }

    /// @notice Internal helper to check if TimelockGuard is enabled for a Safe
    /// @param _safe The Safe address
    /// @return The current guard address
    function _isGuardEnabled(Safe _safe) internal view returns (bool) {
        address guard = abi.decode(_safe.getStorageAt(uint256(Constants.GUARD_STORAGE_SLOT), 1), (address));
        return guard == address(this);
    }

    /// @notice Returns a storage reference to the current SafeState for a given Safe.
    /// @param _safe The Safe address to query.
    /// @return The current SafeState storage reference.
    function _currentSafeState(Safe _safe) internal view returns (SafeState storage) {
        return _safeStates[_safe][_safeConfigNonces[_safe]];
    }

    /// @notice Internal helper function which can be overridden in a child contract to check if the
    ///         guard's configuration is valid in the context of other extensions that are enabled
    ///         on the Safe.
    function _checkCombinedConfig(Safe _safe) internal view virtual;

    ////////////////////////////////////////////////////////////////
    //                  External View Functions                   //
    ////////////////////////////////////////////////////////////////

    /// @notice Returns the cancellation threshold for a given safe
    /// @param _safe The Safe address to query
    /// @return The current cancellation threshold
    function cancellationThreshold(Safe _safe) public view returns (uint256) {
        return _currentSafeState(_safe).cancellationThreshold;
    }

    /// @notice Returns the maximum cancellation threshold for a given safe
    /// @dev The cancellation threshold must be capped in order to preserve the ability of honest
    ///      users to cancel malicious transactions. The rationale for the calculation of the
    ///      maximum cancellation threshold is as follows:
    ///      If the quorum is lower, then it is used as the maximum cancellation threshold, so that
    ///      even if an attacker has _joint control_ of a quorum of keys, the honest users can
    ///      still indefinitely cancel a malicious transaction.
    ///      If the blocking threshold is lower, then it is used as the maximum cancellation
    ///      threshold, so that if an attacker has less than a quorum of keys, honest users can
    ///      still remove an attacker from the Safe by refusing to respond to a malicious
    ///      transaction.
    /// @param _safe The Safe address to query
    /// @return The maximum cancellation threshold
    function maxCancellationThreshold(Safe _safe) public view returns (uint256) {
        uint256 blockingThreshold = _blockingThreshold(_safe);
        uint256 quorum = _safe.getThreshold();
        // Return the minimum of the blocking threshold and the quorum
        return (blockingThreshold < quorum ? blockingThreshold : quorum);
    }

    /// @notice Returns the timelock delay for a given Safe
    /// @param _safe The Safe address to query
    /// @return The timelock delay in seconds
    function timelockDelay(Safe _safe) public view returns (uint256) {
        return _currentSafeState(_safe).timelockDelay;
    }

    /// @notice Returns the scheduled transaction for a given Safe and tx hash
    /// @dev This function is necessary to properly expose the scheduledTransactions mapping, as
    ///      simply making the mapping public will return a tuple instead of a struct.
    function scheduledTransaction(Safe _safe, bytes32 _txHash) public view returns (ScheduledTransaction memory) {
        return _currentSafeState(_safe).scheduledTransactions[_txHash];
    }

    /// @notice Returns the list of all scheduled but not cancelled or executed transactions for
    ///         for a given safe
    /// @dev WARNING: This operation will copy the entire set of pending transactions to memory,
    ///      which can be quite expensive. This is designed only to be used by view accessors that
    ///      are queried without any gas fees. Developers should keep in mind that this function
    ///      has an unbounded cost, and using it as part of a state-changing function may render
    ///      the function uncallable if the set grows to a point where copying to memory consumes
    ///      too much gas to fit in a block.
    /// @return List of pending transaction hashes
    function pendingTransactions(Safe _safe) external view returns (ScheduledTransaction[] memory) {
        SafeState storage safeState = _currentSafeState(_safe);

        // Get the list of pending transaction hashes
        bytes32[] memory hashes = safeState.pendingTxHashes.values();

        // We want to provide the caller with the full parameters of each pending transaction, but
        // mappings are not iterable, so we use the enumerable set of pending transaction hashes to
        // retrieve the ScheduledTransaction struct for each hash, and then return an array of the
        // ScheduledTransaction structs.
        ScheduledTransaction[] memory scheduled = new ScheduledTransaction[](hashes.length);
        for (uint256 i = 0; i < hashes.length; i++) {
            scheduled[i] = safeState.scheduledTransactions[hashes[i]];
        }
        return scheduled;
    }

    ////////////////////////////////////////////////////////////////
    //                 Guard Interface Functions                  //
    ////////////////////////////////////////////////////////////////

    /// @notice Implementation of Guard interface.Called by the Safe before executing a transaction
    /// @dev This function is used to check that the transaction has been scheduled and is ready to
    /// execute. It only reads the state of the contract, and potentially reverts in order to
    /// protect against execution of unscheduled, early or cancelled transactions.
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
        bytes memory, /* signatures */
        address _msgSender
    )
        external
        override
    {
        Safe callingSafe = Safe(payable(msg.sender));

        if (_currentSafeState(callingSafe).timelockDelay == 0) {
            // We return immediately. This is important in order to allow a Safe which has the
            // guard set, but not configured, to complete the setup process.

            // It is also just a reasonable thing to do, since an unconfigured Safe must have a
            // delay of zero.
            return;
        }

        // Limit execution of transactions to owners of the Safe only.
        // This ensures that an attacker cannot simply collect valid signatures, but must also
        // control a private key. It is accepted as a trade-off that paymasters, relayers or UX
        // wrappers cannot execute transactions with the TimelockGuard enabled.
        if (!callingSafe.isOwner(_msgSender)) {
            revert TimelockGuard_NotOwner();
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
        ScheduledTransaction storage scheduledTx = _currentSafeState(callingSafe).scheduledTransactions[txHash];

        // Check if the transaction was cancelled
        if (scheduledTx.state == TransactionState.Cancelled) {
            revert TimelockGuard_TransactionAlreadyCancelled();
        }

        // Check if the transaction has already been executed
        // Note: this is of course enforced by the Safe itself, but we check it here for
        // completeness
        if (scheduledTx.state == TransactionState.Executed) {
            revert TimelockGuard_TransactionAlreadyExecuted();
        }

        // Check if the transaction has been scheduled
        if (scheduledTx.state == TransactionState.NotScheduled) {
            revert TimelockGuard_TransactionNotScheduled();
        }

        // Check if the timelock delay has passed
        if (scheduledTx.executionTime > block.timestamp) {
            revert TimelockGuard_TransactionNotReady();
        }

        // Reset the cancellation threshold
        _resetCancellationThreshold(callingSafe);

        // Set the transaction as executed.
        // Reverts in transaction as called from the Safe will be caught and ignored, with the Safe
        // bumping the nonce regardless. We accordingly set the transaction as executed and remove
        // it from the pending transactions anyway, as it can't be retried.
        scheduledTx.state = TransactionState.Executed;
        _currentSafeState(callingSafe).pendingTxHashes.remove(txHash);

        emit TransactionExecuted(callingSafe, txHash);
    }

    /// @notice Implementation of Guard interface. Called by the Safe after executing a transaction
    function checkAfterExecution(bytes32 _txHash, bool _success) external override { }

    ////////////////////////////////////////////////////////////////
    //              Internal State-Changing Functions             //
    ////////////////////////////////////////////////////////////////

    /// @notice Increase the cancellation threshold for a safe
    /// @dev This function must be called only once and only when calling cancel
    /// @param _safe The Safe address to increase the cancellation threshold for.
    function _increaseCancellationThreshold(Safe _safe) internal {
        SafeState storage safeState = _currentSafeState(_safe);

        if (safeState.cancellationThreshold < maxCancellationThreshold(_safe)) {
            uint256 oldThreshold = safeState.cancellationThreshold;
            safeState.cancellationThreshold++;
            emit CancellationThresholdUpdated(_safe, oldThreshold, safeState.cancellationThreshold);
        }
    }

    /// @notice Reset the cancellation threshold for a safe
    /// @dev This function must be called only once and only when calling checkAfterExecution
    /// @param _safe The Safe address to reset the cancellation threshold for.
    function _resetCancellationThreshold(Safe _safe) internal {
        SafeState storage safeState = _currentSafeState(_safe);
        uint256 oldThreshold = safeState.cancellationThreshold;
        safeState.cancellationThreshold = 1;
        emit CancellationThresholdUpdated(_safe, oldThreshold, 1);
    }

    ////////////////////////////////////////////////////////////////
    //              External State-Changing Functions             //
    ////////////////////////////////////////////////////////////////

    /// @notice Configure the contract as a timelock guard by setting the timelock delay
    /// @dev This function is only callable by the Safe itself. Requiring a call from the Safe
    ///      itself (rather than accepting signatures directly as in cancelTransaction()) is
    ///      important to ensure that maliciously gathered signatures will not be able to instantly
    ///      reconfigure the delay to zero. This function does not check that the guard is enabled
    ///      on the Safe, the recommended approach is to atomically enable the guard and configure
    ///      the delay in a single batched transaction.
    /// @param _timelockDelay The timelock delay in seconds (0 to clear configuration)
    function configureTimelockGuard(uint256 _timelockDelay) external {
        // Record the calling Safe
        Safe callingSafe = Safe(payable(msg.sender));

        // Check that the safe contract is version 1.4.1
        // There have been breaking changes at every minor version, and we can only support one
        // version.
        if (!SemverComp.eq(callingSafe.VERSION(), "1.4.1")) {
            revert TimelockGuard_InvalidVersion();
        }

        // Check that this guard is enabled on the calling Safe
        if (!_isGuardEnabled(callingSafe)) {
            revert TimelockGuard_GuardNotEnabled();
        }

        // Validate timelock delay - must not be zero or longer than 1 year
        if (_timelockDelay == 0 || _timelockDelay > 365 days) {
            revert TimelockGuard_InvalidTimelockDelay();
        }

        // Store the timelock delay for this safe
        _currentSafeState(callingSafe).timelockDelay = _timelockDelay;

        // Initialize (or reset) the cancellation threshold to 1.
        _resetCancellationThreshold(callingSafe);
        emit GuardConfigured(callingSafe, _timelockDelay);

        // Verify that any other extensions which are enabled on the Safe are configured correctly.
        _checkCombinedConfig(callingSafe);
    }

    /// @notice Clears the timelock guard configuration for a Safe.
    /// @dev Note: Clearing the configuration also cancels all pending transactions.
    ///      This function is intended for use when a Safe wants to permanently remove
    ///      the TimelockGuard configuration. Typical usage pattern:
    ///      1. Safe disables the guard via GuardManager.setGuard(address(0)).
    ///      2. Safe calls this clearTimelockGuard() function to remove stored configuration.
    ///      3. If Safe later re-enables the guard, it must call configureTimelockGuard() again.
    ///      Warning: Clearing the configuration allows all transactions previously scheduled to be
    ///      scheduled again, including cancelled transactions. It is strongly recommended to
    ///      manually increment the Safe's nonce when a scheduled transaction is cancelled.
    function clearTimelockGuard() external {
        Safe callingSafe = Safe(payable(msg.sender));

        // Check that this guard is NOT enabled on the calling Safe
        // This prevents clearing configuration while guard is still enabled
        if (_isGuardEnabled(callingSafe)) {
            revert TimelockGuard_GuardStillEnabled();
        }

        // Clear the configuration by bumping the nonce, all config and pending transactions will
        // be effectively wiped.
        _safeConfigNonces[callingSafe]++;
    }

    /// @notice Schedule a transaction for execution after the timelock delay.
    /// @dev This function validates signatures in the exact same way as the Safe's own
    ///      execTransaction function, meaning that the same signatures used to schedule a
    ///      transaction can be used to execute it later. This maintains compatibility with
    ///      existing signature generation tools. Owners can use any method to sign the a
    ///      transaction, including signing with a private key, calling the Safe's approveHash
    ///      function, or EIP1271 contract signatures.
    ///      The Safe doesn't increase its nonce when a transaction is cancelled in the Timelock.
    ///      This means that it is possible to add the very same transaction a second time to the
    ///      safe queue, but it won't be possible to schedule it again in the Timelock. It is
    ///      recommended that the safe nonce is manually incremented when a scheduled transaction
    ///      is cancelled.
    /// @param _safe The Safe address to schedule the transaction for.
    /// @param _nonce The nonce of the Safe for the transaction being scheduled.
    /// @param _params The parameters of the transaction being scheduled.
    /// @param _signatures The signatures of the owners who are scheduling the transaction.
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
        if (_currentSafeState(_safe).timelockDelay == 0) {
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
        // A transaction can only be scheduled once, regardless of whether it has been cancelled or
        // not, as otherwise an observer could reuse the same signatures to either:
        // 1. Reschedule a transaction after it has been cancelled
        // 2. Reschedule a pending transaction, which would update the execution time thus
        //    extending the delay for the original transaction.
        if (_currentSafeState(_safe).scheduledTransactions[txHash].executionTime != 0) {
            revert TimelockGuard_TransactionAlreadyScheduled();
        }

        // Verify signatures using the Safe's signature checking logic
        // This function call reverts if the signatures are invalid.
        _safe.checkSignatures(txHash, txHashData, _signatures);

        // Calculate the execution time
        uint256 executionTime = block.timestamp + _currentSafeState(_safe).timelockDelay;

        // Schedule the transaction and add it to the pending transactions set
        _currentSafeState(_safe).scheduledTransactions[txHash] = ScheduledTransaction({
            txHash: txHash,
            executionTime: executionTime,
            state: TransactionState.Pending,
            params: _params,
            nonce: _nonce
        });
        _currentSafeState(_safe).pendingTxHashes.add(txHash);

        emit TransactionScheduled(_safe, txHash, executionTime);
    }

    /// @notice Cancel a scheduled transaction if cancellation threshold is met
    /// @dev This function aims to mimic the approach which would be used by a quorum of signers to
    ///      cancel a partially signed transaction, by signing and executing an empty transaction
    ///      at the same nonce.
    ///      This enables us to define a standard "cancellation transaction" format using the Safe
    ///      address, nonce, and hash of the transaction being cancelled. This is necessary to
    ///      ensure that the cancellation transaction is unique and cannot be used to cancel
    ///      another transaction at the same nonce.
    ///
    ///      Signature verification uses the Safe's checkNSignatures function, so that the number
    ///      of signatures can be set by the Safe's current cancellation threshold. Another benefit
    ///      of checkNSignatures is that owners can use any method to sign the cancellation
    ///      transaction inputs, including signing with a private key, calling the Safe's
    ///      approveHash function, or EIP1271 contract signatures.
    ///
    ///      It is allowed to cancel transactions from a disabled TimelockGuard, as a way of
    ///      clearing the queue that wouldn't be as blunt as calling `clearTimelockConfiguration`.
    /// @param _safe The Safe address to cancel the transaction for.
    /// @param _txHash The hash of the transaction being cancelled.
    /// @param _nonce The nonce of the Safe for the transaction being cancelled.
    /// @param _signatures The signatures of the owners who are cancelling the transaction.
    function cancelTransaction(Safe _safe, bytes32 _txHash, uint256 _nonce, bytes memory _signatures) external {
        // The following checks ensure that the transaction has:
        // 1. Been scheduled
        // 2. Not already been cancelled
        // 3. Not already been executed
        // There is nothing inherently wrong with cancelling a transaction a transaction that
        // doesn't meet these criteria, but we revert in order to inform the user, and avoid
        // emitting a misleading TransactionCancelled event.
        if (_currentSafeState(_safe).scheduledTransactions[_txHash].state == TransactionState.Cancelled) {
            revert TimelockGuard_TransactionAlreadyCancelled();
        }
        if (_currentSafeState(_safe).scheduledTransactions[_txHash].state == TransactionState.Executed) {
            revert TimelockGuard_TransactionAlreadyExecuted();
        }
        if (_currentSafeState(_safe).scheduledTransactions[_txHash].state == TransactionState.NotScheduled) {
            revert TimelockGuard_TransactionNotScheduled();
        }

        // Generate the cancellation transaction data
        bytes memory txData = abi.encodeCall(this.signCancellation, (_txHash));
        // Any nonce can be used here, as long as all of the signatures are for the same
        // nonce. In practice we expect the nonce to be the same as the nonce of the transaction
        // being cancelled, as this most closely mimics the behaviour of the Safe UI's transaction
        // replacement feature. However we do not enforce that here, to allow for flexibility,
        // and to avoid the need for logic to retrieve the nonce from the transaction being
        // cancelled.
        bytes memory cancellationTxData = _safe.encodeTransactionData(
            address(this), 0, txData, Enum.Operation.Call, 0, 0, 0, address(0), address(0), _nonce
        );
        bytes32 cancellationTxHash = _safe.getTransactionHash(
            address(this), 0, txData, Enum.Operation.Call, 0, 0, 0, address(0), address(0), _nonce
        );

        // Verify signatures using the Safe's signature checking logic, with the cancellation
        // threshold as the number of signatures required.
        _safe.checkNSignatures(
            cancellationTxHash, cancellationTxData, _signatures, _currentSafeState(_safe).cancellationThreshold
        );

        // Set the transaction as cancelled, and remove it from the pending transactions set
        _currentSafeState(_safe).scheduledTransactions[_txHash].state = TransactionState.Cancelled;
        _currentSafeState(_safe).pendingTxHashes.remove(_txHash);

        // Increase the cancellation threshold
        _increaseCancellationThreshold(_safe);

        emit TransactionCancelled(_safe, _txHash);
    }

    ////////////////////////////////////////////////////////////////
    //                      Dummy Functions                       //
    ////////////////////////////////////////////////////////////////

    /// @notice Dummy function provided as a utility to facilitate signing cancelTransaction data
    ///         in the Safe UI.
    function signCancellation(bytes32) public {
        emit Message("This function is not meant to be called, did you mean to call cancelTransaction?");
    }
}
