// SPDX-License-Identifier: MIT
// @unsupported: ovm
pragma solidity >0.5.0 <0.8.0;

/* Library Imports */
import { Lib_AddressResolver } from "../../libraries/resolver/Lib_AddressResolver.sol";

/* Interface Imports */
import { iOVM_StateTransitioner } from "../../iOVM/verification/iOVM_StateTransitioner.sol";
import { iOVM_StateTransitionerFactory } from "../../iOVM/verification/iOVM_StateTransitionerFactory.sol";
import { iOVM_FraudVerifier } from "../../iOVM/verification/iOVM_FraudVerifier.sol";

/* Contract Imports */
import { OVM_StateTransitioner } from "./OVM_StateTransitioner.sol";

/**
 * @title OVM_StateTransitionerFactory
 * @dev The State Transitioner Factory is used by the Fraud Verifier to create a new State 
 * Transitioner during the initialization of a fraud proof.
 * 
 * Compiler used: solc
 * Runtime target: EVM
 */
contract OVM_StateTransitionerFactory is iOVM_StateTransitionerFactory, Lib_AddressResolver {

    constructor(
        address _libAddressManager
    )
        public
        Lib_AddressResolver(_libAddressManager)
    {}

    /***************************************
     * Public Functions: Contract Creation *
     ***************************************/

    /**
     * Creates a new OVM_StateTransitioner
     * @param _libAddressManager Address of the Address Manager.
     * @param _stateTransitionIndex Index of the state transition being verified.
     * @param _preStateRoot State root before the transition was executed.
     * @param _transactionHash Hash of the executed transaction.
     * @return _ovmStateTransitioner New OVM_StateTransitioner instance.
     */
    function create(
        address _libAddressManager,
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
        require(
            msg.sender == resolve("OVM_FraudVerifier"),
            "Create can only be done by the OVM_FraudVerifier."
        );
        return new OVM_StateTransitioner(
            _libAddressManager,
            _stateTransitionIndex,
            _preStateRoot,
            _transactionHash
        );
    }
}
