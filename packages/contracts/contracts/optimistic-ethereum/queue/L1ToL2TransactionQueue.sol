pragma solidity ^0.5.0;
pragma experimental ABIEncoderV2;

/* Contract Imports */
import { CanonicalTransactionChain } from "../chain/CanonicalTransactionChain.sol";
import { RollupQueue } from "./RollupQueue.sol";

/* Library Imports */
import { ContractResolver } from "../utils/resolvers/ContractResolver.sol";

/**
 * @title L1ToL2TransactionQueue
 */
contract L1ToL2TransactionQueue is ContractResolver, RollupQueue {
    /*
     * Contract Variables
     */

    address public l1ToL2TransactionPasser;


    /*
     * Constructor
     */

    /**
     * @param _addressResolver Address of the AddressResolver contract.
     * @param _l1ToL2TransactionPasser Address of the L1-L2 transaction passer.
     */
    constructor(
        address _addressResolver,
        address _l1ToL2TransactionPasser
    )
        public
        ContractResolver(_addressResolver)
    {
        l1ToL2TransactionPasser = _l1ToL2TransactionPasser;
    }


    /*
     * Public Functions
     */

    /**
     * Checks whether a sender is allowed to enqueue.
     * @param _sender Sender address to check.
     * @return Whether or not the sender can enqueue.
     */
    function authenticateEnqueue(
        address _sender
    )
        public
        view
        returns (bool)
    {
        return _sender == l1ToL2TransactionPasser;
    }

    /**
     * Checks whether a sender is allowed to dequeue.
     * @param _sender Sender address to check.
     * @return Whether or not the sender can dequeue.
     */
    function authenticateDequeue(
        address _sender
    )
        public
        view
        returns (bool)
    {
        return _sender == address(resolveCanonicalTransactionChain());
    }

    /**
     * Checks whether this is a calldata transaction queue.
     * @return Whether or not this is a calldata tx queue.
     */
    function isCalldataTxQueue()
        public
        returns (bool)
    {
        return false;
    }


    /*
     * Contract Resolution
     */

    function resolveCanonicalTransactionChain()
        internal
        view
        returns (CanonicalTransactionChain)
    {
        return CanonicalTransactionChain(resolveContract("CanonicalTransactionChain"));
    }
}
