// SPDX-License-Identifier: MIT
pragma solidity >0.5.0 <0.8.0;

/* Inherited Interface Imports */
import { iOVM_SafetyCache } from "../../iOVM/predeploys/iOVM_SafetyCache.sol";

/* External Interface Imports */
import { iOVM_SafetyChecker } from "../../iOVM/execution/iOVM_SafetyChecker.sol";

/**
 * @title OVM_SafetyCache
 * @dev This contract implements a simple registry for caching the hash of any bytecode strings which have
 * already been confirmed safe by the Safety Checker.
 *
 * Compiler used: solc
 * Runtime target: EVM
 */
contract OVM_SafetyCache is iOVM_SafetyCache {


    /*******************************************
     * Contract Variables: Contract References *
     ******************************************/

    iOVM_SafetyChecker internal ovmSafetyChecker = iOVM_SafetyChecker(0x4200000000000000000000000000000000000010);


    /****************************************
     * Contract Variables: Internal Storage *
     ****************************************/

    mapping(bytes32 => bool) internal isSafeCodehash;


    /**********************
     * External Functions *
     *********************/


    /** Checks the registry to see if the verified bytecode is registered as safe. If not, calls to the
    * SafetyChecker to see.
    * @param _code A bytes32 hash of the code
    * @return `true` if the bytecode is safe, `false` otherwise.
    */
    function checkAndRegisterSafeBytecode(
        bytes memory _code
    )
        override
        external
        returns (
            bool
    ) {
        bytes32 codehash = keccak256(_code);
        if(isSafeCodehash[codehash] == true) {
            return true;
        }

        bool safe = ovmSafetyChecker.isBytecodeSafe(_code);
        if(safe) {
            isSafeCodehash[codehash] = true;
        }
        return safe;
    }

    /** Used to check if bytecode has already been recorded as safe.
    * @param _codehash A bytes32 hash of the code
    */
    function isRegisteredSafeBytecode(
        bytes32 _codehash
    )
        override
        external
        view
        returns (
            bool
        )
    {
        return isSafeCodehash[_codehash] == true;
    }
}
