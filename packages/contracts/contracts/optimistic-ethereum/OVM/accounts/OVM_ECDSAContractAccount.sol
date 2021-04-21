// SPDX-License-Identifier: MIT
// @unsupported: evm
pragma solidity >0.5.0 <0.8.0;
pragma experimental ABIEncoderV2;

/* Interface Imports */
import { iOVM_ERC20 } from "../../iOVM/predeploys/iOVM_ERC20.sol";
import { iOVM_ECDSAContractAccount } from "../../iOVM/accounts/iOVM_ECDSAContractAccount.sol";

/* Library Imports */
import { Lib_OVMCodec } from "../../libraries/codec/Lib_OVMCodec.sol";
import { Lib_ECDSAUtils } from "../../libraries/utils/Lib_ECDSAUtils.sol";
import { Lib_ExecutionManagerWrapper } from "../../libraries/wrappers/Lib_ExecutionManagerWrapper.sol";

/* External Imports */
import { SafeMath } from "@openzeppelin/contracts/math/SafeMath.sol";

/**
 * @title OVM_ECDSAContractAccount
 * @dev The ECDSA Contract Account can be used as the implementation for a ProxyEOA deployed by the
 * ovmCREATEEOA operation. It enables backwards compatibility with Ethereum's Layer 1, by 
 * providing eth_sign and EIP155 formatted transaction encodings.
 *
 * Compiler used: optimistic-solc
 * Runtime target: OVM
 */
contract OVM_ECDSAContractAccount is iOVM_ECDSAContractAccount {

    /*************
     * Constants *
     *************/

    // TODO: should be the amount sufficient to cover the gas costs of all of the transactions up
    // to and including the CALL/CREATE which forms the entrypoint of the transaction.
    uint256 constant EXECUTION_VALIDATION_GAS_OVERHEAD = 25000;
    iOVM_ERC20 constant ETH_ERC20 = iOVM_ERC20(0x4200000000000000000000000000000000000006);


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
     * @return Whether or not the call returned (rather than reverted).
     * @return Data returned by the call.
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
            bool,
            bytes memory
        )
    {
        bool isEthSign = _signatureType == Lib_OVMCodec.EOASignatureType.ETH_SIGNED_MESSAGE;

        // Address of this contract within the ovm (ovmADDRESS) should be the same as the
        // recovered address of the user who signed this message. This is how we manage to shim
        // account abstraction even though the user isn't a contract.
        // Need to make sure that the transaction nonce is right and bump it if so.
        require(
            Lib_ECDSAUtils.recover(
                _transaction,
                isEthSign,
                _v,
                _r,
                _s
            ) == address(this),
            "Signature provided for EOA transaction execution is invalid."
        );

        Lib_OVMCodec.EIP155Transaction memory decodedTx = Lib_OVMCodec.decodeEIP155Transaction(
            _transaction,
            isEthSign
        );

        // Grab the chain ID of the current network.
        uint256 chainId;
        assembly {
            chainId := chainid()
        }

        // Need to make sure that the transaction chainId is correct.
        require(
            decodedTx.chainId == chainId,
            "Transaction chainId does not match expected OVM chainId."
        );

        // Need to make sure that the transaction nonce is right.
        require(
            decodedTx.nonce == Lib_ExecutionManagerWrapper.ovmGETNONCE(),
            "Transaction nonce does not match the expected nonce."
        );

        // TEMPORARY: Disable gas checks for mainnet.
        // // Need to make sure that the gas is sufficient to execute the transaction.
        // require(
        //    gasleft() >= SafeMath.add(decodedTx.gasLimit, EXECUTION_VALIDATION_GAS_OVERHEAD),
        //    "Gas is not sufficient to execute the transaction."
        // );

        // Transfer fee to relayer.
        require(
            ETH_ERC20.transfer(
                msg.sender,
                SafeMath.mul(decodedTx.gasLimit, decodedTx.gasPrice)
            ),
            "Fee was not transferred to relayer."
        );

        // Contract creations are signalled by sending a transaction to the zero address.
        if (decodedTx.to == address(0)) {
            (address created, bytes memory revertdata) = Lib_ExecutionManagerWrapper.ovmCREATE(
                decodedTx.data
            );

            // Return true if the contract creation succeeded, false w/ revertdata otherwise.
            if (created != address(0)) {
                return (true, abi.encode(created));
            } else {
                return (false, revertdata);
            }
        } else {
            // We only want to bump the nonce for `ovmCALL` because `ovmCREATE` automatically bumps
            // the nonce of the calling account. Normally an EOA would bump the nonce for both
            // cases, but since this is a contract we'd end up bumping the nonce twice.
            Lib_ExecutionManagerWrapper.ovmINCREMENTNONCE();

            return decodedTx.to.call(decodedTx.data);
        }
    }
}
