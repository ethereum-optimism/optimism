pragma solidity ^0.5.0;
pragma experimental ABIEncoderV2;

contract IPredicate {
    struct StateObject {
        bytes data;
    }

    function validStateTransition(StateObject memory _oldState, StateObject memory _newState, bytes memory _witness) public view returns (bool);
}
