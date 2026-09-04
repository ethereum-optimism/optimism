// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { ISemver } from "interfaces/universal/ISemver.sol";
import { Identifier } from "interfaces/L2/ICrossL2Inbox.sol";

/// @title IL2ToL2CrossDomainMessengerReplay
/// @notice Interface for the L2ToL2CrossDomainMessengerReplay contract.
interface IL2ToL2CrossDomainMessengerReplay is ISemver {
    error L2ToL2CrossDomainMessengerReplay_ETHBridgeSender();
    error L2ToL2CrossDomainMessengerReplay_ETHBridgeTarget();
    error L2ToL2CrossDomainMessengerReplay_Unsupported();

    event SentMessage(
        uint256 indexed destination, address indexed target, uint256 indexed messageNonce, address sender, bytes message
    );

    function messageVersion() external view returns (uint16);

    function replaySentMessage(
        uint256 _destination,
        uint256 _nonce,
        address _sender,
        address _target,
        bytes calldata _message
    )
        external
        returns (bytes32 messageHash_);

    function sendMessage(uint256, address, bytes calldata) external returns (bytes32);
    function resendMessage(uint256, uint256, address, address, bytes calldata) external returns (bytes32);
    function relayMessage(Identifier calldata, bytes calldata) external payable returns (bytes memory);
    function crossDomainMessageSender() external view returns (address);
    function crossDomainMessageSource() external view returns (uint256);
    function crossDomainMessageContext() external view returns (address, uint256);
    function messageNonce() external view returns (uint256);
    function successfulMessages(bytes32) external view returns (bool);
    function sentMessages(uint256) external view returns (bytes32);

}
