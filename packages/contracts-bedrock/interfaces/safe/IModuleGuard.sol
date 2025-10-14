// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { Enum } from "safe-contracts/common/Enum.sol";
import { IERC165 } from "safe-contracts/interfaces/IERC165.sol";

/**
 * @title IModuleGuard Interface
 */
interface IModuleGuard is IERC165 {
    /**
     * @notice Checks the module transaction details.
     * @dev The function needs to implement module transaction validation logic.
     * @param to The address to which the transaction is intended.
     * @param value The value of the transaction in Wei.
     * @param data The transaction data.
     * @param operation Operation type (0 for `CALL`, 1 for `DELEGATECALL`) of the module transaction.
     * @param module The module involved in the transaction.
     * @return moduleTxHash A guard-specific module transaction hash. This value is passed to the matching {checkAfterModuleExecution} call.
     */
    function checkModuleTransaction(
        address to,
        uint256 value,
        bytes memory data,
        Enum.Operation operation,
        address module
    ) external returns (bytes32 moduleTxHash);

    /**
     * @notice Checks after execution of module transaction.
     * @dev The function needs to implement a check after the execution of the module transaction.
     * @param txHash The guard-specific module transaction hash returned from the matching {checkModuleTransaction} call.
     * @param success The status of the module transaction execution.
     */
    function checkAfterModuleExecution(bytes32 txHash, bool success) external;
}
