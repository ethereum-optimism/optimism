pragma solidity ^0.8.9;

contract ICrossDomainMessenger {
    address public xDomainMessageSender;
}

contract SimpleStorage {
    address public msgSender;
    address public txOrigin;
    address public xDomainSender;
    bytes32 public value;
    uint256 public totalCount;

    function setValue(bytes32 newValue) public {
        msgSender = msg.sender;
        txOrigin = tx.origin;
        xDomainSender = ICrossDomainMessenger(msg.sender)
            .xDomainMessageSender();
        value = newValue;
        totalCount++;
    }

    function setValueNotXDomain(bytes32 newValue) public {
        msgSender = msg.sender;
        txOrigin = tx.origin;
        value = newValue;
        totalCount++;
    }

    function dumbSetValue(bytes32 newValue) public {
        value = newValue;
    }
}
