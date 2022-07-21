// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

contract MockTeleportr {
    function withdrawBalance() external {
        payable(msg.sender).transfer(address(this).balance);
    }
}
