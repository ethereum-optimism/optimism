// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.7.0;

/* Interface Imports */
import { iOVM_StateManager } from "../../iOVM/execution/iOVM_StateManager.sol";
import { iOVM_StateManagerFactory } from "../../iOVM/execution/iOVM_StateManagerFactory.sol";

/* Contract Imports */
import { OVM_StateManager } from "./OVM_StateManager.sol";

/**
 * @title OVM_StateManagerFactory
 */
contract OVM_StateManagerFactory is iOVM_StateManagerFactory {

    /***************************************
     * Public Functions: Contract Creation *
     ***************************************/

    function create()
        override
        public
        returns (
            iOVM_StateManager _ovmStateManager
        )
    {
        return new OVM_StateManager();
    }
}
