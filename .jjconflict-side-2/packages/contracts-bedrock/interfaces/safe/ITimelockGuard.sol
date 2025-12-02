// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

interface ISafe { }

interface IEnum {
    enum Operation {
        Call,
        DelegateCall
    }
}

interface ITimelockGuard {
    enum TransactionState {
        NotScheduled,
        Pending,
        Cancelled,
        Executed
    }

    struct ScheduledTransaction {
        bytes32 txHash;
        uint256 executionTime;
        TransactionState state;
        ExecTransactionParams params;
        uint256 nonce;
    }

    struct ExecTransactionParams {
        address to;
        uint256 value;
        bytes data;
        IEnum.Operation operation;
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
    error TimelockGuard_NotOwner();
    error TimelockGuard_TransactionAlreadyCancelled();
    error TimelockGuard_TransactionAlreadyScheduled();
    error TimelockGuard_TransactionNotScheduled();
    error TimelockGuard_TransactionNotReady();
    error TimelockGuard_TransactionAlreadyExecuted();
    error TimelockGuard_InvalidVersion();
    error SemverComp_InvalidSemverParts();

    event CancellationThresholdUpdated(ISafe indexed safe, uint256 oldThreshold, uint256 newThreshold);
    event GuardConfigured(ISafe indexed safe, uint256 timelockDelay);
    event TransactionCancelled(ISafe indexed safe, bytes32 indexed txHash);
    event TransactionScheduled(ISafe indexed safe, bytes32 indexed txHash, uint256 executionTime);
    event TransactionExecuted(ISafe indexed safe, bytes32 indexed txHash);
    event Message(string message);

    function cancelTransaction(ISafe _safe, bytes32 _txHash, uint256 _nonce, bytes memory _signatures) external;
    function signCancellation(bytes32) external;
    function cancellationThreshold(ISafe _safe) external view returns (uint256);
    function supportsInterface(bytes4 interfaceId) external view returns (bool);
    function checkTransaction(
        address _to,
        uint256 _value,
        bytes memory _data,
        IEnum.Operation _operation,
        uint256 _safeTxGas,
        uint256 _baseGas,
        uint256 _gasPrice,
        address _gasToken,
        address payable _refundReceiver,
        bytes memory,
        address _msgSender
    )
        external;
    function checkAfterExecution(bytes32 _txHash, bool _success) external;
    function configureTimelockGuard(uint256 _timelockDelay) external;
    function clearTimelockGuard() external;
    function scheduledTransaction(
        ISafe _safe,
        bytes32 _txHash
    )
        external
        view
        returns (ScheduledTransaction memory);
    function scheduleTransaction(
        ISafe _safe,
        uint256 _nonce,
        ExecTransactionParams memory _params,
        bytes memory _signatures
    )
        external;
    function timelockDelay(ISafe _safe) external view returns (uint256);
    function maxCancellationThreshold(ISafe _safe) external view returns (uint256);
    function pendingTransactions(ISafe _safe)
        external
        view
        returns (ScheduledTransaction[] memory);
}
