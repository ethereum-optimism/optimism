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
        StateObject stateObject;
        Range range;
        uint256 plasmaBlockNumber;
        address depositAddress;
    }

    struct Checkpoint {
        StateUpdate stateUpdate;
        Range subrange;
    }

    struct Transaction {
        address depositAddress;
        bytes32 methodId;
        bytes parameters;
        Range range;
    }
}
