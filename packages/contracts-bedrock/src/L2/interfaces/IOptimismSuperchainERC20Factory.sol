// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { IOptimismERC20Factory } from "./IOptimismERC20Factory.sol";

/// @title IOptimismSuperchainERC20Factory
/// @notice Interface for OptimismSuperchainERC20Factory.
interface IOptimismSuperchainERC20Factory is IOptimismERC20Factory {
    /// @notice Deploys a OptimismSuperchainERC20 Beacon Proxy using CREATE3.
    /// @param _remoteToken      Address of the remote token.
    /// @param _name             Name of the OptimismSuperchainERC20.
    /// @param _symbol           Symbol of the OptimismSuperchainERC20.
    /// @param _decimals         Decimals of the OptimismSuperchainERC20.
    /// @return _superchainERC20 Address of the OptimismSuperchainERC20 deployment.
    function deploy(
        address _remoteToken,
        string memory _name,
        string memory _symbol,
        uint8 _decimals
    )
        external
        returns (address _superchainERC20);
}
