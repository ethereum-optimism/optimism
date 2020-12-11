// SPDX-License-Identifier: MIT
// +build ovm
pragma solidity >0.6.0 <0.8.0;
pragma experimental ABIEncoderV2;

/* Interface Imports */
import { iOVM_BaseCrossDomainMessenger } from "./iOVM_BaseCrossDomainMessenger.sol";

/**
 * @title iOVM_L2CrossDomainMessenger
 */
interface iOVM_L2CrossDomainMessenger is iOVM_BaseCrossDomainMessenger {

    /********************
     * Public Functions * 
     ********************/

    /**
     * Relays a cross domain message to a contract.
     * @param _target Target contract address.
     * @param _sender Message sender address.
     * @param _message Message to send to the target.
     * @param _messageNonce Nonce for the provided message.
     */
    function relayMessage(
        address _target,
        address _sender,
        bytes memory _message,
        uint256 _messageNonce
    ) external;
}
