// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.7.0;
pragma experimental ABIEncoderV2;

/* Library Imports */
import { Lib_OVMCodec } from "../../../libraries/codec/Lib_OVMCodec.sol";

/* Interface Imports */
import { iOVM_StateCommitmentChain } from "../../iOVM/chain/iOVM_StateCommitmentChain.sol";
import { iOVM_CanonicalTransactionChain } from "../../iOVM/chain/iOVM_CanonicalTransactionChain.sol";
import { iOVM_FraudVerifier } from "../../iOVM/execution/iOVM_FraudVerifier.sol";

/* Contract Imports */
import { OVM_BaseChain } from "./OVM_BaseChain.sol";

contract OVM_StateCommitmentChain is iOVM_StateCommitmentChain, OVM_BaseChain {
    iOVM_CanonicalTransactionChain internal ovmCanonicalTransactionChain;
    iOVM_FraudVerifier internal ovmFraudVerifier;

    constructor(
        address _ovmCanonicalTransactionChain,
        address _ovmFraudVerifier
    )
        Lib_ContractProxyResolver(_libContractProxyManager)
    {
        ovmCanonicalTransactionChain = iOVM_CanonicalTransactionChain(_ovmCanonicalTransactionChain);
        ovmFraudVerifier = iOVM_FraudVerifier(_ovmFraudVerifier);
    }

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

    function deleteStateBatch(
        Lib_OVMDataTypes.OVMChainBatchHeader memory _batchHeader
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
