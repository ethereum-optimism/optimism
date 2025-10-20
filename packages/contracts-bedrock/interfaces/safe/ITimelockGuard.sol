// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.4;

library Enum {
    type Operation is uint8;
}

interface ITimelockGuard {
    enum TransactionState {
        NotScheduled,
        Pending,
        Cancelled,
        Executed
    }
    struct ScheduledTransaction {
        uint256 executionTime;
        TransactionState state;
        ExecTransactionParams params;
    }

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

    error TimelockGuard_GuardNotConfigured();
    error TimelockGuard_GuardNotEnabled();
    error TimelockGuard_GuardStillEnabled();
    error TimelockGuard_InvalidTimelockDelay();
    error TimelockGuard_TransactionAlreadyCancelled();
    error TimelockGuard_TransactionAlreadyScheduled();
    error TimelockGuard_TransactionNotScheduled();
    error TimelockGuard_TransactionNotReady();
    error TimelockGuard_TransactionAlreadyExecuted();
    error TimelockGuard_InvalidVersion();

    event CancellationThresholdUpdated(address indexed safe, uint256 oldThreshold, uint256 newThreshold);
    event GuardConfigured(address indexed safe, uint256 timelockDelay);
    event TransactionCancelled(address indexed safe, bytes32 indexed txHash);
    event TransactionScheduled(address indexed safe, bytes32 indexed txHash, uint256 executionTime);
    event TransactionExecuted(address indexed safe, bytes32 txHash);
    event Message(string message);
    event TransactionsNotCancelled(address indexed safe, uint256 uncancelledCount);

    function cancelTransaction(address _safe, bytes32 _txHash, uint256 _nonce, bytes memory _signatures) external;
    function signCancellation(bytes32 _txHash) external;
    function cancellationThreshold(address _safe) external view returns (uint256);
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
        bytes memory _signatures,
        address _msgSender
    )
        external;
    function checkAfterExecution(bytes32, bool) external;
    function configureTimelockGuard(uint256 _timelockDelay) external;
    function scheduledTransaction(
        address _safe,
        bytes32 _txHash
    )
        external
        view
        returns (ScheduledTransaction memory);
    function safeConfigs(address) external view returns (uint256 timelockDelay);
    function scheduleTransaction(
        address _safe,
        uint256 _nonce,
        ExecTransactionParams memory _params,
        bytes memory _signatures
    )
        external;
    function timelockConfiguration(address _safe) external view returns (uint256 timelockDelay);
    function maxCancellationThreshold(address _safe) external view returns (uint256);
    function pendingTransactions(address _safe)
        external
        view
        returns (ScheduledTransaction[] memory);
}
