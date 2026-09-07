// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { Identifier } from "interfaces/L2/ICrossL2Inbox.sol";

/// @notice Pluggable authentication boundary for events exported by CrossL2Inbox.
interface IEventProofVerifier {
    /// @notice Accepts an event in the caller's canonical source history, or returns false/reverts.
    ///         Implementations must authenticate the source inbox and domain, and bind every field
    ///         of the identifier and payload hash. Any required state-effect consumption belongs
    ///         here and is atomic with export. This interface does not imply private execution
    ///         validity: that guarantee depends on the configured verifier.
    function verifyAndConsumeEvent(
        Identifier calldata _id,
        bytes32 _payloadHash,
        bytes calldata _proof
    )
        external
        returns (bool);
}
