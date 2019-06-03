pragma solidity ^0.5.0;
pragma experimental ABIEncoderV2;

/**
 * @title DataTypes
 * @notice TODO
 */
contract DataTypes {

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
        address depositAddress;
        uint256 plasmaBlockNumber;
    }

    struct Checkpoint {
        StateUpdate stateUpdate;
        Range subrange;
    }
}
