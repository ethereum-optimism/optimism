// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.7.0;
pragma experimental ABIEncoderV2;

/* Interface Imports */
import { iOVM_ECDSAContractAccount } from "../../iOVM/accounts/iOVM_ECDSAContractAccount.sol";

/* Library Imports */
import { Lib_OVMCodec } from "../../libraries/codec/Lib_OVMCodec.sol";
import { Lib_ECDSAUtils } from "../../libraries/utils/Lib_ECDSAUtils.sol";
import { Lib_SafeExecutionManagerWrapper } from "../../libraries/wrappers/Lib_SafeExecutionManagerWrapper.sol";

/**
 * @title mockOVM_ECDSAContractAccount
 */
contract mockOVM_ECDSAContractAccount is iOVM_ECDSAContractAccount {

    /********************
     * Public Functions *
     ********************/

    /**
     * Executes a signed transaction.
     * @param _transaction Signed EOA transaction.
     * @param _signatureType Hashing scheme used for the transaction (e.g., ETH signed message).
     * @param _v Signature `v` parameter.
     * @param _r Signature `r` parameter.
     * @param _s Signature `s` parameter.
     * @return _success Whether or not the call returned (rather than reverted).
     * @return _returndata Data returned by the call.
     */
    function execute(
        bytes memory _transaction,
        Lib_OVMCodec.EOASignatureType _signatureType,
        uint8 _v,
        bytes32 _r,
        bytes32 _s
    )
        override
        public
        returns (
            bool _success,
            bytes memory _returndata
        )
    {
        address ovmExecutionManager = msg.sender;

        // Skip signature validation in this mock.
        Lib_OVMCodec.EOATransaction memory decodedTx = Lib_OVMCodec.decodeEOATransaction(_transaction);

        // Contract creations are signalled by sending a transaction to the zero address.
        if (decodedTx.target == address(0)) {
            address created = Lib_SafeExecutionManagerWrapper.safeCREATE(
                ovmExecutionManager,
                decodedTx.gasLimit,
                decodedTx.data
            );

            // EVM doesn't tell us whether a contract creation failed, even if it reverted during
            // initialization. Always return `true` for our success value here.
            return (true, abi.encode(created));
        } else {
            return Lib_SafeExecutionManagerWrapper.safeCALL(
                ovmExecutionManager,
                decodedTx.gasLimit,
                decodedTx.target,
                decodedTx.data
            );
        }
    }
}
