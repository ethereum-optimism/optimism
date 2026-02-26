// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Contracts
import { Compliance } from "src/universal/Compliance.sol";

// Interfaces
import { IL2ToL1MessagePasser } from "interfaces/L2/IL2ToL1MessagePasser.sol";

/// @custom:proxied true
/// @title L2Compliance
/// @notice Concrete compliance implementation for L2. Forwards approved transactions
///         to L2ToL1MessagePasser.
contract L2Compliance is Compliance {
    /// @inheritdoc Compliance
    function _executeApproved(
        address _from,
        address _to,
        uint256 _value,
        uint256 _mint,
        uint64 _gasLimit,
        bool,
        bytes calldata _data,
        uint256 _nonce
    )
        internal
        override
    {
        IL2ToL1MessagePasser(bridge).approved{ value: _mint }(_from, _to, _value, _gasLimit, _data, _nonce);
    }
}
