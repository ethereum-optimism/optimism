pragma solidity ^0.5.0;
pragma experimental ABIEncoderV2;

/* External Imports */
import "openzeppelin-solidity/contracts/math/Math.sol";

/* Internal Imports */
import { DataTypes as types } from "./DataTypes.sol";
import { Deposit } from "./Deposit.sol";

/**
 * @title TransactionPredicate
 * @notice TODO
 */
contract TransactionPredicate {

    /* Functions which must be defined in each inheriting predicate */
    function verifyTransaction(
        types.StateUpdate memory _preState,
        types.Transaction memory _transaction,
        bytes memory _witness,
        types.StateUpdate memory _postState
    ) public returns (bool) {
        return true;
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

    /* Standard functions used by these predicates */
    function deprecateExit(
        types.Checkpoint memory _deprecatedExit,
        types.Transaction memory _transaction,
        bytes memory _witness,
        types.StateUpdate memory _postState
    ) public {
        address preStateDepositAddress = _deprecatedExit.stateUpdate.depositAddress;
        address transactionDepositAddress = _transaction.depositAddress;
        address postStateDepositAddress = _postState.depositAddress;
        require(preStateDepositAddress == transactionDepositAddress, "Transactions can only act on SUs with the same deposit contract");
        require(transactionDepositAddress == postStateDepositAddress, "Transactions can only produce SUs with the same deposit contract");
        require(intersects(_deprecatedExit.subrange, _postState.range), "Transactions can only deprecate an exit intersecting the postState subrange");
        require(verifyTransaction(_deprecatedExit.stateUpdate, _transaction, _witness, _postState), "Predicate must be able to verify the transaction to deprecate");
        Deposit depositContract = Deposit(_deprecatedExit.stateUpdate.depositAddress);
        depositContract.deprecateExit(_deprecatedExit);
    }

    // Note: in theory we might want custom functionality here, but usually for predicates inheriting this model they will be the same.
    function finalizeExit(types.Checkpoint memory _exit, uint256 depositedRangeId) public {
        Deposit depositContract = Deposit(_exit.stateUpdate.depositAddress);
        depositContract.finalizeExit(_exit, depositedRangeId);
    }

    /* Helpers */
    function intersects(types.Range memory _range1, types.Range memory _range2) public pure returns (bool) {
        return Math.max(_range1.start, _range2.start) < Math.min(_range1.end, _range2.end);
    }
}
