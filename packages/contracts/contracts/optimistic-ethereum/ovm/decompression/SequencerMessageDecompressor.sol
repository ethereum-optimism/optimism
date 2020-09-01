pragma solidity ^0.5.0;

/* Library Imports */
import { ECDSAUtils } from "../../utils/libraries/ECDSAUtils.sol";
import { OVMUtils } from "../../utils/libraries/OVMUtils.sol";

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
        bool isEOACreation;
        bytes1 v;
        bytes32 r;
        bytes32 s;
        assembly {
            // Set up our pointers.
            let ptr_isEOACreation := mload(0x40)
            mstore(0x40, add(ptr_isEOACreation, 1))
            let ptr_v := mload(0x40)
            mstore(0x40, add(ptr_v, 1))
            let ptr_r := mload(0x40)
            mstore(0x40, add(ptr_r, 32))
            let ptr_s := mload(0x40)
            mstore(0x40, add(ptr_s, 32))

            // Copy calldata into our pointers.
            calldatacopy(ptr_isEOACreation, 0, 1)
            calldatacopy(ptr_v, 1, 1)
            calldatacopy(ptr_r, 2, 32)
            calldatacopy(ptr_s, 34, 32)

            // Load results into our variables.
            isEOACreation := byte(0, mload(ptr_isEOACreation))
            v := mload(ptr_v)
            r := mload(ptr_r)
            s := mload(ptr_s)
        }

        if (isEOACreation) {
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
            bool isEthSignedMessage;
            bytes memory message;
            assembly {
                // Set up our pointers.
                let ptr_isEthSignedMessage := mload(0x40)
                mstore(0x40, add(ptr_isEthSignedMessage, 32))
                let ptr_message := mload(0x40)
                let size_message := sub(calldatasize, 67)
                mstore(ptr_message, size_message)
                mstore(0x40, add(ptr_message, add(size_message, 32)))

                // Copy calldata into our pointers.
                calldatacopy(ptr_isEthSignedMessage, 66, 1)
                calldatacopy(add(ptr_message, 0x20), 67, size_message)

                // Load results into our variables.
                isEthSignedMessage := byte(0, mload(ptr_isEthSignedMessage))
                message := ptr_message       
            }

            address target = ECDSAUtils.recover(
                message,
                isEthSignedMessage,
                uint8(v),
                r,
                s
            );

            bytes memory callbytes = abi.encodeWithSelector(
                bytes4(keccak256("execute(bytes,bool,uint8,bytes32,bytes32)")),
                message,
                isEthSignedMessage,
                uint8(v),
                r,
                s
            );

            OVMUtils.ovmCALL(
                msg.sender,
                target,
                callbytes
            );
        }
    }
}
