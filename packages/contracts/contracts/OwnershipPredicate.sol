pragma solidity ^0.5.0;
pragma experimental ABIEncoderV2;

/* Internal Imports */
import { DataTypes as dt } from "./DataTypes.sol";
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
        dt.Range range;
        bytes32 methodId;
        Parameters parameters;
    }

    struct Parameters {
        dt.StateObject newState;
        uint64 originBlock;
        uint64 maxBlock;
    }

    /*** Public ***/
    address public depositContractAddress;

    /**
     * @dev Constructs an ownership predicate contract with a specified deposit contract
     * @param _depositContractAddress TODO
     */
    constructor(address _depositContractAddress) public {
        depositContractAddress = _depositContractAddress;
    }

    function startExit(dt.Checkpoint memory _checkpoint) public {
        // Extract the owner from the state object data field
        address owner = abi.decode(_checkpoint.stateUpdate.stateObject.data, (address));
        // Require that this is called by the owner
        require(msg.sender == owner, "Only owner may initiate the exit");
        // Forward the authenticated startExit to the deposit contract
        Deposit depositContract = Deposit(_checkpoint.stateUpdate.depositAddress);
        depositContract.startExit(_checkpoint);
    }
}
