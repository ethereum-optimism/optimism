// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { Identifier } from "interfaces/L2/ICrossL2Inbox.sol";

/// @notice Successful results are authenticated by their initiating log. A false success flag is
///         an unauthenticated abort hint that forces the entire root transaction to revert.
struct AtomicResultWitness {
    Identifier identifier;
    bool success;
    bytes returnData;
}

/// @notice One operation authenticated by the root chain's request log.
struct AtomicRemoteCall {
    Identifier identifier;
    uint256 sequence;
    address sender;
    address target;
    bytes data;
}

/// @notice Describes a result lookup and the application call waiting for that result.
struct AtomicWitnessRequest {
    uint256 sequence;
    uint256 chainId;
    address target;
    address sender;
    bytes data;
}

/// @notice Describes a remote tape lookup and carries the preceding operation's return data.
struct AtomicStreamCursor {
    uint256 index;
    bytes previousResult;
}
