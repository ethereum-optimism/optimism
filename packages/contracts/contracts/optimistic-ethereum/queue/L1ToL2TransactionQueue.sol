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
     * Constructor
     */

    /**
     * @param _addressResolver Address of the AddressResolver contract.
     */
    constructor(
        address _addressResolver
    )
        public
        ContractResolver(_addressResolver)
    {
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
        // TODO: figure out how we're going to authenticate this
        return true;
        // return _sender != tx.origin;
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
