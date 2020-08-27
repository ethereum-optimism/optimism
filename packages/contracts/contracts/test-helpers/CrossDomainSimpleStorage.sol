pragma solidity ^0.5.0;

import { ICrossDomainMessenger} from "../optimistic-ethereum/bridge/CrossDomainMessenger.interface.sol";
import { SimpleStorage } from "./SimpleStorage.sol";

contract CrossDomainSimpleStorage is SimpleStorage {
    ICrossDomainMessenger crossDomainMessenger;
    address public crossDomainMsgSender;

    function setMessenger(address _crossDomainMessengerAddress) public {
        crossDomainMessenger = ICrossDomainMessenger(_crossDomainMessengerAddress);
    }

    function crossDomainSetStorage(bytes32 key, bytes32 value) public {
        crossDomainMsgSender = crossDomainMessenger.crossDomainMsgSender();
        setStorage(key, value);
    }
}