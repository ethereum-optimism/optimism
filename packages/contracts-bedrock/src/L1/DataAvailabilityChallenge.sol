// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { OwnableUpgradeable } from "@openzeppelin/contracts-upgradeable/access/OwnableUpgradeable.sol";
import { ISemver } from "src/universal/ISemver.sol";
import { SafeCall } from "src/libraries/SafeCall.sol";

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
    address challenger;
    uint256 lockedBond;
    uint256 startBlock;
    uint256 resolvedBlock;
}

/// @title DataAvailabilityChallenge
/// @notice This contract enables data availability of a data commitment at a given block number to be challenged.
///         To challenge a commitment, the challenger must first post a bond (bondSize).
///         Challenging a commitment is only possible within a certain block interval (challengeWindow) after the
///         commitment was made.
///         If the challenge is not resolved within a certain block interval (resolveWindow), the challenge can be
///         expired.
///         If a challenge is expired, the challenger's bond is unlocked and the challenged commitment is added to the
///         chain of expired challenges.
contract DataAvailabilityChallenge is OwnableUpgradeable, ISemver {
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
    error InvalidInputData(bytes32 providedDataHash, bytes32 expectedHash);

    /// @notice Error for when the call to withdraw a bond failed.
    error WithdrawalFailed();

    /// @notice An event that is emitted when the status of a challenge changes.
    /// @param challengedHash The hash of the commitment that is being challenged.
    /// @param challengedBlockNumber The block number at which the commitment was made.
    /// @param status The new status of the challenge.
    event ChallengeStatusChanged(
        bytes32 indexed challengedHash, uint256 indexed challengedBlockNumber, ChallengeStatus status
    );

    /// @notice An event that is emitted when the bond size required to initiate a challenge changes.
    event RequiredBondSizeChanged(uint256 challengeWindow);

    /// @notice An event that is emitted when the percentage of the resolving cost to be refunded to the resolver
    /// changes.
    event ResolverRefundPercentageChanged(uint256 resolverRefundPercentage);

    /// @notice An event that is emitted when a user's bond balance changes.
    event BalanceChanged(address account, uint256 balance);

    /// @notice Semantic version.
    /// @custom:semver 0.0.0
    string public constant version = "0.0.0";

    /// @notice The fixed cost of resolving a challenge.
    uint256 public constant fixedResolutionCost = 44200;

    /// @notice The variable cost of resolving a callenge per byte of calldata.
    /// @dev upper limit; 16 gas per non-zero calldata byte, 4 gas variable execution cost per byte.
    uint256 public constant variableResolutionCost = 16 + 4;

    /// @notice The block interval during which a commitment can be challenged.
    uint256 public challengeWindow;

    /// @notice The block interval during which a challenge can be resolved.
    uint256 public resolveWindow;

    /// @notice The amount required to post a challenge.
    uint256 public bondSize;

    /// @notice The percentage of the resolving cost to be refunded to the resolver.
    uint256 public resolverRefundPercentage;

    /// @notice A mapping from addresses to their bond balance in the contract.
    mapping(address => uint256) public balances;

    /// @notice A mapping from challenged block numbers to challenged hashes to challenges.
    mapping(uint256 => mapping(bytes32 => Challenge)) public challenges;

    /// @notice constructs a new DataAvailabilityChallenge contract.
    constructor() OwnableUpgradeable() { }

    /// @notice Initializes the contract.
    /// @param _owner The owner of the contract.
    /// @param _challengeWindow The block interval during which a commitment can be challenged.
    /// @param _resolveWindow The block interval during which a challenge can be resolved.
    /// @param _bondSize The amount required to post a challenge.
    function initialize(
        address _owner,
        uint256 _challengeWindow,
        uint256 _resolveWindow,
        uint256 _bondSize,
        uint256 _resolverRefundPercentage
    )
        public
        initializer
    {
        __Ownable_init();
        challengeWindow = _challengeWindow;
        resolveWindow = _resolveWindow;
        setBondSize(_bondSize);
        setResolverRefundPercentage(_resolverRefundPercentage);
        _transferOwnership(_owner);
    }

    /// @notice Sets the bond size.
    /// @param _bondSize The amount required to post a challenge.
    function setBondSize(uint256 _bondSize) public onlyOwner {
        bondSize = _bondSize;
        emit RequiredBondSizeChanged(_bondSize);
    }

    /// @notice Sets the percentage of the resolving cost to be refunded to the resolver.
    /// @dev The function reverts if the provided percentage is above 100.
    /// @param _resolverRefundPercentage The percentage of the resolving cost to be refunded to the resolver.
    function setResolverRefundPercentage(uint256 _resolverRefundPercentage) public onlyOwner {
        if (_resolverRefundPercentage > 100) {
            revert InvalidResolverRefundPercentage(_resolverRefundPercentage);
        }
        resolverRefundPercentage = _resolverRefundPercentage;
    }

    /// @notice Post a bond as prerequisite for challenging a commitment.
    receive() external payable {
        deposit();
    }

    /// @notice Post a bond as prerequisite for challenging a commitment.
    function deposit() public payable {
        balances[msg.sender] += msg.value;
        emit BalanceChanged(msg.sender, balances[msg.sender]);
    }

    /// @notice Withdraw a user's unlocked bond.
    function withdraw() external {
        // get caller's balance
        uint256 balance = balances[msg.sender];

        // set caller's balance to 0
        balances[msg.sender] = 0;

        // send caller's balance to caller
        bool success = SafeCall.send(msg.sender, gasleft(), balance);
        if (!success) {
            revert WithdrawalFailed();
        }
    }

    /// @notice Checks if the current block is within the challenge window for a given challenged block number.
    /// @param challengedBlockNumber The block number at which the commitment was made.
    /// @return True if the current block is within the challenge window, false otherwise.
    function _isInChallengeWindow(uint256 challengedBlockNumber) internal view returns (bool) {
        return (block.number >= challengedBlockNumber && block.number <= challengedBlockNumber + challengeWindow);
    }

    /// @notice Checks if the current block is within the resolve window for a given challenge start block number.
    /// @param challengeStartBlockNumber The block number at which the challenge was initiated.
    /// @return True if the current block is within the resolve window, false otherwise.
    function _isInResolveWindow(uint256 challengeStartBlockNumber) internal view returns (bool) {
        return block.number <= challengeStartBlockNumber + resolveWindow;
    }

    /// @notice Returns the status of a challenge for a given challenged block number and challenged hash.
    /// @param challengedBlockNumber The block number at which the commitment was made.
    /// @param challengedHash The data commitment that is being challenged.
    /// @return The status of the challenge.
    function getChallengeStatus(
        uint256 challengedBlockNumber,
        bytes32 challengedHash
    )
        public
        view
        returns (ChallengeStatus)
    {
        Challenge memory _challenge = challenges[challengedBlockNumber][challengedHash];
        // if the address is 0, the challenge is uninitialized
        if (_challenge.challenger == address(0)) return ChallengeStatus.Uninitialized;

        // if the challenge has a resolved block, it is resolved
        if (_challenge.resolvedBlock != 0) return ChallengeStatus.Resolved;

        // if the challenge's start block is in the resolve window, it is active
        if (_isInResolveWindow(_challenge.startBlock)) return ChallengeStatus.Active;

        // if the challenge's start block is not in the resolve window, it is expired
        return ChallengeStatus.Expired;
    }

    /// @notice Challenge a data commitment at a given block number.
    /// @dev The block number parameter is necessary for the contract to verify the challenge window,
    ///      since the contract cannot access the block number of the commitment.
    ///      The function reverts if the caller does not have a bond or if the challenge already exists.
    /// @param challengedBlockNumber The block number at which the commitment was made.
    /// @param challengedHash The data commitment that is being challenged.
    function challenge(uint256 challengedBlockNumber, bytes32 challengedHash) external payable {
        // deposit value sent with the transaction as bond
        deposit();

        // require the caller to have a bond
        if (balances[msg.sender] < bondSize) {
            revert BondTooLow(balances[msg.sender], bondSize);
        }

        // require the challenge status to be uninitialized
        if (getChallengeStatus(challengedBlockNumber, challengedHash) != ChallengeStatus.Uninitialized) {
            revert ChallengeExists();
        }

        // require the current block to be in the challenge window
        if (!_isInChallengeWindow(challengedBlockNumber)) {
            revert ChallengeWindowNotOpen();
        }

        // reduce the caller's balance
        balances[msg.sender] -= bondSize;

        // store the challenger's address, bond size, and start block of the challenge
        challenges[challengedBlockNumber][challengedHash] =
            Challenge({ challenger: msg.sender, lockedBond: bondSize, startBlock: block.number, resolvedBlock: 0 });

        // emit an event to notify that the challenge status is now active
        emit ChallengeStatusChanged(challengedHash, challengedBlockNumber, ChallengeStatus.Active);
    }

    /// @notice Resolve a challenge by providing the pre-image data of the challenged commitment.
    /// @dev The provided pre-image data is hashed (keccak256) to verify that it matches the challenged commitment.
    ///      The function reverts if the challenge is not active or if the resolve window is not open.
    /// @param challengedBlockNumber The block number at which the commitment was made.
    /// @param preImage The pre-image data corresponding to the challenged commitment.
    function resolve(uint256 challengedBlockNumber, bytes32 challengedHash, bytes calldata preImage) external {
        // require the provided input data to match the commitment
        if (challengedHash != keccak256(preImage)) {
            revert InvalidInputData(keccak256(preImage), challengedHash);
        }

        // require the challenge to be active (started, not resolved, and resolve window still open)
        if (getChallengeStatus(challengedBlockNumber, challengedHash) != ChallengeStatus.Active) {
            revert ChallengeNotActive();
        }

        // store the block number at which the challenge was resolved
        Challenge storage activeChallenge = challenges[challengedBlockNumber][challengedHash];
        activeChallenge.resolvedBlock = block.number;

        // emit an event to notify that the challenge status is now resolved
        emit ChallengeStatusChanged(challengedHash, challengedBlockNumber, ChallengeStatus.Resolved);

        // distribute the bond among challenger, resolver and address(0)
        _distributeBond(activeChallenge, preImage.length, msg.sender);
    }

    /// @notice Distribute the bond of a resolved challenge among the resolver, challenger and address(0).
    ///         The challenger is refunded the bond amount exceeding the resolution cost.
    ///         The resolver is refunded a percentage of the resolution cost based on the `resolverRefundPercentage`
    /// state variable.
    ///         The remaining bond is burned by sending it to address(0).
    /// @dev The resolution cost is approximated based on a fixed cost and variable cost depending on the size of the
    /// pre-image.
    ///      The real resolution cost might vary, because calldata is priced differently for zero and non-zero bytes.
    ///      Computing the exact cost adds too much gas overhead to be worth the tradeoff.
    /// @param resolvedChallenge The resolved challenge in storage.
    /// @param preImageLength The size of the pre-image used to resolve the challenge.
    /// @param resolver The address of the resolver.
    function _distributeBond(Challenge storage resolvedChallenge, uint256 preImageLength, address resolver) internal {
        uint256 lockedBond = resolvedChallenge.lockedBond;
        address challenger = resolvedChallenge.challenger;

        // approximate the cost of resolving a challenge with the provided pre-image size
        uint256 resolutionCost = (fixedResolutionCost + preImageLength * variableResolutionCost) * tx.gasprice;

        // refund bond exceeding the resolution cost to the challenger
        if (lockedBond > resolutionCost) {
            balances[challenger] += lockedBond - resolutionCost;
            lockedBond = resolutionCost;
            emit BalanceChanged(challenger, balances[challenger]);
        }

        // refund a percentage of the resolution cost to the resolver (but not more than the locked bond)
        uint256 resolverRefund = resolutionCost * resolverRefundPercentage / 100;
        if (resolverRefund > lockedBond) {
            resolverRefund = lockedBond;
        }
        if (resolverRefund > 0) {
            balances[resolver] += resolverRefund;
            lockedBond -= resolverRefund;
            emit BalanceChanged(resolver, balances[resolver]);
        }

        // burn the remaining bond
        if (lockedBond > 0) {
            payable(address(0)).transfer(lockedBond);
        }
        resolvedChallenge.lockedBond = 0;
    }

    /// @notice Unlock the bond associated wth an expired challenge.
    /// @dev The function reverts if the challenge is not expired.
    ///      If the expiration is successful, the challenger's bond is unlocked.
    /// @param challengedBlockNumber The block number at which the commitment was made.
    /// @param challengedHash The data commitment that is being challenged.
    function unlockBond(uint256 challengedBlockNumber, bytes32 challengedHash) external {
        // require the challenge to be active (started, not resolved, and in the resolve window)
        if (getChallengeStatus(challengedBlockNumber, challengedHash) != ChallengeStatus.Expired) {
            revert ChallengeNotExpired();
        }

        // Unlock the bond associated with the challenge
        Challenge storage expiredChallenge = challenges[challengedBlockNumber][challengedHash];
        balances[expiredChallenge.challenger] += expiredChallenge.lockedBond;
        expiredChallenge.lockedBond = 0;

        // Emit balance update event
        emit BalanceChanged(expiredChallenge.challenger, balances[expiredChallenge.challenger]);
    }
}
