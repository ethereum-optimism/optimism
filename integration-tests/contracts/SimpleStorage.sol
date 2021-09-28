pragma solidity ^0.8.8;

contract ICrossDomainMessenger {
    address public xDomainMessageSender;
}

contract SimpleStorage {
    address public msgSender;
    address public xDomainSender;
    bytes32 public value;
    uint256 public totalCount;

    function setValue(bytes32 newValue) public {
        msgSender = msg.sender;
        xDomainSender = ICrossDomainMessenger(msg.sender)
            .xDomainMessageSender();
        value = newValue;
        totalCount++;
    }

    function setValueNotXDomain(bytes32 newValue) public {
        msgSender = msg.sender;
        value = newValue;
        totalCount++;
    }

    function dumbSetValue(bytes32 newValue) public {
        value = newValue;
    }
}
