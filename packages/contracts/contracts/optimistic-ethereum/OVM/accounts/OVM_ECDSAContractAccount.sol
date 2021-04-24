// SPDX-License-Identifier: MIT
// @unsupported: evm
pragma solidity >0.5.0 <0.8.0;
pragma experimental ABIEncoderV2;

/* Interface Imports */
import { iOVM_ECDSAContractAccount } from "../../iOVM/accounts/iOVM_ECDSAContractAccount.sol";

/* Library Imports */
import { Lib_EIP155Tx } from "../../libraries/codec/Lib_EIP155Tx.sol";
import { Lib_ExecutionManagerWrapper } from "../../libraries/wrappers/Lib_ExecutionManagerWrapper.sol";

/* Contract Imports */
import { OVM_ETH } from "../predeploys/OVM_ETH.sol";

/* External Imports */
import { SafeMath } from "@openzeppelin/contracts/math/SafeMath.sol";

/**
 * @title OVM_ECDSAContractAccount
 * @dev The ECDSA Contract Account can be used as the implementation for a ProxyEOA deployed by the
 * ovmCREATEEOA operation. It enables backwards compatibility with Ethereum's Layer 1, by 
 * providing EIP155 formatted transaction encodings.
 *
 * Compiler used: optimistic-solc
 * Runtime target: OVM
 */
contract OVM_ECDSAContractAccount is iOVM_ECDSAContractAccount {

    /*************
     * Libraries *
     *************/

    using Lib_EIP155Tx for Lib_EIP155Tx.EIP155Tx;


    /*************
     * Constants *
     *************/

    // TODO: should be the amount sufficient to cover the gas costs of all of the transactions up
    // to and including the CALL/CREATE which forms the entrypoint of the transaction.
    uint256 constant EXECUTION_VALIDATION_GAS_OVERHEAD = 25000;
    OVM_ETH constant ovmETH = OVM_ETH(0x4200000000000000000000000000000000000006);


    /********************
     * Public Functions *
     ********************/

    /**
     * Executes a signed transaction.
     * @param _encodedTransaction Signed EIP155 transaction.
     * @return Whether or not the call returned (rather than reverted).
     * @return Data returned by the call.
     */
    function execute(
        bytes memory _encodedTransaction
    )
        override
        public
        returns (
            bool,
            bytes memory
        )
    {
        // Attempt to decode the transaction.
        Lib_EIP155Tx.EIP155Tx memory transaction = Lib_EIP155Tx.decode(
            _encodedTransaction,
            Lib_ExecutionManagerWrapper.ovmCHAINID()
        );

        // Address of this contract within the ovm (ovmADDRESS) should be the same as the
        // recovered address of the user who signed this message. This is how we manage to shim
        // account abstraction even though the user isn't a contract.
        require(
            transaction.sender() == address(this),
            "Signature provided for EOA transaction execution is invalid."
        );

        // Need to make sure that the transaction nonce is right.
        require(
            transaction.nonce == Lib_ExecutionManagerWrapper.ovmGETNONCE(),
            "Transaction nonce does not match the expected nonce."
        );

        // TEMPORARY: Disable gas checks for mainnet.
        // // Need to make sure that the gas is sufficient to execute the transaction.
        // require(
        //    gasleft() >= SafeMath.add(transaction.gasLimit, EXECUTION_VALIDATION_GAS_OVERHEAD),
        //    "Gas is not sufficient to execute the transaction."
        // );

        // Transfer fee to relayer.
        require(
            ovmETH.transfer(
                msg.sender,
                SafeMath.mul(transaction.gasLimit, transaction.gasPrice)
            ),
            "Fee was not transferred to relayer."
        );

        // Contract creations are signalled by sending a transaction to the zero address.
        if (transaction.isCreate) {
            (address created, bytes memory revertdata) = Lib_ExecutionManagerWrapper.ovmCREATE(
                transaction.data
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

            return transaction.to.call(transaction.data);
        }
    }
}
