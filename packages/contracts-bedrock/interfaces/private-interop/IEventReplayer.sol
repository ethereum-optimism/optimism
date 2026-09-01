// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { ISemver } from "interfaces/universal/ISemver.sol";

/// @title IEventReplayer
/// @notice Interface for the EventReplayer contract.
interface IEventReplayer is ISemver {
    error EventReplayer_TooManyTopics();

    function replayEvent(bytes32[] calldata _topics, bytes calldata _data) external;

    function __constructor__() external;
}
