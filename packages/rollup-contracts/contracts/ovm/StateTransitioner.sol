pragma solidity ^0.5.0;

import {FraudVerifier} from "./FraudVerifier.sol";
import {PartialStateManager} from "./PartialStateManager.sol";
import {ExecutionManager} from "../ExecutionManager.sol";

/**
 * @title StateTransitioner
 * @notice A contract which transitions a state from root one to another after a tx execution.
 */
contract StateTransitioner {
    enum TransitionPhases {
        PreExecution,
        PostExecution,
        Complete
    }
    TransitionPhases public currentTransitionPhase;

    uint public transitionIndex;
    bytes32 public stateRoot;
    bool public isTransactionSuccessfullyExecuted;

    FraudVerifier public fraudVerifier;
    PartialStateManager public stateManager;
    ExecutionManager executionManager;

    /************
    * Modifiers *
    ************/
    modifier preExecutionPhase {
        require(currentTransitionPhase == TransitionPhases.PreExecution, "Must be called during correct phase!");
        _;
    }

    modifier postExecutionPhase {
        require(currentTransitionPhase == TransitionPhases.PostExecution, "Must be called during correct phase!");
        _;
    }

    modifier completePhase {
        require(currentTransitionPhase == TransitionPhases.Complete, "Must be called during correct phase!");
        _;
    }

    constructor(
        uint _transitionIndex,
        bytes32 _preStateRoot,
        address _executionManagerAddress
    ) public {
        // The FraudVerifier always initializes a StateTransitioner in order to evaluate fraud.
        fraudVerifier = FraudVerifier(msg.sender);
        // Store the transitionIndex & state root which will be used during the proof.
        transitionIndex = _transitionIndex;
        stateRoot = _preStateRoot;
        // And of course set the ExecutionManager pointer.
        executionManager = ExecutionManager(_executionManagerAddress);
        // Finally we'll initialize a new state manager!
        stateManager = new PartialStateManager(address(this), address(executionManager));
        // And set our TransitionPhases to the PreExecution phase.
        currentTransitionPhase = TransitionPhases.PreExecution;
    }

    /****************************
    * Pre-Transaction Execution *
    ****************************/
    function proveContractInclusion(address _ovmContractAddress, address _codeContractAddress, uint _nonce) external preExecutionPhase {
        bytes32 codeHash;
        assembly {
            codeHash := extcodehash(_codeContractAddress)
        }
        // TODO: Verify an inclusion proof of the codeHash and nonce!

        stateManager.insertVerifiedContract(_ovmContractAddress, _codeContractAddress, _nonce);
    }

    function proveStorageSlotInclusion(address _ovmContractAddress, bytes32 _slot, bytes32 _value) external preExecutionPhase {
        require(stateManager.isVerifiedContract(_ovmContractAddress), "Contract must be verified before proving storage!");
        // TODO: Verify an inclusion proof of the storage slot!

        stateManager.insertVerifiedStorage(_ovmContractAddress, _slot, _value);
    }

    /************************
    * Transaction Execution *
    ************************/
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

    /****************************
    * Post-Transaction Execution *
    ****************************/

    function verifyFraud() public returns(bool) {
        // TODO:
        // Check to make sure that the stateManager root has had all elements in `updatedStorage`
        // and `updatedContracts` accounted for. All of these must update the root.
        // After that, simply compare the computed root to the posted root, if not equal, FRAUD!
        return true;
    }
}
