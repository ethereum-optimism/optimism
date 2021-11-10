// SPDX-License-Identifier: MIT
pragma solidity ^0.8.9;

/* Library Imports */
import { Lib_OVMCodec } from "../../libraries/codec/Lib_OVMCodec.sol";

/* Interface Imports */
import { ICrossDomainMessenger } from "../../libraries/bridge/ICrossDomainMessenger.sol";

/**
 * @title IL1CrossDomainMessenger
 */
interface IL1CrossDomainMessenger is ICrossDomainMessenger {
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

    /**
     * Relays a cross domain message to a contract.
     * @param _target Target contract address.
     * @param _sender Message sender address.
     * @param _message Message to send to the target.
     * @param _messageNonce Nonce for the provided message.
     * @param _proof Inclusion proof for the given message.
     */
    function relayMessage(
        address _target,
        address _sender,
        bytes memory _message,
        uint256 _messageNonce,
        L2MessageInclusionProof memory _proof
    ) external;

    /**
     * Replays a cross domain message to the target messenger.
     * @param _target Target contract address.
     * @param _sender Original sender address.
     * @param _message Message to send to the target.
     * @param _queueIndex CTC Queue index for the message to replay.
     * @param _oldGasLimit Original gas limit used to send the message.
     * @param _newGasLimit New gas limit to be used for this message.
     */
    function replayMessage(
        address _target,
        address _sender,
        bytes memory _message,
        uint256 _queueIndex,
        uint32 _oldGasLimit,
        uint32 _newGasLimit
    ) external;
}
