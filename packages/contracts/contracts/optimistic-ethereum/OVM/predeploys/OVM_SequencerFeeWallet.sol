// SPDX-License-Identifier: MIT
pragma solidity >0.5.0 <0.8.0;

/* Library Imports */
import { Lib_PredeployAddresses } from "../../libraries/constants/Lib_PredeployAddresses.sol";

/* Contract Imports */
import { OVM_ETH } from "../predeploys/OVM_ETH.sol";

/**
 * @title OVM_SequencerFeeWallet
 * @dev Simple holding contract for fees paid to the Sequencer. Likely to be replaced in the future
 * but "good enough for now".
 *
 * Compiler used: optimistic-solc
 * Runtime target: OVM
 */
contract OVM_SequencerFeeWallet {

    /*************
     * Constants *
     *************/

    uint256 constant MIN_WITHDRAWAL_AMOUNT = 10 ether;
    address constant L1_FEE_WALLET = 0x1111111111111111111111111111111111111111;


    /********************
     * Public Functions *
     ********************/

    function withdraw(
        uint256 _amount
    )
        public
    {
        require(
            _amount >= MIN_WITHDRAWAL_AMOUNT,
            "OVM_SequencerFeeWallet: withdrawal amount must be greater than minimum withdrawal amount"
        );

        OVM_ETH(Lib_PredeployAddresses.OVM_ETH).withdrawTo(
            L1_FEE_WALLET,
            _amount,
            0,
            bytes("")
        );
    }
}
