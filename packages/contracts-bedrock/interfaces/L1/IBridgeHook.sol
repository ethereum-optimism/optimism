// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { BridgeHookTypes } from "src/libraries/BridgeHookTypes.sol";

interface IBridgeHook {
    function screen(BridgeHookTypes.Item calldata _item, uint256 _finalizableAt) external returns (bool pass_);
    function hold(BridgeHookTypes.Item calldata _item) external payable;
}
