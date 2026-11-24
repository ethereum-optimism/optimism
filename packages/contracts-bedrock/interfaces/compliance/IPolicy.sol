// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { BridgeHookTypes } from "src/libraries/BridgeHookTypes.sol";

interface IPolicy {
    struct Parties {
        address sender;
        address recipient;
    }

    function screenDeposit(BridgeHookTypes.Item calldata _item, Parties calldata _parties) external returns (bool pass_);
    function screenWithdrawal(
        BridgeHookTypes.Item calldata _item,
        Parties calldata _parties,
        uint64 _expiresAt,
        uint256 _finalizableAt
    )
        external
        returns (bool pass_);
    function screenRelease(
        BridgeHookTypes.Item calldata _item,
        Parties calldata _parties,
        uint64 _expiresAt
    )
        external
        returns (bool pass_);
}
