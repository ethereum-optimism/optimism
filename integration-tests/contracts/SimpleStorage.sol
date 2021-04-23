pragma solidity >=0.7.0;

contract ICrossDomainMessenger {
    address public xDomainMessageSender;
}

contract SimpleStorage {
    bytes32 public value;
    address public msgSender;
    address public xDomainSender;
    uint256 public totalCount;

    function setValue(bytes32 newValue) public {
        msgSender = msg.sender;
        xDomainSender = ICrossDomainMessenger(msg.sender)
            .xDomainMessageSender();
        value = newValue;
        totalCount++;
    }

    function dumbSetValue(bytes32 newValue) public {
        value = newValue;
    }
}
