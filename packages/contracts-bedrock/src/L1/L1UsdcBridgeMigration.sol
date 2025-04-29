// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { L1UsdcBridge } from "src/L1/L1UsdcBridge.sol";

interface IHybridLockReleaseUSDCTokenPool {
    function provideLiquidity(uint64 remoteChainSelector, uint256 amount) external;
}

// Upgrade to L1UsdcBridge that allows to migrate all liquidity into the CCIP token pool
contract L1UsdcBridgeMigration is L1UsdcBridge {
    // The address of the CCIP token pool
    address private constant tokenPool = 0xc2e3A3C18ccb634622B57fF119a1C8C7f12e8C0c;

    // The remote chain selector for BOB - TODO: use the right one
    uint64 private constant remoteChainSelector = 1;

    function migrateLiquidity() external onlyOwner {
        // migrate all the liquidity into the token pool
        IHybridLockReleaseUSDCTokenPool(tokenPool).provideLiquidity(remoteChainSelector, address(this).balance);

        // remove the liquidity from this contract by deleting all deposits
        delete deposits[l1Usdc][l2Usdc];

        // pause contract so no more deposits are allowed
        _pause();
    }
}
