pragma solidity ^0.5.0;

/**
 * @title FraudVerifier
 * @notice The contract which is able to delete invalid state roots.
 */
contract FraudVerifier {
    function initNewFraudProof() public returns(bool) {
        // TODO:
        // Create a new stateManager & executionManager which both point at each other.
        return true;
    }

    function applyTransaction() public returns(bool) {
        // TODO:
        // First, call stateManager.initNewTransactionExecution()
        // (probably do the same with the ExecutionManager)
        // Then actually call `executeTransaction(tx)` with the tx in question!
        // Now check to make sure stateManager.existsInvalidStateAccess == false
        // BOOM. Successful tx execution, so now call stateManager.setIsTransitionSuccessfullyExecuted(true)
        // This will allow people to start updating the state root in the partial state manager.
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
