// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { IOptimismERC20Factory } from "./IOptimismERC20Factory.sol";
import { ISemver } from "src/universal/interfaces/ISemver.sol";

/// @title IOptimismSuperchainERC20Factory
/// @notice Interface for OptimismSuperchainERC20Factory.
interface IOptimismSuperchainERC20Factory is IOptimismERC20Factory, ISemver {
    event OptimismSuperchainERC20Created(
        address indexed superchainToken, address indexed remoteToken, address deployer
    );

    function deploy(
        address _remoteToken,
        string memory _name,
        string memory _symbol,
        uint8 _decimals
    )
        external
        returns (address superchainERC20_);

    function __constructor__() external;
}
