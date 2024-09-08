// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

/// @dev An enum representing the status of a DA challenge.
enum ChallengeStatus {
    Uninitialized,
    Active,
    Resolved,
    Expired
}

/// @dev An enum representing known commitment types.
enum CommitmentType {
    Keccak256
}

/// @dev A struct representing a single DA challenge.
/// @custom:field status The status of the challenge.
/// @custom:field challenger The address that initiated the challenge.
/// @custom:field startBlock The block number at which the challenge was initiated.
struct Challenge {
    address challenger;
    uint256 lockedBond;
    uint256 startBlock;
    uint256 resolvedBlock;
}

/// @title IDataAvailabilityChallenge
/// @notice Interface for the DataAvailabilityChallenge contract.
interface IDataAvailabilityChallenge {
    /// @notice Error for when the provided resolver refund percentage exceeds 100%.
    error InvalidResolverRefundPercentage(uint256 invalidResolverRefundPercentage);

    /// @notice Error for when the challenger's bond is too low.
    error BondTooLow(uint256 balance, uint256 required);

    /// @notice Error for when attempting to challenge a commitment that already has a challenge.
    error ChallengeExists();

    /// @notice Error for when attempting to resolve a challenge that is not active.
    error ChallengeNotActive();

    /// @notice Error for when attempting to unlock a bond from a challenge that is not expired.
    error ChallengeNotExpired();

    /// @notice Error for when attempting to challenge a commitment that is not in the challenge window.
    error ChallengeWindowNotOpen();

    /// @notice Error for when the provided input data doesn't match the commitment.
    error InvalidInputData(bytes providedDataCommitment, bytes expectedCommitment);

    /// @notice Error for when the call to withdraw a bond failed.
    error WithdrawalFailed();

    /// @notice Error for when a the type of a given commitment is unknown
    error UnknownCommitmentType(uint8 commitmentType);

    /// @notice Error for when the commitment length does not match the commitment type
    error InvalidCommitmentLength(uint8 commitmentType, uint256 expectedLength, uint256 actualLength);

    /// @notice An event that is emitted when the status of a challenge changes.
    /// @param challengedCommitment The commitment that is being challenged.
    /// @param challengedBlockNumber The block number at which the commitment was made.
    /// @param status The new status of the challenge.
    event ChallengeStatusChanged(
        uint256 indexed challengedBlockNumber, bytes challengedCommitment, ChallengeStatus status
    );

    /// @notice An event that is emitted when the bond size required to initiate a challenge changes.
    event RequiredBondSizeChanged(uint256 challengeWindow);

    /// @notice An event that is emitted when the percentage of the resolving cost to be refunded to the resolver
    /// changes.
    event ResolverRefundPercentageChanged(uint256 resolverRefundPercentage);

    /// @notice An event that is emitted when a user's bond balance changes.
    event BalanceChanged(address account, uint256 balance);

    function withdraw() external;
    function challenge(uint256 challengedBlockNumber, bytes calldata challengedCommitment) external payable;
    function resolve(
        uint256 challengedBlockNumber,
        bytes calldata challengedCommitment,
        bytes calldata resolveData
    )
        external;
    function unlockBond(uint256 challengedBlockNumber, bytes calldata challengedCommitment) external;
}
