// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

interface ICrossDomainMessenger {
    function xDomainMessageSender() external view returns (address);
    function sendMessage(
        address _target,
        bytes calldata _message,
        uint32 _gasLimit
    ) external;
}

contract PPSUpdater {
    ICrossDomainMessenger public immutable MESSENGER;
    address public SENDER;
    uint public PricePerShare;

    constructor(
        ICrossDomainMessenger _messenger
    ) {
        MESSENGER = _messenger;
    }

    function setPricePerShare(uint _pps) public {
        require(
            msg.sender == address(MESSENGER),
            "Greeter: Direct sender must be the CrossDomainMessenger"
        );

        PricePerShare = _pps;
    }

}