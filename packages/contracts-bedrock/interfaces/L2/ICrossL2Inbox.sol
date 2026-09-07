// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { IProxyAdminOwnedBase } from "interfaces/universal/IProxyAdminOwnedBase.sol";

/// @notice Identifier of a cross chain message.

struct Identifier {
    address origin;
    uint256 blockNumber;
    uint256 logIndex;
    uint256 timestamp;
    uint256 chainId;
}

interface ICrossL2Inbox is IProxyAdminOwnedBase {
    error CrossL2Inbox_NotProjectionEventExporter();

    error CrossL2Inbox_NoExecutingDeposits();
    error CrossL2Inbox_InvalidEventRegistry();
    error CrossL2Inbox_NotEventRegistry();
    error CrossL2Inbox_EventNotFound();
    error CrossL2Inbox_EventTooOld();
    error CrossL2Inbox_EventFromAnotherChain();
    error CrossL2Inbox_EventNotInPreviousBlock();
    error CrossL2Inbox_InvalidEventProofVerifier();
    error CrossL2Inbox_InvalidEventProof();
    error CrossL2Inbox_EventAlreadyExported();
    error NotInAccessList();
    error BlockNumberTooHigh();
    error TimestampTooHigh();
    error LogIndexTooHigh();

    event ExecutingMessage(bytes32 indexed msgHash, Identifier id);
    event ExecutingCertifiedMessage(bytes32 indexed msgHash, Identifier id);
    event EventExported(bytes32 indexed checksum, bytes32 indexed payloadHash, Identifier id);
    event EventImported(bytes32 indexed checksum, bytes32 indexed payloadHash, Identifier id);
    event L1EventRegistryUpdated(address indexed oldRegistry, address indexed newRegistry);
    event EventProofVerifierUpdated(address indexed oldVerifier, address indexed newVerifier);

    function version() external view returns (string memory);

    function EVENT_LOOKUP_WINDOW() external view returns (uint256);

    function REGISTER_EVENT_GAS_LIMIT() external view returns (uint32);

    function validateMessage(Identifier calldata _id, bytes32 _msgHash) external;

    function exportEvent(Identifier calldata _id, bytes32 _payloadHash) external;
    function exportProvenEvent(Identifier calldata _id, bytes32 _payloadHash, bytes calldata _proof) external;
    function setEventProofVerifier(address _verifier) external;
    function eventProofVerifier() external view returns (address);
    function provenEvents(bytes32) external view returns (bool);

    function importEvent(Identifier calldata _id, bytes32 _payloadHash) external;

    function importAndExecute(Identifier calldata _id, bytes calldata _sentMessage) external;

    function setL1EventRegistry(address _l1EventRegistry) external;

    function l1EventRegistry() external view returns (address);

    function certifiedMessages(bytes32) external view returns (bool);

    function calculateChecksum(Identifier memory _id, bytes32 _msgHash) external pure returns (bytes32 checksum_);
}
