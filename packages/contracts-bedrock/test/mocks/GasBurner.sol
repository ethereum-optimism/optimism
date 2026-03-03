// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

// Libraries
import { Burn } from "src/libraries/Burn.sol";

/// @title GasBurner
/// @notice Contract that burns a specified amount of gas on receive or fallback.
contract GasBurner {
    /// @notice The amount of gas to burn on receive or fallback.
    uint256 immutable GAS_TO_BURN;

    /// @notice Constructor.
    /// @param _gas The amount of gas to burn on receive or fallback.
    constructor(uint256 _gas) {
        // 500 gas buffer for Solidity overhead.
        GAS_TO_BURN = _gas - 500;
    }

    /// @notice Receive function that burns the specified amount of gas.
    receive() external payable {
        _burn();
    }

    /// @notice Fallback function that burns the specified amount of gas.
    fallback() external payable {
        _burn();
    }

    /// @notice Internal function that burns the specified amount of gas.
    function _burn() internal view {
        Burn.gas(GAS_TO_BURN);
    }
}
