pragma solidity ^0.5.0;
pragma experimental ABIEncoderV2;

import {ExecutionManager} from "../ExecutionManager.sol";

/**
 * @title SimpleCall
 * @notice A simple contract testing the execution manager's CALL.
 */
contract SimpleCall {
    ExecutionManager executionManager;
    bytes32 testStorageVal;

    /**
     * Constructor currently accepts an execution manager & stores that in storage.
     * Note this should be the only storage that this contract ever uses & it should be replaced
     * by a hardcoded value once we have the transpiler.
     */
    constructor(address _executionManager) public {
        executionManager = ExecutionManager(_executionManager);
    }

    function makeCall(address _targetContract, bytes memory _calldata) public returns(bytes memory) {
        (bool success, bytes memory result) = executionManager.ovmCALL(_targetContract, _calldata);
        return result;
    }
}
