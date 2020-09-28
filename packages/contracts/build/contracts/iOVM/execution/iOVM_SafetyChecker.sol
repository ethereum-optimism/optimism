// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.7.0;

/**
 * @title iOVM_SafetyChecker
 */
interface iOVM_SafetyChecker {

    /********************
     * Public Functions *
     ********************/

    function isBytecodeSafe(bytes memory _bytecode) external view returns (bool _safe);
}
