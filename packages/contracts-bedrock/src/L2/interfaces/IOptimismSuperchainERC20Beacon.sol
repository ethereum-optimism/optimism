// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { IBeacon } from "@openzeppelin/contracts/proxy/beacon/IBeacon.sol";
import { ISemver } from "src/universal/interfaces/ISemver.sol";

/// @title IOptimismSuperchainERC20Beacon
/// @notice Interface for the OptimismSuperchainERC20Beacon contract
interface IOptimismSuperchainERC20Beacon is IBeacon, ISemver {
    function version() external view returns (string memory);
    function implementation() external view override returns (address);

    function __constructor__(address _implementation) external;
}
