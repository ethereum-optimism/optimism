pragma solidity ^0.5.0;
pragma experimental ABIEncoderV2;

/* External Imports */
import "openzeppelin-solidity/contracts/math/Math.sol";

/* Internal Imports */
import { DataTypes as types } from "./DataTypes.sol";
import { TransactionPredicate } from "./TransactionPredicate.sol";
import { Deposit } from "./Deposit.sol";


/**
 * @title OwnershipTransactionPredicate
 * @notice TODO
 */
contract OwnershipTransactionPredicate is TransactionPredicate {
    /* Structs */
    struct OwnershipTransactionBody {
        bytes32 newStateObject;
        uint128 originBlock;
        uint128 maxBlock;
    }

    function startExitByOwner(types.Checkpoint memory _checkpoint) public {
        // Extract the owner from the state object data field
        address owner = getOwner(_checkpoint.stateUpdate);
        // Require that this is called by the owner
        require(msg.sender == owner, "Only owner may initiate the exit");
        // Forward the authenticated startExit to the deposit contract
        super.startExit(_checkpoint);
    }

    function finalizeExitByOwner(types.Checkpoint memory _exit, uint256 depositedRangeId) public {
        // Extract the owner from the state object data field
        address owner = getOwner(_exit.stateUpdate);
        // Require that this is called by the owner
        require(msg.sender == owner, "Only owner may finalize the exit");
        // handle the finalization from the parent class now thaat we've verified it's authenticated
        super.finalizeExit(_exit, depositedRangeId);
    }

    /* Functions which must be defined in each inheriting predicate */
    function verifyTransaction(
        types.StateUpdate memory _preState,
        types.Transaction memory _transaction,
        bytes memory _witness,
        types.StateUpdate memory _postState
    ) public returns (bool) {
        // check prestate.owner consented
        address owner = getOwner(_preState);
        require(checkSignature(_transaction, owner, _witness), 'Owner must have signed the transaction!');
        // decode parameters manually, nested struct is broken
        (bytes memory encodedNewState, uint128 originBlock, uint128 maxBlock) = abi.decode(_transaction.body, (bytes, uint128, uint128));
        // check the prestate came after or at the originating block
        require(_preState.plasmaBlockNumber <= originBlock, 'Transaction preState must come before or on the transaction body origin block.');
        // check the poststate came before or at the max block
        require(_postState.plasmaBlockNumber <= maxBlock, 'Transaction postState must come before or on the transaction body max block.');
        // check the state objects are the same
        bytes memory encodedPostState = abi.encode(_postState.stateObject.predicateAddress, _postState.stateObject.data);
        require(keccak256(encodedNewState) == keccak256(encodedPostState), 'postState must be the transaction.body.newState');

        return true;
    }

    function getOwner(types.StateUpdate memory _stateUpdate) public pure returns(address) {
        return abi.decode(_stateUpdate.stateObject.data, (address));
    }

    function checkSignature(types.Transaction memory _transaction, address _owner, bytes memory _signature) public pure returns(bool) {
        // TODO
        return true;
    }
}
