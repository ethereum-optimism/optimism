// SPDX-License-Identifier: MIT
pragma solidity >0.5.0 <0.8.0;

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
        pure
        returns (
            uint256
        )
    {
        // This curve fit was calculated empirically via integration tests.
        // See integration-tests/test/fee-payment.spec.ts "should use the correctly estimated intrinsic gas for transactions of varying lengths"
        return 383213
            + (161 * _datalength) / 10
            + (762 * (_datalength ** 2)) / 10000000;
    }
}
