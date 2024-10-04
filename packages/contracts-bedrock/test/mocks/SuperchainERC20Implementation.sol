// SPDX-License-Identifier: MIT
pragma solidity 0.8.25;

import { SuperchainERC20 } from "src/L2/SuperchainERC20.sol";

/// @title SuperchainERC20Implementation Mock contract
/// @notice Mock contract just to create tests over an implementation of the SuperchainERC20 abstract contract.
contract SuperchainERC20Implementation_MockContract is SuperchainERC20 {
    /// @notice Semantic version.
    /// @custom:semver 1.0.0-beta.1
    string public constant override version = "1.0.0-beta.1";

    function name() public pure override returns (string memory) {
        return "SuperchainERC20";
    }

    function symbol() public pure override returns (string memory) {
        return "SCE";
    }
}
