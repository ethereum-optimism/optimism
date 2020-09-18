// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.7.0;
pragma experimental ABIEncoderV2;

/* Proxy Imports */
import { Proxy_Resolver } from "../../proxy/Proxy_Resolver.sol";

/* Library Imports */
import { Lib_OVMCodec } from "../../libraries/codec/Lib_OVMCodec.sol";

/* Interface Imports */
import { iOVM_L1ToL2TransactionQueue } from "../../iOVM/queue/iOVM_L1ToL2TransactionQueue.sol";
import { iOVM_CanonicalTransactionChain } from "../../iOVM/chain/iOVM_CanonicalTransactionChain.sol";

/* Contract Imports */
import { OVM_BaseQueue } from "./OVM_BaseQueue.sol";

/**
 * @title OVM_L1ToL2TransactionQueue
 */
contract OVM_L1ToL2TransactionQueue is iOVM_L1ToL2TransactionQueue, OVM_BaseQueue, Proxy_Resolver {

    /*******************************************
     * Contract Variables: Contract References *
     *******************************************/

    iOVM_CanonicalTransactionChain internal ovmCanonicalTransactionChain;


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
        ovmCanonicalTransactionChain = iOVM_CanonicalTransactionChain(resolve("OVM_CanonicalTransactionChain"));
    }


    /****************************************
     * Public Functions: Queue Manipulation *
     ****************************************/

    /**
     * Adds an element to the queue.
     * @param _element Queue element to add to the queue.
     */
    function enqueue(
        Lib_OVMCodec.QueueElement memory _element
    )
        override
        public
    {
        _enqueue(_element);
    }

    /**
     * Pops an element from the queue.
     * @return _element Queue element popped from the queue.
     */
    function dequeue()
        override
        public
        returns (
            Lib_OVMCodec.QueueElement memory _element
        )
    {
        require(
            msg.sender == address(ovmCanonicalTransactionChain),
            "Sender is not allowed to enqueue."
        );

        return _dequeue();
    }
}
