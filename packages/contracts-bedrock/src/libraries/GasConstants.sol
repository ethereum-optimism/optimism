// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

// Gas budgets for `CrossDomainMessenger.relayMessage`, named for the chain the relay executes on. A
// messenger uses its own chain's values in `relayMessage` and the other chain's values in `baseGas`,
// so each value must cover the worst case on the chain it is named for.
// File-level rather than a library so that contract constants can be initialized from them. Import
// as `import "src/libraries/GasConstants.sol" as GasConstants;`. NatSpec on file-level constants
// accepts only `@dev`.

////////////////////////////////////////////////////////////////
//                     Constant overhead                      //
////////////////////////////////////////////////////////////////

// Covers `relayMessage` from entry up to the `hasMinGas` check: the pause check, version decode,
// message hashing, cold reads of `successfulMessages` and `failedMessages`, the other-messenger
// check (on L1, the portal's `l2Sender()` call) and the target checks. Used only by `baseGas`. Too
// low, and a relay given exactly `baseGas` fails the check and is recorded as failed instead of
// executing.
// To set: measure that path on the chain's EVM with cold storage, and add margin.

/// @dev Set empirically for pre-Glamsterdam L1.
uint64 constant L1_RELAY_CONSTANT_OVERHEAD = 200_000;

/// @dev Set empirically.
uint64 constant L2_RELAY_CONSTANT_OVERHEAD = 200_000;

////////////////////////////////////////////////////////////////
//                       Call overhead                        //
////////////////////////////////////////////////////////////////

// `SafeCall.hasMinGas` on the receiving chain requires 40,000 gas above `minGasLimit * 64 / 63` for
// the CALL's dynamic costs (cold access, value transfer, new account). Used only by `baseGas`, which
// must supply at least that much.
// To set: at least the allowance hardcoded in `SafeCall.hasMinGas`. Change them together.

/// @dev Does not cover EIP-8037's 183,600 state gas for sending value to a new account.
uint64 constant L1_RELAY_CALL_OVERHEAD = 40_000;

/// @dev Matches `SafeCall.hasMinGas`.
uint64 constant L2_RELAY_CALL_OVERHEAD = 40_000;

////////////////////////////////////////////////////////////////
//                        Reserved gas                        //
////////////////////////////////////////////////////////////////

// `relayMessage` forwards `gasleft()` minus this to the target, so it covers the CALL's own charges
// (value transfer, cold target access, EIP-7702 delegate access) and everything after the call:
// resetting `xDomainMsgSender`, writing `successfulMessages` or `failedMessages` (a fresh slot) and
// emitting the event. Too low, and a target that uses all its gas makes the relay revert after the
// call, so the message is recorded in neither mapping and cannot be replayed.
// To set: on the chain's EVM, find the smallest reserve that still records a value-carrying message
// to a cold EIP-7702 delegated target that burns all forwarded gas, and add margin.
// `test_relayMessage_valueToColdDelegatedTarget_succeeds` covers this case on L1.

/// @dev Includes EIP-8037 state gas for the fresh slot (97,920). Measured minimum under Amsterdam:
///      127,089.
uint64 constant L1_RELAY_RESERVED_GAS = 135_000;

/// @dev Measured minimum under Prague: 35,969.
uint64 constant L2_RELAY_RESERVED_GAS = 40_000;

////////////////////////////////////////////////////////////////
//                      Gas check buffer                      //
////////////////////////////////////////////////////////////////

// Covers the work between the `hasMinGas` check and the CALL: setting `xDomainMsgSender` (a warm,
// nonzero-to-nonzero write, so no state gas) and preparing the call's arguments. Too low, and the
// target can receive less than `minGasLimit`.
// To set: the cost of that window on the chain's EVM, rounded up generously.

/// @dev Conservative round number.
uint64 constant L1_RELAY_GAS_CHECK_BUFFER = 5_000;

/// @dev Conservative round number.
uint64 constant L2_RELAY_GAS_CHECK_BUFFER = 5_000;
