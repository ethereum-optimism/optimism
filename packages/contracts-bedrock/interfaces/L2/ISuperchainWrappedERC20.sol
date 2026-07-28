// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

// Interfaces
import { ISuperchainERC20 } from "interfaces/L2/ISuperchainERC20.sol";

/// @title ISuperchainWrappedERC20
/// @notice Interface for the SuperchainWrappedERC20 contract.
interface ISuperchainWrappedERC20 is ISuperchainERC20 {
    function FACTORY() external view returns (address);

    function ORIGINAL_TOKEN() external view returns (address);

    function ORIGINAL_CHAIN_ID() external view returns (uint256);

    function mint(address _to, uint256 _amount) external;

    function burn(address _from, uint256 _amount) external;

    function __constructor__() external;
}
