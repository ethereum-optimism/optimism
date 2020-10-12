// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.7.0;
pragma experimental ABIEncoderV2;

/* Library Imports */
import { Lib_OVMCodec } from "../../libraries/codec/Lib_OVMCodec.sol";
import { Lib_AddressResolver } from "../../libraries/resolver/Lib_AddressResolver.sol";

/* Interface Imports */
import { iOVM_FraudVerifier } from "../../iOVM/verification/iOVM_FraudVerifier.sol";
import { iOVM_StateCommitmentChain } from "../../iOVM/chain/iOVM_StateCommitmentChain.sol";
import { iOVM_CanonicalTransactionChain } from "../../iOVM/chain/iOVM_CanonicalTransactionChain.sol";

/* Contract Imports */
import { OVM_BaseChain } from "./OVM_BaseChain.sol";

/**
 * @title OVM_StateCommitmentChain
 */
contract OVM_StateCommitmentChain is iOVM_StateCommitmentChain, OVM_BaseChain, Lib_AddressResolver {

    /*************
     * Constants *
     *************/

    uint256 constant public FRAUD_PROOF_WINDOW = 7 days;

    
    /*******************************************
     * Contract Variables: Contract References *
     *******************************************/

    iOVM_CanonicalTransactionChain internal ovmCanonicalTransactionChain;
    iOVM_FraudVerifier internal ovmFraudVerifier;


    /***************
     * Constructor *
     ***************/

    /**
     * @param _libAddressManager Address of the Address Manager.
     */
    constructor(
        address _libAddressManager
    )
        Lib_AddressResolver(_libAddressManager)
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
            "State batches can only be deleted by the OVM_FraudVerifier."
        );

        require(
            insideFraudProofWindow(_batchHeader),
            "State batches can only be deleted within the fraud proof window."
        );

        _deleteBatch(_batchHeader);
    }


    /**********************************
     * Public Functions: Batch Status *
     **********************************/

    function insideFraudProofWindow(
        Lib_OVMCodec.ChainBatchHeader memory _batchHeader
    )
        override
        public
        view
        returns (
            bool _inside
        )
    {
        require(
            _batchHeader.timestamp != 0,
            "Batch header timestamp cannot be zero"
        );

        return _batchHeader.timestamp + FRAUD_PROOF_WINDOW > block.timestamp;
    }
}
