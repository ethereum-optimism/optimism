pragma solidity ^0.5.0;
pragma experimental ABIEncoderV2;

/* Library Imports */
import { ContractResolver } from "../utils/resolvers/ContractResolver.sol";
import { EthMerkleTrie } from "../utils/libraries/EthMerkleTrie.sol";
import { BytesLib } from "../utils/libraries/BytesLib.sol";
import { DataTypes } from "../utils/libraries/DataTypes.sol";

/* Contract Imports */
import { BaseCrossDomainMessenger } from "./BaseCrossDomainMessenger.sol";
import { L1ToL2TransactionQueue } from "../queue/L1ToL2TransactionQueue.sol";
import { StateCommitmentChain } from "../chain/StateCommitmentChain.sol";

/**
 * @title L1CrossDomainMessenger
 */
contract L1CrossDomainMessenger is BaseCrossDomainMessenger, ContractResolver {

    event RelayedL2ToL1Message(bytes32 msgHash);

    /*
     * Data Structures
     */

    struct L2MessageInclusionProof {
        bytes32 stateRoot;
        uint256 stateRootIndex;
        DataTypes.StateElementInclusionProof stateRootProof;
        bytes stateTrieWitness;
        bytes storageTrieWitness;
    }
    /*
     * Constructor
     */

    /**
     * @param _addressResolver Address of the AddressResolver contract.
     */
    constructor(
        address _addressResolver
    )
        public
        ContractResolver(_addressResolver)
    {}


    /*
     * Public Functions
     */

    /**
     * Relays a cross domain message to a contract.
     * .inheritdoc IL1CrossDomainMessenger
     */
    function relayMessage(
        address _target,
        address _sender,
        bytes memory _message,
        uint256 _messageNonce,
        L2MessageInclusionProof memory _proof
    )
        public
    {
        bytes memory xDomainCalldata = _getXDomainCalldata(
            _target,
            _sender,
            _message,
            _messageNonce
        );
        bytes32 msgHash = keccak256(xDomainCalldata);

        require(
            _verifyXDomainMessage(
                xDomainCalldata,
                _proof
            ) == true,
            "Provided message could not be verified."
        );

        require(
            receivedMessages[msgHash] == false,
            "Provided message has already been received."
        );

        xDomainMessageSender = _sender;
        _target.call(_message);

        // Messages are considered successfully executed if they complete
        // without running out of gas (revert or not). As a result, we can
        // ignore the result of the call and always mark the message as
        // successfully executed because we won't get here unless we have
        // enough gas left over.
        receivedMessages[msgHash] = true;

        emit RelayedL2ToL1Message(msgHash);
    }

    /**
     * Replays a cross domain message to the target messenger.
     * .inheritdoc IL1CrossDomainMessenger
     */
    function replayMessage(
        address _target,
        address _sender,
        bytes memory _message,
        uint256 _messageNonce,
        uint32 _gasLimit
    )
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


    /*
     * Internal Functions
     */

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
            _verifyStateRootProof(_proof) && _verifyStorageProof(_xDomainCalldata, _proof)
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

        StateCommitmentChain stateCommitmentChain = resolveStateCommitmentChain();
        return stateCommitmentChain.verifyElement(
            abi.encodePacked(_proof.stateRoot),
            _proof.stateRootIndex,
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
            BytesLib.concat(
                abi.encodePacked(keccak256(_xDomainCalldata)),
                abi.encodePacked(uint256(0))
            )
        );

        return EthMerkleTrie.proveAccountStorageSlotValue(
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
        L1ToL2TransactionQueue l1ToL2TransactionQueue = resolveL1ToL2TransactionQueue();
        l1ToL2TransactionQueue.enqueueL1ToL2Message(
            targetMessengerAddress,
            _gasLimit,
            _message
        );
    }
    

    /*
     * Contract Resolution
     */

    function resolveL1ToL2TransactionQueue()
        internal
        view
        returns (L1ToL2TransactionQueue)
    {
        return L1ToL2TransactionQueue(resolveContract("L1ToL2TransactionQueue"));
    }

    function resolveStateCommitmentChain()
        internal
        view
        returns (StateCommitmentChain)
    {
        return StateCommitmentChain(resolveContract("StateCommitmentChain"));
    }
}
