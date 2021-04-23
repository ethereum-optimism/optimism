// SPDX-License-Identifier: MIT
// @unsupported: evm
pragma solidity >0.5.0 <0.8.0;

/* Library Imports */
import { Lib_ExecutionManagerWrapper } from "../../libraries/wrappers/Lib_ExecutionManagerWrapper.sol";

/**
 * @title OVM_UpgradeExecutor
 * @dev The OVM_UpgradeExecutor is the contract which authenticates and executes (i.e.
 * calls the relevant Execution Manager upgrade functions) upgrades to the OVM State.
 * This enables us to update the predeploy and execution contracts directly from within
 * L2 when there is a new release.
 * 
 * Compiler used: optimistic-solc
 * Runtime target: OVM
 */
contract OVM_UpgradeExecutor {
    function setCode(
        address _address,
        bytes memory _code
    )
        external
    {
        Lib_ExecutionManagerWrapper.ovmSETCODE(
            _address,
            _code
        );
    }

    function setStorage(
        address _address,
        bytes32 _key,
        bytes32 _value
    )
        external
    {
        Lib_ExecutionManagerWrapper.ovmSETSTORAGE(
            _address,
            _key,
            _value
        );
    }
}
