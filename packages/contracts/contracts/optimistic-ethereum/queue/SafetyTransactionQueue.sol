pragma solidity ^0.5.0;
pragma experimental ABIEncoderV2;

/* Contract Imports */
import { CanonicalTransactionChain } from "../chain/CanonicalTransactionChain.sol";
import { RollupQueue } from "./RollupQueue.sol";

/* Library Imports */
import { ContractResolver } from "../utils/resolvers/ContractResolver.sol";
import { GasConsumer } from "../utils/libraries/GasConsumer.sol";

/* Testing Imports */
import { console } from "@nomiclabs/buidler/console.sol";

/**
 * @title SafetyTransactionQueue
 */
contract SafetyTransactionQueue is ContractResolver, RollupQueue {
    /*
     * Events
     */

    event CalldataTxEnqueued();
    
    /*
     * Constants
     */

    uint constant public L2_GAS_DISCOUNT_DIVISOR = 10;

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

        uint gasToConsume = decodeL2TxGasLimit(_tx)/L2_GAS_DISCOUNT_DIVISOR;
        resolveGasConsumer().consumeGasInternalCall(gasToConsume);

        emit CalldataTxEnqueued();
        _enqueue(_tx);
    }

    /*
     * Internal Functions
     */

    function decodeL2TxGasLimit(
        bytes memory _l2Tx
    ) 
        internal
        returns(uint)
    {
        uint gasLimit;
        assembly {
            let a := _l2Tx
            gasLimit := mload(add(_l2Tx, 72)) // 40 (start of gasLimit in tx encoding) + 32 (abi prefix)
        }
        return gasLimit;
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

    function resolveGasConsumer()
        internal
        view
        returns (GasConsumer)
    {
        return GasConsumer(resolveContract("GasConsumer"));
    }
}
