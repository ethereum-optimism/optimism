// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Messenger_Initializer, Reverter, CallerCaller } from "./CommonTest.t.sol";

// CrossDomainMessenger_Test is for testing functionality which is common to both the L1 and L2
// CrossDomainMessenger contracts. For simplicity, we use the L1 Messenger as the test contract.
contract CrossDomainMessenger_Test is Messenger_Initializer {
    // Ensure that baseGas passes for the max value of _minGasLimit,
    // this is about 4 Billion.
    function test_baseGas() external view {
        L1Messenger.baseGas(hex"ff", type(uint32).max);
    }

    // Fuzz for other values which might cause a revert in baseGas.
    function testFuzz_baseGas(uint32 _minGasLimit) external view {
        L1Messenger.baseGas(hex"ff", _minGasLimit);
    }
}
