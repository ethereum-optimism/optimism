// SPDX-License-Identifier: MIT
pragma solidity ^0.8.9;

import { AddressAliasHelper } from "@eth-optimism/contracts/standards/AddressAliasHelper.sol";
import {
    Lib_PredeployAddresses
} from "@eth-optimism/contracts/libraries/constants/Lib_PredeployAddresses.sol";
import { CrossDomainMessenger } from "../universal/CrossDomainMessenger.sol";
import { L2ToL1MessagePasser } from "./L2ToL1MessagePasser.sol";

contract L2CrossDomainMessenger is CrossDomainMessenger {
    /********************
     * Public Functions *
     ********************/

    /**
     * @notice initialize the L2CrossDomainMessenger by giving
     * it the address of the L1CrossDomainMessenger on L1
     */
    function initialize(address _l1CrossDomainMessenger) external {
        address[] memory blockedSystemAddresses = new address[](2);
        blockedSystemAddresses[0] = address(this);
        blockedSystemAddresses[1] = Lib_PredeployAddresses.L2_TO_L1_MESSAGE_PASSER;

        _initialize(_l1CrossDomainMessenger, blockedSystemAddresses);
    }

    /**
     * @notice Legacy getter for the remote messenger. This is included
     * to prevent any existing contracts that relay messages from breaking.
     * Use `otherMessenger()` for a standard API that works on both
     * the L1 and L2 cross domain messengers.
     */
    function l1CrossDomainMessenger() public returns (address) {
        return otherMessenger;
    }

    /**********************
     * Internal Functions *
     **********************/

    /**
     * @notice Only the L1CrossDomainMessenger can call the
     * L2CrossDomainMessenger
     */
    function _isSystemMessageSender() internal view override returns (bool) {
        return AddressAliasHelper.undoL1ToL2Alias(msg.sender) == otherMessenger;
    }

    /**
     * @notice Sending a message from L2 to L1 involves calling the L2ToL1MessagePasser
     * where it stores in a storage slot a commitment to the message being
     * sent to L1. A proof is then verified against that storage slot on L1.
     */
    function _sendMessage(
        address _to,
        uint64 _gasLimit,
        uint256 _value,
        bytes memory _data
    ) internal override {
        L2ToL1MessagePasser(payable(Lib_PredeployAddresses.L2_TO_L1_MESSAGE_PASSER))
            .initiateWithdrawal{ value: _value }(_to, _gasLimit, _data);
    }
}
