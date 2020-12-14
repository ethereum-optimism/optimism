// SPDX-License-Identifier: MIT
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
        bool isEthSign = _signatureType == Lib_OVMCodec.EOASignatureType.ETH_SIGNED_MESSAGE;
        Lib_OVMCodec.EIP155Transaction memory decodedTx = Lib_OVMCodec.decodeEIP155Transaction(_transaction, isEthSign);

        // Need to make sure that the transaction nonce is right.
        Lib_SafeExecutionManagerWrapper.safeREQUIRE(
            decodedTx.nonce == Lib_SafeExecutionManagerWrapper.safeGETNONCE(),
            "Transaction nonce does not match the expected nonce."
        );

        // Contract creations are signalled by sending a transaction to the zero address.
        if (decodedTx.to == address(0)) {
            address created = Lib_SafeExecutionManagerWrapper.safeCREATE(
                decodedTx.gasLimit,
                decodedTx.data
            );

            // If the created address is the ZERO_ADDRESS then we know the deployment failed, though not why
            return (created != address(0), abi.encode(created));
        } else {
            // We only want to bump the nonce for `ovmCALL` because `ovmCREATE` automatically bumps
            // the nonce of the calling account. Normally an EOA would bump the nonce for both
            // cases, but since this is a contract we'd end up bumping the nonce twice.
            Lib_SafeExecutionManagerWrapper.safeSETNONCE(decodedTx.nonce + 1);

            return Lib_SafeExecutionManagerWrapper.safeCALL(
                decodedTx.gasLimit,
                decodedTx.to,
                decodedTx.data
            );
        }
    }

    function qall(
        uint256 _gasLimit,
        address _to,
        bytes memory _data
    )
        public
        returns (
            bool _success,
            bytes memory _returndata
        )
    {
        return Lib_SafeExecutionManagerWrapper.safeCALL(
            _gasLimit,
            _to,
            _data
        );
    }
}
