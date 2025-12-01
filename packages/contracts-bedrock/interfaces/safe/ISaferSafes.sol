// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import {GnosisSafe} from "safe-contracts/GnosisSafe.sol";
import {Enum} from "safe-contracts/common/Enum.sol";
import {ISemver} from "interfaces/universal/ISemver.sol";

interface ISaferSafes is ISemver {
    struct ModuleConfig {
        uint256 livenessResponsePeriod;
        address fallbackOwner;
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

    enum TransactionState {
        PENDING,
        CANCELLED,
        EXECUTED
    }

    struct ScheduledTransaction {
        uint256 executionTime;
        TransactionState state;
        ExecTransactionParams params;
    }

    event CancellationThresholdUpdated(
        GnosisSafe indexed safe,
        uint256 oldThreshold,
        uint256 newThreshold
    );
    event ChallengeCancelled(address indexed safe);
    event ChallengeStarted(address indexed safe, uint256 challengeStartTime);
    event ChallengeSucceeded(address indexed safe, address fallbackOwner);
    event GuardConfigured(GnosisSafe indexed safe, uint256 timelockDelay);
    event Message(string message);
    event ModuleCleared(address indexed safe);
    event ModuleConfigured(
        address indexed safe,
        uint256 livenessResponsePeriod,
        address fallbackOwner
    );
    event TransactionCancelled(GnosisSafe indexed safe, bytes32 indexed txHash);
    event TransactionExecuted(GnosisSafe indexed safe, bytes32 txHash);
    event TransactionScheduled(
        GnosisSafe indexed safe,
        bytes32 indexed txHash,
        uint256 executionTime
    );

    error LivenessModule2_ChallengeAlreadyExists();
    error LivenessModule2_ChallengeDoesNotExist();
    error LivenessModule2_InvalidFallbackOwner();
    error LivenessModule2_InvalidResponsePeriod();
    error LivenessModule2_ModuleNotConfigured();
    error LivenessModule2_ModuleNotEnabled();
    error LivenessModule2_ModuleStillEnabled();
    error LivenessModule2_OwnershipTransferFailed();
    error LivenessModule2_ResponsePeriodActive();
    error LivenessModule2_ResponsePeriodEnded();
    error LivenessModule2_UnauthorizedCaller();
    error SaferSafes_InsufficientLivenessResponsePeriod();
    error SemverComp_InvalidSemverParts();
    error TimelockGuard_GuardNotConfigured();
    error TimelockGuard_GuardNotEnabled();
    error TimelockGuard_InvalidTimelockDelay();
    error TimelockGuard_InvalidVersion();
    error TimelockGuard_TransactionAlreadyCancelled();
    error TimelockGuard_TransactionAlreadyExecuted();
    error TimelockGuard_TransactionAlreadyScheduled();
    error TimelockGuard_TransactionNotReady();
    error TimelockGuard_TransactionNotScheduled();

    function cancelTransaction(
        GnosisSafe _safe,
        bytes32 _txHash,
        uint256 _nonce,
        bytes calldata _signatures
    ) external;

    function cancellationThreshold(
        GnosisSafe _safe
    ) external view returns (uint256);

    function challenge(address _safe) external;

    function challengeStartTime(address _safe) external view returns (uint256);

    function changeOwnershipToFallback(address _safe) external;

    function checkAfterExecution(bytes32 _txHash, bool _success) external;

    function checkTransaction(
        address _to,
        uint256 _value,
        bytes calldata _data,
        Enum.Operation _operation,
        uint256 _safeTxGas,
        uint256 _baseGas,
        uint256 _gasPrice,
        address _gasToken,
        address payable _refundReceiver,
        bytes calldata,
        address
    ) external view;

    function clearLivenessModule() external;

    function configureLivenessModule(ModuleConfig calldata _config) external;

    function configureTimelockGuard(uint256 _timelockDelay) external;

    function getChallengePeriodEnd(
        address _safe
    ) external view returns (uint256);

    function livenessSafeConfiguration(
        address _safe
    )
        external
        view
        returns (uint256 livenessResponsePeriod, address fallbackOwner);

    function maxCancellationThreshold(
        GnosisSafe _safe
    ) external view returns (uint256);

    function pendingTransactions(
        GnosisSafe _safe
    ) external view returns (ScheduledTransaction[] memory);

    function respond() external;

    function scheduleTransaction(
        GnosisSafe _safe,
        uint256 _nonce,
        ExecTransactionParams calldata _params,
        bytes calldata _signatures
    ) external;

    function scheduledTransaction(
        GnosisSafe _safe,
        bytes32 _txHash
    ) external view returns (ScheduledTransaction memory);

    function signCancellation(bytes32 _txHash) external;

    function timelockConfiguration(
        GnosisSafe _safe
    ) external view returns (uint256);

    function version() external pure returns (string memory);
}
