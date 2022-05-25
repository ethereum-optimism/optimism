// SPDX-License-Identifier: MIT
pragma solidity ^0.8.9;

/* Library Imports */
import {
    Lib_PredeployAddresses
} from "@eth-optimism/contracts/libraries/constants/Lib_PredeployAddresses.sol";

/* Contract Imports */
import { L2StandardBridge } from "./L2StandardBridge.sol";

/**
 * @title SequencerFeeVault
 * @dev Simple holding contract for fees paid to the Sequencer
 */
contract SequencerFeeVault {
    /*************
     * Constants *
     *************/

    // Minimum ETH balance that can be withdrawn in a single withdrawal.
    uint256 public constant MIN_WITHDRAWAL_AMOUNT = 15 ether;

    /*************
     * Variables *
     *************/

    // Address on L1 that will hold the fees once withdrawn. Dynamically
    // initialized in the genesis state
    address public l1FeeWallet;

    /************
     * Fallback *
     ************/

    // slither-disable-next-line locked-ether
    receive() external payable {}

    /********************
     * Public Functions *
     ********************/

    // slither-disable-next-line external-function
    function withdraw() public {
        require(
            address(this).balance >= MIN_WITHDRAWAL_AMOUNT,
            // solhint-disable-next-line max-line-length
            "OVM_SequencerFeeVault: withdrawal amount must be greater than minimum withdrawal amount"
        );

        uint256 balance = address(this).balance;

        L2StandardBridge(payable(Lib_PredeployAddresses.L2_STANDARD_BRIDGE)).withdrawTo{
            value: balance
        }(Lib_PredeployAddresses.OVM_ETH, l1FeeWallet, balance, 0, bytes(""));
    }
}
