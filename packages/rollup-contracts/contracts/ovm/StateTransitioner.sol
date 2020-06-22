pragma solidity ^0.5.0;

import {FraudVerifier} from "./FraudVerifier.sol";
import {PartialStateManager} from "./PartialStateManager.sol";
import {ExecutionManager} from "../ExecutionManager.sol";

/**
 * @title StateTransitioner
 * @notice The contract which is able to delete invalid state roots.
 */
contract StateTransitioner {
    uint public transitionIndex;
    bytes32 public stateRoot;
    bool public isTransactionSuccessfullyExecuted;

    FraudVerifier public fraudVerifier;
    PartialStateManager stateManager;
    ExecutionManager executionManager;

    constructor(
        uint _transitionIndex,
        bytes32 _stateRoot,
        address _executionManagerAddress
    ) public {
        // The FraudVerifier always initializes a StateTransitioner in order to evaluate fraud.
        fraudVerifier = FraudVerifier(msg.sender);
        // Store the transitionIndex & state root which will be used during the proof.
        transitionIndex = _transitionIndex;
        stateRoot = _stateRoot;
        // And of course set the ExecutionManager pointer.
        executionManager = ExecutionManager(_executionManagerAddress);
        // Finally we'll initialize a new state manager!
        stateManager = new PartialStateManager(address(this), address(executionManager));
    }

    function applyTransaction() public returns(bool) {
        // TODO:
        // First, call stateManager.initNewTransactionExecution()
        // Then, call executionManager.setStateManager(stateManager.address)
        // Then actually call `exectuionManager.executeTransaction(tx)` with the tx in question!
        // Now check to make sure stateManager.existsInvalidStateAccess == false
        // BOOM. Successful tx execution, so now set isTransitionSuccessfullyExecuted = true
        // This will allow people to start updating the state root!
        return true;
    }


    function verifyFraud() public returns(bool) {
        // TODO:
        // Check to make sure that the stateManager root has had all elements in `updatedStorage`
        // and `updatedContracts` accounted for. All of these must update the root.
        // After that, simply compare the computed root to the posted root, if not equal, FRAUD!
        return true;
    }
}
