pragma solidity ^0.5.0;
pragma experimental ABIEncoderV2;

/* Internal Imports */
import { DataTypes as types } from "./DataTypes.sol";
import { Deposit } from "./Deposit.sol";

/**
 * @title OwnershipPredicate
 * @notice TODO
 */
contract OwnershipPredicate {

    /*** Structs ***/
    struct OwnershipData {
        address owner;
    }

    struct OwnershipTransaction {
        UnsignedOwnershipTransaction unsignedTransaction;
        bytes ecdsaSignature;
    }

    struct UnsignedOwnershipTransaction {
        address depositAddress;
        types.Range range;
        Body body;
    }

    struct Body {
        types.StateObject newState;
        uint64 originBlock;
        uint64 maxBlock;
    }

    function startExit(types.Checkpoint memory _checkpoint) public {
        // Extract the owner from the state object data field
        address owner = abi.decode(_checkpoint.stateUpdate.stateObject.data, (address));
        // Require that this is called by the owner
        require(msg.sender == owner, "Only owner may initiate the exit");
        // Forward the authenticated startExit to the deposit contract
        Deposit depositContract = Deposit(_checkpoint.stateUpdate.depositAddress);
        depositContract.startExit(_checkpoint);
    }

    function deprecateExit(types.Checkpoint memory _exit) public {
        Deposit depositContract = Deposit(_exit.stateUpdate.depositAddress);
        depositContract.deprecateExit(_exit);
    }

    function finalizeExit(types.Checkpoint memory _exit, uint256 depositedRangeId) public {
        Deposit depositContract = Deposit(_exit.stateUpdate.depositAddress);
        depositContract.finalizeExit(_exit, depositedRangeId);
    }
}
