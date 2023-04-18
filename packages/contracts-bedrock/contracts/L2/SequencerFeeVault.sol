// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Semver } from "../universal/Semver.sol";
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
     * @custom:semver 1.1.0
     *
     * @param _recipient Address that will receive the accumulated fees.
     */
    constructor(address _recipient) FeeVault(_recipient, 10 ether) Semver(1, 1, 0) {}

    /**
     * @custom:legacy
     * @notice Legacy getter for the recipient address.
     *
     * @return The recipient address.
     */
    function l1FeeWallet() public view returns (address) {
        return RECIPIENT;
    }
}
