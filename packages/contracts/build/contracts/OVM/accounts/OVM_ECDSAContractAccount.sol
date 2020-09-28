// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.7.0;
pragma experimental ABIEncoderV2;

/* Interface Imports */
import { iOVM_ECDSAContractAccount } from "../../iOVM/accounts/iOVM_ECDSAContractAccount.sol";
import { iOVM_ExecutionManager } from "../../iOVM/execution/iOVM_ExecutionManager.sol";

/* Library Imports */
import { Lib_OVMCodec } from "../../libraries/codec/Lib_OVMCodec.sol";
import { Lib_ECDSAUtils } from "../../libraries/utils/Lib_ECDSAUtils.sol";

/**
 * @title OVM_ECDSAContractAccount
 */
contract OVM_ECDSAContractAccount is iOVM_ECDSAContractAccount {

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
        iOVM_ExecutionManager ovmExecutionManager = iOVM_ExecutionManager(msg.sender);

        // Address of this contract within the ovm (ovmADDRESS) should be the same as the
        // recovered address of the user who signed this message. This is how we manage to shim
        // account abstraction even though the user isn't a contract.
        require(
            Lib_ECDSAUtils.recover(
                _transaction,
                _signatureType == Lib_OVMCodec.EOASignatureType.ETH_SIGNED_MESSAGE,
                _v,
                _r,
                _s,
                ovmExecutionManager.ovmCHAINID()
            ) == ovmExecutionManager.ovmADDRESS(),
            "Signature provided for EOA transaction execution is invalid."
        );

        Lib_OVMCodec.EOATransaction memory decodedTx = Lib_OVMCodec.decodeEOATransaction(_transaction);

        // Need to make sure that the transaction nonce is right and bump it if so.
        require(
            decodedTx.nonce == ovmExecutionManager.ovmGETNONCE() + 1,
            "Transaction nonce does not match the expected nonce."
        );
        ovmExecutionManager.ovmSETNONCE(decodedTx.nonce);

        // Contract creations are signalled by sending a transaction to the zero address.
        if (decodedTx.target == address(0)) {
            address created = ovmExecutionManager.ovmCREATE{gas: decodedTx.gasLimit}(
                decodedTx.data
            );

            // EVM doesn't tell us whether a contract creation failed, even if it reverted during
            // initialization. Always return `true` for our success value here.
            return (true, abi.encode(created));
        } else {
            return ovmExecutionManager.ovmCALL(
                decodedTx.gasLimit,
                decodedTx.target,
                decodedTx.data
            );
        }
    }
}
