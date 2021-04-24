// SPDX-License-Identifier: MIT
pragma solidity >0.5.0 <0.8.0;
pragma experimental ABIEncoderV2;

/* Library Imports */
import { Lib_OVMCodec } from "../../libraries/codec/Lib_OVMCodec.sol";

/**
 * @title iOVM_ECDSAContractAccount
 */
interface iOVM_ECDSAContractAccount {

    /********************
     * Public Functions *
     ********************/

    function execute(
        bytes memory _encodedTransaction
    )
        external
        returns (
            bool,
            bytes memory
        );
}
