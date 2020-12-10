// SPDX-License-Identifier: MIT
pragma solidity ^0.7.0;

/* Interface Imports */
import { iOVM_StateTransitioner } from "../../iOVM/verification/iOVM_StateTransitioner.sol";
import { iOVM_StateTransitionerFactory } from "../../iOVM/verification/iOVM_StateTransitionerFactory.sol";

/* Contract Imports */
import { OVM_StateTransitioner } from "./OVM_StateTransitioner.sol";

/**
 * @title OVM_StateTransitionerFactory
 */
contract OVM_StateTransitionerFactory is iOVM_StateTransitionerFactory {

    /***************************************
     * Public Functions: Contract Creation *
     ***************************************/

    /**
     * Creates a new OVM_StateTransitioner
     * @param _libAddressManager Address of the Address Manager.
     * @param _preStateRoot State root before the transition was executed.
     * @param _transactionHash Hash of the executed transaction.
     * @return _ovmStateTransitioner New OVM_StateTransitioner instance.
     */
    function create(
        address _libAddressManager,
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
            _libAddressManager,
            _preStateRoot,
            _transactionHash
        );
    }
}
