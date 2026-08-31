// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { ISemver } from "interfaces/universal/ISemver.sol";
import { IProxyAdminOwnedBase } from "interfaces/universal/IProxyAdminOwnedBase.sol";
import { Identifier } from "interfaces/L2/ICrossL2Inbox.sol";

/// @title IL2ToL2CrossDomainMessengerReplay
/// @notice Interface for the L2ToL2CrossDomainMessengerReplay contract.
interface IL2ToL2CrossDomainMessengerReplay is ISemver, IProxyAdminOwnedBase {
    error L2ToL2CrossDomainMessengerReplay_Unauthorized();
    error L2ToL2CrossDomainMessengerReplay_ETHBridgeSender();
    error L2ToL2CrossDomainMessengerReplay_ETHBridgeTarget();
    error L2ToL2CrossDomainMessengerReplay_Unsupported();

    event Initialized(uint8 version);

    event SentMessage(
        uint256 indexed destination, address indexed target, uint256 indexed messageNonce, address sender, bytes message
    );

    event ReplayerSet(address indexed replayer);

    function messageVersion() external view returns (uint16);
    function replayer() external view returns (address);

    function initialize(address _replayer) external;
    function setReplayer(address _replayer) external;

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

    function __constructor__() external;
}
