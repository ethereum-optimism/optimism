pragma solidity ^0.5.0;
pragma experimental ABIEncoderV2;

/* External Imports */
import "openzeppelin-solidity/contracts/math/Math.sol";
import "openzeppelin-solidity/contracts/token/ERC20/ERC20.sol";

/* Internal Imports */
import {DataTypes as dt} from "./DataTypes.sol";

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

    /*** Events ***/
    event CheckpointFinalized(
        bytes32 checkpoint
    );

    event LogCheckpoint(
        dt.Checkpoint checkpoint
    );

    event ExitStarted(
        bytes32 exit,
        uint256 redeemableAfter
    );

    event ExitFinalized(
        bytes32 exit
    );

    /*** Public ***/
    ERC20 public erc20;
    uint256 public totalDeposited;
    mapping (bytes32 => CheckpointStatus) public checkpoints;
    mapping (bytes32 => uint256) public exitRedeemableAfter; // the uint256 when it is "redeemableAfter"
    mapping (uint256 => dt.Range) public depositedRanges;

    /*** Public Constants ***/
    // TODO - Set defaults
    address public constant COMMITMENT_ADDRESS = 0x99EF1a332003a2c93a9f228fd7966CECDE344bcC;
    uint256 public constant CHALLENGE_PERIOD = 10;
    uint256 public constant EXIT_PERIOD = 20;

    /**
     * @dev Constructs a deposit contract with a specified erc20 token
     * @param _erc20 TODO
     */
    constructor(address _erc20) public {
        erc20 = ERC20(_erc20);
    }

    /**
     * @notice 
     * @param _amount TODO
     * @param _initialState  TODO
     */
    function deposit(uint256 _amount, dt.StateObject memory _initialState) public {
        // Transfer tokens to the deposit contract
        erc20.transferFrom(msg.sender, address(this), _amount);
        // Create the Range, StateUpdate & Checkpoint
        dt.Range memory depositRange = dt.Range({start:totalDeposited, end: totalDeposited + _amount });
        dt.StateUpdate memory stateUpdate = dt.StateUpdate({
            range: depositRange, stateObject: _initialState, 
            depositAddress: address(this), plasmaBlockNumber: getLatestPlasmaBlockNumber() 
        });
        dt.Checkpoint memory checkpoint = dt.Checkpoint({
            stateUpdate: stateUpdate,
            subrange: depositRange
        });
        // Extend depositedRanges & increment totalDeposits
        extendDepositedRanges(_amount);
        // Calculate the checkpointId and add it to our finalized checkpoints
        bytes32 checkpointId = getCheckpointId(checkpoint);
        CheckpointStatus memory status = CheckpointStatus(
            {challengeableUntil: block.number + CHALLENGE_PERIOD, outstandingChallenges: 0});
        checkpoints[checkpointId] = status;
        // Emit an event which informs us that the checkpoint was finalized
        emit CheckpointFinalized(checkpointId);
        emit LogCheckpoint(checkpoint);
    }

    function extendDepositedRanges(uint256 _amount) public {
        uint256 oldStart = depositedRanges[totalDeposited].start;
        uint256 oldEnd = depositedRanges[totalDeposited].end;
        dt.Range memory newRange;
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
        // Finally create the range!
        newRange = dt.Range(newStart, newEnd);
        // Store the result
        depositedRanges[newRange.end] = newRange;
        // Increment total deposited now that we've extended our depositedRanges
        totalDeposited += _amount;
    }

    function removeDepositedRange(dt.Range memory range, uint256 depositedRangeId) public {
        dt.Range memory encompasingRange = depositedRanges[depositedRangeId];

        // Split the LEFT side

        // check if we we have a new deposited region to the left
        if (range.start != encompasingRange.start) {
            // new deposited range from the unexited old start until the newly exited start
            dt.Range memory leftSplitRange = dt.Range(encompasingRange.start, range.start);
            // Store the new deposited range
            depositedRanges[leftSplitRange.end] = leftSplitRange;
        }

        // Split the RIGHT side (there 3 possible splits)

        // 1) ##### -> $$$## -- check if we have leftovers to the right which are deposited
        if (range.end != encompasingRange.end) {
            // new deposited range from the newly exited end until the old unexited end
            dt.Range memory rightSplitRange = dt.Range(range.end, encompasingRange.end);
            // Store the new deposited range
            depositedRanges[rightSplitRange.end] = rightSplitRange;
            // We're done!
            return;
        }
        // 3) ##### -> $$$$$ -- without right-side leftovers & not the rightmost deposit, we can simply delete the value
        delete depositedRanges[encompasingRange.end];
    }

    function startCheckpoint(
        dt.Checkpoint memory _checkpoint,
        bytes memory _inclusionProof,
        uint256 _depositedRangeId
    ) public {
        // TODO
    }

    function startExit(dt.Checkpoint memory _checkpoint) public {
        bytes32 checkpointId = getCheckpointId(_checkpoint);
        // Verify this exit may be started
        require(checkpoints[checkpointId].challengeableUntil != 0, "Checkpoint must exist in order to begin exit");
        require(exitRedeemableAfter[checkpointId] == 0, "There must not exist an exit on this checkpoint already");
        require(_checkpoint.stateUpdate.stateObject.predicateAddress == msg.sender, "Exit must be started by its predicate");
        exitRedeemableAfter[checkpointId] = block.number + EXIT_PERIOD;
        emit ExitStarted(checkpointId, exitRedeemableAfter[checkpointId]);
    }

    function finalizeExit(dt.Checkpoint memory _exit, uint256 depositedRangeId) public {
        bytes32 checkpointId = getCheckpointId(_exit);
        // Check that we are authorized to finalize this exit
        require(_exit.stateUpdate.stateObject.predicateAddress == msg.sender, "Exit must be finalized by its predicate");
        require(checkpoints[checkpointId].outstandingChallenges == 0, "Checkpoint must have no outstanding challenges");
        require(block.number > checkpoints[checkpointId].challengeableUntil, "Checkpoint must no longer be challengable");
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

    // TODO: deprecateExit()

    /* 
    * Helpers
    */ 
    function getCheckpointId(dt.Checkpoint memory _checkpoint) private pure returns (bytes32) {
        return keccak256(abi.encode(_checkpoint.stateUpdate, _checkpoint.subrange));
    }

    function getLatestPlasmaBlockNumber() private returns (uint256) {
        return 0;
    }

    function isSubrange(dt.Range memory _subRange, dt.Range memory _surroundingRange) public pure returns (bool) {
        return _subRange.start >= _surroundingRange.start && _subRange.end <= _surroundingRange.end;
    }
}
