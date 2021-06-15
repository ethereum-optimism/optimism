// SPDX-License-Identifier: MIT
pragma solidity >0.5.0 <0.8.0;
pragma experimental ABIEncoderV2;

/* Library Imports */
import { Lib_EIP155Tx } from "../../libraries/codec/Lib_EIP155Tx.sol";

/**
 * @title iOVM_ECDSAContractAccount
 */
interface iOVM_ECDSAContractAccount {

    /********************
     * Public Functions *
     ********************/

    function execute(
        Lib_EIP155Tx.EIP155Tx memory _transaction
    )
        external
        returns (
            bool,
            bytes memory
        );
}
