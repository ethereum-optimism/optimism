// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { Identifier } from "interfaces/L2/ICrossL2Inbox.sol";
import { IETHLockbox } from "interfaces/L1/IETHLockbox.sol";
import { IOptimismPortal2 } from "interfaces/L1/IOptimismPortal2.sol";
import { ISemver } from "interfaces/universal/ISemver.sol";

/// @title IL1EventRegistry
/// @notice L1 registry for event certificates exported through an OptimismPortal withdrawal.
interface IL1EventRegistry is ISemver {
    error L1EventRegistry_InvalidLockbox();
    error L1EventRegistry_UnauthorizedPortal();
    error L1EventRegistry_UnauthorizedL2Sender();
    error L1EventRegistry_WrongSourceChain();
    error L1EventRegistry_EventNotRegistered();

    event EventRegistered(bytes32 indexed certificate, bytes32 indexed payloadHash, Identifier id);
    event EventRelayed(bytes32 indexed certificate, address indexed portal, bool executeMessage);

    function ethLockbox() external view returns (IETHLockbox);

    function registeredEvents(bytes32) external view returns (bool);

    function calculateCertificate(Identifier memory _id, bytes32 _payloadHash) external pure returns (bytes32);

    function registerEvent(Identifier calldata _id, bytes32 _payloadHash) external;

    function relayEvent(
        IOptimismPortal2 _destinationPortal,
        Identifier calldata _id,
        bytes32 _payloadHash,
        uint64 _gasLimit
    )
        external;

    function relayMessage(
        IOptimismPortal2 _destinationPortal,
        Identifier calldata _id,
        bytes calldata _sentMessage,
        uint64 _gasLimit
    )
        external;

    function __constructor__(IETHLockbox _ethLockbox) external;
}
