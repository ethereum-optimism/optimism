// SPDX-License-Identifier: MIT
pragma solidity ^0.8.9;

import { Lib_PredeployAddresses } from "../libraries/Lib_PredeployAddresses.sol";
import { OptimismPortal } from "./OptimismPortal.sol";
import { CrossDomainMessenger } from "../universal/CrossDomainMessenger.sol";

/**
 * @custom:proxied
 * @title L1CrossDomainMessenger
 * @notice The L1CrossDomainMessenger is a message passing interface between L1 and L2 responsible
 *         for sending and receiving data on the L1 side. Users are encouraged to use this
 *         interface instead of interacting with lower-level contracts directly.
 */
contract L1CrossDomainMessenger is CrossDomainMessenger {
    /**
     * @notice Address of the OptimismPortal.
     */
    OptimismPortal public portal;

    /**
     * @notice Initializes the L1CrossDomainMessenger.
     *
     * @param _portal Address of the OptimismPortal to send and receive messages through.
     */
    function initialize(OptimismPortal _portal) external {
        portal = _portal;

        address[] memory blockedSystemAddresses = new address[](1);
        blockedSystemAddresses[0] = address(this);

        _initialize(Lib_PredeployAddresses.L2_CROSS_DOMAIN_MESSENGER, blockedSystemAddresses);
    }

    /**
     * @notice Checks whether the message being sent from the other messenger.
     *
     * @return True if the message was sent from the messenger, false otherwise.
     */
    function _isSystemMessageSender() internal view override returns (bool) {
        return msg.sender == address(portal) && portal.l2Sender() == otherMessenger;
    }

    /**
     * @notice Sends a message via the OptimismPortal contract.
     *
     * @param _to       Address of the recipient on L2.
     * @param _gasLimit Minimum gas limit that the message can be executed with.
     * @param _value    ETH value to attach to the message and send to the recipient.
     * @param _data     Data to attach to the message and call the recipient with.
     */
    function _sendMessage(
        address _to,
        uint64 _gasLimit,
        uint256 _value,
        bytes memory _data
    ) internal override {
        portal.depositTransaction{ value: _value }(_to, _value, _gasLimit, false, _data);
    }
}
