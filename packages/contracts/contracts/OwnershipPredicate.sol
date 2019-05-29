pragma solidity ^0.5.0;
pragma experimental ABIEncoderV2;

/**
 * @title OwnershipPredicate
 * @notice TODO
 */
contract OwnershipPredicate {

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

    struct OwnershipInput {
        StateObject newState;
        uint64 originBlock;
        uint64 maxBlock;
        bytes ecdsaSignature;
    }

    function verifyStateTransition(StateUpdate memory preState, OwnershipInput memory input, StateUpdate memory postState) public {
        // TODO: Actually verify everything
        return true
    }
}
