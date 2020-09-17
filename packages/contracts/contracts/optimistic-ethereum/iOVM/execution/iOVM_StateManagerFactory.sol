// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.7.0;

/* Contract Imports */
import { iOVM_StateManager } from "./iOVM_StateManager.sol";

/**
 * @title iOVM_StateManagerFactory
 */
interface iOVM_StateManagerFactory {

    /***************************************
     * Public Functions: Contract Creation *
     ***************************************/

    function create()
        external
        returns (
            iOVM_StateManager _ovmStateManager
        );
}
