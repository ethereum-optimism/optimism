// SPDX-License-Identifier: MIT
pragma solidity >0.5.0 <0.8.0;

/* Library Imports */
import { Lib_EIP155Tx } from "../../libraries/codec/Lib_EIP155Tx.sol";
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
    using Lib_EIP155Tx for Lib_EIP155Tx.EIP155Tx;


    /*********************
     * Fallback Function *
     *********************/

    /**
     * Expects an RLP-encoded EIP155 transaction as input. See the EIP for a more detailed
     * description of this transaction format:
     * https://github.com/ethereum/EIPs/blob/master/EIPS/eip-155.md
     */
    fallback()
        external
    {
        Lib_EIP155Tx.EIP155Tx memory transaction = Lib_EIP155Tx.decode(
            msg.data,
            Lib_SafeExecutionManagerWrapper.safeCHAINID()
        );

        // Recovery parameter being something other than 0 or 1 indicates that this transaction was
        // signed using the wrong chain ID. We really should have this logic inside of the 
        Lib_SafeExecutionManagerWrapper.safeREQUIRE(
            transaction.recoveryParam < 2,
            "OVM_SequencerEntrypoint: Transaction was signed with the wrong chain ID."
        );

        // Cache this result since we use it twice. Maybe we could move this caching into
        // Lib_EIP155Tx but I'd rather not make optimizations like that right now.
        address sender = transaction.sender();

        // Create an EOA contract for this account if it doesn't already exist.
        if (Lib_SafeExecutionManagerWrapper.safeEXTCODESIZE(sender) == 0) {
            Lib_SafeExecutionManagerWrapper.safeCREATEEOA(
                transaction.hash(),
                transaction.recoveryParam,
                transaction.r,
                transaction.s
            );
        }

        // Now call into the EOA contract (which should definitely exist).
        Lib_SafeExecutionManagerWrapper.safeCALL(
            gasleft(),
            sender,
            abi.encodeWithSignature(
                "execute(bytes)",
                msg.data
            )
        );
    }
}
