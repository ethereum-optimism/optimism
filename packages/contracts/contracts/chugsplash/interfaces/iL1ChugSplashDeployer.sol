// SPDX-License-Identifier: MIT
pragma solidity ^0.8.4;

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
