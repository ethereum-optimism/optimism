pragma solidity ^0.5.0;
pragma experimental ABIEncoderV2;

import "openzeppelin-solidity/contracts/math/Math.sol";

/**
 * @title Deposit
 * @notice TODO
 */
contract Deposit {
  /**
   * @notice TODO 
   */

  struct Range {
    uint256 start;
    uint256 end;
  }

  struct StateObject {
    address predicateAddress;
    bytes data;
  }

  struct StateUpdate {
    Range range;
    StateObject stateObject;
    address plasmaContract;
    uint256 plasmaBlockNumber;
  }

  struct Checkpoint {
    StateUpdate stateUpdate;
    Range checkpointedRange;
  }

  struct CheckpointStatus {
    uint256 challengeableUntil;
    uint256 outstandingChallenges;
  }

  struct Challenge {
    Checkpoint challengedCheckpoint;
    Checkpoint challengingCheckpoint;
  }

  /*** Public Constants ***/
  // TODO - Set defaults
  address public constant COMMITMENT_ADDRESS = 0x99EF1a332003a2c93a9f228fd7966CECDE344bcC;
  address public constant TOKEN_ADDRESS = 0xF6c105ED2f0f5Ffe66501a4EEdaD86E10df19054;
  uint256 public constant CHALLENGE_PERIOD = 10;
  uint256 public constant EXIT_PERIOD = 20;

  /*** Public ***/
  uint256 public totalDeposited;
  mapping (bytes32 => CheckpointStatus) public checkpoints;
  mapping (bytes32 => StateUpdate) public limboCheckpointOrigins;
  mapping (uint256 => Range) public exitableRanges;
  mapping (bytes32 => uint256) public exitsRedeemableAfter;
  mapping (bytes32 => bool) public challengeStatuses;

  event CheckpointStarted(
    bytes32 checkpoint,
    uint256 challengePeriodStart
  );

  event LimboCheckpointStarted(
    bytes32 checkpoint,
    uint256 challengePeriodStart
  );

  event CheckpointChallenged(
    bytes32 checkpoint,
    bytes32 challenge
  );

  event CheckpointFinalized(
    bytes32 checkpoint
  );

  event ExitStarted(
    bytes32 exit,
    uint256 exitPeriodStart
  );

  event ExitFinalized(
    bytes32 exit
  );

  /* 
  * TODO - Methods in this section need to be moved to external contracts
  */ 
  function verifyInclusion(StateUpdate memory _stateUpdate, bytes memory _inclusionProof) private returns (bool) {
    return true;
  }

  // TODO - Implement verification
  function verifySubRange(Range memory _range, Range memory _checkpointedRange) private returns (bool) {
    return true;
  }

  function executeTransaction(StateUpdate memory _stateUpdate, bytes memory _transaction) private returns (StateUpdate memory) {
    return _stateUpdate;
  }

  function getLatestPlasmaBlockNumber() private returns (uint256) {
    return 0;
  }

  /* 
  ** End Section **
  */

    /**
   * @notice 
   * @param _amount TODO
   * @param _initialState  TODO
   */
  function deposit(uint256 _amount, StateObject memory _initialState) public {
    // TODO - Requires?
    Range memory depositRange = Range({start:totalDeposited, end: totalDeposited + _amount });

    StateUpdate memory stateUpdate = StateUpdate({
      range: depositRange, stateObject: _initialState, 
      plasmaContract: address(this), plasmaBlockNumber: getLatestPlasmaBlockNumber() 
    });

    // TODO - Handle deposit?
    totalDeposited += _amount;

    bytes32 checkpointId = getCheckpointId(stateUpdate, stateUpdate.range);
    CheckpointStatus memory status = CheckpointStatus({challengeableUntil: block.number + CHALLENGE_PERIOD, outstandingChallenges: 0});
    checkpoints[checkpointId] = status;
    
    emit CheckpointFinalized(checkpointId);
  }

  /* 
  * Helpers
  */ 
  function getCheckpointId(StateUpdate memory _stateUpdate, Range memory _range) private returns (bytes32) {
    return keccak256(abi.encode(_stateUpdate, _range));
  }

  // TODO - setCheckpoint function with logic below
  // bytes32 checkpointId = getCheckpointId(_stateUpdate, _checkpointedRange);
  // CheckpointStatus memory status = CheckpointStatus({challengeableUntil: block.number + CHALLENGE_PERIOD, outstandingChallenges: 0});
  // checkpoints[checkpointId] = status;

  function startCheckpoint(StateUpdate memory _stateUpdate, bytes memory _inclusionProof, Range memory _checkpointedRange) public {
    require(verifyInclusion(_stateUpdate, _inclusionProof));
    require(verifySubRange(_stateUpdate.range, _checkpointedRange));

    bytes32 checkpointId = getCheckpointId(_stateUpdate, _checkpointedRange);
    CheckpointStatus memory status = CheckpointStatus({challengeableUntil: block.number + CHALLENGE_PERIOD, outstandingChallenges: 0});
    checkpoints[checkpointId] = status;

    emit CheckpointStarted(checkpointId, _stateUpdate.plasmaBlockNumber);
  }

  function startLimboCheckpoint(
    StateUpdate memory _stateUpdate, 
    bytes memory _inclusionProof, 
    bytes memory _transaction,
    Range memory _checkpointedRange
  ) private 
  {
    require(verifyInclusion(_stateUpdate, _inclusionProof));

    // MUST execute transaction against stateUpdate by calling the state update’s predicate.

    require(verifySubRange(_stateUpdate.range, _checkpointedRange));
    StateUpdate memory outputState = executeTransaction(_stateUpdate, _transaction);

    bytes32 checkpointId = getCheckpointId(_stateUpdate, _checkpointedRange);
    CheckpointStatus memory status = CheckpointStatus({challengeableUntil: block.number + CHALLENGE_PERIOD, outstandingChallenges: 0});
    checkpoints[checkpointId] = status;

    limboCheckpointOrigins[checkpointId] = _stateUpdate;

    // TODO - Doublecheck vars
    emit LimboCheckpointStarted(checkpointId, _stateUpdate.plasmaBlockNumber);
  }

  function challengeCheckpointOutdated(Checkpoint memory _olderCheckpoint, Checkpoint memory _newerCheckpoint) private {
    // Ensure checkpoint ranges intersect
    require(Math.max(_olderCheckpoint.checkpointedRange.start, _newerCheckpoint.checkpointedRange.start) < Math.min(_olderCheckpoint.checkpointedRange.end, _newerCheckpoint.checkpointedRange.end));

    // Ensure that the plasma blocknumber of the olderCheckpoint is less than that of newerCheckpoint.
    require(_olderCheckpoint.stateUpdate.plasmaBlockNumber < _newerCheckpoint.stateUpdate.plasmaBlockNumber);

    // Ensure that the newerCheckpoint has no challenges.
    bytes32 checkpointId = getCheckpointId(_newerCheckpoint.stateUpdate, _newerCheckpoint.stateUpdate.range);
    require(checkpoints[checkpointId].outstandingChallenges == 0);
    
    // Ensure that the newerCheckpoint is no longer challengeable.
    require(checkpoints[checkpointId].challengeableUntil < block.number + CHALLENGE_PERIOD);

    // Delete the entries in exits and checkpoints at the olderCheckpointId
    // delete exits
    delete checkpoints[checkpointId];
    delete exitsRedeemableAfter[checkpointId];
  }

  function challengeCheckpointInvalid(Challenge memory _challenge) private {
    // TODO
  }

  function challengeLimboCheckpointAlternateTransaction(
    bytes32 _limboCheckpoint,
    bytes memory _alternateTransaction,
    bytes memory _inclusionProof
  ) private 
  {
    // TODO
  }

  function removeChallengeCheckpointInvalidHistory(Challenge memory _challenge) private {
    // TODO 
  }

  function startExit(Challenge memory _checkpoint, bytes memory _witness) private {
    // TODO
  }

  function challengeExitDeprecated(Challenge memory _checkpoint, bytes memory _transaction, bytes memory _inclusionProof) private {
    // TODO
  }

  function finalizeExit(bytes32 _exit) private {
    // TODO
  }
  
}
