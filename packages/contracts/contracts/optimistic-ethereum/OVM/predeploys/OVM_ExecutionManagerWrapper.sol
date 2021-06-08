// SPDX-License-Identifier: MIT
// @unsupported: evm
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
        bytes memory data = msg.data;
        assembly {
            // kall is a custom yul builtin within optimistic-solc that allows us to directly call
            // the execution manager (since `call` would be compiled).
            kall(add(data, 0x20), mload(data), 0x0, 0x0)

            // Standard returndata loading code.
            let size := returndatasize()
            let returndata := mload(0x40)
            mstore(0x40, add(returndata, and(add(add(size, 0x20), 0x1f), not(0x1f))))
            mstore(returndata, size)
            returndatacopy(add(returndata, 0x20), 0x0, size)

            // kall automatically reverts if the underlying call fails, so we only need to handle
            // the success case.
            return(add(returndata, 0x20), mload(returndata))
        }
    }
}
