pragma solidity ^0.5.0;

import { ExecutionManager } from "../ExecutionManager.sol";
import { ECDSAUtils } from "../../utils/libraries/ECDSAUtils.sol";

contract SequencerMessageDecompressor {
    function()
        external
    {
        bool isEOACreation;
        uint8 v;
        bytes32 r;
        bytes32 s;
        assembly {
            calldatacopy(0, 1, isEOACreation)
            calldatacopy(1, 2, v)
            calldatacopy(2, 34, r)
            calldatacopy(34, 66, s)
        }

        if (isEOACreation) {
            bytes32 messageHash;
            assembly {
                calldatacopy(66, 98, messageHash)
            }

            ExecutionManager(msg.sender).ovmCREATEEOA(messageHash, v, r, s);
        } else {
            bool isEthSignedMessage;
            bytes memory message;
            assembly {
                calldatacopy(66, 1, isEthSignedMessage)
                calldatacopy(67, calldatasize, message)
            }
        }
    }
}