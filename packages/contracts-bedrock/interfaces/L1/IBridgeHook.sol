// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { BridgeHookTypes } from "src/libraries/BridgeHookTypes.sol";

interface IBridgeHook {
    function screenDeposit(BridgeHookTypes.Item calldata _item) external returns (bool pass_);
    function holdDeposit(BridgeHookTypes.Item calldata _item) external payable;
    function screenWithdrawal(
        BridgeHookTypes.Item calldata _item,
        uint256 _finalizableAt
    )
        external
        returns (bool pass_);
    function holdWithdrawal(BridgeHookTypes.Item calldata _item) external payable;
}
