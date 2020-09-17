// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.7.0;

/* Interface Imports */
import { iOVM_StateTransitioner } from "../../iOVM/execution/iOVM_StateTransitioner.sol";
import { iOVM_StateTransitionerFactory } from "../../iOVM/execution/iOVM_StateTransitionerFactory.sol";

/* Contract Imports */
import { OVM_StateTransitioner } from "./OVM_StateTransitioner.sol";

/**
 * @title OVM_StateTransitionerFactory
 */
contract OVM_StateTransitionerFactory is iOVM_StateTransitionerFactory {

    /***************************************
     * Public Functions: Contract Creation *
     ***************************************/

    function create(
        address _libContractProxyManager,
        uint256 _stateTransitionIndex,
        bytes32 _preStateRoot,
        bytes32 _transactionHash
    )
        override
        public
        returns (
            iOVM_StateTransitioner _ovmStateTransitioner
        )
    {
        return new OVM_StateTransitioner(
            _libContractProxyManager,
            _stateTransitionIndex,
            _preStateRoot,
            _transactionHash
        );
    }
}
