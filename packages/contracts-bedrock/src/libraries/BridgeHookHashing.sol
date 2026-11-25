// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

// Libraries
import { BridgeHookTypes } from "src/libraries/BridgeHookTypes.sol";

/// @title BridgeHookHashing
/// @notice Derives the identifiers of bridge hook items.
library BridgeHookHashing {
    /// @notice Hashes all item fields to key verdicts and held items.
    /// @param _item Item to hash.
    /// @return Item ID.
    function hashItem(BridgeHookTypes.Item memory _item) internal pure returns (bytes32) {
        return keccak256(abi.encode(_item));
    }
}
