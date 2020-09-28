// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.7.0;

/* Interface Imports */
import { iOVM_SafetyChecker } from "../../iOVM/execution/iOVM_SafetyChecker.sol";

/**
 * @title OVM_SafetyChecker
 */
contract OVM_SafetyChecker is iOVM_SafetyChecker {

    /********************
     * Public Functions *
     ********************/

    /**
     * Checks that a given bytecode string is considered safe.
     * @param _bytecode Bytecode string to check.
     * @return _safe Whether or not the bytecode is safe.
     */
    function isBytecodeSafe(
        bytes memory _bytecode
    )
        override
        public
        view
        returns (
            bool _safe
        )
    {
        return true;
    }
}
