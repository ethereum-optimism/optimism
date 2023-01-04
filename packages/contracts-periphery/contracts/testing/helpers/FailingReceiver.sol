// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

contract FailingReceiver {
    receive() external payable {
        require(false, "FailingReceiver");
    }
}
