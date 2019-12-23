pragma solidity ^0.5.0;
pragma experimental ABIEncoderV2;

import {ExecutionManager} from "../ExecutionManager.sol";

/**
 * @title SimpleStorage
 * @notice A simple contract testing the execution manager's storage.
 */
contract SimpleStorage {
    ExecutionManager executionManager;

    /**
     * Constructor currently accepts an execution manager & stores that in storage.
     * Note this should be the only storage that this contract ever uses & it should be replaced
     * by a hardcoded value once we have the transpiler.
     */
    constructor(address _executionManager) public {
        executionManager = ExecutionManager(_executionManager);
    }

    function getStorage(bytes32 slot) public view returns(bytes32) {
        return executionManager.ovmSLOAD(slot);
    }

    function setStorage(bytes32 slot, bytes32 value) public {
        executionManager.ovmSSTORE(slot, value);
    }
}
