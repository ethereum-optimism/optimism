// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

// Interfaces
import { ICrosschainERC20 } from "src/L2/interfaces/ICrosschainERC20.sol";

/// @title ISuperchainERC20
/// @notice This interface is available on the SuperchainERC20 contract.
interface ISuperchainERC20 is ICrosschainERC20 {
    /// @notice Thrown when attempting to mint or burn tokens and the function caller is not the SuperchainERC20Bridge.
    error OnlySuperchainERC20Bridge();
}
