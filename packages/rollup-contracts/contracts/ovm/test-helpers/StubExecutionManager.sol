pragma solidity ^0.5.0;

import {PartialStateManager} from "../PartialStateManager.sol";

/**
 * @title StubExecutionManager
 */
contract StubExecutionManager {
    PartialStateManager stateManager;

    function setStateManager(address _stateManagerAddress) external {
        stateManager = PartialStateManager(_stateManagerAddress);
    }

    function executeTransaction(
        uint _timestamp,
        uint _queueOrigin,
        address _ovmEntrypoint,
        bytes memory _callBytes,
        address _fromAddress,
        address _l1MsgSenderAddress,
        bool _allowRevert
    ) public {
        // Just call the state manager to store values a couple times
        stateManager.setStorage(0x1111111111111111111111111111111111111111, 0x1111111111111111111111111111111111111111111111111111111111111111, 0x1111111111111111111111111111111111111111111111111111111111111111);
        stateManager.setStorage(0x2222222222222222222222222222222222222222, 0x2222222222222222222222222222222222222222222222222222222222222222, 0x2222222222222222222222222222222222222222222222222222222222222222);
        // TODO: Make this a bit more comprehensive. Could even make it configurable?
    }
}
