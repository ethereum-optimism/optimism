// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Predeploys } from "src/libraries/Predeploys.sol";
import { ICrossL2Inbox, Identifier } from "interfaces/L2/ICrossL2Inbox.sol";
import { ISemver } from "interfaces/universal/ISemver.sol";

/// @notice Projection-only entry point for proof-carrying deposits. The private genesis leaves
///         this reserved predeploy proxy uninitialized, so the same deposit reverts there.
contract ProjectionEventExporter is ISemver {
    /// @custom:semver 1.0.0
    string public constant version = "1.0.0";

    function exportProvenEvent(Identifier calldata _id, bytes32 _payloadHash, bytes calldata _proof) external {
        ICrossL2Inbox(Predeploys.CROSS_L2_INBOX).exportProvenEvent(_id, _payloadHash, _proof);
    }
}
