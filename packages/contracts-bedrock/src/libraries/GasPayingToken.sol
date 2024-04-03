// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { Storage } from "src/libraries/Storage.sol";
import { Constants } from "src/libraries/Constants.sol";

/// @title GasPayingToken
/// @notice Handles reading and writing the custom gas token to storage.
library GasPayingToken {
    /// @notice The storage slot where the gas paying token address is stored.
    bytes32 internal constant GAS_PAYING_TOKEN_SLOT = bytes32(uint256(keccak256("opstack.gaspayingtoken")) - 1);

    /// @notice Reads the gas paying token and its decimals from the magic
    ///         storage slot. If nothing is set in storage, then the ether
    ///         address is returned instead.
    function get() internal view returns (address addr_, uint8 decimals_) {
        bytes32 slot = Storage.getBytes32(GAS_PAYING_TOKEN_SLOT);
        addr_ = address(uint160(uint256(slot) & uint256(type(uint160).max)));
        if (addr_ == address(0)) {
            addr_ = Constants.ETHER;
            decimals_ = 18;
        } else {
            decimals_ = uint8(uint256(slot) >> 160);
        }
    }

    /// @notice Writes the gas paying token and its decimals to the magic storage slot.
    function set(address _token, uint8 _decimals) internal {
        Storage.setBytes32(GAS_PAYING_TOKEN_SLOT, bytes32(uint256(_decimals) << 160 | uint256(uint160(_token))));
    }
}
