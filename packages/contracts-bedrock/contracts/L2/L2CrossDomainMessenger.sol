// SPDX-License-Identifier: MIT
pragma solidity ^0.8.9;

import { AddressAliasHelper } from "../libraries/AddressAliasHelper.sol";
import { Lib_PredeployAddresses } from "../libraries/Lib_PredeployAddresses.sol";
import { CrossDomainMessenger } from "../universal/CrossDomainMessenger.sol";
import { L2ToL1MessagePasser } from "./L2ToL1MessagePasser.sol";

/**
 * @custom:proxied
 * @custom:predeploy 0x4200000000000000000000000000000000000007
 * @title L2CrossDomainMessenger
 * @notice The L2CrossDomainMessenger is a high-level interface for message passing between L1 and
 *         L2 on the L2 side. Users are generally encouraged to use this contract instead of lower
 *         level message passing contracts.
 */
contract L2CrossDomainMessenger is CrossDomainMessenger {
    /**
     * @notice Initializes the L2CrossDomainMessenger.
     *
     * @param _l1CrossDomainMessenger Address of the L1CrossDomainMessenger contract.
     */
    function initialize(address _l1CrossDomainMessenger) external {
        address[] memory blockedSystemAddresses = new address[](2);
        blockedSystemAddresses[0] = address(this);
        blockedSystemAddresses[1] = Lib_PredeployAddresses.L2_TO_L1_MESSAGE_PASSER;

        _initialize(_l1CrossDomainMessenger, blockedSystemAddresses);
    }

    /**
     * @custom:legacy
     * @notice Legacy getter for the remote messenger. Use otherMessenger going forward.
     *
     * @return Address of the L1CrossDomainMessenger contract.
     */
    function l1CrossDomainMessenger() public returns (address) {
        return otherMessenger;
    }

    /**
     * @notice Checks that the message sender is the L1CrossDomainMessenger on L1.
     *
     * @return True if the message sender is the L1CrossDomainMessenger on L1.
     */
    function _isSystemMessageSender() internal view override returns (bool) {
        return AddressAliasHelper.undoL1ToL2Alias(msg.sender) == otherMessenger;
    }

    /**
     * @notice Sends a message from L2 to L1.
     *
     * @param _to       Address to send the message to.
     * @param _gasLimit Minimum gas limit to execute the message with.
     * @param _value    ETH value to send with the message.
     * @param _data     Data to trigger the recipient with.
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
