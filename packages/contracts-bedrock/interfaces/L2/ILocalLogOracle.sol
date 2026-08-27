// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { Identifier } from "interfaces/L2/ICrossL2Inbox.sol";

/// @title ILocalLogOracle
/// @notice Execution-client interface for proving that a log exists on the current L2.
/// @dev Implementations must reject logs from another chain, the current block, future blocks,
///      and blocks older than the protocol event lookup window.
interface ILocalLogOracle {
    /// @notice Returns whether the identified log has the supplied payload hash.
    /// @param _id Identifier of the log on the current chain.
    /// @param _payloadHash Hash of the complete encoded log payload.
    function containsLog(Identifier calldata _id, bytes32 _payloadHash) external view returns (bool);
}
