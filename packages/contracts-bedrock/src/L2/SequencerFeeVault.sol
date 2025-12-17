// SPDX-License-Identifier: MIT
pragma solidity 0.8.25;

// Contracts
import { FeeVault } from "src/L2/FeeVault.sol";

// Interfaces
import { ISemver } from "interfaces/universal/ISemver.sol";

/// @custom:proxied true
/// @custom:predeploy 0x4200000000000000000000000000000000000011
/// @title SequencerFeeVault
/// @notice The SequencerFeeVault is the contract that holds any fees paid to the Sequencer during
///         transaction processing and block production.
contract SequencerFeeVault is FeeVault, ISemver {
    /// @custom:semver 1.6.0
    string public constant version = "1.6.0";

    /// @custom:legacy
    /// @notice Legacy getter for the recipient address.
    /// @return The recipient address.
    function l1FeeWallet() public view returns (address) {
        return recipient;
    }
}
