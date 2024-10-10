// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { ISemver } from "src/universal/interfaces/ISemver.sol";
import { FeeVault } from "src/L2/FeeVault.sol";
import { Types } from "src/libraries/Types.sol";

/// @custom:proxied true
/// @custom:predeploy 0x4200000000000000000000000000000000000011
/// @title SequencerFeeVault
/// @notice The SequencerFeeVault is the contract that holds any fees paid to the Sequencer during
///         transaction processing and block production.
contract SequencerFeeVault is FeeVault, ISemver {
    /// @custom:semver 1.5.0-beta.3
    string public constant version = "1.5.0-beta.3";

    /// @notice Returns the FeeVault config
    /// @return recipient_           Wallet that will receive the fees.
    /// @return amount_              Minimum balance for withdrawals.
    /// @return withdrawalNetwork_   Network which the recipient will receive fees on.
    function config()
        public
        view
        override
        returns (address recipient_, uint256 amount_, Types.WithdrawalNetwork withdrawalNetwork_)
    {
        (recipient_, amount_, withdrawalNetwork_) = L1_BLOCK().sequencerFeeVaultConfig();
    }

    /// @custom:legacy
    /// @notice Legacy getter for the recipient address.
    /// @return recipient_ The recipient address.
    function l1FeeWallet() public view returns (address recipient_) {
        recipient_ = recipient();
    }
}
