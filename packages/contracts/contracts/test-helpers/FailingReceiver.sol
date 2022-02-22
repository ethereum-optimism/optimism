// SPDX-License-Identifier: MIT
pragma solidity >=0.8.9;

/**
 * @title FailingReceiver
 */
contract FailingReceiver {
    /**
     * @notice Receiver that always reverts upon receiving ether.
     */
    receive() external payable {
        require(false, "FailingReceiver");
    }
}
