// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { Identifier } from "interfaces/L2/ICrossL2Inbox.sol";
import { ISemver } from "interfaces/universal/ISemver.sol";

interface IProjectionEventExporter is ISemver {
    function __constructor__() external;
    function exportProvenEvent(Identifier calldata _id, bytes32 _payloadHash, bytes calldata _proof) external;
}
