// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Semver } from "../universal/Semver.sol";
import { FeeVault } from "../universal/FeeVault.sol";

/**
 * @custom:proxied
 * @custom:predeploy 0x420000000000000000000000000000000000001A
 * @title L1FeeVault
 * @notice The L1FeeVault accumulates the L1 portion of the transaction fees.
 */
contract L1FeeVault is FeeVault, Semver {
    /**
     * @custom:semver 1.2.0
     *
     * @param _recipient           Wallet that will receive the fees.
     * @param _minWithdrawalAmount Minimum balance for withdrawals.
     * @param _withdrawalNetwork   Network which the recipient will receive fees on.
     */
    constructor(
        address _recipient,
        uint256 _minWithdrawalAmount,
        WithdrawalNetwork _withdrawalNetwork
    ) FeeVault(_recipient, _minWithdrawalAmount, _withdrawalNetwork) Semver(1, 2, 0) {}
}
