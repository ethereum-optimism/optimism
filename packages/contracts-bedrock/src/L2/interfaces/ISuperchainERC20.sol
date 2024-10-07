// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

// Interfaces
import { ICrosschainERC20 } from "src/L2/interfaces/ICrosschainERC20.sol";
import { IERC20Solady } from "src/vendor/interfaces/IERC20Solady.sol";
import { ISemver } from "src/universal/interfaces/ISemver.sol";

/// @title ISuperchainERC20
/// @notice This interface is available on the SuperchainERC20 contract.
interface ISuperchainERC20 is ICrosschainERC20, IERC20Solady, ISemver {
    error OnlySuperchainTokenBridge();

    function __constructor__() external;
}
