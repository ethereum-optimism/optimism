pragma solidity ^0.5.0;

import { ICrossDomainMessenger } from "../optimistic-ethereum/bridge/interfaces/CrossDomainMessenger.interface.sol";
import { SimpleStorage } from "./SimpleStorage.sol";

contract CrossDomainSimpleStorage is SimpleStorage {
    ICrossDomainMessenger crossDomainMessenger;
    address public crossDomainMsgSender;

    function setMessenger(address _crossDomainMessengerAddress) public {
        crossDomainMessenger = ICrossDomainMessenger(_crossDomainMessengerAddress);
    }

    function crossDomainSetStorage(bytes32 key, bytes32 value) public {
        crossDomainMessenger = ICrossDomainMessenger(msg.sender);
        crossDomainMsgSender = crossDomainMessenger.xDomainMessageSender();
        setStorage(key, value);
    }
}