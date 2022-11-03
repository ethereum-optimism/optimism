// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Semver } from "../universal/Semver.sol";
import { L2StandardBridge } from "./L2StandardBridge.sol";
import { Predeploys } from "../libraries/Predeploys.sol";
import { FeeVault } from "../universal/FeeVault.sol";

/**
 * @custom:proxied
 * @custom:predeploy 0x4200000000000000000000000000000000000011
 * @title SequencerFeeVault
 * @notice The SequencerFeeVault is the contract that holds any fees paid to the Sequencer during
 *         transaction processing and block production.
 */
contract SequencerFeeVault is FeeVault, Semver {
    /**
     * @custom:spacer l1FeeWallet
     * @notice Spacer for backwards compatibility.
     */
    address private spacer_0_0_20;

    /**
     * @custom:semver 0.0.1
     */
    constructor(address _recipient) FeeVault(_recipient, 10 ether) Semver(0, 0, 1) {}

    /**
     * @custom:legacy
     * @notice: Legacy getter for the recipient
     */
    function l1FeeWallet() public view returns (address) {
        return RECIPIENT;
    }
}
