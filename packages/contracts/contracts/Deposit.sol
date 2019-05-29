pragma solidity ^0.5.0;
pragma experimental ABIEncoderV2;

import "openzeppelin-solidity/contracts/math/Math.sol";
import "openzeppelin-solidity/contracts/token/ERC20/ERC20.sol";

/**
 * @title Deposit
 * @notice TODO
 */
contract Deposit {

    /*** Structs ***/
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

    struct CheckpointStatus {
        uint256 challengeableUntil;
        uint256 outstandingChallenges;
    }

    /*** Events ***/
    event CheckpointFinalized(
        bytes32 checkpoint
    );

    /*** Public ***/
    ERC20 public erc20;
    uint256 public totalDeposited;
    mapping (bytes32 => CheckpointStatus) public checkpoints;

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
    function deposit(uint256 _amount, StateObject memory _initialState) public {
        // Transfer tokens to the deposit contract
        erc20.transferFrom(msg.sender, address(this), _amount);
        // TODO - Requires?
        Range memory depositRange = Range({start:totalDeposited, end: totalDeposited + _amount });

        StateUpdate memory stateUpdate = StateUpdate({
            range: depositRange, stateObject: _initialState, 
            plasmaContract: address(this), plasmaBlockNumber: getLatestPlasmaBlockNumber() 
        });

        // TODO - Handle deposit?
        totalDeposited += _amount;

        bytes32 checkpointId = getCheckpointId(stateUpdate, stateUpdate.range);
        CheckpointStatus memory status = CheckpointStatus(
            {challengeableUntil: block.number + CHALLENGE_PERIOD, outstandingChallenges: 0});
        checkpoints[checkpointId] = status;
        
        emit CheckpointFinalized(checkpointId);
    }

    /* 
    * Helpers
    */ 
    function getCheckpointId(StateUpdate memory _stateUpdate, Range memory _range) private returns (bytes32) {
        return keccak256(abi.encode(_stateUpdate, _range));
    }

    function getLatestPlasmaBlockNumber() private returns (uint256) {
        return 0;
    }
}
