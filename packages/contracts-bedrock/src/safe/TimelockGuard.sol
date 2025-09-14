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
    }

    /// @notice Scheduled transaction
    struct ScheduledTransaction {
        uint256 executionTime;
        bool cancelled;
        bool executed;
    }

    /// @notice Mapping from Safe address to its guard configuration
    mapping(address => GuardConfig) public safeConfigs;

    /// @notice Mapping from Safe and tx id to scheduled transaction.
    mapping(Safe => mapping(bytes32 => ScheduledTransaction)) public scheduledTransactions;

    /// @notice Mapping from Safe and tx id to Safe and owner for rejected transactions.
    /// @dev Transactions are identifed by executing Safe and txHash, owners by Safe and address
    mapping(Safe => mapping(bytes32 => mapping(Safe => mapping(address => bool)))) public rejectedTransactions;

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

    /// @notice Emitted when a Safe configures the guard
    event GuardConfigured(address indexed safe, uint256 timelockDelay);

    /// @notice Emitted when a Safe clears the guard configuration
    event GuardCleared(address indexed safe);

    /// @notice Emitted when a transaction is scheduled for a Safe.
    /// @param safe The Safe whose transaction is scheduled.
    /// @param txId The identifier of the scheduled transaction (nonce-independent).
    /// @param when The timestamp when execution becomes valid.
    event TransactionScheduled(Safe indexed safe, bytes32 indexed txId, uint256 when);

    /// @notice Semantic version.
    /// @custom:semver 1.0.0
    string public constant version = "1.0.0";

    /// @notice Returns the timelock delay for a given Safe
    /// @dev MUST never revert
    /// @param _safe The Safe address to query
    /// @return The timelock delay in seconds
    function viewTimelockGuardConfiguration(address _safe) public view returns (uint256) {
        return safeConfigs[_safe].timelockDelay;
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
        // Validate timelock delay - must be non-zero and not longer than 1 year
        if (_timelockDelay == 0 || _timelockDelay > 365 days) {
            revert TimelockGuard_InvalidTimelockDelay();
        }

        // Check that this guard is enabled on the calling Safe
        if (!_isGuardEnabled(msg.sender)) {
            revert TimelockGuard_GuardNotEnabled();
        }

        // Store the configuration for this safe
        safeConfigs[msg.sender].timelockDelay = _timelockDelay;

        // Initialize cancellation threshold to 1
        safeCancellationThreshold[msg.sender] = 1;

        emit GuardConfigured(msg.sender, _timelockDelay);
    }

    /// @notice Remove the timelock guard configuration by a previously enabled Safe
    /// @dev MUST revert if the contract is not enabled as a guard for the Safe
    /// @dev MUST erase the existing timelock_delay data related to the calling Safe
    /// @dev MUST emit a GuardCleared event
    function clearTimelockGuard() external {
        // Check if the calling safe has configuration set
        if (safeConfigs[msg.sender].timelockDelay == 0) {
            revert TimelockGuard_GuardNotConfigured();
        }

        // Check that this guard is NOT enabled on the calling Safe
        if (_isGuardEnabled(msg.sender)) {
            revert TimelockGuard_GuardStillEnabled();
        }

        // Erase the configuration data for this safe
        delete safeConfigs[msg.sender];

        emit GuardCleared(msg.sender);
    }

    /// @notice Returns the cancellation threshold for a given safe
    /// @dev MUST NOT revert
    /// @dev MUST return 0 if the contract is not enabled as a guard for the safe
    /// @param _safe The Safe address to query
    /// @return The current cancellation threshold
    function cancellationThreshold(address _safe) public view returns (uint256) {
        // Return 0 if guard is not enabled
        if (!_isGuardEnabled(_safe)) {
            return 0;
        }

        return blockingThreshold(_safe);
    }

    /// @notice Returns the blocking threshold threshold for a given safe
    /// @dev MUST NOT revert
    /// @param _safe The Safe address to query
    /// @return The current blocking threshold
    function blockingThreshold(address _safe) public view returns (uint256) {
        return 0;
        // return min(quorum, total_owners - quorum + 1) for _safe;
    }

    /// @notice Internal helper to get the guard address from a Safe
    /// @param _safe The Safe address
    /// @return The current guard address
    function _isGuardEnabled(address _safe) internal view returns (bool) {
        // keccak256("guard_manager.guard.address") from GuardManager
        bytes32 guardSlot = 0x4a204f620c8c5ccdca3fd54d003badd85ba500436a431f0cbda4f558c93c34c8;
        Safe safe = Safe(payable(_safe));
        address guard = abi.decode(safe.getStorageAt(uint256(guardSlot), 1), (address));
        return guard == address(this);
    }

    /// @notice Schedule a transaction for execution after the timelock delay.
    /// @dev Minimal implementation: checks enabled+configured, uniqueness, cancellation, stores execution time and
    /// emits.
    /// @dev The txId is computed independent of Safe nonce using all exec params (with keccak(data)).
    function scheduleTransaction(Safe _safe, uint256 _nonce, ExecTransactionParams memory _params) external {
        // Check that this guard is enabled on the calling Safe
        if (!_isGuardEnabled(address(_safe))) {
            revert TimelockGuard_GuardNotEnabled();
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

        // Verify signatures using the Safe's signature checking logic
        // This function call reverts if the signatures are invalid.
        _safe.checkSignatures(txHash, txHashData, _params.signatures);

        // Check if the transaction exists
        // A transaction can only be scheduled once, regardless of whether it has been cancelled or not.
        if (scheduledTransactions[_safe][txHash].executionTime != 0) {
            revert TimelockGuard_TransactionAlreadyScheduled();
        }

        // Calculate the execution time
        uint256 executionTime = block.timestamp + safeConfigs[address(_safe)].timelockDelay;

        // Schedule the transaction
        scheduledTransactions[_safe][txHash] = ScheduledTransaction({ executionTime: executionTime, cancelled: false });

        emit TransactionScheduled(_safe, txHash, executionTime);
    }

    /// @notice Returns the list of all scheduled but not cancelled transactions for a given safe
    /// @dev MUST NOT revert - NOT IMPLEMENTED YET
    /// @return List of pending transaction hashes
    function checkPendingTransactions(address) external pure returns (bytes32[] memory) {
        return new bytes32[](0);
    }

    /// @notice Signal rejection of a scheduled transaction by a Safe owner
    /// @dev NOT IMPLEMENTED YET
    /// @param safePath An array of Safes starting by the safe that is scheduled to execute txHash, and going through child safes until the safe that msg.sender is owner of
    function rejectTransaction(address[] safePath, bytes32 txHash) external pure {
        // TODO: Implement
        // require(rejectingSafe.isOwner(msg.sender)); // Check if the caller is an owner in the rejectingSafe
        // require(rejectingSafe.isChild(executingSafe)); // Check if the rejectingSafe is a child safe, maybe several levels below
        // rejectedTransactions[executingSafe][txHash][rejectingSafe][msg.sender] = true;
    }

    /// @notice Signal rejection of a scheduled transaction using signatures
    /// @dev NOT IMPLEMENTED YET
    function rejectTransactionWithSignature(address[], bytes32, bytes memory) external pure {
        // TODO: Implement
    }

    /// @notice Cancel a scheduled transaction if cancellation threshold is met, needs to be called for each child safe, until we can call with executingSafe == rejectingSafe
    /// @dev NOT IMPLEMENTED YET
    /// @param safePath An array of Safes starting by the safe that is scheduled to execute txHash, and going through child safes until the safe that rejectingOwners are owners of
    function cancelTransaction(address[] safePath, bytes32 txHash, address[] rejectingOwners) external pure {
        // TODO: Implement
        // require not cancelled
        // require unique owners in rejectingOwners
        // require(rejectingSafe.isChild(executingSafe)); // Check if the rejectingSafe is a child safe, maybe several levels below
        // We would need as an argument a path from executingSafe to rejectingSafe if we want to required that rejectingSafe is a child safe of executingSafe
        // rejectingOwners = 0
        // for owner in rejectingOwners
        //   require rejectingSafe.isOwner(owner)
        //   if rejectedTransactions[executingSafe][txHash][rejectingSafe][owner]
        //     rejectingOwners++
        // if rejectingOwners >= cancellationThreshold(safe)
        //    rejectedTransactions[executingSafe][txHash][parentOfRejectingSafe][rejectingSafe] = true // parentOfRejectingSafe can be found as index -1 or -2 depending on length of safePath
    }

    /// @notice Called by the Safe before executing a transaction
    /// @dev Implementation of IGuard interface
    function checkTransaction(
        address,
        uint256 _value,
        bytes memory,
        Enum.Operation,
        uint256,
        uint256,
        uint256,
        address,
        address payable,
        bytes memory,
        address
    )
        external
        override
    {
        // TODO: Implement
    }

    /// @notice Called by the Safe after executing a transaction
    /// @dev Implementation of IGuard interface
    function checkAfterExecution(bytes32, bool) external override {
        // TODO: Implement
        // extract txHash
        // scheduledTransactions[_safe][txHash].executed = true
    }
}
