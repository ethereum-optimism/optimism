pragma solidity ^0.5.0;

/* Library Imports */
import { DataTypes } from "../../utils/libraries/DataTypes.sol";
import { TransactionParser } from "../../utils/libraries/TransactionParser.sol";
import { ECDSAUtils } from "../../utils/libraries/ECDSAUtils.sol";
import { ExecutionManagerWrapper } from "../../utils/libraries/ExecutionManagerWrapper.sol";

/* Contract Imports */
import { ExecutionManager } from "../ExecutionManager.sol";

/**
 * @title SequencerMessageDecompressor
 */
contract SequencerMessageDecompressor {
    /*
     * Fallback Function
     */

    function()
        external
    {
        bytes1 transactionType;
        bytes1 v;
        bytes32 r;
        bytes32 s;
        assembly {
            // Set up our pointers.
            let ptr_transactionType := mload(0x40)
            mstore(0x40, add(ptr_transactionType, 1))
            let ptr_v := mload(0x40)
            mstore(0x40, add(ptr_v, 1))
            let ptr_r := mload(0x40)
            mstore(0x40, add(ptr_r, 32))
            let ptr_s := mload(0x40)
            mstore(0x40, add(ptr_s, 32))

            // Copy calldata into our pointers.
            calldatacopy(ptr_transactionType, 0, 1)
            calldatacopy(ptr_v, 1, 1)
            calldatacopy(ptr_r, 2, 32)
            calldatacopy(ptr_s, 34, 32)

            // Load results into our variables.
            transactionType := mload(ptr_transactionType)
            v := mload(ptr_v)
            r := mload(ptr_r)
            s := mload(ptr_s)
        }

        if (uint8(transactionType) == 0) {
            bytes32 messageHash;
            assembly {
                // Set up our pointers.
                let ptr_messageHash := mload(0x40)
                mstore(0x40, add(ptr_messageHash, 32))

                // Copy calldata into our pointers.
                calldatacopy(ptr_messageHash, 66, 32)

                // Load results into our variables.
                messageHash := mload(ptr_messageHash)
            }

            ExecutionManager(msg.sender).ovmCREATEEOA(messageHash, uint8(v), r, s);
        } else {
            bytes memory message;
            assembly {
                // Set up our pointers.
                let ptr_message := mload(0x40)
                let size_message := sub(calldatasize, 66)
                mstore(ptr_message, size_message)
                mstore(0x40, add(ptr_message, add(size_message, 32)))

                // Copy calldata into our pointers.
                calldatacopy(add(ptr_message, 0x20), 66, size_message)

                // Load results into our variables.
                message := ptr_message       
            }

            bool isEthSignedMessage = uint8(transactionType) == 2;

            DataTypes.EOATransaction memory decodedTx = TransactionParser.decodeEOATransaction(
                message
            );

            bytes memory encodedTx = TransactionParser.encodeEOATransaction(
                decodedTx,
                isEthSignedMessage
            );

            address target = ECDSAUtils.recover(
                encodedTx,
                isEthSignedMessage,
                uint8(v),
                r,
                s,
                ExecutionManagerWrapper.ovmCHAINID(msg.sender)
            );

            bytes memory callbytes = abi.encodeWithSelector(
                bytes4(keccak256("execute(bytes,bool,uint8,bytes32,bytes32)")),
                message,
                isEthSignedMessage,
                uint8(v),
                r,
                s
            );

            ExecutionManagerWrapper.ovmCALL(
                msg.sender,
                target,
                callbytes,
                gasleft()
            );
        }
    }
}
