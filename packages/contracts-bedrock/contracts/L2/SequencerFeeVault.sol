// SPDX-License-Identifier: MIT
pragma solidity ^0.8.9;

/* Library Imports */
import { Lib_PredeployAddresses } from "../libraries/Lib_PredeployAddresses.sol";

/* Contract Imports */
import { L2StandardBridge } from "./L2StandardBridge.sol";

/**
 * @custom:proxied
 * @custom:predeploy 0x4200000000000000000000000000000000000011
 * @title SequencerFeeVault
 * @notice The SequencerFeeVault is the contract that holds any fees paid to the Sequencer during
 *         transaction processing and block production.
 */
contract SequencerFeeVault {
    /**
     * @notice Minimum balance before a withdrawal can be triggered.
     */
    uint256 public constant MIN_WITHDRAWAL_AMOUNT = 15 ether;

    /**
     * @notice Wallet that will receive the fees on L1.
     */
    address public l1FeeWallet;

    /**
     * @notice Allow the contract to receive ETH.
     */
    receive() external payable {}

    /**
     * @notice Triggers a withdrawal of funds to the L1 fee wallet.
     */
    function withdraw() external {
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
