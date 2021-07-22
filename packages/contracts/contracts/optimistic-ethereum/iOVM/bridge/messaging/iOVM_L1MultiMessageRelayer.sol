// SPDX-License-Identifier: MIT
pragma solidity >0.5.0 <0.8.0;
pragma experimental ABIEncoderV2;

/* Interface Imports */
import { iOVM_L1CrossDomainMessenger } from
    "../../../iOVM/bridge/messaging/iOVM_L1CrossDomainMessenger.sol";

interface iOVM_L1MultiMessageRelayer {

    struct L2ToL1Message {
        address target;
        address sender;
        bytes message;
        uint256 messageNonce;
        iOVM_L1CrossDomainMessenger.L2MessageInclusionProof proof;
    }

    function batchRelayMessages(L2ToL1Message[] calldata _messages) external;
}
