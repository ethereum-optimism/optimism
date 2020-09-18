// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.7.0;
pragma experimental ABIEncoderV2;

/* Proxy Imports */
import { Proxy_Resolver } from "../../proxy/Proxy_Resolver.sol";

/* Library Imports */
import { Lib_OVMCodec } from "../../libraries/codec/Lib_OVMCodec.sol";

/* Interface Imports */
import { iOVM_StateCommitmentChain } from "../../iOVM/chain/iOVM_StateCommitmentChain.sol";
import { iOVM_CanonicalTransactionChain } from "../../iOVM/chain/iOVM_CanonicalTransactionChain.sol";
import { iOVM_FraudVerifier } from "../../iOVM/execution/iOVM_FraudVerifier.sol";

/* Contract Imports */
import { OVM_BaseChain } from "./OVM_BaseChain.sol";

/**
 * @title OVM_StateCommitmentChain
 */
contract OVM_StateCommitmentChain is iOVM_StateCommitmentChain, OVM_BaseChain, Proxy_Resolver {
    
    /*******************************************
     * Contract Variables: Contract References *
     *******************************************/

    iOVM_CanonicalTransactionChain internal ovmCanonicalTransactionChain;
    iOVM_FraudVerifier internal ovmFraudVerifier;


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
        ovmFraudVerifier = iOVM_FraudVerifier(resolve("OVM_FraudVerifier"));
    }


    /****************************************
     * Public Functions: Batch Manipulation *
     ****************************************/

    /**
     * Appends a batch of state roots to the chain.
     * @param _batch Batch of state roots.
     */
    function appendStateBatch(
        bytes32[] memory _batch
    )
        override
        public
    {
        require(
            _batch.length > 0,
            "Cannot submit an empty state batch."
        );

        require(
            getTotalElements() + _batch.length <= ovmCanonicalTransactionChain.getTotalElements(),
            "Number of state roots cannot exceed the number of canonical transactions."
        );

        bytes[] memory elements = new bytes[](_batch.length);
        for (uint256 i = 0; i < _batch.length; i++) {
            elements[i] = abi.encodePacked(_batch[i]);
        }

        _appendBatch(elements);
    }

    /**
     * Deletes all state roots after (and including) a given batch.
     * @param _batchHeader Header of the batch to start deleting from.
     */
    function deleteStateBatch(
        Lib_OVMCodec.ChainBatchHeader memory _batchHeader
    )
        override
        public
    {
        require(
            msg.sender == address(ovmFraudVerifier),
            "State batches can only be deleted by the fraud verifier"
        );

        _deleteBatch(_batchHeader);
    }
}
