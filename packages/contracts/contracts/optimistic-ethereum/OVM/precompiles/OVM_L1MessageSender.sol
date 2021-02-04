// SPDX-License-Identifier: MIT
pragma solidity >0.5.0 <0.8.0;

/* Interface Imports */
import { iOVM_L1MessageSender } from "../../iOVM/precompiles/iOVM_L1MessageSender.sol";
import { iOVM_ExecutionManager } from "../../iOVM/execution/iOVM_ExecutionManager.sol";

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
contract OVM_L1MessageSender is iOVM_L1MessageSender {

    /********************
     * Public Functions *
     ********************/

    /**
     * @return _l1MessageSender L1 message sender address (msg.sender).
     */
    function getL1MessageSender()
        override
        public
        view
        returns (
            address _l1MessageSender
        )
    {
        // Note that on L2 msg.sender (ie. evmCALLER) will always be the Execution Manager 
        return iOVM_ExecutionManager(msg.sender).ovmL1TXORIGIN();
    }
}
