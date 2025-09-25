// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.4;

library Enum {
    type Operation is uint8;
}

library TimelockGuard {
    struct ScheduledTransaction {
        uint256 executionTime;
        bool cancelled;
        bool executed;
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

    event CancellationThresholdUpdated(address indexed _safe, uint256 _oldThreshold, uint256 _newThreshold);
    event GuardConfigured(address indexed _safe, uint256 _timelockDelay);
    event TransactionCancelled(address indexed _safe, bytes32 indexed _txHash);
    event TransactionScheduled(address indexed _safe, bytes32 indexed _txHash, uint256 when);

    function cancelTransaction(address _safe, bytes32 _txHash, uint256 _nonce, bytes memory _signatures) external;
    function cancelTransactionOnSafe(address _safe, bytes32 _txHash) external;
    function cancellationThresholdForSafe(address _safe) external view returns (uint256);
    function pendingTransactionsForSafe(address) external pure returns (bytes32[] memory);
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
    function scheduledTransactionForSafe(
        address _safe,
        bytes32 _txHash
    )
        external
        view
        returns (TimelockGuard.ScheduledTransaction memory);
    function safeConfigs(address) external view returns (uint256 timelockDelay);
    function scheduleTransaction(
        address _safe,
        uint256 _nonce,
        ExecTransactionParams memory _params,
        bytes memory _signatures
    )
        external;
    function version() external view returns (string memory);
    function timelockConfigurationForSafe(address _safe) external view returns (uint256 timelockDelay);
    function maxCancellationThreshold(address _safe) external view returns (uint256);
    function pendingTransactionsForSafe(address _safe)
        external
        view
        returns (TimelockGuard.ScheduledTransaction[] memory);
}
