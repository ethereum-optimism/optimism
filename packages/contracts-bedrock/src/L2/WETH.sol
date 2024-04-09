// SPDX-License-Identifier: MIT
pragma solidity >=0.4.22 <0.6;

import { WETH9 } from "src/vendor/WETH9.sol";

/// @title WETH contract that reads the name and symbol from the L1Block contract
contract WETH is WETH9 {
    /// @notice Address of the L1Block contract
    address L1BlockAddr;

    /// @notice Constructor that sets the L1Block address
    constructor(address _L1BlockAddr) WETH9() {
        L1BlockAddr = _L1BlockAddr;
    }

    /// @notice Returns the name of the token from the L1Block contract
    function name() external override returns (string memory) {
        (bool success, bytes memory data) = L1BlockAddr.call(abi.encodeWithSignature("gasPayingToken()"));

        require(success, "L1Block call failed");

        (,, string memory name_,) = abi.decode(data, (address, uint8, string, string));

        return name_;
    }

    /// @notice Returns the symbol of the token from the L1Block contract
    function symbol() external override returns (string memory) {
        (bool success, bytes memory data) = L1BlockAddr.call(abi.encodeWithSignature("gasPayingToken()"));

        require(success, "L1Block call failed");

        (,,, string memory symbol_) = abi.decode(data, (address, uint8, string, string));

        return symbol_;
    }
}
