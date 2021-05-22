// SPDX-License-Identifier: MIT
pragma solidity ^0.7.0;

contract TestHelpers_SenderAssertions {
    function getSender()
        public
        view
        returns (
            address
        )
    {
        return msg.sender;
    }
}
