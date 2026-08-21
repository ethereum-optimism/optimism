// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { IProxyAdminOwnedBase } from "interfaces/universal/IProxyAdminOwnedBase.sol";

/// @title IMessageExpiryRelay
/// @notice Interface for the MessageExpiryRelay contract.
interface IMessageExpiryRelay is IProxyAdminOwnedBase {
    error MessageExpiryRelay_InvalidExpiryWindow();
    error MessageExpiryRelay_MessageNotSent();
    error MessageExpiryRelay_AlreadyRecorded();
    error MessageExpiryRelay_MessageDelivered();
    error MessageExpiryRelay_InvalidSourceChain();
    error MessageExpiryRelay_NotInitialized();
    error MessageExpiryRelay_InvalidExpirySender();
    error MessageExpiryRelay_MessageNotRecorded();
    error MessageExpiryRelay_WrongAttestor();
    error MessageExpiryRelay_MessageNotExpired();

    event Initialized(uint8 version);
    event SentMessageRecorded(bytes32 indexed msgHash, address indexed app, uint256 destination, uint256 recordedAt);
    event UndeliveredMessageAttested(bytes32 indexed msgHash, uint256 indexed sourceChainId, uint256 attestedAt);
    event ExpiredMessageRelayed(
        bytes32 indexed msgHash, address indexed app, uint256 attestorChainId, uint256 attestedAt
    );

    function version() external view returns (string memory);
    function hub() external view returns (address);
    function expiryWindow() external view returns (uint256);
    function sentMessageRecords(bytes32) external view returns (address app, uint96 recordedAt, uint256 destination);
    function initialize(address _hub, uint256 _expiryWindow) external;
    function recordSentMessage(
        uint256 _destination,
        uint256 _nonce,
        address _target,
        bytes calldata _message
    )
        external
        returns (bytes32 msgHash_);
    function attestUndelivered(bytes32 _msgHash, uint256 _sourceChainId, uint32 _minGasLimit) external;
    function receiveExpiry(bytes32 _msgHash, uint256 _attestorChainId, uint256 _attestedAt) external;

    function __constructor__() external;
}
