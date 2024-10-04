// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

// Interfaces
import { ISuperchainERC20 } from "src/L2/interfaces/ISuperchainERC20.sol";

/// @title IOptimismSuperchainERC20
/// @notice This interface is available on the OptimismSuperchainERC20 contract.
interface IOptimismSuperchainERC20 is ISuperchainERC20 {
    error ZeroAddress();
    error OnlyL2StandardBridge();

    event Mint(address indexed to, uint256 amount);

    event Burn(address indexed from, uint256 amount);

    function mint(address _to, uint256 _amount) external;

    function burn(address _from, uint256 _amount) external;

    function remoteToken() external view returns (address);

    function __constructor__() external;
}
