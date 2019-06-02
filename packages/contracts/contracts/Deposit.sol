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

    /*** Public ***/
    ERC20 public erc20;
    uint256 public totalDeposited;
    mapping (bytes32 => CheckpointStatus) public checkpoints;
    mapping (bytes32 => uint256) public exits; // the uint256 when it is "redeemableAfter"

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
            plasmaContract: address(this), plasmaBlockNumber: getLatestPlasmaBlockNumber() 
        });
        dt.Checkpoint memory checkpoint = dt.Checkpoint({
            stateUpdate: stateUpdate,
            subrange: depositRange
        });
        // Increment the total deposited
        totalDeposited += _amount;
        // Calculate the checkpointId and add it to our finalized checkpoints
        bytes32 checkpointId = getCheckpointId(checkpoint);
        CheckpointStatus memory status = CheckpointStatus(
            {challengeableUntil: block.number + CHALLENGE_PERIOD, outstandingChallenges: 0});
        checkpoints[checkpointId] = status;
        // Emit an event which informs us that the checkpoint was finalized
        emit CheckpointFinalized(checkpointId);
        emit LogCheckpoint(checkpoint);
    }

    function startExit(dt.Checkpoint memory _checkpoint) public {
        bytes32 checkpointId = getCheckpointId(_checkpoint);
        // Verify this exit may be started
        require(checkpoints[checkpointId].challengeableUntil != 0, "Checkpoint must be exist in order to begin exit");
        require(exits[checkpointId] == 0, "There must not exist an exit on this checkpoint already");
        require(_checkpoint.stateUpdate.stateObject.predicateAddress == msg.sender, "Checkpoint must be started by its predicate");
        exits[checkpointId] = block.number + EXIT_PERIOD;
        emit ExitStarted(checkpointId, exits[checkpointId]);
    }

    // TODO: startCheckpoint()
    // TODO: deprecateExit()
    // TODO: finalizeExit()

    /* 
    * Helpers
    */ 
    function getCheckpointId(dt.Checkpoint memory _checkpoint) private pure returns (bytes32) {
        return keccak256(abi.encode(_checkpoint.stateUpdate, _checkpoint.subrange));
    }

    function getLatestPlasmaBlockNumber() private returns (uint256) {
        return 0;
    }
}
