// SPDX-License-Identifier: MIT
pragma solidity ^0.8.8;

/**
 * @title iL1ChugSplashDeployer
 */
interface iL1ChugSplashDeployer {
    function isUpgrading()
        external
        view
        returns (
            bool
        );
}
