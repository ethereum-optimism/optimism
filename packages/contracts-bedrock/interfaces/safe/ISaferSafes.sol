// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { ITimelockGuard, IEnum, ISafe } from "interfaces/safe/ITimelockGuard.sol";
import { ILivenessModule2 } from "interfaces/safe/ILivenessModule2.sol";
import { ISemver } from "interfaces/universal/ISemver.sol";

interface ISaferSafes is ISemver {
    function __constructor__() external;

    event CancellationThresholdUpdated(ISafe indexed safe, uint256 oldThreshold, uint256 newThreshold);
    event ChallengeCancelled(address indexed safe);
    event ChallengeStarted(address indexed safe, uint256 challengeStartTime);
    event ChallengeSucceeded(address indexed safe, address fallbackOwner);
    event GuardConfigured(ISafe indexed safe, uint256 timelockDelay);
    event Message(string message);
    event ModuleCleared(address indexed safe);
    event ModuleConfigured(address indexed safe, uint256 livenessResponsePeriod, address fallbackOwner);
    event TransactionCancelled(ISafe indexed safe, bytes32 indexed txHash);
    event TransactionExecuted(ISafe indexed safe, bytes32 indexed txHash);
    event TransactionScheduled(ISafe indexed safe, bytes32 indexed txHash, uint256 executionTime);

    error LivenessModule2_ChallengeAlreadyExists();
    error LivenessModule2_ChallengeDoesNotExist();
    error LivenessModule2_InvalidFallbackOwner();
    error LivenessModule2_InvalidResponsePeriod();
    error LivenessModule2_InvalidVersion();
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
    error TimelockGuard_GuardStillEnabled();
    error TimelockGuard_InvalidTimelockDelay();
    error TimelockGuard_InvalidVersion();
    error TimelockGuard_NotOwner();
    error TimelockGuard_TransactionAlreadyCancelled();
    error TimelockGuard_TransactionAlreadyExecuted();
    error TimelockGuard_TransactionAlreadyScheduled();
    error TimelockGuard_TransactionNotReady();
    error TimelockGuard_TransactionNotScheduled();

    function cancelTransaction(
        ISafe _safe,
        bytes32 _txHash,
        uint256 _nonce,
        bytes calldata _signatures
    )
        external;

    function cancellationThreshold(ISafe _safe) external view returns (uint256);

    function challenge(ISafe _safe) external;

    function challengeStartTime(ISafe) external view returns (uint256);

    function changeOwnershipToFallback(ISafe _safe) external;

    function checkAfterExecution(bytes32 _txHash, bool _success) external;

    function checkTransaction(
        address _to,
        uint256 _value,
        bytes calldata _data,
        IEnum.Operation _operation,
        uint256 _safeTxGas,
        uint256 _baseGas,
        uint256 _gasPrice,
        address _gasToken,
        address payable _refundReceiver,
        bytes calldata,
        address _msgSender
    )
        external;

    function clearLivenessModule() external;

    function clearTimelockGuard() external;

    function configureLivenessModule(ILivenessModule2.ModuleConfig calldata _config) external;

    function configureTimelockGuard(uint256 _timelockDelay) external;

    function getChallengePeriodEnd(ISafe _safe) external view returns (uint256);

    function livenessSafeConfiguration(ISafe _safe) external view returns (ILivenessModule2.ModuleConfig memory);

    function maxCancellationThreshold(ISafe _safe) external view returns (uint256);

    function pendingTransactions(ISafe _safe) external view returns (ITimelockGuard.ScheduledTransaction[] memory);

    function respond() external;

    function scheduleTransaction(
        ISafe _safe,
        uint256 _nonce,
        ITimelockGuard.ExecTransactionParams calldata _params,
        bytes calldata _signatures
    )
        external;

    function scheduledTransaction(
        ISafe _safe,
        bytes32 _txHash
    )
        external
        view
        returns (ITimelockGuard.ScheduledTransaction memory);

    function signCancellation(bytes32) external;

    function supportsInterface(bytes4 interfaceId) external view returns (bool);

    function timelockDelay(ISafe _safe) external view returns (uint256);

    function version() external pure returns (string memory);
}
