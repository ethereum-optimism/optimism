// SPDX-License-Identifier: MIT
pragma solidity ^0.8.9;

contract MockTeleportr {
    function withdrawBalance() external {
        payable(msg.sender).transfer(address(this).balance);
    }
}
