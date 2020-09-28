// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.7.0;
pragma experimental ABIEncoderV2;

/* Library Imports */
import { Lib_AddressResolver } from "../../libraries/resolver/Lib_AddressResolver.sol";
import { Lib_EthMerkleTrie } from "../../libraries/trie/Lib_EthMerkleTrie.sol";
import { Lib_ByteUtils } from "../../libraries/utils/Lib_ByteUtils.sol";

/* Interface Imports */
import { iOVM_L1CrossDomainMessenger } from "../../iOVM/bridge/iOVM_L1CrossDomainMessenger.sol";
import { iOVM_L1ToL2TransactionQueue } from "../../iOVM/queue/iOVM_L1ToL2TransactionQueue.sol";
import { iOVM_StateCommitmentChain } from "../../iOVM/chain/iOVM_StateCommitmentChain.sol";

/* Contract Imports */
import { OVM_BaseCrossDomainMessenger } from "./OVM_BaseCrossDomainMessenger.sol";

/**
 * @title OVM_L1CrossDomainMessenger
 */
contract OVM_L1CrossDomainMessenger is iOVM_L1CrossDomainMessenger, OVM_BaseCrossDomainMessenger, Lib_AddressResolver {
    
    /*******************************************
     * Contract Variables: Contract References *
     *******************************************/
    
    iOVM_L1ToL2TransactionQueue internal ovmL1ToL2TransactionQueue;
    iOVM_StateCommitmentChain internal ovmStateCommitmentChain;


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
        ovmL1ToL2TransactionQueue = iOVM_L1ToL2TransactionQueue(resolve("OVM_L1ToL2TransactionQueue"));
        ovmStateCommitmentChain = iOVM_StateCommitmentChain(resolve("OVM_StateCommitmentChain"));
    }


    /********************
     * Public Functions *
     ********************/

    /**
     * Relays a cross domain message to a contract.
     * @inheritdoc iOVM_L1CrossDomainMessenger
     */
    function relayMessage(
        address _target,
        address _sender,
        bytes memory _message,
        uint256 _messageNonce,
        L2MessageInclusionProof memory _proof
    )
        override
        public
    {
        bytes memory xDomainCalldata = _getXDomainCalldata(
            _target,
            _sender,
            _message,
            _messageNonce
        );

        require(
            _verifyXDomainMessage(
                xDomainCalldata,
                _proof
            ) == true,
            "Provided message could not be verified."
        );

        require(
            receivedMessages[keccak256(xDomainCalldata)] == false,
            "Provided message has already been received."
        );

        xDomainMessageSender = _sender;
        _target.call(_message);

        // Messages are considered successfully executed if they complete
        // without running out of gas (revert or not). As a result, we can
        // ignore the result of the call and always mark the message as
        // successfully executed because we won't get here unless we have
        // enough gas left over.
        receivedMessages[keccak256(xDomainCalldata)] = true;
    }

    /**
     * Sends a cross domain message to the target messenger.
     * @inheritdoc iOVM_L1CrossDomainMessenger
     */
    function sendMessage(
        address _target,
        bytes memory _message,
        uint32 _gasLimit
    )
        override
        public
    {
        bytes memory xDomainCalldata = _getXDomainCalldata(
            _target,
            msg.sender,
            _message,
            messageNonce
        );

        _sendXDomainMessage(xDomainCalldata, _gasLimit);

        messageNonce += 1;
        sentMessages[keccak256(xDomainCalldata)] = true;
    }

    /**
     * Replays a cross domain message to the target messenger.
     * @inheritdoc iOVM_L1CrossDomainMessenger
     */
    function replayMessage(
        address _target,
        address _sender,
        bytes memory _message,
        uint256 _messageNonce,
        uint32 _gasLimit
    )
        override
        public
    {
        bytes memory xDomainCalldata = _getXDomainCalldata(
            _target,
            _sender,
            _message,
            _messageNonce
        );

        require(
            sentMessages[keccak256(xDomainCalldata)] == true,
            "Provided message has not already been sent."
        );

        _sendXDomainMessage(xDomainCalldata, _gasLimit);
    }


    /**********************
     * Internal Functions *
     **********************/

    /**
     * Verifies that the given message is valid.
     * @param _xDomainCalldata Calldata to verify.
     * @param _proof Inclusion proof for the message.
     * @return Whether or not the provided message is valid.
     */
    function _verifyXDomainMessage(
        bytes memory _xDomainCalldata,
        L2MessageInclusionProof memory _proof
    )
        internal
        returns (
            bool
        )
    {
        return (
            _verifyStateRootProof(_proof)
            && _verifyStorageProof(_xDomainCalldata, _proof)
        );
    }

    /**
     * Verifies that the state root within an inclusion proof is valid.
     * @param _proof Message inclusion proof.
     * @return Whether or not the provided proof is valid.
     */
    function _verifyStateRootProof(
        L2MessageInclusionProof memory _proof
    )
        internal
        returns (
            bool
        )
    {
        // TODO: We *must* verify that the batch timestamp is sufficiently old.
        // However, this requires that we first add timestamps to state batches
        // and account for that change in various tests. Change of that size is
        // out of scope for this ticket, so "TODO" for now.

        return ovmStateCommitmentChain.verifyElement(
            abi.encodePacked(_proof.stateRoot),
            _proof.stateRootBatchHeader,
            _proof.stateRootProof
        );
    }

    /**
     * Verifies that the storage proof within an inclusion proof is valid.
     * @param _xDomainCalldata Encoded message calldata.
     * @param _proof Message inclusion proof.
     * @return Whether or not the provided proof is valid.
     */
    function _verifyStorageProof(
        bytes memory _xDomainCalldata,
        L2MessageInclusionProof memory _proof
    )
        internal
        returns (
            bool
        )
    {
        bytes32 storageKey = keccak256(
            Lib_ByteUtils.concat(
                abi.encodePacked(keccak256(_xDomainCalldata)),
                abi.encodePacked(uint256(0))
            )
        );

        return Lib_EthMerkleTrie.proveAccountStorageSlotValue(
            0x4200000000000000000000000000000000000000,
            storageKey,
            bytes32(uint256(1)),
            _proof.stateTrieWitness,
            _proof.storageTrieWitness,
            _proof.stateRoot
        );
    }

    /**
     * Sends a cross domain message.
     * @param _message Message to send.
     * @param _gasLimit OVM gas limit for the message.
     */
    function _sendXDomainMessage(
        bytes memory _message,
        uint32 _gasLimit
    )
        internal
    {
        ovmL1ToL2TransactionQueue.enqueue(
            targetMessengerAddress,
            _gasLimit,
            _message
        );
    }
}
