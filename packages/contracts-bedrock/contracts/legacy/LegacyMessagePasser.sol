// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Semver } from "../universal/Semver.sol";

/**
 * @custom:legacy
 * @custom:proxied
 * @custom:predeploy 0x4200000000000000000000000000000000000000
 * @title LegacyMessagePasser
 * @notice The LegacyMessagePasser was the low-level mechanism used to send messages from L2 to L1
 *         before the Bedrock upgrade. It is now deprecated in favor of the new MessagePasser.
 */
contract LegacyMessagePasser is Semver {
    /**
     * @notice Mapping of sent message hashes to boolean status.
     */
    mapping(bytes32 => bool) public sentMessages;

    /**
     * @custom:semver 1.0.0
     */
    constructor() Semver(1, 0, 0) {}

    /**
     * @notice Passes a message to L1.
     *
     * @param _message Message to pass to L1.
     */
    function passMessageToL1(bytes memory _message) external {
        sentMessages[keccak256(abi.encodePacked(_message, msg.sender))] = true;
    }
}
