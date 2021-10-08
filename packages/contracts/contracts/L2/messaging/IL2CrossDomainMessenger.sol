// SPDX-License-Identifier: MIT
pragma solidity ^0.8.9;

/* Interface Imports */
import { ICrossDomainMessenger } from "../../libraries/bridge/ICrossDomainMessenger.sol";

/**
 * @title IL2CrossDomainMessenger
 */
interface IL2CrossDomainMessenger is ICrossDomainMessenger {
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
