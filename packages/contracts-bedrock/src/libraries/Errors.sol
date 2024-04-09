// SPDX-License-Identifier: MIT
pragma solidity ^0.8.15;

////////////////////////////////////////////////////////////////
//                       Common Errors                        //
////////////////////////////////////////////////////////////////

/// @notice Thrown when an unauthorized address attempts to interact with a contract.
error BadAuth(string expectedRole);
