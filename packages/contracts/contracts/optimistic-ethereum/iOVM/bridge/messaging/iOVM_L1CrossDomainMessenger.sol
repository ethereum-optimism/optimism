// SPDX-License-Identifier: MIT
pragma solidity >0.5.0 <0.8.0;
pragma experimental ABIEncoderV2;

/* Library Imports */
import { Lib_OVMCodec } from "../../../libraries/codec/Lib_OVMCodec.sol";

/* Interface Imports */
import { iAbs_BaseCrossDomainMessenger } from "./iAbs_BaseCrossDomainMessenger.sol";

/**
 * @title iOVM_L1CrossDomainMessenger
 */
interface iOVM_L1CrossDomainMessenger is iAbs_BaseCrossDomainMessenger {

    /*******************
     * Data Structures *
     *******************/

    struct L2MessageInclusionProof {
        bytes32 stateRoot;
        Lib_OVMCodec.ChainBatchHeader stateRootBatchHeader;
        Lib_OVMCodec.ChainInclusionProof stateRootProof;
        bytes stateTrieWitness;
        bytes storageTrieWitness;
    }


    /********************
     * Public Functions *
     ********************/


    function relayMessage(
        address _target,
        address _sender,
        bytes memory _message,
        uint256 _messageNonce,
        L2MessageInclusionProof memory _proof
    ) external;


    function replayMessage(
        address _target,
        address _sender,
        bytes memory _message,
        uint256 _messageNonce,
        uint32 _gasLimit
    ) external;
}
