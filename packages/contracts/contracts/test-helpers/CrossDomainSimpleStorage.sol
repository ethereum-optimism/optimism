pragma solidity ^0.5.0;

import { IL1CrossDomainMessenger } from "../optimistic-ethereum/bridge/L1CrossDomainMessenger.interface.sol";
import { SimpleStorage } from "./SimpleStorage.sol";

contract CrossDomainSimpleStorage is SimpleStorage {
    IL1CrossDomainMessenger crossDomainMessenger;
    address public crossDomainMsgSender;

    function setMessenger(address _crossDomainMessengerAddress) public {
        crossDomainMessenger = IL1CrossDomainMessenger(_crossDomainMessengerAddress);
    }

    function crossDomainSetStorage(bytes32 key, bytes32 value) public {
        crossDomainMsgSender = crossDomainMessenger.xDomainMessageSender();
        setStorage(key, value);
    }
}