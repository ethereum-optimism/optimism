// SPDX-License-Identifier: MIT
pragma solidity >0.5.0 <0.8.0;

/* Library Imports */
import { Lib_ErrorUtils } from "../../libraries/utils/Lib_ErrorUtils.sol";

/**
 * @title OVM_ExecutionManagerWrapper
 * @dev This contract is a thin wrapper around the `kall` builtin. By making this contract a
 *  predeployed contract, we can restrict evm solc incompatibility to this one contract. Other
 *  contracts will typically call this contract via `Lib_ExecutionManagerWrapper`.
 *
 * Compiler used: optimistic-solc
 * Runtime target: OVM
 */
contract OVM_ExecutionManagerWrapper {

    /*********************
     * Fallback Function *
     *********************/

    fallback()
        external
        payable
    {
        // DO NOTHING FOR NOW
    }
}
