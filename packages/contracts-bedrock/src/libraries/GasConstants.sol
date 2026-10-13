// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

// Gas budgets for `CrossDomainMessenger.relayMessage`, keyed by the chain the relay executes on.
// File-level rather than a library so that contract constants can be initialized from them.
// Import as `import "src/libraries/GasConstants.sol" as GasConstants;`.

// Constant costs of `relayMessage` on L1, added to the base gas for a message.
uint64 constant L1_RELAY_CONSTANT_OVERHEAD = 200_000;

// Gas L1 needs for the dynamic parts of the CALL opcode in `relayMessage`.
uint64 constant L1_RELAY_CALL_OVERHEAD = 40_000;

// Gas reserved on L1 for the safe call's own charges and for finalizing `relayMessage`
// after it. Covers the value transfer, cold target and EIP-7702 delegate accesses, and
// recording the outcome in a fresh storage slot, including EIP-8037 state gas.
uint64 constant L1_RELAY_RESERVED_GAS = 135_000;

// Gas reserved on L1 for the execution between the `hasMinGas` check and the
// external call in `relayMessage`.
uint64 constant L1_RELAY_GAS_CHECK_BUFFER = 5_000;

// Constant costs of `relayMessage` on L2, added to the base gas for a message.
uint64 constant L2_RELAY_CONSTANT_OVERHEAD = 200_000;

// Gas L2 needs for the dynamic parts of the CALL opcode in `relayMessage`.
uint64 constant L2_RELAY_CALL_OVERHEAD = 40_000;

// Gas reserved on L2 for finalizing `relayMessage` after the safe call.
uint64 constant L2_RELAY_RESERVED_GAS = 40_000;

// Gas reserved on L2 for the execution between the `hasMinGas` check and the
// external call in `relayMessage`.
uint64 constant L2_RELAY_GAS_CHECK_BUFFER = 5_000;
