// SPDX-License-Identifier: MIT
pragma solidity >0.5.0 <0.8.0;

/* Interface Imports */
import { iOVM_StateManager } from "../../iOVM/execution/iOVM_StateManager.sol";
import { iOVM_StateManagerFactory } from "../../iOVM/execution/iOVM_StateManagerFactory.sol";

/* Contract Imports */
import { OVM_StateManager } from "./OVM_StateManager.sol";

/**
 * @title OVM_StateManagerFactory
 * @dev The State Manager Factory is called by a State Transitioner's init code, to create a new
 * State Manager for use in the Fraud Verification process.
 * 
 * Compiler used: solc
 * Runtime target: EVM
 */
contract OVM_StateManagerFactory is iOVM_StateManagerFactory {

    /********************
     * Public Functions *
     ********************/

    /**
     * Creates a new OVM_StateManager
     * @param _owner Owner of the created contract.
     * @return New OVM_StateManager instance.
     */
    function create(
        address _owner
    )
        override
        public
        returns (
            iOVM_StateManager
        )
    {
        return new OVM_StateManager(_owner);
    }
}
