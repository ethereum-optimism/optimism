// SPDX-License-Identifier: MIT
// @unsupported: evm
pragma solidity >0.5.0 <0.8.0;

/* Interface Imports */
import { iOVM_ECDSAContractAccount } from "../../iOVM/accounts/iOVM_ECDSAContractAccount.sol";

/* Library Imports */
import { Lib_EIP155Tx } from "../../libraries/codec/Lib_EIP155Tx.sol";
import { Lib_ExecutionManagerWrapper } from "../../libraries/wrappers/Lib_ExecutionManagerWrapper.sol";

/**
 * @title OVM_SequencerEntrypoint
 * @dev The Sequencer Entrypoint is a predeploy which, despite its name, can in fact be called by 
 * any account. It accepts a more efficient compressed calldata format, which it decompresses and 
 * encodes to the standard EIP155 transaction format.
 * 
 * Compiler used: optimistic-solc
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
            Lib_ExecutionManagerWrapper.ovmCHAINID()
        );

        // Value is computed on the fly. Keep it in the stack to save some gas.
        address target = transaction.sender();

        bool isEmptyContract;
        assembly {
            isEmptyContract := iszero(extcodesize(target))
        }

        if (isEmptyContract) {
            Lib_ExecutionManagerWrapper.ovmCREATEEOA(
                transaction.hash(),
                transaction.recoveryParam,
                transaction.r,
                transaction.s
            );
        }

        iOVM_ECDSAContractAccount(target).execute(msg.data);
    }
}
