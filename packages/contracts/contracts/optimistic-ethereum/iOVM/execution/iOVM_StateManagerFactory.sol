// SPDX-License-Identifier: MIT
pragma solidity >0.5.0 <0.8.0;

/* Contract Imports */
import { iOVM_StateManager } from "./iOVM_StateManager.sol";

/**
 * @title iOVM_StateManagerFactory
 */
interface iOVM_StateManagerFactory {

    /***************************************
     * Public Functions: Contract Creation *
     ***************************************/

    function create(
        address _owner
    )
        external
        returns (
            iOVM_StateManager _ovmStateManager
        );
}
