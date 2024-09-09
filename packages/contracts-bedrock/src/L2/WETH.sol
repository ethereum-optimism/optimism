// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { WETH98 } from "src/dispute/weth/WETH98.sol";
import { Predeploys } from "src/libraries/Predeploys.sol";
import { L1Block } from "src/L2/L1Block.sol";
import { ISemver } from "src/universal/interfaces/ISemver.sol";

/// @title WETH contract that reads the name and symbol from the L1Block contract.
///        Allows for nice rendering of token names for chains using custom gas token.
contract WETH is WETH98, ISemver {
    /// @custom:semver 1.0.0-beta.1
    string public constant version = "1.0.0-beta.1";

    /// @notice Returns the name of the wrapped native asset. Will be "Wrapped Ether"
    ///         if the native asset is Ether.
    function name() external view override returns (string memory name_) {
        name_ = string.concat("Wrapped ", L1Block(Predeploys.L1_BLOCK_ATTRIBUTES).gasPayingTokenName());
    }

    /// @notice Returns the symbol of the wrapped native asset. Will be "WETH" if the
    ///         native asset is Ether.
    function symbol() external view override returns (string memory symbol_) {
        symbol_ = string.concat("W", L1Block(Predeploys.L1_BLOCK_ATTRIBUTES).gasPayingTokenSymbol());
    }
}
