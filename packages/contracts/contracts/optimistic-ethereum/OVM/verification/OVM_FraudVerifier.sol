// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.7.0;
pragma experimental ABIEncoderV2;

/* Proxy Imports */
import { Proxy_Resolver } from "../../proxy/Proxy_Resolver.sol";

/* Library Imports */
import { Lib_OVMCodec } from "../../libraries/codec/Lib_OVMCodec.sol";

/* Interface Imports */
import { iOVM_FraudVerifier } from "../../iOVM/verification/iOVM_FraudVerifier.sol";
import { iOVM_StateTransitioner } from "../../iOVM/verification/iOVM_StateTransitioner.sol";
import { iOVM_StateTransitionerFactory } from "../../iOVM/verification/iOVM_StateTransitionerFactory.sol";
import { iOVM_ExecutionManager } from "../../iOVM/execution/iOVM_ExecutionManager.sol";
import { iOVM_StateManagerFactory } from "../../iOVM/execution/iOVM_StateManagerFactory.sol";
import { iOVM_StateCommitmentChain } from "../../iOVM/chain/iOVM_StateCommitmentChain.sol";
import { iOVM_CanonicalTransactionChain } from "../../iOVM/chain/iOVM_CanonicalTransactionChain.sol";

contract OVM_FraudVerifier is iOVM_FraudVerifier, Proxy_Resolver {

    /*******************************************
     * Contract Variables: Contract References *
     *******************************************/

    iOVM_StateCommitmentChain internal ovmStateCommitmentChain;
    iOVM_CanonicalTransactionChain internal ovmCanonicalTransactionChain;

    
    /*******************************************
     * Contract Variables: Internal Accounting *
     *******************************************/

    mapping (bytes32 => iOVM_StateTransitioner) internal transitioners;
    

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
        ovmStateCommitmentChain = iOVM_StateCommitmentChain(resolve("OVM_StateCommitmentChain"));
        ovmCanonicalTransactionChain = iOVM_CanonicalTransactionChain(resolve("OVM_CanonicalTransactionChain"));
    }


    /***************************************
     * Public Functions: Transition Status *
     ***************************************/

    /**
     * Retrieves the state transitioner for a given root.
     * @param _preStateRoot State root to query a transitioner for.
     * @return _transitioner Corresponding state transitioner contract.
     */
    function getStateTransitioner(
        bytes32 _preStateRoot
    )
        override
        public
        view
        returns (
            iOVM_StateTransitioner _transitioner
        )
    {
        return transitioners[_preStateRoot];
    }


    /****************************************
     * Public Functions: Fraud Verification *
     ****************************************/

    /**
     * Begins the fraud verification process.
     * @param _preStateRoot State root before the fraudulent transaction.
     * @param _preStateRootBatchHeader Batch header for the provided pre-state root.
     * @param _preStateRootProof Inclusion proof for the provided pre-state root.
     * @param _transaction OVM transaction claimed to be fraudulent.
     * @param _transactionBatchHeader Batch header for the provided transaction.
     * @param _transactionProof Inclusion proof for the provided transaction.
     */
    function initializeFraudVerification(
        bytes32 _preStateRoot,
        Lib_OVMCodec.ChainBatchHeader memory _preStateRootBatchHeader,
        Lib_OVMCodec.ChainInclusionProof memory _preStateRootProof,
        Lib_OVMCodec.Transaction memory _transaction,
        Lib_OVMCodec.ChainBatchHeader memory _transactionBatchHeader,
        Lib_OVMCodec.ChainInclusionProof memory _transactionProof
    )
        override
        public
    {
        if (_hasStateTransitioner(_preStateRoot)) {
            return;
        }

        require(
            _verifyStateRoot(
                _preStateRoot,
                _preStateRootBatchHeader,
                _preStateRootProof
            ),
            "Invalid pre-state root inclusion proof."
        );

        require(
            _verifyTransaction(
                _transaction,
                _transactionBatchHeader,
                _transactionProof
            ),
            "Invalid transaction inclusion proof."
        );

        transitioners[_preStateRoot] = iOVM_StateTransitionerFactory(
            resolve("OVM_StateTransitionerFactory")
        ).create(
            address(proxyManager),
            _preStateRootProof.index,
            _preStateRoot,
            Lib_OVMCodec.hashTransaction(_transaction)
        );
    }

    /**
     * Finalizes the fraud verification process.
     * @param _preStateRoot State root before the fraudulent transaction.
     * @param _preStateRootBatchHeader Batch header for the provided pre-state root.
     * @param _preStateRootProof Inclusion proof for the provided pre-state root.
     * @param _postStateRoot State root after the fraudulent transaction.
     * @param _postStateRootBatchHeader Batch header for the provided post-state root.
     * @param _postStateRootProof Inclusion proof for the provided post-state root.
     */
    function finalizeFraudVerification(
        bytes32 _preStateRoot,
        Lib_OVMCodec.ChainBatchHeader memory _preStateRootBatchHeader,
        Lib_OVMCodec.ChainInclusionProof memory _preStateRootProof,
        bytes32 _postStateRoot,
        Lib_OVMCodec.ChainBatchHeader memory _postStateRootBatchHeader,
        Lib_OVMCodec.ChainInclusionProof memory _postStateRootProof
    )
        override
        public
    {
        iOVM_StateTransitioner transitioner = transitioners[_preStateRoot];

        require(
            transitioner.isComplete() == true,
            "State transition process must be completed prior to finalization."
        );

        require(
            _postStateRootProof.index == _preStateRootProof.index + 1,
            "Invalid post-state root index."
        );

        require(
            _verifyStateRoot(
                _preStateRoot,
                _preStateRootBatchHeader,
                _preStateRootProof
            ),
            "Invalid pre-state root inclusion proof."
        );

        require(
            _verifyStateRoot(
                _postStateRoot,
                _postStateRootBatchHeader,
                _postStateRootProof
            ),
            "Invalid post-state root inclusion proof."
        );

        require(
            _postStateRoot != transitioner.getPostStateRoot(),
            "State transition has not been proven fraudulent."
        );

        ovmStateCommitmentChain.deleteStateBatch(
            _postStateRootBatchHeader
        );
    }


    /************************************
     * Internal Functions: Verification *
     ************************************/

    /**
     * Checks whether a transitioner already exists for a given pre-state root.
     * @param _preStateRoot Pre-state root to check.
     * @return _exists Whether or not we already have a transitioner for the root.
     */
    function _hasStateTransitioner(
        bytes32 _preStateRoot
    )
        internal
        view
        returns (
            bool _exists
        )
    {
        return address(transitioners[_preStateRoot]) != address(0);
    }

    /**
     * Verifies inclusion of a state root.
     * @param _stateRoot State root to verify
     * @param _stateRootBatchHeader Batch header for the provided state root.
     * @param _stateRootProof Inclusion proof for the provided state root.
     * @return _verified Whether or not the root was included.
     */
    function _verifyStateRoot(
        bytes32 _stateRoot,
        Lib_OVMCodec.ChainBatchHeader memory _stateRootBatchHeader,
        Lib_OVMCodec.ChainInclusionProof memory _stateRootProof
    )
        internal
        view
        returns (
            bool _verified
        )
    {
        return ovmStateCommitmentChain.verifyElement(
            abi.encodePacked(_stateRoot),
            _stateRootBatchHeader,
            _stateRootProof
        );
    }

    /**
     * Verifies inclusion of a given transaction.
     * @param _transaction OVM transaction to verify.
     * @param _transactionBatchHeader Batch header for the provided transaction.
     * @param _transactionProof Inclusion proof for the provided transaction.
     * @return _verified Whether or not the transaction was included.
     */
    function _verifyTransaction(
        Lib_OVMCodec.Transaction memory _transaction,
        Lib_OVMCodec.ChainBatchHeader memory _transactionBatchHeader,
        Lib_OVMCodec.ChainInclusionProof memory _transactionProof
    )
        internal
        view
        returns (
            bool _verified
        )
    {
        return ovmCanonicalTransactionChain.verifyElement(
            Lib_OVMCodec.encodeTransaction(_transaction),
            _transactionBatchHeader,
            _transactionProof
        );
    }
}
