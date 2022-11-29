// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Semver } from "../universal/Semver.sol";
import { L2StandardBridge } from "./L2StandardBridge.sol";
import { Predeploys } from "../libraries/Predeploys.sol";
import { FeeVault } from "../universal/FeeVault.sol";

/**
 * @custom:proxied
 * @custom:predeploy 0x4200000000000000000000000000000000000019
 * @title BaseFeeVault
 * @notice The BaseFeeVault accumulates the base fee that is paid by
 *         transactions.
 */
contract BaseFeeVault is FeeVault, Semver {
    /**
     * @custom:semver 0.0.1
     */
    constructor(address _recipient) FeeVault(_recipient, 10 ether) Semver(0, 0, 1) {}
}
