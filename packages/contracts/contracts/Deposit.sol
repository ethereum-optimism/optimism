pragma solidity ^0.5.0;
pragma experimental ABIEncoderV2;

/* External Imports */
import "openzeppelin-solidity/contracts/math/Math.sol";
import "openzeppelin-solidity/contracts/token/ERC20/ERC20.sol";

/* Internal Imports */
import {DataTypes as types} from "./DataTypes.sol";
import {CommitmentChain} from "./CommitmentChain.sol";

/**
 * @title Deposit
 * @notice TODO
 */
contract Deposit {

    /*** Structs ***/
    struct CheckpointStatus {
        uint256 challengeableUntil;
        uint256 outstandingChallenges;
    }

    struct Challenge {
        types.Checkpoint challengedCheckpoint;
        types.Checkpoint challengingCheckpoint;
    }

    /*** Events ***/
    event CheckpointStarted(
        types.Checkpoint checkpoint,
        uint256 challengeableUntil
    );

    event CheckpointFinalized(
        bytes32 checkpoint
    );

    event LogCheckpoint(
        types.Checkpoint checkpoint
    );

    event ExitStarted(
        bytes32 exit,
        uint256 redeemableAfter
    );

    event ExitFinalized(
        bytes32 exit
    );

    event CheckpointChallenged(
        Challenge challenge
    );

    /*** Public ***/
    ERC20 public erc20;
    CommitmentChain public commitmentChain;
    uint256 public totalDeposited;
    mapping (bytes32 => CheckpointStatus) public checkpoints;
    mapping (bytes32 => uint256) public exitRedeemableAfter;
    mapping (uint256 => types.Range) public depositedRanges;
    mapping (bytes32 => bool) public challenges;

    /*** Public Constants ***/
    // TODO - Set defaults
    uint256 public constant CHALLENGE_PERIOD = 10;
    uint256 public constant EXIT_PERIOD = 20;

    /**
     * @dev Constructs a deposit contract with a specified erc20 token
     * @param _erc20 TODO
     * @param _commitmentChain TODO
     */
    constructor(address _erc20, address _commitmentChain) public {
        erc20 = ERC20(_erc20);
        commitmentChain = CommitmentChain(_commitmentChain);
    }

    /**
     * @notice 
     * @param _amount TODO
     * @param _initialState  TODO
     */
    function deposit(uint256 _amount, types.StateObject memory _initialState) public {
        // Transfer tokens to the deposit contract
        erc20.transferFrom(msg.sender, address(this), _amount);
        // Create the Range, StateUpdate & Checkpoint
        types.Range memory depositRange = types.Range({start:totalDeposited, end: totalDeposited + _amount });
        types.StateUpdate memory stateUpdate = types.StateUpdate({
            range: depositRange, stateObject: _initialState, 
            depositAddress: address(this), plasmaBlockNumber: getLatestPlasmaBlockNumber() 
        });
        types.Checkpoint memory checkpoint = types.Checkpoint({
            stateUpdate: stateUpdate,
            subrange: depositRange
        });
        // Extend depositedRanges & increment totalDeposits
        extendDepositedRanges(_amount);
        // Calculate the checkpointId and add it checkpoints "pre-finalzed"
        bytes32 checkpointId = getCheckpointId(checkpoint);
        CheckpointStatus memory status = CheckpointStatus(
            {challengeableUntil: block.number - 1, outstandingChallenges: 0});
        checkpoints[checkpointId] = status;
        // Emit an event which informs us that the checkpoint was finalized
        emit CheckpointFinalized(checkpointId);
        emit LogCheckpoint(checkpoint);
    }

    function extendDepositedRanges(uint256 _amount) public {
        uint256 oldStart = depositedRanges[totalDeposited].start;
        uint256 oldEnd = depositedRanges[totalDeposited].end;
        // Set the newStart for the last range
        uint256 newStart;
        if (oldStart == 0 && oldEnd == 0) {
            // Case 1: We are creating a new range (this is the case when the rightmost range has been removed)
            newStart = totalDeposited;
        } else {
            // Case 2: We are extending the old range (deleting the old range and making a new one with the total length)
            delete depositedRanges[oldEnd];
            newStart = oldStart;
        }
        // Set the newEnd to the totalDeposited plus how much was deposited
        uint256 newEnd = totalDeposited + _amount;
        // Finally create and store the range!
        depositedRanges[newEnd] = types.Range(newStart, newEnd);
        // Increment total deposited now that we've extended our depositedRanges
        totalDeposited += _amount;
    }

    // This function is called when an exit is finalized to "burn" it--so that checkpoints and exits 
    // on the range cannot be made.  It is equivalent to the range having never been deposited.
    function removeDepositedRange(types.Range memory range, uint256 depositedRangeId) public {
        types.Range memory encompasingRange = depositedRanges[depositedRangeId];

        // Split the LEFT side

        // check if we we have a new deposited region to the left
        if (range.start != encompasingRange.start) {
            // new deposited range from the unexited old start until the newly exited start
            types.Range memory leftSplitRange = types.Range(encompasingRange.start, range.start);
            // Store the new deposited range
            depositedRanges[leftSplitRange.end] = leftSplitRange;
        }

        // Split the RIGHT side (there 3 possible splits)

        // 1) ##### -> $$$## -- check if we have leftovers to the right which are deposited
        if (range.end != encompasingRange.end) {
            // new deposited range from the newly exited end until the old unexited end
            types.Range memory rightSplitRange = types.Range(range.end, encompasingRange.end);
            // Store the new deposited range
            depositedRanges[rightSplitRange.end] = rightSplitRange;
            // We're done!
            return;
        }
        // 3) ##### -> $$$$$ -- without right-side leftovers & not the rightmost deposit, we can simply delete the value
        delete depositedRanges[encompasingRange.end];
    }

    function startCheckpoint(
        types.Checkpoint memory _checkpoint,
        bytes memory _inclusionProof,
        uint256 _depositedRangeId
    ) public {
        bytes32 checkpointId = getCheckpointId(_checkpoint);
        require(commitmentChain.verifyInclusion(_checkpoint.stateUpdate, _inclusionProof), "Checkpoint must be included");
        require(isSubrange(_checkpoint.subrange, _checkpoint.stateUpdate.range), "Checkpoint must be on a subrange of the StateUpdate");
        require(isSubrange(_checkpoint.subrange, depositedRanges[_depositedRangeId]), "Checkpoint subrange must be on a depositedRange");
        require(!checkpointExists(checkpointId), "Checkpoint must not already exist");
        // Create a new checkpoint
        checkpoints[checkpointId] = CheckpointStatus(block.number + CHALLENGE_PERIOD, 0);
        emit CheckpointStarted(_checkpoint, checkpoints[checkpointId].challengeableUntil);
    }

    function startExit(types.Checkpoint memory _checkpoint) public {
        bytes32 checkpointId = getCheckpointId(_checkpoint);
        // Verify this exit may be started
        require(checkpointExists(checkpointId), "Checkpoint must exist in order to begin exit");
        require(exitRedeemableAfter[checkpointId] == 0, "There must not exist an exit on this checkpoint already");
        require(_checkpoint.stateUpdate.stateObject.predicateAddress == msg.sender, "Exit must be started by its predicate");
        exitRedeemableAfter[checkpointId] = block.number + EXIT_PERIOD;
        emit ExitStarted(checkpointId, exitRedeemableAfter[checkpointId]);
    }

    function finalizeExit(types.Checkpoint memory _exit, uint256 depositedRangeId) public {
        bytes32 checkpointId = getCheckpointId(_exit);
        // Check that we are authorized to finalize this exit
        require(_exit.stateUpdate.stateObject.predicateAddress == msg.sender, "Exit must be finalized by its predicate");
        require(checkpointFinalized(checkpointId), "Checkpoint must be finalized to finalize an exit");
        require(block.number > exitRedeemableAfter[checkpointId], "Exit must be redeemable after this block");
        require(isSubrange(_exit.subrange, depositedRanges[depositedRangeId]), "Exit must be of an deposited range (one that hasn't been exited)");
        // Remove the deposited range
        removeDepositedRange(_exit.subrange, depositedRangeId);
        // Delete the exit & checkpoint entries
        delete checkpoints[checkpointId];
        delete exitRedeemableAfter[checkpointId];
        // Transfer tokens to the deposit contract
        uint256 amount = _exit.subrange.end - _exit.subrange.start;
        erc20.transfer(_exit.stateUpdate.stateObject.predicateAddress, amount);
        // Emit an event recording the exit's finalization
        emit ExitFinalized(checkpointId);
    }

    function deprecateExit(types.Checkpoint memory _exit) public {
        bytes32 checkpointId = getCheckpointId(_exit);
        require(_exit.stateUpdate.stateObject.predicateAddress == msg.sender, "Exit must be deprecated by its predicate");
        delete exitRedeemableAfter[checkpointId];
    }

    function deleteOutdatedExit(types.Checkpoint memory _exit, types.Checkpoint memory _newerCheckpoint) public {
        bytes32 outdatedExitId = getCheckpointId(_exit);
        bytes32 newerCheckpointId = getCheckpointId(_newerCheckpoint);
        require(intersects(_exit.subrange, _newerCheckpoint.subrange), "Exit and newer checkpoint must overlap");
        require(_exit.stateUpdate.plasmaBlockNumber < _newerCheckpoint.stateUpdate.plasmaBlockNumber, "Exit must be before a checkpoint");
        require(checkpointFinalized(newerCheckpointId), "Newer checkpoint must be finalized to delete an earlier exit");
        delete exitRedeemableAfter[outdatedExitId];
    }

    function challengeCheckpoint(Challenge memory _challenge) public {
        bytes32 challengedCheckpointId = getCheckpointId(_challenge.challengedCheckpoint);
        bytes32 challengingCheckpointId = getCheckpointId(_challenge.challengingCheckpoint);
        bytes32 challengeId = getChallengeId(_challenge);
        // Verify that the challenge may be added
        require(exitExists(challengingCheckpointId), "Challenging exit must exist");
        require(checkpointExists(challengedCheckpointId), "Challenged checkpoint must exist");
        require(intersects(_challenge.challengedCheckpoint.subrange, _challenge.challengingCheckpoint.subrange), "Challenge ranges must intersect");
        require(_challenge.challengingCheckpoint.stateUpdate.plasmaBlockNumber < _challenge.challengedCheckpoint.stateUpdate.plasmaBlockNumber, "Challenging cp after challenged cp");
        require(!challenges[challengeId], "Challenge must not already exist");
        require(checkpoints[challengedCheckpointId].challengeableUntil > block.number, "Checkpoint must still be challengable");
        // Add the challenge
        checkpoints[challengedCheckpointId].outstandingChallenges += 1;
        challenges[challengeId] = true;
    }

    function removeChallenge(Challenge memory _challenge) public {
        bytes32 challengedCheckpointId = getCheckpointId(_challenge.challengedCheckpoint);
        bytes32 challengingCheckpointId = getCheckpointId(_challenge.challengingCheckpoint);
        bytes32 challengeId = getChallengeId(_challenge);
        // Verify that the challenge may be added
        require(challenges[challengeId], "Challenge must exist");
        require(!exitExists(challengingCheckpointId), "Challenging exit must no longer exist");
        // Remove the challenge
        challenges[challengeId] = false;
        checkpoints[challengedCheckpointId].outstandingChallenges -= 1;
    }

    /* 
    * Helpers
    */ 
    function getCheckpointId(types.Checkpoint memory _checkpoint) private pure returns (bytes32) {
        return keccak256(abi.encode(_checkpoint.stateUpdate, _checkpoint.subrange));
    }

    function getChallengeId(Challenge memory _challenge) private pure returns (bytes32) {
        return keccak256(abi.encode(_challenge));
    }

    function getLatestPlasmaBlockNumber() private returns (uint256) {
        return 0;
    }

    function isSubrange(types.Range memory _subRange, types.Range memory _surroundingRange) public pure returns (bool) {
        return _subRange.start >= _surroundingRange.start && _subRange.end <= _surroundingRange.end;
    }

    function intersects(types.Range memory _range1, types.Range memory _range2) public pure returns (bool) {
        return Math.max(_range1.start, _range2.start) < Math.min(_range1.end, _range2.end);
    }

    function checkpointExists(bytes32 checkpointId) public view returns (bool) {
        return checkpoints[checkpointId].challengeableUntil != 0 || checkpoints[checkpointId].outstandingChallenges != 0;
    }

    function checkpointFinalized(bytes32 checkpointId) public view returns (bool) {
        // To be considered finalized, a checkpoint:
        // - MUST have no outstanding challenges
        // - MUST no longer be challengable
        return checkpoints[checkpointId].outstandingChallenges == 0 && checkpoints[checkpointId].challengeableUntil < block.number;
    }

    function exitExists(bytes32 checkpointId) public view returns (bool) {
        return exitRedeemableAfter[checkpointId] != 0;
    }
}
