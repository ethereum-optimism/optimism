// SPDX-License-Identifier: MIT
pragma solidity >0.5.0 <0.8.0;

/* Library Imports */
import { Lib_PredeployAddresses } from "../../libraries/constants/Lib_PredeployAddresses.sol";

/* Contract Imports */
import { L2StandardERC20 } from "../../libraries/standards/L2StandardERC20.sol";

/**
 * @title OVM_ETH
 * @dev The ETH predeploy provides an ERC20 interface for ETH deposited to Layer 2. Note that
 * unlike on Layer 1, Layer 2 accounts do not have a balance field.
 *
 * Compiler used: optimistic-solc
 * Runtime target: OVM
 */
contract OVM_ETH is L2StandardERC20 {

    /***************
     * Constructor *
     ***************/

    constructor()
        L2StandardERC20(
            Lib_PredeployAddresses.L2_STANDARD_BRIDGE,
            address(0),
            "Ether",
            "ETH"
        )
    {}
}
