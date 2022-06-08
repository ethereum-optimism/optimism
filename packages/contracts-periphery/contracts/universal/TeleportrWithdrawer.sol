// SPDX-License-Identifier: MIT
pragma solidity ^0.8.9;

import { AssetReceiver } from "./AssetReceiver.sol";

/**
 * @notice Stub interface for Teleportr.
 */
interface Teleportr {
    function withdrawBalance() external;
}

/**
 * @title TeleportrWithdrawer
 * @notice The TeleportrWithdrawer is a simple contract capable of withdrawing funds from the
 *         TeleportrContract and sending them to some recipient address.
 */
contract TeleportrWithdrawer is AssetReceiver {
    /**
     * @notice Address of the Teleportr contract.
     */
    address public teleportr;

    /**
     * @notice Address that will receive Teleportr withdrawals.
     */
    address public recipient;

    /**
     * @notice Data to be sent to the recipient address.
     */
    bytes public data;

    /**
     * @param _owner Initial owner of the contract.
     */
    constructor(address _owner) AssetReceiver(_owner) {}

    /**
     * @notice Allows the owner to update the recipient address.
     *
     * @param _recipient New recipient address.
     */
    function setRecipient(address _recipient) external onlyOwner {
        recipient = _recipient;
    }

    /**
     * @notice Allows the owner to update the Teleportr contract address.
     *
     * @param _teleportr New Teleportr contract address.
     */
    function setTeleportr(address _teleportr) external onlyOwner {
        teleportr = _teleportr;
    }

    /**
     * @notice Allows the owner to update the data to be sent to the recipient address.
     *
     * @param _data New data to be sent to the recipient address.
     */
    function setData(bytes memory _data) external onlyOwner {
        data = _data;
    }

    /**
     * @notice Withdraws the full balance of the Teleportr contract to the recipient address.
     *         Anyone is allowed to trigger this function since the recipient address cannot be
     *         controlled by the msg.sender.
     */
    function withdrawFromTeleportr() external {
        Teleportr(teleportr).withdrawBalance();
        (bool success, ) = recipient.call{ value: address(this).balance }(data);
        require(success, "TeleportrWithdrawer: send failed");
    }
}
