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
    address public immutable ADMIN;
    address public RECEIVER;
    uint public PricePerShare;

    constructor(
        ICrossDomainMessenger _messenger,
        address _admin
    ) {
        MESSENGER = _messenger;
        ADMIN = _admin;
    }

    function setReceiverAddress(address _reciever) public {
        RECEIVER = _reciever;
    }

    function setPricePerShare(uint _pps) public {
        require(
            msg.sender == ADMIN,
            "Greeter: Direct sender must be the L1Admin"
        );

        PricePerShare = _pps;

        MESSENGER.sendMessage(
            RECEIVER,
            abi.encodeCall(
                this.setPricePerShare,
                (
                    _pps
                )
            ),
            200000
        );
    }

}