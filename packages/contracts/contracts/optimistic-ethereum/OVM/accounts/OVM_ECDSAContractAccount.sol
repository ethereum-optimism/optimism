// SPDX-License-Identifier: MIT
pragma solidity >0.5.0 <0.8.0;
pragma experimental ABIEncoderV2;

/* Interface Imports */
import { iOVM_ECDSAContractAccount } from "../../iOVM/accounts/iOVM_ECDSAContractAccount.sol";

/* Library Imports */
import { Lib_EIP155Tx } from "../../libraries/codec/Lib_EIP155Tx.sol";
import { Lib_ExecutionManagerWrapper } from "../../libraries/wrappers/Lib_ExecutionManagerWrapper.sol";
import { Lib_PredeployAddresses } from "../../libraries/constants/Lib_PredeployAddresses.sol";

/* Contract Imports */
import { OVM_ETH } from "../predeploys/OVM_ETH.sol";

/* External Imports */
import { SafeMath } from "@openzeppelin/contracts/math/SafeMath.sol";
import { Math } from "@openzeppelin/contracts/math/Math.sol";

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


    /*********************
     * Private Functions *
     *********************/

    /**
     * Compute the intrinsic gas of a transaction
     * @param _encodedTransaction Signed EIP155 transaction bytes.
     * @param _isCreate Whether the transaction creates a contract
     * @return The amount of intrinsic gas
     */
    function _intrinsicGas(
        bytes memory _encodedTransaction,
        bool _isCreate
    )
        private
        returns (
            uint256
        )
    {
        uint256 intrinsicGas = SafeMath.mul(
            _encodedTransaction.length,
            16
        );
        if (_isCreate) {
            intrinsicGas = SafeMath.add(
                intrinsicGas,
                53000
            );
        } else {
            intrinsicGas = SafeMath.add(
                intrinsicGas,
                21000
            );
        }
        return intrinsicGas;
    }


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

        // Compute the intrinsic gas
        uint256 intrinsicGas = _intrinsicGas(
            _encodedTransaction,
            transaction.isCreate
        );

        // Need to make sure that the gas is sufficient to execute the transaction.
        uint256 gasLimit = SafeMath.mul((transaction.gasLimit % 10000), 10000);
        require(
            gasleft() >= gasLimit,
            "Gas is not sufficient to execute the transaction."
        );

        // Address of this contract within the ovm (ovmADDRESS) should be the same as the
        // recovered address of the user who signed this message. This is how we manage to shim
        // account abstraction even though the user isn't a contract.
        require(
            transaction.sender() == Lib_ExecutionManagerWrapper.ovmADDRESS(),
            "Signature provided for EOA transaction execution is invalid."
        );

        // Need to make sure that the transaction nonce is right.
        require(
            transaction.nonce == Lib_ExecutionManagerWrapper.ovmGETNONCE(),
            "Transaction nonce does not match the expected nonce."
        );

        // Transfer fee to relayer.
        require(
            OVM_ETH(Lib_PredeployAddresses.OVM_ETH).transfer(
                Lib_PredeployAddresses.SEQUENCER_FEE_WALLET,
                SafeMath.mul(transaction.gasLimit, transaction.gasPrice)
            ),
            "Fee was not transferred to relayer."
        );

        // Compute the gas limit passed to the sub call
        uint256 callGasLimit = SafeMath.sub(
            Math.max(gasLimit, intrinsicGas),
            intrinsicGas
        );

        if (transaction.isCreate) {
            // TEMPORARY: Disable value transfer for contract creations.
            require(
                transaction.value == 0,
                "Value transfer in contract creation not supported."
            );

            (address created, bytes memory revertdata) = Lib_ExecutionManagerWrapper.ovmCREATE(
                transaction.data,
                callGasLimit
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

            // Value transfer currently only supported for CALL but not for CREATE.
            if (transaction.value > 0) {
                // TEMPORARY: Block value transfer if the transaction has input data.
                require(
                    transaction.data.length == 0,
                    "Value is nonzero but input data was provided."
                );

                require(
                    OVM_ETH(Lib_PredeployAddresses.OVM_ETH).transfer(
                        transaction.to,
                        transaction.value
                    ),
                    "Value could not be transferred to recipient."
                );

                return (true, bytes(""));
            } else {
                // NOTE: Upgrades are temporarily disabled because users can, in theory, modify their EOA
                // so that they don't have to pay any fees to the sequencer. Function will remain disabled
                // until a robust solution is in place.
                require(
                    transaction.to != Lib_ExecutionManagerWrapper.ovmADDRESS(),
                    "Calls to self are disabled until upgradability is re-enabled."
                );

                return transaction.to.call{gas: callGasLimit}(transaction.data);
            }
        }
    }
}
