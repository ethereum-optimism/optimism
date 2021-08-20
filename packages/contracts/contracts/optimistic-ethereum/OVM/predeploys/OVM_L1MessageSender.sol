// SPDX-License-Identifier: MIT
// @unsupported: ovm
pragma solidity >0.5.0 <0.8.0;

/**
 * @title OVM_L1MessageSender
 * @dev The L1MessageSender is a predeploy contract running on L2. During the execution of cross
 * domain transaction from L1 to L2, it returns the address of the L1 account (either an EOA or
 * contract) which sent the message to L2 via the Canonical Transaction Chain's `enqueue()`
 * function.
 *
 * This contract exclusively serves as a getter for the ovmL1TXORIGIN operation. This is necessary
 * because there is no corresponding operation in the EVM which the the optimistic solidity compiler
 * can be replaced with a call to the ExecutionManager's ovmL1TXORIGIN() function.
 *
 *
 * Compiler used: solc
 * Runtime target: OVM
 */
contract OVM_L1MessageSender {
    constructor() {
        // By using the low-level assembly `return` we can dictate the final code of this contract
        // directly. Any call to this contract will simply return the L1MessageSender address.
        // Code of this contract will be:
        // 4A - L1MESSAGESENDER
        // 60 - PUSH1
        // 00
        // 52 - MSTORE (store L1MESSAGESENDER at memory location 0x00)
        // 60 - PUSH1
        // 20
        // 60 - PUSH1
        // 00
        // F3 - RETURN (return memory at 0x00...0x20)
        bytes memory code = hex"4A60005260206000F3";
        assembly {
            return(add(code, 0x20), mload(code))
        }
    }
}
