// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

// Libraries
import { BridgeHookTypes } from "src/libraries/BridgeHookTypes.sol";
import { Hashing } from "src/libraries/Hashing.sol";
import { Types } from "src/libraries/Types.sol";

/// @title BridgeHookHashing
/// @notice Derives the identifiers of bridge hook items.
library BridgeHookHashing {
    /// @notice Hashes every field to identify a deposit or withdrawal and look up its verdict.
    /// @param _item Item to hash.
    /// @return Identifier of the item.
    function hashItem(BridgeHookTypes.Item memory _item) internal pure returns (bytes32) {
        return keccak256(abi.encode(_item));
    }

    /// @notice Returns the Portal's withdrawal hash for the withdrawal represented by this item.
    /// @param _item Withdrawal item to hash.
    /// @return Withdrawal hash of the item.
    function hashWithdrawalItem(BridgeHookTypes.Item memory _item) internal pure returns (bytes32) {
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
