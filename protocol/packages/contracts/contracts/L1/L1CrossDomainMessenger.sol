// SPDX-License-Identifier: MIT
pragma solidity ^0.8.9;

import {
    Lib_PredeployAddresses
} from "@eth-optimism/contracts/libraries/constants/Lib_PredeployAddresses.sol";
import { OptimismPortal } from "./OptimismPortal.sol";
import { CrossDomainMessenger } from "../universal/CrossDomainMessenger.sol";

/**
 * @title L1CrossDomainMessenger
 * @dev The L1 Cross Domain Messenger contract sends messages from L1 to L2, and relays messages
 * from L2 onto L1.
 * This contract should be deployed behind an upgradable proxy
 */
contract L1CrossDomainMessenger is CrossDomainMessenger {
    /*************
     * Variables *
     *************/

    /**
     * @notice Address of the OptimismPortal.
     */
    OptimismPortal public portal;

    /********************
     * Public Functions *
     ********************/

    /**
     * @notice Initialize the L1CrossDomainMessenger
     * @param _portal The OptimismPortal
     */
    function initialize(OptimismPortal _portal) external {
        portal = _portal;

        address[] memory blockedSystemAddresses = new address[](1);
        blockedSystemAddresses[0] = address(this);

        _initialize(Lib_PredeployAddresses.L2_CROSS_DOMAIN_MESSENGER, blockedSystemAddresses);
    }

    /**********************
     * Internal Functions *
     **********************/

    /**
     * @notice Ensure that the L1CrossDomainMessenger can only be called
     * by the OptimismPortal and the L2 sender is the L2CrossDomainMessenger.
     */
    function _isSystemMessageSender() internal view override returns (bool) {
        return msg.sender == address(portal) && portal.l2Sender() == otherMessenger;
    }

    /**
     * @notice Sending a message in the L1CrossDomainMessenger involves
     * depositing through the OptimismPortal.
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
