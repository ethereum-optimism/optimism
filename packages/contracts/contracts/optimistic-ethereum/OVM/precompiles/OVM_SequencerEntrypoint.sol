// SPDX-License-Identifier: MIT
pragma solidity >0.5.0 <0.8.0;

/* Library Imports */
import { Lib_BytesUtils } from "../../libraries/utils/Lib_BytesUtils.sol";
import { Lib_OVMCodec } from "../../libraries/codec/Lib_OVMCodec.sol";
import { Lib_ECDSAUtils } from "../../libraries/utils/Lib_ECDSAUtils.sol";
import { Lib_SafeExecutionManagerWrapper } from "../../libraries/wrappers/Lib_SafeExecutionManagerWrapper.sol";

/**
 * @title OVM_SequencerEntrypoint
 * @dev The Sequencer Entrypoint is a predeploy which, despite its name, can in fact be called by 
 * any account. It accepts a more efficient compressed calldata format, which it decompresses and 
 * encodes to the standard EIP155 transaction format.
 * This contract is the implementation referenced by the Proxy Sequencer Entrypoint, thus enabling
 * the Optimism team to upgrade the decompression of calldata from the Sequencer.
 * 
 * Compiler used: solc
 * Runtime target: OVM
 */
contract OVM_SequencerEntrypoint {

    /*********
     * Enums *
     *********/
    
    enum TransactionType {
        NATIVE_ETH_TRANSACTION,
        ETH_SIGNED_MESSAGE
    }


    /*********************
     * Fallback Function *
     *********************/

    /**
     * Uses a custom "compressed" format to save on calldata gas:
     * calldata[00:01]: transaction type (0 == EIP 155, 2 == Eth Sign Message)
     * calldata[01:33]: signature "r" parameter
     * calldata[33:65]: signature "s" parameter
     * calldata[65:66]: signature "v" parameter
     * calldata[66:69]: transaction gas limit
     * calldata[69:72]: transaction gas price
     * calldata[72:75]: transaction nonce
     * calldata[75:95]: transaction target address
     * calldata[95:XX]: transaction data
     */
    fallback()
        external
    {
        TransactionType transactionType = _getTransactionType(Lib_BytesUtils.toUint8(msg.data, 0));

        bytes32 r = Lib_BytesUtils.toBytes32(Lib_BytesUtils.slice(msg.data, 1, 32));
        bytes32 s = Lib_BytesUtils.toBytes32(Lib_BytesUtils.slice(msg.data, 33, 32));
        uint8 v = Lib_BytesUtils.toUint8(msg.data, 65);

        // Remainder is the transaction to execute.
        bytes memory compressedTx = Lib_BytesUtils.slice(msg.data, 66);
        bool isEthSignedMessage = transactionType == TransactionType.ETH_SIGNED_MESSAGE;

        // Need to decompress and then re-encode the transaction based on the original encoding.
        bytes memory encodedTx = Lib_OVMCodec.encodeEIP155Transaction(
            Lib_OVMCodec.decompressEIP155Transaction(compressedTx),
            isEthSignedMessage
        );

        address target = Lib_ECDSAUtils.recover(
            encodedTx,
            isEthSignedMessage,
            uint8(v),
            r,
            s
        );

        if (Lib_SafeExecutionManagerWrapper.safeEXTCODESIZE(target) == 0) {
            // ProxyEOA has not yet been deployed for this EOA.
            bytes32 messageHash = Lib_ECDSAUtils.getMessageHash(encodedTx, isEthSignedMessage);
            Lib_SafeExecutionManagerWrapper.safeCREATEEOA(messageHash, uint8(v), r, s);
        }

        // ProxyEOA has been deployed for this EOA, continue to CALL.
        bytes memory callbytes = abi.encodeWithSignature(
            "execute(bytes,uint8,uint8,bytes32,bytes32)",
            encodedTx,
            isEthSignedMessage,
            uint8(v),
            r,
            s
        );

        Lib_SafeExecutionManagerWrapper.safeCALL(
            gasleft(),
            target,
            callbytes
        );
    }
    

    /**********************
     * Internal Functions *
     **********************/

    /**
     * Converts a uint256 into a TransactionType enum.
     * @param _transactionType Transaction type index.
     * @return Transaction type enum value.
     */
    function _getTransactionType(
        uint8 _transactionType
    )
        internal
        returns (
            TransactionType
        )
    {
        if (_transactionType == 0) {
            return TransactionType.NATIVE_ETH_TRANSACTION;
        } if (_transactionType == 2) {
            return TransactionType.ETH_SIGNED_MESSAGE;
        } else {
            Lib_SafeExecutionManagerWrapper.safeREVERT(
                "Transaction type must be 0 or 2"
            );
        }
    }
}
