pragma solidity ^0.5.0;

/* Library Imports */
import { ContractResolver } from "../utils/resolvers/ContractResolver.sol";

/* Contract Imports */
import { BaseMessenger } from "./BaseMessenger.sol";
import { L1ToL2TransactionQueue } from "../queue/L1ToL2TransactionQueue.sol";
import { CanonicalTransactionChain } from "../chain/CanonicalTransactionChain.sol";

contract L1ToL2Messenger is ContractResolver, BaseMessenger {
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
     * Internal Functions
     */

    /**
     * Verifies that a received cross domain message is valid.
     * @param _target Target contract address.
     * @param _sender Message sender address.
     * @param _message Message to send to the target.
     * @param _messageNonce Nonce for the provided message.
     * @return whether or not the message is valid.
     */
    function _verifyXDomainMessage(
        address _target,
        address _sender,
        bytes memory _message,
        uint256 _messageNonce
    )
        internal
        returns (
            bool
        )
    {
        // TODO: Check that the message was included in the canonical transaction chain.
        return true;
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

    function resolveCanonicalTransactionChain()
        internal
        view
        returns (CanonicalTransactionChain)
    {
        return CanonicalTransactionChain(resolveContract("CanonicalTransactionChain"));
    }
}
