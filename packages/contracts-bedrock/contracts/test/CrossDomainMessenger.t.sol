// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Messenger_Initializer } from "./CommonTest.t.sol";

// CrossDomainMessenger_Test is for testing functionality which is common to both the L1 and L2
// CrossDomainMessenger contracts. For simplicity, we use the L1 Messenger as the test contract.
contract CrossDomainMessenger_Test is Messenger_Initializer {
    function test_baseGas_succeeds() external {
        L1Messenger.baseGas(hex"ff", 100);
    }

    // A test indicating the maximum supported minGasLimit value
    function test_baseGas_revertOverflow() external {
        // passes
        uint32 maxInput = 4_227_133_165;
        L1Messenger.baseGas(hex"", maxInput);

        // overflow
        vm.expectRevert("CrossDomainMessenger: overflow in baseGas calculation");
        L1Messenger.baseGas(hex"", maxInput + 1);
    }

    // A test indicating the maximum supported message length
    function test_baseGas_maxMessageLength() external {
        uint256 maxLength = 16_777_056;
        bytes memory maxMessage = new bytes(maxLength);
        L1Messenger.baseGas(maxMessage, 0);

        // This due to the full test exceeding forge's gas limit of 9_223_372_036_854_754_743
        // From this we can conclude that during normal operation with a block
        // gasLimit closer to 30MM, it is not possible to submit a long enough message
        // to trigger an overflow of the uint64 math in baseGas()
        vm.expectRevert();
        bytes memory oversizedMessage = new bytes(maxLength + 1);
        L1Messenger.baseGas(oversizedMessage, 0);
    }
}
