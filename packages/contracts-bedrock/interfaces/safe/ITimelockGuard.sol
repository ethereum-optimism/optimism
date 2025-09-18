// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.4;

library Enum {
    type Operation is uint8;
}

library TimelockGuard {
    struct GuardConfig {
        uint256 timelockDelay;
    }

    struct ScheduledTransaction {
        uint256 executionTime;
        bool cancelled;
        bool executed;
    }
}

interface Interface {
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

    event CancellationThresholdUpdated(address indexed safe, uint256 oldThreshold, uint256 newThreshold);
    event GuardCleared(address indexed safe);
    event GuardConfigured(address indexed safe, uint256 timelockDelay);
    event TransactionCancelled(address indexed safe, bytes32 indexed txId);
    event TransactionScheduled(address indexed safe, bytes32 indexed txId, uint256 when);

    function blockingThreshold(address) external pure returns (uint256);
    function cancelTransaction(address _safe, bytes32 _txHash, uint256 _nonce, bytes memory _signatures) external;
    function cancellationThreshold(address _safe) external view returns (uint256);
    function checkAfterExecution(bytes32, bool) external;
    function checkPendingTransactions(address) external pure returns (bytes32[] memory);
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
    ) external;
    function clearTimelockGuard() external;
    function configureTimelockGuard(uint256 _timelockDelay) external;
    function getScheduledTransaction(address _safe, bytes32 _txHash)
        external
        view
        returns (TimelockGuard.ScheduledTransaction memory);
    function safeConfigs(address) external view returns (uint256 timelockDelay);
    function scheduleTransaction(
        address _safe,
        uint256 _nonce,
        ExecTransactionParams memory _params,
        bytes memory _signatures
    ) external;
    function version() external view returns (string memory);
    function viewTimelockGuardConfiguration(address _safe) external view returns (TimelockGuard.GuardConfig memory);
}
