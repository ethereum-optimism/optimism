// SPDX-License-Identifier: MIT
pragma solidity ^0.8.9;

import { PredeployAddresses } from "../libraries/PredeployAddresses.sol";
import { OptimismPortal } from "./OptimismPortal.sol";
import { CrossDomainMessenger } from "../universal/CrossDomainMessenger.sol";
import { Semver } from "../universal/Semver.sol";

/**
 * @custom:proxied
 * @title L1CrossDomainMessenger
 * @notice The L1CrossDomainMessenger is a message passing interface between L1 and L2 responsible
 *         for sending and receiving data on the L1 side. Users are encouraged to use this
 *         interface instead of interacting with lower-level contracts directly.
 */
contract L1CrossDomainMessenger is CrossDomainMessenger, Semver {
    /**
     * @notice Address of the OptimismPortal.
     */
    OptimismPortal public immutable portal;

    /**
     * @custom:semver 0.0.1
     *
     * @param _portal Address of the OptimismPortal contract on this network.
     */
    constructor(OptimismPortal _portal) Semver(0, 0, 1) {
        portal = _portal;
        initialize();
    }

    /**
     * @notice Initializer.
     */
    function initialize() public initializer {
        address[] memory blockedSystemAddresses = new address[](1);
        blockedSystemAddresses[0] = address(this);
        __CrossDomainMessenger_init(
            PredeployAddresses.L2_CROSS_DOMAIN_MESSENGER,
            blockedSystemAddresses
        );
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
