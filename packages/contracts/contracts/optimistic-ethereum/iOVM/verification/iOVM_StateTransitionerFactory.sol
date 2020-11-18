// SPDX-License-Identifier: MIT
pragma solidity >0.5.0 <0.8.0;

/* Contract Imports */
import { iOVM_StateTransitioner } from "./iOVM_StateTransitioner.sol";

/**
 * @title iOVM_StateTransitionerFactory
 */
interface iOVM_StateTransitionerFactory {

    /***************************************
     * Public Functions: Contract Creation *
     ***************************************/

    function create(
        address _proxyManager,
        uint256 _stateTransitionIndex,
        bytes32 _preStateRoot,
        bytes32 _transactionHash
    )
        external
        returns (
            iOVM_StateTransitioner _ovmStateTransitioner
        );
}
