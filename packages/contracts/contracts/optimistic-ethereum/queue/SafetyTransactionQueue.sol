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
     * Events
     */
     
    event CalldataTxEnqueued();

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
     * Checks that that a dequeue is authenticated, and dequques if authenticated.
     */
    function dequeue()
        public
    {
        require(msg.sender == address(resolveCanonicalTransactionChain()), "Only the canonical transaction chain can dequeue safety queue transactions.");
        _dequeue();
    }

    /**
     * Makes a gas payment to 
     */
    function enqueueTx(
        bytes memory _tx
        // todo add gasLimit here (and eventually decode from _tx)
    )
        public
    {
        require(msg.sender == tx.origin, "Only EOAs can enqueue rollup transactions to the safety queue.");
        // todo burn gas proportional to limit here
        emit CalldataTxEnqueued();
        _enqueue(_tx);
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
