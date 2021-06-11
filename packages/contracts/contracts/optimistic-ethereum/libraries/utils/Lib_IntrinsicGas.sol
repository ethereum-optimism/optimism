// SPDX-License-Identifier: MIT
pragma solidity >0.5.0 <0.8.0;
pragma experimental ABIEncoderV2;

/* Library Imports */
import { Lib_OVMCodec } from "../codec/Lib_OVMCodec.sol";

/**
 * @title Lib_IntrinsicGas
 */
library Lib_IntrinsicGas {

    /**
     * Computes the intrinsic gas of the OVM_ECDSAContractAccount
     * execute method.
     * @param _datalength Size of the calldata
     * @return Amount of intrinsic gas used
     */
    function ecdsaContractAccount(
        uint256 _datalength
    )
        internal
        returns (
            uint256
        )
    {
       return 50000 + (_datalength + 109) * 16;
    }
}
