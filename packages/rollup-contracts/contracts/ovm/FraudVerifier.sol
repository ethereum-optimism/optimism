pragma solidity ^0.5.0;

import {StateTransitioner} from "./StateTransitioner.sol";

/**
 * @title FraudVerifier
 * @notice The contract which is able to delete invalid state roots.
 */
contract FraudVerifier {
    mapping(uint=>StateTransitioner) stateTransitioners;

    function initNewStateTransitioner(uint _preStateTransitionIndex) public returns(bool) {
        // TODO:
        // Create a new state transitioner for some specific pre-state transition index (assuming one hasn't already been made).
        // Note that the invalid state root that we are verifying is at _preStateTransitionIndex+1.
        // Add it to the stateTransitioners mapping! -- stateTransitioners[_preStateTransitionIndex] = newStateTransitioner;
        return true;
    }


    function verifyFraud(uint _transitionIndex) public returns(bool) {
        // TODO:
        // Simply verify that the state transitioner has completed, and that the state root
        // at _preStateTransitionIndex+1 is not equal to the state root which was committed for that index.
        return true;
    }
}
