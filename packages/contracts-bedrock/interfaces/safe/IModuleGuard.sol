// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

// Safe
import { Enum } from "safe-contracts/common/Enum.sol";
import { IERC165 } from "safe-contracts/interfaces/IERC165.sol";

/// @title IModuleGuard
/// @notice Safe v1.5.0 module guard interface (interfaceId 0x58401ed8). Vendored from
///         safe-global/safe-smart-account v1.5.0 (contracts/base/ModuleManager.sol) because the
///         local safe-contracts dependency does not include module-guard support. Safe v1.5.0's
///         setModuleGuard() requires the guard to report support for this interface id via
///         ERC-165.
interface IModuleGuard is IERC165 {
    /// @notice Called by the Safe (>= v1.5.0) before a module transaction is executed.
    /// @param _to Destination address of the module transaction.
    /// @param _value Ether value of the module transaction.
    /// @param _data Data payload of the module transaction.
    /// @param _operation Operation type of the module transaction.
    /// @param _module Module executing the transaction.
    /// @return moduleTxHash_ Hash of the module transaction, echoed back into
    ///         checkAfterModuleExecution.
    function checkModuleTransaction(
        address _to,
        uint256 _value,
        bytes memory _data,
        Enum.Operation _operation,
        address _module
    )
        external
        returns (bytes32 moduleTxHash_);

    /// @notice Called by the Safe (>= v1.5.0) after a module transaction is executed.
    /// @param _txHash Hash of the module transaction returned by checkModuleTransaction.
    /// @param _success Whether the module transaction succeeded.
    function checkAfterModuleExecution(bytes32 _txHash, bool _success) external;
}
