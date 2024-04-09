// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { Storage } from "src/libraries/Storage.sol";
import { Constants } from "src/libraries/Constants.sol";
import { Bytes } from "src/libraries/Bytes.sol";

/// @title GasPayingToken
/// @notice Handles reading and writing the custom gas token to storage.
///         To be used in any place where gas token information is read or
///         written to state. If multiple contracts use this library, the
///         values in storage should be kept in sync between them.
library GasPayingToken {
    /// @notice The storage slot that contains the address and decimals of the gas paying token
    bytes32 internal constant GAS_PAYING_TOKEN_SLOT = bytes32(uint256(keccak256("opstack.gaspayingtoken")) - 1);

    /// @notice The storage slot that contains the ERC20 `name()` of the gas paying token
    bytes32 internal constant GAS_PAYING_TOKEN_NAME_SLOT = bytes32(uint256(keccak256("opstack.gaspayingtokenname")) - 1);

    /// @notice the storage slot that contains the ERC20 `symbol()` of the gas paying token
    bytes32 internal constant GAS_PAYING_TOKEN_SYMBOL_SLOT =
        bytes32(uint256(keccak256("opstack.gaspayingtokensymbol")) - 1);

    /// @notice Reads the gas paying token and its decimals from the magic
    ///         storage slot. If nothing is set in storage, then the ether
    ///         address is returned instead.
    function getToken() internal view returns (address addr_, uint8 decimals_) {
        bytes32 slot = Storage.getBytes32(GAS_PAYING_TOKEN_SLOT);
        addr_ = address(uint160(uint256(slot) & uint256(type(uint160).max)));
        if (addr_ == address(0)) {
            addr_ = Constants.ETHER;
            decimals_ = 18;
        } else {
            decimals_ = uint8(uint256(slot) >> 160);
        }
    }

    /// @notice Reads the gas paying token's name from the magic storage slot.
    ///         If nothing is set in storage, then the ether name, 'Ether', is returned instead.
    function getName() internal view returns (string memory name_) {
        (address addr,) = getToken();
        if (addr == Constants.ETHER) {
            name_ = "Ether";
        } else {
            name_ = string(abi.encodePacked(Storage.getBytes32(GAS_PAYING_TOKEN_NAME_SLOT)));
        }
    }

    /// @notice Reads the gas paying token's symbol from the magic storage slot.
    ///         If nothing is set in storage, then the ether symbol, 'ETH', is returned instead.
    function getSymbol() internal view returns (string memory symbol_) {
        (address addr,) = getToken();
        if (addr == Constants.ETHER) {
            symbol_ = "ETH";
        } else {
            symbol_ = string(abi.encodePacked(Storage.getBytes32(GAS_PAYING_TOKEN_SYMBOL_SLOT)));
        }
    }

    /// @notice Writes the gas paying token, its decimals, name and symbol to the magic storage slot.
    function set(address _token, uint8 _decimals, bytes32 _name, bytes32 _symbol) internal {
        Storage.setBytes32(GAS_PAYING_TOKEN_SLOT, bytes32(uint256(_decimals) << 160 | uint256(uint160(_token))));
        Storage.setBytes32(GAS_PAYING_TOKEN_NAME_SLOT, _name);
        Storage.setBytes32(GAS_PAYING_TOKEN_SYMBOL_SLOT, _symbol);
    }

    /// @notice Maps a string to a bytes32 without leading or trailing zeroes.
    function sanitize(string memory _str) internal pure returns (bytes32 _output) {
        uint256 len = bytes(_str).length;
        require(len <= 32, "GasPayingToken: string cannot be greater than 32 bytes");
        assembly {
            _output := mload(add(_str, 0x20))
        }
        _output = (_output >> 32 - len) << 32 - len;
    }
}
