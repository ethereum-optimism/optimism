// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { Enum } from "safe-contracts/common/Enum.sol";

/// @notice Parameters for the Safe's execTransaction function
struct ExecTransactionParams {
    address to;
    uint256 value;
    bytes data;
    Enum.Operation operation;
    uint256 safeTxGas;
    uint256 baseGas;
    uint256 gasPrice;
    address gasToken;
    address payable refundReceiver;
    // TODO: Life might be easier if this was left out of the struct
    bytes signatures;
}
