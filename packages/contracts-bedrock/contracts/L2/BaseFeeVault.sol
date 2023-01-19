// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Semver } from "../universal/Semver.sol";
import { FeeVault } from "../universal/FeeVault.sol";

/**
 * @custom:proxied
 * @custom:predeploy 0x4200000000000000000000000000000000000019
 * @title BaseFeeVault
 * @notice The BaseFeeVault accumulates the base fee that is paid by transactions.
 */
contract BaseFeeVault is FeeVault, Semver {
    /**
     * @custom:semver 1.1.0
     *
     * @param _recipient Address that will receive the accumulated fees.
     */
    constructor(address _recipient) FeeVault(_recipient, 10 ether) Semver(1, 1, 0) {}
}
