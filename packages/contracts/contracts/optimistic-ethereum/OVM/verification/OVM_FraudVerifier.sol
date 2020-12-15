// SPDX-License-Identifier: MIT
pragma solidity ^0.7.0;
pragma experimental ABIEncoderV2;

/* Library Imports */
import { Lib_OVMCodec } from "../../libraries/codec/Lib_OVMCodec.sol";
import { Lib_AddressResolver } from "../../libraries/resolver/Lib_AddressResolver.sol";

/* Interface Imports */
import { iOVM_FraudVerifier } from "../../iOVM/verification/iOVM_FraudVerifier.sol";
import { iOVM_StateTransitioner } from "../../iOVM/verification/iOVM_StateTransitioner.sol";
import { iOVM_StateTransitionerFactory } from "../../iOVM/verification/iOVM_StateTransitionerFactory.sol";
import { iOVM_BondManager } from "../../iOVM/verification/iOVM_BondManager.sol";
import { iOVM_StateCommitmentChain } from "../../iOVM/chain/iOVM_StateCommitmentChain.sol";
import { iOVM_CanonicalTransactionChain } from "../../iOVM/chain/iOVM_CanonicalTransactionChain.sol";

/* Contract Imports */
import { OVM_FraudContributor } from "./OVM_FraudContributor.sol";

contract OVM_FraudVerifier is Lib_AddressResolver, OVM_FraudContributor, iOVM_FraudVerifier {

    /*******************************************
     * Contract Variables: Internal Accounting *
     *******************************************/

    mapping (bytes32 => iOVM_StateTransitioner) internal transitioners;


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
    {}


    /***************************************
     * Public Functions: Transition Status *
     ***************************************/

    /**
     * Retrieves the state transitioner for a given root.
     * @param _preStateRoot State root to query a transitioner for.
     * @return _transitioner Corresponding state transitioner contract.
     */
    function getStateTransitioner(
        bytes32 _preStateRoot,
        bytes32 _txHash
    )
        override
        public
        view
        returns (
            iOVM_StateTransitioner _transitioner
        )
    {
        return transitioners[keccak256(abi.encodePacked(_preStateRoot, _txHash))];
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
     * @param _txChainElement OVM transaction chain element.
     * @param _transactionBatchHeader Batch header for the provided transaction.
     * @param _transactionProof Inclusion proof for the provided transaction.
     */
    function initializeFraudVerification(
        bytes32 _preStateRoot,
        Lib_OVMCodec.ChainBatchHeader memory _preStateRootBatchHeader,
        Lib_OVMCodec.ChainInclusionProof memory _preStateRootProof,
        Lib_OVMCodec.Transaction memory _transaction,
        Lib_OVMCodec.TransactionChainElement memory _txChainElement,
        Lib_OVMCodec.ChainBatchHeader memory _transactionBatchHeader,
        Lib_OVMCodec.ChainInclusionProof memory _transactionProof
    )
        override
        public
        contributesToFraudProof(_preStateRoot, Lib_OVMCodec.hashTransaction(_transaction))
    {
        bytes32 _txHash = Lib_OVMCodec.hashTransaction(_transaction);

        if (_hasStateTransitioner(_preStateRoot, _txHash)) {
            return;
        }

        iOVM_StateCommitmentChain ovmStateCommitmentChain = iOVM_StateCommitmentChain(resolve("OVM_StateCommitmentChain"));
        iOVM_CanonicalTransactionChain ovmCanonicalTransactionChain = iOVM_CanonicalTransactionChain(resolve("OVM_CanonicalTransactionChain"));

        require(
            ovmStateCommitmentChain.verifyStateCommitment(
                _preStateRoot,
                _preStateRootBatchHeader,
                _preStateRootProof
            ),
            "Invalid pre-state root inclusion proof."
        );

        require(
            ovmCanonicalTransactionChain.verifyTransaction(
                _transaction,
                _txChainElement,
                _transactionBatchHeader,
                _transactionProof
            ),
            "Invalid transaction inclusion proof."
        );

        require (
            _preStateRootBatchHeader.prevTotalElements + _preStateRootProof.index + 1 == _transactionBatchHeader.prevTotalElements + _transactionProof.index,
            "Pre-state root global index must equal to the transaction root global index."
        );

        _deployTransitioner(_preStateRoot, _txHash, _preStateRootProof.index);
    }

    /**
     * Finalizes the fraud verification process.
     * @param _preStateRoot State root before the fraudulent transaction.
     * @param _preStateRootBatchHeader Batch header for the provided pre-state root.
     * @param _preStateRootProof Inclusion proof for the provided pre-state root.
     * @param _txHash The transaction for the state root
     * @param _postStateRoot State root after the fraudulent transaction.
     * @param _postStateRootBatchHeader Batch header for the provided post-state root.
     * @param _postStateRootProof Inclusion proof for the provided post-state root.
     */
    function finalizeFraudVerification(
        bytes32 _preStateRoot,
        Lib_OVMCodec.ChainBatchHeader memory _preStateRootBatchHeader,
        Lib_OVMCodec.ChainInclusionProof memory _preStateRootProof,
        bytes32 _txHash,
        bytes32 _postStateRoot,
        Lib_OVMCodec.ChainBatchHeader memory _postStateRootBatchHeader,
        Lib_OVMCodec.ChainInclusionProof memory _postStateRootProof
    )
        override
        public
        contributesToFraudProof(_preStateRoot, _txHash)
    {
        iOVM_StateTransitioner transitioner = getStateTransitioner(_preStateRoot, _txHash);
        iOVM_StateCommitmentChain ovmStateCommitmentChain = iOVM_StateCommitmentChain(resolve("OVM_StateCommitmentChain"));
        iOVM_BondManager ovmBondManager = iOVM_BondManager(resolve("OVM_BondManager"));

        require(
            transitioner.isComplete() == true,
            "State transition process must be completed prior to finalization."
        );

        require (
            _postStateRootBatchHeader.prevTotalElements + _postStateRootProof.index == _preStateRootBatchHeader.prevTotalElements + _preStateRootProof.index + 1,
            "Post-state root global index must equal to the pre state root global index plus one."
        );

        require(
            ovmStateCommitmentChain.verifyStateCommitment(
                _preStateRoot,
                _preStateRootBatchHeader,
                _preStateRootProof
            ),
            "Invalid pre-state root inclusion proof."
        );

        require(
            ovmStateCommitmentChain.verifyStateCommitment(
                _postStateRoot,
                _postStateRootBatchHeader,
                _postStateRootProof
            ),
            "Invalid post-state root inclusion proof."
        );

        // If the post state root did not match, then there was fraud and we should delete the batch
        require(
            _postStateRoot != transitioner.getPostStateRoot(),
            "State transition has not been proven fraudulent."
        );
        
        _cancelStateTransition(_postStateRootBatchHeader, _preStateRoot);

        // TEMPORARY: Remove the transitioner; for minnet.
        transitioners[keccak256(abi.encodePacked(_preStateRoot, _txHash))] = iOVM_StateTransitioner(0x0000000000000000000000000000000000000000);
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
        bytes32 _preStateRoot,
        bytes32 _txHash
    )
        internal
        view
        returns (
            bool _exists
        )
    {
        return address(getStateTransitioner(_preStateRoot, _txHash)) != address(0);
    }

    /**
     * Deploys a new state transitioner.
     * @param _preStateRoot Pre-state root to initialize the transitioner with.
     * @param _txHash Hash of the transaction this transitioner will execute.
     * @param _stateTransitionIndex Index of the transaction in the chain.
     */
    function _deployTransitioner(
        bytes32 _preStateRoot,
        bytes32 _txHash,
        uint256 _stateTransitionIndex
    )
        internal
    {
        transitioners[keccak256(abi.encodePacked(_preStateRoot, _txHash))] = iOVM_StateTransitionerFactory(
            resolve("OVM_StateTransitionerFactory")
        ).create(
            address(libAddressManager),
            _stateTransitionIndex,
            _preStateRoot,
            _txHash
        );
    }

    /**
     * Removes a state transition from the state commitment chain.
     * @param _postStateRootBatchHeader Header for the post-state root.
     * @param _preStateRoot Pre-state root hash.
     */
    function _cancelStateTransition(
        Lib_OVMCodec.ChainBatchHeader memory _postStateRootBatchHeader,
        bytes32 _preStateRoot
    )
        internal
    {
        iOVM_StateCommitmentChain ovmStateCommitmentChain = iOVM_StateCommitmentChain(resolve("OVM_StateCommitmentChain"));
        iOVM_BondManager ovmBondManager = iOVM_BondManager(resolve("OVM_BondManager"));

        // Delete the state batch.
        ovmStateCommitmentChain.deleteStateBatch(
            _postStateRootBatchHeader
        );

        // Get the timestamp and publisher for that block.
        (uint256 timestamp, address publisher) = abi.decode(_postStateRootBatchHeader.extraData, (uint256, address));

        // Slash the bonds at the bond manager.
        ovmBondManager.finalize(
            _preStateRoot,
            publisher,
            timestamp
        );
    }
}
