pragma solidity ^0.5.0;
pragma experimental ABIEncoderV2;

/* Contract Imports */
import { CanonicalTransactionChain } from "../chain/CanonicalTransactionChain.sol";
import { RollupQueue } from "./RollupQueue.sol";

/* Library Imports */
import { ContractResolver } from "../utils/resolvers/ContractResolver.sol";

/**
 * @title SafetyTransactionQueue
 */
contract SafetyTransactionQueue is ContractResolver, RollupQueue {
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
    {}


    /*
     * Public Functions
     */

    /**
     * Checks that a sender is authenticated to dequeue.
     * @param _sender Address to check.
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
