// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

/// @title IUnorderedExecutionModule
/// @notice Interface used by the LivenessGuard to read the signers of an in-flight unordered execution.
interface IUnorderedExecutionModule {
    /// @notice Returns the owners whose signatures were validated for the current module execution.
    /// @dev The implementation must populate this value only after Safe.checkSignatures succeeds and clear it after
    ///      execTransactionFromModule returns. The value must be scoped to the Safe currently executing the module call.
    function signers() external view returns (address[] memory signers_);
}
