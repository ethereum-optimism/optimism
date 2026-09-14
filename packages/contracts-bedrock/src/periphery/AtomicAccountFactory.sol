// SPDX-License-Identifier: GPL-3.0
pragma solidity 0.8.25;

import { SimpleAccountFactory } from "@account-abstraction/samples/SimpleAccountFactory.sol";
import { IEntryPoint } from "@account-abstraction/interfaces/IEntryPoint.sol";
import { Preinstalls } from "src/libraries/Preinstalls.sol";

/// @notice CREATE2 factory for upstream ERC-4337 accounts using the OP Stack's existing EntryPoint v0.7.
///         Accounts are provisioned before atomic discovery; initCode execution is not part of the demo.
contract AtomicAccountFactory is SimpleAccountFactory {
    /// @custom:semver 0.1.0
    string public constant version = "0.1.0";

    constructor() SimpleAccountFactory(IEntryPoint(Preinstalls.EntryPoint_v070)) { }
}
