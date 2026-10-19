// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { BridgeHookTypes } from "src/libraries/BridgeHookTypes.sol";

interface IPolicy {
    function screen(BridgeHookTypes.Item calldata _item, address[] calldata _parties) external returns (bool pass_);
    function screenRelease(
        BridgeHookTypes.Item calldata _item,
        address[] calldata _parties,
        uint64 _clearedAt,
        uint256 _finalizableAt
    )
        external
        returns (bool pass_);
}
