// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

// Libraries
import { BridgeHookTypes } from "src/libraries/BridgeHookTypes.sol";
import { Hashing } from "src/libraries/Hashing.sol";
import { Types } from "src/libraries/Types.sol";

/// @title BridgeHookHashing
/// @notice Derives the identifiers of bridge hook items.
library BridgeHookHashing {
    /// @notice Hashes all item fields to key verdicts and held deposits.
    /// @param _item Item to hash.
    /// @return Item ID.
    function hashItem(BridgeHookTypes.Item memory _item) internal pure returns (bytes32) {
        return keccak256(abi.encode(_item));
    }

    /// @notice Computes the Portal's withdrawal hash to key held withdrawals.
    ///         Panics if the item is not a withdrawal.
    /// @param _item Withdrawal item.
    /// @return Withdrawal transaction hash.
    function hashWithdrawalItem(BridgeHookTypes.Item memory _item) internal pure returns (bytes32) {
        assert(_item.direction == BridgeHookTypes.Direction.Withdrawal);
        return Hashing.hashWithdrawal(
            Types.WithdrawalTransaction({
                nonce: _item.nonce,
                sender: _item.from,
                target: _item.to,
                value: _item.ethAmount,
                gasLimit: _item.gasLimit,
                data: _item.data
            })
        );
    }
}
