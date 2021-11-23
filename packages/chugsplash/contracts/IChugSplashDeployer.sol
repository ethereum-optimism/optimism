// SPDX-License-Identifier: MIT
pragma solidity ^0.8.9;

/**
 * @title IChugSplashDeployer
 */
interface IChugSplashDeployer {
    function isUpgrading() external view returns (bool);
}
