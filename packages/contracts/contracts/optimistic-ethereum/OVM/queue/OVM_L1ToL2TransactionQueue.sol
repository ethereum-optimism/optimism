// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.7.0;
pragma experimental ABIEncoderV2;

/* Proxy Imports */
import { Proxy_Resolver } from "../../proxy/Proxy_Resolver.sol";

/* Library Imports */
import { Lib_OVMCodec } from "../../libraries/codec/Lib_OVMCodec.sol";

/* Interface Imports */
import { iOVM_L1ToL2TransactionQueue } from "../../iOVM/queue/iOVM_L1ToL2TransactionQueue.sol";

/* Contract Imports */
import { OVM_BaseQueue } from "./OVM_BaseQueue.sol";

/**
 * @title OVM_L1ToL2TransactionQueue
 */
contract OVM_L1ToL2TransactionQueue is iOVM_L1ToL2TransactionQueue, OVM_BaseQueue, Proxy_Resolver {

    /*******************************************
     * Contract Variables: Contract References *
     *******************************************/

    address internal ovmCanonicalTransactionChain;


    /***************
     * Constructor *
     ***************/

    /**
     * @param _proxyManager Address of the Proxy_Manager.
     */
    constructor(
        address _proxyManager
    )
        Proxy_Resolver(_proxyManager)
    {
        ovmCanonicalTransactionChain = resolve("OVM_CanonicalTransactionChain");
    }


    /****************************************
     * Public Functions: Queue Manipulation *
     ****************************************/

    /**
     * Adds a transaction to the queue.
     * @param _target Target contract to send the transaction to.
     * @param _gasLimit Gas limit for the given transaction.
     * @param _data Transaction data.
     */
    function enqueue(
        address _target,
        uint256 _gasLimit,
        bytes memory _data
    )
        override
        public
    {
        Lib_OVMCodec.QueueElement memory element = Lib_OVMCodec.QueueElement({
            timestamp: block.timestamp,
            batchRoot: keccak256(abi.encodePacked(
                _target,
                _gasLimit,
                _data
            )),
            isL1ToL2Batch: true
        });

        _enqueue(element);
    }

    /**
     * Pops an element from the queue.
     */
    function dequeue()
        override
        public
    {
        require(
            msg.sender == ovmCanonicalTransactionChain,
            "Sender is not allowed to enqueue."
        );

        _dequeue();
    }
}
