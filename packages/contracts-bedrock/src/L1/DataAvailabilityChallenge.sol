// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { OwnableUpgradeable } from "@openzeppelin/contracts-upgradeable/access/OwnableUpgradeable.sol";
import { ISemver } from "src/universal/ISemver.sol";

/// @dev An enum representing the status of a DA challenge.
enum ChallengeStatus {
    Uninitialized,
    Active,
    Resolved,
    Expired
}

/// @dev A struct representing a single DA challenge.
/// @custom:field status The status of the challenge.
/// @custom:field challenger The address that initiated the challenge.
/// @custom:field startBlock The block number at which the challenge was initiated.
struct Challenge {
    ChallengeStatus status;
    address challenger;
    uint256 startBlock;
}

/// @title DataAvailabilityChallenge
/// @notice This contract enables data availability of a data commitment at a given block number to be challenged.
///         To challenge a commitment, the challenger must first post a bond (bondSize).
///         Challenging a commitment is only possible within a certain block interval (challengeWindow) after the commitment was made.
///         If the challenge is not resolved within a certain block interval (resolveWindow), the challenge can be expired.
///         If a challenge is expired, the challenger's bond is unlocked and the challenged commitment is added to the chain of expired challenges.
contract DataAvailabilityChallenge is OwnableUpgradeable, ISemver {
    /// @notice Error for when the challenger's bond is too low.
    error BondTooLow(uint256 balance, uint256 required);

    /// @notice Error for when attempting to challenge a commitment that already has a challenge.
    error ChallengeExists();

    /// @notice Error for when attempting to resolve a challenge that is not active.
    error ChallengeNotActive();

    /// @notice Error for when attempting to challenge a commitment that is not in the challenge window.
    error ChallengeWindowNotOpen();

    /// @notice Error for when attempting to resolve a challenge that is not in the resolve window.
    error ResolveWindowNotOpen();

    /// @notice Error for when attempting to expire a challenge that is still in the resolve window.
    error ResolveWindowNotClosed();

    /// @notice An event that is emitted when the status of a challenge changes.
    /// @param challengedHash The hash of the commitment that is being challenged.
    /// @param challengedBlockNumber The block number at which the commitment was made.
    /// @param status The new status of the challenge.
    event ChallengeStatusChanged(
        bytes32 indexed challengedHash, uint256 indexed challengedBlockNumber, ChallengeStatus status
    );

    /// @notice An event that is emitted when the head of the chain of expired challenges is updated.
    /// @param expiredChallengesHead The new head of the chain of expired challenges.
    event ExpiredChallengesHeadUpdated(bytes32 expiredChallengesHead);

    /// @notice Semantic version.
    /// @custom:semver 0.0.0
    string public constant version = "0.0.0";

    /// @notice The block interval during which a commitment can be challenged.
    uint256 public challengeWindow;

    /// @notice The block interval during which a challenge can be resolved.
    uint256 public resolveWindow;

    /// @notice The amount required to post a challenge.
    uint256 public bondSize;

    /// @notice A mapping from addresses to their bond balance in the contract.
    mapping(address => uint256) public balances;

    /// @notice A mapping from challenged block numbers to challenged hashes to challenges.
    mapping(uint256 => mapping(bytes32 => Challenge)) public challenges;

    /// @notice The head of the chain of expired challenges.
    bytes32 public expiredChallengesHead;

    /// @notice constructs a new DataAvailabilityChallenge contract.
    constructor() OwnableUpgradeable() {}

    /// @notice Sets the challenge window.
    /// @param _challengeWindow The block interval during which a commitment can be challenged.
    function setChallengeWindow(uint256 _challengeWindow) public onlyOwner {
        challengeWindow = _challengeWindow;
    }

    /// @notice Sets the resolve window.
    /// @param _resolveWindow The block interval during which a challenge can be resolved.
    function setResolveWindow(uint256 _resolveWindow) public onlyOwner {
        resolveWindow = _resolveWindow;
    }

    /// @notice Sets the bond size.
    /// @param _bondSize The amount required to post a challenge.
    function setBondSize(uint256 _bondSize) public onlyOwner {
        bondSize = _bondSize;
    }

    /// @notice Initializes the contract.
    /// @param _owner The owner of the contract.
    /// @param _challengeWindow The block interval during which a commitment can be challenged.
    /// @param _resolveWindow The block interval during which a challenge can be resolved.
    /// @param _bondSize The amount required to post a challenge.
    function initialize(address _owner, uint256 _challengeWindow, uint256 _resolveWindow, uint256 _bondSize) public initializer {
        __Ownable_init();
        setChallengeWindow(_challengeWindow);
        setResolveWindow(_resolveWindow);
        setBondSize(_bondSize);
        _transferOwnership(_owner);
    }

    /// @notice Post a bond as prerequisite for challenging a commitment.
    function deposit() external payable {
        balances[msg.sender] += msg.value;
    }

    /// @notice Withdraw a user's unlocked bond.
    function withdraw() external {
        // get caller's balance
        uint256 balance = balances[msg.sender];

        // set caller's balance to 0
        balances[msg.sender] = 0;

        // send caller's balance to caller
        payable(msg.sender).transfer(balance);
    }

    /// @notice Checks if the current block is within the challenge window for a given challenged block number.
    /// @param challengedBlockNumber The block number at which the commitment was made.
    /// @return True if the current block is within the challenge window, false otherwise.
    function _isInChallengeWindow(uint256 challengedBlockNumber) internal view returns (bool) {
        return (block.number > challengedBlockNumber && block.number <= challengedBlockNumber + challengeWindow);
    }

    /// @notice Checks if the current block is within the resolve window for a given challenge start block number.
    /// @param challengeStartBlockNumber The block number at which the challenge was initiated.
    /// @return True if the current block is within the resolve window, false otherwise.
    function _isInResolveWindow(uint256 challengeStartBlockNumber) internal view returns (bool) {
        return block.number <= challengeStartBlockNumber + resolveWindow;
    }

    /// @notice Challenge a data commitment at a given block number.
    /// @dev The block number parameter is necessary for the contract to verify the challenge window,
    ///      since the contract cannot access the block number of the commitment.
    ///      The function reverts if the caller does not have a bond or if the challenge already exists.
    /// @param challengedBlockNumber The block number at which the commitment was made.
    /// @param challengedHash The data commitment that is being challenged.
    function challenge(uint256 challengedBlockNumber, bytes32 challengedHash) external {
        // require the caller to have a bond
        if (balances[msg.sender] < bondSize) {
            revert BondTooLow(balances[msg.sender], bondSize);
        }

        // reduce the caller's bond
        balances[msg.sender] -= bondSize;

        // require the challenge status to be uninitialized
        Challenge storage existingChallenge = challenges[challengedBlockNumber][challengedHash];
        if (existingChallenge.status != ChallengeStatus.Uninitialized) {
            revert ChallengeExists();
        }

        // require the current block to be in the challenge window
        if (!_isInChallengeWindow(challengedBlockNumber)) {
            revert ChallengeWindowNotOpen();
        }

        // set the status of this challenge to active, store the current block number and address of the challenger
        challenges[challengedBlockNumber][challengedHash] =
            Challenge({status: ChallengeStatus.Active, challenger: msg.sender, startBlock: block.number});

        // emit an event to notify that the challenge status is now active
        emit ChallengeStatusChanged(challengedHash, challengedBlockNumber, ChallengeStatus.Active);
    }

    /// @notice Resolve a challenge by providing the pre-image data of the challenged commitment.
    /// @dev The provided pre-image data is hashed (keccak256) to verify that it matches the challenged commitment.
    ///      The function reverts if the challenge is not active or if the resolve window is not open.
    /// @param challengedBlockNumber The block number at which the commitment was made.
    /// @param preImage The pre-image data corresponding to the challenged commitment.
    function resolve(uint256 challengedBlockNumber, bytes calldata preImage) external {
        // hash the preImage
        bytes32 challengedHash = keccak256(preImage);

        // require the challenge to be active
        Challenge storage activeChallenge = challenges[challengedBlockNumber][challengedHash];
        if (activeChallenge.status != ChallengeStatus.Active) {
            revert ChallengeNotActive();
        }

        // require the resolve window to be open
        if (!_isInResolveWindow(activeChallenge.startBlock)) {
            revert ResolveWindowNotOpen();
        }

        // set the challenge status to resolved
        activeChallenge.status = ChallengeStatus.Resolved;

        // emit an event to notify that the challenge status is now resolved
        emit ChallengeStatusChanged(challengedHash, challengedBlockNumber, ChallengeStatus.Resolved);
    }

    /// @notice Expire a challenge that has not been resolved within the resolve window.
    /// @dev The function reverts if the challenge is not active or if the resolve window is still open.
    ///      If the expiration is successful, the challenger's bond is unlocked
    ///      and the challenged commitment is added to the chain of expired challenges.
    /// @param challengedBlockNumber The block number at which the commitment was made.
    /// @param challengedHash The data commitment that is being challenged.
    function expire(uint256 challengedBlockNumber, bytes32 challengedHash) external {
        // require the challenge to be active
        Challenge storage activeChallenge = challenges[challengedBlockNumber][challengedHash];
        if (activeChallenge.status != ChallengeStatus.Active) {
            revert ChallengeNotActive();
        }

        // require the resolve window to be closed
        if (_isInResolveWindow(activeChallenge.startBlock)) {
            revert ResolveWindowNotClosed();
        }

        // set the status to expired
        activeChallenge.status = ChallengeStatus.Expired;

        // restore the challenger's bond
        balances[activeChallenge.challenger] += bondSize;

        // update the head of the chain of expired challenges
        expiredChallengesHead = keccak256(abi.encode(expiredChallengesHead, challengedHash));

        // emit an event to notify that the challenge status is now expired
        emit ChallengeStatusChanged(challengedHash, challengedBlockNumber, ChallengeStatus.Expired);

        // emit an event to notify that the head of the chain of expired challenges has been updated
        emit ExpiredChallengesHeadUpdated(expiredChallengesHead);
    }
}
