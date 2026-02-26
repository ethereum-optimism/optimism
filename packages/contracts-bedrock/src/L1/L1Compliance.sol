// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Contracts
import { Compliance } from "src/universal/Compliance.sol";

// Interfaces
import { IOptimismPortal2 } from "interfaces/L1/IOptimismPortal2.sol";

/// @custom:proxied true
/// @title L1Compliance
/// @notice Concrete compliance implementation for L1. Forwards approved transactions
///         to OptimismPortal2.
contract L1Compliance is Compliance {
    /// @inheritdoc Compliance
    function _executeApproved(
        address _from,
        address _to,
        uint256 _value,
        uint256 _mint,
        uint64 _gasLimit,
        bool _isCreation,
        bytes calldata _data,
        uint256
    )
        internal
        override
    {
        IOptimismPortal2(bridge).approved{ value: _mint }(_from, _to, _value, _mint, _gasLimit, _isCreation, _data);
    }
}
