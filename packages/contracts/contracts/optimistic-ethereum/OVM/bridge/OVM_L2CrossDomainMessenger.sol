// SPDX-License-Identifier: MIT
// +build ovm
pragma solidity >0.6.0 <0.8.0;
pragma experimental ABIEncoderV2;

/* Library Imports */
import { Lib_AddressResolver } from "../../libraries/resolver/Lib_AddressResolver.sol";

/* Interface Imports */
import { iOVM_L2CrossDomainMessenger } from "../../iOVM/bridge/iOVM_L2CrossDomainMessenger.sol";
import { iOVM_L1MessageSender } from "../../iOVM/precompiles/iOVM_L1MessageSender.sol";
import { iOVM_L2ToL1MessagePasser } from "../../iOVM/precompiles/iOVM_L2ToL1MessagePasser.sol";

/* Contract Imports */
import { OVM_BaseCrossDomainMessenger } from "./OVM_BaseCrossDomainMessenger.sol";

/**
 * @title OVM_L2CrossDomainMessenger
 * @dev L2 CONTRACT (COMPILED)
 */
contract OVM_L2CrossDomainMessenger is iOVM_L2CrossDomainMessenger, OVM_BaseCrossDomainMessenger, Lib_AddressResolver {

    /***************
     * Constructor *
     ***************/
    
    /**
     * @param _libAddressManager Address of the Address Manager.
     */
    constructor(
        address _libAddressManager
    )
        public
        Lib_AddressResolver(_libAddressManager)
    {}


    /********************
     * Public Functions *
     ********************/

    /**
     * Relays a cross domain message to a contract.
     * @inheritdoc iOVM_L2CrossDomainMessenger
     */
    function relayMessage(
        address _target,
        address _sender,
        bytes memory _message,
        uint256 _messageNonce
    )
        override
        public
    {
        require(
            _verifyXDomainMessage() == true,
            "Provided message could not be verified."
        );

        bytes memory xDomainCalldata = _getXDomainCalldata(
            _target,
            _sender,
            _message,
            _messageNonce
        );

        bytes32 xDomainCalldataHash = keccak256(xDomainCalldata);

        require(
            successfulMessages[xDomainCalldataHash] == false,
            "Provided message has already been received."
        );

        xDomainMessageSender = _sender;
        (bool success, ) = _target.call(_message);

        // Mark the message as received if the call was successful. Ensures that a message can be
        // relayed multiple times in the case that the call reverted.
        if (success == true) {
            successfulMessages[xDomainCalldataHash] = true;
            emit RelayedMessage(xDomainCalldataHash);
        }

        // Store an identifier that can be used to prove that the given message was relayed by some
        // user. Gives us an easy way to pay relayers for their work.
        bytes32 relayId = keccak256(
            abi.encodePacked(
                xDomainCalldata,
                msg.sender,
                block.number
            )
        );
        relayedMessages[relayId] = true;
    }


    /**********************
     * Internal Functions *
     **********************/

    /**
     * Verifies that a received cross domain message is valid.
     * @return _valid Whether or not the message is valid.
     */
    function _verifyXDomainMessage()
        internal
        returns (
            bool _valid
        )
    {
        return (
            iOVM_L1MessageSender(resolve("OVM_L1MessageSender")).getL1MessageSender() == resolve("OVM_L1CrossDomainMessenger")
        );
    }

    /**
     * Sends a cross domain message.
     * @param _message Message to send.
     * @param _gasLimit Gas limit for the provided message.
     */
    function _sendXDomainMessage(
        bytes memory _message,
        uint256 _gasLimit
    )
        override
        internal
    {
        iOVM_L2ToL1MessagePasser(resolve("OVM_L2ToL1MessagePasser")).passMessageToL1(_message);
    }
}
