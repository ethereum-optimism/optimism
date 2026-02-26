// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { ICompliance } from "interfaces/universal/ICompliance.sol";

/// @title IRule
/// @notice Interface for compliance rule contracts. Rules are evaluated by the Compliance
///         contract to determine whether a cross-chain transaction should be allowed.
interface IRule {
    /// @notice Checks a transaction against this rule.
    /// @param _from       The sender of the transaction.
    /// @param _to         The recipient of the transaction.
    /// @param _value      The ETH value of the transaction.
    /// @param _gasLimit   The gas limit for execution.
    /// @param _isCreation Whether the transaction creates a contract.
    /// @param _data       The calldata of the transaction.
    /// @param _nonce      The nonce of the transaction.
    /// @return The compliance status result of this rule's evaluation.
    function check(
        address _from,
        address _to,
        uint256 _value,
        uint64 _gasLimit,
        bool _isCreation,
        bytes calldata _data,
        uint256 _nonce
    )
        external
        returns (ICompliance.Status);
}
