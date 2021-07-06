// SPDX-License-Identifier: MIT
pragma solidity >0.5.0 <0.8.0;

/* Library Imports */
import { Lib_PredeployAddresses } from "../../libraries/constants/Lib_PredeployAddresses.sol";

/* Contract Imports */
import { OVM_ETH } from "../predeploys/OVM_ETH.sol";
import { OVM_L2StandardBridge } from "../bridge/tokens/OVM_L2StandardBridge.sol";

/**
 * @title OVM_SequencerFeeVault
 * @dev Simple holding contract for fees paid to the Sequencer. Likely to be replaced in the future
 * but "good enough for now".
 *
 * Compiler used: optimistic-solc
 * Runtime target: OVM
 */
contract OVM_SequencerFeeVault {

    /*************
     * Constants *
     *************/

    // Minimum ETH balance that can be withdrawn in a single withdrawal.
    uint256 public constant MIN_WITHDRAWAL_AMOUNT = 15 ether;


    /*************
     * Variables *
     *************/

    // Address on L1 that will hold the fees once withdrawn. Dynamically initialized within l2geth.
    address public l1FeeWallet;


    /***************
     * Constructor *
     ***************/

    /**
     * @param _l1FeeWallet Initial address for the L1 wallet that will hold fees once withdrawn.
     * Currently HAS NO EFFECT in production because l2geth will mutate this storage slot during
     * the genesis block. This is ONLY for testing purposes.
     */
    constructor(
        address _l1FeeWallet
    ) {
        l1FeeWallet = _l1FeeWallet;
    }


    /********************
     * Public Functions *
     ********************/

    function withdraw()
        public
    {
        uint256 balance = OVM_ETH(Lib_PredeployAddresses.OVM_ETH).balanceOf(address(this));

        require(
            balance >= MIN_WITHDRAWAL_AMOUNT,
            // solhint-disable-next-line max-line-length
            "OVM_SequencerFeeVault: withdrawal amount must be greater than minimum withdrawal amount"
        );

        OVM_L2StandardBridge(Lib_PredeployAddresses.L2_STANDARD_BRIDGE).withdrawTo(
            Lib_PredeployAddresses.OVM_ETH,
            l1FeeWallet,
            balance,
            0,
            bytes("")
        );
    }
}
